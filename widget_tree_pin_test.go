package termforge

import (
	"testing"

	tcell "github.com/gdamore/tcell/v2"
)

// pinnedTree is the shape every host builds: a two-pane workspace with the
// cmdline pinned to the bottom row.
func pinnedTree() (*WidgetTree, NodeWidget) {
	tree := NewWidgetTree(NewStubWidget("code"))
	tree.Split(Horizontal, NewStubWidget("gdb"))
	cmd := NewStubWidget("")
	tree.PinBottom(cmd, 1)
	return tree, cmd
}

func buildPinned(t *testing.T, tree *WidgetTree, w, h int) Canvas {
	t.Helper()
	c := NewCanvas(NewGrid(w, h)).WithRect(NewRect(0, 0, w, h))
	tree.BuildLayout(c)
	return c
}

// The pinned leaf keeps exactly its rows at the bottom edge, whatever the
// ratios say, and the panes divide what is left above the separator.
func TestPinBottomGeometry(t *testing.T) {
	tree, cmd := pinnedTree()
	buildPinned(t, tree, 80, 24)

	// A 1-row leaf is far below minPaneCells; it must survive anyway.
	if got, want := tree.PinnedBottomRect(), NewRect(0, 23, 80, 1); got != want {
		t.Errorf("cmdline rect %v, want %v", got, want)
	}
	if cmd != tree.Root().Second.Widget {
		t.Fatal("pinned widget lost")
	}

	// The panes divide rows 0..21; row 22 is the chrome separator.
	top := tree.FindLeaf(func(w Widget) bool { return w.(*stubWidget).PaneName == "code" })
	bottom := tree.FindLeaf(func(w Widget) bool { return w.(*stubWidget).PaneName == "gdb" })
	if top == nil || bottom == nil {
		t.Fatal("panes missing from the tree")
	}
	if r := tree.leafRect(bottom); r.Bottom() != 22 {
		t.Errorf("bottom pane ends at %d, want 22 (row 22 is the separator)", r.Bottom())
	}
	if r := tree.leafRect(top); r.Y() != 0 {
		t.Errorf("top pane starts at %d, want 0", r.Y())
	}
}

// The point of pinning: the line above the cmdline is the split's own
// separator, painted by redrawGrid, not a border some layout draws by hand.
func TestPinnedSeparatorIsDrawn(t *testing.T) {
	tree, _ := pinnedTree()
	g := NewGrid(80, 24)
	c := NewCanvas(g).WithRect(NewRect(0, 0, 80, 24))
	tree.BuildLayout(c)
	tree.Draw(c)

	// Border cells carry edge flags and become runes at paint time.
	for _, x := range []int{0, 40, 79} {
		if cell := g.Cells[x][22]; !cell.Left && !cell.Right {
			t.Errorf("no separator edge at (%d,22)", x)
		}
	}
	// The cmdline row itself stays clear for the widget.
	if cell := g.Cells[40][23]; cell.Left || cell.Right {
		t.Error("the separator was drawn on the cmdline row")
	}
}

// Everything that walks panes must not see the chrome leaf, which is what
// keeps focus, :close, :only and click-to-focus behaving as before.
func TestPinnedLeafIsNotAPane(t *testing.T) {
	tree, cmd := pinnedTree()
	buildPinned(t, tree, 80, 24)

	for _, n := range CollectLeaves(tree.Root()) {
		if n.Widget == cmd {
			t.Fatal("CollectLeaves returned the pinned chrome leaf")
		}
	}
	if got, want := len(CollectLeaves(tree.Root())), 2; got != want {
		t.Errorf("pane count %d, want %d", got, want)
	}

	// WalkLeaves still visits it, or it would never be drawn.
	seen := false
	WalkLeaves(tree.Root(), func(n *Node) {
		if n.Widget == cmd {
			seen = true
		}
	})
	if !seen {
		t.Error("WalkLeaves skipped the pinned leaf; it would never paint")
	}

	// Focus must not land on it from the bottom pane.
	tree.FocusWidget(tree.FindLeaf(func(w Widget) bool {
		return w.(*stubWidget).PaneName == "gdb"
	}).Widget)
	tree.FocusDown()
	if tree.FocusedWidget() == cmd {
		t.Error("Ctrl-W j moved focus into the cmdline")
	}

	// Clicking the cmdline row must not focus it either.
	if tree.FocusAt(10, 23) && tree.FocusedWidget() == cmd {
		t.Error("clicking the cmdline row focused it as a pane")
	}
}

// :only collapses the panes but keeps the chrome row.
func TestOnlyFocusKeepsPinnedLeaf(t *testing.T) {
	tree, cmd := pinnedTree()
	buildPinned(t, tree, 80, 24)

	tree.FocusWidget(tree.FindLeaf(func(w Widget) bool {
		return w.(*stubWidget).PaneName == "code"
	}).Widget)
	tree.OnlyFocus()
	buildPinned(t, tree, 80, 24)

	if got := len(CollectLeaves(tree.Root())); got != 1 {
		t.Errorf("pane count after :only = %d, want 1", got)
	}
	if tree.PinnedBottomRect() != NewRect(0, 23, 80, 1) {
		t.Errorf(":only dropped the cmdline: rect %v", tree.PinnedBottomRect())
	}
	if tree.Root().Second.Widget != cmd {
		t.Error("pinned widget replaced by :only")
	}
}

// The chrome separator is not a drag handle: the row below it must stay one row.
func TestPinnedSeparatorIsNotDraggable(t *testing.T) {
	tree, _ := pinnedTree()
	buildPinned(t, tree, 80, 24)

	if n := tree.findSeparator(10, 22); n != nil && n.FixedSecond > 0 {
		t.Error("the pinned chrome separator offered itself as a drag handle")
	}
	tree.HandleEvent(tcell.NewEventMouse(10, 22, tcell.ButtonPrimary, 0))
	tree.HandleEvent(tcell.NewEventMouse(10, 15, tcell.ButtonPrimary, 0))
	tree.HandleEvent(tcell.NewEventMouse(10, 15, tcell.ButtonNone, 0))
	buildPinned(t, tree, 80, 24)

	if got := tree.PinnedBottomRect(); got.H() != 1 || got.Y() != 23 {
		t.Errorf("drag resized the cmdline: rect %v", got)
	}
}

// Re-pinning is what a :layout switch does after mounting a freshly built tree:
// the same widget lands in the new tree without stacking another split.
func TestPinBottomIsIdempotent(t *testing.T) {
	tree, cmd := pinnedTree()
	tree.PinBottom(cmd, 1)
	buildPinned(t, tree, 80, 24)

	if tree.Root().First.Type != NodeSplit {
		t.Fatal("re-pinning wrapped the tree a second time")
	}
	if got, want := tree.PinnedBottomRect(), NewRect(0, 23, 80, 1); got != want {
		t.Errorf("cmdline rect %v, want %v", got, want)
	}
}

// A console with no room for a pane above the chrome collapses to the panes
// rather than painting a cmdline over nothing.
func TestPinBottomCollapsesOnTinyCanvas(t *testing.T) {
	tree, _ := pinnedTree()
	c := NewCanvas(NewGrid(20, 2)).WithRect(NewRect(0, 0, 20, 2))
	tree.BuildLayout(c)
	tree.Draw(c)
}
