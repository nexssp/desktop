//go:build !windows

package desktop

import (
	"context"
	"errors"
	"testing"
)

func TestNativePickerIsUnsupportedOffWindows(t *testing.T) {
	picker := NativePicker{}
	if _, err := picker.PickFile(context.Background(), FilePickerRequest{}); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("PickFile() error = %v, want ErrUnsupported", err)
	}
	if _, err := picker.PickFolder(context.Background(), FolderPickerRequest{}); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("PickFolder() error = %v, want ErrUnsupported", err)
	}
	result, err := picker.PickFolders(context.Background(), FolderPickerRequest{})
	if !errors.Is(err, ErrUnsupported) || result.Paths == nil {
		t.Fatalf("PickFolders() = %#v, %v; want non-nil empty paths and ErrUnsupported", result, err)
	}
}
