package main

import (
	"strings"

	"github.com/yairgd/termforge"
	"github.com/yairgd/termforge/commands"
	"github.com/yairgd/termforge/internal/demo"
	"github.com/yairgd/termforge/platform"
)

func (a *DemoApp) ExapData() {
	a.commandReg.Root.
		Group("window",
			commands.Cmd("left", a.OnFocusLeft),
			commands.Cmd("right", a.OnFocusRight),
			commands.Cmd("up", a.OnFocusUp),
			commands.Cmd("down", a.OnFocusDown),
		).
		LeafRestComplete("b", a.OnBuffer, a.bufferCompletions).
		LeafRestComplete("help", a.OnHelp, a.helpCompletions).
		Leaf("vs", a.SplitVertical).
		Leaf("split", a.SplitHorizontal).
		Leaf("close", a.CloseFocus).
		Leaf("only", a.OnlyFocus).
		Leaf("equal", a.EqualizeSplits).
		Leaf("clear", a.ClearFocus).
		Leaf("quit", a.Quit)
}

// argString returns the first rest-arg as a trimmed string.
func argString(args []any) string {
	if len(args) == 0 {
		return ""
	}
	s, ok := args[0].(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(s)
}

// completeFrom filters names by prefix for a rest-arg completer.
func completeFrom(names []string, prefix string) []string {
	prefix = strings.ToLower(strings.TrimSpace(prefix))
	var out []string
	for _, n := range names {
		if prefix == "" || strings.HasPrefix(n, prefix) {
			out = append(out, n)
		}
	}
	return out
}

func (a *DemoApp) bufferCompletions(prefix string, _ bool) []string {
	return completeFrom([]string{"main", "table", "side", "log"}, prefix)
}

func (a *DemoApp) helpCompletions(prefix string, _ bool) []string {
	return completeFrom(commandNames(), prefix)
}

func (a *DemoApp) OnFocusLeft(args ...any) {
	if a.layout != nil {
		a.layout.FocusLeft()
		a.RequestFrame()
	}
}
func (a *DemoApp) OnFocusRight(args ...any) {
	if a.layout != nil {
		a.layout.FocusRight()
		a.RequestFrame()
	}
}
func (a *DemoApp) OnFocusUp(args ...any) {
	if a.layout != nil {
		a.layout.FocusUp()
		a.RequestFrame()
	}
}
func (a *DemoApp) OnFocusDown(args ...any) {
	if a.layout != nil {
		a.layout.FocusDown()
		a.RequestFrame()
	}
}

func (a *DemoApp) EnterInsertMode(args ...any) {
	if a.layout != nil {
		a.layout.SetInsertActive(true)
	}
	a.SetMode(platform.ModeInsert)
	a.RequestRedraw()
}

func (a *DemoApp) SplitVertical(args ...any) {
	a.splitWith(termforge.Vertical)
}

func (a *DemoApp) SplitHorizontal(args ...any) {
	a.splitWith(termforge.Horizontal)
}

func (a *DemoApp) splitWith(dir termforge.SplitDir) {
	if a.layout == nil {
		return
	}
	p := demo.NewScrollPane("split",
		"new pane — :close removes it",
		"",
		"Drag the separator to resize; only the two adjacent panes move.",
	)
	p.SetClipboard(a.ClipboardIO())
	a.layout.Split(dir, p)

	// Split turns a leaf into a split node but leaves that node's Ratio at the
	// leaf's old value, which is usually 1 and would give the new pane a
	// one-column sliver. With equalalways on the framework hides this by
	// rebalancing the whole tree; this demo keeps its proportions instead, so
	// it evens out just the split it created.
	if n := parentSplit(a.layout.Root(), a.layout.FocusedLeaf()); n != nil {
		n.Ratio = 0.5
	}
	a.RequestFrame()
}

// parentSplit finds the split node that has leaf as a direct child.
func parentSplit(node, leaf *termforge.Node) *termforge.Node {
	if node == nil || leaf == nil || node.Type != termforge.NodeSplit {
		return nil
	}
	if node.First == leaf || node.Second == leaf {
		return node
	}
	if n := parentSplit(node.First, leaf); n != nil {
		return n
	}
	return parentSplit(node.Second, leaf)
}

// CloseFocus deletes the focused pane. DeleteFocus reports true when it
// declined the request because the pane is the last one left.
func (a *DemoApp) CloseFocus(args ...any) {
	if a.layout == nil {
		return
	}
	if a.layout.DeleteFocus() {
		a.ctx.Log.Named("demo").Warn("cannot close the last pane")
		return
	}
	a.RequestFrame()
}

// OnlyFocus collapses the tree to the focused pane (Vim :only).
func (a *DemoApp) OnlyFocus(args ...any) {
	if a.layout != nil {
		a.layout.OnlyFocus()
		a.RequestFrame()
	}
}

// EqualizeSplits resets every split to an even ratio. equalalways is off in
// this demo, so drags and the default ratios persist until this runs.
func (a *DemoApp) EqualizeSplits(args ...any) {
	if a.layout != nil {
		a.layout.Rebalance()
		a.RequestFrame()
	}
}

func (a *DemoApp) ClearFocus(args ...any) {
	if c, ok := a.focusedWidget().(termforge.Clearable); ok {
		c.Clear()
	}
	a.RequestFrame()
}

// OnHelp opens the floating help window, for one command when given an
// argument and for the whole demo otherwise.
func (a *DemoApp) OnHelp(args ...any) {
	topic := strings.TrimPrefix(argString(args), ":")
	if topic == "" {
		a.showHelp("help", helpOverview())
		return
	}
	c, ok := findCommand(topic)
	if !ok {
		a.ctx.Log.Named("demo").Warn("no help for: " + topic)
		return
	}
	a.showHelp("help :"+c.Name, helpForCommand(c))
}

func (a *DemoApp) showHelp(title string, lines []string) {
	if a.help == nil {
		return
	}
	a.help.Show(title, lines)
	a.RequestFrame()
}

// hideHelp closes the window. RequestFrame clears the grid, so the panes the
// window covered are repainted instead of keeping its cells.
func (a *DemoApp) hideHelp() {
	if a.help == nil || !a.help.Visible() {
		return
	}
	a.help.Hide()
	a.RequestFrame()
}

func (a *DemoApp) OnBuffer(args ...any) {
	name := argString(args)
	if name == "" {
		return
	}
	w, ok := a.builtins[name]
	if !ok || a.layout == nil {
		a.ctx.Log.Named("demo").Warn("unknown buffer: " + name)
		return
	}
	a.layout.FocusWidget(w)
	a.RequestFrame()
}

// Quit closes the focused pane and exits once the last one is gone (Vim :q).
func (a *DemoApp) Quit(args ...any) {
	if a.layout == nil || a.layout.DeleteFocus() {
		a.Exit()
		return
	}
	a.RequestFrame()
}

// ForceQuit exits however many panes are open, which is what the Ctrl-D reflex
// expects. ':quit' follows Vim and closes one pane at a time instead.
func (a *DemoApp) ForceQuit(args ...any) {
	a.Exit()
}
