package service

import (
	"encoding/json"
	"fmt"
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
		res, err := r.Call(q.Context(), strings.TrimPrefix(q.URL.Path, "/api/"), a)
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
		res, err := r.Call(q.Context(), req.Params.Name, req.Params.Arguments)
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
