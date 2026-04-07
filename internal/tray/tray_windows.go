//go:build windows

package tray

import (
	"os"

	"github.com/getlantern/systray"

	"web-share/internal/manager"
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
	systray.SetTitle("Web Share")
	systray.SetTooltip("Web Share Manager")

	openItem := systray.AddMenuItem("打开管理页面", "Open local manager page")
	quitItem := systray.AddMenuItem("退出托盘", "Exit tray")

	go func() {
		for {
			select {
			case <-openItem.ClickedCh:
				_ = shell.OpenBrowser(manager.LocalManageURL())
			case <-quitItem.ClickedCh:
				systray.Quit()
				return
			}
		}
	}()
}
