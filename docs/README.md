---
description: Documentation for termforge, a Vim-inspired terminal UI framework for Go with split-tree windows, tabs, colon commands, and a built-in terminal emulator.
---

# termforge Documentation

**termforge** is a Vim-inspired terminal application framework written in Go on top of
[tcell](https://github.com/gdamore/tcell). It gives an application a split-tree
workspace, tabs, a colon command line with tab completion, key-sequence bindings, and
a real xterm emulator pane for child processes — so the application only has to supply
its own models, widgets, and commands.

The framework is deliberately opinionated about *structure* and silent about *domain*:
`App` owns the frame loop, `WidgetTree` owns geometry and focus, `CommandRegistry` owns
the command tree, and the application owns everything else through the `AppApi`
interface.

## Where to start

| If you want to | Read |
|----------------|------|
| Understand widgets, focus, and the frame loop | [UI_ARCHITECTURE.md](UI_ARCHITECTURE.md) |
| Lay out panes, splits, tabs, and chrome | [WINDOW_MANAGEMENT.md](WINDOW_MANAGEMENT.md) |
| Know how cells, borders, and ANSI get painted | [RENDERING.md](RENDERING.md) |
| Route keys, mouse, and interaction modes | [INPUT.md](INPUT.md) |
| Add `:commands`, completion, and key chords | [COMMAND_SYSTEM.md](COMMAND_SYSTEM.md) |

A runnable example application lives in [`cmd/demo`](https://github.com/yairgd/termforge/tree/main/cmd/demo).

---

## Package layout

| Import | Contents |
|--------|----------|
| `github.com/yairgd/termforge` | The engine: `App`, `Widget`, `WidgetTree`, `Canvas`, `Grid`, `Viewport`, `TableWidget`, `CompositeTerminal`, `CmdWidget`, `TabWidget` |
| `github.com/yairgd/termforge/platform` | Base layer with no framework dependencies: `AppState`, `Buffer`, `EventBus`, `Logger`, key parsing, theme colors |
| `github.com/yairgd/termforge/commands` | `CommandNode`, `CommandRegistry`, `CommandParser`, the builder DSL, `KeyBindingRegistry` |
| `github.com/yairgd/termforge/collections` | `Trie`, used for both command names and key sequences |
| `github.com/yairgd/termforge/ptyx` | PTY plumbing: sessions, TTY allocation, output draining |
| `github.com/yairgd/termforge/execcli` | Running a child process on a PTY behind a small client API |
| `github.com/yairgd/termforge/devport` | Serial port helper for device-backed panes |

`platform` is the dependency-free base: it must not import the engine root or any
sibling package, and `scripts/check_imports.sh` enforces that.

---

## Related

termforge was extracted from [gdbforge](https://github.com/yairgd/gdbforge), which
remains its first and largest consumer. Debugger-specific documentation — GDB and
Delve integration, breakpoints, MI parsing, Lua workflows — lives in that repository.
