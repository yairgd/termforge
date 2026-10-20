package main

// luaEnded is posted when the Lua REPL returns.
type luaEnded struct{}

// luaPane is the Lua console pane as the app sees it, so app.go builds without the lua tag.
type luaPane interface {
	Slave() string
	Note(string)
	Close()
}
