//go:build chrome && gpu

package ui_test

import (
	"runtime"
	"testing"

	"github.com/lantervale/ember/ui/internal/chrome"
	"github.com/lantervale/ember/ui/internal/fonts"
)

// TestTheSystemFallbackLooksLikeChrome compares the page of text drawn
// with macOS's faces with Chrome's screenshot of it, where the machine
// has the faces both fall back to. The difference is glyph edges, the
// Han of the line without a language, which Chrome draws in PingFang SC
// and ui in Hiragino Sans GB, and Geeza Pro's Arabic, which Chrome joins
// by its morx table and ui by its presentation forms (0.0104 measured
// with Chrome 158).
func TestTheSystemFallbackLooksLikeChrome(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("the platform's fallback is compared on macOS")
	}
	idx := fonts.ScanSystem().Wait()
	for _, f := range []string{"Hiragino Sans", "Hiragino Kaku Gothic ProN", "Hiragino Mincho ProN", "Apple SD Gothic Neo",
		"Geeza Pro", "Songti SC"} {
		if !idx.Has(f) {
			t.Skipf("%s is not installed", f)
		}
	}
	d := chrome.Within(t, "./internal/pages/system-fallback", 360, 260, 0.012)
	t.Logf("mean %.4f", d.Mean)
}
