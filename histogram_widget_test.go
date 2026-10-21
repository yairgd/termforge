package termforge

import (
	"strings"
	"testing"

	"github.com/yairgd/termforge/platform"
)

func TestHistogramWidgetBarsStayInsideRect(t *testing.T) {
	w := NewHistogramWidget(platform.NewAppContext())
	w.Graph().SetTitle("price")
	w.Graph().SetPoints([]string{"0", "", "", "9"}, []float64{1, 4, 8, 2})
	g := NewGrid(30, 8)
	w.Draw(NewCanvas(g).WithRect(NewRect(5, 1, 20, 6)))

	if top := graphRow(g, 1); !strings.Contains(top, "price") || !strings.Contains(top, "max 8") {
		t.Fatalf("top row %q", top)
	}
	if !strings.Contains(graphRow(g, 2), "█") { // the tallest bar fills the plot top
		t.Fatalf("no full-height bar: %q", graphRow(g, 2))
	}
	if bottom := graphRow(g, 6); !strings.Contains(bottom, "0") || !strings.Contains(bottom, "9") {
		t.Fatalf("bottom row %q", bottom)
	}
	for y := 0; y < g.H; y++ {
		row := []rune(graphRow(g, y))
		if strings.TrimSpace(string(row[:5])) != "" || strings.TrimSpace(string(row[25:])) != "" {
			t.Fatalf("row %d bled outside the rect: %q", y, string(row))
		}
	}
}

func TestHistogramWidgetMoreBinsThanColumns(t *testing.T) {
	w := NewHistogramWidget(platform.NewAppContext())
	vals := make([]float64, 100)
	vals[50] = 5
	w.Graph().SetPoints(nil, vals)
	g := NewGrid(10, 5)
	w.Draw(NewCanvas(g).WithRect(NewRect(0, 0, 10, 5)))
	if !strings.Contains(graphRow(g, 1)+graphRow(g, 2)+graphRow(g, 3), "█") {
		t.Fatal("merged bar missing")
	}
}
