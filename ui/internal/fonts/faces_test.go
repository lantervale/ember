package fonts //nolint:testpackage // tests unexported internals

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/goregular"

	"github.com/lantervale/ember/ui/internal/style"
)

func loaded(t *testing.T, s *Set, d Descriptors, ttf []byte) *Face {
	t.Helper()
	f, err := Parse(ttf)
	require.NoError(t, err)
	file := &File{}
	c := s.AddFace(d, file)
	s.Loaded(file, ttf, f)
	return c
}

func TestAFaceIsChosenByWidthBeforeWeight(t *testing.T) {
	s := Builtin()
	normal := loaded(t, s, Descriptors{Family: "W", Weight: 700, WeightMax: 700, Stretch: 100, StretchMax: 100},
		gobold.TTF)
	narrow := loaded(t, s, Descriptors{Family: "W", Weight: 400, WeightMax: 400, Stretch: 75, StretchMax: 75},
		goregular.TTF)
	wide := loaded(t, s, Descriptors{Family: "W", Weight: 400, WeightMax: 400, Stretch: 125, StretchMax: 150},
		gomono.TTF)
	for _, c := range []struct {
		stretch float32
		want    *Face
	}{
		{100, normal}, {0, normal}, {75, narrow}, {50, narrow}, {87.5, narrow}, {112.5, wide}, {200, wide},
		{130, wide},
	} {
		got := s.Face(style.Font{Family: []string{"W"}, Size: 16, Weight: 400, Stretch: c.stretch})
		assert.Same(t, c.want, got, "font-stretch %v%%", c.stretch)
	}
}

func TestAFaceIsMadeBoldOrItalicWhereTheFamilyHasNone(t *testing.T) {
	c := &Face{Weight: 400, WeightMax: 400}
	for _, k := range []struct {
		font         style.Font
		bold, italic bool
	}{
		{style.Font{Weight: 700}, true, false},
		{style.Font{Weight: 600}, true, false},
		{style.Font{Weight: 500}, false, false},
		{style.Font{Weight: 400, Italic: true}, false, true},
		{style.Font{Weight: 700, Italic: true, Synthesis: style.NoSyntheticBold}, false, true},
		{style.Font{Weight: 700, Italic: true, Synthesis: style.NoSyntheticBold | style.NoSyntheticItalic}, false, false},
	} {
		bold, italic := c.Synthetic(k.font)
		assert.Equal(t, k.bold, bold, "%+v", k.font)
		assert.Equal(t, k.italic, italic, "%+v", k.font)
	}
	heavy := &Face{Weight: 100, WeightMax: 900, Italic: true}
	bold, italic := heavy.Synthetic(style.Font{Weight: 700, Italic: true})
	assert.False(t, bold || italic, "a variable face covering the weight, italic already")
}

func TestAFacesMetricsAreItsOverridesScaledBySizeAdjust(t *testing.T) {
	s := Builtin()
	own := loaded(t, s, Descriptors{Family: "Own", Weight: 400, WeightMax: 400}, goregular.TTF)
	over := loaded(t, s, Descriptors{Family: "Over", Weight: 400, WeightMax: 400, Look: Look{
		SizeAdjust: 2, Ascent: Override{Set: true, Of: 0.5}, Descent: Override{Set: true},
		LineGap: Override{Set: true, Of: 0.25},
	}}, goregular.TTF)
	m := s.metricsOf(over, 20)
	assert.InDelta(t, 20, m.Ascent, 0.01)
	assert.InDelta(t, 0, m.Descent, 0.01)
	assert.InDelta(t, 10, m.LineGap, 0.01)
	assert.InDelta(t, 2*s.metricsOf(own, 20).XHeight, m.XHeight, 0.01)
	ownFont := style.Font{Family: []string{"Own"}, Size: 20, Weight: 400}
	overFont := style.Font{Family: []string{"Over"}, Size: 20, Weight: 400}
	assert.InDelta(t, 2*s.Advance(ownFont, "Handgloves"), s.Advance(overFont, "Handgloves"), 0.01)
}

func TestAFaceLeftOutOfTheSheetsDrawsNothing(t *testing.T) {
	s := Builtin()
	heavy := loaded(t, s, Descriptors{Family: "Heavy", Weight: 400, WeightMax: 400}, gobold.TTF)
	font := style.Font{Family: []string{"Heavy", "Go"}, Size: 16, Weight: 400}
	assert.Same(t, heavy, s.Face(font))
	assert.True(t, s.SetWeb(nil))
	assert.False(t, s.SetWeb(nil))
	assert.Equal(t, "Go", s.Face(font).Family)
	assert.NotContains(t, s.Faces(), heavy)
	again := s.AddFace(Descriptors{Family: "Heavy", Weight: 400, WeightMax: 400}, heavy.File())
	assert.Same(t, heavy, again, "the face known, its file landed")
	assert.Same(t, heavy, s.Face(font))
}
