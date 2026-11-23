package draw //nolint:testpackage // white-box: reaches the pictures a text draws

import (
	"image"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/internal/colourfont"
	"github.com/lantervale/ember/internal/colourfont/colourtest"
	"github.com/lantervale/ember/internal/colourfont/svgglyph"
	"github.com/lantervale/ember/render"
)

// colourPictures are the RGBA textures c draws, in order.
func colourPictures(c *List) []*render.Texture {
	var out []*render.Texture
	for _, b := range c.Batches {
		for i := range b.ImageCount {
			if tex := b.Images[i]; tex != nil && tex.Image != nil {
				if _, rgba := tex.Image.(*image.RGBA); rgba {
					out = append(out, tex)
				}
			}
		}
	}
	return out
}

func TestAColourGlyphIsDrawnAsItsPictureAtItsAdvance(t *testing.T) {
	face, err := ParseFace("painted", colourtest.COLRv1())
	require.NoError(t, err)
	s := Style{Face: face, Size: 40, Color: color.Black}
	var c List
	c.Text(geom.Vec2{}, "AxA", s)
	pics := colourPictures(&c)
	require.Len(t, pics, 1, "both A share one picture")
	assert.InDelta(t, Style{Face: Regular, Size: 40}.Measure("AxA").X, s.Measure("AxA").X, 0.01,
		"measured with the outlines' advances")

	var stroked List
	st := s
	st.Stroke = 2
	stroked.Text(geom.Vec2{}, "A", st)
	assert.Empty(t, colourPictures(&stroked), "a stroke draws nothing of a colour glyph, as Chrome")
}

func TestAColourGlyphTakesItsLooksPaletteAndTheTextsColour(t *testing.T) {
	face, err := ParseFace("layered", colourtest.COLRv0())
	require.NoError(t, err)
	draw := func(f *Face, col color.Color) *image.RGBA {
		var c List
		c.Text(geom.Vec2{}, "B", Style{Face: f, Size: 40, Color: col})
		pics := colourPictures(&c)
		require.Len(t, pics, 1)
		img, ok := pics[0].Image.(*image.RGBA)
		require.True(t, ok)
		return img
	}
	plain := draw(face, color.Black)
	assert.NotEqual(t, plain.Pix, draw(face, color.RGB(1, 0, 0)).Pix, "B's ring is in the text's colour")
	dark := face.Styled(Look{Palette: &Palette{Dark: true}})
	assert.NotEqual(t, plain.Pix, draw(dark, color.Black).Pix, "in the palette for dark backgrounds")
	over := face.Styled(Look{Palette: &Palette{Overrides: []colourfont.Override{{Index: 1}}}})
	assert.NotEqual(t, plain.Pix, draw(over, color.Black).Pix, "an entry overridden")
	drawn, err := ParseFace("drawn", colourtest.SVG(false))
	require.NoError(t, err)
	var c List
	c.Text(geom.Vec2{}, "A", Style{Face: drawn.Styled(Look{NoSVG: true}), Size: 40})
	assert.Empty(t, colourPictures(&c), "an SVG glyph left to its outline")
	c = List{}
	c.Text(geom.Vec2{}, "A", Style{Face: drawn, Size: 40})
	assert.Empty(t, colourPictures(&c), "to its outline too while no SVG reader is registered")
	prev := colourfont.RegisterSVG(svgglyph.Read)
	t.Cleanup(func() { colourfont.RegisterSVG(prev) })
	drawn, err = ParseFace("drawn", colourtest.SVG(false))
	require.NoError(t, err)
	c = List{}
	c.Text(geom.Vec2{}, "A", Style{Face: drawn, Size: 40})
	assert.Len(t, colourPictures(&c), 1, "else drawn")
}
