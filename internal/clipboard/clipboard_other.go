//go:build !windows

package clipboard

import "errors"

type Snapshot struct {
	Kind      string
	Name      string
	Text      string
	ImageData []byte
	MimeType  string
}

func CaptureSnapshot() (*Snapshot, error) {
	return nil, errors.New("clipboard sharing is only supported on Windows")
}
