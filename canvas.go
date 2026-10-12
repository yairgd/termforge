package termforge

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
)

type Canvas struct {
	rect Rect
	// clip is the visible screen area (rect ∩ every parent's clip); writes
	// outside it are dropped. Unused until hasClip, so a bare Canvas{rect, grid}
	// keeps clipping to the screen only.
	clip    Rect
	hasClip bool
	grid    *Grid
	// origin is the content cell drawn at local (0,0); pan by changing it.
	origin Point
}

func NewCanvas(g *Grid) *Canvas {
	return &Canvas{
		grid: g,
	}
}

func (c Canvas) Rect() Rect { return c.rect }
func (c Canvas) W() int     { return c.rect.w }
func (c Canvas) H() int     { return c.rect.h }

func (c Canvas) ScreenX(x int) int { return c.rect.x + x - c.origin.X }
func (c Canvas) ScreenY(y int) int { return c.rect.y + y - c.origin.Y }

func (c Canvas) ChildRect(x, y, w, h int) Rect {
	return NewRect(
		c.ScreenX(x),
		c.ScreenY(y),
		w,
		h,
	)
}

// WithRect returns a child canvas on r, clipped to this canvas's visible area.
// The child starts with a zero origin.
func (c Canvas) WithRect(r Rect) Canvas {
	clip := r
	if c.hasClip {
		clip = intersectRect(c.clip, r)
	}
	return Canvas{rect: r, clip: clip, hasClip: true, grid: c.grid}
}

// WithOrigin returns a copy that draws content (x,y) at local (x-p.X, y-p.Y).
func (c Canvas) WithOrigin(p Point) Canvas {
	c.origin = p
	return c
}

func intersectRect(a, b Rect) Rect {
	x, y := max(a.x, b.x), max(a.y, b.y)
	r, bt := min(a.Right(), b.Right()), min(a.Bottom(), b.Bottom())
	return NewRect(x, y, max(r-x, 0), max(bt-y, 0))
}

func (c Canvas) SetContent(
	x,
	y int,
	ch rune,
	style tcell.Style,
) {
	sx, sy := c.ScreenX(x), c.ScreenY(y)
	if c.hasClip && !c.clip.Contains(sx, sy) {
		return
	}
	c.grid.SetContent(sx, sy, ch, style)
}

func (c Canvas) printClipped(x, y int, style tcell.Style, text string) {
	for _, r := range text {
		c.SetContent(x, y, r, style)
		x++
	}
}

func (c Canvas) Fill(
	ch rune,
	style tcell.Style,
) {
	for y := 0; y < c.H(); y++ {
		for x := 0; x < c.W(); x++ {
			c.SetContent(x, y, ch, style)
		}
	}
}

func (c Canvas) Print(
	x,
	y int,
	style tcell.Style,
	args ...any,
) {
	c.printClipped(x, y, style, fmt.Sprint(args...))
}

func (c Canvas) Printf(
	x,
	y int,
	style tcell.Style,
	format string,
	args ...any,
) {
	c.printClipped(x, y, style, fmt.Sprintf(format, args...))
}

func (c Canvas) DrawVerticalLocal(
	x,
	y1,
	y2 int,
	bold bool,
) {
	c.grid.DrawVertical(
		c.ScreenX(x),
		c.ScreenY(y1),
		c.ScreenY(y2),
		bold,
	)
}

func (c Canvas) DrawHorizontalLocal(
	y,
	x1,
	x2 int,
	bold bool,
) {
	c.grid.DrawHorizontal(
		c.ScreenY(y),
		c.ScreenX(x1),
		c.ScreenX(x2),
		bold,
	)
}

func (c Canvas) ClearLineRange(localY, x1, x2 int, style tcell.Style) {
	if x1 < 0 {
		x1 = 0
	}
	if x2 > c.W() {
		x2 = c.W()
	}

	for x := x1; x < x2; x++ {
		c.SetContent(x, localY, ' ', style)
	}
}

func (c Canvas) ClearLine(localY int, style tcell.Style) {
	// Only clear this canvas's columns — a full-grid clear would wipe sibling
	// panes after :vs / nested splits.
	c.ClearLineRange(localY, 0, c.W(), style)
}

func (c Canvas) ShowCursor(localX, localY int) {
	c.grid.ShowCursor(
		c.ScreenX(localX),
		c.ScreenY(localY),
	)
}

func (c Canvas) ShowNativeCursor(localX, localY int) {
	c.ShowNativeCursorStyle(localX, localY, 0)
}

func (c Canvas) ShowNativeCursorStyle(localX, localY int, style tcell.CursorStyle) {
	c.grid.ShowNativeCursorStyle(
		c.ScreenX(localX),
		c.ScreenY(localY),
		style,
	)
}

func (c Canvas) HideCursor() {
	c.grid.HideCursor()
}
