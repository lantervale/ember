package fonts //nolint:testpackage // tests unexported internals

import (
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/image/font/gofont/gobold"

	"github.com/lantervale/ember/ui/internal/style"
)

// waiting is a set ticked to start with a face of display whose file
// text has just asked for, its chain and the file.
func waiting(t *testing.T, display Display) (*Set, *Face, *Chain, *File) {
	t.Helper()
	s := Builtin()
	s.Tick(10)
	file := &File{}
	c := s.AddFace(Descriptors{Family: "Heavy", Weight: 400, WeightMax: 400, Display: display}, file)
	f := style.Font{Family: []string{"Heavy", "Go"}, Size: 16, Weight: 400}
	s.Advance(f, "AVATAR")
	require.Equal(t, []*File{file}, s.Wanted())
	s.Tick(10)
	return s, c, s.Chain(f), file
}

var everything = []Range{{Lo: 0, Hi: utf8.MaxRune}} //nolint:gochecknoglobals // a test value

func TestAFacesTextIsHiddenForItsBlockPeriodThenInTheNextFamily(t *testing.T) {
	for _, k := range []struct {
		display Display
		block   float32
	}{{DisplayAuto, 3}, {DisplayBlock, 3}, {DisplayFallback, 0.1}, {DisplayOptional, 0.1}} {
		s, _, ch, _ := waiting(t, k.display)
		assert.Equal(t, everything, ch.Hidden(s), "display %d at once", k.display)
		assert.False(t, s.Tick(10+k.block*0.9))
		assert.True(t, s.Waiting())
		assert.Equal(t, everything, s.Chain(ch.Font()).Hidden(s), "display %d in its block period", k.display)
		assert.True(t, s.Tick(10+k.block+0.01), "display %d: its block period ends", k.display)
		assert.Empty(t, s.Chain(ch.Font()).Hidden(s), "display %d after it", k.display)
	}
	s, _, ch, _ := waiting(t, DisplaySwap)
	assert.Empty(t, ch.Hidden(s), "swap has no block period")
}

func TestAFileLandingInItsSwapPeriodIsUsedAndAfterItNever(t *testing.T) {
	parsed, err := Parse(gobold.TTF)
	require.NoError(t, err)
	for _, k := range []struct {
		display Display
		at      float32
		used    bool
	}{
		{DisplayAuto, 2.9, true}, {DisplayAuto, 60, true}, {DisplaySwap, 60, true},
		{DisplayFallback, 2.9, true}, {DisplayFallback, 3.1, false},
		{DisplayOptional, 0.05, true}, {DisplayOptional, 0.2, false},
	} {
		s, c, ch, file := waiting(t, k.display)
		s.Tick(10 + k.at)
		s.Loaded(file, gobold.TTF, parsed)
		s.Tick(10 + k.at + 0.02)
		got := s.Chain(ch.Font())
		if k.used {
			assert.Equal(t, []*Face{c}, got.Faces()[:1], "display %d landing after %vs", k.display, k.at)
			continue
		}
		assert.NotContains(t, got.Faces(), c, "display %d landing after %vs", k.display, k.at)
		assert.Empty(t, got.Loading(), "the face counts as failed")
		assert.Empty(t, got.Hidden(s))
		assert.False(t, s.Waiting())
	}
}

func TestAFacePastItsSwapPeriodStaysInTheNextFamilyBeforeItsFileLands(t *testing.T) {
	s, c, ch, file := waiting(t, DisplayFallback)
	assert.True(t, s.Tick(13.5))
	assert.True(t, c.expired)
	assert.True(t, s.Ready(ch.Font(), "AVATAR", false), "the text waits for its file no more")
	parsed, err := Parse(gobold.TTF)
	require.NoError(t, err)
	s.Loaded(file, gobold.TTF, parsed)
	assert.NotContains(t, s.Chain(ch.Font()).Faces(), c)
	assert.True(t, s.Ready(ch.Font(), "AVATAR", false))
}

func TestWithoutAClockAFaceIsInTheStartOfItsPeriods(t *testing.T) {
	s := Builtin()
	file := &File{}
	c := s.AddFace(Descriptors{Family: "Heavy", Weight: 400, WeightMax: 400, Display: DisplayOptional}, file)
	f := style.Font{Family: []string{"Heavy", "Go"}, Size: 16, Weight: 400}
	s.Advance(f, "AVATAR")
	parsed, err := Parse(gobold.TTF)
	require.NoError(t, err)
	s.Loaded(file, gobold.TTF, parsed)
	assert.Same(t, c, s.Face(f), "a Document laid out but never drawn takes its fonts whenever they land")

	s = Builtin()
	file = &File{}
	s.AddFace(Descriptors{Family: "Heavy", Weight: 400, WeightMax: 400}, file)
	s.Advance(f, "AVATAR")
	s.Tick(5)
	s.Tick(7.9)
	assert.Equal(t, everything, s.Chain(f).Hidden(s), "timed from the first tick")
	s.Tick(8.1)
	assert.Empty(t, s.Chain(f).Hidden(s))
}
