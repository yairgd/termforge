package demo

import (
	"sync"

	tcell "github.com/gdamore/tcell/v2"

	"github.com/yairgd/termforge"
)

// Matrix size in pixels.
const (
	MatrixW = 64
	MatrixH = 32
)

// Matrix is an RGB pixel grid written from the Lua thread and drawn on the UI thread.
type Matrix struct {
	mu  sync.Mutex
	rgb [MatrixW * MatrixH * 3]byte
}

func (m *Matrix) Plot(x, y int, r, g, b byte) {
	if x < 0 || y < 0 || x >= MatrixW || y >= MatrixH {
		return
	}
	i := (y*MatrixW + x) * 3
	m.mu.Lock()
	m.rgb[i], m.rgb[i+1], m.rgb[i+2] = r, g, b
	m.mu.Unlock()
}

func (m *Matrix) Clear() {
	m.mu.Lock()
	m.rgb = [len(m.rgb)]byte{}
	m.mu.Unlock()
}

// Line draws with Bresenham's algorithm.
func (m *Matrix) Line(x0, y0, x1, y1 int, r, g, b byte) {
	dx, dy := abs(x1-x0), -abs(y1-y0)
	sx, sy := sign(x1-x0), sign(y1-y0)
	err := dx + dy
	for {
		m.Plot(x0, y0, r, g, b)
		if x0 == x1 && y0 == y1 {
			return
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func sign(v int) int {
	switch {
	case v < 0:
		return -1
	case v > 0:
		return 1
	}
	return 0
}

// MatrixWidget draws a Matrix two pixels per cell: '▀' with the upper pixel as
// foreground and the lower one as background.
type MatrixWidget struct {
	termforge.BaseWidget
	m *Matrix
}

func NewMatrixWidget(name string, m *Matrix) *MatrixWidget {
	return &MatrixWidget{BaseWidget: termforge.BaseWidget{PaneName: name}, m: m}
}

func (w *MatrixWidget) HandleEvent(tcell.Event) {}

func (w *MatrixWidget) Draw(c termforge.Canvas) {
	dx := 1
	if c.W() >= 2*MatrixW {
		dx = 2 // square-ish pixels when there is room
	}
	ox, oy := max(0, (c.W()-MatrixW*dx)/2), max(0, (c.H()-MatrixH/2)/2) // centered
	w.m.mu.Lock()
	defer w.m.mu.Unlock()
	pixel := func(x, y int) tcell.Color {
		i := (y*MatrixW + x) * 3
		return tcell.NewRGBColor(int32(w.m.rgb[i]), int32(w.m.rgb[i+1]), int32(w.m.rgb[i+2]))
	}
	for cy := 0; cy < MatrixH/2; cy++ {
		for x := 0; x < MatrixW; x++ {
			st := tcell.StyleDefault.Foreground(pixel(x, 2*cy)).Background(pixel(x, 2*cy+1))
			for k := 0; k < dx; k++ {
				c.SetContent(ox+x*dx+k, oy+cy, '▀', st)
			}
		}
	}
}
