//go:build windows

package shell

import (
	"os/exec"
	"strings"
	"syscall"

	"golang.org/x/sys/windows"
)

type mutexHandle = windows.Handle

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
	return exec.Command("reg", "add", `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`, "/v", name, "/d", command, "/f").Run()
}

func DeleteCurrentUserRun(name string) error {
	err := exec.Command("reg", "delete", `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`, "/v", name, "/f").Run()
	if err != nil && strings.Contains(strings.ToLower(err.Error()), "exit status 1") {
		return nil
	}
	return err
}
