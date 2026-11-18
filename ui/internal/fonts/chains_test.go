package fonts //nolint:testpackage // tests unexported internals

import (
	"testing"

	"github.com/lantervale/ember/ui/internal/style"
	"github.com/stretchr/testify/assert"
)

func TestFontsOfTheSameFamiliesShareOneChainHoweverOftenTheyAreComputed(t *testing.T) {
	s := Builtin()
	s.AddWeb("Heavy", 400, 400, false, nil, &File{})
	first := s.Chain(style.Font{Family: []string{"Heavy", "sans-serif"}, Weight: 400})
	for range 100 {
		again := s.Chain(style.Font{Family: []string{"Heavy", "sans-serif"}, Weight: 400})
		assert.Same(t, first, again, "a style computed again names the same families in another slice")
	}
	assert.Len(t, s.chains, 1)

	assert.NotSame(t, first, s.Chain(style.Font{Family: []string{"Heavy", "sans-serif"}, Weight: 700}))
	assert.NotSame(t, first, s.Chain(style.Font{Family: []string{"Heavy", "sans-serif"}, Weight: 400, Italic: true}))
	assert.NotSame(t, first, s.Chain(style.Font{Family: []string{"Heavy", "serif"}, Weight: 400}))
	assert.NotSame(t, first, s.Chain(style.Font{Family: []string{"Heavy"}, Weight: 400}))
	assert.Len(t, s.chains, 5)
}
