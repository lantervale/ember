package fonts_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/ui/internal/fonts"
	"github.com/lantervale/ember/ui/internal/style"
)

func TestInkIsWhereTheGlyphsDraw(t *testing.T) {
	s := fonts.Builtin()
	f := style.Font{Family: []string{"sans-serif"}, Size: 100, Weight: 400}
	left, width, ok := s.Ink(f, "D", style.OrientSideways)
	assert.True(t, ok)
	assert.Greater(t, left, float32(0), "a D stands right of its pen")
	assert.Less(t, left+width, s.Advance(f, "D"), "and short of its advance")
	_, two, _ := s.Ink(f, "DD", style.OrientSideways)
	assert.InDelta(t, s.Advance(f, "D")+width, two, 0.01, "the second one an advance on")
	_, _, ok = s.Ink(f, " ", style.OrientSideways)
	assert.False(t, ok, "a space draws nothing")
}

func TestUprightInkRunsDownItsRoom(t *testing.T) {
	s := fonts.Builtin()
	f := style.Font{Family: []string{"sans-serif"}, Size: 100, Weight: 400}
	m := s.Metrics(f)
	top, length, ok := s.Ink(f, "D", style.OrientUpright)
	assert.True(t, ok)
	assert.InDelta(t, m.Ascent-m.CapHeight, top, 0.5, "a capital's top a cap height above its baseline")
	assert.InDelta(t, m.CapHeight, length, 0.5)
	_, two, _ := s.Ink(f, "DD", style.OrientUpright)
	assert.InDelta(t, m.Ascent+m.Descent+length, two, 0.01, "the second one a room down")
	_, turned, _ := s.Ink(f, "D", style.OrientMixed)
	_, across, _ := s.Ink(f, "D", style.OrientSideways)
	assert.InDelta(t, across, turned, 1e-6, "a Latin letter of mixed lines is turned")
}
