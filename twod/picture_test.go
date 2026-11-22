package twod //nolint:testpackage // white-box: shares twod's test helpers, reaches unexported state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/internal/draw"
	"github.com/lantervale/ember/render"
)

const tint = `fn effect(vUV: vec2f) -> vec4f { return color(vUV) * uK; }`

// pictured draws a square under a picture through one effect, or twice,
// a shadow under it, and a nested picture in it when nested.
type pictured struct {
	Item
	twice, nested bool
	fx            []*render.Effect
	pic           *Picture
}

func (p *pictured) Draw(c *Canvas) {
	c.SetTransform(c.Transform().Mul(geom.Transform2DFrom(geom.V2(100, 50), 0, 0, geom.V2(1, 1))))
	c.BeginPicture()
	c.Rect(geom.Rect{W: 20, H: 10}, color.White)
	if p.nested {
		c.BeginPicture()
		c.Rect(geom.Rect{X: 5, W: 5, H: 5}, color.White)
		c.DrawPicture(c.EndPicture(0), c.Effect(tint))
	}
	p.pic = c.EndPicture(4)
	a := c.Effect(tint)
	a.SetFloat("uK", 0.5)
	p.fx = []*render.Effect{a}
	if p.twice {
		b := c.Effect(tint)
		p.fx = append(p.fx, b)
		c.DrawPicture(p.pic, b)
	}
	c.DrawPicture(p.pic, a)
	c.Rect(geom.Rect{X: 30, W: 2, H: 2}, color.White)
}

func TestAPictureIsDrawnApartAndShownThroughItsEffects(t *testing.T) {
	w, win, front := app()
	p := &pictured{Item: Base()}
	win.Add(Build(p, "pictured"))
	w.RunFrames(1)

	passes := draw.PassesFor(w, win)
	require.Len(t, passes, 1)
	pass := passes[0]
	assert.Equal(t, p.fx, pass.Effects)
	assert.Equal(t, geom.Rect{X: 94, Y: 44, W: 32, H: 32}, p.pic.Rect(),
		"what was drawn, its margin and a cropped layer's, on whole pixels, its size rounded up")
	assert.Equal(t, geom.V2(32, 32), pass.Screen)
	lo, hi := corners(pass.List.Unindexed())
	assert.Equal(t, geom.V2(6, 6), lo, "the square, in the picture's own space")
	assert.Equal(t, geom.V2(26, 16), hi)

	vs := front.Unindexed()
	require.Len(t, vs, 12, "the picture, then the square after it")
	assert.Equal(t, p.pic.Rect(), shownAt(vs[:6]))
	assert.Equal(t, pass.Target, front.Batches[0].Image())
	assert.Equal(t, geom.Rect{X: 130, Y: 50, W: 2, H: 2}, shownAt(vs[6:]), "the transform holds after it")
}

func TestAPictureShownTwiceDrawsItsListTwice(t *testing.T) {
	w, win, front := app()
	p := &pictured{Item: Base(), twice: true}
	win.Add(Build(p, "pictured"))
	w.RunFrames(1)

	passes := draw.PassesFor(w, win)
	require.Len(t, passes, 2)
	assert.Same(t, passes[0].List, passes[1].List, "one drawing")
	assert.Equal(t, []*render.Effect{p.fx[1]}, passes[1].Effects, "the picture's own pass, shown first")
	assert.Equal(t, []*render.Effect{p.fx[0]}, passes[0].Effects)
	assert.NotSame(t, passes[0].Target, passes[1].Target)
	require.GreaterOrEqual(t, len(front.Batches), 2)
	assert.Equal(t, passes[1].Target, front.Batches[0].Image(), "under")
	assert.Equal(t, passes[0].Target, front.Batches[1].Image(), "over")
}

func TestANestedPictureIsDrawnBeforeTheOneItIsIn(t *testing.T) {
	w, win, _ := app()
	p := &pictured{Item: Base(), nested: true, twice: true}
	win.Add(Build(p, "pictured"))
	w.RunFrames(1)

	passes := draw.PassesFor(w, win)
	require.Len(t, passes, 3)
	assert.Equal(t, p.pic.pass, passes[2], "the outer picture last")
	assert.Same(t, passes[1].List, passes[2].List, "its second showing just before")
	assert.Equal(t, passes[0].Target, passes[1].List.Batches[1].Image(), "the inner one first, shown in the outer")
}

func TestAPicturesPassesAndEffectsAreKeptFromFrameToFrame(t *testing.T) {
	w, win, _ := app()
	p := &pictured{Item: Base(), twice: true}
	win.Add(Build(p, "pictured"))
	w.RunFrames(1)
	before, fx := draw.PassesFor(w, win), p.fx
	targets := []*render.Texture{before[0].Target, before[1].Target}
	w.RunFrames(1)
	after := draw.PassesFor(w, win)
	require.Len(t, after, 2)
	assert.Equal(t, targets, []*render.Texture{after[0].Target, after[1].Target}, "the GPU keeps their textures")
	assert.Equal(t, fx, p.fx, "and the effects their shaders")
}
