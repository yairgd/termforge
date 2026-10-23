//go:build unix

package detach

import (
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"golang.org/x/term"
)

// DefaultDetachKey is Ctrl-].
const DefaultDetachKey = 0x1d

// Option configures Handle and Attach.
type Option func(*config)

type config struct{ key byte }

// WithDetachKey sets the key that detaches, as the control byte the terminal sends: CtrlKey(']')
// (the default), CtrlKey('\\') ... The app never sees this key.
func WithDetachKey(key byte) Option { return func(c *config) { c.key = key } }

// CtrlKey is the byte the terminal sends for Ctrl+r: CtrlKey(']') is 0x1d, CtrlKey('\\') 0x1c, CtrlKey('a') 1.
func CtrlKey(r rune) byte { return byte(r) & 0x1f }

func newConfig(opts []Option) (config, error) {
	c := config{key: DefaultDetachKey}
	for _, o := range opts {
		o(&c)
	}
	switch c.key {
	case 0, 0x09, 0x0a, 0x0d, 0x1b: // NUL, Tab, Enter, Esc: the app needs them
		return c, fmt.Errorf("detach: %#x cannot be the detach key", c.key)
	}
	if c.key > 0x1f {
		return c, fmt.Errorf("detach: the detach key must be a control key, not %#x", c.key)
	}
	return c, nil
}

// Attach connects the terminal to the session served on socket. It puts the terminal in raw
// mode and returns when the user detaches (Ctrl-], or the key set with WithDetachKey) or the
// program ends.
func Attach(socket string, opts ...Option) error {
	cfg, err := newConfig(opts)
	if err != nil {
		return err
	}
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return fmt.Errorf("detach: stdin is not a terminal")
	}
	conn, err := net.Dial("unix", socket)
	if err != nil {
		return err
	}
	old, err := term.MakeRaw(fd)
	if err != nil {
		conn.Close()
		return err
	}
	size := func() (uint16, uint16) {
		w, h, err := term.GetSize(fd)
		if err != nil || w <= 0 || h <= 0 {
			return 24, 80
		}
		return uint16(h), uint16(w)
	}
	winch := make(chan os.Signal, 1)
	signal.Notify(winch, syscall.SIGWINCH)
	defer signal.Stop(winch)

	detached, err := attachConn(conn, os.Stdin, os.Stdout, size, winch, cfg.key)
	term.Restore(fd, old)
	switch {
	case err != nil:
		return err
	case detached:
		fmt.Println("[detached]")
	default:
		fmt.Println("[session ended]")
	}
	return nil
}

// attachConn relays between the terminal (in, out) and the session on conn until the detach
// key (key) is typed (detached = true) or the session closes the connection. When it returns the
// terminal modes the program had turned on are off again.
func attachConn(conn net.Conn, in io.Reader, out io.Writer, size func() (rows, cols uint16),
	winch <-chan os.Signal, key byte) (detached bool, err error) {

	var wmu sync.Mutex // the socket is written by the keys and the resizes
	send := func(typ byte, p []byte) error {
		wmu.Lock()
		defer wmu.Unlock()
		if typ == frameData {
			return writeData(conn, p)
		}
		return writeFrame(conn, typ, p)
	}
	sendSize := func() error { return send(frameResize, resizePayload(size())) }
	if err := sendSize(); err != nil { // also what makes this an attach, for the server
		conn.Close()
		return false, err
	}

	type end struct {
		detached bool
		err      error
	}
	done := make(chan end, 3)
	var modes modeTracker

	go func() { // the program's output
		for {
			typ, p, err := readFrame(conn)
			if err != nil {
				done <- end{false, nil}
				return
			}
			if typ == frameData {
				modes.feed(p)
				if _, err := out.Write(p); err != nil {
					done <- end{false, err}
					return
				}
			}
		}
	}()
	go func() { // the keys
		buf := make([]byte, 4096)
		for {
			n, err := in.Read(buf)
			if n > 0 {
				p := buf[:n]
				for i, b := range p {
					if b == key {
						if i > 0 {
							send(frameData, p[:i])
						}
						done <- end{true, nil}
						return
					}
				}
				if err := send(frameData, p); err != nil {
					done <- end{false, nil}
					return
				}
			}
			if err != nil {
				done <- end{false, nil}
				return
			}
		}
	}()
	go func() {
		for range winch {
			if sendSize() != nil {
				return
			}
		}
	}()

	e := <-done
	conn.Close()
	// modes is written by the output goroutine; after Close it ends, but may still be in Write.
	wmu.Lock()
	out.Write(modes.restore())
	wmu.Unlock()
	return e.detached, e.err
}
