package ui_test

import (
	"encoding/binary"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/sfnt"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
)

// verticalFont is cjkFont with vertical metrics and alternates: each glyph's advance
// down a vertical line (vmtx) is verticalAdvance's, unlike its ascent and descent; the
// brackets 「 and 」 are drawn by < and >, and the ideographic comma and full stop by
// the comma and full stop, each replaced in vertical lines (vert) by V, Λ, ' and °'s
// glyphs. With origins, VORG puts each glyph's baseline verticalOrigin below the top of
// its room.
func verticalFont(t testing.TB, origins bool) []byte {
	t.Helper()
	f, err := sfnt.Parse(goregular.TTF)
	if err != nil {
		t.Fatal(err)
	}
	var buf sfnt.Buffer
	glyph := func(r rune) uint16 {
		g, lookupErr := f.GlyphIndex(&buf, r)
		if lookupErr != nil || g == 0 {
			t.Fatalf("Go Regular has no %q", r)
		}
		return uint16(g)
	}
	glyphs := map[rune]uint16{}
	for r := rune(0x20); r < 0x2000; r++ {
		if g, lookupErr := f.GlyphIndex(&buf, r); lookupErr == nil && g != 0 {
			glyphs[r] = uint16(g)
		}
	}
	for i := range rune(26) {
		glyphs[0x4e00+i] = glyphs['A'+i]
	}
	glyphs['「'], glyphs['」'], glyphs['、'], glyphs['。'] = glyph('<'), glyph('>'), glyph(','), glyph('.')
	be := binary.BigEndian
	n := f.NumGlyphs()
	vhea := make([]byte, 36)
	be.PutUint32(vhea, 0x00011000)
	be.PutUint16(vhea[4:], 1024)
	be.PutUint16(vhea[6:], 0xfc00)
	be.PutUint16(vhea[34:], uint16(n))
	var vmtx []byte
	for g := range n {
		vmtx = be.AppendUint16(vmtx, verticalAdvance(uint16(g)))
		vmtx = be.AppendUint16(vmtx, uint16(100+g%7*40))
	}
	tables := map[string][]byte{"cmap": cmap4(glyphs), "vhea": vhea, "vmtx": vmtx,
		"GSUB": verticalSubstitutions(map[uint16]uint16{
			glyph('<'): glyph('V'), glyph('>'): glyph('Λ'), glyph(','): glyph('\''), glyph('.'): glyph('°'),
		})}
	if origins {
		vorg := []byte{0, 1, 0, 0}
		vorg = be.AppendUint16(vorg, verticalOrigin)
		tables["VORG"] = be.AppendUint16(vorg, 0)
	}
	return withTables(t, goregular.TTF, tables)
}

// verticalAdvance is glyph g's advance down a vertical line in verticalFont, in its
// units (2048 an em): 1,500 to 2,180, where Go's ascent and descent take 2,288.
func verticalAdvance(g uint16) uint16 { return 1500 + g*37%700 }

// verticalOrigin is how far below the top of its room a glyph's baseline stands in the
// verticalFont with origins, in its units.
const verticalOrigin = 1400

// verticalSubstitutions is a GSUB table whose one feature, vert, of the default script,
// replaces each glyph of subs by its value.
func verticalSubstitutions(subs map[uint16]uint16) []byte {
	be := binary.BigEndian
	from := make([]uint16, 0, len(subs))
	for g := range subs {
		from = append(from, g)
	}
	for i := range from {
		for j := i + 1; j < len(from); j++ {
			if from[j] < from[i] {
				from[i], from[j] = from[j], from[i]
			}
		}
	}
	gsub := []byte{0, 1, 0, 0, 0, 10, 0, 30, 0, 44}
	// the script list: DFLT, its default language system with feature 0
	gsub = append(gsub, 0, 1, 'D', 'F', 'L', 'T', 0, 8, 0, 4, 0, 0, 0, 0, 0xff, 0xff, 0, 1, 0, 0)
	// the feature list: vert, of lookup 0
	gsub = append(gsub, 0, 1, 'v', 'e', 'r', 't', 0, 8, 0, 0, 0, 1, 0, 0)
	// the lookup list: one single substitution, format 2, its coverage after it
	gsub = append(gsub, 0, 1, 0, 4, 0, 1, 0, 0, 0, 1, 0, 8)
	k := len(from)
	gsub = be.AppendUint16(gsub, 2)
	gsub = be.AppendUint16(gsub, uint16(6+2*k))
	gsub = be.AppendUint16(gsub, uint16(k))
	for _, g := range from {
		gsub = be.AppendUint16(gsub, subs[g])
	}
	gsub = be.AppendUint16(gsub, 1)
	gsub = be.AppendUint16(gsub, uint16(k))
	for _, g := range from {
		gsub = be.AppendUint16(gsub, g)
	}
	return gsub
}

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
		return float32(verticalAdvance(uint16(g))) * 20 / 2048
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
