package termforge

import (
	"strings"
	"sync"
	"time"

	"github.com/yairgd/termforge/ptyx"
)

// CoalesceRunner runs at most one worker; bursts set pending for one trailing run.
type CoalesceRunner struct {
	mu      sync.Mutex
	running bool
	pending bool
}

// Schedule starts work or marks a trailing rerun if already running.
func (r *CoalesceRunner) Schedule(work func()) {
	if r == nil || work == nil {
		return
	}
	r.mu.Lock()
	if r.running {
		r.pending = true
		r.mu.Unlock()
		return
	}
	r.running = true
	r.mu.Unlock()
	go r.loop(work)
}

func (r *CoalesceRunner) loop(work func()) {
	for {
		work()
		r.mu.Lock()
		if r.pending {
			r.pending = false
			r.mu.Unlock()
			continue
		}
		r.running = false
		r.mu.Unlock()
		return
	}
}

// PtyCoalesceOpts batches PTY chunks onto post callbacks.
type PtyCoalesceOpts struct {
	Interval time.Duration
	MaxBytes int
	// HardMax > 0 trims pending under flood (keep head+tail + marker).
	HardMax int
	Post    func(data string, err error)
	OnExit  func()
}

// CoalescePtyOutput batches PTY chunks onto opts.Post so a chatty child
// process cannot force one repaint per read.
func CoalescePtyOutput(ch <-chan ptyx.PtyOutputMsg, opts PtyCoalesceOpts) {
	if opts.Interval <= 0 {
		opts.Interval = 16 * time.Millisecond
	}
	if opts.MaxBytes <= 0 {
		opts.MaxBytes = 64 * 1024
	}
	var pending strings.Builder
	var flushTimer *time.Timer
	var flushC <-chan time.Time
	dropped := 0

	disarm := func() {
		if flushTimer == nil {
			return
		}
		if !flushTimer.Stop() {
			select {
			case <-flushTimer.C:
			default:
			}
		}
		flushTimer = nil
		flushC = nil
	}
	trimPending := func() {
		if opts.HardMax <= 0 || pending.Len() <= opts.HardMax {
			return
		}
		s := pending.String()
		keepHead := opts.HardMax / 4
		keepTail := opts.HardMax - keepHead - 80
		if keepTail < 1024 {
			keepTail = opts.HardMax / 2
			keepHead = opts.HardMax - keepTail
		}
		marker := "\n... [output truncated under flood] ...\n"
		dropped += len(s) - keepHead - keepTail
		pending.Reset()
		pending.WriteString(s[:keepHead])
		pending.WriteString(marker)
		pending.WriteString(s[len(s)-keepTail:])
	}
	flush := func() {
		disarm()
		if pending.Len() == 0 {
			return
		}
		data := pending.String()
		pending.Reset()
		if dropped > 0 {
			data = "... [earlier output truncated under flood] ...\n" + data
			dropped = 0
		}
		if opts.Post != nil {
			opts.Post(data, nil)
		}
	}
	arm := func() {
		if flushTimer != nil {
			return
		}
		flushTimer = time.NewTimer(opts.Interval)
		flushC = flushTimer.C
	}

	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				flush()
				if opts.OnExit != nil {
					opts.OnExit()
				}
				return
			}
			if msg.Err != nil {
				flush()
				if opts.Post != nil {
					opts.Post("", msg.Err)
				}
				if opts.OnExit != nil && ptyx.ClosedError(msg.Err) {
					opts.OnExit()
					return
				}
				continue
			}
			if msg.Data == "" {
				continue
			}
			pending.WriteString(msg.Data)
			trimPending()
			if pending.Len() >= opts.MaxBytes {
				flush()
			} else {
				arm()
			}
		case <-flushC:
			flush()
		}
	}
}
