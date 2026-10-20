//go:build lua

package demo

import (
	tcell "github.com/gdamore/tcell/v2"

	"github.com/yairgd/termforge"
	"github.com/yairgd/termforge/lua"
	"github.com/yairgd/termforge/ptyx"
)

const luaScrollback = 5000

// LuaPane shows the embedded Lua REPL: an xterm view on a PTY whose slave Lua reads and
// writes (lua.Run). Line editing and history are the interpreter's own.
type LuaPane struct {
	termforge.BaseWidget
	term *termforge.CompositeTerminal
	tty  *ptyx.TTY
}

// NewLuaPane opens the PTY and wires it to a terminal view; postFrame repaints the UI.
func NewLuaPane(name string, postFrame func()) (*LuaPane, error) {
	tty, err := ptyx.Open()
	if err != nil {
		return nil, err
	}
	t := termforge.NewCompositeTerminal(80, 24, luaScrollback)
	t.AttachTTY(tty, termforge.WireTTYOpts{PostFrame: postFrame})
	return &LuaPane{BaseWidget: termforge.BaseWidget{PaneName: name}, term: t, tty: tty}, nil
}

// Slave is the PTY path to give lua.Run.
func (p *LuaPane) Slave() string { return p.tty.SlaveName() }

func (p *LuaPane) SetClipboard(io termforge.ClipboardIO) { p.term.SetClipboard(io) }

func (p *LuaPane) SetMouseOrigin(screenX, screenY int) { p.term.SetMouseOrigin(screenX, screenY) }

func (p *LuaPane) Draw(c termforge.Canvas) { p.term.Paint(c, p.Focused()) }

func (p *LuaPane) HandleEvent(ev tcell.Event) {
	switch e := ev.(type) {
	case *tcell.EventMouse:
		p.term.HandleMouse(e)
	case *tcell.EventClipboard:
		p.term.PasteBytes(e.Data())
	case *tcell.EventKey:
		// The PTY has no foreground process to signal, so Ctrl-C stops the running
		// chunk through Lua (unless it is copying a selection).
		if e.Key() == tcell.KeyCtrlC && !p.term.HasSelection() {
			lua.Interrupt()
			return
		}
		p.term.HandleKey(e)
	}
}

func (p *LuaPane) Close() {
	p.term.Close()
	p.tty.Close()
}

// Note prints an app message in the terminal, on its own lines.
func (p *LuaPane) Note(text string) { p.term.WriteRaw("\r\n" + text + "\r\n") }
