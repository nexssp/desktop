package desktop

import (
	"errors"
	"strconv"
	"strings"
)

const (
	ModifierAlt     uint32 = 0x0001
	ModifierControl uint32 = 0x0002
	ModifierShift   uint32 = 0x0004
)

// Shortcut is a parsed global keyboard shortcut.
type Shortcut struct {
	Modifiers  uint32
	VirtualKey uint32
	Canonical  string
}

// ParseShortcut validates and canonicalizes a shortcut using the supported
// Windows global-hotkey key set. The parser is portable so applications can
// validate saved preferences on every platform.
func ParseShortcut(value string) (Shortcut, error) {
	parts := strings.Split(strings.TrimSpace(value), "+")
	var result Shortcut
	var key string
	seen := make(map[string]bool, 3)
	for _, part := range parts {
		token := strings.TrimSpace(part)
		if token == "" {
			return Shortcut{}, errors.New("enter a key combination such as Ctrl+Shift+Space")
		}
		switch strings.ToLower(token) {
		case "ctrl", "control":
			if seen["Ctrl"] {
				return Shortcut{}, errors.New("ctrl modifier is listed more than once")
			}
			seen["Ctrl"] = true
			result.Modifiers |= ModifierControl
		case "alt":
			if seen["Alt"] {
				return Shortcut{}, errors.New("alt modifier is listed more than once")
			}
			seen["Alt"] = true
			result.Modifiers |= ModifierAlt
		case "shift":
			if seen["Shift"] {
				return Shortcut{}, errors.New("shift modifier is listed more than once")
			}
			seen["Shift"] = true
			result.Modifiers |= ModifierShift
		default:
			if key != "" {
				return Shortcut{}, errors.New("choose exactly one non-modifier key")
			}
			key = token
		}
	}
	if result.Modifiers == 0 {
		return Shortcut{}, errors.New("include at least one modifier: Ctrl, Alt, or Shift")
	}
	if key == "" {
		return Shortcut{}, errors.New("choose a key after the modifiers")
	}
	canonicalKey, virtualKey, ok := virtualKeyFor(key)
	if !ok {
		return Shortcut{}, errors.New("use a letter, number, F1–F24, Space, Enter, Esc, Backspace, Delete, Insert, Home, End, PageUp, PageDown, or an arrow key")
	}
	modifiers := make([]string, 0, 3)
	if seen["Ctrl"] {
		modifiers = append(modifiers, "Ctrl")
	}
	if seen["Alt"] {
		modifiers = append(modifiers, "Alt")
	}
	if seen["Shift"] {
		modifiers = append(modifiers, "Shift")
	}
	result.VirtualKey = virtualKey
	result.Canonical = strings.Join(append(modifiers, canonicalKey), "+")
	return result, nil
}

func virtualKeyFor(value string) (canonical string, code uint32, ok bool) {
	if len(value) == 1 {
		upper := strings.ToUpper(value)
		if upper[0] >= 'A' && upper[0] <= 'Z' || upper[0] >= '0' && upper[0] <= '9' {
			return upper, uint32(upper[0]), true
		}
	}
	upper := strings.ToUpper(value)
	if strings.HasPrefix(upper, "F") {
		if number, err := strconv.Atoi(upper[1:]); err == nil && number >= 1 && number <= 24 {
			return "F" + strconv.Itoa(number), uint32(0x70 + number - 1), true
		}
	}
	keys := map[string]struct {
		name string
		code uint32
	}{
		"SPACE": {"Space", 0x20}, "ENTER": {"Enter", 0x0d}, "ESC": {"Esc", 0x1b},
		"BACKSPACE": {"Backspace", 0x08}, "DELETE": {"Delete", 0x2e}, "INSERT": {"Insert", 0x2d},
		"HOME": {"Home", 0x24}, "END": {"End", 0x23}, "PAGEUP": {"PageUp", 0x21}, "PAGEDOWN": {"PageDown", 0x22},
		"UP": {"Up", 0x26}, "DOWN": {"Down", 0x28}, "LEFT": {"Left", 0x25}, "RIGHT": {"Right", 0x27},
	}
	if key, ok := keys[upper]; ok {
		return key.name, key.code, true
	}
	return "", 0, false
}
