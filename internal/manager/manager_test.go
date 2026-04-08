package manager

import (
	"bytes"
	"encoding/json"
	"html/template"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
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

func TestSequentialChunkUploadCompletesFile(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := &Manager{
		cfg:       DefaultConfig(),
		templates: mustParseTemplates(),
		shares: map[string]*Share{
			"share-1": {
				ID:       "share-1",
				Code:     "abcd",
				Path:     tmpDir,
				Name:     "tmp",
				IsDir:    true,
				Password: "123456",
			},
		},
		uploads: make(map[string]*uploadSession),
	}

	startBody, err := json.Marshal(uploadStartRequest{
		Path:        "",
		Password:    "123456",
		FileName:    "movie.bin",
		FileSize:    10,
		ChunkSize:   4,
		TotalChunks: 3,
	})
	if err != nil {
		t.Fatalf("marshal start request: %v", err)
	}

	startReq := httptest.NewRequest(http.MethodPost, "/s/abcd/upload/start", bytes.NewReader(startBody))
	startRec := httptest.NewRecorder()
	mgr.handleShare(startRec, startReq)

	if startRec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, startRec.Code, startRec.Body.String())
	}

	var startResp uploadStartResponse
	if err := json.Unmarshal(startRec.Body.Bytes(), &startResp); err != nil {
		t.Fatalf("decode start response: %v", err)
	}
	if startResp.UploadID == "" {
		t.Fatal("expected upload id")
	}

	for index, chunk := range []string{"abcd", "efgh", "ij"} {
		chunkReq := httptest.NewRequest(http.MethodPost, "/s/abcd/upload/chunk?upload_id="+startResp.UploadID+"&index="+strconv.Itoa(index), strings.NewReader(chunk))
		chunkReq.ContentLength = int64(len(chunk))
		chunkRec := httptest.NewRecorder()
		mgr.handleShare(chunkRec, chunkReq)
		if chunkRec.Code != http.StatusOK {
			t.Fatalf("expected status %d for chunk %d, got %d: %s", http.StatusOK, index, chunkRec.Code, chunkRec.Body.String())
		}
	}

	content, err := os.ReadFile(filepath.Join(tmpDir, "movie.bin"))
	if err != nil {
		t.Fatalf("read uploaded file: %v", err)
	}
	if string(content) != "abcdefghij" {
		t.Fatalf("expected uploaded content %q, got %q", "abcdefghij", string(content))
	}

	if _, err := os.Stat(filepath.Join(tmpDir, "movie.bin.webshare.part")); !os.IsNotExist(err) {
		t.Fatalf("expected temp file to be removed, got %v", err)
	}
}

func TestChunkUploadRejectsUnexpectedIndex(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := &Manager{
		cfg:       DefaultConfig(),
		templates: mustParseTemplates(),
		shares: map[string]*Share{
			"share-1": {
				ID:       "share-1",
				Code:     "abcd",
				Path:     tmpDir,
				Name:     "tmp",
				IsDir:    true,
				Password: "123456",
			},
		},
		uploads: make(map[string]*uploadSession),
	}

	startBody, _ := json.Marshal(uploadStartRequest{
		Password:    "123456",
		FileName:    "wrong.bin",
		FileSize:    8,
		ChunkSize:   4,
		TotalChunks: 2,
	})
	startReq := httptest.NewRequest(http.MethodPost, "/s/abcd/upload/start", bytes.NewReader(startBody))
	startRec := httptest.NewRecorder()
	mgr.handleShare(startRec, startReq)

	var startResp uploadStartResponse
	if err := json.Unmarshal(startRec.Body.Bytes(), &startResp); err != nil {
		t.Fatalf("decode start response: %v", err)
	}

	chunkReq := httptest.NewRequest(http.MethodPost, "/s/abcd/upload/chunk?upload_id="+startResp.UploadID+"&index=1", strings.NewReader("efgh"))
	chunkReq.ContentLength = 4
	chunkRec := httptest.NewRecorder()
	mgr.handleShare(chunkRec, chunkReq)

	if chunkRec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, chunkRec.Code)
	}
	if !strings.Contains(chunkRec.Body.String(), "unexpected chunk index") {
		t.Fatalf("expected unexpected chunk index error, got %q", chunkRec.Body.String())
	}
}

func mustParseTemplates() *template.Template {
	return template.Must(template.New("pages").Parse(homeHTML + manageHTML + shareHTML))
}
