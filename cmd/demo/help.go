package main

import "strings"

// demoCommand documents one DSL command. termforge's CommandNode carries no
// description field, so the demo keeps help as its own data rather than
// changing the framework's command tree.
type demoCommand struct {
	Name  string
	Usage string
	Help  string
}

// Declaration order is display order in the help window.
var demoCommands = []demoCommand{
	{"window", ":window left|right|up|down", "Move focus to the adjacent pane."},
	{"b", ":b main|table|side|log", "Focus a pane by name."},
	{"vs", ":vs", "Split the focused pane side by side."},
	{"split", ":split", "Split the focused pane stacked."},
	{"close", ":close", "Delete the focused pane."},
	{"only", ":only", "Collapse the tree to the focused pane."},
	{"equal", ":equal", "Reset every split to an even ratio."},
	{"clear", ":clear", "Clear the focused pane's contents."},
	{"help", ":help [command]", "Open this window, or help for one command."},
	{"quit", ":quit", "Close the pane; exit when it is the last."},
}

func findCommand(name string) (demoCommand, bool) {
	for _, c := range demoCommands {
		if c.Name == name {
			return c, true
		}
	}
	return demoCommand{}, false
}

func commandNames() []string {
	out := make([]string, 0, len(demoCommands))
	for _, c := range demoCommands {
		out = append(out, c.Name)
	}
	return out
}

// helpForCommand is the ':help <command>' page.
func helpForCommand(c demoCommand) []string {
	return []string{
		c.Usage,
		"",
		c.Help,
		"",
		"Run ':help' with no argument for the full reference.",
	}
}

// helpOverview is the ':help' page: what the demo shows and how to drive it.
func helpOverview() []string {
	lines := []string{
		"termforge demo — a host application with no debugger in it.",
		"",
		"Four panes in a nested split tree, a ':' command line, mouse-driven",
		"focus and resize, and this floating window. Everything here is",
		"framework machinery; the demo only supplies panes and commands.",
		"",
		"COMMANDS",
	}
	lines = append(lines, commandTable()...)
	lines = append(lines,
		"",
		"MOUSE",
		"  click pane          focus it",
		"  click status label  double-click copies the pane name",
		"  drag separator      resize the two adjacent panes only",
		"  wheel               scroll the pane under the pointer",
		"  drag in pane        select text; middle-click pastes",
		"  click cmdline       focus it and place the caret",
		"",
		"KEYS",
		"  ?                   open this window",
		"  :                   command line",
		"  Tab                 complete a command or pane name",
		"  Esc                 back to normal mode / close this window",
		"  Ctrl-W h/j/k/l      move focus",
		"  j/k, wheel          scroll the focused pane",
		"  Ctrl-D              exit, however many panes are open",
	)
	return lines
}

// commandTable renders the command list with the help column aligned.
func commandTable() []string {
	width := 0
	for _, c := range demoCommands {
		if n := len(c.Usage); n > width {
			width = n
		}
	}
	out := make([]string, 0, len(demoCommands))
	for _, c := range demoCommands {
		pad := strings.Repeat(" ", width-len(c.Usage))
		out = append(out, "  "+c.Usage+pad+"   "+c.Help)
	}
	return out
}
