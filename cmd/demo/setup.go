package main

import (
	"github.com/yairgd/termforge"
	"github.com/yairgd/termforge/internal/demo"
	"github.com/yairgd/termforge/platform"
)

func (a *DemoApp) Init() error {
	a.ctx = platform.NewAppContext()

	a.mainPane = demo.NewScrollPane("main",
		"termforge demo — a host app with no debugger in it.",
		"",
		"Type :help for the command reference, or click a pane to focus it.",
		"Drag a separator to resize. Wheel scrolls the pane under the pointer.",
	)
	a.mainPane.SetClipboard(a.ClipboardIO())
	a.builtins["main"] = a.mainPane

	a.tablePane = a.newWidgetTable()
	a.builtins["table"] = a.tablePane

	a.sidePane = demo.NewScrollPane("side",
		"side pane — :b side",
		"",
		"Panes are plain widgets; the split tree owns their geometry.",
	)
	a.sidePane.SetClipboard(a.ClipboardIO())
	a.builtins["side"] = a.sidePane

	a.logPane = termforge.NewLoggerWidget(a.ctx)
	a.logPane.PaneName = "log"
	a.builtins["log"] = a.logPane
	a.ctx.Log.Named("demo").Info("demo started")

	a.layout = demo.BuildDefault(demo.Panes{
		Main:  a.mainPane,
		Table: a.tablePane,
		Side:  a.sidePane,
		Log:   a.logPane,
	})
	a.tab = termforge.NewTabWidget("demo", a.layout)
	a.layout.SetStatusClipboard(a.ClipboardIO())
	a.layout.FocusWidget(a.mainPane)
	a.layout.SetOnResize(a.RequestFrame)
	// equalalways off: the default ratios and any separator drag survive a
	// later :vs / :split. ':equal' equalizes on demand instead.
	a.State().SetEqualAlways(false)
	a.layout.SetEqualAlways(false)
	// The workspace fills the screen; the cmdline is a pinned 1-row leaf at the
	// bottom of the tree, so the line above it is that split's own separator.
	a.AddWidget(a.tab)

	a.cmdWidget = termforge.NewCmdWidget(a.commandReg)
	a.cmdWidget.Ctx = a.ctx
	a.cmdWidget.SetPostInterrupt(a.PostInterrupt)
	a.cmdWidget.SetClipboard(a.ClipboardIO())
	a.SetCmdline(a.cmdWidget)
	a.layout.PinBottom(a.cmdWidget, 1)

	// Added last so it paints over the workspace: the App draws widgets in
	// registration order into one grid.
	a.help = demo.NewHelpOverlay()
	a.help.SetClipboard(a.ClipboardIO())
	a.AddFloatingWidget(a.help, helpRect)

	if a.ctx.Bus != nil {
		platform.Subscribe(a.ctx.Bus, func(msg termforge.SubmitMsg) {
			switch msg.CmdID {
			case termforge.CmdExitMode:
				a.leaveCommandMode()
			}
		})
	}

	a.InitKeyBindings()
	a.ExapData()

	a.RegisterModeHandler(platform.ModeNormal, a.handleNormalKey)
	a.RegisterModeHandler(platform.ModeInsert, a.handleInsertKey)
	a.RegisterModeHandler(platform.ModeCommand, a.handleCommandKey)
	return nil
}

// newWidgetTable builds the list pane. Its contents double as a cheat sheet of
// what termforge ships, so the pane demonstrates TableWidget and documents the
// framework at the same time.
func (a *DemoApp) newWidgetTable() *termforge.TableWidget {
	w := termforge.NewTableWidget(a.ctx)
	w.PaneName = "table"

	t := w.Table()
	t.SetTitle("termforge widgets")
	t.SetShowTitle(true)
	t.SetShowHeader(true)
	t.AddColumn("widget")
	t.AddColumn("kind")
	t.AddColumn("provides")
	t.AddRow("ScrollDocument", "text", "lines, search, selection")
	t.AddRow("TableWidget", "list", "columns, row select, /search")
	t.AddRow("CompositeTerminal", "pty", "xterm emulation, WireTTY")
	t.AddRow("ConsolePane", "repl", "scrollback + InputLine")
	t.AddRow("CmdWidget", "chrome", "':' command line")
	t.AddRow("CompletionBar", "chrome", "wildmenu row")
	t.AddRow("CompletionPopup", "overlay", "wildmenu window")
	t.AddRow("LoggerWidget", "log", "leveled log pane")
	t.AddRow("TabWidget", "frame", "holds the active Layout")
	t.AddRow("WidgetTree", "frame", "the split tree itself")

	w.InitSelectionKeyBindings()
	w.SetClipboard(a.ClipboardIO())
	return w
}
