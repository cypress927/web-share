package systemstate

import "web-share/internal/logx"

func (s *Service) EnsureProgramStopped() OperationResult {
	if s.Program == nil {
		return Failure("program port is not configured", "missing program port")
	}
	state, err := s.Program.Inspect()
	if err != nil {
		return logInspectError(s.Logger, "program inspect failed before stop", err)
	}
	if !state.Installed && !state.Dirty {
		return unchangedWithDefaultWarning("Program is already stopped.", "Program was already stopped.", state.Warnings)
	}
	if err := s.Program.Stop(); err != nil {
		s.Logger.Error("program stop request failed", logx.Field{Key: "error", Value: err.Error()})
		return Failure("Failed to stop program.", err.Error())
	}
	return Success("Program stop requested.", true, cloneWarnings(state.Warnings)...)
}
