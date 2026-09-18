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
	layout    *termforge.WidgetTree
	cmdWidget *termforge.CmdWidget
	ctx       platform.AppContext

	mainPane  *demo.ScrollPane
	sidePane  *demo.ScrollPane
	tablePane *termforge.TableWidget
	logPane   *termforge.LoggerWidget
	help      *demo.HelpOverlay
	builtins  map[string]termforge.Widget
}

// NewDemoApp builds and initializes the demo TUI.
func NewDemoApp() (*DemoApp, error) {
	a := &DemoApp{
		App:        termforge.NewApp(),
		commandReg: commands.NewCommandRegistry(),
		builtins:   make(map[string]termforge.Widget),
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
