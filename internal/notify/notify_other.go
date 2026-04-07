//go:build !windows

package notify

func Info(string, string) error {
	return nil
}

func Error(string, string) error {
	return nil
}
