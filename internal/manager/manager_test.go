package manager

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestServeFileDownloadSupportsRangeRequests(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "sample.txt")
	content := "abcdefghijklmnopqrstuvwxyz"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/s/demo/raw", nil)
	req.Header.Set("Range", "bytes=5-9")
	rec := httptest.NewRecorder()

	serveFileDownload(rec, req, path, "sample.txt")

	resp := rec.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusPartialContent {
		t.Fatalf("expected status %d, got %d", http.StatusPartialContent, resp.StatusCode)
	}
	if got := resp.Header.Get("Accept-Ranges"); got != "bytes" {
		t.Fatalf("expected Accept-Ranges bytes, got %q", got)
	}
	if got := resp.Header.Get("Content-Range"); got != "bytes 5-9/26" {
		t.Fatalf("expected Content-Range bytes 5-9/26, got %q", got)
	}

	body := rec.Body.String()
	if body != content[5:10] {
		t.Fatalf("expected body %q, got %q", content[5:10], body)
	}
}

func TestHandleShareAllowsHeadForRawDownloads(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "head.txt")
	if err := os.WriteFile(path, []byte("head-body"), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	mgr := &Manager{
		cfg:       DefaultConfig(),
		templates: mustParseTemplates(),
		shares: map[string]*Share{
			"share-1": {
				ID:   "share-1",
				Code: "abcd",
				Path: path,
				Name: "head.txt",
			},
		},
	}

	req := httptest.NewRequest(http.MethodHead, "/s/abcd/raw", nil)
	rec := httptest.NewRecorder()

	mgr.handleShare(rec, req)

	resp := rec.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}
	if got := resp.Header.Get("Accept-Ranges"); got != "bytes" {
		t.Fatalf("expected Accept-Ranges bytes, got %q", got)
	}
	if body := rec.Body.String(); strings.TrimSpace(body) != "" {
		t.Fatalf("expected empty body for HEAD, got %q", body)
	}
}

func mustParseTemplates() *template.Template {
	return template.Must(template.New("pages").Parse(homeHTML + manageHTML + shareHTML))
}
