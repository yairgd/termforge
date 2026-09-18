package termforge

import (
	"strings"
	"testing"

	"github.com/yairgd/termforge/platform"
)

func newTestPopup(names []string, selected int) *CompletionPopupWidget {
	p := NewCompletionPopupWidget(platform.NewAppContext())
	p.SetItems(names, selected)
	return p
}

func TestCompletionPopupSetItemsAndClear(t *testing.T) {
	p := newTestPopup([]string{"about", "logger", "gdb"}, 0)
	if !p.Active() {
		t.Fatal("expected active after SetItems")
	}
	p.Clear()
	if p.Active() {
		t.Fatal("cleared popup still active")
	}
	if w, h := p.PreferredSize(); w != 0 || h != 0 {
		t.Errorf("cleared popup wants %dx%d, expected nothing", w, h)
	}
}

// The window is sized to its candidates, capped so a long list cannot cover
// the whole workspace.
func TestCompletionPopupPreferredSize(t *testing.T) {
	p := newTestPopup([]string{"about", "logger", "breakpoint"}, 0)
	w, h := p.PreferredSize()
	if want := len("breakpoint") + 4; w != want {
		t.Errorf("width %d, want %d (longest name plus frame and padding)", w, want)
	}
	if want := 3 + 2; h != want {
		t.Errorf("height %d, want %d (one row per candidate plus frame)", h, want)
	}

	many := make([]string, popupMaxRows*2)
	for i := range many {
		many[i] = "cmd"
	}
	p.SetItems(many, 0)
	if _, h := p.PreferredSize(); h != popupMaxRows+2 {
		t.Errorf("height %d for %d candidates, want the %d-row cap plus frame",
			h, len(many), popupMaxRows)
	}
}

func popupRow(g *Grid, y int) string {
	var b strings.Builder
	for x := 0; x < g.W; x++ {
		r := g.Cells[x][y].Rune
		if r == 0 {
			r = ' '
		}
		b.WriteRune(r)
	}
	return b.String()
}

func TestCompletionPopupDrawsFramedList(t *testing.T) {
	p := newTestPopup([]string{"aaa", "bbb", "ccc"}, 1)
	w, h := p.PreferredSize()

	g := NewGrid(20, 10)
	c := Canvas{rect: NewRect(2, 2, w, h), grid: g}
	p.Draw(c)

	top := popupRow(g, 2)
	if !strings.Contains(top, string(popupTopLeft)) || !strings.Contains(top, string(popupTopRight)) {
		t.Errorf("top frame row = %q", top)
	}
	for i, name := range []string{"aaa", "bbb", "ccc"} {
		if row := popupRow(g, 3+i); !strings.Contains(row, name) {
			t.Errorf("row %d = %q, want it to contain %q", 3+i, row, name)
		}
	}
	bottom := popupRow(g, 2+h-1)
	if !strings.Contains(bottom, string(popupBottomLeft)) {
		t.Errorf("bottom frame row = %q", bottom)
	}

	// Nothing outside the window is touched: the popup owns no layout space
	// and must not bleed into the panes it covers.
	if row := popupRow(g, 2+h); strings.TrimSpace(row) != "" {
		t.Errorf("row below the window = %q, want untouched", row)
	}
}

// A candidate set taller than the window scrolls to keep the highlight visible.
func TestCompletionPopupScrollsToSelection(t *testing.T) {
	names := make([]string, 30)
	for i := range names {
		names[i] = string(rune('a'+i%26)) + "cmd"
	}
	p := newTestPopup(names, 25)

	g := NewGrid(20, 10)
	c := Canvas{rect: NewRect(0, 0, 10, 6), grid: g}
	p.Draw(c)

	rows := c.H() - 2
	if p.top > p.selected || p.selected >= p.top+rows {
		t.Errorf("selection %d not visible in rows %d..%d", p.selected, p.top, p.top+rows-1)
	}
}

// An inactive popup paints nothing, so a closed wildmenu never covers a pane.
func TestCompletionPopupInactiveDrawsNothing(t *testing.T) {
	p := NewCompletionPopupWidget(platform.NewAppContext())
	g := NewGrid(20, 10)
	p.Draw(Canvas{rect: NewRect(0, 0, 20, 10), grid: g})

	for y := 0; y < g.H; y++ {
		if strings.TrimSpace(popupRow(g, y)) != "" {
			t.Fatalf("inactive popup painted row %d", y)
		}
	}
}
