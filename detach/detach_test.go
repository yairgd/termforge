//go:build unix

package detach

import (
	"bytes"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestFrames(t *testing.T) {
	var b bytes.Buffer
	writeData(&b, bytes.Repeat([]byte("x"), 3*chunk+5)) // several frames
	writeFrame(&b, frameResize, resizePayload(30, 100))
	var got int
	for {
		typ, p, err := readFrame(&b)
		if err != nil {
			t.Fatal(err)
		}
		if typ == frameResize {
			if len(p) != 4 || p[0] != 30 || p[2] != 100 {
				t.Fatalf("resize %v", p)
			}
			break
		}
		got += len(p)
	}
	if got != 3*chunk+5 {
		t.Fatalf("data %d", got)
	}
}

func TestModeTracker(t *testing.T) {
	var m modeTracker
	// sequences split across feeds, a non-private CSI in between, two modes in one, one turned off again
	for _, p := range []string{"hi \x1b[?10", "49h", "\x1b[2J\x1b[31m\x1b[?1000;1006h x", "\x1b[?25l\x1b[?1000l"} {
		m.feed([]byte(p))
	}
	// mode 25 was never on, so ?25l changes nothing; 1000 went on and off
	if got, want := string(m.replay()), "\x1b[?1049h\x1b[?1006h"; got != want {
		t.Fatalf("replay = %q, want %q", got, want)
	}
	if got, want := string(m.restore()), "\x1b[?1006l\x1b[?1049l\x1b[0m\x1b[?25h"; got != want {
		t.Fatalf("restore = %q, want %q", got, want)
	}
}

func TestParseArgs(t *testing.T) {
	name, mode, rest, err := parseArgs([]string{"--json", "a.json", "-session=main", "-x"})
	if err != nil || name != "main" || mode != "session" || strings.Join(rest, " ") != "--json a.json -x" {
		t.Fatalf("%q %q %v %v", name, mode, rest, err)
	}
	if name, mode, _, _ = parseArgs([]string{"--attach", "work"}); name != "work" || mode != "attach" {
		t.Fatalf("attach: %q %q", name, mode)
	}
	if _, mode, _, _ = parseArgs([]string{"--sessions"}); mode != "list" {
		t.Fatalf("list: %q", mode)
	}
	if _, mode, _, _ = parseArgs([]string{"--json", "x"}); mode != "" {
		t.Fatalf("plain: %q", mode)
	}
	if _, _, _, err = parseArgs([]string{"--session"}); err == nil {
		t.Fatal("--session without a name")
	}
	if _, _, _, err = parseArgs([]string{"--session", "a/b"}); err == nil {
		t.Fatal("bad name")
	}
}

// screen is the terminal side of a test: a buffer the attach writes to.
type screen struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *screen) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *screen) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

func (s *screen) waitFor(t *testing.T, want string) {
	t.Helper()
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(10 * time.Millisecond) {
		if strings.Contains(s.String(), want) {
			return
		}
	}
	t.Fatalf("no %q in %q", want, s.String())
}

// serveSh runs a shell script on a session and returns its socket and a channel closed when Serve returns.
func serveSh(t *testing.T, script string) (string, <-chan struct{}) {
	t.Helper()
	dir, err := os.MkdirTemp("", "tf")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	sock := filepath.Join(dir, "s.sock")
	ended := make(chan struct{})
	go func() {
		defer close(ended)
		if err := Serve(sock, []string{"/bin/sh", "-c", script}, nil); err != nil {
			t.Error(err)
		}
	}()
	for deadline := time.Now().Add(3 * time.Second); !alive(sock); time.Sleep(10 * time.Millisecond) {
		if time.Now().After(deadline) {
			t.Fatal("server did not start")
		}
	}
	return sock, ended
}

type attached struct {
	scr      *screen
	keys     *io.PipeWriter
	detached chan bool
}

func attachTo(t *testing.T, sock string, rows, cols uint16) *attached {
	return attachKey(t, sock, rows, cols, DefaultDetachKey)
}

func attachKey(t *testing.T, sock string, rows, cols uint16, key byte) *attached {
	t.Helper()
	conn, err := net.Dial("unix", sock)
	if err != nil {
		t.Fatal(err)
	}
	in, keys := io.Pipe()
	a := &attached{scr: &screen{}, keys: keys, detached: make(chan bool, 1)}
	go func() {
		d, err := attachConn(conn, in, a.scr, func() (uint16, uint16) { return rows, cols }, make(chan os.Signal), key)
		if err != nil {
			t.Error(err)
		}
		a.detached <- d
	}()
	return a
}

func (a *attached) typed(s string) { a.keys.Write([]byte(s)) }

func (a *attached) wait(t *testing.T) bool {
	t.Helper()
	select {
	case d := <-a.detached:
		return d
	case <-time.After(5 * time.Second):
		t.Fatal("attach did not end")
		return false
	}
}

// A program keeps its terminal modes across a detach: the second terminal is told which are on,
// gets the program's input and output, and a size change reaches the program.
func TestDetachAndReattach(t *testing.T) {
	// the program sets two modes, then echoes lines with the terminal size it sees
	sock, ended := serveSh(t, `sleep 0.3; printf '\033[?1049h\033[?1000h'; while read l; do echo "got:$l size:$(stty size)"; done`)

	a := attachTo(t, sock, 30, 100)
	a.scr.waitFor(t, "\x1b[?1049h\x1b[?1000h")
	a.typed("one\n")
	a.scr.waitFor(t, "got:one size:30 100")
	a.typed("\x1d") // Ctrl-]
	if !a.wait(t) {
		t.Fatal("not detached")
	}
	if got := a.scr.String(); !strings.HasSuffix(got, "\x1b[?1000l\x1b[?1049l\x1b[0m\x1b[?25h") {
		t.Fatalf("modes not restored: %q", got)
	}

	// the program is still there; another terminal of another size takes over
	b := attachTo(t, sock, 40, 120)
	b.scr.waitFor(t, "\x1b[?1049h\x1b[?1000h") // replayed
	b.typed("two\n")
	b.scr.waitFor(t, "got:two size:40 120")

	// a probe (what Handle does to see if a session lives) does not take the terminal
	if !alive(sock) {
		t.Fatal("not alive")
	}
	b.typed("three\n")
	b.scr.waitFor(t, "got:three")

	// the program ends: the attach ends, the server returns and the socket is gone
	b.typed("\x04") // EOF to read: the loop ends
	if b.wait(t) {
		t.Fatal("detached, want session ended")
	}
	select {
	case <-ended:
	case <-time.After(5 * time.Second):
		t.Fatal("Serve did not return")
	}
	if _, err := os.Stat(sock); !os.IsNotExist(err) {
		t.Fatalf("socket left: %v", err)
	}
}

func TestDetachKey(t *testing.T) {
	if CtrlKey(']') != 0x1d || CtrlKey('\\') != 0x1c || CtrlKey('a') != 1 {
		t.Fatal("CtrlKey")
	}
	for _, bad := range []byte{0, '\t', '\n', '\r', 0x1b, 'x'} {
		if _, err := newConfig([]Option{WithDetachKey(bad)}); err == nil {
			t.Errorf("%#x accepted", bad)
		}
	}
	if c, err := newConfig(nil); err != nil || c.key != 0x1d {
		t.Fatalf("default %v %v", c, err)
	}

	// with Ctrl-\ as the key, Ctrl-] is an ordinary key that reaches the program
	sock, _ := serveSh(t, `sleep 0.2; while IFS= read -r l; do echo "got:$(printf %s "$l" | od -An -c | tr -d ' ')"; done`)
	a := attachKey(t, sock, 24, 80, CtrlKey('\\'))
	a.typed("\x1d\n")
	a.scr.waitFor(t, "got:035")
	a.typed("\x1c")
	if !a.wait(t) {
		t.Fatal("not detached")
	}
}
