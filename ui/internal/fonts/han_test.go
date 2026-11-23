package fonts //nolint:testpackage // tests unexported internals

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/style"
)

func TestAGlyphsHaltAndInkAreReadFromItsFace(t *testing.T) {
	ttf, err := os.ReadFile(filepath.Join("..", "..", "..", "internal", "shape", "testdata", "go-han.ttf"))
	require.NoError(t, err)
	s := Builtin()
	require.NoError(t, s.Add("Han", 400, false, ttf))
	f := style.Font{Family: []string{"Han"}, Size: 20, Weight: 400}

	assert.Equal(t, box.HanGlyph{Halt: true, Shift: -10, Narrow: -10, Full: true}, s.HanGlyph(f, '「'))
	assert.Equal(t, box.HanGlyph{Halt: true, Narrow: -10, Ink: box.InkLeft, Full: true}, s.HanGlyph(f, '」'))
	assert.Equal(t, box.HanGlyph{Halt: true, Narrow: -10, Ink: box.InkLeft, Full: true}, s.HanGlyph(f, '。'))
	assert.Equal(t, box.HanGlyph{Halt: true, Shift: -5, Narrow: -10, Ink: box.InkCentred, Full: true},
		s.HanGlyph(f, '：'))
	assert.Equal(t, box.HanGlyph{Halt: true, Full: true}, s.HanGlyph(f, '漢'), "halt leaves an ideograph")
	assert.Equal(t, box.HanGlyph{}, s.HanGlyph(f, '“'), "a character the face lacks")

	goFace := style.Font{Family: []string{"Go"}, Size: 20, Weight: 400}
	assert.False(t, s.HanGlyph(goFace, '(').Halt, "Go has no halt")
}
