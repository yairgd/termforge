package termforge

import "github.com/gdamore/tcell/v2"

// WidgetsList is the flat Layout an App uses for its chrome, the counterpart of
// WidgetTree one level up: full-width rows stacked top to bottom, plus floating
// widgets that get their rect from a callback.
//
// Geometry comes from how a widget was registered, so an app declares its
// banding once in setup and never assigns rects on resize:
//
//	AddWidget(tab)                 // fills the rows the fixed ones leave over
//	AddRowWidget(completionBar, 1) // fixed-height band, in registration order
//	AddRowWidget(cmdWidget, 1)
//	AddFloatingWidget(help, helpRect)
//
// Widgets are drawn in registration order, so a floating widget registered last
// paints over the rows.
type WidgetsList struct {
	widgets []widgetNode
}

var _ Layout = (*WidgetsList)(nil)

// widgetNode is one entry: the widget, how it is placed, and the rect the last
// BuildLayout gave it.
type widgetNode struct {
	widget Widget
	rect   Rect
	rows   int               // fixed height in rows; 0 means "fill"
	rectFn func(Canvas) Rect // floating widgets only
}

// AddWidget appends a widget that fills the rows the fixed-height ones leave
// over. With several of them the free rows are split evenly.
func (l *WidgetsList) AddWidget(w Widget) {
	l.widgets = append(l.widgets, widgetNode{widget: w})
}

// AddRowWidget appends a full-width band of exactly rows rows (cmdline, wildmenu).
func (l *WidgetsList) AddRowWidget(w Widget, rows int) {
	if rows < 1 {
		rows = 1
	}
	l.widgets = append(l.widgets, widgetNode{widget: w, rows: rows})
}

// AddFloatingWidget appends a widget placed by rect on every layout pass, over
// the row stack and outside it (help window, popups). A zero Rect hides it.
func (l *WidgetsList) AddFloatingWidget(w Widget, rect func(Canvas) Rect) {
	l.widgets = append(l.widgets, widgetNode{widget: w, rectFn: rect})
}

// BuildLayout assigns every widget its rect for this canvas.
func (l *WidgetsList) BuildLayout(c Canvas) {
	fixed, fills := 0, 0
	for _, n := range l.widgets {
		switch {
		case n.rectFn != nil:
		case n.rows > 0:
			fixed += n.rows
		default:
			fills++
		}
	}

	free := c.H() - fixed
	if free < 0 {
		free = 0
	}

	y := 0
	for i := range l.widgets {
		n := &l.widgets[i]
		if n.rectFn != nil {
			n.rect = n.rectFn(c)
			continue
		}
		h := n.rows
		if h == 0 {
			h = free / fills
			free -= h
			fills--
		}
		if y+h > c.H() {
			h = c.H() - y
		}
		if h < 0 {
			h = 0
		}
		n.rect = c.ChildRect(0, y, c.W(), h)
		y += h
	}
}

func (l *WidgetsList) Draw(c Canvas) {
	for _, n := range l.widgets {
		n.widget.Draw(Canvas{rect: n.rect, grid: c.grid})
	}
}

func (l *WidgetsList) HandleEvent(ev tcell.Event) {
	for _, n := range l.widgets {
		n.widget.HandleEvent(ev)
	}
}

// Rect returns the rect the last BuildLayout gave w, or the zero Rect when w is
// not registered. A zero Rect contains no point, so hit tests fail closed.
func (l *WidgetsList) Rect(w Widget) Rect {
	for _, n := range l.widgets {
		if n.widget == w {
			return n.rect
		}
	}
	return Rect{}
}

// ForEach calls fn for every registered widget in registration order.
func (l *WidgetsList) ForEach(fn func(w Widget)) {
	for _, n := range l.widgets {
		fn(n.widget)
	}
}
