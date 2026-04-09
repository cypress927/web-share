//go:build windows

package manager

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

const localeNameMaxLength = 85

var (
	kernel32DLL                  = windows.NewLazySystemDLL("kernel32.dll")
	procGetUserDefaultLocaleName = kernel32DLL.NewProc("GetUserDefaultLocaleName")
)

func detectSystemLanguage() string {
	buf := make([]uint16, localeNameMaxLength)
	r1, _, _ := procGetUserDefaultLocaleName.Call(
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(len(buf)),
	)
	if r1 == 0 {
		return langEN
	}
	return normalizeLanguage(windows.UTF16ToString(buf))
}
