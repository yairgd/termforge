package demo

import (
	"sync"

	tcell "github.com/gdamore/tcell/v2"

	"github.com/yairgd/termforge"
	"github.com/yairgd/termforge/platform"
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
// foreground and the lower one as background. It is centered in its canvas; when the canvas
// is narrower than the matrix it scrolls sideways (horizontal wheel, Ctrl+wheel, Left/Right).
type MatrixWidget struct {
	termforge.BaseWidget
	m    *Matrix
	panX int
	maxX int // widest pan at the last Draw
}

func NewMatrixWidget(name string, m *Matrix) *MatrixWidget {
	return &MatrixWidget{BaseWidget: termforge.BaseWidget{PaneName: name}, m: m}
}

const panStep = 4

func (w *MatrixWidget) pan(dx int) { w.panX = max(0, min(w.panX+dx, w.maxX)) }

func (w *MatrixWidget) HandleEvent(ev tcell.Event) {
	e, ok := ev.(*tcell.EventMouse)
	if !ok {
		return
	}
	switch b := e.Buttons(); {
	case b&tcell.WheelLeft != 0, b&tcell.WheelUp != 0: // Ctrl+wheel reaches us from the sheet
		w.pan(-panStep)
	case b&tcell.WheelRight != 0, b&tcell.WheelDown != 0:
		w.pan(panStep)
	}
}

func (w *MatrixWidget) HandleFocusKey(ev *tcell.EventKey) bool {
	switch ev.Key() {
	case tcell.KeyLeft:
		w.pan(-panStep)
	case tcell.KeyRight:
		w.pan(panStep)
	default:
		return false
	}
	return true
}

func (w *MatrixWidget) Draw(c termforge.Canvas) {
	dx := 1
	if c.W() >= 2*MatrixW {
		dx = 2 // square-ish pixels when there is room
	}
	width := MatrixW * dx
	w.maxX = max(0, width-c.W())
	w.panX = min(w.panX, w.maxX)
	ox, oy := max(0, (c.W()-width)/2)-w.panX, max(0, (c.H()-MatrixH/2)/2) // centered
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

// MatrixSheet puts the matrix in a termforge.Sheet, so a pane shorter than the matrix scrolls
// it vertically (wheel, PgUp/PgDn, Home/End) and a taller one keeps it centered.
type MatrixSheet struct {
	*termforge.Sheet
	paneH int
}

func NewMatrixSheet(ctx platform.AppContext, name string, m *Matrix) *MatrixSheet {
	s := &MatrixSheet{Sheet: termforge.NewSheet(ctx)}
	s.PaneName = name
	s.AddPart(&termforge.SheetPart{
		Widget: NewMatrixWidget(name, m),
		Height: func() int { return max(MatrixH/2, s.paneH) },
		Keys:   true,
	})
	return s
}

func (s *MatrixSheet) Draw(c termforge.Canvas) {
	s.paneH = c.H()
	s.Sheet.Draw(c)
}
