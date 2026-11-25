//go:build chrome

package layout_test

import (
	"runtime"
	"testing"
	"testing/fstest"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
	"github.com/lantervale/ember/ui/internal/fonts"
)

// Texts no face of the engine has, which the machine's faces draw.
const (
	japanese = "日本語のテキストは、漢字とひらがなとカタカナで書かれます。「括弧」も使います。"
	chinese  = "简体中文的文本由汉字组成，标点符号也是全角的。我们在这里测试换行。"
	korean   = "한국어 텍스트는 한글로 쓰여 있습니다. 줄 바꿈도 확인합니다."
	arabic   = "النص العربي يكتب من اليمين إلى اليسار وتتصل حروفه ببعضها."
)

// fallbackPage is a Document of the machine's fonts, its paragraphs in
// family, each text in its language and one Japanese without any.
func fallbackPage(t *testing.T, family string, texts map[string]string) *ui.Document {
	t.Helper()
	sheet := `body { margin: 8px; font-size: 18px; } p { margin: 0 0 12px; width: 280px; }`
	if family != "" {
		sheet += ` body { font-family: ` + family + `; }`
	}
	d := ui.NewDocument(ui.SystemFonts(), ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(sheet)}}, "page.css"))
	for lang, text := range texts {
		args := []ui.Arg{ui.Text(text)}
		if lang != "" {
			args = append(args, ui.Attr("lang", lang))
		}
		if lang == "ar" {
			args = append(args, ui.Attr("dir", "rtl"))
		}
		d.Body().Append(ui.P(args...))
	}
	fonts.Of(d).Await()
	d.Layout(geom.V2(400, 600))
	fonts.Of(d).Await()
	return d
}

// needs skips t where the machine lacks one of the families.
func needs(t *testing.T, families ...string) {
	t.Helper()
	if runtime.GOOS != "darwin" {
		t.Skip("the platform's fallback is compared on macOS")
	}
	idx := fonts.ScanSystem().Wait()
	for _, f := range families {
		if !idx.Has(f) {
			t.Skipf("%s is not installed", f)
		}
	}
}

// TestTextNoFaceOfTheListHasFallsBackAsInChrome compares the boxes of
// Japanese, Korean and Arabic, in the engine's face, in sans-serif and
// in serif, with Chrome's, which asks macOS for a face that has their
// characters as ui asks the machine's fonts.
func TestTextNoFaceOfTheListHasFallsBackAsInChrome(t *testing.T) {
	needs(t, "Hiragino Sans", "Hiragino Kaku Gothic ProN", "Hiragino Mincho ProN", "Apple SD Gothic Neo",
		"AppleMyungjo", "Geeza Pro")
	for _, c := range []struct {
		name, family string
		texts        map[string]string
		within       float32
	}{
		{"Japanese", "", map[string]string{"ja": japanese, "": japanese}, chrome.Tolerance},
		{"Japanese in sans-serif", "sans-serif", map[string]string{"ja": japanese}, chrome.Tolerance},
		{"Japanese in serif", "serif", map[string]string{"ja": japanese}, chrome.Tolerance},
		{"Korean", "", map[string]string{"ko": korean}, chrome.Tolerance},
		// Chrome's serif is Times, whose ascent stands below AppleMyungjo's
		// line where ui's, Go, sets it a point taller (10-missing.md).
		{"Korean in serif", "serif", map[string]string{"ko": korean}, 2},
		{"Arabic", "", map[string]string{"ar": arabic}, chrome.Tolerance},
		{"Arabic in sans-serif", "sans-serif", map[string]string{"ar": arabic}, chrome.Tolerance},
	} {
		t.Run(c.name, func(t *testing.T) {
			d := fallbackPage(t, c.family, c.texts)
			r := chrome.MatchWithin(t, d, geom.V2(400, 600), c.within)
			t.Logf("%d boxes, max delta %.3f", r.Boxes, r.MaxDelta)
			for _, w := range d.Warnings() {
				t.Error(w)
			}
		})
	}
}

// TestSimplifiedChineseFallsBackAsInChrome compares Chinese: in serif
// macOS's Songti SC draws it on both sides; in sans-serif Chrome draws
// PingFang SC, whose outlines no reader but CoreText's draws, and ui
// Hiragino Sans GB in its place, of other widths and metrics.
func TestSimplifiedChineseFallsBackAsInChrome(t *testing.T) {
	needs(t, "Songti SC", "Hiragino Sans GB")
	for _, c := range []struct {
		name, family string
		within       float32
	}{
		{"in serif", "serif", chrome.Tolerance},
		{"in the engine's face", "", 6},
	} {
		t.Run(c.name, func(t *testing.T) {
			d := fallbackPage(t, c.family, map[string]string{"zh-Hans": chinese})
			r := chrome.MatchWithin(t, d, geom.V2(400, 600), c.within)
			t.Logf("%d boxes, max delta %.3f", r.Boxes, r.MaxDelta)
		})
	}
}
