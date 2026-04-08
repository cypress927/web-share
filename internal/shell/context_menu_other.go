//go:build !windows

package shell

func ContextMenuInstalled() bool {
	return false
}
