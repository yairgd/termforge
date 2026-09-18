package termforge

import (
	"github.com/gdamore/tcell/v2"
)

// Widget is anything the app can draw and hand events to: a pane, a layout,
// or chrome such as the command line.
type Widget interface {
	HandleEvent(ev tcell.Event)
	Draw(c Canvas)
}

// StatusLineDrawer paints the one-row status band the layout reserves below a
// pane. Chrome that never occupies a pane does not implement it.
type StatusLineDrawer interface {
	DrawStatusLine(c Canvas, active bool)
}

// Focusable receives focus notifications from the layout before Draw.
type Focusable interface {
	SetFocused(focused bool)
}

// Clearable is implemented by widgets that can clear their content via :clear.
type Clearable interface {
	Clear()
}

// FocusKeyHandler receives keys while focused in normal mode (before insert).
// Used by scrollable panes like LoggerWidget.
type FocusKeyHandler interface {
	HandleFocusKey(ev *tcell.EventKey) bool
}
