//go:build unix

package detach

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"syscall"
	"time"
)

const (
	envServe    = "TERMFORGE_DETACH_SERVE" // set in the server process: the socket to serve
	envSession  = "TERMFORGE_SESSION"      // set for the program inside a session: its name
	startWait   = 5 * time.Second
	probeWait   = 500 * time.Millisecond
	nameChars   = `^[A-Za-z0-9._-]+$`
	sockSuffix  = ".sock"
	maxSockPath = 100
)

var validName = regexp.MustCompile(nameChars)

// Handle adds --session, --attach and --sessions to the command line of the program named
// app (see the package comment); WithDetachKey changes the detach key. It reads os.Args and returns handled = true when it did
// something, then the caller should return (after reporting err). Without those flags it
// does nothing and returns false.
func Handle(app string, opts ...Option) (handled bool, err error) {
	if _, err := newConfig(opts); err != nil {
		return true, err
	}
	if sock := os.Getenv(envServe); sock != "" {
		// This process is the server the first --session started: run the program in a PTY.
		self, err := os.Executable()
		if err != nil {
			return true, err
		}
		return true, Serve(sock, append([]string{self}, os.Args[1:]...), withoutEnv(os.Environ(), envServe))
	}

	name, mode, rest, err := parseArgs(os.Args[1:])
	if err != nil || mode == "" {
		return err != nil, err
	}
	dir, err := socketDir()
	if err != nil {
		return true, err
	}
	if mode == "list" {
		for _, n := range sessions(app, dir, fallbackDir()) {
			fmt.Println(n)
		}
		return true, nil
	}

	sock := filepath.Join(dir, app+"-"+name+sockSuffix)
	if len(sock) > maxSockPath { // sun_path is about 108 bytes; XDG_RUNTIME_DIR can be deep
		dir = fallbackDir()
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return true, err
		}
		sock = filepath.Join(dir, app+"-"+name+sockSuffix)
	}
	if !alive(sock) {
		if mode == "attach" {
			return true, fmt.Errorf("no session %q (start it with --session %s)", name, name)
		}
		if err := start(sock, name, rest); err != nil {
			return true, err
		}
	} else if len(rest) > 0 {
		fmt.Fprintf(os.Stderr, "session %q is running, ignoring %v\n", name, rest)
	}
	return true, Attach(sock, opts...)
}

// parseArgs takes --session NAME, --attach NAME and --sessions (one or two dashes, NAME as
// the next word or after '=') out of args. rest is what is left, for the program.
func parseArgs(args []string) (name, mode string, rest []string, err error) {
	for i := 0; i < len(args); i++ {
		flag, val, hasVal := strings.Cut(strings.TrimLeft(args[i], "-"), "=")
		if !strings.HasPrefix(args[i], "-") {
			rest = append(rest, args[i])
			continue
		}
		switch flag {
		case "session", "attach":
			if !hasVal {
				if i+1 >= len(args) {
					return "", "", nil, fmt.Errorf("--%s needs a session name", flag)
				}
				i++
				val = args[i]
			}
			if !validName.MatchString(val) {
				return "", "", nil, fmt.Errorf("session name %q: use letters, digits, . _ -", val)
			}
			name, mode = val, flag
		case "sessions":
			mode = "list"
		default:
			rest = append(rest, args[i])
		}
	}
	return name, mode, rest, nil
}

// socketDir is $XDG_RUNTIME_DIR/termforge, or /tmp/termforge-UID.
func socketDir() (string, error) {
	dir := filepath.Join(os.TempDir(), fmt.Sprintf("termforge-%d", os.Getuid()))
	if x := os.Getenv("XDG_RUNTIME_DIR"); x != "" {
		dir = filepath.Join(x, "termforge")
	}
	return dir, os.MkdirAll(dir, 0o700)
}

// alive reports whether a server answers on sock. The probe sends nothing, so it does not
// take the terminal from whoever is attached.
func alive(sock string) bool {
	c, err := net.DialTimeout("unix", sock, probeWait)
	if err != nil {
		return false
	}
	c.Close()
	return true
}

// fallbackDir holds the sockets whose path would be too long under XDG_RUNTIME_DIR.
func fallbackDir() string { return filepath.Join("/tmp", fmt.Sprintf("termforge-%d", os.Getuid())) }

func sessions(app string, dirs ...string) []string {
	seen := map[string]bool{}
	var out []string
	for _, dir := range dirs {
		files, _ := filepath.Glob(filepath.Join(dir, app+"-*"+sockSuffix))
		for _, f := range files {
			n := strings.TrimSuffix(strings.TrimPrefix(filepath.Base(f), app+"-"), sockSuffix)
			if !seen[n] && alive(f) {
				seen[n] = true
				out = append(out, n)
			}
		}
	}
	sort.Strings(out)
	return out
}

// start launches this program again as the server of a new session, detached from the
// terminal (own session, no stdio; messages go to <socket>.log), and waits for its socket.
func start(sock, name string, args []string) error {
	self, err := os.Executable()
	if err != nil {
		return err
	}
	os.Remove(sock) // a stale one: nothing answered on it
	logf, err := os.Create(sock + ".log")
	if err != nil {
		return err
	}
	defer logf.Close()
	cmd := exec.Command(self, args...)
	cmd.Env = append(os.Environ(), envServe+"="+sock, envSession+"="+name)
	if os.Getenv("TERM") == "" {
		cmd.Env = append(cmd.Env, "TERM=xterm-256color")
	}
	cmd.Stdout, cmd.Stderr = logf, logf
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return err
	}
	exited := make(chan struct{})
	go func() { cmd.Wait(); close(exited) }()

	for deadline := time.Now().Add(startWait); time.Now().Before(deadline); time.Sleep(50 * time.Millisecond) {
		if alive(sock) {
			return nil
		}
		select {
		case <-exited:
			b, _ := os.ReadFile(sock + ".log")
			return fmt.Errorf("session %q did not start: %s", name, strings.TrimSpace(string(b)))
		default:
		}
	}
	return fmt.Errorf("session %q: no answer on %s after %v", name, sock, startWait)
}

func withoutEnv(env []string, key string) []string {
	out := make([]string, 0, len(env))
	for _, e := range env {
		if !strings.HasPrefix(e, key+"=") {
			out = append(out, e)
		}
	}
	return out
}
