package main

import (
	"testing"

	"github.com/yairgd/termforge"
	"github.com/yairgd/termforge/internal/demo"
)

// splitWith relies on parentSplit to find the node Split just created, so it
// can even out that one split without rebalancing the whole tree.
func TestParentSplitLocatesNewSplitNode(t *testing.T) {
	main := demo.NewScrollPane("main")
	right := demo.NewScrollPane("right")
	extra := demo.NewScrollPane("extra")

	tree := termforge.NewWidgetTree(main)
	tree.SetEqualAlways(false)
	tree.Split(termforge.Vertical, right) // root = Vertical(main, right)
	tree.FocusWidget(right)
	tree.Split(termforge.Horizontal, extra) // right becomes Horizontal(right, extra)

	leaf := tree.FocusedLeaf()
	n := parentSplit(tree.Root(), leaf)
	if n == nil {
		t.Fatal("parentSplit found no owning split")
	}
	if n == tree.Root() {
		t.Fatal("found the outer split, want the inner one just created")
	}
	if n.Dir != termforge.Horizontal {
		t.Errorf("owning split dir = %v, want Horizontal", n.Dir)
	}
	if n.First != leaf {
		t.Error("focused leaf should be the new split's first child")
	}
}

// The quirk being worked around: a fresh split inherits the leaf's ratio
// rather than defaulting to an even one.
func TestSplitInheritsLeafRatioUntilEvened(t *testing.T) {
	a := demo.NewScrollPane("a")
	b := demo.NewScrollPane("b")

	tree := termforge.NewWidgetTree(a)
	tree.SetEqualAlways(false)
	tree.Root().Ratio = 1
	tree.Split(termforge.Vertical, b)

	n := parentSplit(tree.Root(), tree.FocusedLeaf())
	if n == nil {
		t.Fatal("parentSplit found no owning split")
	}
	if n.Ratio != 1 {
		t.Fatalf("expected the inherited ratio 1, got %v", n.Ratio)
	}

	n.Ratio = 0.5
	if tree.Root().Ratio != 0.5 {
		t.Errorf("evening the split did not take: %v", tree.Root().Ratio)
	}
}

func TestParentSplitRejectsMissingNodes(t *testing.T) {
	main := demo.NewScrollPane("main")
	other := demo.NewScrollPane("other")
	tree := termforge.NewWidgetTree(main)
	tree.Split(termforge.Vertical, other)

	if parentSplit(nil, tree.FocusedLeaf()) != nil {
		t.Error("nil root should yield nil")
	}
	if parentSplit(tree.Root(), nil) != nil {
		t.Error("nil leaf should yield nil")
	}
	if parentSplit(tree.Root(), &termforge.Node{Type: termforge.NodeLeaf}) != nil {
		t.Error("a leaf outside the tree should yield nil")
	}
}
