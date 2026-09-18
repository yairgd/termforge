package termforge

type NodeType int

const (
	NodeLeaf  NodeType = iota // A leaf node: contains a single widget (no children)
	NodeSplit                 // A split node: divides space between two child nodes
)

type SplitDir int

const (
	Horizontal SplitDir = iota // Split top/bottom (one above the other)
	Vertical                   // Split left/right (side by side)
)

// NodeWidget is what a leaf of the split tree holds: a widget that also paints
// its own status row. Embedding BaseWidget is enough to satisfy it.
type NodeWidget interface {
	Widget
	StatusLineDrawer
}

// Node is structural tree state only. Per-frame paint/hit geometry lives in
// WidgetTree.geom; tree algorithms live in layout_tree.go.
type Node struct {
	Type NodeType // Defines whether this is a Leaf or Split node

	// --- Leaf node fields ---
	Widget NodeWidget // The UI component stored in this node (only valid if Type == Leaf)

	// --- Split node fields ---
	Dir   SplitDir // Direction of the split (Horizontal or Vertical)
	Ratio float64  // Portion of space given to the First child (range: 0.0–1.0)

	// FixedSecond pins the Second child to this many cells and ignores Ratio.
	// It exists for permanent full-width chrome at an edge of the workspace —
	// the cmdline, so the line above it is this split's own separator. A
	// transient window (wildmenu, help) belongs in the App's floating tier,
	// which costs no layout space; do not give it a node here.
	FixedSecond int

	First  *Node // First child node (top or left depending on Dir)
	Second *Node // Second child node (bottom or right depending on Dir)

	parent *Node
}

// SetWidget replaces the widget on a leaf node in O(1). Layout/geometry are unchanged.
// No-op if n is nil or not a leaf.
func (n *Node) SetWidget(w NodeWidget) {
	if n == nil || n.Type != NodeLeaf {
		return
	}
	n.Widget = w
}

// GetWidget returns the widget on this node (may be nil).
func (n *Node) GetWidget() NodeWidget {
	if n == nil {
		return nil
	}
	return n.Widget
}
