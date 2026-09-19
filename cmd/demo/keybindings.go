package main

import (
	"github.com/yairgd/termforge/commands"
	"github.com/yairgd/termforge/platform"
)

func (a *DemoApp) InitKeyBindings() {
	a.keyBindings = commands.NewKeyBindingRegistry()
	a.insertKeys = commands.NewKeyBindingRegistry()

	a.keyBindings.Bind(
		commands.NewCommand("move-left", func(args ...any) { a.OnFocusLeft() }),
		"<C-w>h", "<C-w><Left>",
	)
	a.keyBindings.Bind(
		commands.NewCommand("move-right", func(args ...any) { a.OnFocusRight() }),
		"<C-w>l", "<C-w><Right>",
	)
	a.keyBindings.Bind(
		commands.NewCommand("move-up", func(args ...any) { a.OnFocusUp() }),
		"<C-w>k", "<C-w><Up>",
	)
	a.keyBindings.Bind(
		commands.NewCommand("move-down", func(args ...any) { a.OnFocusDown() }),
		"<C-w>j", "<C-w><Down>",
	)
	a.keyBindings.Bind(
		commands.NewCommand("tab-next", func(args ...any) { a.NextTab() }),
		"gt",
	)
	a.keyBindings.Bind(
		commands.NewCommand("tab-prev", func(args ...any) { a.PrevTab() }),
		"gT",
	)
	a.keyBindings.Bind(
		commands.NewCommand("escape", func(args ...any) { a.onEscape() }),
		"<Esc>",
	)
	a.keyBindings.Bind(
		commands.NewCommand("command-mode", func(args ...any) { a.enterCommandMode() }),
		":",
	)
	a.keyBindings.Bind(
		commands.NewCommand("insert", func(args ...any) { a.EnterInsertMode() }),
		"i",
	)
	a.keyBindings.Bind(
		commands.NewCommand("help", func(args ...any) { a.OnHelp() }),
		"?",
	)
	a.keyBindings.Bind(
		commands.NewCommand("quit", func(args ...any) { a.ForceQuit() }),
		"<C-d>",
	)

	a.insertKeys.Bind(
		commands.NewCommand("escape", func(args ...any) { a.onEscape() }),
		"<Esc>",
	)
}

func (a *DemoApp) onEscape() {
	if a.help.Visible() {
		a.hideHelp()
		return
	}
	if a.Mode() == platform.ModeCommand {
		a.leaveCommandMode()
		return
	}
	if lay := a.Layout(); lay != nil {
		lay.SetInsertActive(false)
	}
	a.SetMode(platform.ModeNormal)
	a.RequestFrame()
}

func (a *DemoApp) enterCommandMode() {
	if lay := a.Layout(); lay != nil {
		lay.SetInsertActive(false)
	}
	if a.cmdWidget != nil && !a.cmdWidget.Active() {
		a.cmdWidget.Activate()
	}
	a.SetMode(platform.ModeCommand)
	a.RequestFrame()
}

func (a *DemoApp) leaveCommandMode() {
	if a.cmdWidget != nil {
		a.cmdWidget.Deativate()
	}
	a.SetMode(platform.ModeNormal)
	a.RequestFrame()
}
