package termforge

import (
	"fmt"
	"math"

	tcell "github.com/gdamore/tcell/v2"
	"github.com/yairgd/termforge/platform"
)

// GraphWidget is a line graph of one series: zoom in/out, pan left/right, and
// the title, y range and x labels drawn inside its own rect. Draw does all the
// layout and painting, like TableWidget.
type GraphWidget struct {
	BaseWidget
	graph *Graph

	style tcell.Style // of the points

	span  int // visible points; 0 = all
	start int // first visible point

	lastW, lastH int
}

var (
	_ Widget          = (*GraphWidget)(nil)
	_ FocusKeyHandler = (*GraphWidget)(nil)
	_ Focusable       = (*GraphWidget)(nil)
)

func NewGraphWidget(ctx platform.AppContext) *GraphWidget {
	return &GraphWidget{BaseWidget: NewBaseWidget(ctx), graph: NewGraph()}
}

func (w *GraphWidget) Graph() *Graph { return w.graph }

// SetPointStyle sets the style the points are drawn with (default: the terminal's).
func (w *GraphWidget) SetPointStyle(st tcell.Style) { w.style = st }

// InitGraphKeyBindings wires +/- for zoom, Left/Right (h/l) for pan, 0 to reset.
func (w *GraphWidget) InitGraphKeyBindings() {
	w.BindKeyFunc("zoom-in", func(args ...any) { w.ZoomIn() }, "+", "=")
	w.BindKeyFunc("zoom-out", func(args ...any) { w.ZoomOut() }, "-")
	w.BindKeyFunc("graph-left", func(args ...any) { w.PanLeft() }, "<Left>", "h")
	w.BindKeyFunc("graph-right", func(args ...any) { w.PanRight() }, "<Right>", "l")
	w.BindKeyFunc("graph-reset", func(args ...any) { w.ResetView() }, "0")
}

// ---- view ----

func (w *GraphWidget) visible() int {
	n := w.graph.Len()
	if w.span <= 0 || w.span > n {
		return n
	}
	return w.span
}

func (w *GraphWidget) clampView() {
	n := w.graph.Len()
	if w.span >= n {
		w.span = 0
	}
	w.start = max(min(w.start, n-w.visible()), 0)
}

// ResetView shows every point.
func (w *GraphWidget) ResetView() { w.span, w.start = 0, 0 }

// ZoomIn halves the visible points around the centre, down to 2.
func (w *GraphWidget) ZoomIn() { w.zoomTo(w.visible() / 2) }

// ZoomOut doubles the visible points around the centre.
func (w *GraphWidget) ZoomOut() { w.zoomTo(w.visible() * 2) }

func (w *GraphWidget) zoomTo(span int) {
	n := w.graph.Len()
	if n < 2 {
		return
	}
	span = max(span, 2)
	old := w.visible()
	w.start += (old - span) / 2
	w.span = span
	w.clampView()
}

func (w *GraphWidget) PanLeft()  { w.pan(-1) }
func (w *GraphWidget) PanRight() { w.pan(1) }

// pan moves the view by a quarter of the visible points.
func (w *GraphWidget) pan(dir int) {
	w.start += dir * max(w.visible()/4, 1)
	w.clampView()
}

// ---- events ----

func (w *GraphWidget) HandleFocusKey(ev *tcell.EventKey) bool { return w.HandleBoundKey(ev) }

// HandleEvent: horizontal wheel pans, Ctrl+wheel zooms. A plain vertical wheel
// is left alone so the owner can scroll.
func (w *GraphWidget) HandleEvent(ev tcell.Event) {
	switch e := ev.(type) {
	case *tcell.EventKey:
		w.HandleBoundKey(e)
	case *tcell.EventMouse:
		b := e.Buttons()
		ctrl := e.Modifiers()&tcell.ModCtrl != 0
		switch {
		case b&tcell.WheelLeft != 0:
			w.PanLeft()
		case b&tcell.WheelRight != 0:
			w.PanRight()
		case ctrl && b&tcell.WheelUp != 0:
			w.ZoomIn()
		case ctrl && b&tcell.WheelDown != 0:
			w.ZoomOut()
		}
	}
}

// ---- drawing ----

func (w *GraphWidget) Draw(c Canvas) {
	w.lastW, w.lastH = c.W(), c.H()
	w.clampView()

	for y := 0; y < w.lastH; y++ {
		for x := 0; x < w.lastW; x++ {
			c.SetContent(x, y, ' ', tcell.StyleDefault)
		}
	}
	g := w.graph
	if g.Len() == 0 || w.lastW < 4 || w.lastH < 3 {
		c.Print(0, 0, tcell.StyleDefault.Bold(true), g.title)
		return
	}

	n := w.visible()
	first := w.start
	last := w.start + n - 1
	lo, hi := math.Inf(1), math.Inf(-1)
	for i := first; i <= last; i++ {
		lo, hi = math.Min(lo, g.values[i]), math.Max(hi, g.values[i])
	}

	// Row 0 and the last row carry the labels; the points use the rows between.
	top, plotH := 1, w.lastH-2
	for x := 0; x < w.lastW; x++ {
		i := first
		if w.lastW > 1 {
			i += x * (n - 1) / (w.lastW - 1)
		}
		row := plotH / 2
		if hi > lo {
			row = int(math.Round((g.values[i] - lo) / (hi - lo) * float64(plotH-1)))
		}
		c.SetContent(x, top+plotH-1-row, '•', w.style)
	}

	bold := tcell.StyleDefault.Bold(true)
	dim := tcell.StyleDefault.Dim(true)
	c.Print(0, 0, bold, g.title)
	right(c, 0, dim, fmt.Sprintf("max %.2f", hi))
	c.Print(0, w.lastH-1, dim, g.label(first))
	right(c, w.lastH-1, dim, g.label(last))
	mid := fmt.Sprintf("min %.2f", lo)
	c.Print((w.lastW-runeLen(mid))/2, w.lastH-1, dim, mid)
}

// right prints text ending at the right edge of row y.
func right(c Canvas, y int, style tcell.Style, text string) {
	c.Print(max(c.W()-runeLen(text), 0), y, style, text)
}
