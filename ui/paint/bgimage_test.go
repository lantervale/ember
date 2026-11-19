package paint //nolint:testpackage // white-box: builds box trees by hand and reads the painter's own state

import (
	"image"
	"testing"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// arrow is a 10×6 picture, like the select's.
//
//nolint:gochecknoglobals // the tests' fixed values; Go has no constant colours or pictures
var arrow = twod.NewTexture(image.NewRGBA(image.Rect(0, 0, 10, 6)))

func pictures(b *box.Box) *recorder {
	r := newRecorder()
	Paint(r, tree(el(0, 0, 400, 300, nil, b)), Images(func(string) *twod.Texture { return arrow }))
	return r
}

func withImage(w, h float32, edit func(s *style.Computed)) *box.Box {
	return el(10, 10, w, h, func(s *style.Computed) {
		s.BackgroundImage = style.Image{Source: "arrow"}
		s.BackgroundNoRepeat = true
		if edit != nil {
			edit(s)
		}
	})
}

func TestABackgroundImageDrawsAtItsOwnSizeAtTheTopLeft(t *testing.T) {
	r := pictures(withImage(120, 24, nil))
	imgs := r.of("image")
	require.Len(t, imgs, 1)
	assert.Equal(t, geom.Rect{X: 10, Y: 10, W: 10, H: 6}, imgs[0].Rect, "never stretched over the box")
}

func TestABackgroundImageIsPlacedFromTheRightAndCentred(t *testing.T) {
	r := pictures(withImage(120, 24, func(s *style.Computed) {
		s.BackgroundPosition = style.BackgroundPosition{X: style.Px(6), Y: style.Percent(50), FromRight: true}
	}))
	assert.Equal(t, geom.Rect{X: 10 + 120 - 6 - 10, Y: 10 + (24-6)/2, W: 10, H: 6}, r.of("image")[0].Rect,
		"right 6px center, the select's arrow")
}

func TestABackgroundImageIsPlacedInThePaddingBox(t *testing.T) {
	b := withImage(120, 24, func(s *style.Computed) {
		s.BackgroundPosition = style.BackgroundPosition{X: style.Percent(100), Y: style.Px(0), FromBottom: true}
	})
	b.Border = box.Edges{Top: 2, Right: 2, Bottom: 2, Left: 2}
	assert.Equal(t, geom.Rect{X: 10 + 2 + 116 - 10, Y: 10 + 2 + 20 - 6, W: 10, H: 6}, pictures(b).of("image")[0].Rect,
		"100% is the right edge, 0 from the bottom is the bottom edge, both inside the border")
}

func TestARepeatedBackgroundImageTilesThePaddingBoxClipped(t *testing.T) {
	b := withImage(25, 10, func(s *style.Computed) { s.BackgroundNoRepeat = false })
	r := pictures(b)
	imgs := r.of("image")
	require.Len(t, imgs, 6, "three across, two down")
	for _, o := range imgs {
		assert.Equal(t, geom.Rect{X: 10, Y: 10, W: 25, H: 10}, o.Clip)
		assert.Equal(t, float32(10), o.Rect.W)
	}
	assert.Equal(t, geom.V2(30, 16), geom.V2(imgs[5].Rect.X, imgs[5].Rect.Y))
}

func TestARepeatedImageTilesBothWaysFromItsPosition(t *testing.T) {
	b := withImage(25, 10, func(s *style.Computed) {
		s.BackgroundNoRepeat = false
		s.BackgroundPosition = style.BackgroundPosition{X: style.Px(4), Y: style.Px(0)}
	})
	xs := map[float32]bool{}
	for _, o := range pictures(b).of("image") {
		xs[o.Rect.X] = true
	}
	assert.True(t, xs[10+4-10], "a tile before the placed one fills the start")
	assert.True(t, xs[10+4], "the placed one")
}

func TestASelectDrawsNoBuiltInArrow(t *testing.T) {
	assert.Empty(t, paintOne(ctl(dom.Select, 120, 24)).of("triangles"), "its arrow is CSS's background image")
}

func TestAPlaceholderTakesItsPlaceholderColour(t *testing.T) {
	b := ctl(dom.Input, 100, 20, "type", "text", "placeholder", "Name")
	shadow := dom.NewText("Name")
	run := text(2, 0, "Name", b.Style)
	run.Node = shadow
	b.Append(run)
	dom.Data(b.Node).Shadow, dom.Data(b.Node).Placeholder = shadow, true
	grey := color.Hex("#757575")
	b.Style.PlaceholderColor, b.Style.HasPlaceholder = grey, true
	assert.Equal(t, grey, paintOne(b).of("text")[0].Color)

	b.Style.HasPlaceholder = false
	assert.InDelta(t, 0.54, paintOne(b).of("text")[0].Color.A, 1e-6, "no ::placeholder rule: 54% of the colour")
}

func sized(w, h style.Length) style.BackgroundSize { return style.BackgroundSize{Width: w, Height: h} }

func TestABackgroundImageTakesItsBackgroundSize(t *testing.T) {
	auto := style.Length{}
	r := pictures(withImage(120, 24, func(s *style.Computed) { s.BackgroundSize = sized(style.Px(5), style.Px(3)) }))
	assert.Equal(t, geom.Rect{X: 10, Y: 10, W: 5, H: 3}, r.of("image")[0].Rect, "a 2x picture drawn at its point size")
	r = pictures(withImage(120, 24, func(s *style.Computed) { s.BackgroundSize = sized(style.Px(20), auto) }))
	assert.Equal(t, geom.Rect{X: 10, Y: 10, W: 20, H: 12}, r.of("image")[0].Rect,
		"one length keeps the picture's proportions")
	r = pictures(withImage(120, 24, func(s *style.Computed) { s.BackgroundSize = sized(auto, style.Px(3)) }))
	assert.Equal(t, geom.Rect{X: 10, Y: 10, W: 5, H: 3}, r.of("image")[0].Rect, "and so does a height alone")
}

func TestTileSize(t *testing.T) {
	auto, cover, contain := style.Length{}, style.BackgroundSize{Scale: style.ScaleCover},
		style.BackgroundSize{Scale: style.ScaleContain}
	picture, gradient, area := geom.V2(10, 6), geom.Vec2{}, geom.V2(120, 24)
	for _, c := range []struct {
		why           string
		size          style.BackgroundSize
		natural, area geom.Vec2
		want          geom.Vec2
	}{
		{"auto is the picture's own size", sized(auto, auto), picture, area, geom.V2(10, 6)},
		{"a percentage is of the padding box", sized(style.Percent(50), style.Percent(50)), picture, area, geom.V2(60, 12)},
		{"a width alone keeps the proportions", sized(style.Percent(50), auto), picture, area, geom.V2(60, 36)},
		{"so does a height alone", sized(auto, style.Percent(50)), picture, area, geom.V2(20, 12)},
		{"a percentage and a length", sized(style.Percent(25), style.Px(10)), picture, area, geom.V2(30, 10)},
		{"calc() of both", sized(style.Length{Kind: style.Fixed, Px: -20, Percent: 100}, style.Px(6)), picture, area,
			geom.V2(100, 6)},
		{"a negative calc() is nothing", sized(style.Length{Kind: style.Fixed, Px: -200, Percent: 100}, auto), picture,
			area, geom.V2(0, 0)},
		{"cover fills the wider side and overflows the other", cover, picture, area, geom.V2(120, 72)},
		{"contain fits the tighter side", contain, picture, area, geom.V2(40, 24)},
		{"cover in a tall box", cover, picture, geom.V2(24, 120), geom.V2(200, 120)},
		{"contain in a tall box rounds the side it scales, 14.4", contain, picture, geom.V2(24, 120), geom.V2(24, 14)},
		{"and up from a half, 7.5", contain, geom.V2(10, 3), geom.V2(25, 120), geom.V2(25, 8)},
		{"never under a point", contain, geom.V2(100, 1), geom.V2(20, 120), geom.V2(20, 1)},
		{"cover does not round, 16.5", cover, geom.V2(4, 6), geom.V2(11, 12), geom.V2(11, 16.5)},
		{"a gradient has no size: auto is the box", sized(auto, auto), gradient, area, area},
		{"cover too", cover, gradient, area, area},
		{"and contain", contain, gradient, area, area},
		{"its auto beside a width is the box's height", sized(style.Percent(50), auto), gradient, area, geom.V2(60, 24)},
		{"and beside a height the box's width", sized(auto, style.Px(10)), gradient, area, geom.V2(120, 10)},
		{"both given", sized(style.Px(8), style.Percent(25)), gradient, area, geom.V2(8, 6)},
	} {
		assert.Equal(t, c.want, tileSize(c.size, box.Sized(c.natural), c.area, 1), c.why)
	}
	assert.Equal(t, geom.V2(24, 14.5), tileSize(contain, box.Sized(picture), geom.V2(24, 120), 2),
		"on a 2x screen contain rounds to half a point, 14.4 to 29 pixels")
	assert.Equal(t, geom.V2(25, 7.5), tileSize(contain, box.Sized(geom.V2(10, 3)), geom.V2(25, 120), 0),
		"and not at all off the pixel grid")
}

func TestAPicturePlacedInItsBoxStartsOnADevicePixel(t *testing.T) {
	paint := func(w, h float32, edit func(s *style.Computed)) []op {
		r := newRecorder()
		Paint(r, tree(el(0, 0, 400, 300, nil, withImage(w, h, edit))), Scale(2),
			Images(func(string) *twod.Texture { return arrow }))
		return r.of("image")
	}
	centred := paint(120, 70, func(s *style.Computed) {
		s.BackgroundSize = sized(style.Px(8.25), style.Px(41.25))
		s.BackgroundPosition = style.BackgroundPosition{X: style.Percent(50), Y: style.Percent(50)}
	})
	assert.Equal(t, geom.Rect{X: 66, Y: 24.5, W: 8.5, H: 41.5}, centred[0].Rect,
		"from 65.875 and 24.375 to the pixels at 66 and 24.5, then 8.25 and 41.25 on, to those at 74.5 and 66")

	cover := paint(120, 24, func(s *style.Computed) {
		s.BackgroundSize = sized(style.Px(120), style.Px(24.5))
		s.BackgroundPosition = style.BackgroundPosition{X: style.Px(0), Y: style.Percent(50)}
	})
	assert.Equal(t, geom.Rect{X: 10, Y: 9.75, W: 120, H: 24.5}, cover[0].Rect,
		"one starting before the box keeps its place")

	across := paint(120, 70, func(s *style.Computed) {
		s.BackgroundSize = sized(style.Px(36.25), style.Px(24.75))
		s.BackgroundPosition = style.BackgroundPosition{X: style.Px(0.25), Y: style.Percent(100)}
		s.BackgroundNoRepeat, s.BackgroundRepeat = false, style.RepeatX
	})
	assert.Equal(t, geom.Rect{X: 10.25 - 36.25, Y: 55.5, W: 36.25, H: 24.75}, across[0].Rect,
		"repeat-x: down it starts on a pixel, its size kept; across its tiles keep their place")
}

func TestACoverPictureIsPlacedByItsPositionAndCutToThePaddingBox(t *testing.T) {
	b := withImage(120, 24, func(s *style.Computed) {
		s.BackgroundSize = style.BackgroundSize{Scale: style.ScaleCover}
		s.BackgroundPosition = style.BackgroundPosition{X: style.Percent(50), Y: style.Percent(50)}
	})
	imgs := pictures(b).of("image")
	require.Len(t, imgs, 1)
	assert.Equal(t, geom.Rect{X: 10, Y: 10 + (24-72)/2, W: 120, H: 72}, imgs[0].Rect, "centred: cut above and below")
	assert.Equal(t, geom.Rect{X: 10, Y: 10, W: 120, H: 24}, imgs[0].Clip)
}

func TestAContainPictureIsPlacedFromTheRightAndTiles(t *testing.T) {
	contain := func(s *style.Computed) {
		s.BackgroundSize = style.BackgroundSize{Scale: style.ScaleContain}
		s.BackgroundPosition = style.BackgroundPosition{X: style.Percent(100), Y: style.Percent(0)}
	}
	imgs := pictures(withImage(120, 24, contain)).of("image")
	require.Len(t, imgs, 1)
	assert.Equal(t, geom.Rect{X: 10 + 120 - 40, Y: 10, W: 40, H: 24}, imgs[0].Rect,
		"100% puts its right edge on the box's")

	imgs = pictures(withImage(120, 24, func(s *style.Computed) {
		contain(s)
		s.BackgroundNoRepeat = false
	})).of("image")
	require.Len(t, imgs, 3, "three of the scaled picture fill the box")
	for i, o := range imgs {
		assert.Equal(t, geom.Rect{X: 10 + 40*float32(i), Y: 10, W: 40, H: 24}, o.Rect)
	}
}

func TestTilesAreTheSizeBackgroundSizeGives(t *testing.T) {
	half := func(repeat style.RepeatAxes) []op {
		return pictures(withImage(120, 24, func(s *style.Computed) {
			s.BackgroundSize = sized(style.Percent(50), style.Percent(50))
			s.BackgroundNoRepeat, s.BackgroundRepeat = false, repeat
		})).of("image")
	}
	both := half(style.RepeatBoth)
	require.Len(t, both, 4, "two across, two down")
	assert.Equal(t, geom.Rect{X: 70, Y: 22, W: 60, H: 12}, both[3].Rect)
	across := half(style.RepeatX)
	require.Len(t, across, 2, "repeat-x: one row")
	assert.Equal(t, geom.Rect{X: 70, Y: 10, W: 60, H: 12}, across[1].Rect)
	down := half(style.RepeatY)
	require.Len(t, down, 2, "repeat-y: one column")
	assert.Equal(t, geom.Rect{X: 10, Y: 22, W: 60, H: 12}, down[1].Rect)

	placed := pictures(withImage(120, 24, func(s *style.Computed) {
		s.BackgroundSize = sized(style.Percent(50), style.Length{})
		s.BackgroundNoRepeat, s.BackgroundRepeat = false, style.RepeatX
		s.BackgroundPosition = style.BackgroundPosition{X: style.Px(30), Y: style.Percent(100)}
	})).of("image")
	require.Len(t, placed, 3, "60 by 36 tiles from 30 in: one before, the placed one, one after")
	assert.Equal(t, geom.Rect{X: 10 + 30 - 60, Y: 10 + 24 - 36, W: 60, H: 36}, placed[0].Rect,
		"100% down puts the scaled picture's bottom on the box's")
}

func TestABackgroundOfNoSizeDrawsNothing(t *testing.T) {
	for _, size := range []style.BackgroundSize{
		sized(style.Px(0), style.Length{}), sized(style.Length{}, style.Percent(0)), sized(style.Px(0), style.Px(4)),
	} {
		r := pictures(withImage(120, 24, func(s *style.Computed) { s.BackgroundSize, s.BackgroundNoRepeat = size, false }))
		assert.Empty(t, r.of("image", "push"), "%+v", size)
	}
}

func TestAPictureFetchedLateIsSizedOnceItLands(t *testing.T) {
	b := withImage(120, 24, func(s *style.Computed) { s.BackgroundSize = style.BackgroundSize{Scale: style.ScaleContain} })
	var landed *twod.Texture
	paint := func() []op {
		r := newRecorder()
		Paint(r, tree(el(0, 0, 400, 300, nil, b)), Images(func(string) *twod.Texture { return landed }))
		return r.of("image")
	}
	assert.Empty(t, paint(), "nothing while it is on its way")
	landed = arrow
	imgs := paint()
	require.Len(t, imgs, 1)
	assert.Equal(t, geom.Rect{X: 10, Y: 10, W: 40, H: 24}, imgs[0].Rect)
}

func TestAPictureReachingARoundedCornerIsCutByIt(t *testing.T) {
	cover := pictures(withImage(120, 24, func(s *style.Computed) {
		rounded(8)(s)
		s.BackgroundSize = style.BackgroundSize{Scale: style.ScaleCover}
	}))
	clip := cover.of("push-round")
	require.Len(t, clip, 1)
	assert.Equal(t, geom.Rect{X: 10, Y: 10, W: 120, H: 24}, clip[0].Rect)
	assert.Equal(t, [4]float32{8, 8, 8, 8}, clip[0].Radii)

	tiled := pictures(withImage(120, 24, func(s *style.Computed) {
		rounded(8)(s)
		s.BackgroundNoRepeat = false
	}))
	assert.Len(t, tiled.of("push-round"), 1, "tiles reach the corners")

	arrowed := pictures(withImage(120, 24, func(s *style.Computed) {
		rounded(8)(s)
		s.BackgroundPosition = style.BackgroundPosition{X: style.Px(6), Y: style.Percent(50), FromRight: true}
	}))
	assert.Empty(t, arrowed.of("push-round"), "a picture clear of the corners needs no curve")
	assert.Len(t, arrowed.of("image"), 1)
}

// withGradient is a box of 120×24 with a gradient behind it.
func withGradient(edit func(s *style.Computed)) *box.Box {
	return el(10, 10, 120, 24, func(s *style.Computed) {
		s.BackgroundImage = style.Image{Gradient: &style.Gradient{Angle: 90, Stops: stops(red, blue)}}
		if edit != nil {
			edit(s)
		}
	})
}

func TestAGradientTakesItsBackgroundSizePositionAndRepeat(t *testing.T) {
	whole := pictures(withGradient(nil)).of("gradient")
	require.Len(t, whole, 1)
	assert.Equal(t, geom.Rect{X: 10, Y: 10, W: 120, H: 24}, whole[0].Rect, "auto is the padding box")

	for _, scale := range []style.BackgroundScale{style.ScaleCover, style.ScaleContain} {
		m := pictures(withGradient(func(s *style.Computed) { s.BackgroundSize.Scale = scale })).of("gradient")
		require.Len(t, m, 1)
		assert.Equal(t, whole[0].Rect, m[0].Rect, "cover and contain too")
	}

	one := pictures(withGradient(func(s *style.Computed) {
		s.BackgroundSize = sized(style.Percent(25), style.Percent(50))
		s.BackgroundNoRepeat = true
		s.BackgroundPosition = style.BackgroundPosition{X: style.Percent(100), Y: style.Percent(100)}
	})).of("gradient")
	require.Len(t, one, 1)
	assert.Equal(t, geom.Rect{X: 10 + 90, Y: 10 + 12, W: 30, H: 12}, one[0].Rect, "its box is sized, then placed")
	assert.Equal(t, geom.Rect{X: 10, Y: 10, W: 120, H: 24}, one[0].Clip)

	tiled := pictures(withGradient(func(s *style.Computed) {
		s.BackgroundSize = sized(style.Percent(25), style.Length{})
	})).of("gradient")
	require.Len(t, tiled, 4, "a quarter wide and the box high: four across")
	assert.Equal(t, geom.Rect{X: 100, Y: 10, W: 30, H: 24}, tiled[3].Rect)

	round := pictures(withGradient(func(s *style.Computed) {
		rounded(8)(s)
		s.BackgroundSize = sized(style.Percent(25), style.Length{})
	}))
	require.Len(t, round.of("push-round"), 1, "one curve round all the tiles")
	assert.Equal(t, geom.Rect{X: 10, Y: 10, W: 120, H: 24}, round.of("push-round")[0].Rect)
}

func TestRepeatXAndRepeatYTileAlongOneAxisOnly(t *testing.T) {
	across := pictures(withImage(25, 10, func(s *style.Computed) {
		s.BackgroundNoRepeat, s.BackgroundRepeat = false, style.RepeatX
	})).of("image")
	require.Len(t, across, 3, "repeat-x: one row")
	down := pictures(withImage(25, 10, func(s *style.Computed) {
		s.BackgroundNoRepeat, s.BackgroundRepeat = false, style.RepeatY
	})).of("image")
	require.Len(t, down, 2, "repeat-y: one column")
	assert.Equal(t, down[0].Rect.X, down[1].Rect.X)
}

func TestABackgroundFillsItsClipBoxAndStartsAtItsOrigin(t *testing.T) {
	framed := func(edit func(s *style.Computed)) *box.Box {
		b := withImage(120, 40, func(s *style.Computed) {
			s.Background = red
			edit(s)
		})
		b.Border = box.Edges{Top: 2, Right: 2, Bottom: 2, Left: 2}
		b.Padding = box.Edges{Top: 3, Right: 3, Bottom: 3, Left: 3}
		return b
	}
	r := pictures(framed(func(*style.Computed) {}))
	assert.Equal(t, geom.Rect{X: 10, Y: 10, W: 120, H: 40}, r.of("rect")[0].Rect, "the colour under the border")
	assert.Equal(t, geom.Rect{X: 12, Y: 12, W: 10, H: 6}, r.of("image")[0].Rect, "the picture from the padding box")
	assert.Equal(t, geom.Rect{X: 10, Y: 10, W: 120, H: 40}, r.of("image")[0].Clip, "and under the border where it tiles")

	r = pictures(framed(func(s *style.Computed) {
		s.BackgroundClip, s.BackgroundOrigin = style.UnderPadding, style.UnderBorder
	}))
	assert.Equal(t, geom.Rect{X: 12, Y: 12, W: 116, H: 36}, r.of("rect")[0].Rect)
	assert.Equal(t, geom.Rect{X: 10, Y: 10, W: 10, H: 6}, r.of("image")[0].Rect)
	assert.Equal(t, geom.Rect{X: 12, Y: 12, W: 116, H: 36}, r.of("image")[0].Clip)

	r = pictures(framed(func(s *style.Computed) {
		s.BackgroundClip, s.BackgroundOrigin = style.UnderContent, style.UnderContent
	}))
	assert.Equal(t, geom.Rect{X: 15, Y: 15, W: 110, H: 30}, r.of("rect")[0].Rect)
	assert.Equal(t, geom.Rect{X: 15, Y: 15, W: 10, H: 6}, r.of("image")[0].Rect)
}

func TestAFixedBackgroundIsPlacedInTheViewportAndALocalOneScrolls(t *testing.T) {
	fixed := withImage(120, 40, func(s *style.Computed) {
		s.BackgroundAttachment = style.AttachFixed
		s.BackgroundPosition = style.BackgroundPosition{X: style.Px(20), Y: style.Px(15)}
	})
	img := pictures(fixed).of("image")[0]
	assert.Equal(t, geom.Rect{X: 20, Y: 15, W: 10, H: 6}, img.Rect, "from the viewport's corner, wherever the box is")
	assert.Equal(t, geom.Rect{X: 10, Y: 10, W: 120, H: 40}, img.Clip, "shown where the box is")

	scroller := func(a style.BackgroundAttachment) *box.Box {
		b := withImage(120, 40, func(s *style.Computed) {
			s.BackgroundAttachment = a
			s.Overflow = style.Size[style.Overflow]{Width: style.Scroll, Height: style.Scroll}
		})
		b.Content, b.Scroll = geom.V2(120, 400), geom.V2(0, 4)
		return b
	}
	assert.Equal(t, geom.V2(10, 10), placeOfImage(scroller(style.AttachScroll)),
		"a background stays in its box as the content scrolls")
	assert.Equal(t, geom.V2(10, 6), placeOfImage(scroller(style.AttachLocal)), "a local one goes with the content")
}

func placeOfImage(b *box.Box) geom.Vec2 {
	r := pictures(b).of("image")[0].Rect
	return geom.V2(r.X, r.Y)
}

func TestAnInlineOverSeveralLinesShowsItsBackgroundAsOneBoxSliced(t *testing.T) {
	seen := func(b *box.Box) []geom.Rect {
		var out []geom.Rect
		for _, o := range pictures(b).of("image") {
			if in := intersect(o.Rect, o.Clip); in.W > 0 && in.H > 0 {
				out = append(out, o.Rect)
			}
		}
		return out
	}
	b := withImage(100, 20, func(s *style.Computed) {
		s.Display = style.Inline
		s.BackgroundPosition = style.BackgroundPosition{X: style.Percent(100), Y: style.Percent(50)}
	})
	b.Padding = box.Edges{Right: 12}
	b.Fragments = []geom.Rect{{X: 30, Y: 0, W: 70, H: 10}, {X: 0, Y: 10, W: 52, H: 10}}
	assert.Equal(t, []geom.Rect{{X: 10 + 52 - 10, Y: 10 + 10 + 2, W: 10, H: 6}}, seen(b),
		"a picture at the right shows at the end of the last line, not of each")

	b = withImage(100, 20, func(s *style.Computed) { s.Display = style.Inline })
	b.Fragments = []geom.Rect{{X: 30, Y: 0, W: 70, H: 10}, {X: 0, Y: 10, W: 52, H: 10}}
	assert.Equal(t, []geom.Rect{{X: 10 + 30, Y: 10, W: 10, H: 6}}, seen(b), "one at the left at the start of the first")
}
