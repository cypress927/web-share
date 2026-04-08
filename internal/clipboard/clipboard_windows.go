//go:build windows

package clipboard

import (
	"encoding/base64"
	"errors"
	"os/exec"
	"strings"
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
}

func CaptureSnapshot() (*Snapshot, error) {
	if image, err := captureImage(); err == nil {
		return image, nil
	}

	text, err := captureText()
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(text) == "" {
		return nil, errors.New("剪贴板中没有可分享的文本或图片")
	}

	return &Snapshot{
		Kind: kindText,
		Name: "剪贴板文本",
		Text: text,
	}, nil
}

func captureText() (string, error) {
	out, err := exec.Command("powershell.exe", "-NoProfile", "-Command", "Get-Clipboard -Raw").CombinedOutput()
	if err != nil {
		return "", errors.New("读取剪贴板文本失败")
	}
	return strings.ReplaceAll(string(out), "\r\n", "\n"), nil
}

func captureImage() (*Snapshot, error) {
	script := "$ErrorActionPreference='Stop'; Add-Type -AssemblyName System.Windows.Forms; Add-Type -AssemblyName System.Drawing; $img = [Windows.Forms.Clipboard]::GetImage(); if ($null -eq $img) { exit 2 }; $ms = New-Object System.IO.MemoryStream; $img.Save($ms, [System.Drawing.Imaging.ImageFormat]::Png); [Convert]::ToBase64String($ms.ToArray())"
	out, err := exec.Command("powershell.exe", "-NoProfile", "-Command", script).CombinedOutput()
	if err != nil {
		return nil, errors.New("读取剪贴板图片失败")
	}
	raw := strings.TrimSpace(string(out))
	if raw == "" {
		return nil, errors.New("剪贴板中没有图片")
	}
	data, err := base64.StdEncoding.DecodeString(raw)
	if err != nil || len(data) == 0 {
		return nil, errors.New("解析剪贴板图片失败")
	}

	return &Snapshot{
		Kind:      kindImage,
		Name:      "剪贴板图片",
		ImageData: data,
		MimeType:  "image/png",
	}, nil
}
