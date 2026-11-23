package css //nolint:testpackage // reads computed styles off dom with the package's test helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

func TestAMaskIsReadAsABackgroundIsLayerByLayer(t *testing.T) {
	s := gridOf(t, "mask-image: url(https://x/i.svg), linear-gradient(#000, transparent); mask-size: 12px 50%; "+
		"mask-repeat: no-repeat, space round; mask-position: right 2px top 50%, center; mask-mode: luminance; "+
		"mask-origin: padding-box, content-box; mask-clip: no-clip; mask-composite: subtract, exclude")
	require.Equal(t, 2, s.MaskLayers())
	top, bottom := s.MaskLayer(0), s.MaskLayer(1)
	assert.Equal(t, "https://x/i.svg", top.Image.Source)
	assert.NotNil(t, bottom.Image.Gradient, "a gradient is a mask")
	assert.Equal(t, style.Px(12), bottom.Size.Width, "a shorter list starts again")
	assert.InDelta(t, 50, bottom.Size.Height.Percent, 1e-6)
	assert.Equal(t, style.NoRepeat, top.Repeat)
	assert.Equal(t, style.BackgroundRepeat{X: style.TileSpace, Y: style.TileRound}, bottom.Repeat)
	if p := top.Position; p.X != style.Px(2) || !p.FromRight || p.Y.Percent != 50 || p.FromBottom {
		t.Errorf("mask-position %v, want 2px from the right, 50%%", p)
	}
	assert.Equal(t, style.MaskLuminance, bottom.Mode)
	assert.Equal(t, style.UnderPadding, top.Origin)
	assert.Equal(t, style.UnderContent, bottom.Origin)
	assert.Equal(t, style.NoClip, bottom.Clip)
	assert.Equal(t, style.CompositeSubtract, top.Composite)
	assert.Equal(t, style.CompositeExclude, bottom.Composite)
	assert.True(t, s.Masked())
	assert.Equal(t, style.Image{}, s.BackgroundLayer(0).Image, "a mask is no background")
}

func TestAMaskOfNoPictureMasksNothing(t *testing.T) {
	s := gridOf(t, "mask-image: url(a.png); mask-repeat: repeat-y; mask-image: none")
	assert.False(t, s.Masked())
	assert.Equal(t, style.BackgroundRepeat{X: style.TileNoRepeat}, s.MaskLayer(0).Repeat)
	assert.False(t, gridOf(t, "mask-image: none, none").Masked())
	assert.True(t, gridOf(t, "mask-image: none, url(a.png)").Masked())

	initial := gridOf(t, "")
	l := initial.MaskLayer(0)
	assert.Equal(t, style.TopLeft, l.Position)
	assert.Equal(t, style.InitialBox, l.Origin, "the border box, for a mask")
	assert.Equal(t, style.MaskMatchSource, l.Mode)
	assert.Equal(t, style.CompositeAdd, l.Composite)
}

func TestTheMaskShorthandSetsEachLayersLonghands(t *testing.T) {
	s := gridOf(t, "mask-mode: luminance; mask: linear-gradient(red, blue) center / 10px no-repeat padding-box "+
		"content-box subtract alpha, url(x.png) view-box")
	require.Equal(t, 2, s.MaskLayers())
	top, bottom := s.MaskLayer(0), s.MaskLayer(1)
	assert.NotNil(t, top.Image.Gradient)
	assert.InDelta(t, 50, top.Position.X.Percent, 1e-6)
	assert.InDelta(t, 50, top.Position.Y.Percent, 1e-6)
	assert.Equal(t, style.Px(10), top.Size.Width)
	assert.Equal(t, style.NoRepeat, top.Repeat)
	assert.Equal(t, style.UnderPadding, top.Origin)
	assert.Equal(t, style.UnderContent, top.Clip)
	assert.Equal(t, style.CompositeSubtract, top.Composite)
	assert.Equal(t, style.MaskAlpha, top.Mode)
	assert.Equal(t, "x.png", bottom.Image.Source)
	assert.Equal(t, style.ViewBox, bottom.Origin, "one box is the origin and the clip")
	assert.Equal(t, style.ViewBox, bottom.Clip)
	assert.Equal(t, style.MaskMatchSource, bottom.Mode, "what a layer leaves out is reset")
	assert.Equal(t, style.CompositeAdd, bottom.Composite)

	s = gridOf(t, "mask: url(x.png) border-box no-clip")
	assert.Equal(t, style.NoClip, s.MaskLayer(0).Clip)
	assert.False(t, gridOf(t, "mask-image: url(a.png); mask: none").Masked())
}

func TestWhatAMaskDoesNotDrawIsRefused(t *testing.T) {
	for _, decl := range []string{
		"mask-position: right 2px center",
		"mask-composite: source-over",
		"mask-mode: luminosity",
		"mask-clip: margin-box",
		"mask-origin: no-clip",
		"mask: url(a.png) no-clip border-box",
		"mask: url(a.png) url(b.png)",
		"mask: url(a.png) / 10px",
		"mask-image: url(a.png), ",
	} {
		_, warns := Parse("t.css", "div { "+decl+" }")
		assert.Len(t, warns, 1, decl)
	}
}

func TestChromesOwnMaskNamesAreReadAsTheirPlainOnes(t *testing.T) {
	s := computed(t, `-webkit-mask: url(a.png) center / 10px no-repeat padding-box; -webkit-mask-position: 3px 4px;
		-webkit-mask-composite: xor; -webkit-mask-size: contain`)
	l := s.MaskLayer(0)
	assert.Equal(t, "a.png", l.Image.Source)
	assert.Equal(t, style.BackgroundPosition{X: style.Px(3), Y: style.Px(4)}, l.Position)
	assert.Equal(t, style.ScaleContain, l.Size.Scale)
	assert.Equal(t, style.UnderPadding, l.Clip)
	assert.Equal(t, style.CompositeXor, l.Composite, "xor is kept as written")
	assert.Equal(t, style.CompositeExclude, l.Composite.Plain(), "and joins as exclude")
	for word, as := range map[string]style.MaskComposite{
		"source-over": style.CompositeSourceOver, "destination-in": style.CompositeDestinationIn,
		"COPY": style.CompositeCopy, "plus-lighter": style.CompositePlusLighter,
	} {
		assert.Equal(t, as, computed(t, "-webkit-mask-composite: "+word).MaskLayer(0).Composite, word)
	}
	_, warns := Parse("t.css", "div { mask-composite: copy; -webkit-mask-composite: plus-darker; "+
		"-webkit-mask: url(a.png) no-clip }", Browser())
	assert.Len(t, warns, 3, "the plain property refuses the compositing words, as Chrome refuses plus-darker and no-clip")
	l = computed(t, "-webkit-mask: url(a.png) right 4px top / 10px; "+
		"-webkit-mask-position: center bottom 3px, left 2px center").MaskLayer(0)
	half := style.Length{Kind: style.Fixed, Percent: 50, IsPercent: true}
	assert.Equal(t, style.BackgroundPosition{X: half, Y: style.Px(3), FromBottom: true}, l.Position)

	s = computed(t, `-webkit-mask: linear-gradient(#000 0 0) content-box, linear-gradient(#000 0 0);
		-webkit-mask-composite: xor; mask-composite: exclude`)
	require.Equal(t, 2, s.MaskLayers(), "the gradient border")
	assert.Equal(t, style.UnderContent, s.MaskLayer(0).Clip)
	assert.Equal(t, style.CompositeExclude, s.MaskLayer(0).Composite)
	l = computed(t, "-webkit-mask: url(a.png) right 4px top / 10px").MaskLayer(0)
	top := style.Length{Kind: style.Fixed, IsPercent: true}
	assert.Equal(t, style.BackgroundPosition{X: style.Px(4), Y: top, FromRight: true}, l.Position,
		"three values, as Chrome's own longhands take them")
}

// computed is the style of a div under decls, read as a browser's sheet.
func computed(t *testing.T, decls string) *style.Computed {
	t.Helper()
	sh, warns := Parse("b.css", "#d { "+decls+" }", Browser())
	require.Empty(t, warns)
	d := el(dom.Div, "d")
	Compute(page(d), []*Sheet{sh})
	return styleOf(d)
}
