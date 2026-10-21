//go:build lua

package lua

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/yairgd/termforge/service"
)

// ServiceFuncs wraps every service of r as a Func, for Register: arguments are matched to
// the service's params in order (a variadic param takes the rest) and the result is
// returned to Lua as a table, string or number. A Go error is raised in Lua.
func ServiceFuncs(r *service.Registry) map[string]Func {
	fs := map[string]Func{}
	for _, sv := range r.List() {
		sv := sv
		fs[sv.Name] = func(c *Call) (int, error) {
			in := service.Args{}
			for n, p := range sv.Params {
				if p.Variadic {
					rest := []any{}
					for i := n + 1; i <= c.Top(); i++ {
						rest = append(rest, c.value(i))
					}
					in[p.Name] = rest
				} else if n+1 <= c.Top() {
					in[p.Name] = c.value(n + 1)
				}
			}
			res, err := r.Call(context.Background(), sv.Name, in)
			if err != nil {
				return 0, err
			}
			if res == nil {
				return 0, nil
			}
			if err := c.Push(res); err != nil {
				return 0, err
			}
			return 1, nil
		}
	}
	return fs
}

// value is argument i as a JSON-style value (string, float64, bool); other types are nil.
func (c *Call) value(i int) any {
	switch c.kind(i) {
	case kString:
		s, _ := c.String(i)
		return s
	case kNumber:
		f, _ := c.Number(i)
		return f
	case kBool:
		return c.truthy(i)
	}
	return nil
}

// Push pushes any Go value that encodes to JSON: maps and structs become tables, slices
// arrays. A table printed in the REPL shows one readable line per element, not its address;
// scripts can still index it.
func (c *Call) Push(v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	var j any
	if err := json.Unmarshal(b, &j); err != nil {
		return err
	}
	c.pushJSON(j)
	switch j.(type) {
	case []any, map[string]any:
		c.setText(render(j))
	}
	return nil
}

func (c *Call) pushJSON(j any) {
	switch v := j.(type) {
	case nil:
		c.PushNil()
	case bool:
		c.PushBool(v)
	case float64:
		if v == float64(int(v)) {
			c.PushInt(int(v))
		} else {
			c.PushNumber(v)
		}
	case string:
		c.PushString(v)
	case []any:
		c.newTable(len(v), 0)
		for i, e := range v {
			c.pushJSON(e)
			c.setIndex(i + 1)
		}
	case map[string]any:
		c.newTable(0, len(v))
		for k, e := range v {
			c.pushJSON(e)
			c.setField(k)
		}
	}
}

// render is the REPL text of a table: one line per array element, "k=v k=v" per map.
func render(j any) string {
	line := func(e any) string {
		m, ok := e.(map[string]any)
		if !ok {
			return fmt.Sprint(e)
		}
		keys := make([]string, 0, len(m))
		for k := range m {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		parts := make([]string, len(keys))
		for i, k := range keys {
			parts[i] = fmt.Sprintf("%s=%v", k, m[k])
		}
		return strings.Join(parts, " ")
	}
	if a, ok := j.([]any); ok {
		lines := make([]string, len(a))
		for i, e := range a {
			lines[i] = line(e)
		}
		return strings.Join(lines, "\n")
	}
	return line(j)
}
