//go:build windows

package install

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"web-share/internal/manager"
	"web-share/internal/notify"
	"web-share/internal/shell"
	"web-share/internal/tray"
)

const defaultTaskName = "WebShare.AutoStart"

type InstallOptions struct {
	ExePath            string
	Language           string
	InstallContextMenu bool
	InstallStartupTask bool
	StartNow           bool
	NotifyStart        bool
	ForceTask          bool
	TaskName           string
}

type StartOptions struct {
	ExePath      string
	Language     string
	StartManager bool
	StartTray    bool
	NotifyStart  bool
	WaitTimeout  time.Duration
}

type UninstallOptions struct {
	ExePath         string
	Language        string
	TaskName        string
	RemoveData      bool
	StopProcesses   bool
	RemoveMenu      bool
	RemoveAutostart bool
}

func Install(opts InstallOptions) error {
	exePath, err := resolveExecutable(opts.ExePath)
	if err != nil {
		return err
	}
	lang := normalizeLanguage(opts.Language)
	taskName := normalizeTaskName(opts.TaskName)

	if err := manager.SetSystemDefaultLanguage(lang); err != nil {
		return fmt.Errorf("persist default language: %w", err)
	}
	if opts.InstallContextMenu {
		if err := shell.InstallContextMenuWithLanguage(exePath, lang); err != nil {
			return fmt.Errorf("install context menu: %w", err)
		}
	}
	if opts.InstallStartupTask {
		if err := InstallStartupTask(exePath, taskName, lang, opts.NotifyStart, opts.ForceTask); err != nil {
			return err
		}
	}
	if opts.StartNow {
		return Start(StartOptions{
			ExePath:      exePath,
			Language:     lang,
			StartManager: true,
			StartTray:    true,
			NotifyStart:  opts.NotifyStart,
			WaitTimeout:  8 * time.Second,
		})
	}
	return nil
}

func Start(opts StartOptions) error {
	exePath, err := resolveExecutable(opts.ExePath)
	if err != nil {
		return err
	}
	lang := normalizeLanguage(opts.Language)
	waitTimeout := opts.WaitTimeout
	if waitTimeout <= 0 {
		waitTimeout = 8 * time.Second
	}

	managerStarted := false
	trayStarted := false

	managerReady := manager.IsReachable()
	if opts.StartManager && !managerReady {
		if err := shell.StartDetached(exePath, "run-manager"); err != nil {
			return fmt.Errorf("start manager: %w", err)
		}
		managerStarted = true
		deadline := time.Now().Add(waitTimeout)
		for time.Now().Before(deadline) {
			if manager.IsReachable() {
				managerReady = true
				break
			}
			time.Sleep(250 * time.Millisecond)
		}
		if !managerReady {
			return errors.New("manager did not become ready in time")
		}
	}

	if opts.StartTray {
		alreadyRunning, err := trayRunning()
		if err != nil {
			return err
		}
		if err := tray.EnsureStarted(); err != nil {
			return fmt.Errorf("start tray: %w", err)
		}
		trayStarted = !alreadyRunning
	}

	if opts.NotifyStart {
		messageKey := "already_running"
		if managerStarted || trayStarted {
			messageKey = "started"
		}
		_ = notify.Info("Web Share", installMessage(lang, messageKey))
	}

	return nil
}

func Repair(opts InstallOptions) error {
	opts.InstallContextMenu = true
	opts.InstallStartupTask = true
	return Install(opts)
}

func Uninstall(opts UninstallOptions) error {
	if opts.RemoveMenu {
		if err := shell.UninstallContextMenu(); err != nil {
			return fmt.Errorf("remove context menu: %w", err)
		}
	}
	if opts.RemoveAutostart {
		if err := UninstallStartupTask(normalizeTaskName(opts.TaskName)); err != nil {
			return err
		}
	}
	if opts.StopProcesses {
		_ = shutdownManager()
		_ = tray.Stop()
	}
	if err := removePasswordPromptScript(); err != nil {
		return err
	}
	if opts.RemoveData {
		if err := removeDataDir(); err != nil {
			return err
		}
	}
	return nil
}

func InstallStartupTask(exePath, taskName, lang string, notifyStart, force bool) error {
	taskName = normalizeTaskName(taskName)
	exists, err := shell.CurrentUserRunExists(taskName)
	if err != nil {
		return fmt.Errorf("check auto start: %w", err)
	}
	if exists && !force {
		return fmt.Errorf("auto start entry already exists: %s", taskName)
	}

	action := shell.QuoteCommand(exePath, "start", "-lang", normalizeLanguage(lang))
	if notifyStart {
		action += " -notify-start=true"
	}
	if err := shell.SetCurrentUserRun(taskName, action); err != nil {
		return fmt.Errorf("install auto start: %w", err)
	}
	return nil
}

func UninstallStartupTask(taskName string) error {
	taskName = normalizeTaskName(taskName)
	exists, err := shell.CurrentUserRunExists(taskName)
	if err != nil {
		return fmt.Errorf("check auto start: %w", err)
	}
	if !exists {
		return nil
	}
	if err := shell.DeleteCurrentUserRun(taskName); err != nil {
		return fmt.Errorf("remove auto start: %w", err)
	}
	return nil
}

func resolveExecutable(exePath string) (string, error) {
	if strings.TrimSpace(exePath) == "" {
		currentExe, err := os.Executable()
		if err != nil {
			return "", fmt.Errorf("resolve executable: %w", err)
		}
		exePath = currentExe
	}
	exePath = strings.TrimSpace(exePath)
	if filepath.Ext(exePath) == "" {
		exePath += ".exe"
	}
	absPath, err := filepath.Abs(exePath)
	if err != nil {
		return "", fmt.Errorf("resolve executable: %w", err)
	}
	if _, err := os.Stat(absPath); err != nil {
		return "", fmt.Errorf("stat executable: %w", err)
	}
	return absPath, nil
}

func normalizeLanguage(lang string) string {
	lang = strings.TrimSpace(lang)
	if lang == "" {
		lang = manager.SystemDefaultLanguage()
	}
	if strings.HasPrefix(strings.ToLower(lang), "zh") {
		return "zh-CN"
	}
	return "en-US"
}

func normalizeTaskName(taskName string) string {
	taskName = strings.TrimSpace(taskName)
	if taskName == "" {
		return defaultTaskName
	}
	return taskName
}

func installMessage(lang, key string) string {
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(lang)), "zh") {
		switch key {
		case "started":
			return "Web Share 已启动"
		case "already_running":
			return "Web Share 已在运行"
		}
	}
	switch key {
	case "started":
		return "Web Share started successfully."
	case "already_running":
		return "Web Share is already running."
	default:
		return ""
	}
}

func trayRunning() (bool, error) {
	return shell.MutexExists(`Global\WebShareTraySingleton`)
}

func shutdownManager() error {
	req, err := http.NewRequest(http.MethodPost, manager.LocalAPI("/api/shutdown"), nil)
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: 1200 * time.Millisecond}
	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	return nil
}

func removePasswordPromptScript() error {
	baseDir := os.Getenv("LOCALAPPDATA")
	if strings.TrimSpace(baseDir) == "" {
		var err error
		baseDir, err = os.UserConfigDir()
		if err != nil {
			return err
		}
	}
	scriptPath := filepath.Join(baseDir, "WebShare", "prompt-share.vbs")
	if err := os.Remove(scriptPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func removeDataDir() error {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return err
	}
	dataDir := filepath.Join(cacheDir, "WebShare")
	if err := os.RemoveAll(dataDir); err != nil {
		return err
	}
	return nil
}
