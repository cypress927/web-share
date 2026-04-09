package systemstate

import "web-share/internal/logx"

func cloneWarnings(parts ...[]string) []string {
	var warnings []string
	for _, part := range parts {
		if len(part) == 0 {
			continue
		}
		warnings = append(warnings, part...)
	}
	return warnings
}

func unchangedWithDefaultWarning(message, fallback string, warnings []string) OperationResult {
	cloned := cloneWarnings(warnings)
	if len(cloned) == 0 {
		cloned = append(cloned, fallback)
	}
	return Success(message, false, cloned...)
}

func logInspectError(logger Logger, message string, err error, fields ...logx.Field) OperationResult {
	fields = append(fields, logx.Field{Key: "error", Value: err.Error()})
	logger.Error(message, fields...)
	return Failure(humanizeInspectFailure(message), err.Error())
}

func humanizeInspectFailure(message string) string {
	switch message {
	case "context menu inspect failed", "context menu inspect failed before remove":
		return "Failed to inspect context menu state."
	case "autostart inspect failed", "autostart inspect failed before disable":
		return "Failed to inspect auto start state."
	case "tray inspect failed", "tray inspect failed before stop":
		return "Failed to inspect tray state."
	case "program inspect failed before stop":
		return "Failed to inspect program state."
	case "manager inspect failed", "manager inspect failed before stop":
		return "Failed to inspect manager state."
	default:
		return "Failed to inspect system state."
	}
}
