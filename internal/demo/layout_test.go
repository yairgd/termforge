package demo

import (
	"math"
	"testing"

	"github.com/yairgd/termforge"
)

func testPanes() (main, table, side, log *ScrollPane, p Panes) {
	main = NewScrollPane("main")
	table = NewScrollPane("table")
	side = NewScrollPane("side")
	log = NewScrollPane("log")
	return main, table, side, log, Panes{Main: main, Table: table, Side: side, Log: log}
}

// The demo's point is a tree, not a row of panes: an outer horizontal split
// over a vertical one that itself contains a horizontal split.
func TestBuildDefaultNestsSplitsBothWays(t *testing.T) {
	main, table, side, log, panes := testPanes()
	tree := BuildDefault(panes)

	root := tree.Root()
	if root == nil || root.Type != termforge.NodeSplit || root.Dir != termforge.Horizontal {
		t.Fatal("root should be a horizontal split")
	}
	if root.Second == nil || root.Second.Widget != log {
		t.Fatal("log should be the bottom band")
	}

	top := root.First
	if top == nil || top.Type != termforge.NodeSplit || top.Dir != termforge.Vertical {
		t.Fatal("workspace band should be a vertical split")
	}
	if top.First == nil || top.First.Widget != main {
		t.Fatal("main should be the left pane")
	}

	right := top.Second
	if right == nil || right.Type != termforge.NodeSplit || right.Dir != termforge.Horizontal {
		t.Fatal("right column should be a horizontal split")
	}
	if right.First == nil || right.First.Widget != table {
		t.Fatal("table should sit above side")
	}
	if right.Second == nil || right.Second.Widget != side {
		t.Fatal("side should sit below table")
	}
}

func TestBuildDefaultHasFourLeaves(t *testing.T) {
	_, _, _, _, panes := testPanes()
	tree := BuildDefault(panes)

	if n := len(termforge.CollectLeaves(tree.Root())); n != 4 {
		t.Fatalf("leaf count: got %d want 4", n)
	}
}

// Ratios must survive BuildDefault's use of SetEqualAlways during construction:
// it splits evenly, then sets the shape's final proportions.
func TestBuildDefaultKeepsCustomRatios(t *testing.T) {
	_, _, _, _, panes := testPanes()
	tree := BuildDefault(panes)

	root := tree.Root()
	cases := []struct {
		name  string
		node  *termforge.Node
		ratio float64
	}{
		{"workspace/log", root, 0.70},
		{"main/right", root.First, 0.62},
		{"table/side", root.First.Second, 0.5},
	}
	for _, tc := range cases {
		if math.Abs(tc.node.Ratio-tc.ratio) > 1e-9 {
			t.Errorf("%s ratio: got %v want %v", tc.name, tc.node.Ratio, tc.ratio)
		}
	}
	if tree.EqualAlways() {
		t.Error("equalalways must be off so the ratios and drags stick")
	}
}

// The second tab exists to be a different shape, not a second copy of the
// first: a banner over three columns, so its vertical splits nest inside each
// other where BuildDefault nests a horizontal one inside a vertical.
func TestBuildColumnsStacksBannerOverThreeColumns(t *testing.T) {
	top := NewScrollPane("top")
	left := NewScrollPane("left")
	middle := NewScrollPane("middle")
	right := NewScrollPane("right")
	tree := BuildColumns(ColumnPanes{Top: top, Left: left, Middle: middle, Right: right})

	root := tree.Root()
	if root == nil || root.Type != termforge.NodeSplit || root.Dir != termforge.Horizontal {
		t.Fatal("root should be a horizontal split")
	}
	if root.First == nil || root.First.Widget != top {
		t.Fatal("top should be the banner")
	}

	band := root.Second
	if band == nil || band.Type != termforge.NodeSplit || band.Dir != termforge.Vertical {
		t.Fatal("column band should be a vertical split")
	}
	if band.First == nil || band.First.Widget != left {
		t.Fatal("left should be the first column")
	}

	rest := band.Second
	if rest == nil || rest.Type != termforge.NodeSplit || rest.Dir != termforge.Vertical {
		t.Fatal("the remaining columns should be another vertical split")
	}
	if rest.First == nil || rest.First.Widget != middle {
		t.Fatal("middle should be the second column")
	}
	if rest.Second == nil || rest.Second.Widget != right {
		t.Fatal("right should be the third column")
	}
}

func TestBuildColumnsKeepsCustomRatios(t *testing.T) {
	tree := BuildColumns(ColumnPanes{
		Top:    NewScrollPane("top"),
		Left:   NewScrollPane("left"),
		Middle: NewScrollPane("middle"),
		Right:  NewScrollPane("right"),
	})

	root := tree.Root()
	cases := []struct {
		name  string
		node  *termforge.Node
		ratio float64
	}{
		{"banner/columns", root, 0.30},
		{"left/rest", root.Second, 0.34},
		{"middle/right", root.Second.Second, 0.5},
	}
	for _, tc := range cases {
		if math.Abs(tc.node.Ratio-tc.ratio) > 1e-9 {
			t.Errorf("%s ratio: got %v want %v", tc.name, tc.node.Ratio, tc.ratio)
		}
	}
	if tree.EqualAlways() {
		t.Error("equalalways must be off so the ratios and drags stick")
	}
}

// Clicking each quadrant must focus the pane the layout puts there. This
// covers the geometry and the mouse hit-testing the demo relies on in one go.
func TestBuildDefaultFocusAtHitsExpectedPanes(t *testing.T) {
	main, table, side, log, panes := testPanes()
	tree := BuildDefault(panes)

	// 80x24 with ratios 0.70 / 0.62 / 0.5 puts the workspace band above row 16,
	// main left of column 49, and splits the right column at about row 8.
	const w, h = 80, 24
	tree.BuildLayout(termforge.NewCanvas(termforge.NewGrid(w, h)).
		WithRect(termforge.NewRect(0, 0, w, h)))

	cases := []struct {
		name   string
		x, y   int
		expect termforge.Widget
	}{
		{"main", 10, 5, main},
		{"table", 70, 3, table},
		{"side", 70, 13, side},
		{"log", 40, 21, log},
	}
	for _, tc := range cases {
		if !tree.FocusAt(tc.x, tc.y) {
			t.Errorf("%s: FocusAt(%d,%d) hit no pane", tc.name, tc.x, tc.y)
			continue
		}
		if got := tree.FocusedWidget(); got != tc.expect {
			t.Errorf("%s: click at (%d,%d) focused the wrong pane", tc.name, tc.x, tc.y)
		}
	}
}
