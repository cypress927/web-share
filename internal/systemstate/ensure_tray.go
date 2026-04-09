package systemstate

import "web-share/internal/logx"

func (s *Service) EnsureTrayRunning(exePath string) OperationResult {
	if s.Tray == nil {
		return Failure("tray port is not configured", "missing tray port")
	}
	state, err := s.Tray.Inspect()
	if err != nil {
		return logInspectError(s.Logger, "tray inspect failed", err)
	}
	if state.Installed && !state.Dirty {
		return Success("Tray is already running.", false, state.Warnings...)
	}
	if err := s.Tray.Start(exePath); err != nil {
		s.Logger.Error("tray start failed", logx.Field{Key: "error", Value: err.Error()})
		return Failure("Failed to start tray.", err.Error())
	}
	verify, err := s.Tray.Inspect()
	if err != nil {
		s.Logger.Error("tray start recheck failed", logx.Field{Key: "error", Value: err.Error()})
		return Failure("Failed to verify tray state.", err.Error())
	}
	if !verify.Installed || verify.Dirty {
		return Failure("Tray did not reach the expected running state.", "verification failed")
	}
	return Success("Tray started.", true, cloneWarnings(state.Warnings, verify.Warnings)...)
}

func (s *Service) EnsureTrayStopped() OperationResult {
	if s.Tray == nil {
		return Failure("tray port is not configured", "missing tray port")
	}
	state, err := s.Tray.Inspect()
	if err != nil {
		return logInspectError(s.Logger, "tray inspect failed before stop", err)
	}
	if !state.Installed && !state.Dirty {
		return unchangedWithDefaultWarning("Tray is already stopped.", "Tray was already stopped.", state.Warnings)
	}
	if err := s.Tray.Stop(); err != nil {
		s.Logger.Error("tray stop failed", logx.Field{Key: "error", Value: err.Error()})
		return Failure("Failed to stop tray.", err.Error())
	}
	verify, err := s.Tray.Inspect()
	if err != nil {
		s.Logger.Error("tray stop recheck failed", logx.Field{Key: "error", Value: err.Error()})
		return Failure("Failed to verify tray stop state.", err.Error())
	}
	if verify.Installed || verify.Dirty {
		return Failure("Tray did not reach the expected stopped state.", "verification failed")
	}
	return Success("Tray stopped.", true, cloneWarnings(state.Warnings, verify.Warnings)...)
}
