//go:build windows

package clipboard

import (
	"strings"
	"testing"
)

func TestMakeClipboardTextTitle(t *testing.T) {
	title := makeClipboardTextTitle("这是第一行\n这是第二行")
	if !strings.HasPrefix(title, "文本: ") {
		t.Fatalf("unexpected title prefix: %q", title)
	}
	if !strings.Contains(title, "这是第一行") {
		t.Fatalf("unexpected text title: %q", title)
	}
}

func TestMakeClipboardImageTitle(t *testing.T) {
	title := makeClipboardImageTitle(nil)
	if !strings.HasPrefix(title, "图片: ") {
		t.Fatalf("unexpected image title prefix: %q", title)
	}
}
