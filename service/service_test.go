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
	r.Add(Service{Name: "echo", Help: "join words", Params: []Param{
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
