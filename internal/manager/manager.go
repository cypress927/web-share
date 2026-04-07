package manager

import (
	"crypto/rand"
	"crypto/subtle"
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
	"strings"
	"sync"
	"time"
)

const (
	defaultHost = "0.0.0.0"
	defaultPort = 21910
)

type Config struct {
	BindHost string
	Port     int
}

type Manager struct {
	cfg       Config
	server    *http.Server
	templates *template.Template

	mu     sync.RWMutex
	shares map[string]*Share
}

type Share struct {
	ID          string
	Path        string
	Name        string
	IsDir       bool
	Password    string
	CreatedAt   time.Time
	LastUpdated time.Time
}

type CreateShareRequest struct {
	Path     string `json:"path"`
	Password string `json:"password"`
}

type managePageData struct {
	Title  string
	Shares []manageShareCard
}

type manageShareCard struct {
	ID           string
	Name         string
	Path         string
	Type         string
	Mode         string
	CreatedAt    string
	UpdatedAt    string
	LocalURL     string
	NetworkLinks []string
}

type sharePageData struct {
	Title          string
	ShareID        string
	SharedName     string
	SharedPath     string
	IsDir          bool
	UploadEnabled  bool
	Items          []dirItem
	Address        string
	ErrorMessage   string
	SuccessMessage string
}

type dirItem struct {
	Name    string
	Size    string
	ModTime string
	URL     string
}

func DefaultConfig() Config {
	return Config{BindHost: defaultHost, Port: defaultPort}
}

func Run(cfg Config) error {
	mgr := &Manager{
		cfg:       cfg,
		templates: template.Must(template.New("pages").Parse(manageHTML + shareHTML)),
		shares:    make(map[string]*Share),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/ping", mgr.handlePing)
	mux.HandleFunc("/api/shares", mgr.handleCreateShare)
	mux.HandleFunc("/api/shares/", mgr.handleShareAPI)
	mux.HandleFunc("/manage", mgr.handleManage)
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

func (m *Manager) handlePing(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
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
		"id":  share.ID,
		"url": fmt.Sprintf("/s/%s", share.ID),
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

	data := managePageData{
		Title:  "Web Share Manager",
		Shares: m.listManageCards(),
	}

	if err := m.templates.ExecuteTemplate(w, "manage", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (m *Manager) handleShare(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/s/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}

	share := m.getShare(parts[0])
	if share == nil {
		http.NotFound(w, r)
		return
	}

	switch {
	case len(parts) == 1 && r.Method == http.MethodGet:
		m.renderSharePage(w, r, share)
	case len(parts) == 2 && parts[1] == "raw" && r.Method == http.MethodGet:
		m.serveShareRaw(w, r, share)
	case len(parts) == 2 && parts[1] == "upload" && r.Method == http.MethodPost:
		m.handleShareUpload(w, r, share)
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

	share := &Share{
		ID:          newShareID(),
		Path:        target,
		Name:        info.Name(),
		IsDir:       info.IsDir(),
		Password:    req.Password,
		CreatedAt:   time.Now(),
		LastUpdated: time.Now(),
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.shares[share.ID] = share
	return share, nil
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

func (m *Manager) getShare(id string) *Share {
	m.mu.RLock()
	defer m.mu.RUnlock()
	share, ok := m.shares[id]
	if !ok {
		return nil
	}

	copyValue := *share
	return &copyValue
}

func (m *Manager) listManageCards() []manageShareCard {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cards := make([]manageShareCard, 0, len(m.shares))
	localURL := fmt.Sprintf("http://127.0.0.1:%d", m.cfg.Port)
	localIPs := listLocalIPv4s()
	for _, share := range m.shares {
		card := manageShareCard{
			ID:        share.ID,
			Name:      share.Name,
			Path:      share.Path,
			CreatedAt: share.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt: share.LastUpdated.Format("2006-01-02 15:04:05"),
			LocalURL:  localURL + "/s/" + share.ID,
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
		for _, ip := range localIPs {
			card.NetworkLinks = append(card.NetworkLinks, fmt.Sprintf("http://%s:%d/s/%s", ip, m.cfg.Port, share.ID))
		}
		cards = append(cards, card)
	}

	sort.Slice(cards, func(i, j int) bool {
		return cards[i].CreatedAt > cards[j].CreatedAt
	})
	return cards
}

func (m *Manager) renderSharePage(w http.ResponseWriter, r *http.Request, share *Share) {
	data := sharePageData{
		Title:          "Web Share",
		ShareID:        share.ID,
		SharedName:     share.Name,
		SharedPath:     share.Path,
		IsDir:          share.IsDir,
		UploadEnabled:  share.IsDir && share.Password != "",
		Address:        fmt.Sprintf("http://%s/s/%s", r.Host, share.ID),
		ErrorMessage:   r.URL.Query().Get("error"),
		SuccessMessage: r.URL.Query().Get("success"),
		Items:          listItems(share),
	}

	if err := m.templates.ExecuteTemplate(w, "share", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (m *Manager) serveShareRaw(w http.ResponseWriter, r *http.Request, share *Share) {
	if share.IsDir {
		name := filepath.Base(filepath.Clean(r.URL.Query().Get("name")))
		if name == "." || name == "" {
			http.Error(w, "missing file name", http.StatusBadRequest)
			return
		}

		target := filepath.Join(share.Path, name)
		info, err := os.Stat(target)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		if info.IsDir() {
			http.Error(w, "directories are not downloadable", http.StatusBadRequest)
			return
		}

		serveFileDownload(w, r, target, info.Name())
		return
	}

	serveFileDownload(w, r, share.Path, share.Name)
}

func (m *Manager) handleShareUpload(w http.ResponseWriter, r *http.Request, share *Share) {
	if !share.IsDir {
		http.Error(w, "uploads are only supported for directories", http.StatusBadRequest)
		return
	}
	if share.Password == "" {
		http.Error(w, "uploads are disabled for this share", http.StatusForbidden)
		return
	}
	if subtle.ConstantTimeCompare([]byte(r.FormValue("password")), []byte(share.Password)) != 1 {
		http.Redirect(w, r, withShareMessage(share.ID, "error", "密码错误，上传已拒绝"), http.StatusSeeOther)
		return
	}
	if err := r.ParseMultipartForm(64 << 20); err != nil {
		http.Redirect(w, r, withShareMessage(share.ID, "error", "无法解析上传请求"), http.StatusSeeOther)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Redirect(w, r, withShareMessage(share.ID, "error", "请选择要上传的文件"), http.StatusSeeOther)
		return
	}
	defer file.Close()

	name := filepath.Base(header.Filename)
	if name == "." || name == "" {
		http.Redirect(w, r, withShareMessage(share.ID, "error", "无效文件名"), http.StatusSeeOther)
		return
	}

	target := filepath.Join(share.Path, name)
	if err := writeUploadedFile(target, file); err != nil {
		http.Redirect(w, r, withShareMessage(share.ID, "error", "保存上传文件失败："+err.Error()), http.StatusSeeOther)
		return
	}

	m.touchShare(share.ID)
	http.Redirect(w, r, withShareMessage(share.ID, "success", "上传成功"), http.StatusSeeOther)
}

func (m *Manager) touchShare(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if share, ok := m.shares[id]; ok {
		share.LastUpdated = time.Now()
	}
}

func listItems(share *Share) []dirItem {
	info, err := os.Stat(share.Path)
	if err != nil {
		return nil
	}

	if !share.IsDir {
		return []dirItem{{
			Name:    share.Name,
			Size:    formatSize(info.Size()),
			ModTime: info.ModTime().Format("2006-01-02 15:04"),
			URL:     fmt.Sprintf("/s/%s/raw", share.ID),
		}}
	}

	entries, err := os.ReadDir(share.Path)
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
		}
		if entry.IsDir() {
			item.Size = "folder"
		} else {
			item.Size = formatSize(entryInfo.Size())
			item.URL = fmt.Sprintf("/s/%s/raw?name=%s", share.ID, url.QueryEscape(entry.Name()))
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

	contentType := mime.TypeByExtension(filepath.Ext(downloadName))
	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	} else {
		w.Header().Set("Content-Type", "application/octet-stream")
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, downloadName))
	http.ServeContent(w, r, downloadName, time.Time{}, file)
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

func withShareMessage(shareID, key, value string) string {
	return fmt.Sprintf("/s/%s?%s=%s", shareID, key, url.QueryEscape(value))
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

func newShareID() string {
	var raw [8]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(raw[:])
}
