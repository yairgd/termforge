package termforge

import (
	"fmt"
	"math"

	tcell "github.com/gdamore/tcell/v2"
	"github.com/yairgd/termforge/platform"
)

// histBlocks are the partial bar heights, one eighth of a row each.
var histBlocks = []rune("▁▂▃▄▅▆▇")

// HistogramWidget draws a Graph as vertical bars: one bar per point, its height
// the value (a bin count, for a histogram), its label the bin's range. Bars
// widen to fill the rect; with more points than columns they are merged by max.
type HistogramWidget struct {
	BaseWidget
	graph *Graph
	style tcell.Style // of the bars
}

var _ Widget = (*HistogramWidget)(nil)

func NewHistogramWidget(ctx platform.AppContext) *HistogramWidget {
	return &HistogramWidget{BaseWidget: NewBaseWidget(ctx), graph: NewGraph()}
}

func (w *HistogramWidget) Graph() *Graph { return w.graph }

// SetBarStyle sets the style the bars are drawn with (default: the terminal's).
func (w *HistogramWidget) SetBarStyle(st tcell.Style) { w.style = st }

func (w *HistogramWidget) HandleEvent(tcell.Event) {}

func (w *HistogramWidget) Draw(c Canvas) {
	cw, ch := c.W(), c.H()
	for y := 0; y < ch; y++ {
		for x := 0; x < cw; x++ {
			c.SetContent(x, y, ' ', tcell.StyleDefault)
		}
	}
	g := w.graph
	n := g.Len()
	bold := tcell.StyleDefault.Bold(true)
	if n == 0 || cw < 4 || ch < 3 {
		c.Print(0, 0, bold, g.title)
		return
	}

	// Per-column value: one bar spans bw columns (leaving a gap when wide).
	bw := max(cw/n, 1)
	cols := make([]float64, cw)
	used := min(n*bw, cw)
	for x := 0; x < used; x++ {
		lo, hi := x*n/used, (x+1)*n/used
		if n <= cw {
			lo, hi = x/bw, x/bw+1
		}
		for i := lo; i < max(hi, lo+1) && i < n; i++ {
			cols[x] = math.Max(cols[x], g.values[i])
		}
		if n <= cw && bw >= 3 && x%bw == bw-1 {
			cols[x] = 0 // gap between wide bars
		}
	}
	peak := 0.0
	for _, v := range cols {
		peak = math.Max(peak, v)
	}

	top, plotH := 1, ch-2
	for x, v := range cols {
		if v <= 0 || peak <= 0 {
			continue
		}
		eighths := max(int(math.Round(v/peak*float64(plotH*8))), 1)
		for r := 0; r < eighths/8; r++ {
			c.SetContent(x, top+plotH-1-r, '█', w.style)
		}
		if p := eighths % 8; p > 0 {
			c.SetContent(x, top+plotH-1-eighths/8, histBlocks[p-1], w.style)
		}
	}

	dim := tcell.StyleDefault.Dim(true)
	c.Print(0, 0, bold, g.title)
	right(c, 0, dim, fmt.Sprintf("max %.0f", peak))
	c.Print(0, ch-1, dim, g.label(0))
	right(c, ch-1, dim, g.label(n-1))
}
