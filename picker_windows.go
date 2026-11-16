//go:build windows

package desktop

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"syscall"
	"unsafe"
)

const (
	clsctxInprocServer  = 1
	coinitApartment     = 2
	fosPickFolders      = 0x20
	fosForceFileSystem  = 0x40
	fosPathMustExist    = 0x800
	fosFileMustExist    = 0x1000
	fosAllowMultiSelect = 0x200
	sigdnFileSystemPath = 0x80058000
)

type comGUID struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 [8]byte
}

type comDialogFilter struct {
	Name    *uint16
	Pattern *uint16
}

var (
	pickerOle32          = syscall.NewLazyDLL("ole32.dll")
	pickerShell32        = syscall.NewLazyDLL("shell32.dll")
	pickerUser32         = syscall.NewLazyDLL("user32.dll")
	pickerCoInitialize   = pickerOle32.NewProc("CoInitializeEx")
	pickerCoUninitialize = pickerOle32.NewProc("CoUninitialize")
	pickerCoCreate       = pickerOle32.NewProc("CoCreateInstance")
	pickerTaskMemFree    = pickerOle32.NewProc("CoTaskMemFree")
	pickerCreateItem     = pickerShell32.NewProc("SHCreateItemFromParsingName")
	pickerGetForeground  = pickerUser32.NewProc("GetForegroundWindow")
)

func (NativePicker) PickFile(ctx context.Context, request FilePickerRequest) (FileSelection, error) {
	if err := checkContext(ctx); err != nil {
		return FileSelection{}, err
	}
	path, err := showFileDialog(request, false, false)
	return FileSelection{Path: path}, err
}

func (NativePicker) PickFolder(ctx context.Context, request FolderPickerRequest) (FolderSelection, error) {
	if err := checkContext(ctx); err != nil {
		return FolderSelection{}, err
	}
	path, err := showFolderDialog(request, false)
	return FolderSelection{Path: path}, err
}

func (NativePicker) PickFolders(ctx context.Context, request FolderPickerRequest) (FoldersSelection, error) {
	if err := checkContext(ctx); err != nil {
		return FoldersSelection{Paths: []string{}}, err
	}
	paths, err := showFolderDialogMany(request)
	if paths == nil {
		paths = []string{}
	}
	return FoldersSelection{Paths: paths}, err
}

func showFileDialog(request FilePickerRequest, folders, multiple bool) (string, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if err := initializeCOM(); err != nil {
		return "", err
	}
	defer pickerCoUninitialize.Call()
	dialog, err := createOpenDialog()
	if err != nil {
		return "", err
	}
	defer comCall(dialog, 2)

	options := uintptr(fosForceFileSystem)
	if folders {
		options |= fosPickFolders
	} else {
		options |= fosPathMustExist | fosFileMustExist
	}
	if multiple {
		options |= fosAllowMultiSelect
	}
	if _, err := comCall(dialog, 10, uintptr(unsafe.Pointer(&options))); err != nil {
		return "", err
	}
	if _, err := comCall(dialog, 9, options); err != nil {
		return "", err
	}
	if err := setDialogTitle(dialog, request.Title); err != nil {
		return "", err
	}
	if !folders {
		if err := setDialogFilters(dialog, request.Filters); err != nil {
			return "", err
		}
	}
	if err := setDialogFolder(dialog, request.InitialDirectory); err != nil {
		return "", err
	}
	owner, _, _ := pickerGetForeground.Call()
	showHR, showErr := comCall(dialog, 3, owner)
	if IsDialogCancellation(showHR, showErr) {
		return "", nil
	}
	if showErr != nil {
		return "", showErr
	}
	var item unsafe.Pointer
	if _, err := comCall(dialog, 20, uintptr(unsafe.Pointer(&item))); err != nil {
		return "", err
	}
	if item == nil {
		return "", errors.New("native dialog returned no selected item")
	}
	defer comCall(item, 2)
	return itemPath(item)
}

func showFolderDialog(request FolderPickerRequest, multiple bool) (string, error) {
	return showFileDialog(FilePickerRequest{Title: request.Title, InitialDirectory: request.InitialDirectory}, true, multiple)
}

func showFolderDialogMany(request FolderPickerRequest) ([]string, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if err := initializeCOM(); err != nil {
		return nil, err
	}
	defer pickerCoUninitialize.Call()
	dialog, err := createOpenDialog()
	if err != nil {
		return nil, err
	}
	defer comCall(dialog, 2)
	options := uintptr(fosForceFileSystem | fosPickFolders | fosAllowMultiSelect)
	if _, err := comCall(dialog, 10, uintptr(unsafe.Pointer(&options))); err != nil {
		return nil, err
	}
	if _, err := comCall(dialog, 9, options); err != nil {
		return nil, err
	}
	if err := setDialogTitle(dialog, request.Title); err != nil {
		return nil, err
	}
	if err := setDialogFolder(dialog, request.InitialDirectory); err != nil {
		return nil, err
	}
	owner, _, _ := pickerGetForeground.Call()
	showHR, showErr := comCall(dialog, 3, owner)
	if IsDialogCancellation(showHR, showErr) {
		return []string{}, nil
	}
	if showErr != nil {
		return nil, showErr
	}
	var items unsafe.Pointer
	if _, err := comCall(dialog, 27, uintptr(unsafe.Pointer(&items))); err != nil {
		return nil, err
	}
	if items == nil {
		return nil, errors.New("native dialog returned no selected items")
	}
	defer comCall(items, 2)
	var count uint32
	if _, err := comCall(items, 7, uintptr(unsafe.Pointer(&count))); err != nil {
		return nil, err
	}
	paths := make([]string, 0, count)
	for index := uint32(0); index < count; index++ {
		var item unsafe.Pointer
		if _, err := comCall(items, 8, uintptr(index), uintptr(unsafe.Pointer(&item))); err != nil {
			return nil, err
		}
		path, pathErr := itemPath(item)
		comCall(item, 2)
		if pathErr != nil {
			return nil, pathErr
		}
		paths = append(paths, path)
	}
	return paths, nil
}

func initializeCOM() error {
	hr, _, _ := pickerCoInitialize.Call(0, coinitApartment)
	if int32(hr) < 0 {
		return fmt.Errorf("initialize native dialog: HRESULT 0x%08X", uint32(hr))
	}
	return nil
}

func createOpenDialog() (unsafe.Pointer, error) {
	classID := comGUID{Data1: 0xDC1C5A9C, Data2: 0xE88A, Data3: 0x4DDE, Data4: [8]byte{0xA5, 0xA1, 0x60, 0xF8, 0x2A, 0x20, 0xAE, 0xF7}}
	interfaceID := comGUID{Data1: 0xD57C7288, Data2: 0xD4AD, Data3: 0x4768, Data4: [8]byte{0xBE, 0x02, 0x9D, 0x96, 0x95, 0x32, 0xD9, 0x60}}
	var dialog unsafe.Pointer
	hr, _, _ := pickerCoCreate.Call(uintptr(unsafe.Pointer(&classID)), 0, clsctxInprocServer, uintptr(unsafe.Pointer(&interfaceID)), uintptr(unsafe.Pointer(&dialog)))
	if int32(hr) < 0 {
		return nil, fmt.Errorf("create native dialog: HRESULT 0x%08X", uint32(hr))
	}
	if dialog == nil {
		return nil, errors.New("native dialog creation returned no interface")
	}
	return dialog, nil
}

func setDialogTitle(dialog unsafe.Pointer, title string) error {
	if title == "" {
		return nil
	}
	value, err := syscall.UTF16PtrFromString(title)
	if err != nil {
		return fmt.Errorf("encode dialog title: %w", err)
	}
	_, err = comCall(dialog, 17, uintptr(unsafe.Pointer(value)))
	runtime.KeepAlive(value)
	return err
}

func setDialogFilters(dialog unsafe.Pointer, filters []FileFilter) error {
	if len(filters) == 0 {
		return nil
	}
	native := make([]comDialogFilter, len(filters))
	for index, filter := range filters {
		name, err := syscall.UTF16PtrFromString(filter.Name)
		if err != nil {
			return fmt.Errorf("encode file filter name: %w", err)
		}
		pattern, err := syscall.UTF16PtrFromString(filter.Pattern)
		if err != nil {
			return fmt.Errorf("encode file filter pattern: %w", err)
		}
		native[index] = comDialogFilter{Name: name, Pattern: pattern}
	}
	_, err := comCall(dialog, 4, uintptr(len(native)), uintptr(unsafe.Pointer(&native[0])))
	runtime.KeepAlive(native)
	return err
}

func setDialogFolder(dialog unsafe.Pointer, directory string) error {
	if directory == "" {
		return nil
	}
	info, err := os.Stat(directory)
	if err != nil || !info.IsDir() {
		return nil
	}
	path, err := syscall.UTF16PtrFromString(directory)
	if err != nil {
		return fmt.Errorf("encode initial directory: %w", err)
	}
	itemID := comGUID{Data1: 0x43826D1E, Data2: 0xE718, Data3: 0x42EE, Data4: [8]byte{0xBC, 0x55, 0xA1, 0xE2, 0x61, 0xC3, 0x7B, 0xFE}}
	var folder unsafe.Pointer
	hr, _, _ := pickerCreateItem.Call(uintptr(unsafe.Pointer(path)), 0, uintptr(unsafe.Pointer(&itemID)), uintptr(unsafe.Pointer(&folder)))
	if int32(hr) < 0 || folder == nil {
		return fmt.Errorf("create initial directory item: HRESULT 0x%08X", uint32(hr))
	}
	defer comCall(folder, 2)
	_, err = comCall(dialog, 12, uintptr(folder))
	runtime.KeepAlive(path)
	runtime.KeepAlive(folder)
	return err
}

func itemPath(item unsafe.Pointer) (string, error) {
	var displayName unsafe.Pointer
	if _, err := comCall(item, 5, sigdnFileSystemPath, uintptr(unsafe.Pointer(&displayName))); err != nil {
		return "", err
	}
	if displayName == nil {
		return "", errors.New("native dialog returned no selected path")
	}
	defer pickerTaskMemFree.Call(uintptr(displayName))
	path := syscall.UTF16ToString(unsafe.Slice((*uint16)(displayName), 32768))
	return ValidateFilePath(path)
}

func comCall(object unsafe.Pointer, index uintptr, args ...uintptr) (uintptr, error) {
	vtable := *(*unsafe.Pointer)(object)
	method := *(*uintptr)(unsafe.Add(vtable, index*unsafe.Sizeof(uintptr(0))))
	params := make([]uintptr, 1, len(args)+1)
	params[0] = uintptr(object)
	params = append(params, args...)
	hr, _, _ := syscall.SyscallN(method, params...)
	if int32(hr) < 0 {
		return hr, fmt.Errorf("native dialog call failed: HRESULT 0x%08X", uint32(hr))
	}
	return hr, nil
}
