package desktop

import (
	"context"
	"errors"
)

const dialogCanceledHRESULT = 0x800704C7

// ErrUnsupported indicates that a requested native desktop capability is not
// available on the current platform.
var ErrUnsupported = errors.New("native desktop capability is unsupported on this platform")

// FileFilter describes one file-dialog filter. Pattern uses the platform's
// native wildcard syntax, such as "*.go;*.mod".
type FileFilter struct {
	Name    string `json:"name"`
	Pattern string `json:"pattern"`
}

// FilePickerRequest contains caller-owned text and directory policy for a file
// dialog. Empty fields are passed through without app-specific defaults.
type FilePickerRequest struct {
	Title            string       `json:"title,omitempty"`
	InitialDirectory string       `json:"initial_directory,omitempty"`
	Filters          []FileFilter `json:"filters,omitempty"`
}

// FolderPickerRequest contains caller-owned text and directory policy for a
// folder dialog.
type FolderPickerRequest struct {
	Title            string `json:"title,omitempty"`
	InitialDirectory string `json:"initial_directory,omitempty"`
}

// FileSelection is the result of a file picker. An empty Path means the user
// canceled the dialog.
type FileSelection struct {
	Path string `json:"path"`
}

// FolderSelection is the result of a single-folder picker. An empty Path means
// the user canceled the dialog.
type FolderSelection struct {
	Path string `json:"path"`
}

// FoldersSelection is the result of a multi-folder picker. Paths is non-nil,
// including when the user canceled the dialog.
type FoldersSelection struct {
	Paths []string `json:"paths"`
}

// Picker provides native file and folder selection capabilities.
type Picker interface {
	PickFile(context.Context, FilePickerRequest) (FileSelection, error)
	PickFolder(context.Context, FolderPickerRequest) (FolderSelection, error)
	PickFolders(context.Context, FolderPickerRequest) (FoldersSelection, error)
}

// NativePicker uses the current platform's native file and folder dialogs.
type NativePicker struct{}

func checkContext(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	return ctx.Err()
}

// IsDialogCancellation reports whether a failed native dialog call was canceled
// by the user rather than failing for another reason.
func IsDialogCancellation(hresult uintptr, err error) bool {
	return err != nil && hresult == uintptr(dialogCanceledHRESULT)
}

// ValidateFilePath rejects a successful native dialog result with no selected
// path; callers should represent cancellation as an empty result instead.
func ValidateFilePath(path string) (string, error) {
	if path == "" {
		return "", errors.New("native file dialog succeeded without a selected path")
	}
	return path, nil
}
