//go:build lua

/* Lua 5.4 as one translation unit (the same shape as upstream onelua.c), plus the
 * terminal glue for the embedded console. The vendored sources under lua-5.4.x/ are
 * not edited: every hook the interpreter offers is a macro that lua.c / lauxlib.h
 * only define when the host has not, and they are defined here.
 *
 * To use another Lua version: drop it next to lua-5.4.6/, change the include paths
 * below and the -I in lua.go. */
#define LUA_CORE
#define LUA_LIB
#define MAKE_LUA
#include "lua-5.4.6/src/lprefix.h"
#include <signal.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

/* Console I/O: print, errors and the banner go to the PTY slave. */
static FILE *tf_in, *tf_out;
static char *tf_path;

#define lua_writestring(s, l)      fwrite((s), sizeof(char), (l), tf_out)
#define lua_writeline()            (lua_writestring("\n", 1), fflush(tf_out))
#define lua_writestringerror(s, p) (fprintf(tf_out, (s), (p)), fflush(tf_out))

#include "lua.h"
#include "lauxlib.h"
#include "lualib.h"

static lua_State *tf_state;

/* core */
#include "lua-5.4.6/src/lapi.c"
#include "lua-5.4.6/src/lcode.c"
#include "lua-5.4.6/src/lctype.c"
#include "lua-5.4.6/src/ldebug.c"
#include "lua-5.4.6/src/ldo.c"
#include "lua-5.4.6/src/ldump.c"
#include "lua-5.4.6/src/lfunc.c"
#include "lua-5.4.6/src/lgc.c"
#include "lua-5.4.6/src/llex.c"
#include "lua-5.4.6/src/lmem.c"
#include "lua-5.4.6/src/lobject.c"
#include "lua-5.4.6/src/lopcodes.c"
#include "lua-5.4.6/src/lparser.c"
#include "lua-5.4.6/src/lstate.c"
#include "lua-5.4.6/src/lstring.c"
#include "lua-5.4.6/src/ltable.c"
#include "lua-5.4.6/src/ltm.c"
#include "lua-5.4.6/src/lundump.c"
#include "lua-5.4.6/src/lvm.c"
#include "lua-5.4.6/src/lzio.c"
/* alloc hooks used by lauxlib.c */
#include "lua-5.4.6/memory.c"
/* auxiliary and standard libraries */
#include "lua-5.4.6/src/lauxlib.c"
#include "lua-5.4.6/src/lbaselib.c"
#include "lua-5.4.6/src/lcorolib.c"
#include "lua-5.4.6/src/ldblib.c"
#include "lua-5.4.6/src/liolib.c"
#include "lua-5.4.6/src/lmathlib.c"
#include "lua-5.4.6/src/loadlib.c"
#include "lua-5.4.6/src/loslib.c"
#include "lua-5.4.6/src/lstrlib.c"
#include "lua-5.4.6/src/ltablib.c"
#include "lua-5.4.6/src/lutf8lib.c"
#include "lua-5.4.6/src/linit.c"

/* The Go plugin libraries, see plugin.c. */
void tf_open_go_libs(lua_State *L);

/* luaL_openlibs is declared "(luaL_openlibs)" in lualib.h, so this only rewrites the
 * call in pmain: standard libs, the Go libs, then io on the console. */
static void tf_openlibs(lua_State *L)
{
	luaL_openlibs(L);
	tf_open_go_libs(L);
	if (tf_path) {
		lua_getglobal(L, "io");
		lua_getfield(L, -1, "output");
		lua_pushstring(L, tf_path);
		lua_pcall(L, 1, 0, 0);
		lua_getfield(L, -1, "input");
		lua_pushstring(L, tf_path);
		lua_pcall(L, 1, 0, 0);
		lua_pop(L, 1);
	}
	tf_state = L;
}
#define luaL_openlibs(L) tf_openlibs(L)

/* Line editing and history: vendored linenoise (no system library), run on the console
 * fds instead of stdin/stdout (see linenoise/README for the small patch). */
#include <strings.h> /* strcasecmp, hidden by _XOPEN_SOURCE */
#include "linenoise/linenoise.c"

/* Ctrl-C at the prompt drops the line (and its history slot) instead of ending the REPL. */
static char *tf_readline(const char *prompt)
{
	char *res;

	fflush(tf_out);
	for (;;) {
		errno = 0;
		res = linenoiseBlockingEdit(fileno(tf_in), fileno(tf_out), prompt);
		if (res || errno != EAGAIN)
			return res;
		free(history[--history_len]);
	}
}

#define lua_initreadline(L)    ((void)L)
#define lua_readline(L, b, p)  ((void)L, ((b) = tf_readline(p)) != NULL)
#define lua_saveline(L, line)  ((void)L, linenoiseHistoryAdd(line))
#define lua_freeline(L, b)     ((void)L, free(b))

/* lua.c installs a SIGINT handler; replacing Go's would be unsafe. Ctrl-C reaches
 * Lua through tf_interrupt instead. */
#define signal(s, h) SIG_DFL
#define main lua_main
#include "lua-5.4.6/src/lua.c"
#undef main
#undef signal

/* Runs the REPL on the PTY slave named by path until EOF. The console files stay open
 * across runs (readline keeps pointers to them), so every run must use the same path. */
int tf_run(const char *path)
{
	char *argv[] = { "lua", "-v", "-i", NULL };
	int rc;

	if (!tf_out) {
		tf_path = strdup(path);
		tf_in = fopen(path, "r");
		tf_out = fopen(path, "w");
		if (!tf_in || !tf_out)
			return -1;
	}
	rc = lua_main(3, argv);
	tf_state = NULL;
	return rc;
}

/* Stops the running chunk ("interrupted!"); the REPL goes on. Callable from any thread. */
void tf_interrupt(void)
{
	lua_State *L = tf_state;

	if (L)
		lua_sethook(L, lstop, LUA_MASKCALL | LUA_MASKRET | LUA_MASKLINE | LUA_MASKCOUNT, 1);
}
