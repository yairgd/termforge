//go:build lua

package demo

import (
	"errors"
	"math"
	"sync/atomic"
	"time"

	"github.com/yairgd/termforge/lua"
)

// RegisterMatrixLua exposes m to Lua as the mat table:
//
//	mat.plot(x, y [, color])             set a pixel
//	mat.line(x0, y0, x1, y1 [, color])   draw a line
//
// color is 0xRRGGBB (0x123456: r=0x12, g=0x34, b=0x56); the default is white.
//
//	mat.clear()                          blank the matrix
//	mat.size()                           -> width, height
//	mat.delay(ms)                        sleep, for animations
//	mat.fire([frames [, fps [, cooling]]])  fire effect
//	mat.plasma([frames [, fps]])         plasma effect
//	help([name])                         describe all of the above
//
// redraw is called (from another goroutine) at most every 25 ms while the matrix changes.
func RegisterMatrixLua(m *Matrix, redraw func()) {
	var dirty atomic.Bool
	go func() {
		for range time.Tick(25 * time.Millisecond) {
			if dirty.Swap(false) {
				redraw()
			}
		}
	}()

	ints := func(c *lua.Call, n int) ([]int, error) {
		v := make([]int, n)
		for i := range v {
			x, ok := c.Number(i + 1)
			if !ok {
				return nil, errors.New("bad argument #" + string(rune('1'+i)) + " (number expected)")
			}
			v[i] = int(x)
		}
		return v, nil
	}
	colour := func(c *lua.Call, arg int) (r, g, b byte) {
		v := 0xffffff
		if c.Top() >= arg {
			v, _ = c.Int(arg)
		}
		return byte(v >> 16), byte(v >> 8), byte(v)
	}

	// number argument n of c, or def when absent
	num := func(c *lua.Call, n int, def float64) float64 {
		if c.Top() < n {
			return def
		}
		if v, ok := c.Number(n); ok {
			return v
		}
		return def
	}
	// animation runs an effect on the Lua thread; Ctrl-C ends it
	animation := func(c *lua.Call, run func(step func() bool)) (int, error) {
		start := lua.Interrupts()
		run(func() bool {
			dirty.Store(true)
			return lua.Interrupts() == start
		})
		return 0, nil
	}
	// frames: absent or 0 runs until Ctrl-C
	frames := func(v float64) int {
		if v < 1 {
			return math.MaxInt
		}
		return int(v)
	}
	fpsOf := func(v float64) int {
		if v < 1 {
			return 30
		}
		return int(v)
	}

	lua.Register("", map[string]lua.Func{
		"help": func(c *lua.Call) (int, error) {
			if name, ok := c.String(1); ok {
				for _, d := range matDocs {
					if d.name == name || d.name == "mat."+name {
						c.Print(d.usage + "\n    " + d.text)
						return 0, nil
					}
				}
				return 0, errors.New("no help for '" + name + "'")
			}
			out := "Lua console. The mat library draws on the matrix above (64x32);\ncolor is 0xRRGGBB, e.g. 0x123456 is r=0x12 g=0x34 b=0x56.\n"
			for _, d := range matDocs {
				out += "\n  " + d.usage + "\n      " + d.text
			}
			out += "\n\nhelp('name') shows one entry. Ctrl-C stops a running effect."
			c.Print(out)
			return 0, nil
		},
	})

	lua.Register("mat", map[string]lua.Func{
		"plot": func(c *lua.Call) (int, error) {
			p, err := ints(c, 2)
			if err != nil {
				return 0, err
			}
			r, g, b := colour(c, 3)
			m.Plot(p[0], p[1], r, g, b)
			dirty.Store(true)
			return 0, nil
		},
		"line": func(c *lua.Call) (int, error) {
			p, err := ints(c, 4)
			if err != nil {
				return 0, err
			}
			r, g, b := colour(c, 5)
			m.Line(p[0], p[1], p[2], p[3], r, g, b)
			dirty.Store(true)
			return 0, nil
		},
		"clear": func(c *lua.Call) (int, error) {
			m.Clear()
			dirty.Store(true)
			return 0, nil
		},
		"size": func(c *lua.Call) (int, error) {
			c.PushInt(MatrixW)
			c.PushInt(MatrixH)
			return 2, nil
		},
		"fire": func(c *lua.Call) (int, error) {
			n, fps := frames(num(c, 1, 0)), fpsOf(num(c, 2, 30))
			cooling := num(c, 3, 55)
			return animation(c, func(step func() bool) { m.Fire(n, fps, cooling, step) })
		},
		"plasma": func(c *lua.Call) (int, error) {
			n, fps := frames(num(c, 1, 0)), fpsOf(num(c, 2, 30))
			return animation(c, func(step func() bool) { m.Plasma(n, fps, step) })
		},
		"delay": func(c *lua.Call) (int, error) {
			ms, ok := c.Number(1)
			if !ok {
				return 0, errors.New("bad argument #1 (number expected)")
			}
			time.Sleep(time.Duration(ms * float64(time.Millisecond)))
			return 0, nil
		},
	})
}

// matDocs is what help() prints, in order.
var matDocs = []struct{ name, usage, text string }{
	{"mat.plot", "mat.plot(x, y [, color])", "set one pixel; x 0-63, y 0-31, color 0xRRGGBB (default white)"},
	{"mat.line", "mat.line(x0, y0, x1, y1 [, color])", "draw a line between two pixels"},
	{"mat.clear", "mat.clear()", "blank the matrix"},
	{"mat.size", "mat.size()", "returns width, height"},
	{"mat.delay", "mat.delay(ms)", "sleep, for hand-made animations"},
	{"mat.fire", "mat.fire([frames [, fps [, cooling]]])", "fire effect; frames 0 = until Ctrl-C (default), fps 30, cooling 55 (0-100, higher = lower flames)"},
	{"mat.plasma", "mat.plasma([frames [, fps]])", "plasma effect; frames 0 = until Ctrl-C (default), fps 30"},
	{"help", "help([name])", "this text, or one entry, e.g. help('fire')"},
}
