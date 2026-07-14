package bookmarks

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchTitleContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<!doctype html><html><head><title> Example &amp; Docs </title></head></html>`))
	}))
	defer server.Close()

	title, err := FetchTitleContext(context.Background(), server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if title != "Example & Docs" {
		t.Fatalf("title=%q", title)
	}
}

func TestDefaultTitleFallsBackToURL(t *testing.T) {
	const rawURL = "file:///tmp/example.html"
	if got := DefaultTitle(rawURL); got != rawURL {
		t.Fatalf("title=%q, want %q", got, rawURL)
	}
}

func TestExtractHTMLTitleNormalizesWhitespace(t *testing.T) {
	title := extractHTMLTitle("<TITLE>\n  A\t  B  \n</TITLE>")
	if title != "A B" {
		t.Fatalf("title=%q", title)
	}
}
