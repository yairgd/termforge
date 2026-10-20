//go:build lua

package lua

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/yairgd/termforge/ptyx"
)

// One Lua per process, so a single test drives the whole REPL.
func TestREPLPluginInterruptEOF(t *testing.T) {
	tty, err := ptyx.Open()
	if err != nil {
		t.Skipf("pty unavailable: %v", err)
	}
	defer tty.Close()
	out, cancel := tty.Subscribe()
	defer cancel()

	Register("", map[string]Func{
		"double": func(c *Call) (int, error) {
			v, ok := c.Number(1)
			if !ok {
				return 0, errors.New("number expected")
			}
			c.PushNumber(v * 2)
			return 1, nil
		},
	})
	Register("tf", map[string]Func{
		"hello": func(c *Call) (int, error) {
			s, _ := c.String(1)
			c.PushString("hi " + s)
			return 1, nil
		},
	})

	done := make(chan int, 1)
	go func() { done <- Run(tty.SlaveName()) }()

	var seen strings.Builder
	waitFor := func(want string) {
		t.Helper()
		deadline := time.After(5 * time.Second)
		for !strings.Contains(seen.String(), want) {
			select {
			case m := <-out:
				seen.WriteString(m.Data)
			case <-deadline:
				t.Fatalf("timeout waiting for %q; got %q", want, seen.String())
			}
		}
		// keep what follows the match: the prompt can arrive in the same chunk as the banner
		rest := seen.String()
		rest = rest[strings.Index(rest, want)+len(want):]
		seen.Reset()
		seen.WriteString(rest)
	}
	send := func(s string) {
		t.Helper()
		if err := tty.SendRaw(s); err != nil {
			t.Fatal(err)
		}
	}

	waitFor("Lua 5.4")
	waitFor("> ")
	send("print(double(21))\r")
	waitFor("42")
	send("print(tf.hello('x'))\r")
	waitFor("hi x")
	send("\x1b[A\r") // history: Up repeats the last line
	waitFor("hi x")
	send("junk\x03print(123+1)\r") // Ctrl-C drops the line, the REPL goes on
	waitFor("124")
	send("double('a')\r")
	waitFor("number expected")
	send("while true do end\r")
	time.Sleep(200 * time.Millisecond)
	Interrupt()
	waitFor("interrupted!")
	send("\x04")
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return after EOF")
	}
}
