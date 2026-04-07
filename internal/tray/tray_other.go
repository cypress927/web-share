//go:build !windows

package tray

import "errors"

func Run() error {
	return errors.New("tray mode is only supported on Windows")
}

func EnsureStarted() error {
	return errors.New("tray mode is only supported on Windows")
}

func ShutdownProgram() error {
	return errors.New("tray mode is only supported on Windows")
}
