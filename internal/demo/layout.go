// Package demo holds panes/layout for the host showcase binary (cmd/demo).
// It is application code, not part of the termforge library: nothing in the
// framework may import it.
package demo

import "github.com/yairgd/termforge"

// Panes are the four stub views in the default demo layout.
type Panes struct {
	Main  termforge.NodeWidget // primary text pane
	Table termforge.NodeWidget // columnar list pane
	Side  termforge.NodeWidget // status / notes
	Log   termforge.NodeWidget // scrollable log
}

// BuildDefault nests splits in both directions so the demo shows a real tree
// rather than a single row of panes:
//
//	+----------+----------+
//	|          |  table   |
//	|   main   +----------+
//	|          |  side    |
//	+----------+----------+
//	|         log         |
//	+---------------------+
//
// The resulting tree is Horizontal(Vertical(main, Horizontal(table, side)), log),
// which gives separator drag-resize something interesting to act on: dragging
// the inner table/side bar must not move the outer main/log one.
func BuildDefault(p Panes) *termforge.WidgetTree {
	tree := termforge.NewWidgetTree(p.Main)

	// Split evenly first, then set ratios once the shape is final.
	tree.SetEqualAlways(true)
	tree.Split(termforge.Horizontal, p.Log)
	tree.FocusWidget(p.Main)
	tree.Split(termforge.Vertical, p.Table)
	tree.FocusWidget(p.Table)
	tree.Split(termforge.Horizontal, p.Side)
	tree.SetEqualAlways(false)

	root := tree.Root()
	setRatio(root, 0.70) // workspace band taller than the log
	if root != nil {
		setRatio(root.First, 0.62) // main wider than the right column
		if root.First != nil {
			setRatio(root.First.Second, 0.5) // table / side even
		}
	}

	tree.FocusWidget(p.Main)
	return tree
}

// ColumnPanes are the four stub views in the demo's second tab.
type ColumnPanes struct {
	Top    termforge.NodeWidget // full-width banner
	Left   termforge.NodeWidget // first column
	Middle termforge.NodeWidget // second column
	Right  termforge.NodeWidget // third column
}

// BuildColumns is a banner over three columns:
//
//	+---------------------------+
//	|            top            |
//	+--------+--------+---------+
//	|  left  | middle |  right  |
//	+--------+--------+---------+
//
// The tree is Horizontal(top, Vertical(left, Vertical(middle, right))). Two
// vertical splits nested inside each other is a shape BuildDefault never
// produces, so the second tab is a genuinely different geometry rather than a
// second copy of the first one: three separators meet the banner's, and
// left/right focus movement crosses two of them.
func BuildColumns(p ColumnPanes) *termforge.WidgetTree {
	tree := termforge.NewWidgetTree(p.Top)

	tree.SetEqualAlways(true)
	tree.Split(termforge.Horizontal, p.Left)
	tree.FocusWidget(p.Left)
	tree.Split(termforge.Vertical, p.Middle)
	tree.FocusWidget(p.Middle)
	tree.Split(termforge.Vertical, p.Right)
	tree.SetEqualAlways(false)

	root := tree.Root()
	setRatio(root, 0.30) // banner over the column band
	if root != nil {
		setRatio(root.Second, 0.34) // left column of three
		if root.Second != nil {
			setRatio(root.Second.Second, 0.5) // middle / right even
		}
	}

	tree.FocusWidget(p.Top)
	return tree
}

// setRatio adjusts a split node, ignoring leaves and nil so callers can walk
// the tree without guarding every hop.
func setRatio(n *termforge.Node, ratio float64) {
	if n == nil || n.Type != termforge.NodeSplit {
		return
	}
	n.Ratio = ratio
}
