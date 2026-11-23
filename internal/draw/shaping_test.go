package draw //nolint:testpackage // white-box: reaches draw's unexported state

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/image/font/sfnt"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/internal/varfont"
	"github.com/lantervale/ember/internal/varfont/varfonttest"
)

func featuredFace(t *testing.T) *Face {
	t.Helper()
	ttf, err := os.ReadFile(filepath.Join("..", "shape", "testdata", "go-features.ttf"))
	require.NoError(t, err)
	face, err := ParseFace("featured", ttf)
	require.NoError(t, err)
	return face
}

func TestALigatureIsDrawnAsOneGlyph(t *testing.T) {
	face := featuredFace(t)
	var joined, apart List
	joined.Text(geom.Vec2{}, "office", Style{Face: face, Size: 24})
	apart.Text(geom.Vec2{}, "office", Style{Face: face, Size: 24, Features: "-liga"})
	j, _ := quads(&joined)
	a, _ := quads(&apart)
	assert.Len(t, j, 4, "o, ffi, c, e")
	assert.Len(t, a, 6)
	assert.Greater(t, j[1].W, a[1].W*1.5, "the ligature is wider than its f")
}

func TestTextShapedAsOneWithItsNeighboursDrawsTheGlyphsItsClustersStart(t *testing.T) {
	face := featuredFace(t)
	st := Style{Face: face, Size: 24}
	amid := func(_, lead, trail string) Style {
		o := st
		o.Lead, o.Trail = lead, trail
		return o
	}
	var first, rest List
	first.Text(geom.Vec2{}, "of", amid("of", "", "fice"))
	rest.Text(geom.Vec2{}, "fi", amid("fi", "of", "ce"))
	f, _ := quads(&first)
	r, _ := quads(&rest)
	assert.Len(t, f, 2, "o and the ligature ffi")
	assert.Empty(t, r, "its characters are in the ligature before")
	assert.InDelta(t, st.Measure("office").X, amid("of", "", "fice").Measure("of").X+
		amid("fi", "of", "ce").Measure("fi").X+amid("ce", "offi", "").Measure("ce").X, 1e-3, "the parts add up")
	assert.Zero(t, amid("fi", "of", "ce").Measure("fi").X)
	assert.InDelta(t, st.Measure("AV").X-st.Measure("V").X, amid("A", "", "V").Measure("A").X, 1e-3,
		"the pair's kerning is its first glyph's")
	assert.Less(t, amid("A", "", "V").Measure("A").X, st.Measure("A").X)
}

func TestAPairKernedByPlacementMovesItsSecondGlyph(t *testing.T) {
	face := featuredFace(t)
	var kerned, plain List
	kerned.Text(geom.Vec2{}, "To", Style{Face: face, Size: 24})
	plain.Text(geom.Vec2{}, "To", Style{Face: face, Size: 24, Features: "-kern"})
	k, _ := quads(&kerned)
	p, _ := quads(&plain)
	require.Len(t, k, 2)
	// T's advance 90 units shorter, o placed 30 further: 60 of 2048 at 24 px.
	assert.InDelta(t, p[1].X-60.0/2048*24, k[1].X, 0.3)
}

func arabicFace(t *testing.T) *Face {
	t.Helper()
	ttf, err := os.ReadFile(filepath.Join("..", "shape", "testdata", "go-arabic.ttf"))
	require.NoError(t, err)
	face, err := ParseFace("arabic", ttf)
	require.NoError(t, err)
	return face
}

func TestTextRightToLeftIsJoinedInItsOrderAndDrawnFromItsEnd(t *testing.T) {
	f := arabicFace(t).Size(24)
	var ltr, rtl []item
	f.items("بتب", "", false, round{}, func(it item) { ltr = append(ltr, it) })
	f.items("بتب", "", true, round{}, func(it item) { rtl = append(rtl, it) })
	require.Len(t, rtl, 3)
	require.Len(t, ltr, 3)
	for i := range rtl {
		assert.Equal(t, ltr[len(ltr)-1-i].idx, rtl[i].idx, "the glyphs of the text in its order, from its end")
	}
	// hb-shape: beh.fina, teh.medi, beh.init, from the left.
	assert.Equal(t, []sfnt.GlyphIndex{99, 102, 97}, []sfnt.GlyphIndex{rtl[0].idx, rtl[1].idx, rtl[2].idx})
	assert.InDelta(t, (1399+1099+1399)/2048.0*24, f.measure("بتب", "", 0, round{}).X, 0.01)

	var brackets []item
	f.items("(ب)", "", true, round{}, func(it item) { brackets = append(brackets, it) })
	require.Len(t, brackets, 3)
	assert.Equal(t, '(', brackets[0].r, "the closing bracket drawn first, turned round")
}

func TestTextRightToLeftShapedAsOneDrawsTheGlyphsItsClustersStart(t *testing.T) {
	f := arabicFace(t).Size(24)
	glyphs := func(s, lead, trail string) []sfnt.GlyphIndex {
		var out []sfnt.GlyphIndex
		f.items(s, "", true, round{lead: lead, trail: trail}, func(it item) { out = append(out, it.idx) })
		return out
	}
	lamAlef := glyphs("لا", "", "")
	require.Len(t, lamAlef, 1, "lam and alef ligate")
	assert.Equal(t, lamAlef, glyphs("ل", "", "ا"), "the ligature is the lam's, which starts it")
	assert.Empty(t, glyphs("ا", "ل", ""))
	assert.Equal(t, glyphs("ببب", "", "")[:2], glyphs("بب", "ب", ""), "drawn from the end, joined to the lead")
	assert.InDelta(t, f.measure("لا", "", 0, round{}).X, f.measure("ل", "", 0, round{trail: "ا"}).X, 1e-3)
	assert.Zero(t, f.measure("ا", "", 0, round{lead: "ل"}).X)
}

func TestALetterJoinsWithTheTextRoundIt(t *testing.T) {
	f := arabicFace(t).Size(24)
	glyphs := func(s, before, after string) []sfnt.GlyphIndex {
		var out []sfnt.GlyphIndex
		f.items(s, "", false, round{before: before, after: after}, func(it item) { out = append(out, it.idx) })
		return out
	}
	whole := glyphs("ببب", "", "")
	require.Len(t, whole, 3)
	assert.Equal(t, whole[1:2], glyphs("ب", "ب", "ب"), "between two, the medial form")
	assert.Equal(t, whole[:1], glyphs("ب", "", "بب"), "before them, the initial form")
	assert.Equal(t, whole[2:], glyphs("ب", "بب", ""), "after them, the final form")
	assert.Equal(t, glyphs("ب", "", ""), glyphs("ب", "a", " ب"), "a letter that does not join, or a space, joins nothing")
	var middle float32
	f.items("ببب", "", false, round{}, func(it item) {
		if it.idx == whole[1] {
			middle = it.advance
		}
	})
	assert.InDelta(t, middle, Style{Face: arabicFace(t), Size: 24, Before: "ب", After: "ب"}.Measure("ب").X, 0.01,
		"measured as drawn")
}

func TestAVariationSequenceIsDrawnWithItsGlyph(t *testing.T) {
	ttf, err := os.ReadFile(filepath.Join("..", "shape", "testdata", "go-thai.ttf"))
	require.NoError(t, err)
	face, err := ParseFace("thai", ttf)
	require.NoError(t, err)
	f := face.Size(24)
	var its []item
	f.items("a︀b︁a", "", false, round{}, func(it item) { its = append(its, it) })
	require.Len(t, its, 3, "a cluster each, the selectors in them")
	assert.Equal(t, f.glyph('A').idx, its[0].idx, "a with VS1 is the face's A")
	assert.Equal(t, f.glyph('b').idx, its[1].idx, "b with VS2 is its own glyph")
	assert.Equal(t, f.glyph('a').idx, its[2].idx)
}

func TestAMarkIsDrawnOnItsBase(t *testing.T) {
	f := arabicFace(t).Size(24)
	var its []item
	f.items("é", "", false, round{}, func(it item) { its = append(its, it) })
	require.Len(t, its, 2, "the mark is drawn, not left out with its cluster")
	assert.Zero(t, its[1].advance)
	assert.Equal(t, 1, its[0].clusters+its[1].clusters)
	assert.InDelta(t, -690.0/2048*24, its[1].offset.X, 0.01)
	assert.InDelta(t, -1200.0/2048*24, its[1].offset.Y, 0.01)
}

func TestAVariableFontIsShapedAtItsInstance(t *testing.T) {
	ttf := varfonttest.Build(varfonttest.Options{Layout: true})
	face, err := ParseFace("varied", ttf)
	require.NoError(t, err)
	vf, err := varfont.Parse(ttf)
	require.NoError(t, err)
	bold := face.Styled(Look{Variation: &Variation{Font: vf, Coords: vf.Normalize([]float32{900, 100})}}).Size(24)
	var its []item
	bold.items("AVx^$", "", false, round{}, func(it item) { its = append(its, it) })
	require.Len(t, its, 5)
	// What hb-shape --font-size=2048 --variations=wght=900 prints.
	for i, want := range []float32{1246, 1486, 1144, 0, 1486} {
		assert.InDelta(t, want/2048*24, its[i].advance, 1e-3, "glyph %d", i)
	}
	assert.InDelta(t, -792.0/2048*24, its[3].offset.X, 1e-3)
	assert.InDelta(t, -260.0/2048*24, its[3].offset.Y, 1e-3)
	assert.True(t, its[4].shaped, "rvrn draws $ as S")
}

func TestShapedRunsAreKept(t *testing.T) {
	f := featuredFace(t).Size(24)
	f.measure("office office", "", 0, round{})
	n := len(f.shapedRuns)
	f.measure("office office", "", 0, round{})
	assert.Len(t, f.shapedRuns, n, "the same run is shaped once")
	assert.Positive(t, n)
}
