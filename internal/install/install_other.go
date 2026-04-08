//go:build !windows

package install

import (
	"errors"
	"time"
)

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

func Install(InstallOptions) error {
	return errors.New("single-file installer is only supported on Windows")
}
func Start(StartOptions) error {
	return errors.New("single-file installer is only supported on Windows")
}
func Repair(InstallOptions) error {
	return errors.New("single-file installer is only supported on Windows")
}
func Uninstall(UninstallOptions) error {
	return errors.New("single-file installer is only supported on Windows")
}
func InstallStartupTask(string, string, string, bool, bool) error {
	return errors.New("single-file installer is only supported on Windows")
}
func UninstallStartupTask(string) error {
	return errors.New("single-file installer is only supported on Windows")
}
