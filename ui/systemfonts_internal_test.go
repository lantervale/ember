package ui

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/image/font/gofont/gomedium"
	"golang.org/x/image/font/gofont/gomediumitalic"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/fonts"
)

// installed is a folder holding the Go Medium family, as a machine's
// font folder would.
func installed(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Go-Medium.ttf"), gomedium.TTF, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Go-Medium-Italic.ttf"), gomediumitalic.TTF, 0o644))
	return dir
}

// named is a Document with a word in each of families, and the words.
func named(opts []DocumentOption, families ...string) (*Document, []*Node) {
	d := NewDocument(opts...)
	var words []*Node
	for _, family := range families {
		w := Span(Style("font-family: "+family), Text("Handgloves in a face"))
		words = append(words, w)
		d.Body().Append(Div(w))
	}
	return d, words
}

func TestWithoutSystemFontsAnInstalledFamilyIsNotLookedUp(t *testing.T) {
	d, words := named(nil, "Georgia, 'Go Medium'", "Go")
	assert.False(t, d.SystemFonts())
	d.Layout(geom.V2(400, 200))
	d.fonts.Await()
	d.Layout(geom.V2(400, 200))
	assert.Equal(t, words[1].Box().W, words[0].Box().W, "the default face draws both")
	assert.Len(t, d.fonts.Faces(), len(fonts.Builtin().Faces()))
	assert.Empty(t, d.Warnings())
}

func TestAnInstalledFamilyDrawsOnceTheScanLands(t *testing.T) {
	d, words := named(nil, "nope, 'go medium', sans-serif", "Go", "'Go Medium'; font-style: italic", "nope, sans-serif")
	size := geom.V2(400, 200)
	d.Layout(size)
	before := words[0].Box().W
	assert.Equal(t, words[1].Box().W, before, "the default face until the machine's fonts are known")

	d.fonts.Scan(fonts.ScanDirs(installed(t)))
	d.fonts.Await()
	d.Layout(size)
	assert.NotEqual(t, before, words[0].Box().W, "laid out again in the installed face")
	assert.Equal(t, before, words[1].Box().W)
	assert.Equal(t, before, words[3].Box().W, "a family nobody has falls to the next name")
	assert.NotEqual(t, words[0].Box().W, words[2].Box().W, "the italic is its own file")
	assert.Empty(t, d.Warnings())

	medium := d.fonts.Face(dom.Data(words[0]).Style.Font)
	assert.Equal(t, "Go Medium", medium.Family)
	assert.Equal(t, gomedium.TTF, medium.TTF)
	assert.Equal(t, gomediumitalic.TTF, d.fonts.Face(dom.Data(words[2]).Style.Font).TTF)
	assert.NotSame(t, d.face(dom.Data(words[1]).Style.Font), d.face(dom.Data(words[0]).Style.Font),
		"paint draws with the face layout measured")
}

func TestAnInstalledFontThatIsNoFontIsAWarning(t *testing.T) {
	dir := installed(t)
	broken := append([]byte(nil), gomedium.TTF...)
	for i := range int(binary.BigEndian.Uint16(broken[4:])) {
		if rec := broken[12+16*i:]; string(rec[:4]) == "cmap" {
			off := binary.BigEndian.Uint32(rec[8:])
			clear(broken[off : off+binary.BigEndian.Uint32(rec[12:])])
		}
	}
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Go-Medium.ttf"), broken, 0o644))
	require.NoError(t, os.Remove(filepath.Join(dir, "Go-Medium-Italic.ttf")))
	d, words := named(nil, "'Go Medium', sans-serif", "Go")
	d.fonts.Scan(fonts.ScanDirs(dir))
	d.fonts.Await()
	d.Layout(geom.V2(400, 200))
	d.Layout(geom.V2(400, 200))
	require.Len(t, d.Warnings(), 1)
	assert.Contains(t, d.Warnings()[0].Error(), "Go-Medium.ttf")
	assert.Equal(t, words[1].Box().W, words[0].Box().W)
}

func TestSystemFontsDrawWithAFontTheMachineHas(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("looks for Georgia, which macOS ships")
	}
	if _, err := os.Stat("/System/Library/Fonts/Supplemental/Georgia.ttf"); err != nil {
		t.Skip("Georgia is not installed")
	}
	d, words := named([]DocumentOption{SystemFonts()}, "georgia, sans-serif", "Go", "GEORGIA; font-weight: bold",
		"Georgia; font-style: italic")
	assert.True(t, d.SystemFonts())
	d.fonts.Await()
	d.Layout(geom.V2(600, 300))
	assert.Empty(t, d.Warnings())
	assert.NotEqual(t, words[1].Box().W, words[0].Box().W, "Georgia is not the default face")
	assert.Greater(t, words[2].Box().W, words[0].Box().W, "the bold is wider")
	assert.NotEqual(t, words[0].Box().W, words[3].Box().W)
	for i, weight := range map[int]int{0: 400, 2: 700, 3: 400} {
		f := d.fonts.Face(dom.Data(words[i]).Style.Font)
		assert.Equal(t, "Georgia", f.Family)
		assert.Equal(t, weight, f.Weight)
		assert.Equal(t, i == 3, f.Italic)
	}
}
