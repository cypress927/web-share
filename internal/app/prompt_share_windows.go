//go:build windows

package app

import (
	"flag"
	"fmt"

	"web-share/internal/shell"
)

func runPromptShare(args []string) error {
	fs := flag.NewFlagSet("prompt-share", flag.ContinueOnError)
	lang := fs.String("lang", "en-US", "Language for the native password prompt")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("prompt-share requires exactly one file or directory path")
	}

	password, ok, err := shell.PromptForUploadPassword(*lang)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	return runEnqueue([]string{"-password", password, fs.Arg(0)})
}
