package ptyx

import (
	"context"
	"time"
)

// CommandSink accepts commands destined for the process on the far side of a
// PTY. Implementations decide how a command is framed before it is written.
type CommandSink interface {
	Send(cmd string) error
	SendRaw(raw string) error
}

// PTYWriter is the exclusive write handle granted by Session.WithWrite.
// Callers must not use it after WithWrite returns.
type PTYWriter interface {
	Send(cmd string) error
	SendRaw(raw string) error
}

// Session is a CommandSink that owns process lifetime (Close) and supports
// multi-reader PTY output via Subscribe plus exclusive write via WithWrite.
// External APIs (MCP, REST, in-app AI) should use Session — not a concrete
// backend type — and must not Close the session while the UI owns it.
type Session interface {
	CommandSink
	Close()
	// Subscribe fans out raw PTY output. cancel unregisters; Close
	// closes every subscription. Drain promptly — a full buffer drops
	// chunks for that subscriber only.
	Subscribe() (ch <-chan PtyOutputMsg, cancel func())
	// WithWrite runs fn while holding the exclusive PTY write lock so only
	// one writer is active at a time. Readers (Subscribe) are unaffected.
	WithWrite(ctx context.Context, fn func(w PTYWriter) error) error
}

// PtyOutputMsg carries a raw PTY chunk from any ptyx-backed session.
// Used by Session.Subscribe.
type PtyOutputMsg struct {
	Data string
	Err  error
}

// Drain reads and discards PTY messages for wait, or until the channel closes.
func Drain(ch <-chan PtyOutputMsg, wait time.Duration) {
	if ch == nil || wait <= 0 {
		return
	}
	deadline := time.After(wait)
	for {
		select {
		case _, ok := <-ch:
			if !ok {
				return
			}
		case <-deadline:
			return
		}
	}
}
