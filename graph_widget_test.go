package termforge

import (
	"strings"
	"testing"

	"github.com/yairgd/termforge/platform"
)

func graphRow(g *Grid, y int) string {
	var b strings.Builder
	for x := 0; x < g.W; x++ {
		r := g.Cells[x][y].Rune
		if r == 0 {
			r = ' '
		}
		b.WriteRune(r)
	}
	return b.String()
}

func newTestGraph(n int) *GraphWidget {
	w := NewGraphWidget(platform.NewAppContext())
	w.Graph().SetTitle("value")
	labels := make([]string, n)
	values := make([]float64, n)
	for i := range values {
		labels[i] = string(rune('a' + i%26))
		values[i] = float64(i)
	}
	w.Graph().SetPoints(labels, values)
	return w
}

func TestGraphWidgetZoomAndPanClamp(t *testing.T) {
	w := newTestGraph(16)

	w.ZoomIn()
	if w.visible() != 8 {
		t.Fatalf("zoom in: visible=%d want 8", w.visible())
	}
	for i := 0; i < 10; i++ {
		w.ZoomIn()
	}
	if w.visible() != 2 {
		t.Fatalf("zoom in floor: visible=%d want 2", w.visible())
	}
	for i := 0; i < 20; i++ {
		w.PanRight()
	}
	if w.start != 14 {
		t.Fatalf("pan right clamp: start=%d want 14", w.start)
	}
	for i := 0; i < 20; i++ {
		w.PanLeft()
	}
	if w.start != 0 {
		t.Fatalf("pan left clamp: start=%d want 0", w.start)
	}
	for i := 0; i < 10; i++ {
		w.ZoomOut()
	}
	if w.visible() != 16 || w.span != 0 {
		t.Fatalf("zoom out ceiling: visible=%d span=%d", w.visible(), w.span)
	}
}

func TestGraphWidgetLabelsInsideRect(t *testing.T) {
	w := newTestGraph(16)
	g := NewGrid(30, 8)
	c := NewCanvas(g).WithRect(NewRect(5, 1, 20, 6))
	w.Draw(c)

	top, bottom := graphRow(g, 1), graphRow(g, 6)
	if !strings.Contains(top, "value") || !strings.Contains(top, "max 15.00") {
		t.Fatalf("top row %q", top)
	}
	if !strings.Contains(bottom, "min 0.00") || !strings.Contains(bottom, "a") {
		t.Fatalf("bottom row %q", bottom)
	}
	for y := 0; y < g.H; y++ {
		row := []rune(graphRow(g, y))
		if strings.TrimSpace(string(row[:5])) != "" || strings.TrimSpace(string(row[25:])) != "" {
			t.Fatalf("row %d bled outside the rect: %q", y, row)
		}
	}
	if !strings.Contains(graphRow(g, 2)+graphRow(g, 5), "•") {
		t.Fatal("no points drawn")
	}
}

func TestGraphWidgetPanChangesLabels(t *testing.T) {
	w := newTestGraph(16)
	g := NewGrid(30, 6)
	c := NewCanvas(g).WithRect(NewRect(0, 0, 30, 6))
	w.Draw(c)
	w.ZoomIn()
	w.PanRight()
	w.Draw(c)
	if !strings.Contains(graphRow(g, 5), "max") && !strings.Contains(graphRow(g, 0), "max") {
		t.Fatal("max label missing")
	}
	if w.start == 0 {
		t.Fatal("pan right did not move the view")
	}
}
