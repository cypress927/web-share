//go:build windows

package tray

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/getlantern/systray"
	"golang.org/x/sys/windows"

	"web-share/internal/assets"
	"web-share/internal/clipboard"
	"web-share/internal/manager"
	"web-share/internal/notify"
	"web-share/internal/shell"
)

const trayMutexName = `Global\WebShareTraySingleton`
const trayQuitEventName = `Global\WebShareTrayQuitEvent`

func Run() error {
	mutex, acquired, err := shell.AcquireMutex(trayMutexName)
	if err != nil {
		return err
	}
	if !acquired {
		return nil
	}
	defer shell.ReleaseMutex(mutex)

	quitEvent, err := createQuitEvent()
	if err != nil {
		return err
	}
	defer windows.CloseHandle(quitEvent)

	go func() {
		_, _ = windows.WaitForSingleObject(quitEvent, windows.INFINITE)
		systray.Quit()
	}()

	systray.Run(onReady, func() {})
	return nil
}

func EnsureStarted() error {
	locked, err := shell.MutexExists(trayMutexName)
	if err != nil {
		return err
	}
	if locked {
		return nil
	}

	exePath, err := os.Executable()
	if err != nil {
		return err
	}

	return shell.StartDetached(exePath, "tray")
}

func Restart() error {
	_ = signalQuitEvent()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		locked, err := shell.MutexExists(trayMutexName)
		if err != nil {
			return err
		}
		if !locked {
			break
		}
		time.Sleep(120 * time.Millisecond)
	}
	return EnsureStarted()
}

func onReady() {
	lang := manager.SystemDefaultLanguage()
	systray.SetIcon(assets.ShareICO)
	systray.SetTitle("Web Share")
	systray.SetTooltip("Web Share Manager")

	openItem := systray.AddMenuItem(trayMessage(lang, "menu_open_manage"), "Open local manager page")
	clipboardItem := systray.AddMenuItem(trayMessage(lang, "menu_share_clipboard"), "Share current clipboard text or image")
	quitItem := systray.AddMenuItem(trayMessage(lang, "menu_quit"), "Exit program")

	go func() {
		for {
			select {
			case <-openItem.ClickedCh:
				_ = shell.OpenBrowser(manager.LocalManageURL())
			case <-clipboardItem.ClickedCh:
				if err := shareClipboard(lang); err != nil {
					_ = notify.Error("Web Share", trayMessage(lang, "clipboard_failed")+err.Error())
				}
			case <-quitItem.ClickedCh:
				_ = ShutdownProgram()
				systray.Quit()
				return
			}
		}
	}()
}

func shareClipboard(lang string) error {
	if !manager.IsReachable() {
		return errors.New(trayMessage(lang, "manager_not_ready"))
	}

	snapshot, err := clipboard.CaptureSnapshot()
	if err != nil {
		return err
	}

	if len(snapshot.Paths) > 0 {
		created := 0
		for _, path := range snapshot.Paths {
			if _, err := createShare(manager.CreateShareRequest{Path: path}); err != nil {
				return err
			}
			created++
		}
		_ = notify.Info("Web Share", fmt.Sprintf(trayMessage(lang, "clipboard_added_n"), created))
		return nil
	}

	req := manager.CreateShareRequest{
		Kind:        snapshot.Kind,
		Name:        snapshot.Name,
		TextContent: snapshot.Text,
		MimeType:    snapshot.MimeType,
	}
	if len(snapshot.ImageData) > 0 {
		req.ImageBase64 = base64.StdEncoding.EncodeToString(snapshot.ImageData)
	}

	name, err := createShare(req)
	if err != nil {
		return err
	}
	if name == "" {
		name = snapshot.Name
	}
	_ = notify.Info("Web Share", trayMessage(lang, "share_added")+name)
	return nil
}

func createShare(req manager.CreateShareRequest) (string, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return "", err
	}

	resp, err := http.Post(manager.LocalAPI("/api/shares"), "application/json", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		if msg := strings.TrimSpace(string(raw)); msg != "" {
			return "", errors.New(msg)
		}
		return "", errors.New("failed to create share")
	}

	var result struct {
		Name string `json:"name"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&result)
	return result.Name, nil
}

func ShutdownProgram() error {
	req, err := http.NewRequest(http.MethodPost, manager.LocalAPI("/api/shutdown"), nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func trayMessage(lang, key string) string {
	zh := strings.HasPrefix(strings.ToLower(strings.TrimSpace(lang)), "zh")
	if zh {
		switch key {
		case "menu_open_manage":
			return "打开管理页面"
		case "menu_share_clipboard":
			return "分享当前剪贴板"
		case "menu_quit":
			return "退出程序"
		case "clipboard_failed":
			return "剪贴板分享失败："
		case "manager_not_ready":
			return "管理器未就绪"
		case "clipboard_added_n":
			return "已从剪贴板添加 %d 个分享"
		case "share_added":
			return "分享已添加："
		}
	}
	switch key {
	case "menu_open_manage":
		return "Open Manager"
	case "menu_share_clipboard":
		return "Share Clipboard"
	case "menu_quit":
		return "Exit Program"
	case "clipboard_failed":
		return "Clipboard share failed: "
	case "manager_not_ready":
		return "Manager is not ready"
	case "clipboard_added_n":
		return "Added %d shares from clipboard"
	case "share_added":
		return "Share added: "
	default:
		return ""
	}
}

func createQuitEvent() (windows.Handle, error) {
	ptr, err := windows.UTF16PtrFromString(trayQuitEventName)
	if err != nil {
		return 0, err
	}
	return windows.CreateEvent(nil, 0, 0, ptr)
}

func signalQuitEvent() error {
	ptr, err := windows.UTF16PtrFromString(trayQuitEventName)
	if err != nil {
		return err
	}
	handle, err := windows.OpenEvent(windows.EVENT_MODIFY_STATE, false, ptr)
	if err != nil {
		if err == windows.ERROR_FILE_NOT_FOUND {
			return nil
		}
		return err
	}
	defer windows.CloseHandle(handle)
	return windows.SetEvent(handle)
}
