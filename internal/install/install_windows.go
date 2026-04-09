//go:build windows

package install

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"web-share/internal/logx"
	"web-share/internal/manager"
	"web-share/internal/notify"
	"web-share/internal/shell"
	"web-share/internal/systemstate"
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
	service := systemstate.NewWindowsService(logx.New())

	if err := manager.SetSystemDefaultLanguage(lang); err != nil {
		return fmt.Errorf("persist default language: %w", err)
	}
	if opts.InstallContextMenu {
		result := service.EnsureContextMenuInstalled(exePath, lang)
		if !result.OK {
			return fmt.Errorf("install context menu: %s", strings.Join(result.Errors, "; "))
		}
	}
	if opts.InstallStartupTask {
		action := shell.QuoteCommand(exePath, "start", "-lang", lang)
		if opts.NotifyStart {
			action += " -notify-start=true"
		}
		result := service.EnsureAutostartEnabled(taskName, action)
		if !result.OK {
			return fmt.Errorf("install auto start: %s", strings.Join(result.Errors, "; "))
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
		result := systemstate.NewWindowsService(logx.New()).EnsureManagerRunning(exePath)
		if !result.OK {
			return fmt.Errorf("start manager: %s", strings.Join(result.Errors, "; "))
		}
		managerStarted = result.Changed
		managerReady = true
	}

	if opts.StartTray {
		result := systemstate.NewWindowsService(logx.New()).EnsureTrayRunning(exePath)
		if !result.OK {
			return fmt.Errorf("start tray: %s", strings.Join(result.Errors, "; "))
		}
		trayStarted = result.Changed
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
	service := systemstate.NewWindowsService(logx.New())
	command := ""
	if strings.TrimSpace(opts.ExePath) != "" {
		resolved, err := resolveExecutable(opts.ExePath)
		if err == nil {
			command = shell.QuoteCommand(resolved, "start", "-lang", normalizeLanguage(opts.Language), "-notify-start=true")
		}
	}
	if opts.RemoveMenu {
		result := service.EnsureContextMenuRemoved("")
		if !result.OK {
			return fmt.Errorf("remove context menu: %s", strings.Join(result.Errors, "; "))
		}
	}
	if opts.RemoveAutostart {
		result := service.EnsureAutostartDisabled(normalizeTaskName(opts.TaskName), command)
		if !result.OK {
			return fmt.Errorf("remove auto start: %s", strings.Join(result.Errors, "; "))
		}
	}
	if opts.StopProcesses {
		stopService := systemstate.NewWindowsService(logx.New())
		_ = stopService.EnsureManagerStopped()
		_ = stopService.EnsureTrayStopped()
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
	exePath, err := os.Executable()
	if err != nil {
		return err
	}
	dbPath := filepath.Join(filepath.Dir(exePath), "web-share.db")
	for _, path := range []string{dbPath, dbPath + "-shm", dbPath + "-wal"} {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}
