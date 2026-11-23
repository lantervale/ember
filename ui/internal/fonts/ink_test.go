package fonts_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/ui/internal/fonts"
	"github.com/lantervale/ember/ui/internal/style"
)

func TestInterceptsAreWhereDescendersCrossABand(t *testing.T) {
	s := fonts.Builtin()
	f := style.Font{Family: []string{"Go"}, Size: 20, Weight: 400}
	assert.Empty(t, s.Intercepts(f, "ace", 0, 0, 2, 3), "no descender reaches below the baseline")
	in := s.Intercepts(f, "agp", 0, 0, 2, 3)
	require.Len(t, in, 2, "g and p")
	a, g := s.Advance(f, "a"), s.Advance(f, "g")
	assert.Greater(t, in[0][0], a)
	assert.Less(t, in[0][1], a+g)
	assert.Greater(t, in[1][0], a+g-0.5)
	spaced := s.Intercepts(f, "agp", 3, 0, 2, 3)
	assert.InDelta(t, in[1][0]+6, spaced[1][0], 1e-3, "letter spacing moves each glyph after the first")
}
