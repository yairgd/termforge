package main

import (
	"fmt"
	"os"

	"github.com/yairgd/termforge/detach"
)

var version = "dev"

func main() {
	// demo --session NAME keeps the demo running when the terminal goes away; see docs/DETACH.md.
	if handled, err := detach.Handle("demo"); handled {
		if err != nil {
			fmt.Fprintf(os.Stderr, "demo: %v\n", err)
			os.Exit(1)
		}
		return
	}
	for _, a := range os.Args[1:] {
		if a == "-version" || a == "--version" {
			fmt.Println(version)
			os.Exit(0)
		}
	}
	app, err := NewDemoApp()
	if err != nil {
		fmt.Fprintf(os.Stderr, "demo: %v\n", err)
		os.Exit(1)
	}
	defer app.Close()
	app.Run()
}
