//go:build chrome

package ui_test

import (
	"encoding/json"
	"fmt"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
	"github.com/lantervale/ember/ui/internal/fonts"
)

// TestLineHeightsAndBaselinesMatchChromesAtEverySize lays a line of each
// face out from 8 to 48 pixels by halves, at one and two device pixels a
// point, and has its height and its baseline's place be Chrome's to the
// layout unit: the face's ascent, descent and line gap rounded each to
// device pixels, half the leading floored to them, a line height floored
// to the layout unit (decision 0217). The machine's faces are macOS's,
// where Chrome stands Times and Helvetica taller (Courier too, but by a
// device pixel more or less as the page around it changes); Chrome on
// Linux and Windows measures by FreeType and DirectWrite, a device pixel
// apart at some sizes.
func TestLineHeightsAndBaselinesMatchChromesAtEverySize(t *testing.T) {
	families := []string{"Go", "Go Mono", "Noto", "Overridden", "Adjusted"}
	if runtime.GOOS == "darwin" {
		families = append(families, "Arial", "Georgia", "Times New Roman", "Verdana", "Menlo",
			"Times", "Helvetica")
	}
	for _, scale := range []float32{1, 2} {
		for _, lh := range []string{"normal", "1.25", "1.37", "13.3px"} {
			t.Run(fmt.Sprintf("%gx line-height %s", scale, lh), func(t *testing.T) {
				lineMetricsMatch(t, families, scale, lh)
			})
		}
	}
}

func lineMetricsMatch(t *testing.T, families []string, scale float32, lh string) {
	t.Helper()
	type line struct {
		family    string
		size      float32
		div, mark *ui.Node
	}
	d := ui.NewDocument(ui.SystemFonts(), ui.FetchFonts(foundry(t)), ui.CSS(`
@font-face { font-family: Noto; src: url(hebrew.ttf); }
@font-face { font-family: Overridden; src: url(hebrew.ttf); ascent-override: 93.3%; descent-override: 21.7%;
  line-gap-override: 5.3%; }
@font-face { font-family: Adjusted; src: url(hebrew.ttf); size-adjust: 87%; }
body { margin: 0 } div { width: 300px; line-height: `+lh+` } .m { display: inline-block; width: 0; height: 0 }`))
	var lines []line
	for _, family := range families {
		for size := float32(8); size <= 48; size += 0.5 {
			l := line{family: family, size: size, mark: ui.Span(ui.Class("m"))}
			l.div = ui.Div(ui.Style(fmt.Sprintf("font-family: %q; font-size: %gpx", family, size)), ui.Text("Hxg"), l.mark)
			d.Body().Append(l.div)
			lines = append(lines, l)
		}
	}
	size := geom.V2(320, 9000)
	landed(t, d, size)
	out, err := chrome.EvalScaled(d, size, scale, `return [...document.querySelectorAll("div")].map(d => {
	const a = d.getBoundingClientRect();
	return [a.height, d.querySelector(".m").getBoundingClientRect().top - a.top];
});`)
	require.NoError(t, err)
	var want [][2]float32
	require.NoError(t, json.Unmarshal(out, &want))
	require.Len(t, want, len(lines))
	fonts.Of(d).SetScale(scale)
	defer fonts.Of(d).SetScale(1)
	d.Layout(size)
	within := float32(1) / 64 / scale
	if runtime.GOOS != "darwin" {
		within = 1 / scale
	}
	for i, l := range lines {
		box, ok := d.BoxOf(l.div)
		require.True(t, ok)
		mark, ok := d.BoxOf(l.mark)
		require.True(t, ok)
		what := fmt.Sprintf("%s at %gpx", l.family, l.size)
		assert.InDelta(t, want[i][0], box.H, float64(within), "%s: the line's height", what)
		assert.InDelta(t, want[i][1], mark.Y-box.Y, float64(within), "%s: its baseline", what)
	}
}
