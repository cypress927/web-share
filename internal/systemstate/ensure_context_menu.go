package systemstate

import "web-share/internal/logx"

func (s *Service) EnsureContextMenuInstalled(exePath, lang string) OperationResult {
	if s.ContextMenu == nil {
		return Failure("context menu port is not configured", "missing context menu port")
	}
	state, err := s.ContextMenu.Inspect(exePath)
	if err != nil {
		return logInspectError(s.Logger, "context menu inspect failed", err, logx.Field{Key: "exePath", Value: exePath})
	}
	if state.Installed && !state.Dirty {
		if len(state.Warnings) > 0 {
			s.Logger.Warn("context menu already installed with warnings", logx.Field{Key: "warnings", Value: state.Warnings})
		} else {
			s.Logger.Info("context menu already installed", logx.Field{Key: "exePath", Value: exePath})
		}
		return Success("Context menu is already installed.", false, state.Warnings...)
	}
	if state.Dirty {
		s.Logger.Warn("context menu dirty state detected before install", logx.Field{Key: "warnings", Value: state.Warnings})
		if err := s.ContextMenu.Remove(); err != nil {
			s.Logger.Error("context menu cleanup failed", logx.Field{Key: "error", Value: err.Error()})
			return Failure("Failed to clean existing context menu state.", err.Error())
		}
	}
	if err := s.ContextMenu.Install(exePath, lang); err != nil {
		s.Logger.Error("context menu install failed", logx.Field{Key: "exePath", Value: exePath}, logx.Field{Key: "error", Value: err.Error()})
		return Failure("Failed to install context menu.", err.Error())
	}
	verify, err := s.ContextMenu.Inspect(exePath)
	if err != nil {
		s.Logger.Error("context menu recheck failed", logx.Field{Key: "error", Value: err.Error()})
		return Failure("Failed to verify context menu state.", err.Error())
	}
	if !verify.Installed || verify.Dirty {
		s.Logger.Error("context menu install verification failed", logx.Field{Key: "installed", Value: verify.Installed}, logx.Field{Key: "dirty", Value: verify.Dirty}, logx.Field{Key: "warnings", Value: verify.Warnings})
		return Failure("Context menu did not reach the expected installed state.", "verification failed")
	}
	warnings := cloneWarnings(state.Warnings, verify.Warnings)
	s.Logger.Audit("context menu ensured installed", logx.Field{Key: "exePath", Value: exePath}, logx.Field{Key: "warnings", Value: warnings})
	return Success("Context menu installed.", true, warnings...)
}

func (s *Service) EnsureContextMenuRemoved(exePath string) OperationResult {
	if s.ContextMenu == nil {
		return Failure("context menu port is not configured", "missing context menu port")
	}
	state, err := s.ContextMenu.Inspect(exePath)
	if err != nil {
		return logInspectError(s.Logger, "context menu inspect failed before remove", err)
	}
	if !state.Installed && !state.Dirty {
		s.Logger.Warn("context menu already removed", logx.Field{Key: "warnings", Value: state.Warnings})
		return unchangedWithDefaultWarning("Context menu is already removed.", "Context menu entries were already missing.", state.Warnings)
	}
	if err := s.ContextMenu.Remove(); err != nil {
		s.Logger.Error("context menu remove failed", logx.Field{Key: "error", Value: err.Error()})
		return Failure("Failed to remove context menu.", err.Error())
	}
	verify, err := s.ContextMenu.Inspect(exePath)
	if err != nil {
		s.Logger.Error("context menu remove recheck failed", logx.Field{Key: "error", Value: err.Error()})
		return Failure("Failed to verify context menu removal.", err.Error())
	}
	if verify.Installed || verify.Dirty {
		s.Logger.Error("context menu remove verification failed", logx.Field{Key: "installed", Value: verify.Installed}, logx.Field{Key: "dirty", Value: verify.Dirty})
		return Failure("Context menu did not reach the expected removed state.", "verification failed")
	}
	warnings := cloneWarnings(state.Warnings, verify.Warnings)
	s.Logger.Audit("context menu ensured removed", logx.Field{Key: "warnings", Value: warnings})
	return Success("Context menu removed.", true, warnings...)
}
