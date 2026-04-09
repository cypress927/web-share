//go:build !windows

package shell

import "errors"

func PromptForUploadPassword(string) (string, bool, error) {
	return "", false, errors.New("native password prompt is only supported on Windows")
}
