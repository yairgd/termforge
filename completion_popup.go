package termforge

import (
	tcell "github.com/gdamore/tcell/v2"
	"github.com/yairgd/termforge/platform"
)

// The frame is painted cell by cell rather than with Canvas.DrawHorizontalLocal,
// because those helpers run border composition and would join the frame into
// the pane separators underneath. A floating window must not merge into the
// layout it covers.
const (
	popupTopLeft     = '┌'
	popupTopRight    = '┐'
	popupBottomLeft  = '└'
	popupBottomRight = '┘'
	popupHorizontal  = '─'
	popupVertical    = '│'
)

// popupMaxRows caps the list so a long candidate set does not cover the whole
// workspace; the rest is reached by scrolling the selection.
const popupMaxRows = 12

// CompletionPopupWidget paints a CompletionMenu snapshot as a floating list
// window over the workspace — the same CompletionView contract as
// CompletionBarWidget, drawn as a window instead of a chrome row.
//
// It owns no layout space: the app registers it with AddFloatingWidget and
// hands it a rect each frame, so opening or closing it never reshapes the
// splits. Selection stays in CompletionMenu; this only paints.
type CompletionPopupWidget struct {
	BaseWidget
	names    []string
	selected int
	top      int // first visible candidate (vertical scroll)
}

var _ CompletionView = (*CompletionPopupWidget)(nil)

func NewCompletionPopupWidget(ctx platform.AppContext) *CompletionPopupWidget {
	return &CompletionPopupWidget{
		BaseWidget: NewBaseWidget(ctx),
	}
}

// SetItems replaces the candidate list. selected is clamped.
func (w *CompletionPopupWidget) SetItems(names []string, selected int) {
	w.names = append([]string(nil), names...)
	w.selected = selected
	if w.selected < 0 {
		w.selected = 0
	}
	if n := len(w.names); n > 0 && w.selected >= n {
		w.selected = n - 1
	}
	w.top = 0
}

// Active reports whether the window has candidates to paint.
func (w *CompletionPopupWidget) Active() bool {
	return len(w.names) > 0
}

// Clear closes the window.
func (w *CompletionPopupWidget) Clear() {
	w.names = nil
	w.selected = 0
	w.top = 0
}

func (w *CompletionPopupWidget) HandleEvent(ev tcell.Event) {
	// Navigation is owned by CompletionMenu + ModeCompletion keys.
}

// PreferredSize is the window size for the current candidates, frame included.
// Hosts use it in their AddFloatingWidget rect callback so the window is only
// as large as it needs to be. Returns 0,0 when there is nothing to show.
func (w *CompletionPopupWidget) PreferredSize() (width, height int) {
	if !w.Active() {
		return 0, 0
	}
	longest := 0
	for _, n := range w.names {
		if l := nameWidth(n); l > longest {
			longest = l
		}
	}
	rows := len(w.names)
	if rows > popupMaxRows {
		rows = popupMaxRows
	}
	// 2 cells of frame plus one column of padding on each side.
	return longest + 4, rows + 2
}

// ensureSelectedVisible scrolls the window so the highlight is on screen.
func (w *CompletionPopupWidget) ensureSelectedVisible(rows int) {
	n := len(w.names)
	if n == 0 || rows <= 0 {
		w.top = 0
		return
	}
	if w.top > n-rows {
		w.top = n - rows
	}
	if w.top < 0 {
		w.top = 0
	}
	if w.selected < w.top {
		w.top = w.selected
	}
	if w.selected >= w.top+rows {
		w.top = w.selected - rows + 1
	}
}

func (w *CompletionPopupWidget) Draw(c Canvas) {
	if !w.Active() || c.W() < 4 || c.H() < 3 {
		return
	}
	frame := tcell.StyleDefault.Foreground(tcell.ColorSteelBlue)
	item := tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorWhite)
	sel := tcell.StyleDefault.Background(tcell.ColorWhite).Foreground(tcell.ColorBlack).Bold(true)

	// Fill first: the window is opaque, so whatever the panes painted here is
	// covered rather than showing through.
	c.Fill(' ', item)
	w.drawFrame(c, frame)

	rows := c.H() - 2
	width := c.W() - 2
	w.ensureSelectedVisible(rows)

	for i := 0; i < rows && w.top+i < len(w.names); i++ {
		idx := w.top + i
		st := item
		if idx == w.selected {
			st = sel
		}
		name := w.names[idx]
		runes := []rune(name)
		if len(runes) > width-2 {
			if width-3 < 1 {
				continue
			}
			name = string(runes[:width-3]) + "…"
		}
		c.ClearLineRange(i+1, 1, c.W()-1, st)
		c.Print(2, i+1, st, name)
	}
}

func (w *CompletionPopupWidget) drawFrame(c Canvas, style tcell.Style) {
	width, height := c.W(), c.H()
	for x := 1; x < width-1; x++ {
		c.SetContent(x, 0, popupHorizontal, style)
		c.SetContent(x, height-1, popupHorizontal, style)
	}
	for y := 1; y < height-1; y++ {
		c.SetContent(0, y, popupVertical, style)
		c.SetContent(width-1, y, popupVertical, style)
	}
	c.SetContent(0, 0, popupTopLeft, style)
	c.SetContent(width-1, 0, popupTopRight, style)
	c.SetContent(0, height-1, popupBottomLeft, style)
	c.SetContent(width-1, height-1, popupBottomRight, style)
}
