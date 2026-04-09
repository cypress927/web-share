//go:build windows

package shell

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
)

const (
	fileMenuKey         = `Software\Classes\*\shell\web-share`
	folderMenuKey       = `Software\Classes\Directory\shell\web-share`
	fileCommandStoreKey = `Software\Classes\WebShare.FileContextMenu`
	fileReadOnlyKey     = `Software\Classes\WebShare.FileContextMenu\shell\readonly`
	fileReadOnlyCmd     = `Software\Classes\WebShare.FileContextMenu\shell\readonly\command`
	folderCommandStore  = `Software\Classes\WebShare.DirectoryContextMenu`
	folderReadOnlyKey   = `Software\Classes\WebShare.DirectoryContextMenu\shell\readonly`
	folderReadOnlyCmd   = `Software\Classes\WebShare.DirectoryContextMenu\shell\readonly\command`
	folderPasswordKey   = `Software\Classes\WebShare.DirectoryContextMenu\shell\password`
	folderPasswordCmd   = `Software\Classes\WebShare.DirectoryContextMenu\shell\password\command`
)

func InstallContextMenu(exePath string) error {
	return InstallContextMenuWithLanguage(exePath, "en-US")
}

func InstallContextMenuWithLanguage(exePath, lang string) error {
	menu := contextMenuTexts(lang)
	readOnlyCommand := fmt.Sprintf(`"%s" enqueue "%%1"`, exePath)
	passwordCommand, err := buildPasswordCommand(exePath, lang)
	if err != nil {
		return err
	}

	_ = removeLegacyPromptScript()
	_ = UninstallContextMenu()

	if err := createKeyWithValues(fileMenuKey, map[string]string{
		"MUIVerb":                menu.rootVerb,
		"Icon":                   exePath,
		"ExtendedSubCommandsKey": `WebShare.FileContextMenu`,
	}); err != nil {
		return err
	}
	if err := createKeyWithValues(folderMenuKey, map[string]string{
		"MUIVerb":                menu.rootVerb,
		"Icon":                   exePath,
		"ExtendedSubCommandsKey": `WebShare.DirectoryContextMenu`,
	}); err != nil {
		return err
	}
	if err := createKeyWithValues(fileReadOnlyKey, map[string]string{
		"MUIVerb": menu.readOnlyVerb,
	}); err != nil {
		return err
	}
	if err := createKeyWithValues(fileReadOnlyCmd, map[string]string{
		"": readOnlyCommand,
	}); err != nil {
		return err
	}
	if err := createKeyWithValues(folderReadOnlyKey, map[string]string{
		"MUIVerb": menu.readOnlyVerb,
	}); err != nil {
		return err
	}
	if err := createKeyWithValues(folderReadOnlyCmd, map[string]string{
		"": readOnlyCommand,
	}); err != nil {
		return err
	}
	if err := createKeyWithValues(folderPasswordKey, map[string]string{
		"MUIVerb": menu.passwordVerb,
	}); err != nil {
		return err
	}
	if err := createKeyWithValues(folderPasswordCmd, map[string]string{
		"": passwordCommand,
	}); err != nil {
		return err
	}
	return nil
}

func buildPasswordCommand(exePath, lang string) (string, error) {
	return fmt.Sprintf(`"%s" prompt-share -lang "%s" "%%1"`, exePath, normalizePromptLanguage(lang)), nil
}

func UninstallContextMenu() error {
	keys := []string{
		fileMenuKey,
		folderMenuKey,
		fileCommandStoreKey,
		folderCommandStore,
	}
	for _, keyPath := range keys {
		_ = deleteKeyTree(keyPath)
	}
	_ = removeLegacyPromptScript()
	return nil
}

func ContextMenuInstalled() bool {
	if exists, err := registryKeyExists(fileMenuKey); err != nil || !exists {
		return false
	}
	exists, err := registryKeyExists(folderMenuKey)
	return err == nil && exists
}

func createKeyWithValues(path string, values map[string]string) error {
	key, _, err := registry.CreateKey(registry.CURRENT_USER, path, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()
	for name, value := range values {
		if err := key.SetStringValue(name, value); err != nil {
			return err
		}
	}
	return nil
}

func registryKeyExists(path string) (bool, error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, path, registry.READ)
	if err != nil {
		if err == registry.ErrNotExist {
			return false, nil
		}
		return false, err
	}
	defer key.Close()
	return true, nil
}

func deleteKeyTree(path string) error {
	key, err := registry.OpenKey(registry.CURRENT_USER, path, registry.ENUMERATE_SUB_KEYS|registry.QUERY_VALUE)
	if err != nil {
		if err == registry.ErrNotExist {
			return nil
		}
		return err
	}
	names, err := key.ReadSubKeyNames(-1)
	key.Close()
	if err != nil {
		return err
	}
	for _, name := range names {
		if err := deleteKeyTree(path + `\` + name); err != nil {
			return err
		}
	}
	if err := registry.DeleteKey(registry.CURRENT_USER, path); err != nil && err != registry.ErrNotExist {
		return err
	}
	return nil
}

type menuTextSet struct {
	rootVerb     string
	readOnlyVerb string
	passwordVerb string
}

func contextMenuTexts(lang string) menuTextSet {
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(lang)), "zh") {
		return menuTextSet{
			rootVerb:     "通过 Web 分享",
			readOnlyVerb: "只读分享",
			passwordVerb: "设置上传密码后分享",
		}
	}
	return menuTextSet{
		rootVerb:     "Share via Web",
		readOnlyVerb: "Read-Only Share",
		passwordVerb: "Share with Upload Password",
	}
}

func normalizePromptLanguage(lang string) string {
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(lang)), "zh") {
		return "zh-CN"
	}
	return "en-US"
}

func removeLegacyPromptScript() error {
	baseDir := os.Getenv("LOCALAPPDATA")
	if strings.TrimSpace(baseDir) == "" {
		var err error
		baseDir, err = os.UserConfigDir()
		if err != nil {
			return err
		}
	}
	scriptPath := filepath.Join(baseDir, "WebShare", "prompt-share.vbs")
	if err := os.Remove(scriptPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
