//go:build windows

package windowssvc

import (
	"errors"
	"fmt"
	"os/exec"

	"golang.org/x/sys/windows/svc"

	"web-share/internal/manager"
	"web-share/internal/shell"
	"web-share/internal/tray"
)

const serviceName = "WebShareManager"

func RunService() error {
	isService, err := svc.IsWindowsService()
	if err != nil {
		return err
	}
	if !isService {
		return manager.Run(manager.DefaultConfig())
	}

	return svc.Run(serviceName, &serviceHandler{})
}

func Install(exePath string) error {
	if err := exec.Command("sc.exe", "create", serviceName, "binPath=", fmt.Sprintf(`"%s" service`, exePath), "start=", "auto").Run(); err != nil {
		return fmt.Errorf("create service: %w", err)
	}
	_ = exec.Command("sc.exe", "description", serviceName, "Web Share background manager").Run()
	_ = shell.SetCurrentUserRun("WebShareTray", fmt.Sprintf(`"%s" tray`, exePath))
	_ = exec.Command("sc.exe", "start", serviceName).Run()
	_ = tray.EnsureStarted()
	return nil
}

func Uninstall() error {
	_ = exec.Command("sc.exe", "stop", serviceName).Run()
	if err := exec.Command("sc.exe", "delete", serviceName).Run(); err != nil {
		return fmt.Errorf("delete service: %w", err)
	}
	_ = shell.DeleteCurrentUserRun("WebShareTray")
	return nil
}

type serviceHandler struct{}

func (s *serviceHandler) Execute(_ []string, requests <-chan svc.ChangeRequest, changes chan<- svc.Status) (bool, uint32) {
	const accepted = svc.AcceptStop | svc.AcceptShutdown
	changes <- svc.Status{State: svc.StartPending}

	done := make(chan error, 1)
	go func() {
		done <- manager.Run(manager.DefaultConfig())
	}()

	changes <- svc.Status{State: svc.Running, Accepts: accepted}

	for {
		select {
		case change := <-requests:
			switch change.Cmd {
			case svc.Stop, svc.Shutdown:
				changes <- svc.Status{State: svc.StopPending}
				return false, 0
			case svc.Interrogate:
				changes <- change.CurrentStatus
			}
		case err := <-done:
			if err != nil && !errors.Is(err, exec.ErrNotFound) {
				return false, 1
			}
			return false, 0
		}
	}
}
