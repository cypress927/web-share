package systemstate

import "web-share/internal/logx"

func (s *Service) EnsureManagerRunning(exePath string) OperationResult {
	if s.Manager == nil {
		return Failure("manager port is not configured", "missing manager port")
	}
	state, err := s.Manager.Inspect()
	if err != nil {
		return logInspectError(s.Logger, "manager inspect failed", err)
	}
	if state.Installed && !state.Dirty {
		return Success("Manager is already running.", false, state.Warnings...)
	}
	if err := s.Manager.Start(exePath); err != nil {
		s.Logger.Error("manager start failed", logx.Field{Key: "error", Value: err.Error()})
		return Failure("Failed to start manager.", err.Error())
	}
	verify, err := s.Manager.Inspect()
	if err != nil {
		s.Logger.Error("manager start recheck failed", logx.Field{Key: "error", Value: err.Error()})
		return Failure("Failed to verify manager state.", err.Error())
	}
	if !verify.Installed || verify.Dirty {
		return Failure("Manager did not reach the expected running state.", "verification failed")
	}
	return Success("Manager started.", true, cloneWarnings(state.Warnings, verify.Warnings)...)
}

func (s *Service) EnsureManagerStopped() OperationResult {
	if s.Manager == nil {
		return Failure("manager port is not configured", "missing manager port")
	}
	state, err := s.Manager.Inspect()
	if err != nil {
		return logInspectError(s.Logger, "manager inspect failed before stop", err)
	}
	if !state.Installed && !state.Dirty {
		return unchangedWithDefaultWarning("Manager is already stopped.", "Manager was already stopped.", state.Warnings)
	}
	if err := s.Manager.Stop(); err != nil {
		s.Logger.Error("manager stop failed", logx.Field{Key: "error", Value: err.Error()})
		return Failure("Failed to stop manager.", err.Error())
	}
	return Success("Manager stop requested.", true, cloneWarnings(state.Warnings)...)
}
