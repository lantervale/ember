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
	left, width, ok := s.Ink(f, "D")
	assert.True(t, ok)
	assert.Greater(t, left, float32(0), "a D stands right of its pen")
	assert.Less(t, left+width, s.Advance(f, "D"), "and short of its advance")
	_, two, _ := s.Ink(f, "DD")
	assert.InDelta(t, s.Advance(f, "D")+width, two, 0.01, "the second one an advance on")
	_, _, ok = s.Ink(f, " ")
	assert.False(t, ok, "a space draws nothing")
}
