// Package demo holds panes/layout for the host showcase binary (cmd/demo).
// It is application code, not part of the termforge library: nothing in the
// framework may import it.
package demo

import "github.com/yairgd/termforge"

// Panes are the four stub views in the default demo layout.
type Panes struct {
	Main  termforge.Widget // primary text pane
	Table termforge.Widget // columnar list pane
	Side  termforge.Widget // status / notes
	Log   termforge.Widget // scrollable log
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
func BuildDefault(p Panes) *termforge.SplitLayout {
	tree := termforge.NewSplitLayout(p.Main)

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

// setRatio adjusts a split node, ignoring leaves and nil so callers can walk
// the tree without guarding every hop.
func setRatio(n *termforge.Node, ratio float64) {
	if n == nil || n.Type != termforge.NodeSplit {
		return
	}
	n.Ratio = ratio
}
