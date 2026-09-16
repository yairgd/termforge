package demo

import (
	"strings"
	"testing"

	tcell "github.com/gdamore/tcell/v2"

	"github.com/yairgd/termforge"
)

// rowText reads one grid row back as a string.
func rowText(g *termforge.Grid, y, w int) string {
	var b strings.Builder
	for x := 0; x < w; x++ {
		b.WriteRune(g.Cells[x][y].Rune)
	}
	return b.String()
}

// fill paints the whole canvas so tests can tell "overlay painted here" from
// "pane content still showing".
func fill(c termforge.Canvas, ch rune) {
	for y := 0; y < c.H(); y++ {
		for x := 0; x < c.W(); x++ {
			c.SetContent(x, y, ch, tcell.StyleDefault)
		}
	}
}

func newTestCanvas(w, h int) (*termforge.Grid, termforge.Canvas) {
	g := termforge.NewGrid(w, h)
	return g, termforge.NewCanvas(g).WithRect(termforge.NewRect(0, 0, w, h))
}

func TestHelpOverlayHiddenPaintsNothing(t *testing.T) {
	g, c := newTestCanvas(40, 10)
	fill(c, 'X')

	NewHelpOverlay().Draw(c)

	for y := 0; y < 10; y++ {
		if got := rowText(g, y, 40); strings.Trim(got, "X") != "" {
			t.Fatalf("hidden overlay painted row %d: %q", y, got)
		}
	}
}

func TestHelpOverlayDrawsFrameTitleAndBody(t *testing.T) {
	g, c := newTestCanvas(40, 12)
	fill(c, 'X')

	o := NewHelpOverlay()
	o.Show("help", []string{"first line", "second line"})
	o.Draw(c)

	top := rowText(g, 0, 40)
	if !strings.HasPrefix(top, string(overlayTopLeft)) {
		t.Errorf("top-left corner: %q", top)
	}
	if !strings.HasSuffix(top, string(overlayTopRight)) {
		t.Errorf("top-right corner: %q", top)
	}
	if !strings.Contains(top, "help") {
		t.Errorf("title missing from top border: %q", top)
	}

	bottom := rowText(g, 11, 40)
	if !strings.HasPrefix(bottom, string(overlayBottomLeft)) ||
		!strings.HasSuffix(bottom, string(overlayBottomRight)) {
		t.Errorf("bottom corners: %q", bottom)
	}

	if body := rowText(g, 1, 40); !strings.Contains(body, "first line") {
		t.Errorf("first body line not drawn: %q", body)
	}
	if body := rowText(g, 2, 40); !strings.Contains(body, "second line") {
		t.Errorf("second body line not drawn: %q", body)
	}
}

// The window floats over panes rather than blending with them, so no cell
// inside its rect may still show what was underneath.
func TestHelpOverlayIsOpaque(t *testing.T) {
	g, c := newTestCanvas(40, 12)
	fill(c, 'X')

	o := NewHelpOverlay()
	o.Show("help", []string{"body"})
	o.Draw(c)

	for y := 0; y < 12; y++ {
		if strings.ContainsRune(rowText(g, y, 40), 'X') {
			t.Fatalf("row %d still shows pane content: %q", y, rowText(g, y, 40))
		}
	}
}

// Hide must restore the pane content on the next paint, which is why the app
// calls RequestFrame (grid clear) rather than a plain redraw.
func TestHelpOverlayHideStopsPainting(t *testing.T) {
	o := NewHelpOverlay()
	o.Show("help", []string{"body"})
	if !o.Visible() {
		t.Fatal("Show did not make the overlay visible")
	}
	o.Hide()
	if o.Visible() {
		t.Fatal("Hide did not clear visibility")
	}

	g, c := newTestCanvas(40, 12)
	fill(c, 'X')
	o.Draw(c)
	if got := rowText(g, 5, 40); strings.Trim(got, "X") != "" {
		t.Fatalf("hidden overlay still painting: %q", got)
	}
}

func TestHelpOverlayTooSmallToFrame(t *testing.T) {
	g, c := newTestCanvas(3, 2)
	fill(c, 'X')

	o := NewHelpOverlay()
	o.Show("help", []string{"body"})
	o.Draw(c)

	for y := 0; y < 2; y++ {
		if got := rowText(g, y, 3); strings.Trim(got, "X") != "" {
			t.Fatalf("overlay painted into a too-small rect: %q", got)
		}
	}
}
