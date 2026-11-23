//go:build chrome && gpu

package ui_test

import (
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
		{"./internal/pages/dashes", 300, 520, 0.001},
		{"./internal/pages/insets", 420, 500, 0.0012},
		{"./internal/pages/radii", 520, 190, 0.0003},
		{"./internal/pages/shadows", 420, 280, 0.008},
		{"./internal/pages/markers", 400, 330, 0.0056},
		{"./internal/pages/pixelated", 420, 130, 0.0002},
		{"./internal/pages/scrolled", 300, 320, 0.0125},
		{"./internal/pages/scrollbar-parts", 480, 230, 0.0003},
		{"./internal/pages/scrollbar-gutter", 560, 210, 0.0015},
		{"./internal/pages/control-parts", 440, 280, 0.001},
		{"./internal/pages/control-generated", 420, 280, 0.003},
		{"./internal/pages/viewport-top", 320, 260, 0.0055},
		{"./internal/pages/viewport-end", 320, 260, 0.006},
		{"./internal/pages/top-layer", 320, 240, 0.002},
		{"./internal/pages/translated", 300, 200, 0.006},
		{"./internal/pages/tooltip-placed", 320, 200, 0.005},
		{"./internal/pages/details-closed", 240, 120, 0.006},
		{"./internal/pages/canvas", 520, 300, 0.0005},
		{"./internal/pages/iframe", 600, 200, 0.003},
		{"./internal/pages/iframe-borders", 560, 140, 0.0001},
		{"./internal/pages/iframe-faded", 640, 140, 0.001},
		{"./internal/pages/nine-patch", 360, 160, 0.0003},
		{"./internal/pages/border-image-only", 320, 100, 0.0002},
		{"./internal/pages/border-repeat", 420, 130, 0.0017},
		{"./internal/pages/border-image-width", 460, 170, 0.0007},
		{"./internal/pages/flex-negative-margin", 640, 220, 0.0004},
		{"./internal/pages/img-crisp", 160, 104, 0.0002},
		{"./internal/pages/img-crisp-animated", 176, 120, 0.004},
		{"./internal/pages/bevels", 480, 120, 0.0001},
		{"./internal/pages/mitres", 560, 90, 0.0002},
		{"./internal/pages/disabled", 480, 110, 0.0024},
		{"./internal/pages/gradients", 520, 240, 0.0009},
		{"./internal/pages/conic", 360, 180, 0.0009},
		{"./internal/pages/radial", 440, 160, 0.0007},
		{"./internal/pages/pixel-crisp", 160, 104, 0.0013},
		{"./internal/pages/field-case", 420, 200, 0.0035},
		{"./internal/pages/broken-image", 420, 280, 0.008},
		{"./internal/pages/rowspan-rows", 460, 160, 0.0002},
		{"./internal/pages/rounded-clip", 560, 330, 0.0028},
		{"./internal/pages/table-caption", 420, 160, 0.003},
		{"./internal/pages/table-model", 640, 360, 0.0025},
		{"./internal/pages/background-boxes", 640, 330, 0.0065},
		{"./internal/pages/debug-grid", 640, 180, 0.004},
		{"./internal/pages/decorations", 420, 560, 0.007},
		{"./internal/pages/decoration-caps", 440, 420, 0.0095},
		{"./internal/pages/underline-position", 560, 450, 0.009},
		{"./internal/pages/word-spacing", 420, 440, 0.0065},
		{"./internal/pages/text-properties", 420, 490, 0.0086},
		{"./internal/pages/vertical-align", 420, 620, 0.0092},
		{"./internal/pages/font-smoothing", 420, 520, 0.0018},
		{"./internal/pages/background-size", 660, 330, 0.0011},
		{"./internal/pages/background-layers", 640, 330, 0.003},
		{"./internal/pages/clip-path", 680, 470, 0.0015},
		{"./internal/pages/filters", 600, 420, 0.003},
		{"./internal/pages/masks", 680, 560, 0.0009},
		{"./internal/pages/blend-modes", 600, 330, 0.0015},
		{"./internal/pages/plus-lighter", 420, 180, 0.0006},
		{"./internal/pages/mask-composite", 460, 260, 0.0004},
		{"./internal/pages/webkit-masks", 380, 180, 0.001},
		{"./internal/pages/backdrop-filter", 560, 360, 0.001},
		{"./internal/pages/url-filters", 500, 410, 0.0045},
		{"./internal/pages/background-blend", 600, 260, 0.0012},
		{"./internal/pages/svg-pictures", 860, 640, 0.0008},
		{"./internal/pages/multicol", 640, 440, 0.0046},
		{"./internal/pages/column-span", 640, 420, 0.004},
		{"./internal/pages/column-cuts", 560, 420, 0.0025},
		{"./internal/pages/column-cuts-more", 560, 420, 0.0035},
		{"./internal/pages/inline-opacity", 420, 240, 0.002},
		{"./internal/pages/table-collapse", 520, 240, 0.0022},
		{"./internal/pages/generated", 640, 440, 0.0025},
		{"./internal/pages/selection", 560, 560, 0.004},
		{"./internal/pages/selection-marks", 420, 560, 0.0065},
		{"./internal/pages/field-selection", 320, 60, 0.004},
		{"./internal/pages/ico-bmp", 492, 264, 0.0002},
		{"./internal/pages/math-functions", 520, 330, 0.0006},
		{"./internal/pages/color-scheme", 560, 160, 0.0018},
		{"./internal/pages/doubles", 560, 230, 0.0003},
		{"./internal/pages/collapsed-styles", 520, 160, 0.0001},
		{"./internal/pages/rounded-colours", 480, 90, 0.0005},
		{"./internal/pages/collapsed-joins", 600, 220, 0.0001},
		{"./internal/pages/rounded-styles", 520, 150, 0.001},
		{"./internal/pages/rounded-dashes", 560, 160, 0.0012},
		{"./internal/pages/outline-styles", 520, 160, 0.001},
		{"./internal/pages/direction", 400, 400, 0.003},
		{"./internal/pages/direction-more", 400, 420, 0.003},
		{"./internal/pages/lists", 520, 300, 0.0032},
		{"./internal/pages/list-inside", 520, 360, 0.0072},
		{"./internal/pages/first-letter-line", 560, 420, 0.0058},
		{"./internal/pages/initial-letter", 520, 260, 0.008},
		{"./internal/pages/first-line-more", 560, 420, 0.0042},
		{"./internal/pages/font-synthesis", 520, 300, 0.007},
		{"./internal/pages/text-stroke", 480, 420, 0.0095},
		{"./internal/pages/marker-properties", 420, 360, 0.0065},
		{"./internal/pages/opentype", 440, 360, 0.0095},
		{"./internal/pages/background-text", 640, 300, 0.0031},
		{"./internal/pages/text-sizes", 420, 240, 0.008},
		{"./internal/pages/font-variations", 520, 320, 0.0065},
		{"./internal/pages/variable-layout", 480, 280, 0.01},
		{"./internal/pages/colour-fonts", 520, 340, 0.0025},
		{"./internal/pages/joined", 440, 400, 0.0055},
		{"./internal/pages/syllables", 420, 400, 0.0058},
		{"./internal/pages/scripts", 420, 400, 0.0045},
		{"./internal/pages/shaped-whole", 460, 320, 0.0088},
		{"./internal/pages/writing-modes", 480, 240, 0.0075},
		{"./internal/pages/vertical-glyphs", 420, 240, 0.0075},
	} {
		t.Run(c.pkg, func(t *testing.T) {
			chrome.Within(t, c.pkg, c.w, c.h, c.mean)
		})
	}
}

// TestFlaggedPagesLookLikeChrome is TestPagesLookLikeChrome for pages of
// what Chrome ships behind a flag, turned on.
func TestFlaggedPagesLookLikeChrome(t *testing.T) {
	for _, c := range []struct {
		pkg     string
		w, h    int
		mean    float64
		feature string
	}{
		{"./internal/pages/line-clamp-past", 420, 360, 0.003, "CSSLineClamp"},
	} {
		t.Run(c.pkg, func(t *testing.T) {
			chrome.WithinWith(t, c.pkg, c.w, c.h, c.mean, c.feature)
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
