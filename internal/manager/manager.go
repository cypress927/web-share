package manager

import (
	"archive/zip"
	"bytes"
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
	"unicode/utf8"

	qrcode "github.com/skip2/go-qrcode"

	"web-share/internal/shell"
)

const (
	defaultHost            = "0.0.0.0"
	defaultPort            = 21910
	defaultUploadChunkSize = 2 << 20
	cardPreviewTextLimit   = 180
	managePreviewTextLimit = 220
	sharePreviewTextLimit  = 16000

	shareKindFile           = "file"
	shareKindDir            = "dir"
	shareKindClipboardText  = "clipboard_text"
	shareKindClipboardImage = "clipboard_image"
)

type Config struct {
	BindHost            string
	Port                int
	DBPath              string
	ApplySystemLanguage func(lang string) error
}

type Manager struct {
	cfg       Config
	server    *http.Server
	templates *template.Template

	mu          sync.RWMutex
	store       ShareStore
	settings    SettingsStore
	shares      map[string]*Share
	legacyStore ShareStore
	uploads     map[string]*uploadSession
}

type Share struct {
	ID          string
	Code        string
	Kind        string
	Path        string
	Name        string
	IsDir       bool
	Visible     bool
	Password    string
	TextContent string
	BinaryData  []byte
	MimeType    string
	CreatedAt   time.Time
	LastUpdated time.Time
}

type CreateShareRequest struct {
	Kind        string `json:"kind"`
	Path        string `json:"path"`
	Password    string `json:"password"`
	Name        string `json:"name"`
	TextContent string `json:"textContent"`
	MimeType    string `json:"mimeType"`
	ImageBase64 string `json:"imageBase64"`
}

type managePageData struct {
	Title       string
	PublicURL   string
	VisibleURL  string
	Shares      []manageShareCard
	CurrentLang string
	LangZHURL   string
	LangENURL   string
	DefaultLang string
	SetupURL    string
	SystemURL   string
}

type setupPageData struct {
	Title            string
	CurrentLang      string
	LangZHURL        string
	LangENURL        string
	DefaultLang      string
	SetupCompleted   bool
	ManagerRunning   bool
	TrayRunning      bool
	ContextInstalled bool
	AutostartEnabled bool
	ApplySuccess     string
	ApplyError       string
	ManageURL        string
	SystemURL        string
}

type systemPageData struct {
	Title            string
	CurrentLang      string
	LangZHURL        string
	LangENURL        string
	DefaultLang      string
	SetupCompleted   bool
	TrayRunning      bool
	ContextInstalled bool
	AutostartEnabled bool
	ApplySuccess     string
	ApplyError       string
	ManageURL        string
	SetupURL         string
}

type setupStatusPayload struct {
	DefaultLanguage      string `json:"defaultLanguage"`
	SetupCompleted       bool   `json:"setupCompleted"`
	ManagerRunning       bool   `json:"managerRunning"`
	TrayRunning          bool   `json:"trayRunning"`
	ContextMenuInstalled bool   `json:"contextMenuInstalled"`
	AutostartEnabled     bool   `json:"autostartEnabled"`
	ManageURL            string `json:"manageURL"`
	SetupURL             string `json:"setupURL"`
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
	PreviewText    string
	PreviewImage   string
}

type homePageData struct {
	Title         string
	VisibleShares []publicShareCard
	ErrorMessage  string
	CurrentLang   string
	LangZHURL     string
	LangENURL     string
}

type publicShareCard struct {
	Name          string
	Code          string
	Type          string
	URL           string
	Unavailable   bool
	Status        string
	PreviewText   string
	CopyURL       string
	FileName      string
	FileSize      string
	DownloadURL   string
	ContentURL    string
	ShowCopy      bool
	ShowDownload  bool
	ShowThumbnail bool
}

type sharePageData struct {
	Title          string
	ShareCode      string
	SharedName     string
	ShareKind      string
	ShareTypeLabel string
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
	DownloadURL    string
	ContentURL     string
	TextContent    string
	PreviewKind    string
	PreviewText    string
	CurrentLang    string
	LangZHURL      string
	LangENURL      string
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
	dbPath, err := resolveDBPath(cfg.DBPath)
	if err != nil {
		return err
	}
	store, err := openShareStore(dbPath)
	if err != nil {
		return err
	}
	settings, err := openSettingsStore(dbPath)
	if err != nil {
		return err
	}
	_ = ensureSettingsDefaultLanguage(settings)

	mgr := &Manager{
		cfg: cfg,
		templates: template.Must(template.New("pages").Funcs(template.FuncMap{
			"tr": tr,
		}).Parse(homeHTML + manageHTML + shareHTML + setupHTML + systemHTML)),
		store:    store,
		settings: settings,
		uploads:  make(map[string]*uploadSession),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", mgr.handleHome)
	mux.HandleFunc("/api/ping", mgr.handlePing)
	mux.HandleFunc("/api/shutdown", mgr.handleShutdown)
	mux.HandleFunc("/api/shares", mgr.handleCreateShare)
	mux.HandleFunc("/api/shares/", mgr.handleShareAPI)
	mux.HandleFunc("/manage", mgr.handleManage)
	mux.HandleFunc("/manage/settings/system", mgr.handleSystemSettings)
	mux.HandleFunc("/manage/settings/language", mgr.handleManageLanguageSetting)
	mux.HandleFunc("/setup", mgr.handleSetup)
	mux.HandleFunc("/api/setup/status", mgr.handleSetupStatus)
	mux.HandleFunc("/api/setup/apply", mgr.handleSetupApply)
	mux.HandleFunc("/api/system/apply", mgr.handleSystemApply)
	mux.HandleFunc("/manage/shares/", mgr.handleManageShareAction)
	mux.HandleFunc("/s/", mgr.handleShare)

	mgr.server = &http.Server{
		Addr:    fmt.Sprintf("%s:%d", cfg.BindHost, cfg.Port),
		Handler: mux,
	}

	return mgr.server.ListenAndServe()
}

func openShareStore(path string) (ShareStore, error) {
	return newSQLiteShareStore(path)
}

func openSettingsStore(path string) (SettingsStore, error) {
	return newSQLiteSettingsStore(path)
}

func resolveDBPath(path string) (string, error) {
	dbPath := strings.TrimSpace(path)
	if dbPath == "" {
		cacheDir, err := os.UserCacheDir()
		if err != nil {
			return "", err
		}
		dbPath = filepath.Join(cacheDir, "WebShare", "web-share.db")
	}
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return "", err
	}
	return dbPath, nil
}

func (m *Manager) settingsStore() SettingsStore {
	if m.settings != nil {
		return m.settings
	}
	m.settings = newMemorySettingsStore(langEN)
	return m.settings
}

func (m *Manager) currentLanguage(w http.ResponseWriter, r *http.Request) string {
	defaultLang, err := m.settingsStore().GetDefaultLanguage()
	if err != nil {
		defaultLang = langEN
	}
	lang := resolveLanguage(r, defaultLang)
	if queryLang := normalizeLanguage(r.URL.Query().Get("lang")); isSupportedLanguage(queryLang) {
		setLanguageCookie(w, queryLang)
	}
	return lang
}

func (m *Manager) defaultLanguage() string {
	lang, err := m.settingsStore().GetDefaultLanguage()
	if err != nil {
		return langEN
	}
	lang = normalizeLanguage(lang)
	if !isSupportedLanguage(lang) {
		return langEN
	}
	return lang
}

func (m *Manager) languageLinks(r *http.Request) (string, string) {
	return withLanguageInURL(r, langZH), withLanguageInURL(r, langEN)
}

func LocalAPI(path string) string {
	return fmt.Sprintf("http://127.0.0.1:%d%s", defaultPort, path)
}

func LocalManageURL() string {
	return fmt.Sprintf("http://127.0.0.1:%d/manage", defaultPort)
}

func LocalSetupURL() string {
	return fmt.Sprintf("http://127.0.0.1:%d/setup", defaultPort)
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

func (m *Manager) shareStore() ShareStore {
	if m.store != nil {
		return m.store
	}
	if m.legacyStore != nil {
		return m.legacyStore
	}
	m.legacyStore = newMemoryShareStoreFromMap(m.shares)
	return m.legacyStore
}

func (m *Manager) handleHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	currentLang := m.currentLanguage(w, r)
	langZHURL, langENURL := m.languageLinks(r)

	code := strings.TrimSpace(r.URL.Query().Get("code"))
	if code != "" {
		share := m.getShareByCode(code)
		if share == nil {
			data := homePageData{
				Title:         tr(currentLang, "site.brand"),
				VisibleShares: m.listVisibleShares(currentLang),
				ErrorMessage:  tr(currentLang, "home.code_not_found"),
				CurrentLang:   currentLang,
				LangZHURL:     langZHURL,
				LangENURL:     langENURL,
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
		Title:         tr(currentLang, "site.brand"),
		VisibleShares: m.listVisibleShares(currentLang),
		CurrentLang:   currentLang,
		LangZHURL:     langZHURL,
		LangENURL:     langENURL,
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
	go m.shutdownProgram()
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
	if err := json.NewDecoder(io.LimitReader(r.Body, 20<<20)).Decode(&req); err != nil {
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
	currentLang := m.currentLanguage(w, r)
	langZHURL, langENURL := m.languageLinks(r)
	defaultLang, _ := m.settingsStore().GetDefaultLanguage()

	host := "127.0.0.1"
	if r.Host != "" {
		host = r.Host
	}

	data := managePageData{
		Title:       "Web Share Manager",
		PublicURL:   fmt.Sprintf("http://%s/", host),
		VisibleURL:  fmt.Sprintf("http://%s/", host),
		Shares:      m.listManageCards(currentLang),
		CurrentLang: currentLang,
		LangZHURL:   langZHURL,
		LangENURL:   langENURL,
		DefaultLang: defaultLang,
		SetupURL:    "/setup?lang=" + url.QueryEscape(currentLang),
		SystemURL:   "/manage/settings/system?lang=" + url.QueryEscape(currentLang),
	}

	if err := m.templates.ExecuteTemplate(w, "manage", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (m *Manager) handleSetup(w http.ResponseWriter, r *http.Request) {
	if !isLocalRequest(r.RemoteAddr) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	currentLang := m.currentLanguage(w, r)
	langZHURL, langENURL := m.languageLinks(r)

	status := m.setupStatus(currentLang)
	data := setupPageData{
		Title:            tr(currentLang, "setup.title"),
		CurrentLang:      currentLang,
		LangZHURL:        langZHURL,
		LangENURL:        langENURL,
		DefaultLang:      m.defaultLanguage(),
		SetupCompleted:   status.SetupCompleted,
		ManagerRunning:   status.ManagerRunning,
		TrayRunning:      status.TrayRunning,
		ContextInstalled: status.ContextMenuInstalled,
		AutostartEnabled: status.AutostartEnabled,
		ApplySuccess:     r.URL.Query().Get("success"),
		ApplyError:       r.URL.Query().Get("error"),
		ManageURL:        "/manage?lang=" + url.QueryEscape(currentLang),
		SystemURL:        "/manage/settings/system?lang=" + url.QueryEscape(currentLang),
	}
	if err := m.templates.ExecuteTemplate(w, "setup", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (m *Manager) handleSetupStatus(w http.ResponseWriter, r *http.Request) {
	if !isLocalRequest(r.RemoteAddr) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	currentLang := m.currentLanguage(w, r)
	m.writeJSON(w, http.StatusOK, m.setupStatus(currentLang))
}

func (m *Manager) handleSetupApply(w http.ResponseWriter, r *http.Request) {
	if !isLocalRequest(r.RemoteAddr) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	lang := normalizeLanguage(r.FormValue("default_lang"))
	if !isSupportedLanguage(lang) {
		lang = m.defaultLanguage()
	}
	if err := m.settingsStore().SetDefaultLanguage(lang); err != nil {
		if wantsJSON(r) {
			m.writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": tr(lang, "setup.apply_failed")})
			return
		}
		http.Redirect(w, r, "/setup?lang="+url.QueryEscape(lang)+"&error="+url.QueryEscape(tr(lang, "setup.apply_failed")), http.StatusSeeOther)
		return
	}
	if r.FormValue("complete_setup") == "on" {
		if err := m.settingsStore().SetSetupCompleted(true); err != nil {
			if wantsJSON(r) {
				m.writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": tr(lang, "setup.apply_failed")})
				return
			}
			http.Redirect(w, r, "/setup?lang="+url.QueryEscape(lang)+"&error="+url.QueryEscape(tr(lang, "setup.apply_failed")), http.StatusSeeOther)
			return
		}
	}
	if m.cfg.ApplySystemLanguage != nil {
		if err := m.cfg.ApplySystemLanguage(lang); err != nil {
			if wantsJSON(r) {
				m.writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": tr(lang, "setup.apply_failed")})
				return
			}
			http.Redirect(w, r, "/setup?lang="+url.QueryEscape(lang)+"&error="+url.QueryEscape(tr(lang, "setup.apply_failed")), http.StatusSeeOther)
			return
		}
	}
	if r.FormValue("install_context_menu") == "on" {
		exePath, err := os.Executable()
		if err != nil || shell.InstallContextMenuWithLanguage(exePath, lang) != nil {
			if wantsJSON(r) {
				m.writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": tr(lang, "setup.apply_failed")})
				return
			}
			http.Redirect(w, r, "/setup?lang="+url.QueryEscape(lang)+"&error="+url.QueryEscape(tr(lang, "setup.apply_failed")), http.StatusSeeOther)
			return
		}
	}
	if r.FormValue("enable_autostart") == "on" {
		exePath, err := os.Executable()
		if err != nil || installStartupTask(exePath, lang) != nil {
			if wantsJSON(r) {
				m.writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": tr(lang, "setup.apply_failed")})
				return
			}
			http.Redirect(w, r, "/setup?lang="+url.QueryEscape(lang)+"&error="+url.QueryEscape(tr(lang, "setup.apply_failed")), http.StatusSeeOther)
			return
		}
	}
	if r.FormValue("start_tray") == "on" {
		exePath, err := os.Executable()
		if err != nil || startTrayProcess(exePath) != nil {
			if wantsJSON(r) {
				m.writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": tr(lang, "setup.apply_failed")})
				return
			}
			http.Redirect(w, r, "/setup?lang="+url.QueryEscape(lang)+"&error="+url.QueryEscape(tr(lang, "setup.apply_failed")), http.StatusSeeOther)
			return
		}
	}
	setLanguageCookie(w, lang)
	if wantsJSON(r) {
		m.writeJSON(w, http.StatusOK, map[string]any{
			"ok":      true,
			"message": tr(lang, "setup.apply_ok"),
			"status":  m.setupStatus(lang),
		})
		return
	}
	http.Redirect(w, r, "/setup?lang="+url.QueryEscape(lang)+"&success="+url.QueryEscape(tr(lang, "setup.apply_ok")), http.StatusSeeOther)
}

func (m *Manager) handleSystemSettings(w http.ResponseWriter, r *http.Request) {
	if !isLocalRequest(r.RemoteAddr) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	currentLang := m.currentLanguage(w, r)
	langZHURL, langENURL := m.languageLinks(r)
	status := m.setupStatus(currentLang)
	data := systemPageData{
		Title:            tr(currentLang, "system.title"),
		CurrentLang:      currentLang,
		LangZHURL:        langZHURL,
		LangENURL:        langENURL,
		DefaultLang:      status.DefaultLanguage,
		SetupCompleted:   status.SetupCompleted,
		TrayRunning:      status.TrayRunning,
		ContextInstalled: status.ContextMenuInstalled,
		AutostartEnabled: status.AutostartEnabled,
		ApplySuccess:     r.URL.Query().Get("success"),
		ApplyError:       r.URL.Query().Get("error"),
		ManageURL:        "/manage?lang=" + url.QueryEscape(currentLang),
		SetupURL:         "/setup?lang=" + url.QueryEscape(currentLang),
	}
	if err := m.templates.ExecuteTemplate(w, "system", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (m *Manager) handleSystemApply(w http.ResponseWriter, r *http.Request) {
	if !isLocalRequest(r.RemoteAddr) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	lang := normalizeLanguage(r.FormValue("default_lang"))
	if !isSupportedLanguage(lang) {
		lang = m.defaultLanguage()
	}
	if err := m.settingsStore().SetDefaultLanguage(lang); err != nil {
		if wantsJSON(r) {
			m.writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": tr(lang, "system.apply_failed")})
			return
		}
		http.Redirect(w, r, "/manage/settings/system?lang="+url.QueryEscape(lang)+"&error="+url.QueryEscape(tr(lang, "system.apply_failed")), http.StatusSeeOther)
		return
	}
	action := strings.TrimSpace(r.FormValue("action"))
	if err := m.applySystemAction(lang, action); err != nil {
		if wantsJSON(r) {
			m.writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": tr(lang, "system.apply_failed")})
			return
		}
		http.Redirect(w, r, "/manage/settings/system?lang="+url.QueryEscape(lang)+"&error="+url.QueryEscape(tr(lang, "system.apply_failed")), http.StatusSeeOther)
		return
	}
	setLanguageCookie(w, lang)
	if wantsJSON(r) {
		m.writeJSON(w, http.StatusOK, map[string]any{
			"ok":      true,
			"message": tr(lang, "system.apply_ok"),
			"status":  m.setupStatus(lang),
		})
		return
	}
	http.Redirect(w, r, "/manage/settings/system?lang="+url.QueryEscape(lang)+"&success="+url.QueryEscape(tr(lang, "system.apply_ok")), http.StatusSeeOther)
}

func (m *Manager) setupStatus(lang string) setupStatusPayload {
	setupCompleted, _ := m.settingsStore().GetSetupCompleted()
	trayRunning, _ := shell.TrayRunning()
	autostartEnabled, _ := shell.CurrentUserRunExists("WebShare.AutoStart")
	return setupStatusPayload{
		DefaultLanguage:      m.defaultLanguage(),
		SetupCompleted:       setupCompleted,
		ManagerRunning:       true,
		TrayRunning:          trayRunning,
		ContextMenuInstalled: shell.ContextMenuInstalled(),
		AutostartEnabled:     autostartEnabled,
		ManageURL:            withLanguageInURL(&http.Request{URL: &url.URL{Path: "/manage"}}, lang),
		SetupURL:             withLanguageInURL(&http.Request{URL: &url.URL{Path: "/setup"}}, lang),
	}
}

func installStartupTask(exePath, lang string) error {
	action := shell.QuoteCommand(exePath, "start", "-lang", lang, "-notify-start=true")
	return shell.SetCurrentUserRun("WebShare.AutoStart", action)
}

func startTrayProcess(exePath string) error {
	running, err := shell.TrayRunning()
	if err == nil && running {
		return nil
	}
	return shell.StartDetached(exePath, "tray")
}

func uninstallStartupTask(taskName string) error {
	return shell.DeleteCurrentUserRun(taskName)
}

func (m *Manager) applySystemAction(lang, action string) error {
	exePath, err := os.Executable()
	if err != nil {
		return err
	}
	switch action {
	case "", "save_language":
		if m.cfg.ApplySystemLanguage != nil {
			return m.cfg.ApplySystemLanguage(lang)
		}
		return nil
	case "install_context":
		return shell.InstallContextMenuWithLanguage(exePath, lang)
	case "remove_context":
		return shell.UninstallContextMenu()
	case "enable_autostart":
		return installStartupTask(exePath, lang)
	case "disable_autostart":
		return uninstallStartupTask("WebShare.AutoStart")
	case "start_tray":
		return startTrayProcess(exePath)
	case "stop_tray":
		return shell.StopTray()
	case "stop_program":
		go m.shutdownProgram()
		return nil
	case "mark_setup_done":
		return m.settingsStore().SetSetupCompleted(true)
	case "mark_setup_todo":
		return m.settingsStore().SetSetupCompleted(false)
	default:
		return errors.New("unsupported action")
	}
}

func (m *Manager) shutdownProgram() {
	_ = shell.StopTray()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = m.server.Shutdown(ctx)
}

func wantsJSON(r *http.Request) bool {
	return strings.Contains(strings.ToLower(r.Header.Get("Accept")), "application/json") ||
		strings.EqualFold(r.Header.Get("X-Requested-With"), "fetch")
}

func (m *Manager) handleManageLanguageSetting(w http.ResponseWriter, r *http.Request) {
	if !isLocalRequest(r.RemoteAddr) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	lang := normalizeLanguage(r.FormValue("default_lang"))
	if !isSupportedLanguage(lang) {
		http.Error(w, "unsupported language", http.StatusBadRequest)
		return
	}
	if err := m.settingsStore().SetDefaultLanguage(lang); err != nil {
		http.Error(w, "save language failed", http.StatusInternalServerError)
		return
	}
	if m.cfg.ApplySystemLanguage != nil {
		if err := m.cfg.ApplySystemLanguage(lang); err != nil {
			http.Error(w, "apply system language failed", http.StatusInternalServerError)
			return
		}
	}
	setLanguageCookie(w, lang)
	http.Redirect(w, r, "/manage?lang="+url.QueryEscape(lang), http.StatusSeeOther)
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
	case len(parts) == 2 && parts[1] == "text" && r.Method == http.MethodGet:
		m.serveShareText(w, r, share)
	case len(parts) == 2 && parts[1] == "content" && (r.Method == http.MethodGet || r.Method == http.MethodHead):
		m.serveShareContent(w, r, share)
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
	kind := strings.TrimSpace(req.Kind)
	if kind == "" {
		kind = shareKindFile
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	store := m.shareStore()

	switch kind {
	case shareKindClipboardText:
		content := req.TextContent
		if strings.TrimSpace(content) == "" {
			return nil, errors.New("clipboard text is empty")
		}

		name := strings.TrimSpace(req.Name)
		if name == "" {
			name = tr(m.defaultLanguage(), "share.default_clipboard_text_name")
		}
		name = m.allocateUniqueName(name, "")

		share := &Share{
			ID:          newShareID(),
			Code:        m.allocateUniqueCode(),
			Kind:        shareKindClipboardText,
			Name:        name,
			Visible:     false,
			TextContent: content,
			MimeType:    "text/plain; charset=utf-8",
			CreatedAt:   time.Now(),
			LastUpdated: time.Now(),
		}
		if err := store.Create(share); err != nil {
			return nil, err
		}
		return copyShare(share), nil

	case shareKindClipboardImage:
		raw := strings.TrimSpace(req.ImageBase64)
		if raw == "" {
			return nil, errors.New("clipboard image is empty")
		}
		imageData, err := base64.StdEncoding.DecodeString(raw)
		if err != nil {
			return nil, errors.New("invalid clipboard image payload")
		}
		if len(imageData) == 0 {
			return nil, errors.New("clipboard image is empty")
		}

		mimeType := strings.TrimSpace(req.MimeType)
		if mimeType == "" {
			mimeType = "image/png"
		}

		name := strings.TrimSpace(req.Name)
		if name == "" {
			name = tr(m.defaultLanguage(), "share.default_clipboard_image_name")
		}
		name = m.allocateUniqueName(name, "")

		share := &Share{
			ID:          newShareID(),
			Code:        m.allocateUniqueCode(),
			Kind:        shareKindClipboardImage,
			Name:        name,
			Visible:     false,
			BinaryData:  imageData,
			MimeType:    mimeType,
			CreatedAt:   time.Now(),
			LastUpdated: time.Now(),
		}
		if err := store.Create(share); err != nil {
			return nil, err
		}
		return copyShare(share), nil
	}

	if strings.TrimSpace(req.Path) == "" {
		return nil, errors.New("path is required")
	}

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
		kind = shareKindFile
	} else {
		kind = shareKindDir
	}

	name := m.allocateUniqueName(info.Name(), "")
	share := &Share{
		ID:          newShareID(),
		Code:        m.allocateUniqueCode(),
		Kind:        kind,
		Path:        target,
		Name:        name,
		IsDir:       info.IsDir(),
		Visible:     false,
		Password:    req.Password,
		CreatedAt:   time.Now(),
		LastUpdated: time.Now(),
	}

	if err := store.Create(share); err != nil {
		return nil, err
	}
	return copyShare(share), nil
}

func (m *Manager) updateShare(id, name string, visible bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	store := m.shareStore()

	share, err := store.GetByID(id)
	if err != nil {
		return errors.New("share not found")
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("name cannot be empty")
	}

	share.Name = m.allocateUniqueName(name, id)
	share.Visible = visible
	share.LastUpdated = time.Now()
	return store.Update(share)
}

func (m *Manager) deleteShare(id string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	ok, err := m.shareStore().DeleteByID(id)
	if err != nil {
		return false
	}
	return ok
}

func (m *Manager) getShareByCode(code string) *Share {
	share, err := m.shareStore().GetByCode(code)
	if err != nil {
		return nil
	}
	return share
}

func (m *Manager) listVisibleShares(lang string) []publicShareCard {
	allShares, err := m.shareStore().List()
	if err != nil {
		return nil
	}

	shares := make([]publicShareCard, 0, len(allShares))
	for _, share := range allShares {
		if !share.Visible {
			continue
		}

		card := publicShareCard{
			Name: share.Name,
			Code: share.Code,
			URL:  "/s/" + share.Code,
		}
		card.Type = shareTypeLabel(share, lang)
		if isPathBackedShare(share) {
			info, err := os.Stat(share.Path)
			if err != nil {
				card.Unavailable = true
				card.Status = tr(lang, "status.unavailable")
			} else {
				card.Status = tr(lang, "status.available")
				if !share.IsDir {
					card.FileName = filepath.Base(share.Path)
					card.FileSize = formatSize(info.Size())
				}
			}
		} else {
			card.Status = tr(lang, "status.available")
		}
		kind, previewText := buildSharePreview(share, cardPreviewTextLimit)
		if kind == "text" {
			card.PreviewText = previewText
			card.CopyURL = fmt.Sprintf("/s/%s/text", share.Code)
			card.ShowCopy = true
		} else if kind == "image" {
			card.ContentURL = fmt.Sprintf("/s/%s/content", share.Code)
			card.ShowThumbnail = true
		}
		if !share.IsDir {
			card.DownloadURL = fmt.Sprintf("/s/%s/raw", share.Code)
			card.ShowDownload = true
		}
		if isPathBackedShare(share) && card.Status == "" {
			if _, err := os.Stat(share.Path); err != nil {
				card.Unavailable = true
				card.Status = tr(lang, "status.unavailable")
			} else {
				card.Status = tr(lang, "status.available")
			}
		}
		shares = append(shares, card)
	}

	sort.Slice(shares, func(i, j int) bool {
		return strings.ToLower(shares[i].Name) < strings.ToLower(shares[j].Name)
	})
	return shares
}

func truncateText(input string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	runes := []rune(strings.TrimSpace(input))
	if len(runes) <= maxRunes {
		return string(runes)
	}
	return string(runes[:maxRunes]) + "..."
}

func (m *Manager) listManageCards(lang string) []manageShareCard {
	allShares, err := m.shareStore().List()
	if err != nil {
		return nil
	}

	cards := make([]manageShareCard, 0, len(allShares))
	localIPs := listLocalIPv4s()
	for _, share := range allShares {
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
		if !isPathBackedShare(share) {
			card.Path = tr(lang, "manage.path_clipboard_snapshot")
		}
		kind, previewText := buildSharePreview(share, managePreviewTextLimit)
		if kind == "text" {
			card.PreviewText = previewText
		} else if kind == "image" {
			card.PreviewImage = fmt.Sprintf("/s/%s/content", share.Code)
		}
		card.Type = shareTypeLabel(share, lang)
		if share.IsDir && share.Password != "" {
			card.Mode = tr(lang, "manage.mode_upload_enabled")
		} else {
			card.Mode = tr(lang, "manage.mode_readonly")
		}
		if share.Visible {
			card.Visibility = tr(lang, "manage.visibility_public")
		} else {
			card.Visibility = tr(lang, "manage.visibility_hidden")
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
	currentLang := m.currentLanguage(w, r)
	langZHURL, langENURL := m.languageLinks(r)

	switch share.Kind {
	case shareKindClipboardText:
		data := sharePageData{
			Title:          tr(currentLang, "site.brand"),
			ShareCode:      share.Code,
			SharedName:     share.Name,
			ShareKind:      share.Kind,
			ShareTypeLabel: shareTypeLabel(share, currentLang),
			IsDir:          false,
			UploadEnabled:  false,
			Address:        fmt.Sprintf("http://%s%s", r.Host, browseURL(share.Code, "")),
			ErrorMessage:   r.URL.Query().Get("error"),
			SuccessMessage: r.URL.Query().Get("success"),
			TextContent:    share.TextContent,
			DownloadURL:    fmt.Sprintf("/s/%s/raw", share.Code),
			ChunkSize:      defaultUploadChunkSize,
			CurrentLang:    currentLang,
			LangZHURL:      langZHURL,
			LangENURL:      langENURL,
		}
		if err := m.templates.ExecuteTemplate(w, "share", data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	case shareKindClipboardImage:
		data := sharePageData{
			Title:          tr(currentLang, "site.brand"),
			ShareCode:      share.Code,
			SharedName:     share.Name,
			ShareKind:      share.Kind,
			ShareTypeLabel: shareTypeLabel(share, currentLang),
			IsDir:          false,
			UploadEnabled:  false,
			Address:        fmt.Sprintf("http://%s%s", r.Host, browseURL(share.Code, "")),
			ErrorMessage:   r.URL.Query().Get("error"),
			SuccessMessage: r.URL.Query().Get("success"),
			DownloadURL:    fmt.Sprintf("/s/%s/raw", share.Code),
			ContentURL:     fmt.Sprintf("/s/%s/content", share.Code),
			ChunkSize:      defaultUploadChunkSize,
			CurrentLang:    currentLang,
			LangZHURL:      langZHURL,
			LangENURL:      langENURL,
		}
		if err := m.templates.ExecuteTemplate(w, "share", data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	rootInfo, err := os.Stat(share.Path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			data := sharePageData{
				Title:          tr(currentLang, "site.brand"),
				ShareCode:      share.Code,
				SharedName:     share.Name,
				ShareKind:      share.Kind,
				ShareTypeLabel: shareTypeLabel(share, currentLang),
				SharedPath:     share.Path,
				IsDir:          share.IsDir,
				Address:        fmt.Sprintf("http://%s%s", r.Host, browseURL(share.Code, "")),
				ChunkSize:      defaultUploadChunkSize,
				Unavailable:    true,
				ErrorMessage:   tr(currentLang, "share.error_root_missing"),
				CurrentLang:    currentLang,
				LangZHURL:      langZHURL,
				LangENURL:      langENURL,
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
	if !share.IsDir {
		kind, previewText := buildSharePreview(share, sharePreviewTextLimit)
		data := sharePageData{
			Title:          tr(currentLang, "site.brand"),
			ShareCode:      share.Code,
			SharedName:     share.Name,
			ShareKind:      share.Kind,
			ShareTypeLabel: shareTypeLabel(share, currentLang),
			SharedPath:     share.Path,
			IsDir:          false,
			UploadEnabled:  false,
			Address:        fmt.Sprintf("http://%s%s", r.Host, browseURL(share.Code, "")),
			ErrorMessage:   r.URL.Query().Get("error"),
			SuccessMessage: r.URL.Query().Get("success"),
			DownloadURL:    fmt.Sprintf("/s/%s/raw", share.Code),
			ContentURL:     fmt.Sprintf("/s/%s/content", share.Code),
			PreviewKind:    kind,
			PreviewText:    previewText,
			ChunkSize:      defaultUploadChunkSize,
			CurrentLang:    currentLang,
			LangZHURL:      langZHURL,
			LangENURL:      langENURL,
		}
		if err := m.templates.ExecuteTemplate(w, "share", data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
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
			http.Redirect(w, r, withShareMessageAt(share.Code, parentPathOrRoot(currentPath), "error", tr(currentLang, "share.error_current_dir_missing")), http.StatusSeeOther)
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
		Title:          tr(currentLang, "site.brand"),
		ShareCode:      share.Code,
		SharedName:     share.Name,
		ShareKind:      share.Kind,
		ShareTypeLabel: shareTypeLabel(share, currentLang),
		SharedPath:     share.Path,
		CurrentPath:    currentPath,
		CurrentLabel:   currentPathLabel(currentPath, currentLang),
		ParentURL:      parentBrowseURL(share.Code, currentPath),
		Breadcrumbs:    buildBreadcrumbs(share.Code, currentPath, currentLang),
		IsDir:          share.IsDir,
		UploadEnabled:  share.IsDir && share.Password != "",
		Address:        fmt.Sprintf("http://%s%s", r.Host, browseURL(share.Code, currentPath)),
		ErrorMessage:   r.URL.Query().Get("error"),
		SuccessMessage: r.URL.Query().Get("success"),
		Items:          listItems(share, currentPath, currentDir, currentLang),
		ChunkSize:      defaultUploadChunkSize,
		CurrentLang:    currentLang,
		LangZHURL:      langZHURL,
		LangENURL:      langENURL,
	}

	if err := m.templates.ExecuteTemplate(w, "share", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (m *Manager) serveShareRaw(w http.ResponseWriter, r *http.Request, share *Share) {
	switch share.Kind {
	case shareKindClipboardText:
		name := sanitizeArchiveName(share.Name) + ".txt"
		serveBytesDownload(w, r, []byte(share.TextContent), name, "text/plain; charset=utf-8")
		return
	case shareKindClipboardImage:
		ext := extByMimeType(share.MimeType)
		if ext == "" {
			ext = ".png"
		}
		name := sanitizeArchiveName(share.Name) + ext
		serveBytesDownload(w, r, share.BinaryData, name, share.MimeType)
		return
	}

	if share.IsDir {
		relativePath, target, err := resolveShareSubpath(share.Path, r.URL.Query().Get("path"), false)
		if err != nil {
			http.Error(w, "invalid file path", http.StatusBadRequest)
			return
		}
		info, err := os.Stat(target)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				lang := m.currentLanguage(w, r)
				missingText := tr(lang, "share.error_file_missing")
				if maybeRedirectToShareError(w, r, share.Code, parentPathOrRoot(relativePath), missingText) {
					return
				}
				http.Error(w, missingText, http.StatusNotFound)
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
			lang := m.currentLanguage(w, r)
			missingText := tr(lang, "share.error_file_missing")
			if maybeRedirectToShareError(w, r, share.Code, "", missingText) {
				return
			}
			http.Error(w, missingText, http.StatusNotFound)
			return
		}
		http.Error(w, "failed to inspect file", http.StatusInternalServerError)
		return
	}

	serveFileDownload(w, r, share.Path, filepath.Base(share.Path))
}

func (m *Manager) serveShareContent(w http.ResponseWriter, r *http.Request, share *Share) {
	switch share.Kind {
	case shareKindClipboardImage:
		if len(share.BinaryData) == 0 {
			http.Error(w, "clipboard image is empty", http.StatusNotFound)
			return
		}
		contentType := strings.TrimSpace(share.MimeType)
		if contentType == "" {
			contentType = "image/png"
		}
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Cache-Control", "no-store")
		http.ServeContent(w, r, share.Name, time.Time{}, bytes.NewReader(share.BinaryData))
		return
	default:
		if share.IsDir || !isImageExtension(filepath.Ext(share.Path)) {
			http.NotFound(w, r)
			return
		}
		file, err := os.Open(share.Path)
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
		contentType := mime.TypeByExtension(strings.ToLower(filepath.Ext(share.Path)))
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Cache-Control", "no-store")
		http.ServeContent(w, r, info.Name(), info.ModTime(), file)
	}
}

func (m *Manager) serveShareText(w http.ResponseWriter, r *http.Request, share *Share) {
	switch share.Kind {
	case shareKindClipboardText:
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = io.WriteString(w, share.TextContent)
		return
	case shareKindClipboardImage:
		http.NotFound(w, r)
		return
	}

	if share.IsDir || !isTextExtension(filepath.Ext(share.Path)) {
		http.NotFound(w, r)
		return
	}

	raw, err := os.ReadFile(share.Path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "failed to read file", http.StatusInternalServerError)
		return
	}
	if !utf8.Valid(raw) {
		raw = bytes.ToValidUTF8(raw, []byte("?"))
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(raw)
}

func buildSharePreview(share *Share, maxRunes int) (string, string) {
	switch share.Kind {
	case shareKindClipboardText:
		return "text", truncateText(share.TextContent, maxRunes)
	case shareKindClipboardImage:
		return "image", ""
	}
	if share.IsDir || share.Path == "" {
		return "", ""
	}
	if isImageExtension(filepath.Ext(share.Path)) {
		return "image", ""
	}
	if !isTextExtension(filepath.Ext(share.Path)) {
		return "", ""
	}
	text, ok := readTextPreview(share.Path, 64<<10)
	if !ok {
		return "", ""
	}
	return "text", truncateText(text, maxRunes)
}

func readTextPreview(path string, limit int64) (string, bool) {
	file, err := os.Open(path)
	if err != nil {
		return "", false
	}
	defer file.Close()

	if limit <= 0 {
		limit = 64 << 10
	}

	raw, err := io.ReadAll(io.LimitReader(file, limit))
	if err != nil || len(raw) == 0 {
		return "", false
	}
	if bytes.IndexByte(raw, 0) >= 0 {
		return "", false
	}
	if !utf8.Valid(raw) {
		raw = bytes.ToValidUTF8(raw, []byte("?"))
	}
	text := strings.TrimSpace(strings.ReplaceAll(string(raw), "\r\n", "\n"))
	if text == "" {
		return "", false
	}
	return text, true
}

func isImageExtension(ext string) bool {
	switch strings.ToLower(strings.TrimSpace(ext)) {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".bmp":
		return true
	default:
		return false
	}
}

func isTextExtension(ext string) bool {
	switch strings.ToLower(strings.TrimSpace(ext)) {
	case ".txt", ".md", ".json", ".yaml", ".yml", ".toml", ".ini", ".log", ".csv", ".xml", ".html", ".htm", ".css", ".js", ".ts", ".tsx", ".jsx", ".go", ".py", ".java", ".c", ".h", ".cpp", ".hpp", ".rs", ".sh", ".ps1", ".bat":
		return true
	default:
		return false
	}
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
			lang := m.currentLanguage(w, r)
			http.Redirect(w, r, withShareMessageAt(share.Code, parentPathOrRoot(relativePath), "error", tr(lang, "share.error_archive_dir_missing")), http.StatusSeeOther)
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
			return nil, errors.New("share root folder no longer exists")
		}
		return nil, fmt.Errorf("inspect share root failed: %w", err)
	} else if !info.IsDir() {
		return nil, errors.New("share root is not a directory")
	}

	currentPath, currentDir, err := resolveShareSubpath(share.Path, strings.TrimSpace(req.Path), true)
	if err != nil {
		return nil, errors.New("invalid directory path")
	}
	if info, err := os.Stat(currentDir); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, errors.New("current directory no longer exists")
		}
		return nil, fmt.Errorf("inspect current directory failed: %w", err)
	} else if !info.IsDir() {
		return nil, errors.New("current directory is not valid")
	}

	relativePath, parentDir, target, name, err := resolveUploadTargetPath(share.Path, currentPath, req.FilePath)
	if err != nil {
		return nil, err
	}
	if req.FileSize < 0 {
		return nil, errors.New("invalid file size")
	}
	if req.ChunkSize <= 0 {
		return nil, errors.New("invalid chunk size")
	}
	if req.TotalChunks <= 0 {
		return nil, errors.New("invalid chunk count")
	}

	expectedChunks := 1
	if req.FileSize > 0 {
		expectedChunks = int((req.FileSize + req.ChunkSize - 1) / req.ChunkSize)
	}
	if expectedChunks != req.TotalChunks {
		return nil, errors.New("chunk count does not match file size")
	}

	if err := os.MkdirAll(parentDir, 0o755); err != nil {
		return nil, fmt.Errorf("create target directory failed: %w", err)
	}

	if _, err := os.Stat(target); err == nil {
		return nil, errors.New("target file already exists")
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("inspect target file failed: %w", err)
	}

	key := uploadSessionKey(share.ID, relativePath)

	m.mu.Lock()
	if existing, ok := m.uploads[key]; ok {
		m.mu.Unlock()
		existing.mu.Lock()
		defer existing.mu.Unlock()
		if existing.TotalSize != req.FileSize || existing.ChunkSize != req.ChunkSize || existing.TotalChunks != req.TotalChunks {
			return nil, errors.New("a different upload session already exists for this file")
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
			return nil, errors.New("invalid upload temp file state")
		}
		if info.Size() < session.TotalSize && info.Size()%session.ChunkSize != 0 {
			m.mu.Unlock()
			return nil, errors.New("upload temp file is not chunk-aligned")
		}
		session.UploadedBytes = info.Size()
		session.NextIndex = int(info.Size() / session.ChunkSize)
	} else if !errors.Is(err, os.ErrNotExist) {
		m.mu.Unlock()
		return nil, fmt.Errorf("inspect upload temp file failed: %w", err)
	}

	if session.TotalSize == 0 && session.UploadedBytes == 0 {
		file, err := os.OpenFile(session.TargetPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if err != nil {
			m.mu.Unlock()
			return nil, fmt.Errorf("create empty file failed: %w", err)
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
	share, err := m.shareStore().GetByID(id)
	if err != nil {
		return
	}
	share.LastUpdated = time.Now()
	_ = m.shareStore().Update(share)
}

func (m *Manager) allocateUniqueName(baseName, ignoreID string) string {
	baseName = strings.TrimSpace(baseName)
	if baseName == "" {
		baseName = tr(m.defaultLanguage(), "share.default_name")
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
	shares, err := m.shareStore().List()
	if err != nil {
		return false
	}
	for _, share := range shares {
		id := share.ID
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
	shares, err := m.shareStore().List()
	if err != nil {
		return false
	}
	for _, share := range shares {
		if strings.EqualFold(share.Code, code) {
			return true
		}
	}
	return false
}

func listItems(share *Share, currentPath, currentDir, lang string) []dirItem {
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
			item.Size = tr(lang, "share.folder_size_label")
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

func serveBytesDownload(w http.ResponseWriter, r *http.Request, content []byte, downloadName, contentType string) {
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, downloadName))
	http.ServeContent(w, r, downloadName, time.Time{}, bytes.NewReader(content))
}

func shareTypeLabel(share *Share, lang string) string {
	switch share.Kind {
	case shareKindDir:
		return tr(lang, "share.type.dir")
	case shareKindClipboardText:
		return tr(lang, "share.type.clipboard_text")
	case shareKindClipboardImage:
		return tr(lang, "share.type.clipboard_image")
	default:
		if share.IsDir {
			return tr(lang, "share.type.dir")
		}
		return tr(lang, "share.type.file")
	}
}

func isPathBackedShare(share *Share) bool {
	return share.Kind == shareKindFile || share.Kind == shareKindDir || share.Kind == ""
}

func extByMimeType(mimeType string) string {
	switch strings.TrimSpace(strings.ToLower(mimeType)) {
	case "image/png":
		return ".png"
	case "image/jpeg", "image/jpg":
		return ".jpg"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	default:
		return ""
	}
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

func buildBreadcrumbs(shareCode, currentPath, lang string) []breadcrumbItem {
	items := []breadcrumbItem{{
		Name: tr(lang, "share.root_dir"),
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

func currentPathLabel(currentPath, lang string) string {
	if currentPath == "" {
		return tr(lang, "share.root_dir")
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
		return "", "", "", "", errors.New("invalid file name")
	}

	clean := filepath.Clean(filepath.FromSlash(requested))
	if clean == "." || filepath.IsAbs(clean) || filepath.VolumeName(clean) != "" {
		return "", "", "", "", errors.New("invalid file name")
	}

	name := filepath.Base(clean)
	if name == "." || name == "" {
		return "", "", "", "", errors.New("invalid file name")
	}

	parentRequest := currentPath
	if dir := filepath.Dir(clean); dir != "." && dir != "" {
		parentRequest = joinRelativePath(currentPath, filepath.ToSlash(dir))
	}

	parentRelative, parentDir, err := resolveShareSubpath(root, parentRequest, true)
	if err != nil {
		return "", "", "", "", errors.New("invalid directory path")
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
	if share.BinaryData != nil {
		copyValue.BinaryData = append([]byte(nil), share.BinaryData...)
	}
	return &copyValue
}
