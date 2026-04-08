package server

import (
	"crypto/subtle"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"mime"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Config struct {
	BindHost string
	Port     int
	Path     string
	Password string
}

type shareServer struct {
	config   Config
	info     os.FileInfo
	template *template.Template
}

type pageData struct {
	Title          string
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

func Run(cfg Config) error {
	info, err := os.Stat(cfg.Path)
	if err != nil {
		return fmt.Errorf("stat target: %w", err)
	}

	srv := &shareServer{
		config:   cfg,
		info:     info,
		template: template.Must(template.New("page").Parse(pageHTML)),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", srv.handleIndex)
	mux.HandleFunc("/raw", srv.handleRaw)
	mux.HandleFunc("/upload", srv.handleUpload)

	addr := fmt.Sprintf("%s:%d", cfg.BindHost, cfg.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", addr, err)
	}

	actualAddress := fmt.Sprintf("http://%s", listener.Addr().String())
	log.Printf("sharing %s", cfg.Path)
	log.Printf("open %s", actualAddress)
	if info.IsDir() {
		if cfg.Password == "" {
			log.Printf("folder mode: read-only")
		} else {
			log.Printf("folder mode: upload enabled with password")
		}
	} else {
		log.Printf("file mode: read-only")
	}

	return http.Serve(listener, mux)
}

func (s *shareServer) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	data := s.basePageData(r)
	data.Items = s.listItems()
	data.ErrorMessage = r.URL.Query().Get("error")
	data.SuccessMessage = r.URL.Query().Get("success")

	if err := s.template.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *shareServer) handleRaw(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.info.IsDir() {
		name := filepath.Base(filepath.Clean(r.URL.Query().Get("name")))
		if name == "." || name == "" {
			http.Error(w, "missing file name", http.StatusBadRequest)
			return
		}

		target := filepath.Join(s.config.Path, name)
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

	serveFileDownload(w, r, s.config.Path, s.info.Name())
}

func (s *shareServer) handleUpload(w http.ResponseWriter, r *http.Request) {
	if !s.info.IsDir() {
		http.Error(w, "uploads are only supported for directories", http.StatusBadRequest)
		return
	}

	if s.config.Password == "" {
		http.Error(w, "uploads are disabled for this share", http.StatusForbidden)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if subtle.ConstantTimeCompare([]byte(r.FormValue("password")), []byte(s.config.Password)) != 1 {
		http.Redirect(w, r, withMessage("error", "密码错误，上传已拒绝"), http.StatusSeeOther)
		return
	}

	if err := r.ParseMultipartForm(64 << 20); err != nil {
		http.Redirect(w, r, withMessage("error", "无法解析上传请求"), http.StatusSeeOther)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Redirect(w, r, withMessage("error", "请选择要上传的文件"), http.StatusSeeOther)
		return
	}
	defer file.Close()

	name := filepath.Base(header.Filename)
	if name == "." || name == "" {
		http.Redirect(w, r, withMessage("error", "无效文件名"), http.StatusSeeOther)
		return
	}

	target := filepath.Join(s.config.Path, name)
	if err := writeUploadedFile(target, file); err != nil {
		http.Redirect(w, r, withMessage("error", "保存上传文件失败："+err.Error()), http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, withMessage("success", "上传成功"), http.StatusSeeOther)
}

func (s *shareServer) basePageData(r *http.Request) pageData {
	address := fmt.Sprintf("http://%s", r.Host)
	return pageData{
		Title:         "Web Share",
		SharedName:    s.info.Name(),
		SharedPath:    s.config.Path,
		IsDir:         s.info.IsDir(),
		UploadEnabled: s.info.IsDir() && s.config.Password != "",
		Address:       address,
	}
}

func (s *shareServer) listItems() []dirItem {
	if !s.info.IsDir() {
		return []dirItem{{
			Name:    s.info.Name(),
			Size:    formatSize(s.info.Size()),
			ModTime: s.info.ModTime().Format("2006-01-02 15:04"),
			URL:     "/raw",
		}}
	}

	entries, err := os.ReadDir(s.config.Path)
	if err != nil {
		return nil
	}

	items := make([]dirItem, 0, len(entries))
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		item := dirItem{
			Name:    entry.Name(),
			ModTime: info.ModTime().Format("2006-01-02 15:04"),
		}

		if entry.IsDir() {
			item.Size = "folder"
		} else {
			item.Size = formatSize(info.Size())
			item.URL = "/raw?name=" + url.QueryEscape(entry.Name())
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

func withMessage(key, value string) string {
	return "/?" + key + "=" + url.QueryEscape(value)
}
