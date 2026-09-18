```text
████████╗███████╗██████╗ ███╗   ███╗███████╗ ██████╗ ██████╗  ██████╗ ███████╗
╚══██╔══╝██╔════╝██╔══██╗████╗ ████║██╔════╝██╔═══██╗██╔══██╗██╔════╝ ██╔════╝
   ██║   █████╗  ██████╔╝██╔████╔██║█████╗  ██║   ██║██████╔╝██║  ███╗█████╗
   ██║   ██╔══╝  ██╔══██╗██║╚██╔╝██║██╔══╝  ██║   ██║██╔══██╗██║   ██║██╔══╝
   ██║   ███████╗██║  ██║██║ ╚═╝ ██║██║     ╚██████╔╝██║  ██║╚██████╔╝███████╗
   ╚═╝   ╚══════╝╚═╝  ╚═╝╚═╝     ╚═╝╚═╝      ╚═════╝ ╚═╝  ╚═╝ ╚═════╝ ╚══════╝
        >> termforge: Vim-inspired terminal UI framework for Go <<
```

# termforge

**termforge** is a terminal application framework written in Go on top of
[tcell](https://github.com/gdamore/tcell). It provides the parts that every serious
full-screen TUI ends up rewriting: a recursive split-tree workspace, tabs, a `:` command
line with tab completion, Vim-style modes and key chords, mouse and clipboard support,
and a real xterm emulator pane for running child processes.

Your application supplies its models, its widgets, and its commands. termforge supplies
the structure and stays out of your domain.

```go
app := termforge.NewApp()
app.InitB(myApp)                  // myApp implements termforge.AppApi
app.AddWidget(tabWidget)          // fills what the rows below leave over
app.AddRowWidget(cmdWidget, 1)    // one-row band at the bottom
app.Run()
```

## What you get

| Area | Pieces |
|------|--------|
| **Windowing** | Binary split tree, drag-to-resize separators, `equalalways` rebalancing, named leaf marks, tabs, jump list support |
| **Chrome** | Command line, wildmenu completion bar, per-pane status lines — all plain widgets, no popup compositor |
| **Commands** | Hierarchical command tree, incremental parser, prefix completion, rest-args leaves, dynamic completion callbacks |
| **Input** | Interaction modes, multi-key sequence trie (`<C-w>h`), mouse selection, CLIPBOARD and PRIMARY integration |
| **Panes** | `Viewport` over a line buffer, `TableWidget` for columnar lists, `CompositeTerminal` for PTY-backed processes, `ConsolePane` for line REPLs, `LoggerWidget` |
| **Plumbing** | PTY sessions (`ptyx`), child-process tracking, output coalescing, serial ports (`devport`) |

## Install

```bash
go get github.com/yairgd/termforge
```

## Try the demo

```bash
go run ./cmd/demo
```

A host application with no debugger in it: four panes in a nested split tree, a
`:` command line with Tab completion, a floating help window, and mouse-driven
focus, resize, scrolling and selection.

```text
┌──────────────┬──────────────┐
│              │ table        │
│ main         ├──────────────┤
│              │ side         │
├──────────────┴──────────────┤
│ log                         │
└─────────────────────────────┘
```

Press `?` or run `:help` for the full reference, `:help <command>` for one
command. Click a pane to focus it, drag a separator to resize just the two
adjacent panes, and use the wheel to scroll whatever is under the pointer.
`:vs` and `:split` add panes; `:close`, `:only` and `:equal` rearrange them.

[docs/DEMO.md](docs/DEMO.md) explains what the demo shows, how the panes,
commands and floating window are wired, and which parts are framework
behavior rather than application code.

## Documentation

Full documentation: <https://yairgd.github.io/termforge/>

| Document | Covers |
|----------|--------|
| [docs/UI_ARCHITECTURE.md](docs/UI_ARCHITECTURE.md) | Widgets, widget tree, layout engine, canvas, focus, frame loop |
| [docs/WINDOW_MANAGEMENT.md](docs/WINDOW_MANAGEMENT.md) | Workspace, split tree, tabs, chrome bands, command line |
| [docs/RENDERING.md](docs/RENDERING.md) | Cells, grid, borders, Unicode, ANSI paths, screen sync |
| [docs/INPUT.md](docs/INPUT.md) | Event types, dispatch order, modes, key sequences, mouse |
| [docs/COMMAND_SYSTEM.md](docs/COMMAND_SYSTEM.md) | Command tree, parser, DSL, completion, key bindings |

## Package layout

```text
github.com/yairgd/termforge            engine: App, Widget, WidgetTree, Canvas, Grid, widgets
                        /platform      base layer: AppState, Buffer, EventBus, Logger, keys, theme
                        /commands      command tree, parser, DSL, key bindings
                        /collections   Trie
                        /ptyx          PTY sessions, TTY allocation, output draining
                        /execcli       run a child process on a PTY
                        /devport       serial port helper
cmd/demo                               runnable example application
internal/demo                          its panes, layout and help window
```

`platform` is the dependency-free base layer and must not import the engine root or any
sibling package. `scripts/check_imports.sh` enforces this, plus the rule that no
termforge package may import a consuming application.

## Development

With [go-task](https://taskfile.dev) (`go install github.com/go-task/task/v3/cmd/task@latest`):

```bash
task            # build packages + bin/demo
task run        # run the demo
task check      # everything CI runs: build, vet, test, import guardrails, gofmt
task --list     # all tasks
```

Documentation:

```bash
task docs:setup    # one-time: create .venv-docs/ from requirements-docs.txt
task docs          # serve locally on http://127.0.0.1:8775 with live reload
task docs:export   # build the static site into _site/
task docs:check    # build and fail on broken internal links or missing anchors
task docs:preview  # serve the production build on http://127.0.0.1:8776
```

Ports `8775`/`8776` are termforge's; gdbforge uses `8765`/`8766`, so both doc sites can
run at the same time. Override with `task docs DOCS_PORT=9000`.

Without `task`:

```bash
go build ./... && go vet ./... && go test ./... && ./scripts/check_imports.sh
./docs/setup-docs-venv.sh && ./docs/serve.sh
```

## Origin, and the related project: gdbforge

termforge was extracted from **[gdbforge](https://github.com/yairgd/gdbforge)**, a
Vim-inspired multi-pane terminal debugger for GDB and Delve, which remains its first and
largest consumer. Anything debugger-specific — GDB/Delve backends, breakpoints, MI parsing,
inferior PTY handling, Lua target workflows for embedded and kernel debugging — stayed in
that repository.

The two projects have companion documentation sites:

| Looking for | Go to |
|-------------|-------|
| Building a terminal app on this framework | **[termforge docs](https://yairgd.github.io/termforge/)** |
| A large real application built on it, and debugging with GDB/Delve | **[gdbforge docs](https://yairgd.github.io/gdbforge/)** — [architecture](https://yairgd.github.io/gdbforge/ARCHITECTURE/), [debugger integration](https://yairgd.github.io/gdbforge/DEBUGGER_INTEGRATION/) |

## License

See [LICENSE](LICENSE).
