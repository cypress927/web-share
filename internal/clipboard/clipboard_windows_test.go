//go:build windows

package clipboard

import (
	"strings"
	"testing"
)

func TestMakeClipboardTextTitle(t *testing.T) {
	title := makeClipboardTextTitle("This is line one\nThis is line two")
	if !strings.HasPrefix(title, "Text: ") {
		t.Fatalf("unexpected title prefix: %q", title)
	}
	if !strings.Contains(title, "This is line one") {
		t.Fatalf("unexpected text title: %q", title)
	}
}

func TestMakeClipboardImageTitle(t *testing.T) {
	title := makeClipboardImageTitle(nil)
	if !strings.HasPrefix(title, "Image: ") {
		t.Fatalf("unexpected image title prefix: %q", title)
	}
}
