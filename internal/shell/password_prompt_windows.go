//go:build windows

package shell

import (
	"errors"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	credTypeGeneric               = 1
	credUIFlagsGenericCredentials = 0x00040000
	credUIFlagsAlwaysShowUI       = 0x00000080
	credUIFlagsDoNotPersist       = 0x00000002
	credUIFlagsExpectConfirmation = 0x00020000
	credUIMaxUsernameLength       = 513
	credUIMaxPasswordLength       = 256
)

type credUIInfo struct {
	cbSize         uint32
	hwndParent     windows.Handle
	pszMessageText *uint16
	pszCaptionText *uint16
	hbmBanner      windows.Handle
}

var (
	creduiDLL                 = windows.NewLazySystemDLL("credui.dll")
	procCredUIPromptForCredsW = creduiDLL.NewProc("CredUIPromptForCredentialsW")
)

func PromptForUploadPassword(lang string) (string, bool, error) {
	title, message := passwordPromptTexts(lang)
	titlePtr, err := windows.UTF16PtrFromString(title)
	if err != nil {
		return "", false, err
	}
	messagePtr, err := windows.UTF16PtrFromString(message)
	if err != nil {
		return "", false, err
	}
	targetPtr, err := windows.UTF16PtrFromString("Web Share")
	if err != nil {
		return "", false, err
	}

	ui := credUIInfo{
		cbSize:         uint32(unsafe.Sizeof(credUIInfo{})),
		pszMessageText: messagePtr,
		pszCaptionText: titlePtr,
	}

	username := make([]uint16, credUIMaxUsernameLength)
	password := make([]uint16, credUIMaxPasswordLength)
	save := uint32(0)
	flags := uint32(credUIFlagsGenericCredentials | credUIFlagsAlwaysShowUI | credUIFlagsDoNotPersist | credUIFlagsExpectConfirmation)

	r1, _, _ := procCredUIPromptForCredsW.Call(
		uintptr(unsafe.Pointer(&ui)),
		uintptr(unsafe.Pointer(targetPtr)),
		0,
		0,
		uintptr(unsafe.Pointer(&username[0])),
		uintptr(len(username)),
		uintptr(unsafe.Pointer(&password[0])),
		uintptr(len(password)),
		uintptr(unsafe.Pointer(&save)),
		uintptr(flags),
	)

	clearUTF16Buffer(username)
	defer clearUTF16Buffer(password)

	switch windows.Errno(r1) {
	case 0:
		value := strings.TrimSpace(windows.UTF16ToString(password))
		if value == "" {
			return "", false, nil
		}
		return value, true, nil
	case windows.ERROR_CANCELLED:
		return "", false, nil
	default:
		return "", false, errors.New("native password prompt failed")
	}
}

func passwordPromptTexts(lang string) (string, string) {
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(lang)), "zh") {
		return "Web Share 上传密码", "请输入上传密码。留空则取消分享。"
	}
	return "Web Share Upload Password", "Enter upload password. Leave empty to cancel sharing."
}

func clearUTF16Buffer(buf []uint16) {
	for i := range buf {
		buf[i] = 0
	}
}
