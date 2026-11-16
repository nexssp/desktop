package desktop

import "testing"

func TestParseShortcutCanonicalizes(t *testing.T) {
	got, err := ParseShortcut("control + shift + space")
	if err != nil {
		t.Fatal(err)
	}
	if got.Canonical != "Ctrl+Shift+Space" || got.VirtualKey != 0x20 || got.Modifiers != ModifierControl|ModifierShift {
		t.Fatalf("parsed shortcut = %#v", got)
	}
}

func TestParseShortcutRejectsInvalidValues(t *testing.T) {
	for _, value := range []string{"", "Ctrl+Ctrl+A", "A", "Ctrl+Shift+NoSuchKey", "Ctrl+F25", "Ctrl+A+B"} {
		t.Run(value, func(t *testing.T) {
			if _, err := ParseShortcut(value); err == nil {
				t.Fatalf("ParseShortcut(%q) succeeded", value)
			}
		})
	}
}
