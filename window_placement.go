package desktop

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Placement records a top-level window's position and size.
type Placement struct {
	Version   int  `json:"version"`
	X         int  `json:"x"`
	Y         int  `json:"y"`
	Width     int  `json:"width"`
	Height    int  `json:"height"`
	Maximized bool `json:"maximized,omitempty"`
}

// ScreenRect describes a monitor's usable work area in desktop coordinates.
type ScreenRect struct {
	Left   int
	Top    int
	Right  int
	Bottom int
}

// WindowSize describes a default width and height selected by the host.
type WindowSize struct {
	Width  int
	Height int
}

// PlacementLimits constrain persisted and restored geometry.
type PlacementLimits struct {
	MinWidth         int
	MinHeight        int
	MaxWidth         int
	MaxHeight        int
	MinVisibleWidth  int
	MinVisibleHeight int
}

// DefaultPlacementLimits returns conservative bounds suitable for a typical
// desktop window. Hosts can supply their own limits to NormalizePlacement.
func DefaultPlacementLimits() PlacementLimits {
	return PlacementLimits{
		MinWidth: 640, MinHeight: 420, MaxWidth: 4096, MaxHeight: 4096,
		MinVisibleWidth: 120, MinVisibleHeight: 80,
	}
}

// ValidPlacementSize reports whether width and height fit limits.
func ValidPlacementSize(width, height int, limits PlacementLimits) bool {
	return width >= limits.MinWidth && height >= limits.MinHeight && width <= limits.MaxWidth && height <= limits.MaxHeight
}

// ValidPlacementPosition guards against corrupted extreme coordinates.
func ValidPlacementPosition(x, y int) bool {
	return x >= -100000 && x <= 100000 && y >= -100000 && y <= 100000
}

// PlacementVisibleInWorkArea requires enough of the saved window to remain
// reachable; a tiny overlap is not considered a useful restore position.
func PlacementVisibleInWorkArea(placement Placement, work ScreenRect, limits PlacementLimits) bool {
	if !ValidPlacementSize(placement.Width, placement.Height, limits) || !ValidPlacementPosition(placement.X, placement.Y) || work.Right <= work.Left || work.Bottom <= work.Top {
		return false
	}
	left := max(placement.X, work.Left)
	top := max(placement.Y, work.Top)
	right := min(placement.X+placement.Width, work.Right)
	bottom := min(placement.Y+placement.Height, work.Bottom)
	return right-left >= limits.MinVisibleWidth && bottom-top >= limits.MinVisibleHeight
}

// NormalizePlacement restores a visible saved placement or centers the host's
// caller-supplied default size in the supplied work area.
func NormalizePlacement(saved Placement, hasSaved, visible bool, work ScreenRect, defaults WindowSize, limits PlacementLimits) (Placement, error) {
	if work.Right <= work.Left || work.Bottom <= work.Top {
		return Placement{}, errors.New("work area must have positive dimensions")
	}
	if defaults.Width <= 0 || defaults.Height <= 0 {
		return Placement{}, errors.New("default window size must be positive")
	}
	if limits.MinWidth <= 0 || limits.MinHeight <= 0 || limits.MaxWidth < limits.MinWidth || limits.MaxHeight < limits.MinHeight || limits.MinVisibleWidth <= 0 || limits.MinVisibleHeight <= 0 {
		return Placement{}, errors.New("placement limits are invalid")
	}
	if hasSaved && visible && ValidPlacementSize(saved.Width, saved.Height, limits) && ValidPlacementPosition(saved.X, saved.Y) && PlacementVisibleInWorkArea(saved, work, limits) {
		workWidth, workHeight := work.Right-work.Left, work.Bottom-work.Top
		saved.Width = min(saved.Width, workWidth)
		saved.Height = min(saved.Height, workHeight)
		saved.X = max(work.Left, min(saved.X, work.Right-saved.Width))
		saved.Y = max(work.Top, min(saved.Y, work.Bottom-saved.Height))
		saved.Version = 1
		return saved, nil
	}
	width := min(defaults.Width, work.Right-work.Left)
	height := min(defaults.Height, work.Bottom-work.Top)
	return Placement{
		Version: 1,
		X:       work.Left + (work.Right-work.Left-width)/2,
		Y:       work.Top + (work.Bottom-work.Top-height)/2,
		Width:   width,
		Height:  height,
	}, nil
}

// LoadPlacement reads a placement from a caller-supplied file path. A missing
// file is not an error; malformed or unsupported values are reported as absent.
func LoadPlacement(path string) (Placement, bool, error) {
	if strings.TrimSpace(path) == "" {
		return Placement{}, false, errors.New("placement file path is required")
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Placement{}, false, nil
	}
	if err != nil {
		return Placement{}, false, err
	}
	var placement Placement
	if err := json.Unmarshal(data, &placement); err != nil {
		return Placement{}, false, fmt.Errorf("decode saved placement: %w", err)
	}
	limits := DefaultPlacementLimits()
	if placement.Version != 1 || !ValidPlacementSize(placement.Width, placement.Height, limits) || !ValidPlacementPosition(placement.X, placement.Y) {
		return placement, false, nil
	}
	return placement, true, nil
}

// SavePlacement atomically writes placement data to a caller-supplied path with
// owner-only file permissions. The containing directory is created if needed.
func SavePlacement(path string, placement Placement) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("placement file path is required")
	}
	limits := DefaultPlacementLimits()
	if !ValidPlacementSize(placement.Width, placement.Height, limits) || !ValidPlacementPosition(placement.X, placement.Y) {
		return errors.New("window placement is outside the supported range")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	placement.Version = 1
	data, err := json.Marshal(placement)
	if err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+"-*.tmp")
	if err != nil {
		return err
	}
	temporary := file.Name()
	defer os.Remove(temporary)
	if err := file.Chmod(0o600); err != nil {
		_ = file.Close()
		return err
	}
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(temporary, path)
}
