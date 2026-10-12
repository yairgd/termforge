package termforge

import (
	"strings"
	"testing"

	tcell "github.com/gdamore/tcell/v2"
	"github.com/yairgd/termforge/platform"
)

// drawTable draws w into a fresh w x h grid at the origin and returns it.
func drawTable(w *TableWidget, cols, rows int) *Grid {
	g := NewGrid(cols, rows)
	w.Draw(NewCanvas(g).WithRect(NewRect(0, 0, cols, rows)))
	return g
}

func gridLine(g *Grid, y, n int) string {
	var b strings.Builder
	for x := 0; x < n; x++ {
		b.WriteRune(g.Cells[x][y].Rune)
	}
	return b.String()
}

func newTestTable() (*TableWidget, *Table) {
	w := NewTableWidget(platform.NewAppContext())
	return w, w.Table()
}

func TestTableColumnWidths(t *testing.T) {
	tbl := NewTable()
	tbl.AddColumn("ID")
	tbl.AddColumn("Name")
	tbl.AddColumnWidth("Fixed", 3)
	tbl.AddRow("1", "alice")
	tbl.AddRow("22", "bob-long")

	got := tbl.colWidths()
	if want := []int{2, 8, 3}; got[0] != want[0] || got[1] != want[1] || got[2] != want[2] {
		t.Fatalf("colWidths=%v want %v", got, want)
	}
}

func TestTableTruncateInPaint(t *testing.T) {
	w, tbl := newTestTable()
	tbl.AddColumnWidth("X", 4)
	tbl.AddRow("hello-world")

	g := drawTable(w, 4, 1)
	if got := g.Cells[3][0].Rune; got != '…' {
		t.Fatalf("last rune=%q want ellipsis", got)
	}
}

func TestTableStickyHeaderAndVerticalScroll(t *testing.T) {
	w, tbl := newTestTable()
	tbl.SetShowHeader(true)
	tbl.AddColumn("A")
	for i := 0; i < 5; i++ {
		tbl.AddRow(string(rune('a' + i)))
	}
	w.top = 2

	g := drawTable(w, 3, 3) // 1 header + 2 data rows visible
	if got := gridLine(g, 0, 1) + gridLine(g, 1, 1) + gridLine(g, 2, 1); got != "Acd" {
		t.Fatalf("column=%q want Acd", got)
	}
}

func TestTableHorizontalPan(t *testing.T) {
	w, tbl := newTestTable()
	tbl.AddColumnWidth("Left", 4)
	tbl.AddColumnWidth("Right", 4)
	tbl.AddRow("aaaa", "bbbb")
	w.panX = 5 // align window start with second column

	g := drawTable(w, 5, 1)
	if got := strings.TrimSpace(gridLine(g, 0, 5)); got != "bbbb" {
		t.Fatalf("panned line=%q want bbbb", got)
	}
}

// A title and header need more rows than a one-row window has, so neither may
// paint outside it.
func TestTableDrawClipsToWindowHeight(t *testing.T) {
	w, tbl := newTestTable()
	tbl.SetTitle("T")
	tbl.SetShowTitle(true)
	tbl.SetShowHeader(true)
	tbl.AddColumn("A")
	tbl.AddRow("a")

	g := NewGrid(2, 3)
	w.Draw(NewCanvas(g).WithRect(NewRect(0, 0, 2, 1)))

	if got := g.Cells[0][0].Rune; got != 'T' {
		t.Fatalf("title=%q want T", got)
	}
	for y := 1; y < 3; y++ {
		for x := 0; x < 2; x++ {
			if got := g.Cells[x][y].Rune; got != 0 {
				t.Fatalf("cell (%d,%d)=%q should be untouched", x, y, got)
			}
		}
	}
}

func TestTableWidgetDrawAndSelectionKeys(t *testing.T) {
	w, tbl := newTestTable()
	w.InitSelectionKeyBindings()
	tbl.SetShowHeader(true)
	tbl.AddColumn("C1")
	tbl.AddColumn("C2")
	for i := 0; i < 10; i++ {
		tbl.AddRow("r"+string(rune('0'+i)), "x")
	}

	g := NewGrid(6, 4) // header + 3 data rows
	c := NewCanvas(g).WithRect(NewRect(0, 0, 6, 4))
	w.Draw(c)
	for i := 0; i < 4; i++ {
		if !w.HandleFocusKey(tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone)) {
			t.Fatal("Down should be consumed")
		}
	}
	w.Draw(c)

	if w.SelectedRow() != 4 || w.top != 2 {
		t.Fatalf("selected=%d top=%d want 4,2", w.SelectedRow(), w.top)
	}
	w.HandleFocusKey(tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModNone))
	if w.panX != 0 {
		t.Fatalf("panX=%d want 0 with narrow content", w.panX)
	}
}

// A pane created by :vs sits at a non-zero rect origin. Drawing there must stay
// inside the pane's columns and rows, not spill onto its siblings.
func TestTableWidgetDrawClipsToOffsetPane(t *testing.T) {
	w, tbl := newTestTable()
	tbl.SetTitle("Title")
	tbl.SetShowTitle(true)
	tbl.SetShowHeader(true)
	tbl.AddColumn("WideHeader")
	for i := 0; i < 6; i++ {
		tbl.AddRow("row" + string(rune('0'+i)))
	}

	g := NewGrid(9, 5)
	w.Draw(NewCanvas(g).WithRect(NewRect(3, 1, 3, 2)))

	for x := 0; x < 9; x++ {
		for y := 0; y < 5; y++ {
			inPane := x >= 3 && x < 6 && y >= 1 && y < 3
			if got := g.Cells[x][y].Rune; !inPane && got != 0 {
				t.Fatalf("cell (%d,%d)=%q outside pane should be untouched", x, y, got)
			}
		}
	}
	if got := g.Cells[3][1].Rune; got != 'T' {
		t.Fatalf("pane origin=%q want T", got)
	}
}

func TestTableWidgetSetFill(t *testing.T) {
	w, tbl := newTestTable()
	tbl.AddColumn("V")
	n := 0
	w.SetFill(func(t *Table) {
		n++
		t.AddRow("filled")
	})

	g := NewGrid(4, 2)
	c := NewCanvas(g).WithRect(NewRect(0, 0, 4, 2))
	w.Draw(c)
	w.Draw(c)
	if n != 2 {
		t.Fatalf("fill calls=%d want 2", n)
	}
	if tbl.NumRows() != 1 {
		t.Fatalf("rows=%d want 1", tbl.NumRows())
	}
}
