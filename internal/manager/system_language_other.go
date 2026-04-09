//go:build !windows

package manager

func detectSystemLanguage() string {
	return langEN
}
