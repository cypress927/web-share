//go:build windows

package shell

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"unicode/utf16"
)

const (
	fileMenuKey          = `HKCU\Software\Classes\*\shell\web-share`
	folderMenuKey        = `HKCU\Software\Classes\Directory\shell\web-share`
	fileCommandStoreKey  = `HKCU\Software\Classes\WebShare.FileContextMenu`
	fileReadOnlyKey      = `HKCU\Software\Classes\WebShare.FileContextMenu\shell\readonly`
	fileReadOnlyCmd      = `HKCU\Software\Classes\WebShare.FileContextMenu\shell\readonly\command`
	folderCommandStore   = `HKCU\Software\Classes\WebShare.DirectoryContextMenu`
	folderReadOnlyKey    = `HKCU\Software\Classes\WebShare.DirectoryContextMenu\shell\readonly`
	folderReadOnlyCmd    = `HKCU\Software\Classes\WebShare.DirectoryContextMenu\shell\readonly\command`
	folderPasswordKey    = `HKCU\Software\Classes\WebShare.DirectoryContextMenu\shell\password`
	folderPasswordCmd    = `HKCU\Software\Classes\WebShare.DirectoryContextMenu\shell\password\command`
)

func InstallContextMenu(exePath string) error {
	readOnlyCommand := fmt.Sprintf(`"%s" enqueue "%%1"`, exePath)
	passwordCommand, err := buildPasswordCommand(exePath)
	if err != nil {
		return err
	}

	commands := [][]string{
		{"delete", fileMenuKey, "/f"},
		{"delete", folderMenuKey, "/f"},
		{"delete", fileCommandStoreKey, "/f"},
		{"delete", folderCommandStore, "/f"},
		{"add", fileMenuKey, "/f"},
		{"add", fileMenuKey, "/v", "MUIVerb", "/d", "通过 Web 分享", "/f"},
		{"add", fileMenuKey, "/v", "Icon", "/d", exePath, "/f"},
		{"add", fileMenuKey, "/v", "ExtendedSubCommandsKey", "/d", "WebShare.FileContextMenu", "/f"},
		{"add", folderMenuKey, "/f"},
		{"add", folderMenuKey, "/v", "MUIVerb", "/d", "通过 Web 分享", "/f"},
		{"add", folderMenuKey, "/v", "Icon", "/d", exePath, "/f"},
		{"add", folderMenuKey, "/v", "ExtendedSubCommandsKey", "/d", "WebShare.DirectoryContextMenu", "/f"},
		{"add", fileReadOnlyKey, "/v", "MUIVerb", "/d", "只读分享", "/f"},
		{"add", fileReadOnlyCmd, "/ve", "/d", readOnlyCommand, "/f"},
		{"add", folderReadOnlyKey, "/v", "MUIVerb", "/d", "只读分享", "/f"},
		{"add", folderReadOnlyCmd, "/ve", "/d", readOnlyCommand, "/f"},
		{"add", folderPasswordKey, "/v", "MUIVerb", "/d", "设置上传密码后分享", "/f"},
		{"add", folderPasswordCmd, "/ve", "/d", passwordCommand, "/f"},
	}

	for _, args := range commands {
		if err := exec.Command("reg", args...).Run(); err != nil {
			if args[0] == "delete" {
				continue
			}
			return fmt.Errorf("run reg %v: %w", args, err)
		}
	}

	return nil
}

func buildPasswordCommand(exePath string) (string, error) {
	scriptPath, err := ensurePasswordPromptScript()
	if err != nil {
		return "", fmt.Errorf("create password prompt script: %w", err)
	}

	return fmt.Sprintf(`wscript.exe "%s" "%s" "%%1"`, scriptPath, exePath), nil
}

func UninstallContextMenu() error {
	commands := [][]string{
		{"delete", fileMenuKey, "/f"},
		{"delete", folderMenuKey, "/f"},
		{"delete", fileCommandStoreKey, "/f"},
		{"delete", folderCommandStore, "/f"},
	}

	for _, args := range commands {
		if err := exec.Command("reg", args...).Run(); err != nil {
			return fmt.Errorf("run reg %v: %w", args, err)
		}
	}

	return nil
}

func ensurePasswordPromptScript() (string, error) {
	baseDir := os.Getenv("LOCALAPPDATA")
	if baseDir == "" {
		var err error
		baseDir, err = os.UserConfigDir()
		if err != nil {
			return "", err
		}
	}

	scriptDir := filepath.Join(baseDir, "WebShare")
	if err := os.MkdirAll(scriptDir, 0o755); err != nil {
		return "", err
	}

	scriptPath := filepath.Join(scriptDir, "prompt-share.vbs")
	if err := os.WriteFile(scriptPath, encodeUTF16LE(passwordPromptVBScript), 0o644); err != nil {
		return "", err
	}

	return scriptPath, nil
}

const passwordPromptVBScript = `Dim exePath, targetPath, passwordText, shell, quote, commandText
If WScript.Arguments.Count < 2 Then
    WScript.Quit 1
End If

exePath = WScript.Arguments(0)
targetPath = WScript.Arguments(1)
passwordText = InputBox("请输入上传密码。留空则取消分享。", "Web Share 上传密码", "")

If Len(Trim(passwordText)) = 0 Then
    WScript.Quit 0
End If

Set shell = CreateObject("WScript.Shell")
quote = Chr(34)
commandText = quote & exePath & quote & " enqueue -password " & quote & Replace(passwordText, quote, quote & quote) & quote & " " & quote & targetPath & quote
shell.Run commandText, 0, False
`

func encodeUTF16LE(text string) []byte {
	encoded := utf16.Encode([]rune(text))
	buf := make([]byte, 2, 2+len(encoded)*2)
	buf[0] = 0xFF
	buf[1] = 0xFE
	for _, r := range encoded {
		buf = append(buf, byte(r), byte(r>>8))
	}
	return buf
}
