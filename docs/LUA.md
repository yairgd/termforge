---
description: The lua package — C Lua 5.4 embedded through cgo, its REPL on a PTY shown in a CompositeTerminal, and Go functions exposed to Lua.
---

# Embedded Lua

Package `github.com/yairgd/termforge/lua` embeds the reference C Lua interpreter in the
process. Lua's own REPL runs on a PTY, so a `CompositeTerminal` shows it like any other
program, and Go functions can be registered as Lua libraries. Nothing outside the package
needs to know it is C.

```text
 CompositeTerminal  <── bytes ──  PTY master ─┐
 (xterm view, keys) ── keys ───►              │ ptyx.Open()
                                   PTY slave ─┘
                                       ▲  lua.Run(slave)
                          ┌────────────┴──────────────┐
                          │ lua_main (C, one thread)   │   libedit: line editing, history
                          │   tf_trampoline (C)        │
                          └────────────┬──────────────┘
                                       ▼  //export tfCall
                              your lua.Func (Go)
```

## Opt-in build

The package needs a C compiler and libedit, so it is behind the `lua` build tag and nothing
else in termforge depends on it. Without the tag it is an empty package, `go build ./...`
needs no C toolchain, and projects that only use the framework are unaffected.

```bash
task build:lua   # go build -tags lua ./...  and the demo with its Lua tab
task run:lua     # run the demo with the Lua tab
task test:lua    # vet + test including the Lua package
```

An application that embeds Lua builds with `-tags lua` itself. libedit is
`dev-libs/libedit` on Gentoo and `libedit-dev` on Debian/Ubuntu.

## Layout

| Path | What |
|------|------|
| `lua/lua-5.4.6/` | Upstream sources, unmodified |
| `lua/lua_all.c` | Builds them as one unit and sets the console hooks (print, readline, history, interrupt) |
| `lua/plugin.c`, `lua/lua.go` | The cgo glue and the Go plugin API |
| `lua/doc.go` | Package documentation, the only file built without the tag |

Nothing inside the Lua sources is edited: every hook the interpreter offers is a macro that
`lua.c` and `lauxlib.h` only define when the host has not, and `lua_all.c` defines them. To
try another Lua version, add its directory next to `lua-5.4.6/`, change the include paths in
`lua_all.c` and the `-I` in `lua.go`.

## Console

`lua.Run(slave)` runs Lua's own REPL (`lua_main`) on the PTY slave and returns at EOF, on
the calling goroutine, which it locks to its OS thread. The host opens the PTY with
`ptyx.Open()` and attaches it with `CompositeTerminal.AttachTTY`, exactly as for any child
program. Line editing and history come from libedit (the readline API), so they behave like
the stock `lua` binary. Every `Run` must use the same slave path, because libedit keeps
pointers to the console files.

Ctrl-C: a PTY has no foreground process to signal, so the host calls `lua.Interrupt()`. Lua
stops at its next instruction with `interrupted!` and the REPL carries on. A Go function
that runs for a long time never reaches a Lua instruction, so it polls `lua.Interrupts()`
(a counter that goes up on every `Interrupt`): read it on entry and return once it changed.

## Calling Go from Lua

```go
lua.Register("demo", map[string]lua.Func{
    "add": func(c *lua.Call) (int, error) {
        a, ok1 := c.Number(1)
        b, ok2 := c.Number(2)
        if !ok1 || !ok2 {
            return 0, errors.New("add: two numbers expected")
        }
        c.PushNumber(a + b)
        return 1, nil // how many values were pushed
    },
})
```

```text
> demo.add(2, 3)
5
> demo.add("a", 1)
stdin:1: add: two numbers expected
```

- `Register` before `Run`; the library is a global table of that name, and `""` puts the
  functions in the globals.
- `Call` offers `Top`, `Number`, `Int`, `String`, `PushNumber`, `PushInt`, `PushString`,
  `PushBool`, `PushNil` and `Print`. Arguments are numbered from 1, as in Lua.
- A returned error, or a panic, becomes a Lua error. The error is raised from C after Go has
  returned, so Lua never longjmps through a Go frame.
- `Call` is valid only while the function runs. Functions execute on the Lua thread: guard
  state that the UI thread also touches.

Only numbers, strings and booleans cross the boundary today; there are no tables and a Go
function cannot call back into a Lua function.

## Limits

- One Lua per process (the console state is static C); `os.exit` ends the application.
- A long-running chunk blocks the REPL until it ends or is interrupted.
- Only the Lua thread may use the Lua state; other goroutines cannot call into it.

## Demo

With `-tags lua`, the demo (`cmd/demo`) has a third tab: a 64×32 pixel matrix, centered,
over the Lua console. `internal/demo/matrix_lua.go` registers the `mat` library and a global
`help`.

| Call | Does |
|------|------|
| `mat.plot(x, y [, color])` | Set one pixel; `color` is `0xRRGGBB` (`0x123456` is r=0x12, g=0x34, b=0x56), white by default |
| `mat.line(x0, y0, x1, y1 [, color])` | Draw a line |
| `mat.clear()` | Blank the matrix |
| `mat.size()` | Returns width, height |
| `mat.delay(ms)` | Sleep, for hand-made animations |
| `mat.fire([frames [, fps [, cooling]]])` | Fire effect; `cooling` 0–100 (default 55), higher means lower flames |
| `mat.plasma([frames [, fps]])` | Plasma effect |
| `help([name])` | List everything, or describe one entry, e.g. `help("fire")` |

The effects run until Ctrl-C unless `frames` is given; `fps` defaults to 30. Try
`for x=0,63 do mat.plot(x,x//2,0xff0000) end` or `mat.line(0,31,63,0,0x00ff00)`.

In the demo, `Ctrl-D` and `:q!` leave the whole application, so the console is never closed
by an accidental end-of-file; if Lua does exit (for example `os.exit`), the tab restarts it
with a fresh state.
