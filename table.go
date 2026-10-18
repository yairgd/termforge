package termforge

import (
	"strings"
	"unicode/utf8"

	tcell "github.com/gdamore/tcell/v2"
)

const defaultTableGutter = 1

// TableColumn describes one table column.
type TableColumn struct {
	Name  string
	Width int // 0 = auto from header + cells
}

// TableCell is one cell's text and optional style.
type TableCell struct {
	Text  string
	Style tcell.Style
}

// Table is plain data: column definitions, rows and a few display flags.
// TableWidget does all the painting.
type Table struct {
	title      string
	showTitle  bool
	showHeader bool
	gutter     int
	columns    []TableColumn
	rows       [][]TableCell
}

func NewTable() *Table { return &Table{gutter: defaultTableGutter} }

func (t *Table) SetTitle(s string)       { t.title = s }
func (t *Table) SetShowTitle(show bool)  { t.showTitle = show }
func (t *Table) SetShowHeader(show bool) { t.showHeader = show }

func (t *Table) SetGutter(n int) {
	if n < 0 {
		n = 0
	}
	t.gutter = n
}

func (t *Table) Title() string    { return t.title }
func (t *Table) ShowTitle() bool  { return t.showTitle }
func (t *Table) ShowHeader() bool { return t.showHeader }
func (t *Table) Gutter() int      { return t.gutter }

func (t *Table) AddColumn(name string) int { return t.AddColumnWidth(name, 0) }

func (t *Table) AddColumnWidth(name string, width int) int {
	t.columns = append(t.columns, TableColumn{Name: name, Width: width})
	return len(t.columns) - 1
}

func (t *Table) NumCols() int { return len(t.columns) }

// ColumnName is column i's header, or "" out of range.
func (t *Table) ColumnName(i int) string {
	if i < 0 || i >= len(t.columns) {
		return ""
	}
	return t.columns[i].Name
}
func (t *Table) NumRows() int { return len(t.rows) }

func (t *Table) ClearRows() { t.rows = t.rows[:0] }

// ClearColumns drops the columns and the rows with them.
func (t *Table) ClearColumns() { t.columns, t.rows = nil, nil }

func (t *Table) AddRow(values ...string) {
	cells := make([]TableCell, len(values))
	for i, v := range values {
		cells[i] = TableCell{Text: v}
	}
	t.AddRowCells(cells...)
}

// AddRowCells pads or truncates the row to the number of columns.
func (t *Table) AddRowCells(cells ...TableCell) {
	row := make([]TableCell, len(t.columns))
	copy(row, cells)
	t.rows = append(t.rows, row)
}

// CellText returns the full text of one data cell.
func (t *Table) CellText(row, col int) string {
	if row < 0 || row >= len(t.rows) || col < 0 || col >= len(t.rows[row]) {
		return ""
	}
	return t.rows[row][col].Text
}

// RowDisplayLine joins a row's columns with two spaces (copy format).
func (t *Table) RowDisplayLine(row int) string {
	if row < 0 || row >= len(t.rows) {
		return ""
	}
	parts := make([]string, len(t.rows[row]))
	for i, c := range t.rows[row] {
		parts[i] = c.Text
	}
	return strings.Join(parts, "  ")
}

// colWidths returns each column's width: the fixed Width, or the longest of
// the header and the cells.
func (t *Table) colWidths() []int {
	ws := make([]int, len(t.columns))
	for i, col := range t.columns {
		w := col.Width
		if w <= 0 {
			w = runeLen(col.Name)
			for _, row := range t.rows {
				if n := runeLen(row[i].Text); n > w {
					w = n
				}
			}
		}
		ws[i] = w
	}
	return ws
}

func runeLen(s string) int { return utf8.RuneCountInString(s) }

func truncateRunes(s string, max int) string {
	if max <= 0 {
		return ""
	}
	rs := []rune(s)
	if len(rs) <= max {
		return s
	}
	if max == 1 {
		return "…"
	}
	return string(rs[:max-1]) + "…"
}
