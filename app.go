package termforge

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/yairgd/termforge/platform"
)

type AppApi interface {
	HandleMouse(ev *tcell.EventMouse)
	HandleInterrupt(ev *tcell.EventInterrupt)
	// HandleTTYResume runs after Suspend/Resume or RunForeground; reset in-pane terminals.
	HandleTTYResume()
}

type App struct {
	Api     AppApi
	widgets WidgetsList
	screen  tcell.Screen
	exit    bool
	// widgets draw here all the time
	// last frame that was actually displayed
	frontBuffer *Grid
	canvas      Canvas

	mouseActive bool
	mouseX      int
	mouseY      int

	// paintInterval is the frame budget for coalesced output; 0 means default.
	paintInterval time.Duration
	layoutDirty   bool
	appState      *platform.AppState
	modeHandlers  ModeKeyHandlers
	closeOnce     sync.Once
	suspendMu     sync.Mutex
}

func NewApp() *App {

	screen, err := tcell.NewScreen()
	if err != nil {
		log.Fatal(err)
	}
	if err := screen.Init(); err != nil {
		log.Fatal(err)
	}

	screen.EnableMouse(tcell.MouseMotionEvents)
	screen.EnablePaste()

	app := &App{
		screen:       screen,
		exit:         false,
		modeHandlers: make(ModeKeyHandlers),
		appState:     platform.NewAppState(),
	}
	app.UpdateCanvas()
	return app
}

func (app *App) Exit() { app.exit = true }

// AddWidget registers chrome that fills the rows the fixed-height ones leave
// over (the workspace band). See WidgetsList for the placement rules.
func (app *App) AddWidget(w Widget) { app.widgets.AddWidget(w) }

// AddRowWidget registers a full-width band of rows rows, stacked below the
// chrome registered before it.
func (app *App) AddRowWidget(w Widget, rows int) { app.widgets.AddRowWidget(w, rows) }

// AddFloatingWidget registers a widget placed by rect on every frame, painted
// over the chrome registered before it.
func (app *App) AddFloatingWidget(w Widget, rect func(Canvas) Rect) {
	app.widgets.AddFloatingWidget(w, rect)
}

// WidgetRect returns the screen rect the layout gave w, or the zero Rect when w
// is not registered. Used by mouse routing to find the surface under a click.
func (app *App) WidgetRect(w Widget) Rect { return app.widgets.Rect(w) }

func (app *App) Mode() platform.Mode {
	return app.appState.Mode()
}

func (app *App) SetMode(mode platform.Mode) {
	app.appState.SetMode(mode)
}

// State returns the process-global AppState (modes, PTY owner, layout policy).
func (app *App) State() *platform.AppState {
	return app.appState
}

func (app *App) RegisterModeHandler(mode platform.Mode, h KeyHandler) {
	app.modeHandlers[mode] = h
}

func (app *App) HandleKey(ev *tcell.EventKey) {
	if h, ok := app.modeHandlers[app.appState.Mode()]; ok {
		if h(ev) {
			return
		}
	}
}

func (app *App) Close() {
	if app == nil {
		return
	}
	app.closeOnce.Do(func() {
		app.exit = true
		if app.screen == nil {
			return
		}
		// Give terminal enough time to disable mouse reporting.
		// Without this delay, pending mouse escape sequences may
		// leak to the shell after Fini().
		time.Sleep(100 * time.Millisecond)
		app.screen.DisableMouse()
		app.screen.Fini()
	})
}

func (app *App) drainTcellEventQueue() {
	if app == nil || app.screen == nil {
		return
	}
	for app.screen.HasPendingEvent() {
		if ev := app.screen.PollEvent(); ev == nil {
			return
		}
	}
}

func isAlreadyEngaged(err error) bool {
	return err != nil && strings.Contains(err.Error(), "already engaged")
}

func (app *App) syncTerminalDimensions() (w, h int) {
	if app == nil || app.screen == nil {
		return 80, 24
	}
	app.screen.Sync()
	w, h = app.screen.Size()
	if tty, ok := app.screen.Tty(); ok {
		if ws, err := tty.WindowSize(); err == nil && ws.Width > 0 && ws.Height > 0 {
			w, h = ws.Width, ws.Height
		}
	}
	if w < 8 || h < 4 {
		time.Sleep(10 * time.Millisecond)
		app.screen.Sync()
		w, h = app.screen.Size()
	}
	if w < 8 {
		w = 80
	}
	if h < 4 {
		h = 24
	}
	return w, h
}

func (app *App) restoreAfterResume() {
	if app == nil || app.screen == nil {
		return
	}
	flushControllingTTYInput()
	app.screen.EnableMouse(tcell.MouseMotionEvents)
	app.screen.EnablePaste()
	app.screen.Clear()
	app.screen.Sync()
	_, _ = app.syncTerminalDimensions()
	_ = app.UpdateCanvas()
	app.layoutDirty = true
	if app.Api != nil {
		app.Api.HandleTTYResume()
	}
	app.drainTcellEventQueue()
	app.present()
}

func (app *App) resumeAfterSuspend() error {
	if err := app.screen.Resume(); err != nil {
		if isAlreadyEngaged(err) {
			// Stopped mid-disengage on a prior cycle — force clean re-engage.
			_ = app.screen.Suspend()
			if err2 := app.screen.Resume(); err2 != nil && !isAlreadyEngaged(err2) {
				return err2
			}
		} else {
			return err
		}
	}
	app.restoreAfterResume()
	return nil
}

// withTTYReleased disengages tcell, runs fn on the real tty, then re-engages.
// PollEvent runs only on the UI thread — no background poll goroutine — so
// Suspend/Resume cannot race with a second PollEvent caller.
func (app *App) withTTYReleased(fn func() error) error {
	if app == nil || app.screen == nil {
		return fmt.Errorf("no screen")
	}
	app.drainTcellEventQueue()

	if err := app.screen.Suspend(); err != nil {
		return fmt.Errorf("suspend: %w", err)
	}

	runErr := fn()
	if err := app.resumeAfterSuspend(); err != nil {
		app.restoreAfterResume()
		if runErr == nil {
			return err
		}
		return fmt.Errorf("%v (resume: %v)", runErr, err)
	}
	app.drainTcellEventQueue()
	return runErr
}

// Suspend restores the terminal and stops this process with SIGTSTP (job
// control), like Ctrl-Z in GDB/vim. Resumes on SIGCONT / shell `fg`.
//
// Must use Screen.Suspend/Resume — Fini() is once-only (finiOnce), so
// Fini+Init breaks on the second Ctrl-Z with "already engaged".
func (app *App) Suspend() {
	if app == nil || app.screen == nil {
		return
	}
	app.suspendMu.Lock()
	defer app.suspendMu.Unlock()

	if err := app.withTTYReleased(stopForShellJobControl); err != nil {
		log.Printf("suspend: %v", err)
	}
}

// RunForeground suspends the tcell screen, runs argv on the real stdin/stdout
// (terminal vim, less, …), then resumes gdbforge. Must run on the UI thread.
func (app *App) RunForeground(argv []string) error {
	if app == nil || app.screen == nil {
		return fmt.Errorf("no screen")
	}
	if len(argv) == 0 || strings.TrimSpace(argv[0]) == "" {
		return fmt.Errorf("empty command")
	}
	app.suspendMu.Lock()
	defer app.suspendMu.Unlock()

	return app.withTTYReleased(func() error {
		cmd := exec.Command(argv[0], argv[1:]...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	})
}

func (app *App) pollEventBatch(max int) []tcell.Event {
	if app == nil || app.screen == nil {
		return nil
	}
	if max < 1 {
		max = 1
	}
	if app.screen.HasPendingEvent() {
		return app.drainPollBatch(max)
	}
	ev := app.screen.PollEvent()
	if ev == nil {
		return nil
	}
	batch := []tcell.Event{ev}
	if len(batch) < max && app.screen.HasPendingEvent() {
		batch = append(batch, app.drainPollBatch(max-len(batch))...)
	}
	return batch
}

func (app *App) drainPollBatch(max int) []tcell.Event {
	batch := make([]tcell.Event, 0, max)
	for len(batch) < max && app.screen.HasPendingEvent() {
		if ev := app.screen.PollEvent(); ev != nil {
			batch = append(batch, ev)
		} else {
			break
		}
	}
	return batch
}

const defaultPaintInterval = 16 * time.Millisecond

func (app *App) Run() {
	defer app.Close()

	interval := app.paintInterval
	if interval <= 0 {
		interval = defaultPaintInterval
	}
	paintTicker := time.NewTicker(interval)
	defer paintTicker.Stop()
	dirty := false
	for !app.exit {
		select {
		case <-paintTicker.C:
			if dirty {
				app.present()
				dirty = false
			}
		default:
		}

		if app.mustWaitForPaintTick(dirty) {
			<-paintTicker.C
			app.present()
			dirty = false
			continue
		}

		batch := app.pollEventBatch(96)
		if len(batch) == 0 {
			continue
		}
		urgent := app.handleUIEventBatch(batch)
		dirty = true
		if urgent {
			app.present()
			dirty = false
		}
	}
}

// mustWaitForPaintTick reports whether a pending frame has to be driven by the
// paint ticker. pollEventBatch blocks in PollEvent until the next event, so an
// unpainted frame with an empty queue would otherwise stay on screen-behind
// until the user pressed another key (async output needs no further input).
func (app *App) mustWaitForPaintTick(dirty bool) bool {
	if !dirty || app == nil || app.screen == nil {
		return false
	}
	return !app.screen.HasPendingEvent()
}

// handleUIEventBatch processes keys/mouse/resize before interrupts so Ctrl-C
// is not stuck behind a flood of output PostEvents.
// Returns true if the batch needs an immediate paint (input / resize).
func (app *App) handleUIEventBatch(batch []tcell.Event) bool {
	urgent := false
	var keys, mice, resizes, interrupts, other []tcell.Event
	for _, ev := range batch {
		switch ev.(type) {
		case *tcell.EventKey:
			keys = append(keys, ev)
			urgent = true
		case *tcell.EventMouse:
			mice = append(mice, ev)
			urgent = true
		case *tcell.EventResize:
			resizes = append(resizes, ev)
			urgent = true
		case *tcell.EventInterrupt:
			interrupts = append(interrupts, ev)
		default:
			other = append(other, ev)
			urgent = true
		}
	}
	for _, ev := range keys {
		app.HandleEvent(ev)
	}
	for _, ev := range mice {
		app.HandleEvent(ev)
	}
	for _, ev := range resizes {
		app.HandleEvent(ev)
	}
	for _, ev := range other {
		app.HandleEvent(ev)
	}
	for _, ev := range interrupts {
		app.HandleEvent(ev)
	}
	return urgent
}

func (app *App) present() {
	app.Draw(Canvas{
		rect: app.canvas.Rect(),
		grid: app.frontBuffer,
	})
	app.frontBuffer.Draw(app.screen)
	app.frontBuffer.ApplySystemCursor(app.screen)
	app.screen.Show()
}

func (app *App) UpdateCanvas() Canvas {
	app.screen.Sync()
	w, h := app.screen.Size()
	app.frontBuffer = NewGrid(w, h)
	app.canvas = Canvas{rect: NewRect(0, 0, w, h), grid: app.frontBuffer}
	// Mouse routing hit-tests widget rects between frames, so the new geometry
	// has to be there before the first paint on the resized canvas.
	app.widgets.BuildLayout(app.canvas)
	return app.canvas
}

func (app *App) MarkLayoutDirty() {
	app.layoutDirty = true
}

func (app *App) Draw(c Canvas) {
	if app.layoutDirty {
		c.grid.Clear()
		app.layoutDirty = false
	}
	c.grid.HideCursor()

	app.widgets.BuildLayout(c)
	app.widgets.Draw(c)
	if app.mouseActive && !c.grid.nativeCursorSet {
		c.grid.ShowCursor(app.mouseX, app.mouseY)
	}

}

func (app *App) Screen() tcell.Screen {
	return app.screen
}

const redrawInterrupt = "termforge-redraw"
const frameInterrupt = "termforge-frame"

func (app *App) RequestRedraw() {
	app.screen.PostEvent(tcell.NewEventInterrupt(redrawInterrupt))
}

func (app *App) RequestFrame() {
	app.layoutDirty = true
	app.screen.PostEvent(tcell.NewEventInterrupt(frameInterrupt))
}

// PostInterrupt queues payload on the UI thread (worker-safe via tcell).
func (app *App) PostInterrupt(payload any) {
	if app == nil || app.screen == nil {
		return
	}
	_ = app.screen.PostEvent(tcell.NewEventInterrupt(payload))
}

func (a *App) HandleEvent(ev tcell.Event) {

	switch e := ev.(type) {

	case *tcell.EventKey:
		a.mouseActive = false
		a.HandleKey(e)

	case *tcell.EventMouse:
		a.mouseX, a.mouseY = e.Position()
		if e.Buttons()&tcell.ButtonPrimary != 0 {
			a.mouseActive = false
		} else if e.Buttons()&(tcell.WheelUp|tcell.WheelDown) != 0 {
			a.mouseActive = false
		} else if e.Buttons() == tcell.ButtonNone {
			a.mouseActive = true
		}
		if a.Api != nil {
			a.Api.HandleMouse(e)
		}

	case *tcell.EventResize:
		// UpdateCanvas rebuilds the chrome geometry for the new size; panes get
		// theirs from the layout on the next paint. Apps have nothing to add.
		_ = a.UpdateCanvas()
		a.layoutDirty = true

	case *tcell.EventInterrupt:
		switch data := e.Data().(type) {
		case string:
			if data == redrawInterrupt {
				_ = a.UpdateCanvas()
				return
			}
			if data == frameInterrupt {
				return
			}
		}
		if a.Api != nil {
			a.Api.HandleInterrupt(e)
		}

	case *tcell.EventClipboard, *tcell.EventPaste:
		// Command/completion mode: only the cmdline should receive paste
		// (GDB may still be the focused tab leaf).
		if a.Mode() == platform.ModeCommand || a.Mode() == platform.ModeCompletion {
			a.widgets.ForEach(func(w Widget) {
				if _, ok := w.(*CmdWidget); ok {
					w.HandleEvent(e)
				}
			})
			return
		}
		a.widgets.HandleEvent(ev)
	}

}

func (app *App) CopyToClipboard(text string) {
	if text == "" {
		return
	}
	app.screen.SetClipboard([]byte(text))
	platform.SetClipboardText(text)
	// Middle-click outside gdbforge reads X11 PRIMARY; CLIPBOARD alone is not enough.
	platform.SetPrimaryText(text)
}

func (app *App) PasteFromClipboard() string {
	if text, ok := platform.GetClipboardText(); ok {
		return text
	}
	return ""
}

func (app *App) PasteFromPrimary() string {
	if text, ok := platform.GetPrimaryText(); ok {
		return text
	}
	// Fallback when PRIMARY is empty (e.g. copy from an app that only sets CLIPBOARD).
	return app.PasteFromClipboard()
}

// ClipboardIO returns the shared bridge for Viewport-backed widgets.
func (app *App) ClipboardIO() ClipboardIO {
	return ClipboardIO{
		Copy:         app.CopyToClipboard,
		Paste:        app.PasteFromClipboard,
		PastePrimary: app.PasteFromPrimary,
	}
}
