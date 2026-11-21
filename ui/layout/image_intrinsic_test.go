package layout //nolint:testpackage // tests the layout tree and its unexported helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

// shapes are pictures drawn from shapes, by what each asks to be sized by.
type shapes map[string]box.Intrinsic

func (s shapes) ImageSize(url string) (geom.Vec2, bool) {
	in, ok := s[url]
	return in.Natural(geom.V2(300, 150)), ok
}

func (s shapes) ImageIntrinsic(url string) (box.Intrinsic, bool) {
	in, ok := s[url]
	return in, ok
}

func layShape(in box.Intrinsic, edit func(*style.Computed)) geom.Vec2 {
	i := img("icon.svg", edit)
	root := el(dom.Div, width(400), el(dom.P, block, i))
	tr := Layout(root, geom.V2(600, 300), ahemFonts{}, Images(shapes{"icon.svg": in}))
	return tr.Of[i].Size
}

func wide(v float32) func(*style.Computed) {
	return func(st *style.Computed) { st.Size.Width = style.Px(v) }
}
func high(v float32) func(*style.Computed) {
	return func(st *style.Computed) { st.Size.Height = style.Px(v) }
}

func TestAPictureOfShapesAskingForItsSizeIsLaidOutAsOneOfPixels(t *testing.T) {
	in := box.Intrinsic{Width: 200, Height: 100, Ratio: 2}
	assert.Equal(t, geom.V2(200, 100), layShape(in, nil))
	assert.Equal(t, geom.V2(80, 40), layShape(in, wide(80)))
	assert.Equal(t, geom.V2(60, 30), layShape(in, high(30)))
}

func TestAPictureWithProportionsAloneTakesTheWidthItIsGiven(t *testing.T) {
	in := box.Intrinsic{Ratio: 2}
	assert.Equal(t, geom.V2(400, 200), layShape(in, nil), "its block's width, as Chrome")
	assert.Equal(t, geom.V2(120, 60), layShape(in, wide(120)))
	assert.Equal(t, geom.V2(80, 40), layShape(in, high(40)))
	assert.Equal(t, geom.V2(50, 25), layShape(in, func(st *style.Computed) { st.MaxSize.Width = style.Px(50) }))
	assert.Equal(t, geom.V2(390, 185), layShape(in, func(st *style.Computed) {
		st.Margin.Left, st.Padding.Right = style.Px(10), style.Px(20)
	}), "less its margin, its padding around a picture of 370 by 185")
}

func TestAPictureAskingForNothingIs300By150AndItsSidesDoNotHoldEachOther(t *testing.T) {
	assert.Equal(t, geom.V2(300, 150), layShape(box.Intrinsic{}, nil))
	assert.Equal(t, geom.V2(100, 150), layShape(box.Intrinsic{}, wide(100)))
	assert.Equal(t, geom.V2(300, 30), layShape(box.Intrinsic{}, high(30)))
	narrow := func(st *style.Computed) { st.MaxSize.Width = style.Px(50) }
	assert.Equal(t, geom.V2(50, 150), layShape(box.Intrinsic{}, narrow))
}

func TestAPictureAskingForOneSideTakes150Or300ForTheOther(t *testing.T) {
	assert.Equal(t, geom.V2(80, 150), layShape(box.Intrinsic{Width: 80}, nil))
	assert.Equal(t, geom.V2(80, 40), layShape(box.Intrinsic{Width: 80}, high(40)))
	assert.Equal(t, geom.V2(300, 40), layShape(box.Intrinsic{Height: 40}, nil))
	assert.Equal(t, geom.V2(80, 40), layShape(box.Intrinsic{Width: 80, Ratio: 2}, nil), "through its proportions")
}

func TestABlockImgKeepsItsPicturesSizeAndOneThatFillsItsBlocksWidth(t *testing.T) {
	asBlock := func(st *style.Computed) { st.Display = style.Block }
	lay := func(in box.Intrinsic) geom.Vec2 {
		i := img("icon.svg", asBlock)
		tr := Layout(el(dom.Div, width(400), i), geom.V2(600, 300), ahemFonts{}, Images(shapes{"icon.svg": in}))
		return tr.Of[i].Size
	}
	assert.Equal(t, geom.V2(200, 100), lay(box.Intrinsic{Width: 200, Height: 100, Ratio: 2}), "never stretched")
	assert.Equal(t, geom.V2(300, 150), lay(box.Intrinsic{}))
	assert.Equal(t, geom.V2(400, 200), lay(box.Intrinsic{Ratio: 2}))

	png := img("tile.png", asBlock)
	tr := Layout(el(dom.Div, width(400), png), geom.V2(600, 300), ahemFonts{},
		Images(pictures{"tile.png": geom.V2(40, 20)}))
	assert.Equal(t, geom.V2(40, 20), tr.Of[png].Size, "a picture of pixels neither")
}

func TestALineIsAsHighAsThePictureThatFillsIt(t *testing.T) {
	i := img("icon.svg", nil)
	p := el(dom.P, block, i)
	tr := Layout(el(dom.Div, width(400), p), geom.V2(600, 300), ahemFonts{}, Images(shapes{"icon.svg": {Ratio: 2}}))
	assert.Equal(t, geom.V2(400, 200), tr.Of[i].Size)
	assert.GreaterOrEqual(t, tr.Of[p].Size.Y, float32(200))
}

func TestAPictureNarrowedByItsMaxWidthKeepsItsProportions(t *testing.T) {
	lay := func(edit func(*style.Computed)) geom.Vec2 {
		i := img("thumb.png", edit)
		tr := Layout(el(dom.Div, width(300), i), geom.V2(600, 300), ahemFonts{},
			Images(pictures{"thumb.png": geom.V2(1024, 576)}))
		return tr.Of[i].Size
	}
	fit := func(st *style.Computed) {
		st.Size.Width, st.MaxSize.Width = style.Px(800), style.Percent(100)
	}
	assert.Equal(t, geom.V2(300, 168.75), lay(fit), "its height from the width it ends with, as Chrome")
	assert.Equal(t, geom.V2(160, 90), lay(func(st *style.Computed) {
		st.Size.Height, st.MaxSize.Height = style.Px(400), style.Px(90)
	}))
	assert.Equal(t, geom.V2(80, 45), lay(func(st *style.Computed) {
		st.Size.Width, st.MinSize.Width = style.Px(40), style.Px(80)
	}))
}
