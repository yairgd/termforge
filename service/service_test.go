package service

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

func testReg() *Registry {
	r := &Registry{}
	r.Add(Service{Name: "echo", Help: "join words", Remote: true, Params: []Param{
		{Name: "prefix", Type: String, Help: "p"},
		{Name: "words", Type: String, Help: "w", Variadic: true},
	}, Run: func(_ context.Context, a Args) (any, error) {
		return a.String("prefix") + strings.Join(a.Strings("words"), " "), nil
	}})
	return r
}

func TestCall(t *testing.T) {
	r := testReg()
	if _, err := r.Call(context.Background(), "echo", Args{}); err == nil {
		t.Fatal("missing param accepted")
	}
	if _, err := r.Call(context.Background(), "echo", Args{"prefix": 1.0}); err == nil {
		t.Fatal("wrong type accepted")
	}
	got, err := r.Call(context.Background(), "echo", Args{"prefix": ">", "words": []any{"a", "b"}})
	if err != nil || got != ">a b" {
		t.Fatalf("got %v, %v", got, err)
	}
}

func post(t *testing.T, h *httptest.Server, path, body string) map[string]any {
	t.Helper()
	rec := httptest.NewRecorder()
	h.Config.Handler.ServeHTTP(rec, httptest.NewRequest("POST", path, strings.NewReader(body)))
	var out map[string]any
	json.Unmarshal(rec.Body.Bytes(), &out)
	return out
}

func TestREST(t *testing.T) {
	srv := httptest.NewServer(testReg().Handler())
	defer srv.Close()
	if out := post(t, srv, "/api/echo", `{"prefix":">","words":["x"]}`); out["result"] != ">x" {
		t.Fatalf("%v", out)
	}
	if out := post(t, srv, "/api/nope", `{}`); out["error"] == nil {
		t.Fatalf("%v", out)
	}
}

func TestMCP(t *testing.T) {
	srv := httptest.NewServer(testReg().Handler())
	defer srv.Close()
	out := post(t, srv, "/mcp", `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)
	tools := out["result"].(map[string]any)["tools"].([]any)
	if len(tools) != 1 || tools[0].(map[string]any)["name"] != "echo" {
		t.Fatalf("%v", out)
	}
	out = post(t, srv, "/mcp", `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"echo","arguments":{"prefix":">","words":["q"]}}}`)
	c := out["result"].(map[string]any)["content"].([]any)[0].(map[string]any)
	if c["text"] != ">q" {
		t.Fatalf("%v", out)
	}
}

func TestWithToken(t *testing.T) {
	h := WithToken(testReg().Handler(), "s3")
	for token, want := range map[string]int{"": 401, "Bearer x": 401, "Bearer s3": 200} {
		rec := httptest.NewRecorder()
		q := httptest.NewRequest("GET", "/api", nil)
		if token != "" {
			q.Header.Set("Authorization", token)
		}
		h.ServeHTTP(rec, q)
		if rec.Code != want {
			t.Fatalf("%q: %d", token, rec.Code)
		}
	}
	rec := httptest.NewRecorder()
	WithToken(testReg().Handler(), "").ServeHTTP(rec, httptest.NewRequest("GET", "/api", nil))
	if rec.Code != 401 {
		t.Fatal("empty token must reject")
	}
}

func TestListenLoopbackRefusesOthers(t *testing.T) {
	if err := ListenLoopback("0.0.0.0:0", nil); err == nil {
		t.Fatal("non-loopback accepted")
	}
}

func TestLocalServiceNotRemote(t *testing.T) {
	r := testReg()
	r.Add(Service{Name: "local", Help: "lua only", Run: func(context.Context, Args) (any, error) { return "x", nil }})
	if got, err := r.Call(context.Background(), "local", nil); err != nil || got != "x" {
		t.Fatalf("local call: %v %v", got, err)
	}
	if n := len(r.ToolSchemas()); n != 1 {
		t.Fatalf("schemas=%d", n)
	}
	srv := httptest.NewServer(r.Handler())
	defer srv.Close()
	if out := post(t, srv, "/api/local", `{}`); out["error"] == nil {
		t.Fatalf("REST reached a local service: %v", out)
	}
	out := post(t, srv, "/mcp", `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"local","arguments":{}}}`)
	if out["result"].(map[string]any)["isError"] != true {
		t.Fatalf("MCP reached a local service: %v", out)
	}
}
