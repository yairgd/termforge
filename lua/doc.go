// Package lua embeds C Lua 5.4 in the process: Lua's own REPL (lua_main) runs on a PTY
// slave, so a termforge CompositeTerminal can show it the way it shows any program, and Go
// functions can be registered as Lua libraries (Register).
//
// The interpreter sources are vendored under lua-5.4.6/ and built as one unit by
// lua_all.c, unmodified; see that file for how to switch version. There is one Lua per
// process (the console state is static C). Line editing and history come from libedit.
//
// The package is opt-in: it needs a C compiler and libedit, so everything here is behind
// the "lua" build tag (go build -tags lua). Without the tag the package is empty and the
// rest of the module builds with no C dependency.
package lua
