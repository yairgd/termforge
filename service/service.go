// Package service is a registry of named functions (name, help, params, handler) for a terminal app.
// One registry feeds every transport: the Lua console (lua.ServiceFuncs), REST and an MCP server (Handler).
package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
)

// Param types.
const (
	String  = "string"
	Integer = "integer"
	Number  = "number"
	Boolean = "boolean"
)

type Param struct {
	Name, Type, Help string
	Optional         bool
	Variadic         bool // last param only: takes the rest, value is []any
}

type Service struct {
	Name, Help string
	Params     []Param
	Run        func(ctx context.Context, a Args) (any, error)
}

// Args maps param name to value (JSON types: string, float64, bool, []any).
type Args map[string]any

func (a Args) String(n string) string { s, _ := a[n].(string); return s }
func (a Args) Bool(n string) bool     { b, _ := a[n].(bool); return b }

// Strings is a variadic param (or a JSON array) as []string.
func (a Args) Strings(n string) []string {
	var out []string
	switch v := a[n].(type) {
	case []any:
		for _, e := range v {
			out = append(out, fmt.Sprint(e))
		}
	case []string:
		out = v
	}
	return out
}

type Registry struct {
	mu   sync.Mutex
	svcs []Service
}

func (r *Registry) Add(s Service) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.svcs = append(r.svcs, s)
}

func (r *Registry) List() []Service {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]Service(nil), r.svcs...)
}

func (r *Registry) find(name string) (Service, bool) {
	for _, s := range r.List() {
		if s.Name == name {
			return s, true
		}
	}
	return Service{}, false
}

// Call checks the required params and their types, then runs the service.
func (r *Registry) Call(ctx context.Context, name string, a Args) (any, error) {
	s, ok := r.find(name)
	if !ok {
		return nil, fmt.Errorf("no such service: %q", name)
	}
	if a == nil {
		a = Args{}
	}
	for _, p := range s.Params {
		v, ok := a[p.Name]
		if !ok {
			if p.Optional || p.Variadic {
				continue
			}
			return nil, fmt.Errorf("%s: missing %q\n%s", name, p.Name, usage(s))
		}
		if !typeOK(p, v) {
			return nil, fmt.Errorf("%s: %q must be %s", name, p.Name, p.Type)
		}
	}
	return s.Run(ctx, a)
}

func typeOK(p Param, v any) bool {
	if p.Variadic {
		_, ok := v.([]any)
		return ok
	}
	switch p.Type {
	case String:
		_, ok := v.(string)
		return ok
	case Integer, Number:
		_, ok := v.(float64)
		return ok
	case Boolean:
		_, ok := v.(bool)
		return ok
	}
	return true
}

func usage(s Service) string {
	var ps []string
	for _, p := range s.Params {
		n := p.Name
		if p.Variadic {
			n += "..."
		}
		if p.Optional {
			n = "[" + n + "]"
		}
		ps = append(ps, n)
	}
	return s.Name + "(" + strings.Join(ps, ", ") + ")"
}

// Help is the overview with an empty topic, else the detail of one service.
func (r *Registry) Help(topic string) string {
	var b strings.Builder
	if topic != "" {
		s, ok := r.find(topic)
		if !ok {
			return "no such service: " + topic
		}
		fmt.Fprintf(&b, "%s\n  %s\n", usage(s), s.Help)
		for _, p := range s.Params {
			fmt.Fprintf(&b, "  %-12s %-8s %s\n", p.Name, p.Type, p.Help)
		}
		return b.String()
	}
	l := r.List()
	sort.SliceStable(l, func(i, j int) bool { return l[i].Name < l[j].Name })
	for _, s := range l {
		fmt.Fprintf(&b, "  %-32s %s\n", usage(s), s.Help)
	}
	return b.String()
}

// Tool is one service as an MCP tool.
type Tool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

func (r *Registry) ToolSchemas() []Tool {
	var out []Tool
	for _, s := range r.List() {
		props := map[string]any{}
		req := []string{}
		for _, p := range s.Params {
			pr := map[string]any{"description": p.Help}
			if p.Variadic {
				pr["type"] = "array"
				pr["items"] = map[string]any{"type": p.Type}
			} else {
				pr["type"] = p.Type
				if !p.Optional {
					req = append(req, p.Name)
				}
			}
			props[p.Name] = pr
		}
		out = append(out, Tool{s.Name, s.Help, map[string]any{"type": "object", "properties": props, "required": req}})
	}
	return out
}
