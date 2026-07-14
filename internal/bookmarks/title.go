package bookmarks

import (
	"context"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const titleFetchLimit = 512 * 1024

func DefaultTitle(rawURL string) string {
	title, err := FetchTitle(rawURL)
	if err != nil || strings.TrimSpace(title) == "" {
		return strings.TrimSpace(rawURL)
	}
	return title
}

func FetchTitle(rawURL string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return FetchTitleContext(ctx, rawURL)
}

func FetchTitleContext(ctx context.Context, rawURL string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return "", err
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("unsupported URL scheme: %s", parsed.Scheme)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "brmk/0.1")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("unexpected HTTP status: %s", resp.Status)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, titleFetchLimit))
	if err != nil {
		return "", err
	}
	title := extractHTMLTitle(string(data))
	if title == "" {
		return "", fmt.Errorf("title not found")
	}
	return title, nil
}

func extractHTMLTitle(document string) string {
	lower := strings.ToLower(document)
	start := strings.Index(lower, "<title")
	if start < 0 {
		return ""
	}
	afterOpen := strings.Index(lower[start:], ">")
	if afterOpen < 0 {
		return ""
	}
	contentStart := start + afterOpen + 1
	contentEnd := strings.Index(lower[contentStart:], "</title>")
	if contentEnd < 0 {
		return ""
	}
	title := document[contentStart : contentStart+contentEnd]
	title = html.UnescapeString(title)
	title = strings.Join(strings.Fields(title), " ")
	return strings.TrimSpace(title)
}
