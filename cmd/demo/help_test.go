package main

import (
	"strings"
	"testing"

	"github.com/yairgd/termforge"
)

// The help window never grows past helpMaxWidth, so any longer line is
// silently clipped on screen.
func TestHelpTextFitsTheWindow(t *testing.T) {
	limit := helpMaxWidth - helpChromeWidth

	pages := map[string][]string{"overview": helpOverview()}
	for _, c := range demoCommands {
		pages[":"+c.Name] = helpForCommand(c)
	}

	for page, lines := range pages {
		for i, line := range lines {
			if n := len([]rune(line)); n > limit {
				t.Errorf("%s line %d is %d cols, limit %d:\n  %s", page, i, n, limit, line)
			}
		}
	}
}

func TestCommandTableIsAligned(t *testing.T) {
	lines := commandTable()
	if len(lines) != len(demoCommands) {
		t.Fatalf("got %d rows for %d commands", len(lines), len(demoCommands))
	}

	// Every row must start its help column at the same offset.
	want := -1
	for i, line := range lines {
		usage := demoCommands[i].Usage
		idx := strings.Index(line, usage)
		if idx < 0 {
			t.Fatalf("row %d missing usage %q: %q", i, usage, line)
		}
		col := strings.Index(line, demoCommands[i].Help)
		if col < 0 {
			t.Fatalf("row %d missing help text: %q", i, line)
		}
		if want == -1 {
			want = col
		} else if col != want {
			t.Errorf("row %d help column at %d, want %d:\n  %s", i, col, want, line)
		}
	}
}

func TestEveryCommandIsDocumented(t *testing.T) {
	names := commandNames()
	if len(names) != len(demoCommands) {
		t.Fatalf("commandNames returned %d of %d", len(names), len(demoCommands))
	}
	for _, n := range names {
		c, ok := findCommand(n)
		if !ok {
			t.Errorf("findCommand(%q) failed", n)
			continue
		}
		if c.Usage == "" || c.Help == "" {
			t.Errorf("%q has an empty usage or help string", n)
		}
		if !strings.HasPrefix(c.Usage, ":"+n) {
			t.Errorf("%q usage %q should start with :%s", n, c.Usage, n)
		}
	}
	if _, ok := findCommand("nope"); ok {
		t.Error("findCommand accepted an unknown name")
	}
}

func TestHelpRectCentersAndClamps(t *testing.T) {
	canvas := func(w, h int) termforge.Canvas {
		return termforge.NewCanvas(termforge.NewGrid(w, h)).
			WithRect(termforge.NewRect(0, 0, w, h))
	}

	r := helpRect(canvas(200, 60))
	if r.W() != helpMaxWidth || r.H() != helpMaxHeight {
		t.Errorf("large terminal: got %dx%d want %dx%d",
			r.W(), r.H(), helpMaxWidth, helpMaxHeight)
	}
	if got, want := r.X(), (200-helpMaxWidth)/2; got != want {
		t.Errorf("not horizontally centered: x=%d want %d", got, want)
	}

	// Mid-size: the window shrinks with the terminal and stays on screen.
	r = helpRect(canvas(60, 20))
	if r.W() != 52 || r.H() != 14 {
		t.Errorf("mid terminal: got %dx%d want 52x14", r.W(), r.H())
	}
	if r.X() < 0 || r.Y() < 0 || r.X()+r.W() > 60 || r.Y()+r.H() > 20 {
		t.Errorf("window off screen: %+v", r)
	}

	// Too small to frame: zero rect, which HelpOverlay.Draw skips.
	if r := helpRect(canvas(14, 6)); r.W() != 0 || r.H() != 0 {
		t.Errorf("tiny terminal should yield a zero rect, got %+v", r)
	}
}
