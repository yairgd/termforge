//go:build unix

// Package detach lets a program keep running when its terminal goes away, and be picked up
// again from another terminal (an ssh session that dropped, a closed window) without tmux.
//
// The program is started inside a PTY owned by a small server process, which listens on a
// Unix socket and relays the screen output and the keyboard input to whoever is attached.
// While nobody is attached the program keeps running and its output is dropped. On attach
// the server turns the terminal modes the program had on (alternate screen, mouse, bracketed
// paste ...) back on and sends the program a SIGWINCH, so a termforge app repaints the whole
// screen. One terminal is attached at a time; a new attach replaces the old one.
//
// An application opts in with one call at the top of main, before it parses its flags:
//
//	func main() {
//		if handled, err := detach.Handle("myapp"); handled {
//			if err != nil {
//				fmt.Fprintln(os.Stderr, err)
//				os.Exit(1)
//			}
//			return
//		}
//		... start the app as usual
//	}
//
// which adds to the command line of the program:
//
//	myapp --session NAME     attach to session NAME, starting it first if it does not exist
//	myapp --attach NAME      attach to a running session only
//	myapp --sessions         list the running sessions
//
// Ctrl-] detaches (WithDetachKey(CtrlKey('\\')) picks another); the program keeps running. Quitting the program ends the session.
// Without these flags Handle does nothing and returns false.
//
// Serve and Attach are the two halves, for programs that want their own command line.
package detach
