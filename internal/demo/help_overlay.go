package demo

import (
	tcell "github.com/gdamore/tcell/v2"

	"github.com/yairgd/termforge"
	"github.com/yairgd/termforge/platform"
)

// The frame is painted cell by cell rather than with Canvas.DrawVerticalLocal,
// because those helpers run border composition and would join the frame into
// the pane separators underneath. A floating window must not merge into the
// layout it covers.
const (
	overlayTopLeft     = '┌'
	overlayTopRight    = '┐'
	overlayBottomLeft  = '└'
	overlayBottomRight = '┘'
	overlayHorizontal  = '─'
	overlayVertical    = '│'
)

// HelpOverlay is a floating, scrollable window drawn on top of the pane tree.
// It owns no layout space: the app hands it a centered rect and draws it after
// the workspace, so opening or closing it never reshapes the splits.
type HelpOverlay struct {
	termforge.BaseWidget
	doc     *termforge.ScrollDocument
	buf     *platform.Buffer
	title   string
	visible bool
}

func NewHelpOverlay() *HelpOverlay {
	buf := platform.NewBuffer()
	doc := termforge.NewScrollDocument(buf)
	doc.SetReadOnly(true)
	doc.SetCursorVisible(false)
	return &HelpOverlay{
		BaseWidget: termforge.BaseWidget{PaneName: "help"},
		doc:        doc,
		buf:        buf,
	}
}

func (o *HelpOverlay) Visible() bool { return o != nil && o.visible }

// Show replaces the contents and opens the window scrolled to the top.
func (o *HelpOverlay) Show(title string, lines []string) {
	if o == nil {
		return
	}
	o.title = title
	o.buf.Clear()
	for _, line := range lines {
		o.buf.AppendLine(line)
	}
	o.doc.Home()
	o.visible = true
}

func (o *HelpOverlay) Hide() {
	if o != nil {
		o.visible = false
	}
}

func (o *HelpOverlay) SetClipboard(io termforge.ClipboardIO) {
	if o != nil && o.doc != nil {
		o.doc.SetClipboard(io)
	}
}

func (o *HelpOverlay) Draw(c termforge.Canvas) {
	if !o.Visible() || c.W() < 4 || c.H() < 3 {
		return
	}
	frame := tcell.StyleDefault.Foreground(tcell.ColorSteelBlue)

	// Fill first: the window is opaque, so whatever the panes painted here is
	// covered rather than showing through.
	c.Fill(' ', tcell.StyleDefault)
	o.drawFrame(c, frame)

	if o.title != "" {
		c.Print(2, 0, frame.Bold(true), " "+o.title+" ")
	}
	c.Print(2, c.H()-1, frame, " Esc/q close · wheel, j/k scroll ")

	inner := c.WithRect(c.ChildRect(2, 1, c.W()-4, c.H()-2))
	o.doc.SetMouseOrigin(inner.ScreenX(0), inner.ScreenY(0))
	o.doc.Draw(inner)
}

func (o *HelpOverlay) drawFrame(c termforge.Canvas, style tcell.Style) {
	w, h := c.W(), c.H()
	for x := 1; x < w-1; x++ {
		c.SetContent(x, 0, overlayHorizontal, style)
		c.SetContent(x, h-1, overlayHorizontal, style)
	}
	for y := 1; y < h-1; y++ {
		c.SetContent(0, y, overlayVertical, style)
		c.SetContent(w-1, y, overlayVertical, style)
	}
	c.SetContent(0, 0, overlayTopLeft, style)
	c.SetContent(w-1, 0, overlayTopRight, style)
	c.SetContent(0, h-1, overlayBottomLeft, style)
	c.SetContent(w-1, h-1, overlayBottomRight, style)
}

// DrawStatusLine is a no-op: a floating window has no pane status bar.
func (o *HelpOverlay) DrawStatusLine(c termforge.Canvas, active bool) {}

func (o *HelpOverlay) HandleEvent(ev tcell.Event) {
	if !o.Visible() || o.doc == nil {
		return
	}
	switch e := ev.(type) {
	case *tcell.EventMouse:
		o.doc.HandleEvent(e)
	case *tcell.EventKey:
		o.doc.HandleEvent(e)
	}
}
