package systemstate

import "web-share/internal/logx"

func (s *Service) EnsureAutostartEnabled(taskName, command string) OperationResult {
	if s.Autostart == nil {
		return Failure("autostart port is not configured", "missing autostart port")
	}
	state, err := s.Autostart.Inspect(taskName, command)
	if err != nil {
		return logInspectError(s.Logger, "autostart inspect failed", err, logx.Field{Key: "taskName", Value: taskName})
	}
	if state.Installed && !state.Dirty {
		return Success("Auto start is already enabled.", false, state.Warnings...)
	}
	if err := s.Autostart.Enable(taskName, command); err != nil {
		s.Logger.Error("autostart enable failed", logx.Field{Key: "taskName", Value: taskName}, logx.Field{Key: "error", Value: err.Error()})
		return Failure("Failed to enable auto start.", err.Error())
	}
	verify, err := s.Autostart.Inspect(taskName, command)
	if err != nil {
		s.Logger.Error("autostart enable recheck failed", logx.Field{Key: "taskName", Value: taskName}, logx.Field{Key: "error", Value: err.Error()})
		return Failure("Failed to verify auto start state.", err.Error())
	}
	if !verify.Installed || verify.Dirty {
		return Failure("Auto start did not reach the expected enabled state.", "verification failed")
	}
	return Success("Auto start enabled.", true, cloneWarnings(state.Warnings, verify.Warnings)...)
}

func (s *Service) EnsureAutostartDisabled(taskName, command string) OperationResult {
	if s.Autostart == nil {
		return Failure("autostart port is not configured", "missing autostart port")
	}
	state, err := s.Autostart.Inspect(taskName, command)
	if err != nil {
		return logInspectError(s.Logger, "autostart inspect failed before disable", err, logx.Field{Key: "taskName", Value: taskName})
	}
	if !state.Installed && !state.Dirty {
		return unchangedWithDefaultWarning("Auto start is already disabled.", "Auto start entry was already missing.", state.Warnings)
	}
	if err := s.Autostart.Disable(taskName); err != nil {
		s.Logger.Error("autostart disable failed", logx.Field{Key: "taskName", Value: taskName}, logx.Field{Key: "error", Value: err.Error()})
		return Failure("Failed to disable auto start.", err.Error())
	}
	verify, err := s.Autostart.Inspect(taskName, command)
	if err != nil {
		s.Logger.Error("autostart disable recheck failed", logx.Field{Key: "taskName", Value: taskName}, logx.Field{Key: "error", Value: err.Error()})
		return Failure("Failed to verify auto start removal.", err.Error())
	}
	if verify.Installed || verify.Dirty {
		return Failure("Auto start did not reach the expected disabled state.", "verification failed")
	}
	return Success("Auto start disabled.", true, cloneWarnings(state.Warnings, verify.Warnings)...)
}
