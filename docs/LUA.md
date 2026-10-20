---
description: The lua package — C Lua 5.4 embedded through cgo, its REPL on a PTY shown in a CompositeTerminal, and Go functions exposed to Lua.
---

# Embedded Lua

Package `github.com/yairgd/termforge/lua` embeds the reference C interpreter in the process.
Nothing outside the package needs to know it is C.

## Opt-in build

The package needs a C compiler and libedit, so it is behind the `lua` build tag and nothing
else in termforge depends on it. Without the tag it is an empty package, `go build ./...`
needs no C toolchain, and projects that only use the framework (gdbforge, …) are unaffected.

```bash
task build:lua   # go build -tags lua ./...  and the demo with its Lua tab
task run:lua     # run the demo with the Lua tab
task test:lua    # vet + test including the Lua package
```

An application that embeds Lua builds with `-tags lua` itself.

## Layout

| Path | What |
|------|------|
| `lua/lua-5.4.6/` | Upstream sources, unmodified |
| `lua/lua_all.c` | Builds them as one unit and sets the console hooks (print, readline, history) |
| `lua/plugin.c`, `lua/lua.go` | The Go plugin API and the cgo glue |

To try another Lua version, add its directory next to `lua-5.4.6/`, change the include
paths in `lua_all.c` and the `-I` in `lua.go`.

## Console

`lua.Run(slave)` runs Lua's own REPL (`lua_main`) on the PTY slave and returns at EOF. The
host opens the PTY with `ptyx.Open()` and attaches it with
`CompositeTerminal.AttachTTY`, exactly as for any child program. Line editing and history
come from libedit (the readline API), so they behave like the stock `lua` binary.
`lua.Interrupt()` stops the running chunk ("interrupted!"); there is no foreground process
for Ctrl-C to signal on a PTY, so the host calls it on Ctrl-C.

## Calling Go from Lua

```go
lua.Register("mat", map[string]lua.Func{
    "plot": func(c *lua.Call) (int, error) {
        x, _ := c.Int(1)
        y, _ := c.Int(2)
        draw(x, y)
        return 0, nil // number of values pushed
    },
})
```

`Register` before `Run`; the library is a global table of that name (`""` puts the functions
in the globals). A returned error becomes a Lua error. `Call` offers `Number`, `Int`,
`String` and the matching `Push…` methods. Functions run on the Lua thread.

## Limits

- One Lua per process; `os.exit` ends the application.
- A long-running chunk blocks the REPL until it ends or is interrupted.
- Building with `-tags lua` needs a C compiler and libedit (`dev-libs/libedit`, `libedit-dev`).

## Demo

With `-tags lua`, the demo (`cmd/demo`) has a third tab: a 64×32 pixel matrix over the console. Try
`mat.line(0,31,63,0, 0x00ff00)` or `for x=0,63 do mat.plot(x,x//2,0xff0000) end`.
Colours are `0xRRGGBB` (`0x123456` is r=0x12, g=0x34, b=0x56), white by default.
Library: `mat.plot`, `mat.line`, `mat.clear`, `mat.size`, `mat.delay`, and the effects `mat.fire([frames [, fps [, cooling]]])` and `mat.plasma([frames [, fps]])` (run until Ctrl-C unless frames is given). `help()` lists everything, `help("fire")` one entry.
