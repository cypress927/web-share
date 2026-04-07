package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"web-share/internal/manager"
	"web-share/internal/shell"
	"web-share/internal/tray"
	"web-share/internal/windowssvc"
)

func Run() error {
	if len(os.Args) < 2 {
		printUsage()
		return nil
	}

	switch os.Args[1] {
	case "share", "enqueue":
		return runEnqueue(os.Args[2:])
	case "run-manager":
		return manager.Run(manager.DefaultConfig())
	case "tray":
		if runtime.GOOS != "windows" {
			return errors.New("tray mode is only supported on Windows")
		}
		return tray.Run()
	case "service":
		if runtime.GOOS != "windows" {
			return errors.New("service mode is only supported on Windows")
		}
		return windowssvc.RunService()
	case "install-context-menu":
		return runInstallContextMenu(os.Args[2:])
	case "uninstall-context-menu":
		return runUninstallContextMenu()
	case "install-service":
		return runInstallService(os.Args[2:])
	case "uninstall-service":
		return runUninstallService()
	case "-h", "--help", "help":
		printUsage()
		return nil
	default:
		if len(os.Args) == 2 {
			return runEnqueue(os.Args[1:])
		}

		return fmt.Errorf("unknown command: %s", os.Args[1])
	}
}

func runEnqueue(args []string) error {
	fs := flag.NewFlagSet("enqueue", flag.ContinueOnError)
	password := fs.String("password", "", "Upload password for shared folders")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if fs.NArg() != 1 {
		return errors.New("enqueue requires exactly one file or directory path")
	}

	target, err := filepath.Abs(fs.Arg(0))
	if err != nil {
		return fmt.Errorf("resolve target path: %w", err)
	}

	if _, err := os.Stat(target); err != nil {
		return fmt.Errorf("stat target path: %w", err)
	}

	if err := ensureManager(); err != nil {
		return err
	}

	if runtime.GOOS == "windows" {
		_ = tray.EnsureStarted()
	}

	reqBody, err := json.Marshal(manager.CreateShareRequest{
		Path:     target,
		Password: *password,
	})
	if err != nil {
		return err
	}

	resp, err := http.Post(manager.LocalAPI("/api/shares"), "application/json", bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("enqueue share: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("enqueue share failed: %s", bytes.TrimSpace(body))
	}

	return nil
}

func ensureManager() error {
	if manager.IsReachable() {
		return nil
	}

	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable: %w", err)
	}

	if err := shell.StartDetached(exePath, "run-manager"); err != nil {
		return fmt.Errorf("start manager: %w", err)
	}

	deadline := time.Now().Add(6 * time.Second)
	for time.Now().Before(deadline) {
		if manager.IsReachable() {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}

	return errors.New("manager did not become ready in time")
}

func runInstallContextMenu(args []string) error {
	if runtime.GOOS != "windows" {
		return errors.New("context menu installation is only supported on Windows")
	}

	exePath, err := resolveExecutableArg(args)
	if err != nil {
		return err
	}

	return shell.InstallContextMenu(exePath)
}

func runUninstallContextMenu() error {
	if runtime.GOOS != "windows" {
		return errors.New("context menu installation is only supported on Windows")
	}

	return shell.UninstallContextMenu()
}

func runInstallService(args []string) error {
	if runtime.GOOS != "windows" {
		return errors.New("service installation is only supported on Windows")
	}

	exePath, err := resolveExecutableArg(args)
	if err != nil {
		return err
	}

	return windowssvc.Install(exePath)
}

func runUninstallService() error {
	if runtime.GOOS != "windows" {
		return errors.New("service installation is only supported on Windows")
	}

	return windowssvc.Uninstall()
}

func resolveExecutableArg(args []string) (string, error) {
	fs := flag.NewFlagSet("exe", flag.ContinueOnError)
	exe := fs.String("exe", "", "Path to web-share.exe")
	if err := fs.Parse(args); err != nil {
		return "", err
	}

	exePath := *exe
	if exePath == "" {
		currentExe, err := os.Executable()
		if err != nil {
			return "", fmt.Errorf("resolve executable: %w", err)
		}
		exePath = currentExe
	}

	exePath, err := filepath.Abs(exePath)
	if err != nil {
		return "", fmt.Errorf("resolve executable: %w", err)
	}

	if filepath.Ext(exePath) == "" {
		exePath += ".exe"
	}

	if _, err := os.Stat(exePath); err != nil {
		return "", fmt.Errorf("stat executable: %w", err)
	}

	return exePath, nil
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `web-share

Usage:
  web-share enqueue [-password secret] <path>
  web-share share [-password secret] <path>
  web-share tray
  web-share run-manager
  web-share install-context-menu [-exe C:\path\to\web-share.exe]
  web-share uninstall-context-menu
  web-share install-service [-exe C:\path\to\web-share.exe]
  web-share uninstall-service

Notes:
  - enqueue/share sends a new share task to the local manager.
  - The manager keeps all shares in one background process.
  - The tray icon opens the localhost management page.
  - File shares are always read-only.
  - Folder shares become upload-enabled only when -password is set.
`)
}
