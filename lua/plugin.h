#include "lua.h"
void tf_push_func(lua_State *L, int idx);
int tf_run(const char *path);
void tf_interrupt(void);
