package termforge

import (
	"strings"
	"time"

	tcell "github.com/gdamore/tcell/v2"
	"github.com/yairgd/termforge/platform"
)

// TableFillFunc repopulates table rows at Draw time (e.g. from external model data).
type TableFillFunc func(t *Table)

type cellRef struct{ row, col int }

// TableWidget is a columnar table view: row selection, horizontal pan, /search,
// double-click copy. Draw does all the layout and painting.
type TableWidget struct {
	BaseWidget
	table      *Table
	fill       TableFillFunc
	rowStyleFn func(row int) tcell.Style

	selectedRow int
	top         int // first visible data row
	panX        int // first visible content column

	searchPattern   string
	searchCommitted string
	searchColor     tcell.Color
	selectColor     tcell.Color
	onSearchJump    func(row int)

	clipboard       ClipboardIO
	cellSel         *cellRef
	cellSelText     string
	suppressCellClk bool
	clickCount      int
	lastClickTime   time.Time
	lastClickRow    int
	lastClickCol    int

	// set by Draw, used by hit-testing and paging
	screenX, screenY int
	lastW, lastH     int
	colW             []int
	contentW         int
}

var (
	_ Widget          = (*TableWidget)(nil)
	_ FocusKeyHandler = (*TableWidget)(nil)
	_ Focusable       = (*TableWidget)(nil)
	_ SearchHost      = (*TableWidget)(nil)
)

func NewTableWidget(ctx platform.AppContext) *TableWidget {
	return &TableWidget{
		BaseWidget:  NewBaseWidget(ctx),
		table:       NewTable(),
		selectColor: tcell.ColorDarkSlateBlue,
	}
}

func (w *TableWidget) Table() *Table { return w.table }

func (w *TableWidget) SetFill(fn TableFillFunc) { w.fill = fn }

func (w *TableWidget) SetRowStyleFunc(fn func(row int) tcell.Style) { w.rowStyleFn = fn }

func (w *TableWidget) SetOnSearchJump(fn func(row int)) { w.onSearchJump = fn }

func (w *TableWidget) SetClipboard(io ClipboardIO) { w.clipboard = io }

func (w *TableWidget) SelectedRow() int { return w.selectedRow }

// refill reruns the fill callback so row counts are current.
func (w *TableWidget) refill() {
	if w.fill != nil {
		w.table.ClearRows()
		w.fill(w.table)
	}
	w.clampSelected()
}

func (w *TableWidget) clampSelected() {
	w.selectedRow = clampInt(w.selectedRow, 0, max(w.table.NumRows()-1, 0))
}

// ---- view geometry ----

// stickyRows is the title + header rows pinned above the data.
func (w *TableWidget) stickyRows() int {
	n := 0
	if w.hasTitle() {
		n++
	}
	if w.hasHeader() {
		n++
	}
	return n
}

func (w *TableWidget) hasTitle() bool  { return w.table.showTitle && w.table.title != "" }
func (w *TableWidget) hasHeader() bool { return w.table.showHeader && w.table.NumCols() > 0 }

func (w *TableWidget) dataH() int { return max(w.lastH-w.stickyRows(), 0) }

func (w *TableWidget) PageRows() int {
	if h := w.dataH(); h >= 1 {
		return h
	}
	return 10
}

func (w *TableWidget) clampView() {
	w.top = clampInt(w.top, 0, max(w.table.NumRows()-w.dataH(), 0))
	w.panX = clampInt(w.panX, 0, max(w.contentW-w.lastW, 0))
}

// HorizontalPan is the first content column drawn at the pane's left edge.
func (w *TableWidget) HorizontalPan() int { return w.panX }

// ResetHorizontalPan scrolls back to the leftmost column. Call it when the row
// model is replaced.
func (w *TableWidget) ResetHorizontalPan() { w.panX = 0 }

func (w *TableWidget) PanLeft()  { w.pan(-1) }
func (w *TableWidget) PanRight() { w.pan(1) }

func (w *TableWidget) pan(dx int) {
	w.panX += dx
	w.clampView()
}

// ---- selection ----

func (w *TableWidget) SetSelectedRow(row int) {
	w.refill()
	w.selectedRow = row
	w.clampSelected()
}

func (w *TableWidget) MoveSelection(delta int) {
	w.refill()
	n := w.table.NumRows()
	if n == 0 {
		return
	}
	w.selectedRow = (w.selectedRow + delta%n + n) % n
	w.EnsureRowVisible()
}

func (w *TableWidget) EnsureRowVisible() {
	h := w.dataH()
	if w.selectedRow < w.top {
		w.top = w.selectedRow
	}
	if h > 0 && w.selectedRow >= w.top+h {
		w.top = w.selectedRow - h + 1
	}
	w.clampView()
}

// InitSelectionKeyBindings wires Up/Down for row selection and Left/Right for pan.
func (w *TableWidget) InitSelectionKeyBindings() {
	w.BindKeyFunc("up", func(args ...any) { w.MoveSelection(-1) }, "<Up>", "k")
	w.BindKeyFunc("down", func(args ...any) { w.MoveSelection(1) }, "<Down>", "j")
	w.BindKeyFunc("scroll-left", func(args ...any) { w.pan(-1) }, "<Left>", "h")
	w.BindKeyFunc("scroll-right", func(args ...any) { w.pan(1) }, "<Right>", "l")
	w.BindKeyFunc("page-up", func(args ...any) { w.MoveSelection(-w.PageRows()) }, "<PgUp>", "<C-b>")
	w.BindKeyFunc("page-down", func(args ...any) { w.MoveSelection(w.PageRows()) }, "<PgDn>", "<C-f>")
	w.BindKeyFunc("home", func(args ...any) { w.SetSelectedRow(0); w.EnsureRowVisible() }, "<Home>", "g")
	w.BindKeyFunc("end", func(args ...any) {
		w.SetSelectedRow(w.table.NumRows() - 1)
		w.EnsureRowVisible()
	}, "<End>", "G")
}

// ---- events ----

func (w *TableWidget) HandleFocusKey(ev *tcell.EventKey) bool { return w.HandleBoundKey(ev) }

func (w *TableWidget) HandleEvent(ev tcell.Event) {
	switch e := ev.(type) {
	case *tcell.EventKey:
		w.HandleBoundKey(e)
	case *tcell.EventMouse:
		w.handleMouse(e)
	}
}

func (w *TableWidget) handleMouse(e *tcell.EventMouse) {
	if w.TryDoubleClickWord(e) {
		return
	}
	btns := e.Buttons()
	switch {
	case btns&tcell.WheelUp != 0:
		w.MoveSelection(-1)
	case btns&tcell.WheelDown != 0:
		w.MoveSelection(1)
	case btns&tcell.ButtonPrimary != 0:
		if row, ok := w.HitDataRow(e.Position()); ok {
			w.SetSelectedRow(row)
		}
	}
}

// HitDataRow maps screen coordinates to a data row index.
func (w *TableWidget) HitDataRow(mx, my int) (row int, onRow bool) {
	lx, ly := mx-w.screenX, my-w.screenY
	if lx < 0 || lx >= w.lastW || ly < w.stickyRows() || ly >= w.lastH {
		return -1, false
	}
	row = w.top + ly - w.stickyRows()
	if row >= w.table.NumRows() {
		return -1, false
	}
	return row, true
}

// HitCell maps screen coordinates to a data row, column, and rune offset within
// that column's visible (possibly truncated) cell width.
func (w *TableWidget) HitCell(mx, my int) (row, col, runeOff int, ok bool) {
	row, onRow := w.HitDataRow(mx, my)
	if !onRow {
		return -1, -1, 0, false
	}
	contentX := mx - w.screenX + w.panX
	x := 0
	for c, cw := range w.colW {
		if c > 0 {
			x += w.table.gutter
		}
		if contentX >= x && contentX < x+cw {
			return row, c, contentX - x, true
		}
		x += cw
	}
	return row, -1, 0, false
}

// TryDoubleClickWord handles double-click on a cell: highlight the full cell
// text and copy it to the clipboard. Returns true when the event is consumed
// (second click of a double-click, or while holding after one).
func (w *TableWidget) TryDoubleClickWord(e *tcell.EventMouse) bool {
	if e == nil {
		return false
	}
	btns := e.Buttons()
	if btns == tcell.ButtonNone {
		w.suppressCellClk = false
		return false
	}
	if w.suppressCellClk {
		return true
	}
	if btns&tcell.ButtonPrimary == 0 {
		return false
	}
	row, col, _, ok := w.HitCell(e.Position())
	if !ok {
		w.ClearCellSelection()
		return false
	}
	w.noteCellClick(row, col, e.When())
	if w.clickCount == 2 {
		text := w.table.CellText(row, col)
		if text == "" {
			return false
		}
		w.cellSel = &cellRef{row, col}
		w.cellSelText = text
		w.clipboard.copyText(text)
		w.SetSelectedRow(row)
		w.suppressCellClk = true
		return true
	}
	w.ClearCellSelection()
	return false
}

func (w *TableWidget) noteCellClick(row, col int, when time.Time) {
	if when.IsZero() {
		when = time.Now()
	}
	same := row == w.lastClickRow && col == w.lastClickCol
	if same && when.Sub(w.lastClickTime) <= clickMultiTimeoutMs*time.Millisecond {
		w.clickCount++
		if w.clickCount > 3 {
			w.clickCount = 1
		}
	} else {
		w.clickCount = 1
	}
	w.lastClickTime, w.lastClickRow, w.lastClickCol = when, row, col
}

// ClearCellSelection drops the highlighted cell text (e.g. before a new single click).
func (w *TableWidget) ClearCellSelection() {
	w.cellSel = nil
	w.cellSelText = ""
}

// ---- copy ----

func (w *TableWidget) CopySelection() {
	if w.clipboard.Copy != nil {
		w.clipboard.copyText(w.SelectedText())
	}
}

func (w *TableWidget) SelectedText() string {
	if w.cellSelText != "" {
		return w.cellSelText
	}
	return w.table.RowDisplayLine(w.selectedRow)
}

func (w *TableWidget) HasSelection() bool { return w.cellSel != nil }

// ---- search ----

func (w *TableWidget) SetSearchColor(c tcell.Color) { w.searchColor = c }

func (w *TableWidget) SetSearchPattern(pattern string) { w.searchPattern = pattern }

func (w *TableWidget) RevertSearch() { w.searchPattern = w.searchCommitted }

func (w *TableWidget) SearchPattern() string { return w.searchPattern }

func (w *TableWidget) CommitSearch(pattern string) {
	w.searchPattern = pattern
	w.searchCommitted = pattern
	if pattern == "" {
		return
	}
	if w.rowMatches(w.selectedRow) {
		w.jumpToSearchRow(w.selectedRow)
		return
	}
	w.SearchNext()
}

func (w *TableWidget) SearchNext() bool { return w.searchStep(1) }
func (w *TableWidget) SearchPrev() bool { return w.searchStep(-1) }

// searchStep selects the next row (wrapping) after the selection, in direction dir, that matches.
func (w *TableWidget) searchStep(dir int) bool {
	n := w.table.NumRows()
	if n == 0 || w.searchPattern == "" {
		return false
	}
	for i := 1; i <= n; i++ {
		row := ((w.selectedRow+dir*i)%n + n) % n
		if w.rowMatches(row) {
			w.jumpToSearchRow(row)
			return true
		}
	}
	return false
}

func (w *TableWidget) jumpToSearchRow(row int) {
	w.SetSelectedRow(row)
	w.EnsureRowVisible()
	if w.onSearchJump != nil {
		w.onSearchJump(row)
	}
}

func (w *TableWidget) rowMatches(row int) bool {
	if w.searchPattern == "" || row < 0 || row >= w.table.NumRows() {
		return false
	}
	pat := strings.ToLower(w.searchPattern)
	for _, c := range w.table.rows[row] {
		if strings.Contains(strings.ToLower(c.Text), pat) {
			return true
		}
	}
	return false
}

func (w *TableWidget) CursorInSearchMatch() bool { return w.rowMatches(w.selectedRow) }

func (w *TableWidget) WordAtCursor() string {
	if w.selectedRow >= w.table.NumRows() {
		return ""
	}
	return identAtOrNear(w.table.RowDisplayLine(w.selectedRow), 0)
}

// matchSpans returns the [start,end) rune ranges of text matching the search pattern.
func (w *TableWidget) matchSpans(text string) [][2]int {
	if w.searchPattern == "" || text == "" {
		return nil
	}
	rs := []rune(strings.ToLower(text))
	pat := []rune(strings.ToLower(w.searchPattern))
	var spans [][2]int
	for i := 0; i+len(pat) <= len(rs); i++ {
		if string(rs[i:i+len(pat)]) == string(pat) {
			spans = append(spans, [2]int{i, i + len(pat)})
		}
	}
	return spans
}

// ---- draw ----

func (w *TableWidget) Draw(c Canvas) {
	w.refill()
	w.lastW, w.lastH = c.W(), c.H()
	w.screenX, w.screenY = c.ScreenX(0), c.ScreenY(0)

	t := w.table
	w.colW = t.colWidths()
	w.contentW = 0
	for _, cw := range w.colW {
		w.contentW += cw
	}
	if n := len(w.colW); n > 1 {
		w.contentW += t.gutter * (n - 1)
	}
	if w.hasTitle() {
		w.contentW = max(w.contentW, runeLen(t.title))
	}
	w.clampView()

	for y := 0; y < w.lastH; y++ {
		for x := 0; x < w.lastW; x++ {
			c.SetContent(x, y, ' ', tcell.StyleDefault)
		}
	}

	y := 0
	if w.hasTitle() {
		w.drawRow(c, y, -1, []TableCell{{Text: t.title}}, tcell.StyleDefault.Bold(true), []int{runeLen(t.title)})
		y++
	}
	if w.hasHeader() {
		head := make([]TableCell, len(t.columns))
		for i, col := range t.columns {
			head[i].Text = col.Name
		}
		w.drawRow(c, y, -1, head, tcell.StyleDefault.Bold(true), w.colW)
		y++
	}
	for row := w.top; row < t.NumRows() && y < w.lastH; row, y = row+1, y+1 {
		base := tcell.StyleDefault
		if w.rowStyleFn != nil {
			base = w.rowStyleFn(row)
		}
		w.drawRow(c, y, row, t.rows[row], base, w.colW)
	}
}

// drawRow paints one line at window row y. row < 0 is the title/header
// (no search or selection highlight). widths are the cell widths; gutters sit between them.
func (w *TableWidget) drawRow(c Canvas, y, row int, cells []TableCell, base tcell.Style, widths []int) {
	put := func(cx int, ch rune, st tcell.Style) {
		if sx := cx - w.panX; sx >= 0 && sx < w.lastW {
			c.SetContent(sx, y, ch, st)
		}
	}
	x := 0
	for col, cw := range widths {
		if col > 0 {
			for g := 0; g < w.table.gutter; g++ {
				put(x, ' ', base)
				x++
			}
		}
		if x+cw <= w.panX || x-w.panX >= w.lastW {
			x += cw
			continue
		}
		st, full := base, ""
		if col < len(cells) {
			full = cells[col].Text
			if cells[col].Style != (tcell.Style{}) {
				st = cells[col].Style
			}
		}
		rs := []rune(truncateRunes(full, cw))
		var spans [][2]int
		selected := false
		if row >= 0 {
			spans = w.matchSpans(full)
			selected = w.cellSel != nil && w.cellSel.row == row && w.cellSel.col == col
		}
		for i := 0; i < cw; i++ {
			ch := ' '
			if i < len(rs) {
				ch = rs[i]
			}
			s := st
			if selected && i < len(rs) {
				s = highlight(s, w.selectColor)
			}
			for _, sp := range spans {
				if i >= sp[0] && i < sp[1] {
					s = highlight(s, w.searchColor)
					break
				}
			}
			put(x+i, ch, s)
		}
		x += cw
	}
}

func highlight(st tcell.Style, bg tcell.Color) tcell.Style {
	if bg == tcell.ColorDefault {
		return st
	}
	return st.Background(bg).Foreground(platform.ContrastColor(bg))
}

func (w *TableWidget) DrawStatusLine(c Canvas, active bool) {
	w.BaseWidget.DrawStatusLine(c, active)
}
