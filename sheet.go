package termforge

import (
	tcell "github.com/gdamore/tcell/v2"
	"github.com/yairgd/termforge/platform"
)

// SheetPart is one band of a Sheet.
type SheetPart struct {
	Widget Widget
	// Height is the part's full height in rows, asked on every Draw. 0 hides the part.
	Height func() int
	// Cursor is the row inside the part the cursor is on (a table's selected row, counting its
	// title and header rows), or -1. The sheet scrolls to it when it moves. Nil: no cursor.
	Cursor func() int
	// Keys makes the part a Tab stop that gets the keys while it is the active one.
	Keys bool

	top, h int
	rect   Rect // screen rect at the last Draw
}

// Sheet stacks parts top to bottom in one pane and scrolls them as a single sheet: each part is
// painted at its full height into a canvas whose origin is the scroll position, so the pane clips
// what falls outside. It owns the scrolling, the active part, Tab, the follow-the-cursor rule and
// the routing of mouse events, so a window made of several widgets only lists its parts:
//
//	s := NewSheet(ctx)
//	s.AddPart(&SheetPart{Widget: table, Height: tableRows, Cursor: tableCursor, Keys: true})
//	s.AddPart(&SheetPart{Widget: graph, Height: func() int { return 12 }, Keys: true})
//
// Mouse: a plain vertical wheel scrolls the sheet; a horizontal wheel or Ctrl+wheel goes to the
// part under the pointer; a press goes to the part under it and makes it active; a release goes
// to every part (a widget that saw a double click waits for it).
type Sheet struct {
	BaseWidget
	// Gap is the blank rows between shown parts.
	Gap int

	parts    []*SheetPart
	active   int
	followed [2]int // active part and cursor row the pan last followed
	pan      Point
	paneH    int
	contentH int
}

var (
	_ NodeWidget      = (*Sheet)(nil)
	_ Focusable       = (*Sheet)(nil)
	_ FocusKeyHandler = (*Sheet)(nil)
)

func NewSheet(ctx platform.AppContext) *Sheet {
	return &Sheet{BaseWidget: NewBaseWidget(ctx), Gap: 1}
}

// AddPart appends a part below the others and returns its index.
func (s *Sheet) AddPart(p *SheetPart) int {
	s.parts = append(s.parts, p)
	return len(s.parts) - 1
}

// Active is the index of the part that has the keys.
func (s *Sheet) Active() int { return s.active }

// Pan is the sheet cell shown at the pane's top-left.
func (s *Sheet) Pan() Point { return s.pan }

// ContentHeight is the sheet's full height at the last Draw.
func (s *Sheet) ContentHeight() int { return s.contentH }

// ScrollBy moves the sheet up (negative) or down by n rows.
func (s *Sheet) ScrollBy(n int) {
	s.pan.Y += n
	s.clampPan()
}

func (s *Sheet) clampPan() {
	s.pan.Y = max(min(s.pan.Y, s.contentH-s.paneH), 0)
}

func (s *Sheet) shown(i int) bool { return i >= 0 && i < len(s.parts) && s.parts[i].h > 0 }

func (s *Sheet) Draw(c Canvas) {
	y := 0
	for _, p := range s.parts {
		p.h = max(p.Height(), 0)
		if p.h == 0 {
			p.rect = Rect{}
			continue
		}
		if y > 0 {
			y += s.Gap
		}
		p.top = y
		y += p.h
	}
	s.paneH, s.contentH = c.H(), y
	s.clampPan()
	if !s.shown(s.active) || !s.parts[s.active].Keys {
		s.active = s.nextStop(s.active)
	}

	// WithRect(c.Rect()) fixes the pane as the clip before the origin moves.
	cv := c.WithRect(c.Rect()).WithOrigin(s.pan)
	for _, p := range s.parts {
		if p.h == 0 {
			continue
		}
		p.rect = cv.ChildRect(0, p.top, c.W(), p.h)
		p.Widget.Draw(cv.WithRect(p.rect))
	}
}

// nextStop is the next shown part after i that takes keys, or i itself when there is none.
func (s *Sheet) nextStop(i int) int {
	for k := 1; k <= len(s.parts); k++ {
		if j := (i + k) % len(s.parts); s.shown(j) && s.parts[j].Keys {
			return j
		}
	}
	return i
}

// follow scrolls to the active part's cursor, but only when the cursor (or the active part)
// changed since the last time: a wheel scroll away from it must survive the next key or mouse move.
func (s *Sheet) follow() {
	if !s.shown(s.active) {
		return
	}
	p := s.parts[s.active]
	if p.Cursor == nil {
		return
	}
	row := p.Cursor()
	if row < 0 {
		return
	}
	if now := [2]int{s.active, row}; now == s.followed {
		return
	} else {
		s.followed = now
	}
	at := p.top + row
	if at < s.pan.Y {
		s.pan.Y = at
	}
	if s.paneH > 0 && at >= s.pan.Y+s.paneH {
		s.pan.Y = at - s.paneH + 1
	}
	s.clampPan()
}

func (s *Sheet) partAt(x, y int) int {
	for i, p := range s.parts {
		if s.shown(i) && p.rect.Contains(x, y) {
			return i
		}
	}
	return -1
}

func (s *Sheet) HandleEvent(ev tcell.Event) {
	switch e := ev.(type) {
	case *tcell.EventMouse:
		s.handleMouse(e)
	case *tcell.EventKey:
		if s.shown(s.active) {
			s.parts[s.active].Widget.HandleEvent(ev)
		}
	}
	s.follow()
}

func (s *Sheet) handleMouse(e *tcell.EventMouse) {
	b := e.Buttons()
	x, y := e.Position()
	at := s.partAt(x, y)
	sideways := b&(tcell.WheelLeft|tcell.WheelRight) != 0
	ctrlWheel := b&(tcell.WheelUp|tcell.WheelDown) != 0 && e.Modifiers()&tcell.ModCtrl != 0
	switch {
	case (sideways || ctrlWheel) && at >= 0:
		s.parts[at].Widget.HandleEvent(e)
	case b&tcell.WheelUp != 0:
		s.ScrollBy(-1)
	case b&tcell.WheelDown != 0:
		s.ScrollBy(1)
	case b == tcell.ButtonNone:
		for i, p := range s.parts {
			if s.shown(i) {
				p.Widget.HandleEvent(e)
			}
		}
	case b&tcell.ButtonPrimary != 0 && at >= 0:
		if s.parts[at].Keys {
			s.active = at
		}
		s.parts[at].Widget.HandleEvent(e)
	case at >= 0:
		s.parts[at].Widget.HandleEvent(e)
	}
}

// HandleFocusKey: Tab moves the keys to the next part that takes them; other keys go to the
// active part. The sheet follows the part's cursor.
func (s *Sheet) HandleFocusKey(ev *tcell.EventKey) bool {
	if ev.Key() == tcell.KeyTab {
		s.active = s.nextStop(s.active)
		s.follow()
		return true
	}
	ok := false
	if s.shown(s.active) {
		if h, is := s.parts[s.active].Widget.(FocusKeyHandler); is {
			ok = h.HandleFocusKey(ev)
		}
	}
	s.follow()
	return ok
}

func (s *Sheet) SetFocused(focused bool) {
	s.BaseWidget.SetFocused(focused)
	for _, p := range s.parts {
		if f, ok := p.Widget.(Focusable); ok {
			f.SetFocused(focused)
		}
	}
}
