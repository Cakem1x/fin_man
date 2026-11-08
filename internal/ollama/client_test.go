package ollama_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/Cakem1x/fin_man/internal/ollama"
)

func TestEmbed(t *testing.T) {
	var gotPath string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Errorf("decode request: %v", err)
		}
		_, _ = w.Write([]byte(`{"model":"m","embeddings":[[0.5,1],[0,-1]]}`))
	}))
	defer srv.Close()

	c := ollama.NewClient(srv.URL)
	vecs, err := c.Embed(context.Background(), "m", []string{"a", "b"})
	if err != nil {
		t.Fatalf("Embed: %v", err)
	}
	if gotPath != "/api/embed" {
		t.Errorf("path = %q, want /api/embed", gotPath)
	}
	wantBody := map[string]any{"model": "m", "input": []any{"a", "b"}}
	if !reflect.DeepEqual(gotBody, wantBody) {
		t.Errorf("body = %v, want %v", gotBody, wantBody)
	}
	want := [][]float32{{0.5, 1}, {0, -1}}
	if !reflect.DeepEqual(vecs, want) {
		t.Errorf("vecs = %v, want %v", vecs, want)
	}
}

func TestEmbedRejectsCountMismatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"embeddings":[[1]]}`))
	}))
	defer srv.Close()

	if _, err := ollama.NewClient(srv.URL).Embed(context.Background(), "m", []string{"a", "b"}); err == nil {
		t.Fatal("expected error for embedding count mismatch")
	}
}

func TestChatJSON(t *testing.T) {
	var gotPath string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Errorf("decode request: %v", err)
		}
		_, _ = w.Write([]byte(`{"message":{"role":"assistant","content":"{\"category\":\"Food\"}"},"done":true}`))
	}))
	defer srv.Close()

	schema := map[string]any{"type": "object"}
	out, err := ollama.NewClient(srv.URL).ChatJSON(context.Background(), "m",
		[]ollama.Message{{Role: "user", Content: "hi"}}, schema)
	if err != nil {
		t.Fatalf("ChatJSON: %v", err)
	}
	if gotPath != "/api/chat" {
		t.Errorf("path = %q, want /api/chat", gotPath)
	}
	wantBody := map[string]any{
		"model":    "m",
		"messages": []any{map[string]any{"role": "user", "content": "hi"}},
		"stream":   false,
		"think":    false,
		"format":   map[string]any{"type": "object"},
		"options":  map[string]any{"temperature": float64(0)},
	}
	if !reflect.DeepEqual(gotBody, wantBody) {
		t.Errorf("body = %v, want %v", gotBody, wantBody)
	}
	if string(out) != `{"category":"Food"}` {
		t.Errorf("content = %s", out)
	}
}

func TestErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"model not found"}`, http.StatusNotFound)
	}))
	defer srv.Close()

	_, err := ollama.NewClient(srv.URL).Embed(context.Background(), "m", []string{"a"})
	if err == nil {
		t.Fatal("expected error for HTTP 404")
	}
}

func TestContextTimeout(t *testing.T) {
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-release
	}))
	defer srv.Close()
	defer close(release)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if _, err := ollama.NewClient(srv.URL).Embed(ctx, "m", []string{"a"}); err == nil {
		t.Fatal("expected error on context timeout")
	}
}
