package termforge

import (
	tcell "github.com/gdamore/tcell/v2"
	"github.com/yairgd/termforge/platform"
)

type KeyHandler func(ev *tcell.EventKey) bool

type ModeKeyHandlers map[platform.Mode]KeyHandler
