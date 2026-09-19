package termforge

import (
	tcell "github.com/gdamore/tcell/v2"
)

//
// Tab model
//

// Tab is chrome for one tab entry: a title plus a Layout.
//
// Tab is a generic container. It hosts any Layout — the split tree today, any
// future layout tomorrow — and deliberately exposes no operation of its own on
// that content. Callers needing layout-specific behaviour take the Layout (or
// the concrete layout type they built) and work on it directly.
type Tab struct {
	Title   string
	Content Layout
}

//
// TabWidget
//

// TabWidget manages a list of Tabs and forwards Draw/HandleEvent to the active
// tab's Layout.
//
// A tab is a workspace preset, not a session: switching one in swaps the whole
// Layout, and nothing else. Closing a tab is not implemented, and the header is
// a separate widget — TabBarWidget paints the titles as a chrome row.
//
// Do not add methods here that forward into the Layout. Hand out the Layout
// and let the caller drive it.
type TabWidget struct {
	Widget

	tabs   []Tab
	active int
}

//
// Constructors
//

// NewTabWidget creates a TabWidget with a single tab showing content.
func NewTabWidget(title string, content Layout) *TabWidget {
	return &TabWidget{
		tabs: []Tab{
			{
				Title:   title,
				Content: content,
			},
		},
		active: 0,
	}
}

// NewTabTwoHozSplitWins creates a tab with a horizontal split: top over bottom.
func NewTabTwoHozSplitWins(title string, top NodeWidget, bottom NodeWidget) *TabWidget {
	lay := NewWidgetTree(top)
	lay.Split(Horizontal, bottom)
	return NewTabWidget(title, lay)
}

// AddTab appends a tab and returns its index, or -1 for a nil content. The
// active tab is unchanged, so a host builds every tab at startup and stays on
// the one it opened with.
func (t *TabWidget) AddTab(title string, content Layout) int {
	if t == nil || content == nil {
		return -1
	}
	t.tabs = append(t.tabs, Tab{Title: title, Content: content})
	return len(t.tabs) - 1
}

//
// Tab selection
//

// Count is how many tabs exist.
func (t *TabWidget) Count() int {
	if t == nil {
		return 0
	}
	return len(t.tabs)
}

// ActiveIndex is the tab being drawn.
func (t *TabWidget) ActiveIndex() int {
	if t == nil {
		return 0
	}
	return t.active
}

// Titles lists the tab titles in order, for a header to paint.
func (t *TabWidget) Titles() []string {
	if t == nil {
		return nil
	}
	out := make([]string, len(t.tabs))
	for i, tab := range t.tabs {
		out[i] = tab.Title
	}
	return out
}

// SetActive shows tab i, ignoring an index outside the list. It reports whether
// the active tab changed, so a caller can skip the repaint when it did not.
func (t *TabWidget) SetActive(i int) bool {
	if t == nil || i < 0 || i >= len(t.tabs) || i == t.active {
		return false
	}
	t.active = i
	return true
}

// NextTab moves one tab to the right, wrapping at the end (Vim gt).
func (t *TabWidget) NextTab() bool {
	if t == nil || len(t.tabs) < 2 {
		return false
	}
	return t.SetActive((t.active + 1) % len(t.tabs))
}

// PrevTab moves one tab to the left, wrapping at the start (Vim gT).
func (t *TabWidget) PrevTab() bool {
	if t == nil || len(t.tabs) < 2 {
		return false
	}
	return t.SetActive((t.active - 1 + len(t.tabs)) % len(t.tabs))
}

//
// Content access
//

// Layout returns the active tab's content. Callers that need arrangement
// specific operations keep the concrete layout they built, or type-assert.
func (t *TabWidget) Layout() Layout {
	if t == nil || len(t.tabs) == 0 {
		return nil
	}
	return t.tabs[t.active].Content
}

// SetLayout replaces the active tab's content (layout apply / remount).
func (t *TabWidget) SetLayout(content Layout) {
	if t == nil || len(t.tabs) == 0 || content == nil {
		return
	}
	t.tabs[t.active].Content = content
}

//
// Widget
//

// HandleEvent forwards the event to the active tab's layout.
func (t *TabWidget) HandleEvent(ev tcell.Event) {
	if l := t.Layout(); l != nil {
		l.HandleEvent(ev)
	}
}

// Draw lays out and paints the active tab's layout using the full assigned
// rect. Chrome banding (cmdline / wildmenu) is the App's WidgetsList job.
func (t *TabWidget) Draw(c Canvas) {
	l := t.Layout()
	if l == nil {
		return
	}
	l.BuildLayout(c)
	l.Draw(c)
}
