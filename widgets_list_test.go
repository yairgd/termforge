package termforge

import (
	"testing"

	tcell "github.com/gdamore/tcell/v2"
)

type chromeWidget struct{ drawn Rect }

func (w *chromeWidget) HandleEvent(tcell.Event) {}
func (w *chromeWidget) Draw(c Canvas)           { w.drawn = c.Rect() }

// The app chrome banding every host uses: a workspace filling what the cmdline
// and the wildmenu row leave over, plus a floating window over both.
func TestWidgetsListBuildsChromeBanding(t *testing.T) {
	workspace, bar, cmd, float := &chromeWidget{}, &chromeWidget{}, &chromeWidget{}, &chromeWidget{}

	var l WidgetsList
	l.AddWidget(workspace)
	l.AddRowWidget(bar, 1)
	l.AddRowWidget(cmd, 1)
	l.AddFloatingWidget(float, func(c Canvas) Rect {
		return c.ChildRect(2, 3, 10, 4)
	})

	l.BuildLayout(Canvas{rect: NewRect(0, 0, 80, 24)})

	want := map[string]Rect{
		"workspace": NewRect(0, 0, 80, 22),
		"bar":       NewRect(0, 22, 80, 1),
		"cmd":       NewRect(0, 23, 80, 1),
		"float":     NewRect(2, 3, 10, 4),
	}
	for name, got := range map[string]Rect{
		"workspace": l.Rect(workspace),
		"bar":       l.Rect(bar),
		"cmd":       l.Rect(cmd),
		"float":     l.Rect(float),
	} {
		if got != want[name] {
			t.Errorf("%s rect = %v, want %v", name, got, want[name])
		}
	}
}

// A console shorter than the fixed rows must not hand anyone a negative or
// off-canvas rect: the fill collapses first, then the rows are clamped.
func TestWidgetsListClampsToShortCanvas(t *testing.T) {
	workspace, bar, cmd := &chromeWidget{}, &chromeWidget{}, &chromeWidget{}

	var l WidgetsList
	l.AddWidget(workspace)
	l.AddRowWidget(bar, 1)
	l.AddRowWidget(cmd, 1)

	l.BuildLayout(Canvas{rect: NewRect(0, 0, 20, 1)})

	if r := l.Rect(workspace); r.H() != 0 {
		t.Errorf("workspace height = %d, want 0", r.H())
	}
	for _, r := range []Rect{l.Rect(bar), l.Rect(cmd)} {
		if r.H() < 0 || r.Bottom() > 1 {
			t.Errorf("row rect %v leaves the 1-row canvas", r)
		}
	}
}

func TestWidgetsListRectOfUnregisteredWidgetIsEmpty(t *testing.T) {
	var l WidgetsList
	l.AddWidget(&chromeWidget{})
	l.BuildLayout(Canvas{rect: NewRect(0, 0, 80, 24)})

	if r := l.Rect(&chromeWidget{}); r.Contains(0, 0) {
		t.Errorf("unregistered widget got rect %v, want one that hit tests fail on", r)
	}
}

func TestWidgetsListDrawsIntoAssignedRect(t *testing.T) {
	workspace, cmd := &chromeWidget{}, &chromeWidget{}

	var l WidgetsList
	l.AddWidget(workspace)
	l.AddRowWidget(cmd, 1)

	c := Canvas{rect: NewRect(0, 0, 40, 10), grid: NewGrid(40, 10)}
	l.BuildLayout(c)
	l.Draw(c)

	if workspace.drawn != NewRect(0, 0, 40, 9) {
		t.Errorf("workspace drawn into %v, want 0,0 40x9", workspace.drawn)
	}
	if cmd.drawn != NewRect(0, 9, 40, 1) {
		t.Errorf("cmdline drawn into %v, want 0,9 40x1", cmd.drawn)
	}
}
