package service

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
)

// Handler serves REST (GET /api, POST /api/<name>) and MCP (POST /mcp, JSON-RPC 2.0,
// plain JSON replies). Bind it to loopback only: services can run commands.
func (r *Registry) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, r.ToolSchemas())
	})
	mux.HandleFunc("/api/", func(w http.ResponseWriter, q *http.Request) {
		if q.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}
		var a Args
		if err := json.NewDecoder(q.Body).Decode(&a); err != nil && err.Error() != "EOF" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		name := strings.TrimPrefix(q.URL.Path, "/api/")
		if !r.remote(name) {
			writeJSON(w, http.StatusNotFound, map[string]any{"error": "no such service: " + name})
			return
		}
		res, err := r.Call(q.Context(), name, a)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"result": res})
	})
	mux.HandleFunc("/mcp", r.mcp)
	return mux
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

type rpcReq struct {
	ID     json.RawMessage `json:"id"`
	Method string          `json:"method"`
	Params struct {
		Name      string `json:"name"`
		Arguments Args   `json:"arguments"`
	} `json:"params"`
}

func (r *Registry) mcp(w http.ResponseWriter, q *http.Request) {
	if q.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req rpcReq
	if err := json.NewDecoder(q.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.ID == nil { // notification
		w.WriteHeader(http.StatusAccepted)
		return
	}
	reply := map[string]any{"jsonrpc": "2.0", "id": req.ID}
	switch req.Method {
	case "initialize":
		reply["result"] = map[string]any{
			"protocolVersion": "2025-03-26",
			"capabilities":    map[string]any{"tools": map[string]any{}},
			"serverInfo":      map[string]any{"name": "termforge", "version": "1"},
		}
	case "ping":
		reply["result"] = map[string]any{}
	case "tools/list":
		reply["result"] = map[string]any{"tools": r.ToolSchemas()}
	case "tools/call":
		var res any
		var err error
		if r.remote(req.Params.Name) {
			res, err = r.Call(q.Context(), req.Params.Name, req.Params.Arguments)
		} else {
			err = fmt.Errorf("no such service: %q", req.Params.Name)
		}
		text, isErr := "", false
		if err != nil {
			text, isErr = err.Error(), true
		} else if s, ok := res.(string); ok {
			text = s
		} else if res != nil {
			b, _ := json.Marshal(res)
			text = string(b)
		}
		reply["result"] = map[string]any{
			"content": []map[string]any{{"type": "text", "text": text}},
			"isError": isErr,
		}
	default:
		reply["error"] = map[string]any{"code": -32601, "message": fmt.Sprintf("method not found: %s", req.Method)}
	}
	writeJSON(w, http.StatusOK, reply)
}

// WithToken rejects requests without "Authorization: Bearer <token>". An empty token
// rejects everything, so a forgotten token never leaves the endpoint open.
func WithToken(h http.Handler, token string) http.Handler {
	want := []byte("Bearer " + token)
	return http.HandlerFunc(func(w http.ResponseWriter, q *http.Request) {
		got := []byte(q.Header.Get("Authorization"))
		if token == "" || subtle.ConstantTimeCompare(got, want) != 1 {
			w.Header().Set("WWW-Authenticate", "Bearer")
			writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "unauthorized"})
			return
		}
		h.ServeHTTP(w, q)
	})
}

// ListenLoopback starts serving h on addr, which must be a loopback address (services can
// run commands). It returns once the listener is open.
func ListenLoopback(addr string, h http.Handler) error {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return err
	}
	if ip := net.ParseIP(host); host != "localhost" && (ip == nil || !ip.IsLoopback()) {
		return fmt.Errorf("%q is not a loopback address", host)
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	go http.Serve(ln, h)
	return nil
}

// remote reports whether name is a registered service that is served over REST/MCP.
func (r *Registry) remote(name string) bool {
	s, ok := r.find(name)
	return ok && s.Remote
}
