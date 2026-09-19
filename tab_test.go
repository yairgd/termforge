package termforge

import (
	"strings"
	"testing"

	tcell "github.com/gdamore/tcell/v2"
)

// twoTabs builds a TabWidget holding two trees, each with its own pane, which
// is the shape a host gets from building a second tab at startup.
func twoTabs() (*TabWidget, *recordingWidget, *recordingWidget) {
	first, second := &recordingWidget{}, &recordingWidget{}
	tabs := NewTabWidget("panes", NewWidgetTree(first))
	tabs.AddTab("columns", NewWidgetTree(second))
	return tabs, first, second
}

func TestAddTabKeepsTheActiveTab(t *testing.T) {
	tabs, _, _ := twoTabs()

	if got := tabs.Count(); got != 2 {
		t.Fatalf("Count = %d, want 2", got)
	}
	if got := tabs.ActiveIndex(); got != 0 {
		t.Errorf("AddTab moved the active tab to %d, want 0", got)
	}
	if got := strings.Join(tabs.Titles(), ","); got != "panes,columns" {
		t.Errorf("Titles = %q, want \"panes,columns\"", got)
	}
	if i := tabs.AddTab("nil content", nil); i != -1 {
		t.Errorf("AddTab(nil) returned %d, want -1", i)
	}
}

func TestTabSwitchingWraps(t *testing.T) {
	tabs, _, _ := twoTabs()

	steps := []struct {
		name   string
		move   func() bool
		want   int
		change bool
	}{
		{"next", tabs.NextTab, 1, true},
		{"next wraps", tabs.NextTab, 0, true},
		{"prev wraps", tabs.PrevTab, 1, true},
		{"prev", tabs.PrevTab, 0, true},
		{"set to the active tab", func() bool { return tabs.SetActive(0) }, 0, false},
		{"set out of range", func() bool { return tabs.SetActive(7) }, 0, false},
		{"set", func() bool { return tabs.SetActive(1) }, 1, true},
	}
	for _, s := range steps {
		if got := s.move(); got != s.change {
			t.Errorf("%s: reported changed=%v, want %v", s.name, got, s.change)
		}
		if got := tabs.ActiveIndex(); got != s.want {
			t.Errorf("%s: active tab %d, want %d", s.name, got, s.want)
		}
	}
}

// Only the active tab's layout costs anything: the other tree is neither laid
// out nor painted, so its panes keep the geometry they had when it was last up.
func TestOnlyTheActiveTabDrawsAndHandlesEvents(t *testing.T) {
	tabs, first, second := twoTabs()
	c := NewCanvas(NewGrid(40, 12)).WithRect(NewRect(0, 0, 40, 12))

	tabs.Draw(c)
	tabs.HandleEvent(tcell.NewEventKey(tcell.KeyRune, 'x', tcell.ModNone))
	if len(first.rects) == 0 {
		t.Error("first tab's pane was not drawn")
	}
	if len(second.rects) != 0 {
		t.Error("second tab's pane was drawn while inactive")
	}

	tabs.NextTab()
	tabs.Draw(c)
	if len(second.rects) == 0 {
		t.Error("second tab's pane was not drawn after switching")
	}
}

func TestTabBarHitTestMatchesLabels(t *testing.T) {
	tabs, _, _ := twoTabs()
	bar := NewTabBarWidget(tabs)

	// Labels are " 1 panes " (9 cols) then " 2 columns " (11 cols).
	cases := []struct {
		localX int
		want   int
	}{
		{-1, -1},
		{0, 0},
		{8, 0},
		{9, 1},
		{19, 1},
		{20, -1}, // empty run after the last label
		{500, -1},
	}
	for _, tc := range cases {
		if got := bar.TabAt(tc.localX); got != tc.want {
			t.Errorf("TabAt(%d) = %d, want %d", tc.localX, got, tc.want)
		}
	}
}

func TestTabBarPaintsTitlesAndMarksTheActiveOne(t *testing.T) {
	tabs, _, _ := twoTabs()
	bar := NewTabBarWidget(tabs)
	g := NewGrid(30, 3)
	bar.Draw(NewCanvas(g).WithRect(NewRect(0, 0, 30, 1)))

	var row strings.Builder
	for x := 0; x < g.W; x++ {
		row.WriteRune(g.Cells[x][0].Rune)
	}
	if got, want := row.String(), " 1 panes  2 columns           "; got != want {
		t.Fatalf("bar row = %q, want %q", got, want)
	}

	// The active label is the only run painted in the active style.
	active := tabBarActiveStyle()
	for x := 0; x < g.W; x++ {
		if got := g.Cells[x][0].Style == active; got != (x < 9) {
			t.Fatalf("column %d active style = %v, want %v", x, got, x < 9)
		}
	}
}
