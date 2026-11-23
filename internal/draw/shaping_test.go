package draw //nolint:testpackage // white-box: reaches draw's unexported state

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

func TestShapedRunsAreKept(t *testing.T) {
	f := featuredFace(t).Size(24)
	f.measure("office office", "", 0)
	n := len(f.shapedRuns)
	f.measure("office office", "", 0)
	assert.Len(t, f.shapedRuns, n, "the same run is shaped once")
	assert.Positive(t, n)
}
