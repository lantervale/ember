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

	assert.Equal(t, box.HanGlyph{Halt: true, Shift: -10, Narrow: -10, Full: true}, s.HanGlyph(f, '「', false))
	assert.Equal(t, box.HanGlyph{Halt: true, Narrow: -10, Ink: box.InkLeft, Full: true}, s.HanGlyph(f, '」', false))
	assert.Equal(t, box.HanGlyph{Halt: true, Narrow: -10, Ink: box.InkLeft, Full: true}, s.HanGlyph(f, '。', false))
	assert.Equal(t, box.HanGlyph{Halt: true, Shift: -5, Narrow: -10, Ink: box.InkCentred, Full: true},
		s.HanGlyph(f, '：', false))
	assert.Equal(t, box.HanGlyph{Halt: true, Full: true}, s.HanGlyph(f, '漢', false), "halt leaves an ideograph")
	assert.Equal(t, box.HanGlyph{}, s.HanGlyph(f, '“', false), "a character the face lacks")

	goFace := style.Font{Family: []string{"Go"}, Size: 20, Weight: 400}
	assert.False(t, s.HanGlyph(goFace, '(', false).Halt, "Go has no halt")
}

func TestAFacesChwsSpacesAPairOfItsOwn(t *testing.T) {
	s := Builtin()
	for _, face := range []struct{ family, file string }{{"Han", "go-han.ttf"}, {"Chws", "go-han-chws.ttf"}} {
		ttf, err := os.ReadFile(filepath.Join("..", "..", "..", "internal", "shape", "testdata", face.file))
		require.NoError(t, err)
		require.NoError(t, s.Add(face.family, 400, false, ttf))
	}
	f := style.Font{Family: []string{"Chws"}, Size: 20, Weight: 400}

	a, b, ok := s.HanPair(f, '」', f, '「')
	require.True(t, ok)
	assert.Equal(t, box.HanAdjust{Narrow: -10}, a, "the closing bracket, where Chrome's rules trim the opening one")
	assert.Equal(t, box.HanAdjust{}, b)
	a, b, ok = s.HanPair(f, '「', f, '「')
	require.True(t, ok)
	assert.Equal(t, box.HanAdjust{}, a)
	assert.Equal(t, box.HanAdjust{Shift: -10, Narrow: -10}, b)
	a, b, ok = s.HanPair(f, '。', f, '「')
	require.True(t, ok)
	assert.Equal(t, [2]box.HanAdjust{}, [2]box.HanAdjust{a, b}, "a pair its chws leaves")

	_, _, ok = s.HanPair(style.Font{Family: []string{"Han"}, Size: 20, Weight: 400}, '」', f, '「')
	assert.False(t, ok, "two faces")
	big := f
	big.Size = 30
	_, _, ok = s.HanPair(f, '」', big, '「')
	assert.False(t, ok, "two sizes")
	han := style.Font{Family: []string{"Han"}, Size: 20, Weight: 400}
	_, _, ok = s.HanPair(han, '」', han, '「')
	assert.False(t, ok, "a face without chws")
}

func TestAnUprightGlyphsTrimIsItsVhal(t *testing.T) {
	idx := Scan("/System/Library/Fonts")
	s := Builtin()
	s.SetInstalled(idx)
	f := style.Font{Family: []string{"Hiragino Sans"}, Size: 20, Weight: 400}
	if !s.HanGlyph(f, '「', false).Halt {
		t.Skip("Hiragino Sans is not installed")
	}
	assert.Equal(t, box.HanGlyph{Halt: true, Shift: -10, Narrow: -10, Full: true}, s.HanGlyph(f, '「', true),
		"moved up the line by half its height")
	assert.Equal(t, box.HanGlyph{Halt: true, Narrow: -10, Ink: box.InkLeft, Full: true}, s.HanGlyph(f, '」', true))
}
