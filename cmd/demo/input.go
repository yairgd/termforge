package main

import (
	tcell "github.com/gdamore/tcell/v2"

	"github.com/yairgd/termforge"
	"github.com/yairgd/termforge/commands"
	"github.com/yairgd/termforge/platform"
)

// HandleMouse routes pointer input. Everything it drives — focus, separator
// drag-resize, wheel scrolling, text selection, status-label copy — already
// lives in termforge; the app only decides which surface gets the event.
func (a *DemoApp) HandleMouse(ev *tcell.EventMouse) {
	x, y := ev.Position()
	btn := ev.Buttons()
	primary := btn&tcell.ButtonPrimary != 0
	middle := btn&tcell.ButtonMiddle != 0
	wheel := btn&(tcell.WheelUp|tcell.WheelDown) != 0

	// The help window paints over the workspace, so it takes the pointer while
	// open. A click outside closes it, then falls through to the pane under it.
	if a.help.Visible() {
		if a.WidgetRect(a.help).Contains(x, y) {
			a.help.HandleEvent(ev)
			a.RequestFrame()
			return
		}
		if primary || wheel {
			a.hideHelp()
		} else {
			return
		}
	}

	// The tab bar is a chrome row above the workspace, so it is hit-tested
	// against the rect the App chrome list gave it, not against the tree.
	if primary {
		if r := a.WidgetRect(a.tabBar); r.Contains(x, y) {
			if i := a.tabBar.TabAt(x - r.X()); a.tab.SetActive(i) {
				a.RequestFrame()
			}
			return
		}
	}

	inCmd := a.cmdLineRect().Contains(x, y)

	if a.Mode() == platform.ModeCommand || a.Mode() == platform.ModeCompletion {
		if middle && a.cmdWidget != nil {
			a.cmdWidget.HandleEvent(ev) // middle-click paste (X11 convention)
			a.RequestFrame()
			return
		}
		if primary && inCmd {
			a.clickCmdLine(x)
			return
		}
		if primary || wheel {
			a.leaveCommandMode() // clicking away behaves like Esc
		} else {
			return
		}
	}

	if primary && inCmd {
		a.enterCommandMode()
		a.clickCmdLine(x)
		return
	}

	// Touching a pane with any button makes it the focused pane.
	if lay := a.Layout(); lay != nil && (primary || wheel || middle) {
		lay.FocusAt(x, y)
	}

	// Drives separator drag, status-label copy, and the pane's own wheel and
	// selection handling.
	if a.tab != nil {
		a.tab.HandleEvent(ev)
	}
	a.RequestFrame()
}

// cmdLineRect is the cmdline's screen rect. It comes from the layout, not the
// App chrome list, because the cmdline is a leaf pinned to the bottom of the
// workspace tree. The zero Rect contains no point, so hit tests fail closed.
func (a *DemoApp) cmdLineRect() termforge.Rect {
	lay := a.Layout()
	if lay == nil {
		return termforge.Rect{}
	}
	return lay.PinnedBottomRect()
}

func (a *DemoApp) clickCmdLine(screenX int) {
	if a.cmdWidget == nil {
		return
	}
	a.cmdWidget.SetCursorAtLocalX(screenX - a.cmdLineRect().X())
	a.RequestFrame()
}

func (a *DemoApp) HandleInterrupt(ev *tcell.EventInterrupt) {
	if a == nil || a.ctx.Bus == nil {
		return
	}
	switch ev.Data().(type) {
	case termforge.SubmitMsg:
		a.ctx.Bus.Dispatch(ev.Data())
	}
}

const (
	helpMaxWidth  = 78
	helpMaxHeight = 24
	// The frame takes one column per side plus a one-column pad, so body text
	// gets the window width less this much. helpOverview must fit it.
	helpChromeWidth = 4
)

// helpRect centers the help window over the workspace band, leaving a margin
// of panes visible around it so it reads as floating. Returns the zero Rect
// when the terminal is too small to frame a window, which Draw skips.
func helpRect(c termforge.Canvas) termforge.Rect {
	width, height := c.W()-8, c.H()-6
	if width > helpMaxWidth {
		width = helpMaxWidth
	}
	if height > helpMaxHeight {
		height = helpMaxHeight
	}
	if width < 12 || height < 5 {
		return termforge.Rect{}
	}
	return c.ChildRect((c.W()-width)/2, (c.H()-2-height)/2, width, height)
}

func (a *DemoApp) HandleTTYResume() {}

func (a *DemoApp) handleNormalKey(ev *tcell.EventKey) bool {
	// The help window is the top surface, so it owns the keyboard while open.
	if a.help.Visible() {
		if ev.Key() == tcell.KeyEscape || (ev.Key() == tcell.KeyRune && ev.Rune() == 'q') {
			a.hideHelp()
			return true
		}
		a.help.HandleEvent(ev)
		a.RequestFrame()
		return true
	}
	if a.tryKeyBindings(a.keyBindings, ev) {
		return true
	}
	if w := a.focusedWidget(); w != nil {
		if h, ok := w.(termforge.FocusKeyHandler); ok && h.HandleFocusKey(ev) {
			return true
		}
	}
	return true
}

func (a *DemoApp) handleInsertKey(ev *tcell.EventKey) bool {
	if a.tryKeyBindings(a.insertKeys, ev) {
		return true
	}
	if w := a.focusedWidget(); w != nil {
		w.HandleEvent(ev)
	}
	return true
}

func (a *DemoApp) handleCommandKey(ev *tcell.EventKey) bool {
	if a.cmdWidget == nil {
		return true
	}
	a.cmdWidget.HandleEvent(ev)
	if ev.Key() == tcell.KeyEnter {
		a.cmdWidget.Deativate()
		if a.Mode() == platform.ModeCommand {
			a.SetMode(platform.ModeNormal)
		}
	}
	return true
}

func (a *DemoApp) tryKeyBindings(reg *commands.KeyBindingRegistry, ev *tcell.EventKey) bool {
	if reg == nil {
		return false
	}
	key, ok := platform.KeyFromEvent(ev)
	if !ok {
		reg.ResetPartial()
		return false
	}
	completed, handled := reg.HandleKey(key)
	if !handled {
		return false
	}
	if !completed {
		return reg.InPartial()
	}
	return true
}

func (a *DemoApp) focusedWidget() termforge.Widget {
	lay := a.Layout()
	if lay == nil {
		return nil
	}
	return lay.FocusedWidget()
}
