package systemstate

import "web-share/internal/logx"

type ContextMenuPort interface {
	Inspect(exePath string) (InspectResult, error)
	Install(exePath, lang string) error
	Remove() error
}

type AutostartPort interface {
	Inspect(taskName, command string) (InspectResult, error)
	Enable(taskName, command string) error
	Disable(taskName string) error
}

type TrayPort interface {
	Inspect() (InspectResult, error)
	Start(exePath string) error
	Stop() error
}

type ProgramPort interface {
	Inspect() (InspectResult, error)
	Stop() error
}

type ManagerPort interface {
	Inspect() (InspectResult, error)
	Start(exePath string) error
	Stop() error
}

type Logger = logx.Logger
