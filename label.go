package termforge

import "github.com/gdamore/tcell/v2"

// Label is one line of text that can be clicked. It is a leaf painted into a
// child canvas (usually from a list pane that stacks labels and pans them), not a
// tree pane: the tree consumes mouse presses, so the owner calls Click with the
// press position.
type Label struct {
	Text     string
	Style    tcell.Style
	Active   tcell.Style // used while Selected
	Selected bool
	OnClick  func()

	rect Rect // screen rect from the last Draw
}

func NewLabel(text string) *Label {
	return &Label{
		Text:   text,
		Style:  tcell.StyleDefault,
		Active: tcell.StyleDefault.Reverse(true),
	}
}

// Rect is the screen rect the label was last drawn at (zero before the first Draw).
func (l *Label) Rect() Rect { return l.rect }

// Draw fills c with the label's style, so the whole row shows and hits, then prints
// the text from its left edge.
func (l *Label) Draw(c Canvas) {
	l.rect = c.Rect()
	st := l.Style
	if l.Selected {
		st = l.Active
	}
	c.Fill(' ', st)
	c.Print(0, 0, st, l.Text)
}

// Click runs OnClick when the screen point (x, y) is on the label as last drawn.
// It reports whether it hit.
func (l *Label) Click(x, y int) bool {
	if !l.rect.Contains(x, y) {
		return false
	}
	if l.OnClick != nil {
		l.OnClick()
	}
	return true
}
