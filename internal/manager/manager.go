package manager

import (
	"archive/zip"
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"mime"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	qrcode "github.com/skip2/go-qrcode"
)

const (
	defaultHost            = "0.0.0.0"
	defaultPort            = 21910
	defaultUploadChunkSize = 2 << 20
)

type Config struct {
	BindHost string
	Port     int
}

type Manager struct {
	cfg       Config
	server    *http.Server
	templates *template.Template

	mu      sync.RWMutex
	shares  map[string]*Share
	uploads map[string]*uploadSession
}

type Share struct {
	ID          string
	Code        string
	Path        string
	Name        string
	IsDir       bool
	Visible     bool
	Password    string
	CreatedAt   time.Time
	LastUpdated time.Time
}

type CreateShareRequest struct {
	Path     string `json:"path"`
	Password string `json:"password"`
}

type managePageData struct {
	Title      string
	PublicURL  string
	VisibleURL string
	Shares     []manageShareCard
}

type manageShareCard struct {
	ID             string
	Code           string
	Name           string
	Path           string
	Type           string
	Mode           string
	Visibility     string
	CreatedAt      string
	UpdatedAt      string
	LocalURL       string
	PublicURL      string
	PrimaryURL     string
	QRCodeDataURL  template.URL
	NetworkLinks   []string
	NameInput      string
	VisibleChecked bool
}

type homePageData struct {
	Title         string
	VisibleShares []publicShareCard
	ErrorMessage  string
}

type publicShareCard struct {
	Name        string
	Code        string
	Type        string
	URL         string
	Unavailable bool
	Status      string
}

type sharePageData struct {
	Title          string
	ShareCode      string
	SharedName     string
	SharedPath     string
	CurrentPath    string
	CurrentLabel   string
	ParentURL      string
	Breadcrumbs    []breadcrumbItem
	IsDir          bool
	UploadEnabled  bool
	Items          []dirItem
	Address        string
	ErrorMessage   string
	SuccessMessage string
	ChunkSize      int64
	Unavailable    bool
}

type dirItem struct {
	Name       string
	Size       string
	ModTime    string
	URL        string
	ArchiveURL string
	IsDir      bool
}

type breadcrumbItem struct {
	Name string
	URL  string
}

type uploadSession struct {
	mu            sync.Mutex
	ID            string
	ShareID       string
	RelativePath  string
	FileName      string
	TempPath      string
	TargetPath    string
	TotalSize     int64
	ChunkSize     int64
	TotalChunks   int
	UploadedBytes int64
	NextIndex     int
}

type uploadStartRequest struct {
	Path        string `json:"path"`
	Password    string `json:"password"`
	FilePath    string `json:"filePath"`
	FileSize    int64  `json:"fileSize"`
	ChunkSize   int64  `json:"chunkSize"`
	TotalChunks int    `json:"totalChunks"`
}

type uploadStartResponse struct {
	UploadID      string `json:"uploadId"`
	UploadedBytes int64  `json:"uploadedBytes"`
	NextIndex     int    `json:"nextIndex"`
	ChunkSize     int64  `json:"chunkSize"`
	TotalSize     int64  `json:"totalSize"`
	Done          bool   `json:"done"`
}

func DefaultConfig() Config {
	return Config{BindHost: defaultHost, Port: defaultPort}
}

func Run(cfg Config) error {
	mgr := &Manager{
		cfg:       cfg,
		templates: template.Must(template.New("pages").Parse(homeHTML + manageHTML + shareHTML)),
		shares:    make(map[string]*Share),
		uploads:   make(map[string]*uploadSession),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", mgr.handleHome)
	mux.HandleFunc("/api/ping", mgr.handlePing)
	mux.HandleFunc("/api/shutdown", mgr.handleShutdown)
	mux.HandleFunc("/api/shares", mgr.handleCreateShare)
	mux.HandleFunc("/api/shares/", mgr.handleShareAPI)
	mux.HandleFunc("/manage", mgr.handleManage)
	mux.HandleFunc("/manage/shares/", mgr.handleManageShareAction)
	mux.HandleFunc("/s/", mgr.handleShare)

	mgr.server = &http.Server{
		Addr:    fmt.Sprintf("%s:%d", cfg.BindHost, cfg.Port),
		Handler: mux,
	}

	return mgr.server.ListenAndServe()
}

func LocalAPI(path string) string {
	return fmt.Sprintf("http://127.0.0.1:%d%s", defaultPort, path)
}

func LocalManageURL() string {
	return fmt.Sprintf("http://127.0.0.1:%d/manage", defaultPort)
}

func IsReachable() bool {
	client := &http.Client{Timeout: 800 * time.Millisecond}
	resp, err := client.Get(LocalAPI("/api/ping"))
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func (m *Manager) handleHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	code := strings.TrimSpace(r.URL.Query().Get("code"))
	if code != "" {
		share := m.getShareByCode(code)
		if share == nil {
			data := homePageData{
				Title:         "Web Share",
				VisibleShares: m.listVisibleShares(),
				ErrorMessage:  "分享码不存在",
			}
			if err := m.templates.ExecuteTemplate(w, "home", data); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}

		http.Redirect(w, r, "/s/"+share.Code, http.StatusSeeOther)
		return
	}

	data := homePageData{
		Title:         "Web Share",
		VisibleShares: m.listVisibleShares(),
	}
	if err := m.templates.ExecuteTemplate(w, "home", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (m *Manager) handlePing(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (m *Manager) handleShutdown(w http.ResponseWriter, r *http.Request) {
	if !isLocalRequest(r.RemoteAddr) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.WriteHeader(http.StatusNoContent)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = m.server.Shutdown(ctx)
	}()
}

func (m *Manager) handleCreateShare(w http.ResponseWriter, r *http.Request) {
	if !isLocalRequest(r.RemoteAddr) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CreateShareRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	share, err := m.createShare(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"id":   share.ID,
		"code": share.Code,
		"name": share.Name,
		"url":  fmt.Sprintf("/s/%s", share.Code),
	})
}

func (m *Manager) handleShareAPI(w http.ResponseWriter, r *http.Request) {
	if !isLocalRequest(r.RemoteAddr) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/shares/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 2 || parts[1] != "stop" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !m.deleteShare(parts[0]) {
		http.NotFound(w, r)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (m *Manager) handleManage(w http.ResponseWriter, r *http.Request) {
	if !isLocalRequest(r.RemoteAddr) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	host := "127.0.0.1"
	if r.Host != "" {
		host = r.Host
	}

	data := managePageData{
		Title:      "Web Share Manager",
		PublicURL:  fmt.Sprintf("http://%s/", host),
		VisibleURL: fmt.Sprintf("http://%s/", host),
		Shares:     m.listManageCards(),
	}

	if err := m.templates.ExecuteTemplate(w, "manage", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (m *Manager) handleManageShareAction(w http.ResponseWriter, r *http.Request) {
	if !isLocalRequest(r.RemoteAddr) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/manage/shares/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 2 || parts[1] != "update" {
		http.NotFound(w, r)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}

	if err := m.updateShare(parts[0], r.FormValue("name"), r.FormValue("visible") == "on"); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	http.Redirect(w, r, "/manage", http.StatusSeeOther)
}

func (m *Manager) handleShare(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/s/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}

	share := m.getShareByCode(parts[0])
	if share == nil {
		http.NotFound(w, r)
		return
	}

	switch {
	case len(parts) == 1 && r.Method == http.MethodGet:
		m.renderSharePage(w, r, share)
	case len(parts) == 2 && parts[1] == "raw" && (r.Method == http.MethodGet || r.Method == http.MethodHead):
		m.serveShareRaw(w, r, share)
	case len(parts) == 2 && parts[1] == "archive" && r.Method == http.MethodGet:
		m.serveShareArchive(w, r, share)
	case len(parts) == 3 && parts[1] == "upload" && parts[2] == "start" && r.Method == http.MethodPost:
		m.handleUploadStart(w, r, share)
	case len(parts) == 3 && parts[1] == "upload" && parts[2] == "status" && r.Method == http.MethodGet:
		m.handleUploadStatus(w, r, share)
	case len(parts) == 3 && parts[1] == "upload" && parts[2] == "chunk" && r.Method == http.MethodPost:
		m.handleUploadChunk(w, r, share)
	default:
		http.NotFound(w, r)
	}
}

func (m *Manager) createShare(req CreateShareRequest) (*Share, error) {
	target, err := filepath.Abs(req.Path)
	if err != nil {
		return nil, fmt.Errorf("resolve target path: %w", err)
	}

	info, err := os.Stat(target)
	if err != nil {
		return nil, fmt.Errorf("stat target path: %w", err)
	}

	if !info.IsDir() {
		req.Password = ""
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	name := m.allocateUniqueName(info.Name(), "")
	share := &Share{
		ID:          newShareID(),
		Code:        m.allocateUniqueCode(),
		Path:        target,
		Name:        name,
		IsDir:       info.IsDir(),
		Visible:     false,
		Password:    req.Password,
		CreatedAt:   time.Now(),
		LastUpdated: time.Now(),
	}

	m.shares[share.ID] = share
	return copyShare(share), nil
}

func (m *Manager) updateShare(id, name string, visible bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	share, ok := m.shares[id]
	if !ok {
		return errors.New("share not found")
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("name cannot be empty")
	}

	share.Name = m.allocateUniqueName(name, id)
	share.Visible = visible
	share.LastUpdated = time.Now()
	return nil
}

func (m *Manager) deleteShare(id string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.shares[id]; !ok {
		return false
	}
	delete(m.shares, id)
	return true
}

func (m *Manager) getShareByCode(code string) *Share {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, share := range m.shares {
		if strings.EqualFold(share.Code, code) {
			return copyShare(share)
		}
	}

	return nil
}

func (m *Manager) listVisibleShares() []publicShareCard {
	m.mu.RLock()
	defer m.mu.RUnlock()

	shares := make([]publicShareCard, 0, len(m.shares))
	for _, share := range m.shares {
		if !share.Visible {
			continue
		}

		card := publicShareCard{
			Name: share.Name,
			Code: share.Code,
			URL:  "/s/" + share.Code,
		}
		if share.IsDir {
			card.Type = "文件夹"
		} else {
			card.Type = "文件"
		}
		if _, err := os.Stat(share.Path); err != nil {
			card.Unavailable = true
			card.Status = "已失效"
		} else {
			card.Status = "可访问"
		}
		shares = append(shares, card)
	}

	sort.Slice(shares, func(i, j int) bool {
		return strings.ToLower(shares[i].Name) < strings.ToLower(shares[j].Name)
	})
	return shares
}

func (m *Manager) listManageCards() []manageShareCard {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cards := make([]manageShareCard, 0, len(m.shares))
	localIPs := listLocalIPv4s()
	for _, share := range m.shares {
		localURL := fmt.Sprintf("http://127.0.0.1:%d/s/%s", m.cfg.Port, share.Code)
		card := manageShareCard{
			ID:             share.ID,
			Code:           share.Code,
			Name:           share.Name,
			NameInput:      share.Name,
			Path:           share.Path,
			CreatedAt:      share.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt:      share.LastUpdated.Format("2006-01-02 15:04:05"),
			LocalURL:       localURL,
			PublicURL:      fmt.Sprintf("http://127.0.0.1:%d/?code=%s", m.cfg.Port, share.Code),
			VisibleChecked: share.Visible,
		}
		if share.IsDir {
			card.Type = "文件夹"
		} else {
			card.Type = "文件"
		}
		if share.IsDir && share.Password != "" {
			card.Mode = "上传已启用"
		} else {
			card.Mode = "只读"
		}
		if share.Visible {
			card.Visibility = "首页可见"
		} else {
			card.Visibility = "首页隐藏"
		}
		for _, ip := range localIPs {
			card.NetworkLinks = append(card.NetworkLinks, fmt.Sprintf("http://%s:%d/s/%s", ip, m.cfg.Port, share.Code))
		}
		card.PrimaryURL = localURL
		if len(card.NetworkLinks) > 0 {
			card.PrimaryURL = card.NetworkLinks[0]
		}
		card.QRCodeDataURL = buildQRCodeDataURL(card.PrimaryURL)
		cards = append(cards, card)
	}

	sort.Slice(cards, func(i, j int) bool {
		return cards[i].CreatedAt > cards[j].CreatedAt
	})
	return cards
}

func (m *Manager) renderSharePage(w http.ResponseWriter, r *http.Request, share *Share) {
	rootInfo, err := os.Stat(share.Path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			data := sharePageData{
				Title:        "Web Share",
				ShareCode:    share.Code,
				SharedName:   share.Name,
				SharedPath:   share.Path,
				IsDir:        share.IsDir,
				Address:      fmt.Sprintf("http://%s%s", r.Host, browseURL(share.Code, "")),
				ChunkSize:    defaultUploadChunkSize,
				Unavailable:  true,
				ErrorMessage: "该分享对应的文件或文件夹已不存在，可能已被移动或删除。",
			}
			if err := m.templates.ExecuteTemplate(w, "share", data); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}
		http.Error(w, "failed to inspect shared path", http.StatusInternalServerError)
		return
	}

	if share.IsDir && !rootInfo.IsDir() {
		http.Error(w, "share root is not a directory", http.StatusBadRequest)
		return
	}

	currentPath, currentDir, err := resolveShareSubpath(share.Path, r.URL.Query().Get("path"), true)
	if err != nil {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	info, err := os.Stat(currentDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			http.Redirect(w, r, withShareMessageAt(share.Code, parentPathOrRoot(currentPath), "error", "当前目录已不存在，可能已被移动或删除。"), http.StatusSeeOther)
			return
		}
		http.Error(w, "failed to inspect current path", http.StatusInternalServerError)
		return
	}
	if !info.IsDir() {
		http.Error(w, "path is not a directory", http.StatusBadRequest)
		return
	}

	data := sharePageData{
		Title:          "Web Share",
		ShareCode:      share.Code,
		SharedName:     share.Name,
		SharedPath:     share.Path,
		CurrentPath:    currentPath,
		CurrentLabel:   currentPathLabel(currentPath),
		ParentURL:      parentBrowseURL(share.Code, currentPath),
		Breadcrumbs:    buildBreadcrumbs(share.Code, currentPath),
		IsDir:          share.IsDir,
		UploadEnabled:  share.IsDir && share.Password != "",
		Address:        fmt.Sprintf("http://%s%s", r.Host, browseURL(share.Code, currentPath)),
		ErrorMessage:   r.URL.Query().Get("error"),
		SuccessMessage: r.URL.Query().Get("success"),
		Items:          listItems(share, currentPath, currentDir),
		ChunkSize:      defaultUploadChunkSize,
	}

	if err := m.templates.ExecuteTemplate(w, "share", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (m *Manager) serveShareRaw(w http.ResponseWriter, r *http.Request, share *Share) {
	if share.IsDir {
		relativePath, target, err := resolveShareSubpath(share.Path, r.URL.Query().Get("path"), false)
		if err != nil {
			http.Error(w, "invalid file path", http.StatusBadRequest)
			return
		}
		info, err := os.Stat(target)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				if maybeRedirectToShareError(w, r, share.Code, parentPathOrRoot(relativePath), "文件已不存在或已被移动。") {
					return
				}
				http.Error(w, "文件已不存在或已被移动。", http.StatusNotFound)
				return
			}
			http.Error(w, "failed to inspect file", http.StatusInternalServerError)
			return
		}
		if info.IsDir() {
			http.Error(w, "directories are not downloadable", http.StatusBadRequest)
			return
		}

		serveFileDownload(w, r, target, info.Name())
		return
	}

	if _, err := os.Stat(share.Path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if maybeRedirectToShareError(w, r, share.Code, "", "文件已不存在或已被移动。") {
				return
			}
			http.Error(w, "文件已不存在或已被移动。", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to inspect file", http.StatusInternalServerError)
		return
	}

	serveFileDownload(w, r, share.Path, filepath.Base(share.Path))
}

func (m *Manager) serveShareArchive(w http.ResponseWriter, r *http.Request, share *Share) {
	if !share.IsDir {
		http.Error(w, "only directory shares can be archived", http.StatusBadRequest)
		return
	}

	relativePath, target, err := resolveShareSubpath(share.Path, r.URL.Query().Get("path"), true)
	if err != nil {
		http.Error(w, "invalid archive path", http.StatusBadRequest)
		return
	}

	info, err := os.Stat(target)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			http.Redirect(w, r, withShareMessageAt(share.Code, parentPathOrRoot(relativePath), "error", "要下载的文件夹已不存在，可能已被移动或删除。"), http.StatusSeeOther)
			return
		}
		http.Error(w, "failed to inspect archive path", http.StatusInternalServerError)
		return
	}
	if !info.IsDir() {
		http.Error(w, "archive target is not a directory", http.StatusBadRequest)
		return
	}

	archiveBase := share.Name
	if relativePath != "" {
		archiveBase = filepath.Base(filepath.FromSlash(relativePath))
	}
	archiveName := sanitizeArchiveName(archiveBase) + ".zip"
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, archiveName))

	zipWriter := zip.NewWriter(w)
	defer zipWriter.Close()

	if err := writeZipArchive(zipWriter, target, archiveBase); err != nil {
		http.Error(w, "failed to create archive: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

func (m *Manager) handleUploadStart(w http.ResponseWriter, r *http.Request, share *Share) {
	if !share.IsDir {
		http.Error(w, "uploads are only supported for directories", http.StatusBadRequest)
		return
	}
	if share.Password == "" {
		http.Error(w, "uploads are disabled for this share", http.StatusForbidden)
		return
	}
	var req uploadStartRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if subtle.ConstantTimeCompare([]byte(req.Password), []byte(share.Password)) != 1 {
		http.Error(w, "invalid upload password", http.StatusForbidden)
		return
	}

	session, err := m.prepareUploadSession(share, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if session.UploadedBytes == session.TotalSize {
		m.touchShare(share.ID)
	}

	m.writeJSON(w, http.StatusOK, uploadStartResponse{
		UploadID:      session.ID,
		UploadedBytes: session.UploadedBytes,
		NextIndex:     session.NextIndex,
		ChunkSize:     session.ChunkSize,
		TotalSize:     session.TotalSize,
		Done:          session.UploadedBytes == session.TotalSize,
	})
}

func (m *Manager) handleUploadStatus(w http.ResponseWriter, r *http.Request, share *Share) {
	session, err := m.lookupUploadSession(share, r.URL.Query().Get("upload_id"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	session.mu.Lock()
	resp := uploadStartResponse{
		UploadID:      session.ID,
		UploadedBytes: session.UploadedBytes,
		NextIndex:     session.NextIndex,
		ChunkSize:     session.ChunkSize,
		TotalSize:     session.TotalSize,
		Done:          session.UploadedBytes == session.TotalSize,
	}
	session.mu.Unlock()

	m.writeJSON(w, http.StatusOK, resp)
}

func (m *Manager) handleUploadChunk(w http.ResponseWriter, r *http.Request, share *Share) {
	session, err := m.lookupUploadSession(share, r.URL.Query().Get("upload_id"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	index, err := strconv.Atoi(r.URL.Query().Get("index"))
	if err != nil || index < 0 {
		http.Error(w, "invalid chunk index", http.StatusBadRequest)
		return
	}

	uploadedBytes, done, err := m.appendUploadChunk(session, index, r.Body, r.ContentLength)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if done {
		m.touchShare(share.ID)
	}

	m.writeJSON(w, http.StatusOK, map[string]any{
		"uploadId":      session.ID,
		"uploadedBytes": uploadedBytes,
		"nextIndex":     index + 1,
		"done":          done,
	})
}

func (m *Manager) prepareUploadSession(share *Share, req uploadStartRequest) (*uploadSession, error) {
	if info, err := os.Stat(share.Path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, errors.New("该分享对应的文件夹已不存在，无法继续上传")
		}
		return nil, fmt.Errorf("检查共享目录失败: %w", err)
	} else if !info.IsDir() {
		return nil, errors.New("该分享对应的目录已失效")
	}

	currentPath, currentDir, err := resolveShareSubpath(share.Path, strings.TrimSpace(req.Path), true)
	if err != nil {
		return nil, errors.New("目录路径无效")
	}
	if info, err := os.Stat(currentDir); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, errors.New("当前目录已不存在，无法继续上传")
		}
		return nil, fmt.Errorf("检查当前目录失败: %w", err)
	} else if !info.IsDir() {
		return nil, errors.New("当前目录已失效")
	}

	relativePath, parentDir, target, name, err := resolveUploadTargetPath(share.Path, currentPath, req.FilePath)
	if err != nil {
		return nil, err
	}
	if req.FileSize < 0 {
		return nil, errors.New("无效文件大小")
	}
	if req.ChunkSize <= 0 {
		return nil, errors.New("无效分片大小")
	}
	if req.TotalChunks <= 0 {
		return nil, errors.New("无效分片数量")
	}

	expectedChunks := 1
	if req.FileSize > 0 {
		expectedChunks = int((req.FileSize + req.ChunkSize - 1) / req.ChunkSize)
	}
	if expectedChunks != req.TotalChunks {
		return nil, errors.New("分片数量与文件大小不匹配")
	}

	if err := os.MkdirAll(parentDir, 0o755); err != nil {
		return nil, fmt.Errorf("创建目标目录失败: %w", err)
	}

	if _, err := os.Stat(target); err == nil {
		return nil, errors.New("目标文件已存在")
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("检查目标文件失败: %w", err)
	}

	key := uploadSessionKey(share.ID, relativePath)

	m.mu.Lock()
	if existing, ok := m.uploads[key]; ok {
		m.mu.Unlock()
		existing.mu.Lock()
		defer existing.mu.Unlock()
		if existing.TotalSize != req.FileSize || existing.ChunkSize != req.ChunkSize || existing.TotalChunks != req.TotalChunks {
			return nil, errors.New("同名文件已有不同上传任务进行中")
		}
		return existing, nil
	}

	session := &uploadSession{
		ID:           newUploadID(),
		ShareID:      share.ID,
		RelativePath: relativePath,
		FileName:     name,
		TempPath:     target + ".webshare.part",
		TargetPath:   target,
		TotalSize:    req.FileSize,
		ChunkSize:    req.ChunkSize,
		TotalChunks:  req.TotalChunks,
	}

	if info, err := os.Stat(session.TempPath); err == nil {
		if info.Size() > session.TotalSize {
			m.mu.Unlock()
			return nil, errors.New("上传临时文件状态无效")
		}
		if info.Size() < session.TotalSize && info.Size()%session.ChunkSize != 0 {
			m.mu.Unlock()
			return nil, errors.New("上传临时文件未对齐到分片边界")
		}
		session.UploadedBytes = info.Size()
		session.NextIndex = int(info.Size() / session.ChunkSize)
	} else if !errors.Is(err, os.ErrNotExist) {
		m.mu.Unlock()
		return nil, fmt.Errorf("检查上传临时文件失败: %w", err)
	}

	if session.TotalSize == 0 && session.UploadedBytes == 0 {
		file, err := os.OpenFile(session.TargetPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if err != nil {
			m.mu.Unlock()
			return nil, fmt.Errorf("创建空文件失败: %w", err)
		}
		_ = file.Close()
		m.mu.Unlock()
		return session, nil
	}

	m.uploads[key] = session
	m.mu.Unlock()
	return session, nil
}

func (m *Manager) lookupUploadSession(share *Share, uploadID string) (*uploadSession, error) {
	uploadID = strings.TrimSpace(uploadID)
	if uploadID == "" {
		return nil, errors.New("missing upload id")
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, session := range m.uploads {
		if session.ID == uploadID && session.ShareID == share.ID {
			return session, nil
		}
	}
	return nil, errors.New("upload session not found")
}

func (m *Manager) appendUploadChunk(session *uploadSession, index int, src io.Reader, contentLength int64) (int64, bool, error) {
	session.mu.Lock()
	defer session.mu.Unlock()

	if session.UploadedBytes == session.TotalSize {
		return session.UploadedBytes, true, nil
	}
	if index != session.NextIndex {
		return session.UploadedBytes, false, fmt.Errorf("unexpected chunk index: want %d", session.NextIndex)
	}

	expectedSize := session.ChunkSize
	remaining := session.TotalSize - session.UploadedBytes
	if remaining < expectedSize {
		expectedSize = remaining
	}
	if contentLength != expectedSize {
		return session.UploadedBytes, false, fmt.Errorf("unexpected chunk size: want %d bytes", expectedSize)
	}

	dst, err := os.OpenFile(session.TempPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return session.UploadedBytes, false, fmt.Errorf("open upload target: %w", err)
	}
	defer dst.Close()

	written, err := io.Copy(dst, io.LimitReader(src, expectedSize))
	if err != nil {
		return session.UploadedBytes, false, fmt.Errorf("write upload chunk: %w", err)
	}
	if written != expectedSize {
		return session.UploadedBytes, false, fmt.Errorf("short chunk write: wrote %d bytes", written)
	}

	session.UploadedBytes += written
	session.NextIndex++

	if session.UploadedBytes < session.TotalSize {
		return session.UploadedBytes, false, nil
	}

	if err := dst.Close(); err != nil {
		return session.UploadedBytes, false, fmt.Errorf("close upload target: %w", err)
	}
	if err := os.Rename(session.TempPath, session.TargetPath); err != nil {
		return session.UploadedBytes, false, fmt.Errorf("finalize upload: %w", err)
	}

	m.mu.Lock()
	delete(m.uploads, uploadSessionKey(session.ShareID, session.RelativePath))
	m.mu.Unlock()

	return session.UploadedBytes, true, nil
}

func (m *Manager) writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func uploadSessionKey(shareID, relativePath string) string {
	return shareID + "|" + strings.ToLower(filepath.ToSlash(relativePath))
}

func (m *Manager) touchShare(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if share, ok := m.shares[id]; ok {
		share.LastUpdated = time.Now()
	}
}

func (m *Manager) allocateUniqueName(baseName, ignoreID string) string {
	baseName = strings.TrimSpace(baseName)
	if baseName == "" {
		baseName = "未命名分享"
	}

	name := baseName
	index := 2
	for m.nameExists(name, ignoreID) {
		name = fmt.Sprintf("%s (%d)", baseName, index)
		index++
	}
	return name
}

func (m *Manager) nameExists(name, ignoreID string) bool {
	for id, share := range m.shares {
		if id == ignoreID {
			continue
		}
		if strings.EqualFold(share.Name, name) {
			return true
		}
	}
	return false
}

func (m *Manager) allocateUniqueCode() string {
	for {
		code := newShareCode()
		if !m.codeExists(code) {
			return code
		}
	}
}

func (m *Manager) codeExists(code string) bool {
	for _, share := range m.shares {
		if strings.EqualFold(share.Code, code) {
			return true
		}
	}
	return false
}

func listItems(share *Share, currentPath, currentDir string) []dirItem {
	info, err := os.Stat(share.Path)
	if err != nil {
		return nil
	}

	if !share.IsDir {
		return []dirItem{{
			Name:    filepath.Base(share.Path),
			Size:    formatSize(info.Size()),
			ModTime: info.ModTime().Format("2006-01-02 15:04"),
			URL:     fmt.Sprintf("/s/%s/raw", share.Code),
			IsDir:   false,
		}}
	}

	entries, err := os.ReadDir(currentDir)
	if err != nil {
		return nil
	}

	items := make([]dirItem, 0, len(entries))
	for _, entry := range entries {
		entryInfo, err := entry.Info()
		if err != nil {
			continue
		}

		item := dirItem{
			Name:    entry.Name(),
			ModTime: entryInfo.ModTime().Format("2006-01-02 15:04"),
			IsDir:   entry.IsDir(),
		}
		if entry.IsDir() {
			item.Size = "folder"
			item.URL = browseURL(share.Code, joinRelativePath(currentPath, entry.Name()))
			item.ArchiveURL = archiveURL(share.Code, joinRelativePath(currentPath, entry.Name()))
		} else {
			item.Size = formatSize(entryInfo.Size())
			item.URL = fmt.Sprintf("/s/%s/raw?path=%s", share.Code, url.QueryEscape(joinRelativePath(currentPath, entry.Name())))
		}
		items = append(items, item)
	}

	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
	})
	return items
}

func serveFileDownload(w http.ResponseWriter, r *http.Request, path, downloadName string) {
	file, err := os.Open(path)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		http.NotFound(w, r)
		return
	}

	contentType := mime.TypeByExtension(filepath.Ext(downloadName))
	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	} else {
		w.Header().Set("Content-Type", "application/octet-stream")
	}
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, downloadName))
	http.ServeContent(w, r, downloadName, info.ModTime(), file)
}

func writeZipArchive(zipWriter *zip.Writer, root, rootName string) error {
	rootName = sanitizeArchiveName(rootName)

	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if path == root {
			return nil
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		entryName := filepath.ToSlash(filepath.Join(rootName, rel))

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = entryName
		if info.IsDir() {
			header.Name += "/"
			_, err = zipWriter.CreateHeader(header)
			return err
		}

		header.Method = zip.Deflate
		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return err
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(writer, file)
		return err
	})
}

func sanitizeArchiveName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "share"
	}

	replacer := strings.NewReplacer(
		"\\", "_",
		"/", "_",
		":", "_",
		"*", "_",
		"?", "_",
		`"`, "_",
		"<", "_",
		">", "_",
		"|", "_",
	)
	name = replacer.Replace(name)
	name = strings.TrimSpace(name)
	if name == "" {
		return "share"
	}
	return name
}

func writeUploadedFile(target string, src io.Reader) error {
	if _, err := os.Stat(target); err == nil {
		return errors.New("target already exists")
	}

	dst, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	return err
}

func formatSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}

	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f %ciB", float64(size)/float64(div), "KMGTPE"[exp])
}

func withShareMessage(shareCode, key, value string) string {
	return fmt.Sprintf("/s/%s?%s=%s", shareCode, key, url.QueryEscape(value))
}

func withShareMessageAt(shareCode, currentPath, key, value string) string {
	base := browseURL(shareCode, currentPath)
	separator := "?"
	if strings.Contains(base, "?") {
		separator = "&"
	}
	return base + separator + key + "=" + url.QueryEscape(value)
}

func maybeRedirectToShareError(w http.ResponseWriter, r *http.Request, shareCode, currentPath, message string) bool {
	if r.Method != http.MethodGet {
		return false
	}
	http.Redirect(w, r, withShareMessageAt(shareCode, currentPath, "error", message), http.StatusSeeOther)
	return true
}

func parentPathOrRoot(currentPath string) string {
	if currentPath == "" {
		return ""
	}
	parent := filepath.ToSlash(filepath.Dir(filepath.FromSlash(currentPath)))
	if parent == "." {
		return ""
	}
	return parent
}

func browseURL(shareCode, currentPath string) string {
	base := "/s/" + shareCode
	if currentPath == "" {
		return base
	}
	return base + "?path=" + url.QueryEscape(currentPath)
}

func archiveURL(shareCode, currentPath string) string {
	base := "/s/" + shareCode + "/archive"
	if currentPath == "" {
		return base
	}
	return base + "?path=" + url.QueryEscape(currentPath)
}

func parentBrowseURL(shareCode, currentPath string) string {
	if currentPath == "" {
		return ""
	}

	parent := filepath.ToSlash(filepath.Dir(filepath.FromSlash(currentPath)))
	if parent == "." {
		parent = ""
	}
	return browseURL(shareCode, parent)
}

func buildBreadcrumbs(shareCode, currentPath string) []breadcrumbItem {
	items := []breadcrumbItem{{
		Name: "根目录",
		URL:  browseURL(shareCode, ""),
	}}
	if currentPath == "" {
		return items
	}

	parts := strings.Split(currentPath, "/")
	current := ""
	for _, part := range parts {
		current = joinRelativePath(current, part)
		items = append(items, breadcrumbItem{
			Name: part,
			URL:  browseURL(shareCode, current),
		})
	}
	return items
}

func currentPathLabel(currentPath string) string {
	if currentPath == "" {
		return "根目录"
	}
	return currentPath
}

func joinRelativePath(base, name string) string {
	if base == "" {
		return filepath.ToSlash(name)
	}
	return filepath.ToSlash(filepath.Join(filepath.FromSlash(base), name))
}

func resolveShareSubpath(root, requested string, allowDir bool) (string, string, error) {
	requested = strings.TrimSpace(requested)
	if requested == "" {
		return "", root, nil
	}

	clean := filepath.Clean(filepath.FromSlash(requested))
	if clean == "." {
		return "", root, nil
	}
	if filepath.IsAbs(clean) || filepath.VolumeName(clean) != "" {
		return "", "", errors.New("absolute path not allowed")
	}

	target := filepath.Clean(filepath.Join(root, clean))
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return "", "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", "", errors.New("path escapes root")
	}

	normalized := filepath.ToSlash(rel)
	if normalized == "." {
		normalized = ""
	}

	if !allowDir {
		return normalized, target, nil
	}
	return normalized, target, nil
}

func resolveUploadTargetPath(root, currentPath, requested string) (string, string, string, string, error) {
	requested = strings.TrimSpace(requested)
	if requested == "" {
		return "", "", "", "", errors.New("无效文件名")
	}

	clean := filepath.Clean(filepath.FromSlash(requested))
	if clean == "." || filepath.IsAbs(clean) || filepath.VolumeName(clean) != "" {
		return "", "", "", "", errors.New("无效文件名")
	}

	name := filepath.Base(clean)
	if name == "." || name == "" {
		return "", "", "", "", errors.New("无效文件名")
	}

	parentRequest := currentPath
	if dir := filepath.Dir(clean); dir != "." && dir != "" {
		parentRequest = joinRelativePath(currentPath, filepath.ToSlash(dir))
	}

	parentRelative, parentDir, err := resolveShareSubpath(root, parentRequest, true)
	if err != nil {
		return "", "", "", "", errors.New("目录路径无效")
	}

	relativePath := joinRelativePath(parentRelative, name)
	target := filepath.Join(parentDir, name)
	return relativePath, parentDir, target, name, nil
}

func isLocalRequest(remoteAddr string) bool {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}

	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func listLocalIPv4s() []string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}

	var result []string
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok || ipNet.IP == nil {
				continue
			}
			ip := ipNet.IP.To4()
			if ip == nil {
				continue
			}
			result = append(result, ip.String())
		}
	}

	sort.Strings(result)
	return result
}

func buildQRCodeDataURL(content string) template.URL {
	if content == "" {
		return ""
	}

	png, err := qrcode.Encode(content, qrcode.Medium, 256)
	if err != nil {
		return ""
	}

	return template.URL("data:image/png;base64," + base64.StdEncoding.EncodeToString(png))
}

func newShareID() string {
	var raw [8]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(raw[:])
}

func newUploadID() string {
	return newShareID()
}

func newShareCode() string {
	const alphabet = "23456789abcdefghjkmnpqrstuvwxyz"
	var raw [4]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return strings.ToLower(hex.EncodeToString(raw[:])[:6])
	}

	buf := make([]byte, 6)
	for i := range buf {
		buf[i] = alphabet[int(raw[i%len(raw)])%len(alphabet)]
	}
	return string(buf)
}

func copyShare(share *Share) *Share {
	if share == nil {
		return nil
	}
	copyValue := *share
	return &copyValue
}
