//go:build !windows

package desktop

import "context"

func (NativePicker) PickFile(ctx context.Context, _ FilePickerRequest) (FileSelection, error) {
	if err := checkContext(ctx); err != nil {
		return FileSelection{}, err
	}
	return FileSelection{}, ErrUnsupported
}

func (NativePicker) PickFolder(ctx context.Context, _ FolderPickerRequest) (FolderSelection, error) {
	if err := checkContext(ctx); err != nil {
		return FolderSelection{}, err
	}
	return FolderSelection{}, ErrUnsupported
}

func (NativePicker) PickFolders(ctx context.Context, _ FolderPickerRequest) (FoldersSelection, error) {
	if err := checkContext(ctx); err != nil {
		return FoldersSelection{}, err
	}
	return FoldersSelection{Paths: []string{}}, ErrUnsupported
}
