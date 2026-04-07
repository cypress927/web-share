//go:build windows

package main

import (
	"fmt"
	"os"
	"time"

	"web-share/internal/notify"
)

func main() {
	title := "Web Share Toast Test"
	message := "如果你看到了这条通知，说明 Windows Toast 基本可用。"
	if len(os.Args) > 1 {
		message = os.Args[1]
	}

	if err := notify.Info(title, message); err != nil {
		fmt.Fprintf(os.Stderr, "toast notify failed: %v\n", err)
		os.Exit(1)
	}

	// Give the notification process a brief moment before the test exits.
	time.Sleep(800 * time.Millisecond)
	fmt.Println("toast notification dispatched")
}
