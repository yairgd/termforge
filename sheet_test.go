package termforge

import (
	"strings"
	"testing"

	tcell "github.com/gdamore/tcell/v2"
	"github.com/yairgd/termforge/platform"
)

// block is a test widget: n rows of its name, optional cursor row, counts the events it got.
type block struct {
	name   string
	n      int
	cursor int
	events []tcell.Event
}

func (b *block) Draw(c Canvas) {
	for y := 0; y < c.H(); y++ {
		c.Print(0, y, tcell.StyleDefault, b.name)
	}
}
func (b *block) HandleEvent(ev tcell.Event)          { b.events = append(b.events, ev) }
func (b *block) HandleFocusKey(*tcell.EventKey) bool { return true }

func newSheet(h int) (*Sheet, []*block, *Grid, Canvas) {
	s := NewSheet(platform.NewAppContext())
	bs := []*block{{name: "A", n: 3}, {name: "B", n: 0}, {name: "C", n: 20}}
	for _, b := range bs {
		b := b
		s.AddPart(&SheetPart{
			Widget: b, Keys: true,
			Height: func() int { return b.n },
			Cursor: func() int { return b.cursor },
		})
	}
	g := NewGrid(10, h+4) // taller than the pane: bleed would show
	return s, bs, g, NewCanvas(g).WithRect(NewRect(0, 0, 10, h))
}

func sheetRows(g *Grid) []string {
	var out []string
	for y := 0; y < g.H; y++ {
		var b strings.Builder
		for x := 0; x < g.W; x++ {
			r := g.Cells[x][y].Rune
			if r == 0 {
				r = ' '
			}
			b.WriteRune(r)
		}
		out = append(out, strings.TrimSpace(b.String()))
	}
	return out
}

func TestSheetStacksPartsWithGapAndHidesEmpty(t *testing.T) {
	s, _, g, c := newSheet(8)
	s.Draw(c)
	rows := sheetRows(g)
	want := []string{"A", "A", "A", "", "C", "C", "C", "C", "", "", "", ""}
	for i, w := range want {
		if rows[i] != w {
			t.Fatalf("row %d = %q want %q (all %q)", i, rows[i], w, rows)
		}
	}
	if s.ContentHeight() != 3+1+20 {
		t.Fatalf("content %d", s.ContentHeight())
	}
}

func TestSheetWheelScrollsAndClamps(t *testing.T) {
	s, _, g, c := newSheet(8)
	s.Draw(c)
	for i := 0; i < 100; i++ {
		s.HandleEvent(tcell.NewEventMouse(1, 1, tcell.WheelDown, 0))
	}
	s.Draw(c)
	if s.Pan().Y != 24-8 {
		t.Fatalf("pan %d want %d", s.Pan().Y, 16)
	}
	if sheetRows(g)[7] != "C" || sheetRows(g)[8] != "" {
		t.Fatalf("rows %q", sheetRows(g))
	}
	for i := 0; i < 100; i++ {
		s.HandleEvent(tcell.NewEventMouse(1, 1, tcell.WheelUp, 0))
	}
	if s.Pan().Y != 0 {
		t.Fatalf("pan %d", s.Pan().Y)
	}
}

// A scroll away from the cursor survives other events; moving the cursor scrolls to it.
func TestSheetFollowsCursorOnlyWhenItMoves(t *testing.T) {
	s, bs, _, c := newSheet(8)
	s.Draw(c)
	for i := 0; i < 5; i++ {
		s.HandleEvent(tcell.NewEventMouse(1, 1, tcell.WheelDown, 0))
	}
	s.Draw(c)
	if s.Pan().Y != 5 {
		t.Fatalf("pan %d", s.Pan().Y)
	}
	s.HandleFocusKey(tcell.NewEventKey(tcell.KeyRune, 'x', tcell.ModNone))
	s.HandleEvent(tcell.NewEventMouse(1, 1, tcell.ButtonNone, 0))
	if s.Pan().Y != 5 {
		t.Fatalf("pan jumped to %d", s.Pan().Y)
	}

	s.HandleFocusKey(tcell.NewEventKey(tcell.KeyTab, 0, tcell.ModNone)) // A -> C (part 2, top 4)
	bs[2].cursor = 19
	s.HandleFocusKey(tcell.NewEventKey(tcell.KeyRune, 'j', tcell.ModNone))
	if want := 4 + 19 - 8 + 1; s.Pan().Y != want {
		t.Fatalf("pan %d want %d", s.Pan().Y, want)
	}
}

func TestSheetTabSkipsHiddenParts(t *testing.T) {
	s, _, _, c := newSheet(8)
	s.Draw(c)
	tab := tcell.NewEventKey(tcell.KeyTab, 0, tcell.ModNone)
	for _, want := range []int{2, 0, 2} { // B is hidden
		s.HandleFocusKey(tab)
		if s.Active() != want {
			t.Fatalf("active %d want %d", s.Active(), want)
		}
	}
}

func TestSheetMouseRouting(t *testing.T) {
	s, bs, _, c := newSheet(8)
	s.Draw(c)

	s.HandleEvent(tcell.NewEventMouse(1, 5, tcell.Button1, 0)) // press on C
	if s.Active() != 2 || len(bs[2].events) != 1 || len(bs[0].events) != 0 {
		t.Fatalf("press: active %d events %d/%d", s.Active(), len(bs[0].events), len(bs[2].events))
	}
	s.HandleEvent(tcell.NewEventMouse(1, 1, tcell.WheelRight, 0)) // sideways over A
	if len(bs[0].events) != 1 {
		t.Fatal("sideways wheel did not reach the part under the pointer")
	}
	s.HandleEvent(tcell.NewEventMouse(1, 1, tcell.WheelDown, tcell.ModCtrl)) // ctrl+wheel over A
	if len(bs[0].events) != 2 || s.Pan().Y != 0 {
		t.Fatal("ctrl+wheel should go to the part, not scroll")
	}
	s.HandleEvent(tcell.NewEventMouse(1, 1, tcell.ButtonNone, 0)) // release
	if len(bs[0].events) != 3 || len(bs[2].events) != 2 || len(bs[1].events) != 0 {
		t.Fatalf("release: %d/%d/%d", len(bs[0].events), len(bs[1].events), len(bs[2].events))
	}
}
