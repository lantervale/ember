package fonts //nolint:testpackage // tests unexported internals

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/internal/draw"
	"github.com/lantervale/ember/internal/varfont/varfonttest"
	"github.com/lantervale/ember/ui/internal/style"
)

func TestAVariableFaceIsShapedAtItsInstance(t *testing.T) {
	s, featuredTTF := featured(t)
	ttf := varfonttest.Build(varfonttest.Options{Source: featuredTTF})
	loaded(t, s, Descriptors{Family: "Varied", Weight: 400, WeightMax: 400, AutoWeight: true,
		Stretch: 100, StretchMax: 100, AutoStretch: true}, ttf)
	plain := style.Font{Family: []string{"Varied"}, Size: 20, Weight: 400}
	bold := plain
	bold.Weight = 900
	unshaped := bold
	unshaped.Features = "-liga"
	// office is o, ffi, c, e shaped: four glyphs, each grown on either side.
	grown := float32(2 * varfonttest.Bolder * 20.0 / 2048)
	assert.InDelta(t, s.Advance(plain, "office")+4*grown, s.Advance(bold, "office"), 1e-3)
	assert.InDelta(t, s.Advance(unshaped, "office")-units(569+569+505-1383, 20)-2*grown, s.Advance(bold, "office"),
		1e-3, "the ligature, at the instance")
}

// featured is the builtin set with Featured, Go with ligatures, small
// capitals, figures and kerning (internal/shape/testdata).
func featured(t *testing.T) (*Set, []byte) {
	t.Helper()
	ttf, err := os.ReadFile(filepath.Join("..", "..", "..", "internal", "shape", "testdata", "go-features.ttf"))
	require.NoError(t, err)
	s := Builtin()
	require.NoError(t, s.Add("Featured", 400, false, ttf))
	return s, ttf
}

// units is n units of the 2048 of Go's em at size.
func units(n, size float32) float32 { return n * size / 2048 }

func TestTextIsMeasuredShaped(t *testing.T) {
	s, _ := featured(t)
	f := style.Font{Family: []string{"Featured"}, Size: 20, Weight: 400}
	plain := f
	plain.Features = "-liga,-kern"
	// f_f_i is 1383 units where f, f and i are 569, 569 and 505.
	assert.InDelta(t, s.Advance(plain, "office")-units(569+569+505-1383, 20), s.Advance(f, "office"), 1e-3)
	assert.InDelta(t, s.Advance(plain, "AV")-units(160, 20), s.Advance(f, "AV"), 1e-3, "class kerning")
	tab := f
	tab.Features = "tnum"
	assert.InDelta(t, units(4*1339, 20), s.Advance(tab, "0123"), 1e-3)
	goFont := style.Font{Family: []string{"Go"}, Size: 20, Weight: 400}
	goPlain := goFont
	goPlain.Features = "smcp,tnum"
	assert.InDelta(t, s.Advance(goFont, "office 0123"), s.Advance(goPlain, "office 0123"), 1e-4,
		"a face without the features measures as it is")
}

func TestSmallCapitalsAreTheFacesWhenItHasThem(t *testing.T) {
	s, _ := featured(t)
	f := style.Font{Family: []string{"Featured"}, Size: 20, Weight: 400, Caps: style.SmallCaps}
	drawn := s.Caps(f, "Small")
	assert.Equal(t, style.NormalCaps, drawn.Caps)
	assert.Equal(t, "smcp|", drawn.Features)
	assert.Nil(t, drawn.SmallCaps("Small"), "nothing synthesised")
	all := f
	all.Caps, all.Features = style.AllPetiteCaps, "tnum|kern=0"
	assert.Equal(t, "smcp,c2sc,tnum|kern=0", s.Caps(all, "Small").Features, "small capitals for petite ones")
	uni := f
	uni.Caps = style.Unicase
	assert.Equal(t, style.UnicaseFeatures, s.Caps(uni, "Uni").Caps)
	runs := s.Caps(uni, "Uni").SmallCaps("Uni")
	require.Len(t, runs, 2)
	assert.Equal(t, capsRun("U", "smcp,c2sc", 20), runs[0], "the capital drawn by the features, not made smaller")
	gof := f
	gof.Family = []string{"Go"}
	assert.Equal(t, style.SmallCaps, s.Caps(gof, "Small").Caps, "Go has none: synthesised")
	assert.Greater(t, s.Advance(f, "small"), s.Advance(style.Font{Family: []string{"Featured"}, Size: 20,
		Weight: 400}, "small"))
}

// capsRun is a run of text in Featured at size with features.
func capsRun(text, features string, size float32) style.CapsRun {
	return style.CapsRun{Text: text, Font: style.Font{Family: []string{"Featured"}, Size: size, Weight: 400,
		Features: features}}
}

func TestASpaceTiesItsWordsWhereTheFaceKernsIt(t *testing.T) {
	s, _ := featured(t)
	assert.True(t, s.SpaceTies(style.Font{Family: []string{"Featured"}, Size: 20, Weight: 400}))
	assert.False(t, s.SpaceTies(style.Font{Family: []string{"Go"}, Size: 20, Weight: 400}))
}

func TestLayoutAndDrawingShapeAlike(t *testing.T) {
	s, ttf := featured(t)
	face, err := draw.ParseFace("Featured", ttf)
	require.NoError(t, err)
	for _, c := range []struct{ text, features string }{
		{"office AVATAR To", ""}, {"Small Caps 1/2", "smcp,c2sc,frac"}, {"0123 office", "tnum,-liga,dlig"},
		{"act", "dlig"},
	} {
		f := style.Font{Family: []string{"Featured"}, Size: 20, Weight: 400, Features: c.features}
		assert.InDelta(t, s.Advance(f, c.text), draw.Style{Face: face, Size: 20, Features: c.features}.Measure(c.text).X,
			1e-3, "%q with %q", c.text, c.features)
	}
}
