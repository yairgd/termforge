---
title: Changelog
description: Release history for termforge — what changed in each tagged version, and what to watch for when upgrading.
---

# Changelog

Release history for termforge. Each version is a Go module release, resolvable with
`go get github.com/yairgd/termforge@vX.Y.Z`; tags are listed on the
    10|[GitHub releases page](https://github.com/yairgd/termforge/releases).

## Unreleased

### App chrome is a layout

The App held a flat widget slice and handed the application a `HandleResize` hook to
assign rects into it by index. Both consumers wrote the same three lines against `w[0]`,
`w[1]`, `w[2]`, behind a `len(w) < 3` guard that silently skipped layout whenever the
registration order changed. Chrome is now a `WidgetsList` — the flat `Layout`
implementation, the counterpart of `WidgetTree` one level up — and placement is stated
where the widget is registered:

```go
app.AddWidget(tabWidget)               // fills the rows the fixed ones leave over
app.AddRowWidget(completionBar, 1)     // full-width band, in registration order
app.AddRowWidget(cmdWidget, 1)
app.AddFloatingWidget(help, helpRect)  // rect recomputed per frame, owns no rows
```

`BuildLayout` stacks the rows top to bottom and gives the fill widget what is left, so the
usual banding — workspace `H-2`, wildmenu on row `H-2`, cmdline on row `H-1` — falls out of
the registration order instead of being arithmetic in every app. Geometry is rebuilt on
every frame and on every `UpdateCanvas`, which also fixes the negative rect the old code
handed the workspace on a canvas shorter than the bands.

### Upgrading

- **`AppApi.HandleResize` is gone.** Delete the method. `UpdateCanvas` reallocates the grid
  and the layouts recompute from the new canvas, so there is nothing left for an app to do
  on a resize.
- **`App.Widgets()`, `WidgetNode` and `WidgetNode.SetRect` are gone.** Rect assignment
  becomes the `Add*` call that matches the placement, and widget hit-testing becomes
  `App.WidgetRect`:

  ```go
  // before — in HandleResize
  w := app.Widgets()
  w[2].SetRect(c.ChildRect(0, c.H()-1, c.W(), 1))
  // before — hit test
  for _, n := range app.Widgets() {
      if n.Widget() == cmdWidget {
          r = n.Rect()
      }
  }

  // after — in setup
  app.AddRowWidget(cmdWidget, 1)
  // after — hit test
  r := app.WidgetRect(cmdWidget)
  ```

- **`NewApp` allocates the canvas.** It calls `UpdateCanvas` itself, so the startup
  `HandleResize()` an app made after `Init` can be dropped.
- **Rows always reserve their height**, painted or not — the same as the hand-written
  bands, but there is no "collapse when inactive" placement yet.

## v0.1.0

First tagged release: a Vim-inspired terminal application framework for Go on top of
[tcell](https://github.com/gdamore/tcell), extracted from
[gdbforge](https://yairgd.github.io/gdbforge/), where this code grew in-tree as
`internal/termui` and its supporting packages.

### Highlights

    20|- **Windowing** — recursive binary split tree, drag-to-resize separators, `equalalways` rebalancing, named leaf marks, tabs; `Tab.Content` is a `Layout` interface with `WidgetTree` as the tiling implementation.
- **Chrome** — `CmdWidget` command line, wildmenu completion bar, per-pane status lines — all ordinary widgets, no popup compositor.
- **Commands** — hierarchical command tree, incremental parser, builder DSL, prefix completion, rest-args leaves, dynamic completion callbacks.
- **Input** — interaction modes, multi-key sequence trie (`<C-w>h`), mouse focus / selection / resize, CLIPBOARD and PRIMARY integration.
- **Panes** — `DocumentView` and `ScrollDocument` over a line buffer, `TableWidget` for columnar lists, `CompositeTerminal` for PTY-backed child processes, `LoggerWidget`.
- **Rendering** — widgets draw in local coordinates through `Canvas` → `Grid` → tcell, against an off-screen grid with a single front buffer.
- **Plumbing** — `ptyx` PTY sessions and TTY allocation, `execcli` to run a command on a PTY, `devport` serial helper, `CoalesceRunner` burst scheduler, `ChildProcCtl` process-group supervision.
- **Domain-free by construction** — nothing about GDB, Delve or MCP came along, and `scripts/check_imports.sh` enforces that `platform` stays the dependency-free base and that no termforge package imports a consuming application.
- **Example application** — [`cmd/demo`](DEMO.md): four panes in a nested split tree, a `:` command line with Tab completion, a floating help window, and mouse-driven focus, resize, scrolling and selection — with no debugger in it.
- **Docs and CI** — this documentation site, a Taskfile, and GitHub Actions for build / vet / test / import guardrails plus docs publishing.
    30|
### The one API change made during extraction

The framework used to hardcode debugger prompt strings, and branched its own scroll logic
on them. A terminal pane now takes that vocabulary from the application, so Home/End
versus scrollback navigation is application policy rather than framework knowledge:

```go
term.SetPromptPrefixes(termforge.PromptPrefixes{
    Console: []string{"(gdb) ", "(dlv) "},  // rows where Home/End edit the line
    Strip:   []string{"(gdb) ", "(dlv) ", "> "},
    40|})
```

termforge ships no default prompt vocabulary. See [Input](INPUT.md) for where this sits in
the dispatch order.

### Adopting v0.1.0

- **Six direct dependencies**: `tcell/v2`, `creack/pty`, `gitpod-io/xterm-go`,
  `golang.design/x/clipboard`, `golang.org/x/sys`, `go.bug.st/serial`. No debugger, no
  syntax highlighter, no Lua interpreter.
    50|- **Pre-1.0, so the API can still move.** This surface was shaped by a single consumer;
  it will be revised against the second one rather than frozen now.
- **tcell appears in the public API** — `Widget.HandleEvent(ev tcell.Event)` and
  `KeyHandler func(ev *tcell.EventKey) bool` expose it deliberately. Abstracting over one
  implementation would produce the wrong abstraction, so this is documented as a
  limitation rather than presented as a design goal. See
  [UI Architecture](UI_ARCHITECTURE.md#design-goals).
- **No history before the extraction.** termforge starts at its initial commit, so
  `git blame` does not reach back into gdbforge.
