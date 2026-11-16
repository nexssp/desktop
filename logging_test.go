package desktop

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestOpenLogUsesCallerNameAndRotatesBackups(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "logs")
	path := filepath.Join(directory, "service.log")
	if err := os.MkdirAll(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("current-old"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path+".1", []byte("previous"), 0o600); err != nil {
		t.Fatal(err)
	}
	file, err := OpenLog(LogOptions{Directory: directory, FileName: "service.log", MaxBytes: 5, Backups: 2})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.WriteString("new"); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	for path, want := range map[string]string{path: "new", path + ".1": "current-old", path + ".2": "previous"} {
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != want {
			t.Errorf("%s = %q, want %q", path, got, want)
		}
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if got := info.Mode().Perm(); got&0o077 != 0 {
			t.Errorf("log file permissions = %04o, want owner-only", got)
		}
	}
}

func TestOpenLogRejectsPathInFileName(t *testing.T) {
	if _, err := OpenLog(LogOptions{Directory: t.TempDir(), FileName: "../outside.log"}); err == nil {
		t.Fatal("path traversal in log filename was accepted")
	}
}
