package manager

import (
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
	"strings"
	"sync"
	"time"

	qrcode "github.com/skip2/go-qrcode"
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
	Name string
	Code string
	Type string
	URL  string
}

type sharePageData struct {
	Title          string
	ShareCode      string
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
		templates: template.Must(template.New("pages").Parse(homeHTML + manageHTML + shareHTML)),
		shares:    make(map[string]*Share),
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
	data := sharePageData{
		Title:          "Web Share",
		ShareCode:      share.Code,
		SharedName:     share.Name,
		SharedPath:     share.Path,
		IsDir:          share.IsDir,
		UploadEnabled:  share.IsDir && share.Password != "",
		Address:        fmt.Sprintf("http://%s/s/%s", r.Host, share.Code),
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

	serveFileDownload(w, r, share.Path, filepath.Base(share.Path))
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
		http.Redirect(w, r, withShareMessage(share.Code, "error", "密码错误，上传已拒绝"), http.StatusSeeOther)
		return
	}
	if err := r.ParseMultipartForm(64 << 20); err != nil {
		http.Redirect(w, r, withShareMessage(share.Code, "error", "无法解析上传请求"), http.StatusSeeOther)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Redirect(w, r, withShareMessage(share.Code, "error", "请选择要上传的文件"), http.StatusSeeOther)
		return
	}
	defer file.Close()

	name := filepath.Base(header.Filename)
	if name == "." || name == "" {
		http.Redirect(w, r, withShareMessage(share.Code, "error", "无效文件名"), http.StatusSeeOther)
		return
	}

	target := filepath.Join(share.Path, name)
	if err := writeUploadedFile(target, file); err != nil {
		http.Redirect(w, r, withShareMessage(share.Code, "error", "保存上传文件失败："+err.Error()), http.StatusSeeOther)
		return
	}

	m.touchShare(share.ID)
	http.Redirect(w, r, withShareMessage(share.Code, "success", "上传成功"), http.StatusSeeOther)
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

func listItems(share *Share) []dirItem {
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
			item.URL = fmt.Sprintf("/s/%s/raw?name=%s", share.Code, url.QueryEscape(entry.Name()))
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

func withShareMessage(shareCode, key, value string) string {
	return fmt.Sprintf("/s/%s?%s=%s", shareCode, key, url.QueryEscape(value))
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
