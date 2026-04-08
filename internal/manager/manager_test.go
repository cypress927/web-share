package manager

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"html/template"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
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
		FilePath:    "movie.bin",
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
		FilePath:    "wrong.bin",
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

func TestSequentialChunkUploadCreatesNestedFolders(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, "photos"), 0o755); err != nil {
		t.Fatalf("mkdir current upload dir: %v", err)
	}
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
		Path:        "photos",
		Password:    "123456",
		FilePath:    "trip/day1/pic.txt",
		FileSize:    5,
		ChunkSize:   5,
		TotalChunks: 1,
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

	chunkReq := httptest.NewRequest(http.MethodPost, "/s/abcd/upload/chunk?upload_id="+startResp.UploadID+"&index=0", strings.NewReader("hello"))
	chunkReq.ContentLength = 5
	chunkRec := httptest.NewRecorder()
	mgr.handleShare(chunkRec, chunkReq)
	if chunkRec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, chunkRec.Code, chunkRec.Body.String())
	}

	content, err := os.ReadFile(filepath.Join(tmpDir, "photos", "trip", "day1", "pic.txt"))
	if err != nil {
		t.Fatalf("read nested upload: %v", err)
	}
	if string(content) != "hello" {
		t.Fatalf("expected nested uploaded content %q, got %q", "hello", string(content))
	}
}

func TestServeShareArchiveDownloadsRootAsZip(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, "docs", "nested"), 0o755); err != nil {
		t.Fatalf("mkdir tree: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "readme.txt"), []byte("root"), 0o644); err != nil {
		t.Fatalf("write root file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "docs", "nested", "note.txt"), []byte("nested"), 0o644); err != nil {
		t.Fatalf("write nested file: %v", err)
	}

	mgr := &Manager{
		cfg:       DefaultConfig(),
		templates: mustParseTemplates(),
		shares: map[string]*Share{
			"share-1": {
				ID:    "share-1",
				Code:  "abcd",
				Path:  tmpDir,
				Name:  "My Share",
				IsDir: true,
			},
		},
		uploads: make(map[string]*uploadSession),
	}

	req := httptest.NewRequest(http.MethodGet, "/s/abcd/archive", nil)
	rec := httptest.NewRecorder()
	mgr.handleShare(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != "application/zip" {
		t.Fatalf("expected application/zip, got %q", got)
	}

	readerAt := bytes.NewReader(rec.Body.Bytes())
	zr, err := zip.NewReader(readerAt, int64(readerAt.Len()))
	if err != nil {
		t.Fatalf("open zip: %v", err)
	}

	entries := map[string]string{}
	for _, file := range zr.File {
		if file.FileInfo().IsDir() {
			entries[file.Name] = ""
			continue
		}
		rc, err := file.Open()
		if err != nil {
			t.Fatalf("open zip entry %s: %v", file.Name, err)
		}
		body, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatalf("read zip entry %s: %v", file.Name, err)
		}
		entries[file.Name] = string(body)
	}

	if got := entries["My Share/readme.txt"]; got != "root" {
		t.Fatalf("expected root file in archive, got %q", got)
	}
	if got := entries["My Share/docs/nested/note.txt"]; got != "nested" {
		t.Fatalf("expected nested file in archive, got %q", got)
	}
}

func TestServeShareArchiveDownloadsSubfolderAsZip(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, "docs", "nested"), 0o755); err != nil {
		t.Fatalf("mkdir tree: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "readme.txt"), []byte("root"), 0o644); err != nil {
		t.Fatalf("write root file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "docs", "nested", "note.txt"), []byte("nested"), 0o644); err != nil {
		t.Fatalf("write nested file: %v", err)
	}

	mgr := &Manager{
		cfg:       DefaultConfig(),
		templates: mustParseTemplates(),
		shares: map[string]*Share{
			"share-1": {
				ID:    "share-1",
				Code:  "abcd",
				Path:  tmpDir,
				Name:  "My Share",
				IsDir: true,
			},
		},
		uploads: make(map[string]*uploadSession),
	}

	req := httptest.NewRequest(http.MethodGet, "/s/abcd/archive?path=docs", nil)
	rec := httptest.NewRecorder()
	mgr.handleShare(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Disposition"); !strings.Contains(got, `filename="docs.zip"`) {
		t.Fatalf("expected docs.zip content disposition, got %q", got)
	}

	readerAt := bytes.NewReader(rec.Body.Bytes())
	zr, err := zip.NewReader(readerAt, int64(readerAt.Len()))
	if err != nil {
		t.Fatalf("open zip: %v", err)
	}

	entries := map[string]string{}
	for _, file := range zr.File {
		if file.FileInfo().IsDir() {
			entries[file.Name] = ""
			continue
		}
		rc, err := file.Open()
		if err != nil {
			t.Fatalf("open zip entry %s: %v", file.Name, err)
		}
		body, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatalf("read zip entry %s: %v", file.Name, err)
		}
		entries[file.Name] = string(body)
	}

	if _, ok := entries["docs/readme.txt"]; ok {
		t.Fatal("root file should not be included in subfolder archive")
	}
	if got := entries["docs/nested/note.txt"]; got != "nested" {
		t.Fatalf("expected nested file in docs archive, got %q", got)
	}
}

func TestRenderSharePageShowsUnavailableWhenRootMissing(t *testing.T) {
	tmpDir := t.TempDir()
	missingPath := filepath.Join(tmpDir, "missing.txt")

	mgr := &Manager{
		cfg:       DefaultConfig(),
		templates: mustParseTemplates(),
		shares: map[string]*Share{
			"share-1": {
				ID:   "share-1",
				Code: "abcd",
				Path: missingPath,
				Name: "missing.txt",
			},
		},
		uploads: make(map[string]*uploadSession),
	}

	req := httptest.NewRequest(http.MethodGet, "/s/abcd?lang=zh-CN", nil)
	rec := httptest.NewRecorder()
	mgr.handleShare(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "该分享对应的文件或文件夹已不存在") {
		t.Fatalf("expected unavailable message, got %q", rec.Body.String())
	}
}

func TestServeShareRawRedirectsWhenFileMissing(t *testing.T) {
	tmpDir := t.TempDir()
	missingPath := filepath.Join(tmpDir, "missing.txt")

	mgr := &Manager{
		cfg:       DefaultConfig(),
		templates: mustParseTemplates(),
		shares: map[string]*Share{
			"share-1": {
				ID:   "share-1",
				Code: "abcd",
				Path: missingPath,
				Name: "missing.txt",
			},
		},
		uploads: make(map[string]*uploadSession),
	}

	req := httptest.NewRequest(http.MethodGet, "/s/abcd/raw", nil)
	rec := httptest.NewRecorder()
	mgr.handleShare(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected status %d, got %d", http.StatusSeeOther, rec.Code)
	}
	location := rec.Header().Get("Location")
	if !strings.Contains(location, "/s/abcd?error=") || !strings.Contains(location, url.QueryEscape("The file no longer exists or has been moved.")) {
		t.Fatalf("expected redirect to share page with error, got %q", location)
	}
}

func TestHomeShowsUnavailableVisibleShare(t *testing.T) {
	tmpDir := t.TempDir()
	missingPath := filepath.Join(tmpDir, "missing.txt")

	mgr := &Manager{
		cfg:       DefaultConfig(),
		templates: mustParseTemplates(),
		shares: map[string]*Share{
			"share-1": {
				ID:      "share-1",
				Code:    "abcd",
				Path:    missingPath,
				Name:    "missing.txt",
				Visible: true,
			},
		},
		uploads: make(map[string]*uploadSession),
	}

	req := httptest.NewRequest(http.MethodGet, "/?lang=zh-CN", nil)
	rec := httptest.NewRecorder()
	mgr.handleHome(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "已失效") {
		t.Fatalf("expected unavailable marker, got %q", body)
	}
}

func TestRenderSharePageRedirectsWhenSubfolderMissing(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		t.Fatalf("mkdir root: %v", err)
	}

	mgr := &Manager{
		cfg:       DefaultConfig(),
		templates: mustParseTemplates(),
		shares: map[string]*Share{
			"share-1": {
				ID:    "share-1",
				Code:  "abcd",
				Path:  tmpDir,
				Name:  "folder",
				IsDir: true,
			},
		},
		uploads: make(map[string]*uploadSession),
	}

	req := httptest.NewRequest(http.MethodGet, "/s/abcd?lang=zh-CN&path=gone/sub", nil)
	rec := httptest.NewRecorder()
	mgr.handleShare(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected status %d, got %d", http.StatusSeeOther, rec.Code)
	}
	location := rec.Header().Get("Location")
	if !strings.Contains(location, "/s/abcd?path=gone&error=") {
		t.Fatalf("expected redirect back to parent with error, got %q", location)
	}
}

func TestUploadStartRejectsWhenCurrentDirectoryMissing(t *testing.T) {
	tmpDir := t.TempDir()

	mgr := &Manager{
		cfg:       DefaultConfig(),
		templates: mustParseTemplates(),
		shares: map[string]*Share{
			"share-1": {
				ID:       "share-1",
				Code:     "abcd",
				Path:     tmpDir,
				Name:     "folder",
				IsDir:    true,
				Password: "123456",
			},
		},
		uploads: make(map[string]*uploadSession),
	}

	startBody, err := json.Marshal(uploadStartRequest{
		Path:        "gone",
		Password:    "123456",
		FilePath:    "file.txt",
		FileSize:    4,
		ChunkSize:   4,
		TotalChunks: 1,
	})
	if err != nil {
		t.Fatalf("marshal start request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/s/abcd/upload/start", bytes.NewReader(startBody))
	rec := httptest.NewRecorder()
	mgr.handleShare(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "current directory no longer exists") {
		t.Fatalf("expected missing directory message, got %q", rec.Body.String())
	}
}

func TestCreateClipboardTextShare(t *testing.T) {
	mgr := &Manager{
		cfg:       DefaultConfig(),
		templates: mustParseTemplates(),
		shares:    make(map[string]*Share),
		uploads:   make(map[string]*uploadSession),
	}

	share, err := mgr.createShare(CreateShareRequest{
		Kind:        shareKindClipboardText,
		Name:        "剪贴板文本",
		TextContent: "hello clipboard",
	})
	if err != nil {
		t.Fatalf("create clipboard text share: %v", err)
	}
	if share.Kind != shareKindClipboardText {
		t.Fatalf("expected kind %q, got %q", shareKindClipboardText, share.Kind)
	}
	if share.TextContent != "hello clipboard" {
		t.Fatalf("expected text content, got %q", share.TextContent)
	}
}

func TestRenderFileSharePageDoesNotRequireDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "sample.txt")
	if err := os.WriteFile(filePath, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	mgr := &Manager{
		cfg:       DefaultConfig(),
		templates: mustParseTemplates(),
		shares: map[string]*Share{
			"share-1": {
				ID:    "share-1",
				Code:  "abcd",
				Kind:  shareKindFile,
				Path:  filePath,
				Name:  "sample.txt",
				IsDir: false,
			},
		},
		uploads: make(map[string]*uploadSession),
	}

	req := httptest.NewRequest(http.MethodGet, "/s/abcd", nil)
	rec := httptest.NewRecorder()
	mgr.handleShare(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}
}

func TestListVisibleSharesIncludesClipboardTextPreview(t *testing.T) {
	mgr := &Manager{
		cfg:       DefaultConfig(),
		templates: mustParseTemplates(),
		shares: map[string]*Share{
			"share-1": {
				ID:          "share-1",
				Code:        "abcd",
				Kind:        shareKindClipboardText,
				Name:        "text",
				Visible:     true,
				TextContent: "第一行\n第二行",
			},
		},
		uploads: make(map[string]*uploadSession),
	}

	cards := mgr.listVisibleShares(langZH)
	if len(cards) != 1 {
		t.Fatalf("expected 1 card, got %d", len(cards))
	}
	card := cards[0]
	if !card.ShowCopy {
		t.Fatal("expected clipboard text card to show copy button")
	}
	if !card.ShowDownload {
		t.Fatal("expected clipboard text card to show download button")
	}
	if card.CopyURL != "/s/abcd/text" {
		t.Fatalf("unexpected copy url: %q", card.CopyURL)
	}
	if card.DownloadURL != "/s/abcd/raw" {
		t.Fatalf("unexpected download url: %q", card.DownloadURL)
	}
	if !strings.Contains(card.PreviewText, "第一行") {
		t.Fatalf("unexpected preview text: %q", card.PreviewText)
	}
}

func TestListVisibleSharesIncludesFileQuickDownloadInfo(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "hello.txt")
	if err := os.WriteFile(filePath, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	mgr := &Manager{
		cfg:       DefaultConfig(),
		templates: mustParseTemplates(),
		shares: map[string]*Share{
			"share-1": {
				ID:      "share-1",
				Code:    "abcd",
				Kind:    shareKindFile,
				Name:    "hello share",
				Path:    filePath,
				Visible: true,
			},
		},
		uploads: make(map[string]*uploadSession),
	}

	cards := mgr.listVisibleShares(langZH)
	if len(cards) != 1 {
		t.Fatalf("expected 1 card, got %d", len(cards))
	}
	card := cards[0]
	if !card.ShowDownload {
		t.Fatal("expected file card to show quick download")
	}
	if card.FileName != "hello.txt" {
		t.Fatalf("unexpected file name: %q", card.FileName)
	}
	if card.FileSize == "" {
		t.Fatal("expected file size to be filled")
	}
}

func TestListVisibleSharesIncludesTextFilePreview(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "note.txt")
	if err := os.WriteFile(filePath, []byte("第一行\n第二行"), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	mgr := &Manager{
		cfg:       DefaultConfig(),
		templates: mustParseTemplates(),
		shares: map[string]*Share{
			"share-1": {
				ID:      "share-1",
				Code:    "abcd",
				Kind:    shareKindFile,
				Name:    "note share",
				Path:    filePath,
				Visible: true,
			},
		},
		uploads: make(map[string]*uploadSession),
	}

	cards := mgr.listVisibleShares(langZH)
	if len(cards) != 1 {
		t.Fatalf("expected 1 card, got %d", len(cards))
	}
	card := cards[0]
	if !card.ShowCopy {
		t.Fatal("expected text file card to show copy button")
	}
	if card.CopyURL != "/s/abcd/text" {
		t.Fatalf("unexpected copy url: %q", card.CopyURL)
	}
	if !strings.Contains(card.PreviewText, "第一行") {
		t.Fatalf("unexpected text preview: %q", card.PreviewText)
	}
}

func TestServeShareTextReturnsFullClipboardText(t *testing.T) {
	full := strings.Repeat("ab", 300)
	mgr := &Manager{
		cfg:       DefaultConfig(),
		templates: mustParseTemplates(),
		shares: map[string]*Share{
			"share-1": {
				ID:          "share-1",
				Code:        "abcd",
				Kind:        shareKindClipboardText,
				Name:        "clip",
				TextContent: full,
			},
		},
		uploads: make(map[string]*uploadSession),
	}

	req := httptest.NewRequest(http.MethodGet, "/s/abcd/text", nil)
	rec := httptest.NewRecorder()
	mgr.handleShare(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if rec.Body.String() != full {
		t.Fatalf("expected full clipboard text")
	}
}

func TestServeShareTextReturnsFullFileText(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "full.txt")
	full := strings.Repeat("line-", 600)
	if err := os.WriteFile(filePath, []byte(full), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	mgr := &Manager{
		cfg:       DefaultConfig(),
		templates: mustParseTemplates(),
		shares: map[string]*Share{
			"share-1": {
				ID:   "share-1",
				Code: "abcd",
				Kind: shareKindFile,
				Name: "full",
				Path: filePath,
			},
		},
		uploads: make(map[string]*uploadSession),
	}

	req := httptest.NewRequest(http.MethodGet, "/s/abcd/text", nil)
	rec := httptest.NewRecorder()
	mgr.handleShare(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if rec.Body.String() != full {
		t.Fatalf("expected full file text")
	}
}

func TestClipboardImageShareContentAndRaw(t *testing.T) {
	image := []byte{0x89, 0x50, 0x4E, 0x47}
	mgr := &Manager{
		cfg:       DefaultConfig(),
		templates: mustParseTemplates(),
		shares: map[string]*Share{
			"share-1": {
				ID:         "share-1",
				Code:       "abcd",
				Kind:       shareKindClipboardImage,
				Name:       "clip-image",
				BinaryData: image,
				MimeType:   "image/png",
			},
		},
		uploads: make(map[string]*uploadSession),
	}

	contentReq := httptest.NewRequest(http.MethodGet, "/s/abcd/content", nil)
	contentRec := httptest.NewRecorder()
	mgr.handleShare(contentRec, contentReq)
	if contentRec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, contentRec.Code)
	}
	if got := contentRec.Header().Get("Content-Type"); got != "image/png" {
		t.Fatalf("expected image/png content type, got %q", got)
	}
	if !bytes.Equal(contentRec.Body.Bytes(), image) {
		t.Fatalf("expected image content bytes, got %v", contentRec.Body.Bytes())
	}

	rawReq := httptest.NewRequest(http.MethodGet, "/s/abcd/raw", nil)
	rawRec := httptest.NewRecorder()
	mgr.handleShare(rawRec, rawReq)
	if rawRec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rawRec.Code)
	}
	if got := rawRec.Header().Get("Content-Disposition"); !strings.Contains(got, "clip-image.png") {
		t.Fatalf("expected png download filename, got %q", got)
	}
}

func TestImageFileShareContentEndpoint(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "pic.png")
	image := []byte{0x89, 0x50, 0x4E, 0x47}
	if err := os.WriteFile(filePath, image, 0o644); err != nil {
		t.Fatalf("write image: %v", err)
	}

	mgr := &Manager{
		cfg:       DefaultConfig(),
		templates: mustParseTemplates(),
		shares: map[string]*Share{
			"share-1": {
				ID:   "share-1",
				Code: "abcd",
				Kind: shareKindFile,
				Name: "pic share",
				Path: filePath,
			},
		},
		uploads: make(map[string]*uploadSession),
	}

	req := httptest.NewRequest(http.MethodGet, "/s/abcd/content", nil)
	rec := httptest.NewRecorder()
	mgr.handleShare(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); !strings.Contains(got, "image/png") {
		t.Fatalf("expected image/png content type, got %q", got)
	}
}

func mustParseTemplates() *template.Template {
	return template.Must(template.New("pages").Funcs(template.FuncMap{
		"tr": tr,
	}).Parse(homeHTML + manageHTML + shareHTML))
}
