//go:build !windows

package shell

import "errors"

func InstallContextMenu(string) error {
	return errors.New("Windows only")
}

func UninstallContextMenu() error {
	return errors.New("Windows only")
}
