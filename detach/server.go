//go:build unix

package detach

import (
	"net"
	"os"
	"sync"
	"syscall"
	"time"

	"github.com/yairgd/termforge/ptyx"
)

const writeTimeout = 2 * time.Second

// Serve runs argv on a PTY and serves it on the Unix socket until the program exits. env is
// the program's environment (nil: inherit). It returns when the program has ended; the
// socket is removed. The socket must not exist (see Handle, which removes a stale one).
func Serve(socket string, argv, env []string) error {
	tty, err := ptyx.Start(argv, ptyx.Options{Env: env, Rows: 24, Cols: 80})
	if err != nil {
		return err
	}
	defer tty.Close()

	ln, err := net.Listen("unix", socket)
	if err != nil {
		return err
	}
	defer os.Remove(socket)
	defer ln.Close()
	if err := os.Chmod(socket, 0o600); err != nil {
		return err
	}

	s := &server{tty: tty}
	out, cancel := tty.Subscribe()
	defer cancel()
	go s.accept(ln)

	// The output goes on, with or without a terminal attached; the channel closes when the
	// program exits.
	for msg := range out {
		s.output([]byte(msg.Data))
	}
	s.drop(nil)
	return nil
}

type server struct {
	tty *ptyx.TTY

	mu     sync.Mutex
	client net.Conn
	modes  modeTracker
}

// output tracks the terminal modes and forwards p to the attached terminal, if any. A terminal
// that does not take the output for writeTimeout is dropped, the program must not stall on it.
func (s *server) output(p []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.modes.feed(p)
	if s.client == nil {
		return
	}
	s.client.SetWriteDeadline(time.Now().Add(writeTimeout))
	if err := writeData(s.client, p); err != nil {
		s.client.Close()
		s.client = nil
	}
}

func (s *server) accept(ln net.Listener) {
	for {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		go s.serve(c)
	}
}

// serve handles one connection. It counts as an attach only once it sends its first frame (the
// size), so a probe that connects and closes (Handle checking for a live session) leaves the
// attached terminal alone.
func (s *server) serve(c net.Conn) {
	typ, p, err := readFrame(c)
	if err != nil {
		c.Close()
		return
	}

	s.mu.Lock()
	if s.client != nil {
		s.client.Close()
	}
	s.client = c
	c.SetWriteDeadline(time.Now().Add(writeTimeout))
	if err := writeData(c, s.modes.replay()); err != nil {
		s.client = nil
		s.mu.Unlock()
		c.Close()
		return
	}
	s.mu.Unlock()

	for {
		s.handle(typ, p)
		if typ, p, err = readFrame(c); err != nil {
			break
		}
	}
	s.drop(c)
}

func (s *server) handle(typ byte, p []byte) {
	switch typ {
	case frameData:
		s.tty.SendRaw(string(p))
	case frameResize:
		if len(p) == 4 {
			s.tty.SetSize(uint16(p[0])|uint16(p[1])<<8, uint16(p[2])|uint16(p[3])<<8)
		}
		// The kernel signals only a changed size; an attach needs the repaint in any case.
		if pid := s.tty.Pid(); pid > 0 {
			syscall.Kill(pid, syscall.SIGWINCH)
		}
	}
}

// drop closes c if it is the attached terminal; nil closes whoever is attached.
func (s *server) drop(c net.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.client != nil && (c == nil || s.client == c) {
		s.client.Close()
		s.client = nil
	} else if c != nil {
		c.Close()
	}
}
