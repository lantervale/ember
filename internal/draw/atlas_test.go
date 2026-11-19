package draw //nolint:testpackage // white-box: reaches draw's unexported state

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAHugeFontsAtlasStaysWithinWhatADeviceTakes(t *testing.T) {
	f := Regular.Size(400)
	assert.LessOrEqual(t, f.Atlas.Rect.Dx(), MaxAtlasSide)
	assert.LessOrEqual(t, f.Atlas.Rect.Dy(), MaxAtlasSide)

	g := f.glyphAt('G', 0)
	assert.False(t, g.rect.Empty(), "and it still draws its glyphs")
	assert.True(t, g.rect.In(f.Atlas.Rect))
	assert.InDelta(t, Regular.Size(40).glyphAt('G', 0).advance*10, g.advance, 1, "at the font's advances")
}

func TestAFullAtlasDrawsNoNewGlyphAndKeepsItsAdvance(t *testing.T) {
	f := Regular.Size(1500)
	var drawn, left int
	for r := 'A'; r <= 'z'; r++ {
		g := f.glyphAt(r, 0)
		assert.Positive(t, g.advance, "%c", r)
		if g.rect.Empty() {
			left++
		} else {
			drawn++
			assert.True(t, g.rect.In(f.Atlas.Rect), "%c", r)
		}
	}
	assert.LessOrEqual(t, f.Atlas.Rect.Dx(), MaxAtlasSide)
	assert.LessOrEqual(t, f.Atlas.Rect.Dy(), MaxAtlasSide)
	assert.Positive(t, drawn, "the first glyphs have their room")
	assert.Positive(t, left, "the atlas is full before the alphabet ends")
}

func TestAFontOfAnOrdinarySizeKeepsItsAtlasAndItsGlyphsReady(t *testing.T) {
	f := Regular.Size(16)
	assert.Equal(t, nextPow2(16*(f.cell+2)), f.Atlas.Rect.Dx())
	assert.True(t, f.ascii[int('a')*phases].ready, "its ASCII is rasterized when it is made")
}
