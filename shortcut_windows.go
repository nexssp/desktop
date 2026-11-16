//go:build windows

package desktop

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	windowShortcutHotkeyID    = 0x4e58
	windowShortcutAlternateID = 0x4e59
	modNoRepeat               = 0x4000
	wmQuit                    = 0x0012
	wmHotkey                  = 0x0312
	wmSetWindowShortcut       = 0x8001
	swRestore                 = 9
)

type windowShortcutRequest struct {
	shortcut string
	result   chan error
}

type windowShortcutState struct {
	changeMu  sync.Mutex
	pendingMu sync.Mutex
	pending   *windowShortcutRequest
	threadID  uint32
	stopped   chan struct{}
}

var (
	windowShortcutMu     sync.RWMutex
	activeWindowShortcut *windowShortcutState
)

type windowMessage struct {
	hwnd    uintptr
	message uint32
	wParam  uintptr
	lParam  uintptr
	time    uint32
	point   struct{ x, y int32 }
	private uint32
}

var shortcutUser32 = windows.NewLazySystemDLL("user32.dll")
var (
	registerHotKeyProc      = shortcutUser32.NewProc("RegisterHotKey")
	unregisterHotKeyProc    = shortcutUser32.NewProc("UnregisterHotKey")
	getMessageProc          = shortcutUser32.NewProc("GetMessageW")
	postThreadMessageProc   = shortcutUser32.NewProc("PostThreadMessageW")
	enumWindowsProc         = shortcutUser32.NewProc("EnumWindows")
	getWindowProcessIDProc  = shortcutUser32.NewProc("GetWindowThreadProcessId")
	isWindowVisibleProc     = shortcutUser32.NewProc("IsWindowVisible")
	isWindowIconicProc      = shortcutUser32.NewProc("IsIconic")
	showWindowProc          = shortcutUser32.NewProc("ShowWindow")
	bringWindowToTopProc    = shortcutUser32.NewProc("BringWindowToTop")
	setForegroundWindowProc = shortcutUser32.NewProc("SetForegroundWindow")
)

// EnumWindows calls this process-lifetime callback synchronously while the mutex is held.
var (
	processWindowActivationMu     sync.Mutex
	processWindowActivationPID    uint32
	processWindowActivationTarget uintptr
	processWindowEnumCallback     = syscall.NewCallback(findVisibleProcessWindow)
)

func findVisibleProcessWindow(hwnd, _ uintptr) uintptr {
	var ownerPID uint32
	getWindowProcessIDProc.Call(hwnd, uintptr(unsafe.Pointer(&ownerPID)))
	if ownerPID != processWindowActivationPID {
		return 1
	}
	visible, _, _ := isWindowVisibleProc.Call(hwnd)
	if visible == 0 {
		return 1
	}
	processWindowActivationTarget = hwnd
	return 0
}

// EnableWindowShortcut owns a dedicated message thread so the registered
// global shortcut can be changed while the app is running.
func EnableWindowShortcut(ctx context.Context, process *os.Process, shortcut string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if process == nil || process.Pid <= 0 {
		return errors.New("the target process is unavailable")
	}
	parsed, err := ParseShortcut(shortcut)
	if err != nil {
		return err
	}
	state := &windowShortcutState{stopped: make(chan struct{})}
	windowShortcutMu.Lock()
	if activeWindowShortcut != nil {
		windowShortcutMu.Unlock()
		return errors.New("the window shortcut listener is already running")
	}
	activeWindowShortcut = state
	windowShortcutMu.Unlock()

	ready := make(chan error, 1)
	go runWindowShortcut(ctx, uint32(process.Pid), state, parsed, ready)
	return <-ready
}

func runWindowShortcut(ctx context.Context, pid uint32, state *windowShortcutState, initial Shortcut, ready chan<- error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	threadID := windows.GetCurrentThreadId()
	state.pendingMu.Lock()
	state.threadID = threadID
	state.pendingMu.Unlock()

	currentID := uintptr(0)
	currentShortcut := ""
	if err := registerWindowShortcut(windowShortcutHotkeyID, initial); err != nil {
		ready <- err
	} else {
		currentID = windowShortcutHotkeyID
		currentShortcut = initial.Canonical
		ready <- nil
	}
	defer func() {
		if currentID != 0 {
			unregisterHotKeyProc.Call(0, currentID)
		}
		windowShortcutMu.Lock()
		if activeWindowShortcut == state {
			activeWindowShortcut = nil
		}
		windowShortcutMu.Unlock()
		close(state.stopped)
	}()

	stopWatcher := make(chan struct{})
	defer close(stopWatcher)
	go func() {
		select {
		case <-ctx.Done():
			postThreadMessageProc.Call(uintptr(threadID), wmQuit, 0, 0)
		case <-stopWatcher:
		}
	}()

	for {
		var message windowMessage
		result, _, _ := getMessageProc.Call(uintptr(unsafe.Pointer(&message)), 0, 0, 0)
		if result == 0 || result == ^uintptr(0) {
			return
		}
		switch message.message {
		case wmHotkey:
			if message.wParam == currentID && currentID != 0 {
				activateProcessWindow(pid)
			}
		case wmSetWindowShortcut:
			applyPendingWindowShortcut(state, &currentID, &currentShortcut)
		}
	}
}

func registerWindowShortcut(id uintptr, shortcut Shortcut) error {
	registered, _, callErr := registerHotKeyProc.Call(0, id, uintptr(shortcut.Modifiers|modNoRepeat), uintptr(shortcut.VirtualKey))
	if registered != 0 {
		return nil
	}
	reason := "the shortcut may already be in use or Windows denied access"
	if callErr != syscall.Errno(0) {
		reason = callErr.Error()
	}
	return fmt.Errorf("Windows could not register %s (%s); choose another shortcut", shortcut.Canonical, reason)
}

func setWindowShortcutInternal(value string) error {
	shortcut, err := ParseShortcut(value)
	if err != nil {
		return err
	}
	windowShortcutMu.RLock()
	state := activeWindowShortcut
	windowShortcutMu.RUnlock()
	if state == nil {
		return nil
	}

	state.changeMu.Lock()
	defer state.changeMu.Unlock()
	request := &windowShortcutRequest{shortcut: shortcut.Canonical, result: make(chan error, 1)}
	state.pendingMu.Lock()
	threadID := state.threadID
	if threadID == 0 {
		state.pendingMu.Unlock()
		return errors.New("the window shortcut listener is not ready")
	}
	state.pending = request
	state.pendingMu.Unlock()
	posted, _, callErr := postThreadMessageProc.Call(uintptr(threadID), wmSetWindowShortcut, 0, 0)
	if posted == 0 {
		state.pendingMu.Lock()
		if state.pending == request {
			state.pending = nil
		}
		state.pendingMu.Unlock()
		if callErr == syscall.Errno(0) {
			callErr = errors.New("Windows could not reach the shortcut listener")
		}
		return callErr
	}
	select {
	case err := <-request.result:
		return err
	case <-state.stopped:
		select {
		case err := <-request.result:
			return err
		default:
			return errors.New("the window shortcut listener stopped before it could apply the change")
		}
	}
}

// SetWindowShortcut updates the active Windows desktop shortcut.
func SetWindowShortcut(value string) error { return setWindowShortcutInternal(value) }

func applyPendingWindowShortcut(state *windowShortcutState, currentID *uintptr, currentShortcut *string) {
	state.pendingMu.Lock()
	request := state.pending
	state.pending = nil
	state.pendingMu.Unlock()
	if request == nil {
		return
	}
	if request.shortcut == *currentShortcut {
		request.result <- nil
		return
	}
	parsed, err := ParseShortcut(request.shortcut)
	if err == nil {
		nextID := uintptr(windowShortcutHotkeyID)
		if *currentID == nextID {
			nextID = windowShortcutAlternateID
		}
		err = registerWindowShortcut(nextID, parsed)
		if err == nil {
			if *currentID != 0 {
				unregisterHotKeyProc.Call(0, *currentID)
			}
			*currentID = nextID
			*currentShortcut = parsed.Canonical
		}
	}
	request.result <- err
}

func activateProcessWindow(pid uint32) {
	processWindowActivationMu.Lock()
	defer processWindowActivationMu.Unlock()
	processWindowActivationPID = pid
	for attempt := 0; attempt < 10; attempt++ {
		processWindowActivationTarget = 0
		enumWindowsProc.Call(processWindowEnumCallback, 0)
		target := processWindowActivationTarget
		if target != 0 {
			if iconic, _, _ := isWindowIconicProc.Call(target); iconic != 0 {
				showWindowProc.Call(target, swRestore)
			}
			bringWindowToTopProc.Call(target)
			setForegroundWindowProc.Call(target)
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
}
