package termforge

import (
	"sync"
	"syscall"
	"time"
)

// ChildProcCtl tracks processes the application has spawned so they can be
// terminated as a group on shutdown. PIDs are registered by whoever starts
// the process; nothing is discovered automatically.
type ChildProcCtl struct {
	mu sync.Mutex
	// pid -> kill the whole process group (Setpgid leader).
	groups map[int]bool
}

func (c *ChildProcCtl) Track(pid int, killGroup bool) {
	if pid <= 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.groups == nil {
		c.groups = make(map[int]bool)
	}
	c.groups[pid] = killGroup
}

func (c *ChildProcCtl) KillAll() {
	c.mu.Lock()
	entries := make([]struct {
		pid       int
		killGroup bool
	}, 0, len(c.groups))
	for pid, killGroup := range c.groups {
		entries = append(entries, struct {
			pid       int
			killGroup bool
		}{pid, killGroup})
	}
	c.groups = nil
	c.mu.Unlock()

	for _, e := range entries {
		SignalProcess(e.pid, e.killGroup, syscall.SIGTERM)
	}
	time.Sleep(300 * time.Millisecond)
	for _, e := range entries {
		if ProcessAlive(e.pid) {
			SignalProcess(e.pid, e.killGroup, syscall.SIGKILL)
		}
	}
}

// SignalProcess sends sig to pid, and to its process group when killGroup.
func SignalProcess(pid int, killGroup bool, sig syscall.Signal) {
	if killGroup {
		_ = syscall.Kill(-pid, sig)
	}
	_ = syscall.Kill(pid, sig)
}

// ProcessAlive reports whether pid is still running.
func ProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	return syscall.Kill(pid, 0) == nil
}

func (c *ChildProcCtl) KillTracked(pid int) {
	if pid <= 0 {
		return
	}
	c.mu.Lock()
	killGroup, ok := c.groups[pid]
	if ok {
		delete(c.groups, pid)
	}
	c.mu.Unlock()
	if !ok {
		killGroup = true
	}
	SignalProcess(pid, killGroup, syscall.SIGTERM)
	time.Sleep(300 * time.Millisecond)
	if ProcessAlive(pid) {
		SignalProcess(pid, killGroup, syscall.SIGKILL)
	}
}

func (c *ChildProcCtl) KillOne(pid int) {
	c.KillTracked(pid)
}
