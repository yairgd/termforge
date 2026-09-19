package main

import (
	"github.com/yairgd/termforge"
	"github.com/yairgd/termforge/commands"
	"github.com/yairgd/termforge/internal/demo"
	"github.com/yairgd/termforge/platform"
)

// DemoApp is a host-only showcase with gdbforge-like chrome and basic commands.
type DemoApp struct {
	*termforge.App
	commandReg  *commands.CommandRegistry
	keyBindings *commands.KeyBindingRegistry
	insertKeys  *commands.KeyBindingRegistry

	tab       *termforge.TabWidget
	tabBar    *termforge.TabBarWidget
	cmdWidget *termforge.CmdWidget
	ctx       platform.AppContext

	help *demo.HelpOverlay
	// builtins are the panes ':b' can name, in declaration order.
	builtins     map[string]paneRef
	builtinNames []string
}

// paneRef is one named pane: the widget and the tab whose tree holds it, so
// ':b' can reach a pane that is not on screen.
type paneRef struct {
	widget termforge.Widget
	tab    int
}

// Layout returns the active tab's split tree — the single place the demo
// narrows the Layout interface to the concrete tiling type. It is nil if a tab
// ever hosts a Layout that is not a tree, which is the signal that pane and
// split operations do not apply.
func (a *DemoApp) Layout() *termforge.WidgetTree {
	if a == nil || a.tab == nil {
		return nil
	}
	tree, _ := a.tab.Layout().(*termforge.WidgetTree)
	return tree
}

// NewDemoApp builds and initializes the demo TUI.
func NewDemoApp() (*DemoApp, error) {
	a := &DemoApp{
		App:        termforge.NewApp(),
		commandReg: commands.NewCommandRegistry(),
		builtins:   make(map[string]paneRef),
	}
	a.App.Api = a
	if err := a.Init(); err != nil {
		a.Close()
		return nil, err
	}
	return a, nil
}

// Close releases app resources.
func (a *DemoApp) Close() {
	if a.App != nil {
		a.App.Close()
	}
}
