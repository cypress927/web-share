package systemstate

import "web-share/internal/logx"

type Service struct {
	Logger      Logger
	ContextMenu ContextMenuPort
	Autostart   AutostartPort
	Tray        TrayPort
	Program     ProgramPort
	Manager     ManagerPort
}

func NewService(logger Logger) *Service {
	if logger == nil {
		logger = logx.New()
	}
	return &Service{Logger: logger}
}
