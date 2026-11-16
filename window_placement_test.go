package desktop

import (
	"path/filepath"
	"testing"
)

func TestNormalizePlacementRestoresOrCenters(t *testing.T) {
	limits := DefaultPlacementLimits()
	primary := ScreenRect{Left: 0, Top: 0, Right: 1920, Bottom: 1040}
	secondary := ScreenRect{Left: -1920, Top: 0, Right: 0, Bottom: 1040}
	defaults := WindowSize{Width: 1180, Height: 800}
	tests := []struct {
		name     string
		saved    Placement
		hasSaved bool
		visible  bool
		work     ScreenRect
		want     Placement
	}{
		{"restore visible coordinates", Placement{Version: 1, X: 240, Y: 90, Width: 1180, Height: 800}, true, true, primary, Placement{Version: 1, X: 240, Y: 90, Width: 1180, Height: 800}},
		{"preserve negative monitor coordinates", Placement{Version: 1, X: -1800, Y: 80, Width: 1180, Height: 800}, true, true, secondary, Placement{Version: 1, X: -1800, Y: 80, Width: 1180, Height: 800}},
		{"center off-screen window", Placement{Version: 1, X: 7000, Y: 6000, Width: 1180, Height: 800, Maximized: true}, true, false, primary, Placement{Version: 1, X: 370, Y: 120, Width: 1180, Height: 800}},
		{"center invalid size", Placement{Version: 1, X: 20, Y: 30, Width: 0, Height: -1}, true, true, primary, Placement{Version: 1, X: 370, Y: 120, Width: 1180, Height: 800}},
		{"center mostly hidden window", Placement{Version: 1, X: 1910, Y: 90, Width: 1180, Height: 800}, true, true, primary, Placement{Version: 1, X: 370, Y: 120, Width: 1180, Height: 800}},
		{"clamp oversized visible window", Placement{Version: 1, X: -100, Y: 0, Width: 4096, Height: 1200}, true, true, primary, Placement{Version: 1, X: 0, Y: 0, Width: 1920, Height: 1040}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := NormalizePlacement(test.saved, test.hasSaved, test.visible, test.work, defaults, limits)
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Fatalf("placement = %#v, want %#v", got, test.want)
			}
		})
	}
}

func TestPlacementPersistenceUsesCallerPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "custom", "placement.json")
	placement := Placement{X: -1810, Y: 60, Width: 1120, Height: 740, Maximized: true}
	if err := SavePlacement(path, placement); err != nil {
		t.Fatal(err)
	}
	loaded, exists, err := LoadPlacement(path)
	if err != nil || !exists || loaded != (Placement{Version: 1, X: -1810, Y: 60, Width: 1120, Height: 740, Maximized: true}) {
		t.Fatalf("placement round-trip = %#v, exists %t, err %v", loaded, exists, err)
	}
	if PlacementVisibleInWorkArea(Placement{X: 999999, Y: 999999, Width: 1180, Height: 800}, ScreenRect{Left: 0, Top: 0, Right: 1920, Bottom: 1080}, DefaultPlacementLimits()) {
		t.Fatal("extreme placement was considered visible")
	}
}

func TestNormalizePlacementRequiresValidWorkAreaAndDefaults(t *testing.T) {
	if _, err := NormalizePlacement(Placement{}, false, false, ScreenRect{}, WindowSize{Width: 800, Height: 600}, DefaultPlacementLimits()); err == nil {
		t.Fatal("invalid work area was accepted")
	}
	if _, err := NormalizePlacement(Placement{}, false, false, ScreenRect{Right: 100, Bottom: 100}, WindowSize{}, DefaultPlacementLimits()); err == nil {
		t.Fatal("missing default size was accepted")
	}
}
