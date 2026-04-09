package systemstate

type OperationResult struct {
	OK       bool
	Changed  bool
	Message  string
	Warnings []string
	Errors   []string
}

func Success(message string, changed bool, warnings ...string) OperationResult {
	return OperationResult{
		OK:       true,
		Changed:  changed,
		Message:  message,
		Warnings: append([]string(nil), warnings...),
	}
}

func Failure(message string, errs ...string) OperationResult {
	return OperationResult{
		OK:      false,
		Message: message,
		Errors:  append([]string(nil), errs...),
	}
}

type InspectResult struct {
	Installed bool
	Dirty     bool
	Warnings  []string
}

type StatusSnapshot struct {
	ManagerRunning       bool
	ContextMenuInstalled bool
	ContextMenuDirty     bool
	AutostartEnabled     bool
	AutostartDirty       bool
	TrayRunning          bool
	TrayDirty            bool
	Warnings             []string
}
