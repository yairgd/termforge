//go:build lua

package lua

/*
#cgo CFLAGS: -I${SRCDIR}/lua-5.4.6/src -I${SRCDIR}/lua-5.4.6 -DLUA_USE_POSIX -DUSE_ORIGNAL_LUA_CODE
#cgo LDFLAGS: -lm
#include <stdlib.h>
#include "plugin.h"
*/
import "C"

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"
)

// Func is a Go function callable from Lua. It returns how many values it pushed
// (the Push methods); a non-nil error is raised in Lua as an error message.
type Func func(c *Call) (int, error)

// Call is the Lua stack of one call into Go; it is valid only while the Func runs.
// Arguments are 1-based, like Lua's own API.
type Call struct{ l *C.lua_State }

type lib struct {
	name  string
	names []string
	idx   []int
}

var (
	mu    sync.Mutex
	libs  []lib
	funcs []Func
)

// Register makes funcs available to every Lua state created afterwards, as fields of the
// global table lib (lib "" puts them in the globals). Call it before Run.
func Register(name string, fs map[string]Func) {
	mu.Lock()
	defer mu.Unlock()
	l := lib{name: name}
	for n, f := range fs {
		l.names = append(l.names, n)
		l.idx = append(l.idx, len(funcs))
		funcs = append(funcs, f)
	}
	libs = append(libs, l)
}

// Run executes the Lua REPL on the PTY slave at path, on the calling goroutine (which is
// locked to its thread), until the console sees EOF. Every call must use the same path.
func Run(slave string) int {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	cs := C.CString(slave)
	defer C.free(unsafe.Pointer(cs))
	return int(C.tf_run(cs))
}

var interrupts atomic.Uint64

// Interrupt stops the chunk being executed ("interrupted!"); the REPL keeps going. Lua stops
// at its next instruction; a Go function that runs long should poll Interrupts instead.
func Interrupt() {
	interrupts.Add(1)
	C.tf_interrupt()
}

// Interrupts counts Interrupt calls. A long-running Go function reads it on entry and
// returns once it has changed.
func Interrupts() uint64 { return interrupts.Load() }

//export tfOpen
func tfOpen(l *C.lua_State) {
	mu.Lock()
	defer mu.Unlock()
	for _, lb := range libs {
		if lb.name != "" {
			C.lua_createtable(l, 0, C.int(len(lb.names)))
		}
		for i, n := range lb.names {
			C.tf_push_func(l, C.int(lb.idx[i]))
			cn := C.CString(n)
			if lb.name == "" {
				C.lua_setglobal(l, cn)
			} else {
				C.lua_setfield(l, -2, cn)
			}
			C.free(unsafe.Pointer(cn))
		}
		if lb.name != "" {
			cn := C.CString(lb.name)
			C.lua_setglobal(l, cn)
			C.free(unsafe.Pointer(cn))
		}
	}
}

//export tfCall
func tfCall(l *C.lua_State, idx C.int, errmsg **C.char) (n C.int) {
	defer func() {
		if r := recover(); r != nil {
			*errmsg = C.CString(fmt.Sprint("go panic: ", r))
		}
	}()
	mu.Lock()
	f := funcs[idx]
	mu.Unlock()
	nret, err := f(&Call{l})
	if err != nil {
		*errmsg = C.CString(err.Error())
		return 0
	}
	return C.int(nret)
}

// Top is the number of arguments.
func (c *Call) Top() int { return int(C.lua_gettop(c.l)) }

// Number returns argument i as a float; ok is false if it is not a number (or numeric string).
func (c *Call) Number(i int) (float64, bool) {
	var ok C.int
	v := C.lua_tonumberx(c.l, C.int(i), &ok)
	return float64(v), ok != 0
}

// Int returns argument i as an integer; ok is false unless it has an integer value.
func (c *Call) Int(i int) (int, bool) {
	var ok C.int
	v := C.lua_tointegerx(c.l, C.int(i), &ok)
	return int(v), ok != 0
}

// String returns argument i if it is a string or a number.
func (c *Call) String(i int) (string, bool) {
	var n C.size_t
	p := C.lua_tolstring(c.l, C.int(i), &n)
	if p == nil {
		return "", false
	}
	return C.GoStringN(p, C.int(n)), true
}

func (c *Call) PushNumber(v float64) { C.lua_pushnumber(c.l, C.lua_Number(v)) }
func (c *Call) PushInt(v int)        { C.lua_pushinteger(c.l, C.lua_Integer(v)) }
func (c *Call) PushBool(v bool) {
	b := C.int(0)
	if v {
		b = 1
	}
	C.lua_pushboolean(c.l, b)
}
func (c *Call) PushNil() { C.lua_pushnil(c.l) }
func (c *Call) PushString(s string) {
	if s == "" {
		C.lua_pushlstring(c.l, nil, 0)
		return
	}
	C.lua_pushlstring(c.l, (*C.char)(unsafe.Pointer(unsafe.StringData(s))), C.size_t(len(s)))
}

// Print writes s through Lua's print, so it lands on the console like any Lua output.
func (c *Call) Print(s string) {
	cn := C.CString("print")
	C.lua_getglobal(c.l, cn)
	C.free(unsafe.Pointer(cn))
	c.PushString(s)
	if C.lua_pcallk(c.l, 1, 0, 0, 0, nil) != 0 {
		C.lua_settop(c.l, -2) // drop the error message
	}
}
