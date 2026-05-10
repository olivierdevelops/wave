package http

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGzipCompressesLargeJSON(t *testing.T) {
	body := strings.Repeat(`{"k":"vvvvvvvvvv"},`, 200)
	h := GzipMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Accept-Encoding", "gzip, deflate")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Header().Get("Content-Encoding") != "gzip" {
		t.Fatalf("encoding = %q", w.Header().Get("Content-Encoding"))
	}
	gr, err := gzip.NewReader(w.Body)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := io.ReadAll(gr)
	if string(got) != body {
		t.Errorf("decompressed mismatch")
	}
}

func TestGzipSkipsTinyBody(t *testing.T) {
	h := GzipMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"x":1}`))
	}))
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Header().Get("Content-Encoding") == "gzip" {
		t.Errorf("tiny body should not be compressed")
	}
	if w.Body.String() != `{"x":1}` {
		t.Errorf("body altered: %q", w.Body.String())
	}
}

func TestGzipSkipsWithoutAcceptEncoding(t *testing.T) {
	body := strings.Repeat("x", 1024)
	h := GzipMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte(body))
	}))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	if w.Header().Get("Content-Encoding") == "gzip" {
		t.Error("compressed without client opt-in")
	}
	if w.Body.String() != body {
		t.Error("body altered")
	}
}

func TestGzipSkipsSSE(t *testing.T) {
	h := GzipMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(strings.Repeat("data: x\n\n", 200)))
	}))
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Header().Get("Content-Encoding") == "gzip" {
		t.Error("SSE should not be gzipped")
	}
}

func TestGzipFlushOptsOut(t *testing.T) {
	h := GzipMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.(http.Flusher).Flush()
		_, _ = w.Write([]byte(strings.Repeat("y", 2000)))
	}))
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Header().Get("Content-Encoding") == "gzip" {
		t.Error("Flush before write should opt out of gzip")
	}
}

// Regression: a streaming handler (SSE-like) that writes, flushes, then
// writes again must not trigger spurious WriteHeader calls. Before the
// fix, gzipWriter.Write called commitPassthrough on every write after
// the first Flush, producing "superfluous response.WriteHeader call"
// warnings every heartbeat tick (15s in production).
func TestGzipStreamingNoSpuriousWriteHeader(t *testing.T) {
	rec := &writeHeaderCounter{ResponseWriter: httptest.NewRecorder()}

	h := GzipMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		// Initial connect frame + flush (mirrors infra/connections/sse.go)
		_, _ = w.Write([]byte(": connected\n\n"))
		w.(http.Flusher).Flush()
		// Several heartbeats — each one writes then flushes.
		for i := 0; i < 5; i++ {
			_, _ = w.Write([]byte(": ping\n\n"))
			w.(http.Flusher).Flush()
		}
	}))
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Accept-Encoding", "gzip")
	h.ServeHTTP(rec, r)

	if rec.headerCalls != 1 {
		t.Errorf("want exactly 1 WriteHeader call, got %d", rec.headerCalls)
	}
}

// writeHeaderCounter wraps a ResponseWriter and counts WriteHeader calls.
type writeHeaderCounter struct {
	http.ResponseWriter
	headerCalls int
}

func (w *writeHeaderCounter) WriteHeader(code int) {
	w.headerCalls++
	w.ResponseWriter.WriteHeader(code)
}

func (w *writeHeaderCounter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}
