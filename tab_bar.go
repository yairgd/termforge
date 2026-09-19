package termforge

import (
	"strconv"

	tcell "github.com/gdamore/tcell/v2"
)

// TabBarWidget paints the tab titles on one full-width chrome row. It is a
// band, not a pane: register it with AddRowWidget before the workspace widget
// so it takes the top row, and it draws no status line.
//
// The bar reads the titles and the active index off the TabWidget on every
// paint, so adding a tab or switching one in needs nothing kept in sync here.
type TabBarWidget struct {
	tabs *TabWidget
}

var _ Widget = (*TabBarWidget)(nil)

// NewTabBarWidget builds the header for tabs.
func NewTabBarWidget(tabs *TabWidget) *TabBarWidget {
	return &TabBarWidget{tabs: tabs}
}

// HandleEvent ignores input: the bar has no rect of its own to hit-test with,
// so the host routes clicks through WidgetRect and TabAt.
func (w *TabBarWidget) HandleEvent(ev tcell.Event) {}

// tabLabel is the cells one tab occupies: " 2 columns ".
func tabLabel(index int, title string) string {
	if title == "" {
		title = "tab"
	}
	return " " + strconv.Itoa(index+1) + " " + title + " "
}

// TabAt maps a column of the bar to a tab index, or -1 for the empty run past
// the last label.
func (w *TabBarWidget) TabAt(localX int) int {
	if w == nil || w.tabs == nil || localX < 0 {
		return -1
	}
	col := 0
	for i, title := range w.tabs.Titles() {
		col += len([]rune(tabLabel(i, title)))
		if localX < col {
			return i
		}
	}
	return -1
}

func tabBarStyle() tcell.Style {
	return tcell.StyleDefault.
		Foreground(tcell.ColorSilver).
		Background(tcell.ColorBlack)
}

func tabBarActiveStyle() tcell.Style {
	return tcell.StyleDefault.
		Foreground(tcell.ColorWhite).
		Background(tcell.ColorNavy).
		Bold(true)
}

func (w *TabBarWidget) Draw(c Canvas) {
	if w == nil || w.tabs == nil || c.W() <= 0 || c.H() <= 0 {
		return
	}
	inactive := tabBarStyle()
	c.ClearLine(0, inactive)

	col := 0
	for i, title := range w.tabs.Titles() {
		style := inactive
		if i == w.tabs.ActiveIndex() {
			style = tabBarActiveStyle()
		}
		for _, ch := range tabLabel(i, title) {
			if col >= c.W() {
				return
			}
			c.SetContent(col, 0, ch, style)
			col++
		}
	}
}
