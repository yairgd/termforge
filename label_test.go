package termforge

import "testing"

func TestLabelDrawAndClick(t *testing.T) {
	g := NewGrid(20, 5)
	pane := NewCanvas(g).WithRect(NewRect(2, 1, 10, 3))
	l := NewLabel("hello")
	clicked := 0
	l.OnClick = func() { clicked++ }

	l.Draw(pane.WithRect(pane.ChildRect(0, 1, 10, 1)))
	if cellRune(g, 2, 2) != 'h' || cellRune(g, 6, 2) != 'o' {
		t.Fatal("text not at the label row")
	}
	if l.Click(2, 1) || clicked != 0 {
		t.Fatal("click above the label hit")
	}
	if !l.Click(11, 2) || clicked != 1 {
		t.Fatal("click on the padded row missed")
	}
}

func TestLabelFollowsPan(t *testing.T) {
	g := NewGrid(20, 5)
	pane := NewCanvas(g).WithRect(NewRect(0, 0, 10, 3)).WithOrigin(Point{Y: 2})
	l := NewLabel("x")
	l.Draw(pane.WithRect(pane.ChildRect(0, 2, 10, 1))) // content row 2 is at pane row 0
	if r := l.Rect(); r.Y() != 0 {
		t.Fatalf("label screen y = %d, want 0", r.Y())
	}
}
