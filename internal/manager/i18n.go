package manager

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
)

const (
	langZH = "zh-CN"
	langEN = "en-US"
)

var supportedLanguages = map[string]struct{}{
	langZH: {},
	langEN: {},
}

var i18n = map[string]map[string]string{
	langZH: {
		"site.brand":                         "Web Share",
		"home.title":                         "可访问的分享",
		"home.subtitle":                      "首页只展示分享者设置为可见的内容。你也可以直接输入分享码进入指定分享。",
		"home.input_code":                    "输入分享码",
		"home.code_placeholder":              "例如 a7k2m3",
		"home.open_share":                    "打开分享",
		"home.visible":                       "首页可见的分享",
		"home.empty":                         "当前没有可见的分享。",
		"home.open":                          "打开分享",
		"home.copy":                          "一键复制",
		"home.download":                      "一键下载",
		"home.share_code":                    "分享码",
		"home.code_not_found":                "分享码不存在",
		"home.file_meta_prefix":              "文件",
		"home.copy_failed":                   "复制失败，请稍后重试。",
		"manage.title":                       "正在共享的内容",
		"manage.subtitle":                    "你可以为分享设置短码访问入口、修改显示名称，并决定它是否出现在公开首页。",
		"manage.save":                        "保存设置",
		"manage.stop":                        "停止分享",
		"manage.stop_confirm":                "停止这个分享？",
		"manage.stop_failed":                 "停止分享失败",
		"manage.default_lang":                "默认语言",
		"manage.default_lang_apply":          "设为默认",
		"manage.system_setup":                "系统初始化",
		"manage.system_settings":             "系统设置",
		"manage.path_clipboard_snapshot":     "剪贴板快照",
		"manage.mode_upload_enabled":         "上传已启用",
		"manage.mode_readonly":               "只读",
		"manage.visibility_public":           "首页可见",
		"manage.visibility_hidden":           "首页隐藏",
		"manage.section_share_code":          "分享码",
		"manage.section_public_home":         "公开首页",
		"manage.section_local_access":        "本机访问",
		"manage.section_lan_access":          "局域网访问",
		"manage.section_time":                "时间",
		"manage.no_lan_ipv4":                 "未检测到可用局域网 IPv4 地址",
		"manage.qr_hint":                     "手机扫码直接打开当前分享",
		"manage.show_on_home":                "在首页显示这个分享",
		"manage.created_at":                  "创建于",
		"manage.updated_at":                  "最近活动",
		"status.available":                   "可访问",
		"status.unavailable":                 "已失效",
		"share.type.dir":                     "文件夹",
		"share.type.file":                    "文件",
		"share.type.clipboard_text":          "剪贴板文本",
		"share.type.clipboard_image":         "剪贴板图片",
		"share.open":                         "打开分享",
		"share.download":                     "下载文件",
		"share.download_text":                "下载文本",
		"share.download_image":               "下载原图",
		"share.current_dir":                  "当前目录",
		"share.back_parent":                  "返回上一级",
		"share.path":                         "路径",
		"share.h2.clipboard_text":            "剪贴板文本",
		"share.h2.clipboard_image":           "剪贴板图片",
		"share.h2.dir":                       "内容列表",
		"share.h2.file":                      "文件内容",
		"share.hint.clipboard_text":          "该分享来自剪贴板文本快照，仅支持只读查看和下载。",
		"share.hint.clipboard_image":         "该分享来自剪贴板图片快照，可预览和下载原图。",
		"share.hint.dir":                     "目录默认只读。只有设置上传密码时，页面才允许上传文件到当前目录。",
		"share.hint.text_file":               "这是文本文件预览，可复制或下载原文件。",
		"share.hint.image_file":              "这是图片文件预览，可下载原图。",
		"share.hint.file":                    "文件分享始终只读，可直接下载。",
		"share.download_all":                 "下载整个分享内容",
		"share.col_name":                     "名称",
		"share.col_size":                     "大小",
		"share.col_mod_time":                 "修改时间",
		"share.col_actions":                  "操作",
		"share.enter":                        "进入",
		"share.archive_download":             "打包下载",
		"share.folder_empty":                 "目录为空",
		"share.h2.upload":                    "上传入口",
		"share.h2.access_mode":               "访问模式",
		"share.hint.upload_enabled":          "输入分享者设置的上传密码后，可把文件分片上传到当前目录。上传过程中会显示实时进度。",
		"share.upload_password_placeholder":  "上传密码",
		"share.upload_file":                  "上传文件",
		"share.upload_folder":                "选择文件夹",
		"share.readonly.unavailable":         "该分享当前不可用，因此不能上传或下载内容。",
		"share.readonly.clipboard":           "剪贴板分享为只读快照，不提供上传能力。",
		"share.readonly.dir":                 "当前目录分享为只读模式。若需要上传文件，请由分享者重新设置带密码的共享。",
		"share.readonly.file":                "文件分享不提供上传能力。",
		"share.readonly.missing":             "该分享仍然存在于管理器中，但它指向的原始文件或文件夹已不存在。请联系分享者重新创建分享。",
		"share.error_root_missing":           "该分享对应的文件或文件夹已不存在，可能已被移动或删除。",
		"share.error_current_dir_missing":    "当前目录已不存在，可能已被移动或删除。",
		"share.error_file_missing":           "文件已不存在或已被移动。",
		"share.error_archive_dir_missing":    "要下载的文件夹已不存在，可能已被移动或删除。",
		"share.folder_size_label":            "文件夹",
		"share.root_dir":                     "根目录",
		"share.default_clipboard_text_name":  "剪贴板文本",
		"share.default_clipboard_image_name": "剪贴板图片",
		"share.default_name":                 "未命名分享",
		"lang.zh":                            "中文",
		"lang.en":                            "English",
		"lang.switch":                        "语言",
		"setup.title":                        "Web Share 本地设置",
		"setup.subtitle":                     "这是可选的本地设置页。日常使用以托盘为主，需要时再通过这里调整语言、右键菜单、托盘和开机自启。",
		"setup.section_language":             "默认语言",
		"setup.section_actions":              "初始化动作",
		"setup.section_status":               "当前状态",
		"setup.apply":                        "应用初始化设置",
		"setup.open_manage":                  "进入管理页面",
		"setup.install_context":              "安装右键菜单",
		"setup.enable_autostart":             "启用开机自启",
		"setup.start_tray":                   "立即启动托盘",
		"setup.complete":                     "标记初始化完成",
		"setup.completed_yes":                "已完成",
		"setup.completed_no":                 "未完成",
		"setup.status_manager":               "管理器",
		"setup.status_tray":                  "托盘",
		"setup.status_context":               "右键菜单",
		"setup.status_autostart":             "开机自启",
		"setup.status_running":               "运行中",
		"setup.status_stopped":               "未运行",
		"setup.status_installed":             "已安装",
		"setup.status_missing":               "未安装",
		"setup.status_enabled":               "已启用",
		"setup.status_disabled":              "未启用",
		"setup.apply_ok":                     "初始化设置已应用。",
		"setup.apply_failed":                 "初始化设置应用失败。",
		"system.title":                       "系统设置",
		"system.subtitle":                    "在这里调整默认语言和 Windows 集成项。移除右键菜单或开机自启后，你可以退出程序并手动删除可执行文件与本地数据。",
		"system.section_status":              "当前状态",
		"system.section_language":            "默认语言",
		"system.section_actions":             "系统操作",
		"system.action_save_language":        "保存语言",
		"system.action_install_context":      "安装右键菜单",
		"system.action_remove_context":       "卸载右键菜单",
		"system.action_enable_autostart":     "启用开机自启",
		"system.action_disable_autostart":    "禁用开机自启",
		"system.action_start_tray":           "启动托盘",
		"system.action_stop_tray":            "停止托盘",
		"system.action_stop_program":         "停止程序",
		"system.action_uninstall_all":        "一键卸载系统集成",
		"system.action_mark_setup_done":      "标记初始化完成",
		"system.action_mark_setup_todo":      "标记未完成初始化",
		"system.apply_ok":                    "系统设置已应用。",
		"system.apply_failed":                "系统设置应用失败。",
		"system.back_manage":                 "返回管理页面",
		"system.open_setup":                  "打开初始化页面",
	},
	langEN: {
		"site.brand":                         "Web Share",
		"home.title":                         "Visible Shares",
		"home.subtitle":                      "Only shares marked visible by the owner are listed here. You can also open a share directly by code.",
		"home.input_code":                    "Open by Share Code",
		"home.code_placeholder":              "e.g. a7k2m3",
		"home.open_share":                    "Open Share",
		"home.visible":                       "Public Shares",
		"home.empty":                         "No public shares yet.",
		"home.open":                          "Open",
		"home.copy":                          "Copy",
		"home.download":                      "Download",
		"home.share_code":                    "Code",
		"home.code_not_found":                "Share code not found",
		"home.file_meta_prefix":              "File",
		"home.copy_failed":                   "Copy failed. Please try again later.",
		"manage.title":                       "Active Shares",
		"manage.subtitle":                    "You can configure share names, code-based access, and whether they are visible on the home page.",
		"manage.save":                        "Save",
		"manage.stop":                        "Stop",
		"manage.stop_confirm":                "Stop this share?",
		"manage.stop_failed":                 "Failed to stop share",
		"manage.default_lang":                "Default Language",
		"manage.default_lang_apply":          "Apply",
		"manage.system_setup":                "System Setup",
		"manage.system_settings":             "System Settings",
		"manage.path_clipboard_snapshot":     "Clipboard Snapshot",
		"manage.mode_upload_enabled":         "Upload Enabled",
		"manage.mode_readonly":               "Read Only",
		"manage.visibility_public":           "Public",
		"manage.visibility_hidden":           "Hidden",
		"manage.section_share_code":          "Share Code",
		"manage.section_public_home":         "Public Home",
		"manage.section_local_access":        "Local Access",
		"manage.section_lan_access":          "LAN Access",
		"manage.section_time":                "Timestamps",
		"manage.no_lan_ipv4":                 "No available LAN IPv4 address detected",
		"manage.qr_hint":                     "Scan with phone to open this share",
		"manage.show_on_home":                "Show this share on home page",
		"manage.created_at":                  "Created at",
		"manage.updated_at":                  "Last active",
		"status.available":                   "Available",
		"status.unavailable":                 "Unavailable",
		"share.type.dir":                     "Folder",
		"share.type.file":                    "File",
		"share.type.clipboard_text":          "Clipboard Text",
		"share.type.clipboard_image":         "Clipboard Image",
		"share.open":                         "Open Share",
		"share.download":                     "Download File",
		"share.download_text":                "Download Text",
		"share.download_image":               "Download Image",
		"share.current_dir":                  "Current Folder",
		"share.back_parent":                  "Back to Parent",
		"share.path":                         "Path",
		"share.h2.clipboard_text":            "Clipboard Text",
		"share.h2.clipboard_image":           "Clipboard Image",
		"share.h2.dir":                       "Contents",
		"share.h2.file":                      "File Content",
		"share.hint.clipboard_text":          "This share is a read-only snapshot from clipboard text.",
		"share.hint.clipboard_image":         "This share is a clipboard image snapshot and supports preview/download.",
		"share.hint.dir":                     "Folder shares are read-only by default. Upload is enabled only when an upload password is set.",
		"share.hint.text_file":               "Text file preview. You can copy or download the original file.",
		"share.hint.image_file":              "Image preview. You can download the original image.",
		"share.hint.file":                    "File shares are always read-only and can be downloaded directly.",
		"share.download_all":                 "Download Entire Share",
		"share.col_name":                     "Name",
		"share.col_size":                     "Size",
		"share.col_mod_time":                 "Modified",
		"share.col_actions":                  "Actions",
		"share.enter":                        "Open",
		"share.archive_download":             "Archive Download",
		"share.folder_empty":                 "Folder is empty",
		"share.h2.upload":                    "Upload",
		"share.h2.access_mode":               "Access Mode",
		"share.hint.upload_enabled":          "Enter the upload password to upload files in chunks to the current folder with real-time progress.",
		"share.upload_password_placeholder":  "Upload Password",
		"share.upload_file":                  "Upload File",
		"share.upload_folder":                "Choose Folder",
		"share.readonly.unavailable":         "This share is unavailable, so upload/download is disabled.",
		"share.readonly.clipboard":           "Clipboard shares are read-only snapshots.",
		"share.readonly.dir":                 "This folder share is read-only. Ask the owner to recreate it with upload password enabled.",
		"share.readonly.file":                "File shares do not support upload.",
		"share.readonly.missing":             "This share still exists in manager, but its source file/folder no longer exists.",
		"share.error_root_missing":           "The source file/folder for this share no longer exists (moved or deleted).",
		"share.error_current_dir_missing":    "The current folder no longer exists (moved or deleted).",
		"share.error_file_missing":           "The file no longer exists or has been moved.",
		"share.error_archive_dir_missing":    "The folder to download no longer exists (moved or deleted).",
		"share.folder_size_label":            "Folder",
		"share.root_dir":                     "Root",
		"share.default_clipboard_text_name":  "Clipboard Text",
		"share.default_clipboard_image_name": "Clipboard Image",
		"share.default_name":                 "Untitled Share",
		"lang.zh":                            "中文",
		"lang.en":                            "English",
		"lang.switch":                        "Language",
		"setup.title":                        "Web Share Local Settings",
		"setup.subtitle":                     "This optional local page is a settings center. Daily use is tray-first, and you can adjust language, context menu, tray, and auto start here when needed.",
		"setup.section_language":             "Default Language",
		"setup.section_actions":              "Setup Actions",
		"setup.section_status":               "Current Status",
		"setup.apply":                        "Apply Setup",
		"setup.open_manage":                  "Open Manager",
		"setup.install_context":              "Install Context Menu",
		"setup.enable_autostart":             "Enable Auto Start",
		"setup.start_tray":                   "Start Tray Now",
		"setup.complete":                     "Mark Setup Completed",
		"setup.completed_yes":                "Completed",
		"setup.completed_no":                 "Not Completed",
		"setup.status_manager":               "Manager",
		"setup.status_tray":                  "Tray",
		"setup.status_context":               "Context Menu",
		"setup.status_autostart":             "Auto Start",
		"setup.status_running":               "Running",
		"setup.status_stopped":               "Stopped",
		"setup.status_installed":             "Installed",
		"setup.status_missing":               "Missing",
		"setup.status_enabled":               "Enabled",
		"setup.status_disabled":              "Disabled",
		"setup.apply_ok":                     "Setup settings applied.",
		"setup.apply_failed":                 "Failed to apply setup settings.",
		"system.title":                       "System Settings",
		"system.subtitle":                    "Adjust default language and Windows integration here. After removing context menu or auto start, you can exit the program and delete the executable and local data manually.",
		"system.section_status":              "Current Status",
		"system.section_language":            "Default Language",
		"system.section_actions":             "System Actions",
		"system.action_save_language":        "Save Language",
		"system.action_install_context":      "Install Context Menu",
		"system.action_remove_context":       "Remove Context Menu",
		"system.action_enable_autostart":     "Enable Auto Start",
		"system.action_disable_autostart":    "Disable Auto Start",
		"system.action_start_tray":           "Start Tray",
		"system.action_stop_tray":            "Stop Tray",
		"system.action_stop_program":         "Stop Program",
		"system.action_uninstall_all":        "One-Click Uninstall",
		"system.action_mark_setup_done":      "Mark Setup Completed",
		"system.action_mark_setup_todo":      "Mark Setup Not Completed",
		"system.apply_ok":                    "System settings applied.",
		"system.apply_failed":                "Failed to apply system settings.",
		"system.back_manage":                 "Back to Manager",
		"system.open_setup":                  "Open Setup Page",
	},
}

func tr(lang, key string) string {
	lang = normalizeLanguage(lang)
	if dict, ok := i18n[lang]; ok {
		if value, exists := dict[key]; exists {
			return value
		}
	}
	if value, ok := i18n[langEN][key]; ok {
		return value
	}
	return key
}

func normalizeLanguage(lang string) string {
	lang = strings.TrimSpace(strings.ToLower(lang))
	switch {
	case strings.HasPrefix(lang, "zh"):
		return langZH
	case strings.HasPrefix(lang, "en"):
		return langEN
	default:
		return ""
	}
}

func isSupportedLanguage(lang string) bool {
	_, ok := supportedLanguages[lang]
	return ok
}

func resolveLanguage(r *http.Request, defaultLang string) string {
	if lang := normalizeLanguage(r.URL.Query().Get("lang")); isSupportedLanguage(lang) {
		return lang
	}
	if cookie, err := r.Cookie("webshare_lang"); err == nil {
		if lang := normalizeLanguage(cookie.Value); isSupportedLanguage(lang) {
			return lang
		}
	}
	if lang := normalizeLanguage(defaultLang); isSupportedLanguage(lang) {
		return lang
	}

	accept := strings.ToLower(r.Header.Get("Accept-Language"))
	for _, part := range strings.Split(accept, ",") {
		if lang := normalizeLanguage(strings.TrimSpace(part)); isSupportedLanguage(lang) {
			return lang
		}
	}
	return langEN
}

func setLanguageCookie(w http.ResponseWriter, lang string) {
	if !isSupportedLanguage(lang) {
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "webshare_lang",
		Value:    lang,
		Path:     "/",
		HttpOnly: false,
		SameSite: http.SameSiteLaxMode,
	})
}

func withLanguageInURL(r *http.Request, lang string) string {
	values := cloneValues(r.URL.Query())
	values.Set("lang", lang)
	if encoded := values.Encode(); encoded != "" {
		return r.URL.Path + "?" + encoded
	}
	return r.URL.Path
}

func cloneValues(values url.Values) url.Values {
	out := make(url.Values, len(values))
	for k, items := range values {
		cloned := make([]string, len(items))
		copy(cloned, items)
		out[k] = cloned
	}
	return out
}

func SystemDefaultLanguage() string {
	dbPath, err := resolveDBPath("")
	if err != nil {
		return langEN
	}
	store, err := openSettingsStore(dbPath)
	if err != nil {
		return langEN
	}
	return ensureSettingsDefaultLanguage(store)
}

func SetSystemDefaultLanguage(lang string) error {
	lang = normalizeLanguage(lang)
	if !isSupportedLanguage(lang) {
		return errors.New("unsupported language")
	}

	dbPath, err := resolveDBPath("")
	if err != nil {
		return err
	}
	store, err := openSettingsStore(dbPath)
	if err != nil {
		return err
	}
	return store.SetDefaultLanguage(lang)
}

func SetupCompleted() bool {
	dbPath, err := resolveDBPath("")
	if err != nil {
		return false
	}
	store, err := openSettingsStore(dbPath)
	if err != nil {
		return false
	}
	done, err := store.GetSetupCompleted()
	if err != nil {
		return false
	}
	return done
}

func ensureSettingsDefaultLanguage(store SettingsStore) string {
	if store == nil {
		return langEN
	}

	lang, err := store.GetDefaultLanguage()
	if err == nil {
		lang = normalizeLanguage(lang)
		if isSupportedLanguage(lang) {
			return lang
		}
	}

	lang = normalizeLanguage(detectSystemLanguage())
	if !isSupportedLanguage(lang) {
		lang = langEN
	}
	if err := store.SetDefaultLanguage(lang); err != nil {
		return lang
	}
	return lang
}
