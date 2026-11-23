package fonts

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/internal/varfont/varfonttest"
	"github.com/lantervale/ember/ui/internal/style"
)

func TestAutoLeavesTheInkOfCJKHangulAndSlashesWhole(t *testing.T) {
	for _, r := range "gpyQ,;é" {
		assert.True(t, SkipsInk(r, false), "%q", r)
	}
	for _, r := range "/\\_漢字あア한ᄀ①、〮ＡＧ𠀀†" {
		assert.False(t, SkipsInk(r, false), "%q", r)
		assert.Equal(t, r != '_', SkipsInk(r, true), "%q: all skips every glyph but the low line", r)
	}
}

func TestSkipInkAllCutsHangulWhereAutoDoesNot(t *testing.T) {
	ttf, err := os.ReadFile(filepath.Join("..", "..", "..", "internal", "shape", "testdata", "noto-hangul.ttf"))
	require.NoError(t, err)
	s := Builtin()
	require.NoError(t, s.Add("Hangul", 400, false, ttf))
	f := style.Font{Family: []string{"Hangul"}, Size: 20, Weight: 400}
	assert.Empty(t, s.Intercepts(f, "한글", 0, 0, -7, -6, false))
	all := s.Intercepts(f, "한글", 0, 0, -7, -6, true)
	require.Len(t, all, 2)
	assert.Greater(t, all[1][0], s.Advance(f, "한")-1, "the second syllable after the first")
	mixed := s.Intercepts(f, "a한/b", 0, 0, -7, -6, false)
	assert.Len(t, mixed, 2, "a and b, not the syllable nor the slash")
}

func TestUnderlinesSkipTheFacesOwnSmallCapitals(t *testing.T) {
	s, _ := featured(t)
	f := style.Font{Family: []string{"Featured"}, Size: 20, Weight: 400}
	own := f
	own.Features = "smcp"
	caps := f
	caps.Caps = style.SmallCaps
	want := s.Intercepts(own, "Small jumps", 0, 0, -4, -3, false)
	require.NotEmpty(t, want)
	assert.Equal(t, want, s.Intercepts(caps, "Small jumps", 0, 0, -4, -3, false))
	assert.NotEqual(t, s.Intercepts(f, "Small jumps", 0, 0, -4, -3, false), want, "the lowercase's own glyphs differ")
}

func TestAVariableFontsInkIsSkippedAtTheInstanceDrawn(t *testing.T) {
	s := Builtin()
	loaded(t, s, Descriptors{Family: "Weighty", Weight: 100, WeightMax: 900, AutoWeight: true,
		Stretch: 50, StretchMax: 150, AutoStretch: true}, varfonttest.Weighty())
	plain := style.Font{Family: []string{"Weighty"}, Size: 40, Weight: 400}
	heavy, wide := plain, plain
	heavy.Weight = 900
	wide.Stretch = 150
	in := s.Intercepts(plain, "gjp", 0, 0, 3, 5, false)
	require.Len(t, in, 3)
	grown := float32(varfonttest.Bolder) * 40 / 2048
	bold := s.Intercepts(heavy, "gjp", 0, 0, 3, 5, false)
	require.Len(t, bold, 3)
	assert.InDelta(t, in[0][1]-in[0][0]+2*grown, bold[0][1]-bold[0][0], 0.5, "the g's strokes grown out by the weight")
	assert.Greater(t, bold[1][0], in[1][0]+grown, "the j after a wider g")
	widened := s.Intercepts(wide, "gjp", 0, 0, 3, 5, false)
	require.Len(t, widened, 3)
	assert.InDelta(t, s.Advance(wide, "gj")-s.Advance(plain, "gj"), widened[2][0]-in[2][0], 2,
		"the p after the wider g and j, as the line is drawn")
}
