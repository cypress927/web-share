//go:build windows

package shell

import (
	"fmt"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

type mutexHandle = windows.Handle

const webShareTrayMutexName = `Global\WebShareTraySingleton`
const webShareTrayQuitEventName = `Global\WebShareTrayQuitEvent`

func StartDetached(exePath string, args ...string) error {
	cmd := exec.Command(exePath, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: windows.DETACHED_PROCESS | windows.CREATE_NEW_PROCESS_GROUP,
	}
	return cmd.Start()
}

func OpenBrowser(target string) error {
	cmd := exec.Command("rundll32.exe", "url.dll,FileProtocolHandler", target)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd.Start()
}

func AcquireMutex(name string) (*mutexHandle, bool, error) {
	ptr, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return nil, false, err
	}

	handle, err := windows.CreateMutex(nil, true, ptr)
	if err != nil {
		if err == windows.ERROR_ALREADY_EXISTS {
			return nil, false, nil
		}
		return nil, false, err
	}

	if windows.GetLastError() == windows.ERROR_ALREADY_EXISTS {
		_ = windows.CloseHandle(handle)
		return nil, false, nil
	}

	return &handle, true, nil
}

func ReleaseMutex(handle *mutexHandle) error {
	if handle == nil {
		return nil
	}
	return windows.CloseHandle(*handle)
}

func MutexExists(name string) (bool, error) {
	ptr, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return false, err
	}

	handle, err := windows.OpenMutex(windows.SYNCHRONIZE, false, ptr)
	if err != nil {
		if err == windows.ERROR_FILE_NOT_FOUND {
			return false, nil
		}
		return false, err
	}
	_ = windows.CloseHandle(handle)
	return true, nil
}

func SetCurrentUserRun(name, command string) error {
	key, _, err := registry.CreateKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()
	return key.SetStringValue(name, command)
}

func DeleteCurrentUserRun(name string) error {
	key, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, registry.SET_VALUE)
	if err != nil {
		if err == registry.ErrNotExist {
			return nil
		}
		return err
	}
	defer key.Close()
	if err := key.DeleteValue(name); err != nil && err != registry.ErrNotExist {
		return err
	}
	return nil
}

func CurrentUserRunExists(name string) (bool, error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, registry.QUERY_VALUE)
	if err != nil {
		if err == registry.ErrNotExist {
			return false, nil
		}
		return false, err
	}
	defer key.Close()
	_, _, err = key.GetStringValue(name)
	if err != nil {
		if err == registry.ErrNotExist {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func QuoteCommand(exePath string, args ...string) string {
	command := fmt.Sprintf(`"%s"`, exePath)
	for _, arg := range args {
		command += " " + quoteArg(arg)
	}
	return command
}

func quoteArg(arg string) string {
	if arg == "" {
		return `""`
	}
	escaped := ""
	backslashes := 0
	for _, r := range arg {
		switch r {
		case '\\':
			backslashes++
			escaped += `\`
		case '"':
			escaped += strings.Repeat(`\`, backslashes+1) + `"`
			backslashes = 0
		default:
			backslashes = 0
			escaped += string(r)
		}
	}
	if strings.ContainsAny(arg, " \t\"") {
		return `"` + escaped + `"`
	}
	return escaped
}

func TrayRunning() (bool, error) {
	return MutexExists(webShareTrayMutexName)
}

func StopTray() error {
	ptr, err := windows.UTF16PtrFromString(webShareTrayQuitEventName)
	if err != nil {
		return err
	}
	handle, err := windows.OpenEvent(windows.EVENT_MODIFY_STATE, false, ptr)
	if err == nil {
		_ = windows.SetEvent(handle)
		_ = windows.CloseHandle(handle)
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		running, checkErr := TrayRunning()
		if checkErr != nil {
			return checkErr
		}
		if !running {
			return nil
		}
		time.Sleep(120 * time.Millisecond)
	}
	return nil
}
