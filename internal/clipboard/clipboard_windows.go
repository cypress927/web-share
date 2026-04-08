//go:build windows

package clipboard

import (
	"encoding/base64"
	"errors"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

const (
	kindText  = "clipboard_text"
	kindImage = "clipboard_image"
)

type Snapshot struct {
	Kind      string
	Name      string
	Text      string
	ImageData []byte
	MimeType  string
	Paths     []string
}

func CaptureSnapshot() (*Snapshot, error) {
	if paths, err := captureFiles(); err == nil && len(paths) > 0 {
		return &Snapshot{
			Name:  "Clipboard Files",
			Paths: paths,
		}, nil
	}

	if image, err := captureImage(); err == nil {
		return image, nil
	}

	text, err := captureText()
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(text) == "" {
		return nil, errors.New("clipboard has no shareable text or image")
	}

	return &Snapshot{
		Kind: kindText,
		Name: makeClipboardTextTitle(text),
		Text: text,
	}, nil
}

func captureFiles() ([]string, error) {
	script := "[Console]::OutputEncoding = [System.Text.UTF8Encoding]::new($false); $OutputEncoding = [Console]::OutputEncoding; $items = Get-Clipboard -Format FileDropList -ErrorAction Stop; if ($null -eq $items -or $items.Count -eq 0) { exit 2 }; $items | ForEach-Object { $_.ToString() }"
	out, err := runHiddenPowerShell(script)
	if err != nil {
		return nil, errors.New("failed to read clipboard files")
	}

	lines := strings.Split(strings.ReplaceAll(string(out), "\r\n", "\n"), "\n")
	paths := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		paths = append(paths, trimmed)
	}
	if len(paths) == 0 {
		return nil, errors.New("clipboard has no files")
	}
	return paths, nil
}

func captureText() (string, error) {
	script := "[Console]::OutputEncoding = [System.Text.UTF8Encoding]::new($false); $OutputEncoding = [Console]::OutputEncoding; Get-Clipboard -Raw"
	out, err := runHiddenPowerShell(script)
	if err != nil {
		return "", errors.New("failed to read clipboard text")
	}
	return strings.ReplaceAll(string(out), "\r\n", "\n"), nil
}

func captureImage() (*Snapshot, error) {
	script := "$ErrorActionPreference='Stop'; Add-Type -AssemblyName System.Windows.Forms; Add-Type -AssemblyName System.Drawing; $img = [Windows.Forms.Clipboard]::GetImage(); if ($null -eq $img) { exit 2 }; $ms = New-Object System.IO.MemoryStream; $img.Save($ms, [System.Drawing.Imaging.ImageFormat]::Png); [Convert]::ToBase64String($ms.ToArray())"
	out, err := runHiddenPowerShell(script)
	if err != nil {
		return nil, errors.New("failed to read clipboard image")
	}
	raw := strings.TrimSpace(string(out))
	if raw == "" {
		return nil, errors.New("clipboard has no image")
	}
	data, err := base64.StdEncoding.DecodeString(raw)
	if err != nil || len(data) == 0 {
		return nil, errors.New("failed to parse clipboard image")
	}

	return &Snapshot{
		Kind:      kindImage,
		Name:      makeClipboardImageTitle(data),
		ImageData: data,
		MimeType:  "image/png",
	}, nil
}

func runHiddenPowerShell(script string) ([]byte, error) {
	cmd := exec.Command("powershell.exe", "-NoProfile", "-Command", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}
	return cmd.CombinedOutput()
}

func makeClipboardTextTitle(text string) string {
	clean := strings.ReplaceAll(strings.TrimSpace(text), "\r\n", "\n")
	if clean == "" {
		return "Clipboard Text"
	}

	first := clean
	if idx := strings.IndexByte(clean, '\n'); idx >= 0 {
		first = clean[:idx]
	}
	first = strings.TrimSpace(first)
	if first == "" {
		first = clean
	}

	runes := []rune(first)
	const maxRunes = 20
	if len(runes) > maxRunes {
		first = string(runes[:maxRunes]) + "..."
	}
	return "Text: " + first
}

func makeClipboardImageTitle(_ []byte) string {
	now := time.Now().Format("2006-01-02 15:04")
	return "Image: " + now
}
