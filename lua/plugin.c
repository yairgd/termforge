//go:build lua

/* Go functions as Lua C functions: every Go func is a closure of tf_trampoline with
 * its registry index as upvalue. Errors are raised here, after Go has returned, so
 * Lua never longjmps through a Go frame. */
#include <stdlib.h>

#include "lua.h"
#include "lauxlib.h"
#include "_cgo_export.h"

static int tf_trampoline(lua_State *L)
{
	char *err = NULL;
	int n = tfCall(L, (int)lua_tointeger(L, lua_upvalueindex(1)), &err);

	if (err) {
		lua_pushstring(L, err);
		free(err);
		return lua_error(L);
	}
	return n;
}

void tf_push_func(lua_State *L, int idx)
{
	lua_pushinteger(L, idx);
	lua_pushcclosure(L, tf_trampoline, 1);
}

void tf_open_go_libs(lua_State *L)
{
	tfOpen(L);
}
