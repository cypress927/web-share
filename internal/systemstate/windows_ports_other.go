//go:build !windows

package systemstate

func NewWindowsService(logger Logger) *Service {
	return NewService(logger)
}
