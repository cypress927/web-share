//go:build !windows

package app

import "errors"

func runPromptShare([]string) error {
	return errors.New("prompt-share is only supported on Windows")
}
