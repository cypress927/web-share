package systemstate

func (s *Service) Snapshot(exePath, taskName, command string) (StatusSnapshot, error) {
	var snapshot StatusSnapshot
	warnings := make([]string, 0, 8)

	if s.Manager != nil {
		state, err := s.Manager.Inspect()
		if err != nil {
			return snapshot, err
		}
		snapshot.ManagerRunning = state.Installed
		warnings = append(warnings, state.Warnings...)
	}
	if s.ContextMenu != nil {
		state, err := s.ContextMenu.Inspect(exePath)
		if err != nil {
			return snapshot, err
		}
		snapshot.ContextMenuInstalled = state.Installed
		snapshot.ContextMenuDirty = state.Dirty
		warnings = append(warnings, state.Warnings...)
	}
	if s.Autostart != nil {
		state, err := s.Autostart.Inspect(taskName, command)
		if err != nil {
			return snapshot, err
		}
		snapshot.AutostartEnabled = state.Installed
		snapshot.AutostartDirty = state.Dirty
		warnings = append(warnings, state.Warnings...)
	}
	if s.Tray != nil {
		state, err := s.Tray.Inspect()
		if err != nil {
			return snapshot, err
		}
		snapshot.TrayRunning = state.Installed
		snapshot.TrayDirty = state.Dirty
		warnings = append(warnings, state.Warnings...)
	}
	snapshot.Warnings = warnings
	return snapshot, nil
}
