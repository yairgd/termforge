package main

import (
	"github.com/yairgd/termforge"
	"github.com/yairgd/termforge/internal/demo"
	"github.com/yairgd/termforge/platform"
)

// Tab indices. Both tabs are built at startup, so ':b' can name a pane in
// either one and switch to the tab holding it.
const (
	tabPanes = iota
	tabColumns
)

func (a *DemoApp) Init() error {
	a.ctx = platform.NewAppContext()

	// Built before the trees: every tab pins this one widget as its bottom
	// chrome leaf, so the ':' line is in the same place whichever tab is up.
	a.cmdWidget = termforge.NewCmdWidget(a.commandReg)
	a.cmdWidget.Ctx = a.ctx
	a.cmdWidget.SetPostInterrupt(a.PostInterrupt)
	a.cmdWidget.SetClipboard(a.ClipboardIO())
	a.SetCmdline(a.cmdWidget)

	a.tab = termforge.NewTabWidget("panes", a.buildPanesTab())
	a.tab.AddTab("columns", a.buildColumnsTab())
	a.tabBar = termforge.NewTabBarWidget(a.tab)

	// Registration order is the geometry: the tab bar takes the top row and the
	// workspace fills what is left, down to the cmdline pinned inside the tree.
	a.AddRowWidget(a.tabBar, 1)
	a.AddWidget(a.tab)

	// equalalways off: the default ratios and any separator drag survive a
	// later :vs / :split. ':equal' equalizes on demand instead.
	a.State().SetEqualAlways(false)

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

// buildPanesTab is the first tab: one big pane beside a stacked pair, over a
// full-width log.
func (a *DemoApp) buildPanesTab() *termforge.WidgetTree {
	main := demo.NewScrollPane("main",
		"termforge demo — a host app with no debugger in it.",
		"",
		"Type :help for the command reference, or click a pane to focus it.",
		"Drag a separator to resize. Wheel scrolls the pane under the pointer.",
		"gt, or the tab bar above, switches to the second tab.",
	)
	main.SetClipboard(a.ClipboardIO())

	table := a.newWidgetTable()

	side := demo.NewScrollPane("side",
		"side pane — :b side",
		"",
		"Panes are plain widgets; the split tree owns their geometry.",
	)
	side.SetClipboard(a.ClipboardIO())

	logPane := termforge.NewLoggerWidget(a.ctx)
	logPane.PaneName = "log"
	a.ctx.Log.Named("demo").Info("demo started")

	a.addBuiltin("main", main, tabPanes)
	a.addBuiltin("table", table, tabPanes)
	a.addBuiltin("side", side, tabPanes)
	a.addBuiltin("log", logPane, tabPanes)

	tree := demo.BuildDefault(demo.Panes{
		Main:  main,
		Table: table,
		Side:  side,
		Log:   logPane,
	})
	a.wireTree(tree)
	tree.FocusWidget(main)
	return tree
}

// buildColumnsTab is the second tab: a banner over three columns. The two tabs
// hold unrelated trees, so switching swaps the whole geometry — and each keeps
// its own focused pane, ratios and separator drags.
func (a *DemoApp) buildColumnsTab() *termforge.WidgetTree {
	notes := demo.NewScrollPane("notes",
		"second tab — a banner over three columns.",
		"",
		"A tab owns a whole split tree, so this one nests its splits the other",
		"way round: Horizontal(notes, Vertical(keys, Vertical(mouse, about))).",
		"Split, resize and close panes here without touching the first tab.",
	)
	notes.SetClipboard(a.ClipboardIO())

	keys := demo.NewScrollPane("keys",
		"keys",
		"",
		"gt / gT   next / prev tab",
		"C-w hjkl  move focus",
		":         command line",
		"?         help window",
		"C-d       exit",
	)
	keys.SetClipboard(a.ClipboardIO())

	mouse := demo.NewScrollPane("mouse",
		"mouse",
		"",
		"click tab   switch tab",
		"click pane  focus it",
		"drag sep    resize",
		"wheel       scroll",
		"drag text   select",
	)
	mouse.SetClipboard(a.ClipboardIO())

	about := demo.NewScrollPane("about",
		"about",
		"",
		"Everything on screen is framework machinery: the demo supplies",
		"panes, two tab layouts and a handful of commands.",
	)
	about.SetClipboard(a.ClipboardIO())

	a.addBuiltin("notes", notes, tabColumns)
	a.addBuiltin("keys", keys, tabColumns)
	a.addBuiltin("mouse", mouse, tabColumns)
	a.addBuiltin("about", about, tabColumns)

	tree := demo.BuildColumns(demo.ColumnPanes{
		Top:    notes,
		Left:   keys,
		Middle: mouse,
		Right:  about,
	})
	a.wireTree(tree)
	return tree
}

// wireTree applies what a freshly built WidgetTree does not carry: clipboard
// for status-band copy, the resize hook, this demo's equalalways policy, and
// the shared command line pinned to its bottom edge.
func (a *DemoApp) wireTree(tree *termforge.WidgetTree) {
	tree.SetStatusClipboard(a.ClipboardIO())
	tree.SetOnResize(a.RequestFrame)
	tree.SetEqualAlways(false)
	tree.PinBottom(a.cmdWidget, 1)
}

// addBuiltin registers a pane under the name ':b' uses for it.
func (a *DemoApp) addBuiltin(name string, w termforge.Widget, tab int) {
	if _, seen := a.builtins[name]; !seen {
		a.builtinNames = append(a.builtinNames, name)
	}
	a.builtins[name] = paneRef{widget: w, tab: tab}
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
	t.AddRow("TabBarWidget", "chrome", "tab titles, one row")
	t.AddRow("WidgetTree", "frame", "the split tree itself")

	w.InitSelectionKeyBindings()
	w.SetClipboard(a.ClipboardIO())
	return w
}
