//go:build chrome && gpu

package ui_test

import (
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/lantervale/ember/ui/internal/chrome"
)

// TestPagesLookLikeChrome compares pages drawn by ember with Chrome's
// screenshots of the same pages, pixel by pixel. Run with
// -tags 'chrome gpu'; EMBER_UI_REPORTS keeps the reports in a folder. The
// thresholds are today's differences, lowered as text rendering and the
// controls' looks close in on Chrome's.
func TestPagesLookLikeChrome(t *testing.T) {
	for _, c := range []struct {
		pkg  string
		w, h int
		mean float64
	}{
		{"./internal/pages/states", 520, 80, 0.004},
		{"./internal/pages/m0", 800, 600, 0.0002},
		{"./internal/pages/mocka", 520, 720, 0.002},
		{"./internal/pages/glyphs", 400, 320, 0.0029},
		{"./internal/pages/corners", 360, 270, 0.0003},
		{"./internal/pages/dashes", 300, 520, 0.0025},
		{"./internal/pages/insets", 420, 500, 0.0012},
		{"./internal/pages/radii", 520, 190, 0.0003},
		{"./internal/pages/shadows", 420, 280, 0.008},
		{"./internal/pages/markers", 400, 330, 0.0056},
		{"./internal/pages/pixelated", 420, 130, 0.0002},
		{"./internal/pages/scrolled", 300, 320, 0.0125},
		{"./internal/pages/viewport-top", 320, 260, 0.0055},
		{"./internal/pages/viewport-end", 320, 260, 0.006},
		{"./internal/pages/top-layer", 320, 240, 0.002},
		{"./internal/pages/translated", 300, 200, 0.006},
		{"./internal/pages/tooltip-placed", 320, 200, 0.005},
		{"./internal/pages/details-closed", 240, 120, 0.006},
		{"./internal/pages/canvas", 520, 300, 0.0005},
		{"./internal/pages/iframe", 600, 200, 0.003},
		{"./internal/pages/nine-patch", 360, 160, 0.0003},
		{"./internal/pages/border-image-only", 320, 100, 0.0002},
		{"./internal/pages/border-repeat", 420, 130, 0.0017},
		{"./internal/pages/border-image-width", 460, 170, 0.0007},
		{"./internal/pages/flex-negative-margin", 640, 220, 0.0004},
		{"./internal/pages/img-crisp", 160, 104, 0.0002},
		{"./internal/pages/bevels", 480, 120, 0.0001},
		{"./internal/pages/mitres", 560, 90, 0.0002},
		{"./internal/pages/disabled", 480, 110, 0.0024},
		{"./internal/pages/gradients", 520, 240, 0.0009},
		{"./internal/pages/conic", 360, 180, 0.0009},
		{"./internal/pages/radial", 440, 160, 0.0007},
		{"./internal/pages/pixel-crisp", 160, 104, 0.0013},
		{"./internal/pages/field-case", 420, 200, 0.0035},
		{"./internal/pages/broken-image", 420, 160, 0.006},
		{"./internal/pages/rowspan-rows", 460, 160, 0.0002},
		{"./internal/pages/rounded-clip", 560, 330, 0.0028},
		{"./internal/pages/table-caption", 420, 160, 0.003},
		{"./internal/pages/table-model", 640, 360, 0.0025},
		{"./internal/pages/background-boxes", 640, 330, 0.0065},
		{"./internal/pages/debug-grid", 640, 180, 0.004},
		{"./internal/pages/decorations", 420, 560, 0.007},
		{"./internal/pages/word-spacing", 420, 440, 0.0065},
		{"./internal/pages/vertical-align", 420, 620, 0.0092},
		{"./internal/pages/font-smoothing", 420, 520, 0.0018},
		{"./internal/pages/background-size", 660, 330, 0.0011},
		{"./internal/pages/background-layers", 640, 330, 0.003},
		{"./internal/pages/clip-path", 680, 470, 0.0015},
		{"./internal/pages/filters", 600, 420, 0.003},
		{"./internal/pages/blend-modes", 600, 330, 0.0015},
		{"./internal/pages/backdrop-filter", 560, 360, 0.001},
		{"./internal/pages/background-blend", 600, 260, 0.0012},
		{"./internal/pages/svg-pictures", 860, 640, 0.0008},
		{"./internal/pages/multicol", 640, 440, 0.0046},
		{"./internal/pages/generated", 640, 440, 0.0025},
		{"./internal/pages/selection", 560, 560, 0.004},
		{"./internal/pages/ico-bmp", 492, 204, 0.0002},
		{"./internal/pages/math-functions", 520, 330, 0.0006},
		{"./internal/pages/color-scheme", 560, 160, 0.0018},
		{"./internal/pages/doubles", 560, 230, 0.0003},
		{"./internal/pages/collapsed-styles", 520, 160, 0.0015},
		{"./internal/pages/rounded-colours", 480, 90, 0.0008},
	} {
		t.Run(c.pkg, func(t *testing.T) {
			if strings.HasSuffix(c.pkg, "/mitres") && os.Getenv("CI") != "" && runtime.GOOS != "windows" {
				t.Skip("Chrome on the macOS and Linux runners draws the page's background and none of its boxes")
			}
			chrome.Within(t, c.pkg, c.w, c.h, c.mean)
		})
	}
}

func TestZoomedPagesLookLikeChrome(t *testing.T) {
	for _, c := range []struct {
		pkg  string
		w, h int
		zoom float32
		mean float64
	}{
		{"./internal/pages/zoom-snap", 232 * 3, 101 * 3, 3, 0.0002},
		{"./internal/pages/img-zoomed", 240, 156, 1.5, 0.0002},
		{"./internal/pages/img-zoomed", 400, 260, 2.5, 0.0002},
	} {
		t.Run(c.pkg, func(t *testing.T) {
			chrome.WithinZoomed(t, c.pkg, c.w, c.h, c.zoom, c.mean)
		})
	}
}
