//go:build unix

package detach

import (
	"strconv"
	"strings"
)

// modeTracker follows the DEC private modes (ESC [ ? n h / l) in a stream of terminal output:
// alternate screen, mouse reporting, bracketed paste, cursor visibility ... A program sets them
// once at startup, so a terminal that attaches later has to be told which are on.
type modeTracker struct {
	state int // 0 text, 1 after ESC, 2 inside CSI
	param []byte
	on    []int // in the order they were set
}

func (m *modeTracker) feed(p []byte) {
	for _, c := range p {
		switch m.state {
		case 0:
			if c == 0x1b {
				m.state = 1
			}
		case 1:
			m.param = m.param[:0]
			if c == '[' {
				m.state = 2
			} else {
				m.state = 0
			}
		case 2:
			switch {
			case c >= 0x30 && c <= 0x3f: // parameter bytes
				if len(m.param) < 64 {
					m.param = append(m.param, c)
				}
			case c >= 0x20 && c <= 0x2f: // intermediate bytes
			case c >= 0x40 && c <= 0x7e: // final byte
				if (c == 'h' || c == 'l') && len(m.param) > 0 && m.param[0] == '?' {
					m.apply(string(m.param[1:]), c == 'h')
				}
				m.state = 0
			default: // not a CSI: start over
				m.state = 0
			}
		}
	}
}

func (m *modeTracker) apply(params string, set bool) {
	for _, s := range strings.Split(params, ";") {
		n, err := strconv.Atoi(s)
		if err != nil {
			continue
		}
		i := indexOf(m.on, n)
		switch {
		case set && i < 0:
			m.on = append(m.on, n)
		case !set && i >= 0:
			m.on = append(m.on[:i], m.on[i+1:]...)
		}
	}
}

func indexOf(l []int, n int) int {
	for i, v := range l {
		if v == n {
			return i
		}
	}
	return -1
}

// replay is what turns the modes on in a fresh terminal, in the order the program set them.
func (m *modeTracker) replay() []byte {
	var b []byte
	for _, n := range m.on {
		b = append(b, "\x1b[?"+strconv.Itoa(n)+"h"...)
	}
	return b
}

// restore turns them off again, last first, and leaves the terminal usable: default colors, cursor shown.
func (m *modeTracker) restore() []byte {
	var b []byte
	for i := len(m.on) - 1; i >= 0; i-- {
		b = append(b, "\x1b[?"+strconv.Itoa(m.on[i])+"l"...)
	}
	return append(b, "\x1b[0m\x1b[?25h"...)
}
