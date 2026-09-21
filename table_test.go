package termforge

import (
	"strings"
	"testing"

	tcell "github.com/gdamore/tcell/v2"
	"github.com/yairgd/termforge/platform"
)

func TestTableLayoutColumnWidths(t *testing.T) {
	tbl := NewTable()
	tbl.SetShowHeader(true)
	tbl.AddColumn("ID")
	tbl.AddColumn("Name")
	tbl.AddRow("1", "alice")
	tbl.AddRow("22", "bob-long")

	lay := tbl.Layout()
	if len(lay.colWidths) != 2 {
		t.Fatalf("colWidths len=%d", len(lay.colWidths))
	}
	if lay.colWidths[0] != 2 {
		t.Fatalf("col0 width=%d want 2", lay.colWidths[0])
	}
	if lay.colWidths[1] != 8 {
		t.Fatalf("col1 width=%d want 8", lay.colWidths[1])
	}
	if lay.stickyRows != 1 {
		t.Fatalf("stickyRows=%d want 1", lay.stickyRows)
	}
}

func TestTableTruncateInPaint(t *testing.T) {
	tbl := NewTable()
	tbl.SetShowHeader(false)
	tbl.AddColumnWidth("X", 4)
	tbl.AddRow("hello-world")

	g := NewGrid(4, 1)
	c := NewCanvas(g).WithRect(NewRect(0, 0, 4, 1))
	rv := NewRectViewport()
	tbl.PaintVisibleDefault(c, rv, 4, 1)

	if got := g.Cells[3][0].Rune; got != '…' {
		t.Fatalf("last rune=%q want ellipsis", got)
	}
}

func TestTableStickyHeaderAndVerticalPan(t *testing.T) {
	tbl := NewTable()
	tbl.SetShowHeader(true)
	tbl.AddColumn("A")
	for i := 0; i < 5; i++ {
		tbl.AddRow(string(rune('a' + i)))
	}

	g := NewGrid(3, 3) // 1 header + 2 data rows visible
	c := NewCanvas(g).WithRect(NewRect(0, 0, 3, 3))
	rv := NewRectViewport()
	rv.SetOrigin(0, 2)

	tbl.PaintVisibleDefault(c, rv, 3, 3)

	// row 0 = header
	if got := g.Cells[0][0].Rune; got != 'A' {
		t.Fatalf("header=%q want A", got)
	}
	// row 1 = data row 2 ('c')
	if got := g.Cells[0][1].Rune; got != 'c' {
		t.Fatalf("data row0=%q want c", got)
	}
	if got := g.Cells[0][2].Rune; got != 'd' {
		t.Fatalf("data row1=%q want d", got)
	}
}

func TestTableHorizontalPan(t *testing.T) {
	tbl := NewTable()
	tbl.SetShowHeader(false)
	tbl.AddColumnWidth("Left", 4)
	tbl.AddColumnWidth("Right", 4)
	tbl.SetGutter(1)
	tbl.AddRow("aaaa", "bbbb")

	g := NewGrid(5, 1)
	c := NewCanvas(g).WithRect(NewRect(0, 0, 5, 1))
	rv := NewRectViewport()
	rv.SetOrigin(5, 0) // align window start with second column

	tbl.PaintVisibleDefault(c, rv, 5, 1)

	var b strings.Builder
	for x := 0; x < 5; x++ {
		b.WriteRune(g.Cells[x][0].Rune)
	}
	got := strings.TrimSpace(b.String())
	if got != "bbbb" {
		t.Fatalf("panned line=%q want bbbb", got)
	}
}

// PaintVisible writes straight to the Canvas, which clips to the screen rather
// than the pane. A title and header need more rows than a one-row window has,
// so neither may paint outside it.
func TestTablePaintClipsToWindowHeight(t *testing.T) {
	tbl := NewTable()
	tbl.SetTitle("T")
	tbl.SetShowTitle(true)
	tbl.SetShowHeader(true)
	tbl.AddColumn("A")
	tbl.AddRow("a")

	g := NewGrid(2, 3)
	c := NewCanvas(g).WithRect(NewRect(0, 0, 2, 1))
	rv := NewRectViewport()

	tbl.PaintVisibleDefault(c, rv, 2, 1)

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

func TestTableWidgetDrawAndPanKeys(t *testing.T) {
	ctx := platform.NewAppContext()
	w := NewTableWidget(ctx)
	w.InitPanKeyBindings()
	w.PaneName = "Demo"
	tbl := w.Table()
	tbl.SetShowHeader(true)
	tbl.AddColumn("C1")
	tbl.AddColumn("C2")
	for i := 0; i < 10; i++ {
		tbl.AddRow("r"+string(rune('0'+i)), "x")
	}

	g := NewGrid(6, 4)
	c := NewCanvas(g).WithRect(NewRect(0, 0, 6, 4))
	w.Draw(c)

	ev := tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone)
	if !w.HandleFocusKey(ev) {
		t.Fatal("Down should be consumed")
	}
	w.Draw(c)

	if w.rv.Origin.Y != 1 {
		t.Fatalf("Origin.Y=%d want 1 after Down", w.rv.Origin.Y)
	}

	evLeft := tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModNone)
	w.HandleFocusKey(evLeft)
	if w.rv.Origin.X != 0 {
		t.Fatalf("Origin.X=%d want 0 with narrow content", w.rv.Origin.X)
	}
}

// A pane created by :vs sits at a non-zero rect origin. Drawing there must stay
// inside the pane's columns and rows, not spill onto its siblings.
func TestTableWidgetDrawClipsToOffsetPane(t *testing.T) {
	ctx := platform.NewAppContext()
	w := NewTableWidget(ctx)
	tbl := w.Table()
	tbl.SetTitle("Title")
	tbl.SetShowTitle(true)
	tbl.SetShowHeader(true)
	tbl.AddColumn("WideHeader")
	for i := 0; i < 6; i++ {
		tbl.AddRow("row" + string(rune('0'+i)))
	}

	g := NewGrid(9, 5)
	pane := NewCanvas(g).WithRect(NewRect(3, 1, 3, 2))
	w.Draw(pane)

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
	ctx := platform.NewAppContext()
	w := NewTableWidget(ctx)
	tbl := w.Table()
	tbl.SetShowHeader(false)
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

func TestTableContentOverflows(t *testing.T) {
	tbl := NewTable()
	tbl.SetShowHeader(true)
	tbl.AddColumn("WideColumnName")
	tbl.AddRow("data")
	if !tbl.ContentOverflows(4, 3) {
		t.Fatal("expected horizontal overflow")
	}
}
