//go:build !windows

package desktop

import (
	"context"
	"os"
)

// EnableWindowShortcut reports unsupported platforms; callers may treat this
// as an optional capability and continue without a global shortcut.
func EnableWindowShortcut(ctx context.Context, _ *os.Process, _ string) error {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return err
		}
	}
	return ErrUnsupported
}

// SetWindowShortcut is a no-op on platforms without a native global-hotkey
// implementation, allowing preference persistence to remain portable.
func SetWindowShortcut(string) error { return nil }
