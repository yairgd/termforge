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
                          │ lua_main (C, one thread)   │   linenoise: line editing, history
                          │   tf_trampoline (C)        │
                          └────────────┬──────────────┘
                                       ▼  //export tfCall
                              your lua.Func (Go)
```

## Opt-in build

The package needs a C compiler, so it is behind the `lua` build tag and nothing
else in termforge depends on it. Without the tag it is an empty package, `go build ./...`
needs no C toolchain, and projects that only use the framework are unaffected.

```bash
task build:lua   # go build -tags lua ./...  and the demo with its Lua tab
task run:lua     # run the demo with the Lua tab
task test:lua    # vet + test including the Lua package
```

An application that embeds Lua builds with `-tags lua` itself. No library has to be
installed: Lua and the line editor are vendored and compiled in, and the binary links only
libc and libm.

## Layout

| Path | What |
|------|------|
| `lua/lua-5.4.6/` | Upstream sources, unmodified |
| `lua/linenoise/` | [linenoise](https://github.com/antirez/linenoise) line editor (BSD), patched to run on the console fds (see its `README`) |
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
program. Line editing and history come from the vendored linenoise (arrows, Home/End,
Ctrl-A/E/K/U/W, Up/Down history); Ctrl-C at the prompt drops the line. Every `Run` must use
the same slave path, because the console files are opened once per process.

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

## Services (Lua, REST and MCP from one definition)

`termforge/service` is a registry of named functions: name, help, typed params and a handler.
Define each function once, then expose the registry to every caller:

```go
reg := &service.Registry{}
reg.Add(service.Service{
    Name: "ps", Help: "list processes",
    Run: func(ctx context.Context, a service.Args) (any, error) { return rows, nil },
})
lua.Register("app", lua.ServiceFuncs(reg)) // app.ps() in the console
go http.Serve(ln, reg.Handler())           // REST and MCP, bind to loopback only
```

- **Lua:** arguments are matched to the params in order, and a variadic param takes the rest.
  A returned map, struct or slice becomes a table; printed in the console it shows one readable
  line per element, and scripts can still index it. `Call.Push(v)` does the same conversion for
  any Go value that encodes to JSON.
- **REST:** `GET /api` lists the services, `POST /api/<name>` takes JSON arguments and answers
  `{"result": ...}` or `{"error": ...}`.
- **MCP:** `POST /mcp` speaks JSON-RPC 2.0 (`initialize`, `ping`, `tools/list`, `tools/call`)
  with plain JSON replies, so an AI agent can discover each function and read its help.
- `reg.Help(topic)` gives the text for a `help()` function; `reg.ToolSchemas()` gives the
  name, description and JSON schema of each service.

Services can run commands, so serve them on a loopback address and only on request.

## Demo

With `-tags lua`, the demo (`cmd/demo`) has a third tab: a 64×32 pixel matrix over the Lua
console. The matrix sits in a `termforge.Sheet`: centered when the pane is large, otherwise
scrollable (wheel, `PgUp`/`PgDn`, `Home`/`End` vertically; horizontal wheel, Ctrl+wheel or
`Left`/`Right` sideways). `internal/demo/matrix_lua.go` registers the `mat` library and a global
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
