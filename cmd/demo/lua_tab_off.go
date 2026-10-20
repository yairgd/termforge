//go:build !lua

package main

// The Lua tab is opt-in (go build -tags lua; task build:lua), so the default demo needs
// no C compiler.
func (a *DemoApp) addLuaTab() error { return nil }

func (a *DemoApp) onLuaEnded() {}
