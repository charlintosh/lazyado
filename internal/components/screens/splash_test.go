package screens

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/ansi"
)

func TestLogoGridsAreRectangular(t *testing.T) {
	for size, grid := range logoGrids {
		if len(grid)%2 != 0 {
			t.Errorf("logo %d has %d pixel rows, want an even count", size, len(grid))
		}
		for y, row := range grid {
			if len(row) != size {
				t.Errorf("logo %d row %d is %d wide, want %d", size, y, len(row), size)
			}
			if bad := strings.Trim(row, ".abcwz"); bad != "" {
				t.Errorf("logo %d row %d has unknown pixels %q", size, y, bad)
			}
		}
	}
}

func TestLayoutPicksTheLargestLogoThatFits(t *testing.T) {
	tests := []struct {
		width, height int
		mode          SplashMode
		want          splashLayout
	}{
		{width: 200, height: 60, mode: SplashAnimated, want: splashLayout{logo: 48}},
		{width: 100, height: 34, mode: SplashStatic, want: splashLayout{logo: 48}},
		{width: 80, height: 24, mode: SplashAnimated, want: splashLayout{logo: 32}},
		{width: 64, height: 16, mode: SplashAnimated, want: splashLayout{logo: 24, side: true}},
		{width: 40, height: 10, mode: SplashAnimated, want: splashLayout{}},
		{width: 200, height: 60, mode: SplashOff, want: splashLayout{}},
	}
	for _, tt := range tests {
		if got := layoutFor(tt.width, tt.height, tt.mode); got != tt.want {
			t.Errorf("layoutFor(%d, %d, %s) = %+v, want %+v", tt.width, tt.height, tt.mode, got, tt.want)
		}
	}
}

func TestSplashFillsTheTerminalExactly(t *testing.T) {
	sizes := [][2]int{{200, 60}, {100, 34}, {80, 24}, {64, 16}, {56, 14}, {40, 10}}
	for _, mode := range []SplashMode{SplashAnimated, SplashStatic, SplashOff} {
		for _, size := range sizes {
			s := NewSplashScreen().WithMode(mode)
			s.SetSize(size[0], size[1])
			for range 30 {
				s, _ = s.Update(FrameMsg{})
			}
			view := s.View()
			if got := lipgloss.Height(view); got != size[1] {
				t.Errorf("%s at %dx%d: %d lines, want %d", mode, size[0], size[1], got, size[1])
			}
			for i, line := range strings.Split(view, "\n") {
				if w := ansi.StringWidth(line); w > size[0] {
					t.Errorf("%s at %dx%d: line %d is %d wide", mode, size[0], size[1], i, w)
				}
			}
		}
	}
}

func TestRevealPaintsRowsTopToBottom(t *testing.T) {
	art := buildLogo(32, false)
	hidden := strings.Repeat(" ", art.size)
	lines := strings.Split(art.view(4, -1), "\n")
	for r := 0; r < 4; r++ {
		if lines[r] != art.rows[r] {
			t.Errorf("row %d is not painted after the scanline passed it", r)
		}
	}
	if lines[4] != art.glow[4] {
		t.Error("the scanline row is not drawn in the glow colour")
	}
	for r := 5; r < art.height(); r++ {
		if lines[r] != hidden {
			t.Errorf("row %d is painted before the scanline reaches it", r)
		}
	}
}

func TestStaticSplashNeverSchedulesFrames(t *testing.T) {
	s := NewSplashScreen().WithMode(SplashStatic)
	s.SetSize(100, 34)
	if _, cmd := s.Update(FrameMsg{}); cmd != nil {
		t.Error("a static splash asked for another frame")
	}
	animated := NewSplashScreen()
	animated.SetSize(100, 34)
	if _, cmd := animated.Update(FrameMsg{}); cmd == nil {
		t.Error("an animated splash stopped asking for frames")
	}
}

func TestNoColorDrawsShapesWithoutBackgrounds(t *testing.T) {
	s := NewSplashScreen()
	s.SetProfile(colorprofile.ASCII)
	s.SetSize(100, 34)
	for _, row := range s.art.rows {
		if strings.Contains(row, "[48;") || strings.Contains(row, ";48;") {
			t.Fatalf("a NO_COLOR row relies on a background colour: %q", row)
		}
	}
}

func TestParseSplashMode(t *testing.T) {
	tests := map[string]SplashMode{"": SplashAnimated, "animated": SplashAnimated, " Static ": SplashStatic, "OFF": SplashOff, "bogus": SplashAnimated}
	for in, want := range tests {
		if got := ParseSplashMode(in); got != want {
			t.Errorf("ParseSplashMode(%q) = %s, want %s", in, got, want)
		}
	}
}
