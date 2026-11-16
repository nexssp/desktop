package desktop

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// LogOptions describes a caller-owned log file and rotation policy.
type LogOptions struct {
	Directory     string
	FileName      string
	MaxBytes      int64
	Backups       int
	DirectoryMode fs.FileMode
	FileMode      fs.FileMode
}

// OpenLog creates the requested directory and opens the configured log for
// append. A file at or above MaxBytes is rotated before opening. Zero MaxBytes
// disables size-based rotation; Backups is the maximum numbered backup count.
func OpenLog(options LogOptions) (*os.File, error) {
	if strings.TrimSpace(options.Directory) == "" {
		return nil, errors.New("log directory is required")
	}
	if options.FileName == "" || filepath.Base(options.FileName) != options.FileName || options.FileName == "." || options.FileName == ".." {
		return nil, errors.New("log filename must be a simple filename")
	}
	if options.MaxBytes < 0 || options.Backups < 0 {
		return nil, errors.New("log rotation limits cannot be negative")
	}
	directoryMode := options.DirectoryMode
	if directoryMode == 0 {
		directoryMode = 0o700
	}
	fileMode := options.FileMode
	if fileMode == 0 {
		fileMode = 0o600
	}
	if err := os.MkdirAll(options.Directory, directoryMode); err != nil {
		return nil, fmt.Errorf("create log directory: %w", err)
	}
	path := filepath.Join(options.Directory, options.FileName)
	if options.MaxBytes > 0 {
		info, err := os.Stat(path)
		if err == nil && info.Size() >= options.MaxBytes {
			if rotateErr := rotateLog(path, options.Backups); rotateErr != nil {
				return nil, fmt.Errorf("rotate log file: %w", rotateErr)
			}
		} else if err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("inspect log file: %w", err)
		}
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, fileMode)
	if err != nil {
		return nil, fmt.Errorf("open log file: %w", err)
	}
	return file, nil
}

func rotateLog(path string, backups int) error {
	if backups == 0 {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		return nil
	}
	for index := backups; index >= 1; index-- {
		source := path
		if index > 1 {
			source = fmt.Sprintf("%s.%d", path, index-1)
		}
		destination := fmt.Sprintf("%s.%d", path, index)
		if _, err := os.Stat(source); errors.Is(err, os.ErrNotExist) {
			continue
		} else if err != nil {
			return err
		}
		if err := os.Remove(destination); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		if err := os.Rename(source, destination); err != nil {
			return err
		}
	}
	return nil
}
