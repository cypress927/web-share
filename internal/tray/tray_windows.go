//go:build windows

package tray

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/getlantern/systray"

	"web-share/internal/assets"
	"web-share/internal/clipboard"
	"web-share/internal/manager"
	"web-share/internal/notify"
	"web-share/internal/shell"
)

const trayMutexName = `Global\WebShareTraySingleton`

func Run() error {
	mutex, acquired, err := shell.AcquireMutex(trayMutexName)
	if err != nil {
		return err
	}
	if !acquired {
		return nil
	}
	defer shell.ReleaseMutex(mutex)

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

func onReady() {
	systray.SetIcon(assets.ShareICO)
	systray.SetTitle("Web Share")
	systray.SetTooltip("Web Share Manager")

	openItem := systray.AddMenuItem("打开管理页面", "Open local manager page")
	clipboardItem := systray.AddMenuItem("分享当前剪贴板", "Share current clipboard text or image")
	quitItem := systray.AddMenuItem("退出程序", "Exit program")

	go func() {
		for {
			select {
			case <-openItem.ClickedCh:
				_ = shell.OpenBrowser(manager.LocalManageURL())
			case <-clipboardItem.ClickedCh:
				if err := shareClipboard(); err != nil {
					_ = notify.Error("Web Share", "剪贴板分享失败："+err.Error())
				}
			case <-quitItem.ClickedCh:
				_ = ShutdownProgram()
				systray.Quit()
				return
			}
		}
	}()
}

func shareClipboard() error {
	if !manager.IsReachable() {
		return errors.New("管理器未就绪")
	}

	snapshot, err := clipboard.CaptureSnapshot()
	if err != nil {
		return err
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

	body, err := json.Marshal(req)
	if err != nil {
		return err
	}

	resp, err := http.Post(manager.LocalAPI("/api/shares"), "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		if msg := strings.TrimSpace(string(raw)); msg != "" {
			return errors.New(msg)
		}
		return errors.New("创建分享失败")
	}

	var result struct {
		Name string `json:"name"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&result)
	if result.Name == "" {
		result.Name = snapshot.Name
	}
	_ = notify.Info("Web Share", "分享已添加："+result.Name)
	return nil
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
