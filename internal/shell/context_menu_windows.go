//go:build windows

package shell

import (
	"fmt"
	"os/exec"
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
	passwordCommand := buildPasswordCommand(exePath)

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

func buildPasswordCommand(exePath string) string {
	return fmt.Sprintf(
		`powershell.exe -NoProfile -WindowStyle Hidden -Command "Add-Type -AssemblyName Microsoft.VisualBasic; $p=[Microsoft.VisualBasic.Interaction]::InputBox('请输入上传密码。留空则取消分享。','Web Share 上传密码',''); if ([string]::IsNullOrWhiteSpace($p)) { exit 0 }; Start-Process -WindowStyle Hidden -FilePath '%s' -ArgumentList @('enqueue','-password',$p,'%s')"`,
		exePath,
		`%1`,
	)
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
