package termforge

import (
	"testing"

	"github.com/gdamore/tcell/v2"
)

func cellRune(g *Grid, x, y int) rune { return g.Cells[x][y].Rune }

func TestCanvasOrigin(t *testing.T) {
	g := NewGrid(30, 5)
	c := NewCanvas(g).WithRect(NewRect(10, 1, 20, 3))
	st := tcell.StyleDefault

	c.Print(0, 0, st, "A") // P(0,0)
	if cellRune(g, 10, 1) != 'A' {
		t.Fatal("origin 0: A not at pane (0,0)")
	}

	c.WithOrigin(Point{X: -1}).Print(0, 0, st, "B") // shifted right
	if cellRune(g, 11, 1) != 'B' {
		t.Fatal("origin -1: B not at local x=1")
	}

	c.WithOrigin(Point{X: 1}).Print(1, 0, st, "C") // x=1 lands on col 0
	if cellRune(g, 10, 1) != 'C' {
		t.Fatal("origin 1: C not at local x=0")
	}
}

func TestCanvasClipsToPane(t *testing.T) {
	g := NewGrid(30, 5)
	c := NewCanvas(g).WithRect(NewRect(10, 1, 20, 3))
	st := tcell.StyleDefault

	c.WithOrigin(Point{X: 1}).Print(0, 0, st, "XY") // X off the left edge, Y at col 0
	if cellRune(g, 9, 1) != 0 {
		t.Fatal("write bled left of the pane")
	}
	if cellRune(g, 10, 1) != 'Y' {
		t.Fatal("visible rune missing")
	}
	c.Print(18, 0, st, "abcd") // only 2 cells left in the pane
	if cellRune(g, 28, 1) != 'a' || cellRune(g, 29, 1) != 'b' {
		t.Fatal("in-pane runes missing")
	}
	c.Print(0, 3, st, "z") // below the pane
	if cellRune(g, 10, 4) != 0 {
		t.Fatal("write bled below the pane")
	}
}

func TestCanvasChildInheritsClipAndPans(t *testing.T) {
	g := NewGrid(30, 5)
	pane := NewCanvas(g).WithRect(NewRect(0, 0, 10, 3))
	cv := pane.WithOrigin(Point{X: 5})
	// Child table at content x=8..28 is wider than the pane.
	child := cv.WithRect(cv.ChildRect(8, 0, 20, 1))
	child.Print(0, 0, tcell.StyleDefault, "0123456789")
	// content x=8 -> screen 3; screen >= 10 is outside the pane
	if cellRune(g, 3, 0) != '0' || cellRune(g, 9, 0) != '6' {
		t.Fatalf("pan wrong: %q %q", cellRune(g, 3, 0), cellRune(g, 9, 0))
	}
	if cellRune(g, 10, 0) != 0 {
		t.Fatal("child bled past parent pane")
	}
}

// Pane status lines sit on the row below the pane (StatusLineRow); the tree
// paints them through WithStatusRow so the pane clip does not drop them.
func TestWidgetTreeDrawsStatusLineBelowPane(t *testing.T) {
	g := NewGrid(30, 10)
	root := NewCanvas(g).WithRect(NewRect(0, 0, 30, 9))
	tree := NewWidgetTree(&statusPane{})
	tree.BuildLayout(root)
	tree.Draw(root)
	if got := cellRune(g, 2, 9); got != 'S' {
		t.Fatalf("status row=%q want S", got)
	}
}

type statusPane struct{ BaseWidget }

func (p *statusPane) Draw(c Canvas)              {}
func (p *statusPane) HandleEvent(ev tcell.Event) {}
func (p *statusPane) DrawStatusLine(c Canvas, active bool) {
	PaintStatusBar(c, "STATUS", active)
}
