package ui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/fonts"
)

// ideographs is a font folder holding a face of two ideographs and the
// punctuation of Chinese and Japanese, and one of Hangul.
func ideographs(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, file := range []string{"go-han.ttf", "noto-hangul.ttf"} {
		ttf, err := os.ReadFile(filepath.Join("..", "internal", "shape", "testdata", file))
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(dir, file), ttf, 0o644))
	}
	return dir
}

const twoIdeographs = "漢字"

// fallingBack is a Document with a word of two ideographs and one of
// Latin letters and an ideograph, laid out once the machine's fonts,
// those of dir, are scanned, with the system fallback or without.
func fallingBack(t *testing.T, dir string, on bool) (*Document, *Node, *Node) {
	t.Helper()
	d := NewDocument()
	han, mixed := Span(Text(twoIdeographs)), Span(Text("ab漢"))
	d.Body().Append(Div(han), Div(mixed))
	d.fonts.SetSystemFallback(on)
	d.fonts.Scan(fonts.ScanDirs(dir))
	d.fonts.Await()
	d.Layout(geom.V2(400, 200))
	d.Layout(geom.V2(400, 200))
	return d, han, mixed
}

func TestWithoutTheSystemFallbackIdeographsAreMissing(t *testing.T) {
	d, han, _ := fallingBack(t, ideographs(t), false)
	f := dom.Data(han).Style.Font
	assert.False(t, d.face(f).Draws(twoIdeographs))
	assert.NotEqual(t, float32(32), han.Box().W)
}

func TestTheSystemFallbackLaysOutAndDrawsWithTheMachinesFace(t *testing.T) {
	d, han, mixed := fallingBack(t, ideographs(t), true)
	f := dom.Data(han).Style.Font
	assert.InDelta(t, 32, han.Box().W, 0.01, "two ideographs an em wide")
	tf := d.face(f)
	assert.True(t, tf.Draws(twoIdeographs), "paint draws them with the face layout measured")
	assert.False(t, tf.Draws("한"), "the machine's faces draw only what layout gave them")
	assert.Greater(t, mixed.Box().W, float32(16))
	assert.Empty(t, d.Warnings())
}

func TestANewCharacterFallenBackForDrawsWithANewFace(t *testing.T) {
	d, han, _ := fallingBack(t, ideographs(t), true)
	f := dom.Data(han).Style.Font
	before := d.face(f)
	d.Body().Append(Div(Text("한국")))
	d.Layout(geom.V2(400, 200))
	after := d.face(f)
	assert.NotSame(t, before, after)
	assert.True(t, after.Draws("한국"+twoIdeographs))
}

func TestSystemFontsTurnTheFallbackOn(t *testing.T) {
	assert.True(t, NewDocument(SystemFonts()).fonts.SystemFallback())
	assert.False(t, NewDocument().fonts.SystemFallback())
}
