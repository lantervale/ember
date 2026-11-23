package ui_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/sfnt"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

// verticalFont is cjkFont with vertical metrics and alternates (pages.VerticalFont): the
// brackets 「 and 」 are drawn by < and >, and the ideographic comma and full stop by the
// comma and full stop, each replaced in vertical lines (vert) by V, Λ, ' and °'s glyphs.
// With origins, VORG puts each glyph's baseline verticalOrigin below the top of its room.
func verticalFont(t testing.TB, origins bool) []byte {
	t.Helper()
	f, err := sfnt.Parse(goregular.TTF)
	if err != nil {
		t.Fatal(err)
	}
	var buf sfnt.Buffer
	glyphs := map[rune]uint16{}
	for r := rune(0x20); r < 0x2000; r++ {
		if g, lookupErr := f.GlyphIndex(&buf, r); lookupErr == nil && g != 0 {
			glyphs[r] = uint16(g)
		}
	}
	for i := range rune(26) {
		glyphs[0x4e00+i] = glyphs['A'+i]
	}
	glyphs['「'], glyphs['」'], glyphs['、'], glyphs['。'] = glyphs['<'], glyphs['>'], glyphs[','], glyphs['.']
	var origin uint16
	if origins {
		origin = verticalOrigin
	}
	font, err := pages.VerticalFont(withCmap(t, goregular.TTF, cmap4(glyphs)),
		map[rune]rune{'<': 'V', '>': 'Λ', ',': '\'', '.': '°'}, origin)
	if err != nil {
		t.Fatal(err)
	}
	return font
}

// verticalOrigin is how far below the top of its room a glyph's baseline stands in the
// verticalFont with origins, in its units.
const verticalOrigin = 1400

func verticalDocument(t *testing.T, origins bool, sheet string) *ui.Document {
	t.Helper()
	font := verticalFont(t, origins)
	return ui.NewDocument(ui.FetchFonts(func(_, _ string) ([]byte, error) { return font, nil }),
		ui.CSS(`@font-face { font-family: Vertical; src: url(vertical.ttf); } body { font-family: Vertical; }`+sheet))
}

func TestUprightGlyphsTakeTheirVerticalAdvances(t *testing.T) {
	f, err := sfnt.Parse(verticalFont(t, false))
	require.NoError(t, err)
	var buf sfnt.Buffer
	advanceOf := func(r rune) float32 {
		g, lookupErr := f.GlyphIndex(&buf, r)
		require.NoError(t, lookupErr)
		return float32(pages.VerticalAdvance(uint16(g))) * 20 / 2048
	}
	d := verticalDocument(t, false, `p { writing-mode: vertical-rl; font-size: 20px; margin: 0; }`)
	plain := ui.Span(ui.Text("一丁"))
	bracket := ui.Span(ui.Text("「"))
	turned := ui.Span(ui.Text("ab"))
	d.Body().Append(ui.P(plain, bracket, turned))
	size := geom.V2(400, 300)
	d.Layout(size)
	require.Eventually(t, func() bool {
		d.Layout(size)
		return d.FontsLoading() == 0
	}, 5*time.Second, 5*time.Millisecond, "the font never landed")
	d.Layout(size)
	r, ok := d.BoxOf(plain)
	require.True(t, ok)
	assert.InDelta(t, advanceOf('A')+advanceOf('B'), r.H, 0.02, "the ideographs take their vmtx advances")
	r, ok = d.BoxOf(bracket)
	require.True(t, ok)
	assert.InDelta(t, advanceOf('V'), r.H, 0.02, "the bracket takes its vertical alternate's advance")
	r, ok = d.BoxOf(turned)
	require.True(t, ok)
	assert.Greater(t, r.H, float32(0))
}
