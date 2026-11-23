package css //nolint:testpackage // reads the parser and the test helpers (computed, componentsOf)

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFontPaletteValuesAreReadAndWrittenBack(t *testing.T) {
	sh, warns := Parse("p.css", `
@font-palette-values --blue { font-family: Layered, "Two Words"; base-palette: 2; override-colors: 0 #00f, 3 red; }
@font-palette-values --dark { font-family: Layered; base-palette: dark; }
@font-palette-values nameless { font-family: Layered; }
@font-palette-values --odd { font-family: Layered; override-colors: 1 currentcolor; }
p { font-palette: --blue; }`)
	require.Len(t, sh.FontPaletteValues(), 3)
	blue := sh.FontPaletteValues()[0]
	assert.Equal(t, "--blue", blue.Name)
	assert.True(t, blue.For("layered"))
	assert.True(t, blue.For("Two Words"))
	assert.False(t, blue.For("Other"))
	assert.Equal(t, 2, blue.Base)
	assert.Equal(t, []PaletteOverride{{0, Color{0, 0, 1, 1}}, {3, Color{1, 0, 0, 1}}}, blue.Overrides)
	assert.True(t, sh.FontPaletteValues()[1].Dark)
	assert.Empty(t, sh.FontPaletteValues()[2].Overrides, "currentcolor is refused")
	assert.Len(t, warns, 3, "a rule without a --name, and the override")
	assert.Contains(t, sh.String(), `@font-palette-values --blue { font-family: "Layered", "Two Words"; base-palette: 2; `+
		`override-colors: 0 rgb(0, 0, 255), 3 rgb(255, 0, 0); }`)
}

func TestFontPaletteIsInheritedAndTakesItsWords(t *testing.T) {
	assert.Equal(t, "dark", computed(t, "font-palette: Dark").Font.Palette)
	assert.Equal(t, "--blue", computed(t, "font-palette: --blue").Font.Palette)
	assert.Empty(t, computed(t, "font-palette: normal").Font.Palette)
	_, warns := Parse("p.css", "p { font-palette: blue; }")
	assert.NotEmpty(t, warns)
}

func TestColourTechsAreReadButSVGs(t *testing.T) {
	for tech, reads := range map[string]bool{"color-COLRv0": true, "color-colrv1": true, "color-sbix": true,
		"color-CBDT": true, "color-SVG": false} {
		assert.Equal(t, reads, FontSource{URL: "f.ttf", Tech: []string{tech}}.Reads(), tech)
		assert.Equal(t, reads, supported(componentsOf("font-tech("+tech+")")), tech)
	}
}
