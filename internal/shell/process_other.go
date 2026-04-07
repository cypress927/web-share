//go:build !windows

package shell

import "errors"

type mutexHandle struct{}

func StartDetached(string, ...string) error {
	return errors.New("Windows only")
}

func OpenBrowser(string) error {
	return errors.New("Windows only")
}

func AcquireMutex(string) (*mutexHandle, bool, error) {
	return nil, false, errors.New("Windows only")
}

func ReleaseMutex(*mutexHandle) error {
	return errors.New("Windows only")
}

func MutexExists(string) (bool, error) {
	return false, errors.New("Windows only")
}

func SetCurrentUserRun(string, string) error {
	return errors.New("Windows only")
}

func DeleteCurrentUserRun(string) error {
	return errors.New("Windows only")
}
