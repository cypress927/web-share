package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"web-share/internal/install"
	"web-share/internal/manager"
	"web-share/internal/notify"
	"web-share/internal/shell"
	"web-share/internal/tray"
)

func Run() error {
	if len(os.Args) < 2 {
		printUsage()
		return nil
	}

	switch os.Args[1] {
	case "share", "enqueue":
		return runEnqueue(os.Args[2:])
	case "install":
		return runInstall(os.Args[2:])
	case "start":
		return runStart(os.Args[2:])
	case "repair":
		return runRepair(os.Args[2:])
	case "uninstall":
		return runUninstall(os.Args[2:])
	case "run-manager":
		cfg := manager.DefaultConfig()
		if runtime.GOOS == "windows" {
			cfg.ApplySystemLanguage = applySystemLanguage
		}
		return manager.Run(cfg)
	case "tray":
		if runtime.GOOS != "windows" {
			return errors.New("tray mode is only supported on Windows")
		}
		return tray.Run()
	case "install-context-menu":
		return runInstallContextMenu(os.Args[2:])
	case "uninstall-context-menu":
		return runUninstallContextMenu()
	case "-h", "--help", "help":
		printUsage()
		return nil
	default:
		if len(os.Args) == 2 {
			return runEnqueue(os.Args[1:])
		}

		return fmt.Errorf("unknown command: %s", os.Args[1])
	}
}

func runEnqueue(args []string) error {
	fs := flag.NewFlagSet("enqueue", flag.ContinueOnError)
	password := fs.String("password", "", "Upload password for shared folders")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if fs.NArg() != 1 {
		return errors.New("enqueue requires exactly one file or directory path")
	}

	target, err := filepath.Abs(fs.Arg(0))
	if err != nil {
		return fmt.Errorf("resolve target path: %w", err)
	}

	if _, err := os.Stat(target); err != nil {
		return fmt.Errorf("stat target path: %w", err)
	}

	if runtime.GOOS == "windows" {
		lang := manager.SystemDefaultLanguage()
		if manager.IsReachable() {
			_ = notify.Info("Web Share", appMessage(lang, "adding_share"))
		} else {
			_ = notify.Info("Web Share", appMessage(lang, "starting"))
		}
	}

	managerStarted, err := ensureManager()
	if err != nil {
		if runtime.GOOS == "windows" {
			lang := manager.SystemDefaultLanguage()
			_ = notify.Error("Web Share", appMessage(lang, "start_failed")+err.Error())
		}
		return err
	}

	if runtime.GOOS == "windows" {
		lang := manager.SystemDefaultLanguage()
		if err := tray.EnsureStarted(); err != nil {
			_ = notify.Error("Web Share", appMessage(lang, "tray_start_failed")+err.Error())
		}
		if managerStarted {
			_ = notify.Info("Web Share", appMessage(lang, "started"))
		}
	}

	reqBody, err := json.Marshal(manager.CreateShareRequest{
		Path:     target,
		Password: *password,
	})
	if err != nil {
		return err
	}

	resp, err := http.Post(manager.LocalAPI("/api/shares"), "application/json", bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("enqueue share: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		if runtime.GOOS == "windows" {
			lang := manager.SystemDefaultLanguage()
			reason := strings.TrimSpace(string(body))
			if reason == "" {
				reason = resp.Status
			}
			_ = notify.Error("Web Share", appMessage(lang, "add_failed")+reason)
		}
		return fmt.Errorf("enqueue share failed: %s", bytes.TrimSpace(body))
	}

	if runtime.GOOS == "windows" {
		lang := manager.SystemDefaultLanguage()
		var result struct {
			Name string `json:"name"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&result)
		shareName := result.Name
		if shareName == "" {
			shareName = filepath.Base(target)
		}
		_ = notify.Info("Web Share", appMessage(lang, "added")+shareName)
	}

	return nil
}

func ensureManager() (bool, error) {
	if manager.IsReachable() {
		return false, nil
	}

	exePath, err := os.Executable()
	if err != nil {
		return false, fmt.Errorf("resolve executable: %w", err)
	}

	if err := shell.StartDetached(exePath, "run-manager"); err != nil {
		return false, fmt.Errorf("start manager: %w", err)
	}

	deadline := time.Now().Add(6 * time.Second)
	for time.Now().Before(deadline) {
		if manager.IsReachable() {
			return true, nil
		}
		time.Sleep(200 * time.Millisecond)
	}

	return false, errors.New("manager did not become ready in time")
}

func runInstallContextMenu(args []string) error {
	if runtime.GOOS != "windows" {
		return errors.New("context menu installation is only supported on Windows")
	}

	exePath, lang, err := resolveExecutableArg(args)
	if err != nil {
		return err
	}

	return shell.InstallContextMenuWithLanguage(exePath, lang)
}

func runInstall(args []string) error {
	fs := flag.NewFlagSet("install", flag.ContinueOnError)
	defaultLang := manager.SystemDefaultLanguage()
	exe := fs.String("exe", "", "Path to web-share.exe")
	lang := fs.String("lang", defaultLang, "Default language (en-US or zh-CN)")
	contextMenu := fs.Bool("context-menu", true, "Install Windows context menu")
	startupTask := fs.Bool("startup-task", true, "Install startup task")
	startNow := fs.Bool("start-now", true, "Start manager and tray immediately")
	notifyStart := fs.Bool("notify-start", true, "Show startup notification")
	forceTask := fs.Bool("force-task", false, "Replace existing startup task")
	taskName := fs.String("task-name", "", "Scheduled task name")
	if err := fs.Parse(args); err != nil {
		return err
	}

	return install.Install(install.InstallOptions{
		ExePath:            *exe,
		Language:           *lang,
		InstallContextMenu: *contextMenu,
		InstallStartupTask: *startupTask,
		StartNow:           *startNow,
		NotifyStart:        *notifyStart,
		ForceTask:          *forceTask,
		TaskName:           *taskName,
	})
}

func runStart(args []string) error {
	fs := flag.NewFlagSet("start", flag.ContinueOnError)
	defaultLang := manager.SystemDefaultLanguage()
	exe := fs.String("exe", "", "Path to web-share.exe")
	lang := fs.String("lang", defaultLang, "Language for notifications (en-US or zh-CN)")
	startManager := fs.Bool("manager", true, "Start manager if needed")
	startTray := fs.Bool("tray", true, "Start tray if needed")
	notifyStart := fs.Bool("notify-start", true, "Show startup notification")
	waitSeconds := fs.Int("wait-seconds", 8, "Seconds to wait for manager ready")
	if err := fs.Parse(args); err != nil {
		return err
	}

	return install.Start(install.StartOptions{
		ExePath:      *exe,
		Language:     *lang,
		StartManager: *startManager,
		StartTray:    *startTray,
		NotifyStart:  *notifyStart,
		WaitTimeout:  time.Duration(*waitSeconds) * time.Second,
	})
}

func runRepair(args []string) error {
	fs := flag.NewFlagSet("repair", flag.ContinueOnError)
	defaultLang := manager.SystemDefaultLanguage()
	exe := fs.String("exe", "", "Path to web-share.exe")
	lang := fs.String("lang", defaultLang, "Default language (en-US or zh-CN)")
	startNow := fs.Bool("start-now", true, "Start manager and tray immediately")
	notifyStart := fs.Bool("notify-start", true, "Show startup notification")
	forceTask := fs.Bool("force-task", true, "Replace existing startup task")
	taskName := fs.String("task-name", "", "Scheduled task name")
	if err := fs.Parse(args); err != nil {
		return err
	}

	return install.Repair(install.InstallOptions{
		ExePath:            *exe,
		Language:           *lang,
		InstallContextMenu: true,
		InstallStartupTask: true,
		StartNow:           *startNow,
		NotifyStart:        *notifyStart,
		ForceTask:          *forceTask,
		TaskName:           *taskName,
	})
}

func runUninstall(args []string) error {
	fs := flag.NewFlagSet("uninstall", flag.ContinueOnError)
	defaultLang := manager.SystemDefaultLanguage()
	exe := fs.String("exe", "", "Path to web-share.exe")
	lang := fs.String("lang", defaultLang, "Language for messages (en-US or zh-CN)")
	taskName := fs.String("task-name", "", "Scheduled task name")
	removeData := fs.Bool("remove-data", false, "Remove local Web Share data")
	stopProcesses := fs.Bool("stop-processes", true, "Stop manager and tray")
	removeMenu := fs.Bool("remove-context-menu", true, "Remove Windows context menu")
	removeAutostart := fs.Bool("remove-startup-task", true, "Remove startup task")
	if err := fs.Parse(args); err != nil {
		return err
	}

	return install.Uninstall(install.UninstallOptions{
		ExePath:         *exe,
		Language:        *lang,
		TaskName:        *taskName,
		RemoveData:      *removeData,
		StopProcesses:   *stopProcesses,
		RemoveMenu:      *removeMenu,
		RemoveAutostart: *removeAutostart,
	})
}

func runUninstallContextMenu() error {
	if runtime.GOOS != "windows" {
		return errors.New("context menu installation is only supported on Windows")
	}

	return shell.UninstallContextMenu()
}

func resolveExecutableArg(args []string) (string, string, error) {
	fs := flag.NewFlagSet("exe", flag.ContinueOnError)
	exe := fs.String("exe", "", "Path to web-share.exe")
	defaultLang := manager.SystemDefaultLanguage()
	lang := fs.String("lang", defaultLang, "Language for context menu labels (en-US or zh-CN)")
	if err := fs.Parse(args); err != nil {
		return "", "", err
	}

	exePath := *exe
	if exePath == "" {
		currentExe, err := os.Executable()
		if err != nil {
			return "", "", fmt.Errorf("resolve executable: %w", err)
		}
		exePath = currentExe
	}

	exePath, err := filepath.Abs(exePath)
	if err != nil {
		return "", "", fmt.Errorf("resolve executable: %w", err)
	}

	if filepath.Ext(exePath) == "" {
		exePath += ".exe"
	}

	if _, err := os.Stat(exePath); err != nil {
		return "", "", fmt.Errorf("stat executable: %w", err)
	}

	return exePath, *lang, nil
}

func applySystemLanguage(lang string) error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable: %w", err)
	}
	if err := shell.InstallContextMenuWithLanguage(exePath, lang); err != nil {
		return err
	}
	return tray.Restart()
}

func appMessage(lang, key string) string {
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(lang)), "zh") {
		switch key {
		case "adding_share":
			return "正在添加分享..."
		case "starting":
			return "正在启动 Web Share..."
		case "start_failed":
			return "启动失败："
		case "tray_start_failed":
			return "托盘启动失败："
		case "started":
			return "Web Share 已启动"
		case "add_failed":
			return "分享添加失败："
		case "added":
			return "分享已添加："
		}
	}
	switch key {
	case "adding_share":
		return "Adding share..."
	case "starting":
		return "Starting Web Share..."
	case "start_failed":
		return "Startup failed: "
	case "tray_start_failed":
		return "Tray startup failed: "
	case "started":
		return "Web Share started"
	case "add_failed":
		return "Failed to add share: "
	case "added":
		return "Share added: "
	default:
		return ""
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `web-share

Usage:
  web-share install [-lang en-US|zh-CN] [-context-menu=true] [-startup-task=true] [-start-now=true]
  web-share start [-lang en-US|zh-CN] [-manager=true] [-tray=true] [-notify-start=true]
  web-share repair [-lang en-US|zh-CN]
  web-share uninstall [-remove-data=false]
  web-share enqueue [-password secret] <path>
  web-share share [-password secret] <path>
  web-share tray
  web-share run-manager
  web-share install-context-menu [-exe C:\path\to\web-share.exe] [-lang en-US|zh-CN]
  web-share uninstall-context-menu

Notes:
  - install configures the default language, context menu, optional startup task, and optional immediate startup.
  - start ensures the local manager and tray are running.
  - repair reapplies the standard Windows integration.
  - uninstall removes Windows integration and can optionally remove local data.
  - enqueue/share sends a new share task to the local manager.
  - The manager keeps all shares in one background process.
  - If manager or tray is not running, enqueue starts them in the background.
  - The tray icon opens the localhost management page.
  - File shares are always read-only.
  - Folder shares become upload-enabled only when -password is set.
`)
}
