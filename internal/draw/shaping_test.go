package draw //nolint:testpackage // white-box: reaches draw's unexported state

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/image/font/sfnt"

	"github.com/lantervale/ember/geom"
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
	f.items("بتب", "", false, func(it item) { ltr = append(ltr, it) })
	f.items("بتب", "", true, func(it item) { rtl = append(rtl, it) })
	require.Len(t, rtl, 3)
	require.Len(t, ltr, 3)
	for i := range rtl {
		assert.Equal(t, ltr[len(ltr)-1-i].idx, rtl[i].idx, "the glyphs of the text in its order, from its end")
	}
	// hb-shape: beh.fina, teh.medi, beh.init, from the left.
	assert.Equal(t, []sfnt.GlyphIndex{99, 102, 97}, []sfnt.GlyphIndex{rtl[0].idx, rtl[1].idx, rtl[2].idx})
	assert.InDelta(t, (1399+1099+1399)/2048.0*24, f.measure("بتب", "", 0).X, 0.01)

	var brackets []item
	f.items("(ب)", "", true, func(it item) { brackets = append(brackets, it) })
	require.Len(t, brackets, 3)
	assert.Equal(t, '(', brackets[0].r, "the closing bracket drawn first, turned round")
}

func TestAMarkIsDrawnOnItsBase(t *testing.T) {
	f := arabicFace(t).Size(24)
	var its []item
	f.items("é", "", false, func(it item) { its = append(its, it) })
	require.Len(t, its, 2, "the mark is drawn, not left out with its cluster")
	assert.Zero(t, its[1].advance)
	assert.Equal(t, 1, its[0].clusters+its[1].clusters)
	assert.InDelta(t, -690.0/2048*24, its[1].offset.X, 0.01)
	assert.InDelta(t, -1200.0/2048*24, its[1].offset.Y, 0.01)
}

func TestShapedRunsAreKept(t *testing.T) {
	f := featuredFace(t).Size(24)
	f.measure("office office", "", 0)
	n := len(f.shapedRuns)
	f.measure("office office", "", 0)
	assert.Len(t, f.shapedRuns, n, "the same run is shaped once")
	assert.Positive(t, n)
}
