package wiki

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func testClient(t *testing.T) (*HTTPClient, *Cache) {
	t.Helper()
	dir := t.TempDir()
	cache := NewCache(dir, true)
	cfg := DefaultConfig()
	cfg.Delay = 0
	cfg.Retries = 3
	return NewHTTPClient(cfg, cache), cache
}

func TestGetJSONAndCache(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		_, _ = w.Write([]byte(`{"name":"turing","n":42}`))
	}))
	defer srv.Close()

	h, _ := testClient(t)
	var out struct {
		Name string `json:"name"`
		N    int    `json:"n"`
	}
	for range 3 {
		if err := h.GetJSON(context.Background(), srv.URL, time.Minute, &out); err != nil {
			t.Fatal(err)
		}
	}
	if out.Name != "turing" || out.N != 42 {
		t.Errorf("decoded wrong: %+v", out)
	}
	if got := atomic.LoadInt32(&hits); got != 1 {
		t.Errorf("expected 1 server hit (cache), got %d", got)
	}
}

func TestRetryOn503(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&hits, 1) < 3 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	h, _ := testClient(t)
	var out struct {
		OK bool `json:"ok"`
	}
	if err := h.GetJSON(context.Background(), srv.URL, 0, &out); err != nil {
		t.Fatal(err)
	}
	if !out.OK {
		t.Error("expected ok after retries")
	}
	if got := atomic.LoadInt32(&hits); got != 3 {
		t.Errorf("expected 3 attempts, got %d", got)
	}
}

func TestNotFoundError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`not here`))
	}))
	defer srv.Close()

	h, _ := testClient(t)
	_, err := h.GetBytes(context.Background(), srv.URL)
	if err == nil {
		t.Fatal("expected error")
	}
	if !NotFound(err) {
		t.Errorf("expected NotFound, got %v", err)
	}
}
