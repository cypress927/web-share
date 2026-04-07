//go:build !windows

package windowssvc

import "errors"

func RunService() error {
	return errors.New("Windows only")
}

func Install(string) error {
	return errors.New("Windows only")
}

func Uninstall() error {
	return errors.New("Windows only")
}
