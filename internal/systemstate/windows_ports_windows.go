//go:build windows

package systemstate

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"golang.org/x/sys/windows/registry"

	"web-share/internal/logx"
	"web-share/internal/shell"
)

func NewWindowsService(logger Logger) *Service {
	service := NewService(logger)
	service.ContextMenu = windowsContextMenuPort{}
	service.Autostart = windowsAutostartPort{}
	service.Tray = windowsTrayPort{}
	service.Manager = windowsManagerPort{}
	return service
}

type windowsContextMenuPort struct{}

func (windowsContextMenuPort) Inspect(exePath string) (InspectResult, error) {
	fileExists, err := registryKeyExists(`Software\Classes\*\shell\web-share`)
	if err != nil {
		return InspectResult{}, err
	}
	folderExists, err := registryKeyExists(`Software\Classes\Directory\shell\web-share`)
	if err != nil {
		return InspectResult{}, err
	}
	fileStoreExists, err := registryKeyExists(`Software\Classes\WebShare.FileContextMenu`)
	if err != nil {
		return InspectResult{}, err
	}
	folderStoreExists, err := registryKeyExists(`Software\Classes\WebShare.DirectoryContextMenu`)
	if err != nil {
		return InspectResult{}, err
	}

	installed := fileExists && folderExists && fileStoreExists && folderStoreExists
	warnings := make([]string, 0, 4)
	if fileExists != folderExists || fileStoreExists != folderStoreExists || fileExists != fileStoreExists {
		warnings = append(warnings, "Detected partial context menu state.")
	}

	dirty := false
	if installed {
		fileCommand, err := registryDefaultValue(`Software\Classes\WebShare.FileContextMenu\shell\readonly\command`)
		if err != nil {
			return InspectResult{}, err
		}
		folderPasswordCommand, err := registryDefaultValue(`Software\Classes\WebShare.DirectoryContextMenu\shell\password\command`)
		if err != nil {
			return InspectResult{}, err
		}
		expectedReadOnly := `"` + exePath + `" enqueue "%1"`
		if fileCommand != expectedReadOnly {
			dirty = true
			warnings = append(warnings, "Read-only context menu command does not match the current executable.")
		}
		if folderPasswordCommand == "" || !strings.Contains(strings.ToLower(folderPasswordCommand), ` prompt-share `) {
			dirty = true
			warnings = append(warnings, "Password share command is missing or still points to a legacy implementation.")
		}
		if strings.Contains(strings.ToLower(folderPasswordCommand), "prompt-share.vbs") {
			dirty = true
			warnings = append(warnings, "Detected legacy prompt-share.vbs command.")
		}
	}

	if !installed && (fileExists || folderExists || fileStoreExists || folderStoreExists) {
		dirty = true
	}
	return InspectResult{
		Installed: installed,
		Dirty:     dirty,
		Warnings:  warnings,
	}, nil
}

func (windowsContextMenuPort) Install(exePath, lang string) error {
	return shell.InstallContextMenuWithLanguage(exePath, lang)
}

func (windowsContextMenuPort) Remove() error {
	return shell.UninstallContextMenu()
}

type windowsAutostartPort struct{}

func (windowsAutostartPort) Inspect(taskName, command string) (InspectResult, error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, registry.QUERY_VALUE)
	if err != nil {
		if err == registry.ErrNotExist {
			return InspectResult{}, nil
		}
		return InspectResult{}, err
	}
	defer key.Close()

	value, _, err := key.GetStringValue(taskName)
	if err != nil {
		if err == registry.ErrNotExist {
			return InspectResult{}, nil
		}
		return InspectResult{}, err
	}

	result := InspectResult{Installed: true}
	if strings.TrimSpace(value) != strings.TrimSpace(command) {
		result.Dirty = true
		result.Warnings = append(result.Warnings, "Auto start command does not match the expected value.")
	}
	return result, nil
}

func (windowsAutostartPort) Enable(taskName, command string) error {
	return shell.SetCurrentUserRun(taskName, command)
}

func (windowsAutostartPort) Disable(taskName string) error {
	return shell.DeleteCurrentUserRun(taskName)
}

type windowsTrayPort struct{}

func (windowsTrayPort) Inspect() (InspectResult, error) {
	running, err := shell.TrayRunning()
	if err != nil {
		return InspectResult{}, err
	}
	return InspectResult{Installed: running}, nil
}

func (windowsTrayPort) Start(exePath string) error {
	return shell.StartDetached(exePath, "tray")
}

func (windowsTrayPort) Stop() error {
	return shell.StopTray()
}

type windowsManagerPort struct{}

func (windowsManagerPort) Inspect() (InspectResult, error) {
	client := &http.Client{Timeout: 800 * time.Millisecond}
	resp, err := client.Get("http://127.0.0.1:21910/api/ping")
	if err != nil {
		return InspectResult{}, nil
	}
	defer resp.Body.Close()
	return InspectResult{Installed: resp.StatusCode == http.StatusOK}, nil
}

func (windowsManagerPort) Start(exePath string) error {
	if err := shell.StartDetached(exePath, "run-manager"); err != nil {
		return err
	}
	deadline := time.Now().Add(6 * time.Second)
	for time.Now().Before(deadline) {
		state, _ := windowsManagerPort{}.Inspect()
		if state.Installed {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return errManagerNotReady
}

func (windowsManagerPort) Stop() error {
	req, err := http.NewRequest(http.MethodPost, "http://127.0.0.1:21910/api/shutdown", nil)
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: 1500 * time.Millisecond}
	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	return nil
}

func registryKeyExists(path string) (bool, error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, path, registry.READ)
	if err != nil {
		if err == registry.ErrNotExist {
			return false, nil
		}
		return false, err
	}
	defer key.Close()
	return true, nil
}

func registryDefaultValue(path string) (string, error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, path, registry.QUERY_VALUE)
	if err != nil {
		if err == registry.ErrNotExist {
			return "", nil
		}
		return "", err
	}
	defer key.Close()
	value, _, err := key.GetStringValue("")
	if err != nil {
		if err == registry.ErrNotExist {
			return "", nil
		}
		return "", err
	}
	return value, nil
}

var _ = logx.LevelInfo

var errManagerNotReady = errors.New("manager did not become ready in time")
