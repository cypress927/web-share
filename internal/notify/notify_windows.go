//go:build windows

package notify

import (
	"github.com/gen2brain/beeep"

	"web-share/internal/assets"
)

func Info(title, message string) error {
	return beeep.Notify(title, message, assets.SharePNG)
}

func Error(title, message string) error {
	return beeep.Alert(title, message, assets.SharePNG)
}
