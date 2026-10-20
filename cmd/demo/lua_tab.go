//go:build lua

package main

import (
	"github.com/yairgd/termforge"
	"github.com/yairgd/termforge/internal/demo"
	"github.com/yairgd/termforge/lua"
)

// repaint asks for a repaint from any goroutine. Any posted interrupt makes the event loop
// repaint; App.RequestRedraw would also rebuild the canvas and resync the whole screen.
type repaint struct{}

func (a *DemoApp) repaint() { a.PostInterrupt(repaint{}) }

// buildLuaTab is the third tab: an LED-matrix-style pixel grid over the embedded Lua
// console. The console's `mat` library (plot, line, clear, size, delay) draws on the grid.
//
//	+------------------+
//	|     matrix       |
//	+------------------+
//	|   lua console    |
//	+------------------+
func (a *DemoApp) addLuaTab() error {
	m := &demo.Matrix{}
	matrix := demo.NewMatrixSheet(a.ctx, "matrix", m)
	demo.RegisterMatrixLua(m, a.repaint)

	pane, err := demo.NewLuaPane("lua", a.repaint)
	if err != nil {
		return err
	}
	pane.SetClipboard(a.ClipboardIO())
	a.luaPane = pane
	a.startLua()

	a.addBuiltin("matrix", matrix, tabLua)
	a.addBuiltin("lua", pane, tabLua)

	tree := termforge.NewWidgetTree(matrix)
	tree.SetEqualAlways(true)
	tree.Split(termforge.Horizontal, pane)
	tree.SetEqualAlways(false)
	if root := tree.Root(); root != nil && root.Type == termforge.NodeSplit {
		root.Ratio = 0.45
	}
	tree.FocusWidget(pane)
	a.wireTree(tree)
	a.tab.AddTab("lua", tree)
	return nil
}

// startLua runs the REPL on the pane's PTY; when it ends, luaEnded asks for a new one.
func (a *DemoApp) startLua() {
	go func() {
		lua.Run(a.luaPane.Slave())
		a.PostInterrupt(luaEnded{})
	}()
}

func (a *DemoApp) onLuaEnded() {
	a.luaPane.Note("[Lua exited; new state]")
	a.startLua()
}
