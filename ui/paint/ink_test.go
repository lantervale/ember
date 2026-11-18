package paint //nolint:testpackage // white-box: builds box trees by hand and reads the painter's own state

import (
	"fmt"
	"testing"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func cutsBoth(s *style.Computed) {
	s.Overflow = style.Size[style.Overflow]{Width: style.Hidden, Height: style.Hidden}
}

func TestABoxsInkHoldsItsShadowsItsOutlineItsTextAndItsChildren(t *testing.T) {
	plain := el(10, 20, 100, 50, nil)
	bound(plain)
	assert.Equal(t, geom.Rect{W: 100, H: 50}, plain.Ink, "its border box, in its own space")

	shadowed := el(10, 20, 100, 50, func(s *style.Computed) {
		s.Shadows = []style.Shadow{{Y: 4, Spread: 2, Color: black}}
	})
	bound(shadowed)
	assert.Equal(t, geom.Rect{X: -6, Y: -6, W: 112, H: 62}, shadowed.Ink)

	ringed := el(10, 20, 100, 50, func(s *style.Computed) {
		s.OutlineStyle, s.OutlineWidth, s.OutlineOffset = style.BorderSolid, 2, 3
	})
	bound(ringed)
	assert.Equal(t, geom.Rect{X: -5, Y: -5, W: 110, H: 60}, ringed.Ink)

	holder := el(0, 0, 100, 20, nil)
	run := text(0, 0, "ab", holder.Style)
	holder.Append(run)
	bound(holder)
	em := holder.Style.Font.Size
	assert.Equal(t, geom.Rect{X: -em, Y: -em, W: 16 + 2*em, H: 20 + 2*em}, run.Ink, "glyphs may overhang by an em")
	assert.Equal(t, union(geom.Rect{W: 100, H: 20}, run.Ink), holder.Ink)

	parent := el(0, 0, 100, 50, nil, el(80, 40, 60, 30, nil, el(-200, 0, 10, 10, nil)))
	bound(parent)
	assert.Equal(t, geom.Rect{X: -120, W: 260, H: 70}, parent.Ink,
		"children overflowing it, and theirs, where they lie in it")
}

func TestABoxCuttingItsOverflowBoundsWhatItHolds(t *testing.T) {
	far := el(0, 5000, 100, 20, bg(red))
	clip := el(0, 0, 100, 50, cutsBoth, far)
	bound(clip)
	assert.Equal(t, geom.Rect{W: 100, H: 50}, clip.Ink, "nothing under it shows outside it")
	assert.Equal(t, geom.Rect{W: 100, H: 20}, far.Ink)
}

func TestBoxesNoInkCanHoldAreLooseAndSoIsWhatHoldsThem(t *testing.T) {
	tagged := func(tag dom.Tag) *box.Box {
		b := el(0, 0, 10, 10, nil)
		b.Node = dom.NewElement(tag)
		return b
	}
	escaping := el(0, 0, 10, 10, nil)
	escaping.Escapes = true
	for name, b := range map[string]*box.Box{
		"absolute": el(0, 0, 10, 10, func(s *style.Computed) { s.Position = style.Absolute }),
		"fixed":    el(0, 0, 10, 10, func(s *style.Computed) { s.Position = style.FixedPosition }),
		"sticky":   el(0, 0, 10, 10, func(s *style.Computed) { s.Position = style.Sticky }),
		"transformed": el(0, 0, 10, 10, func(s *style.Computed) {
			s.Transform = []style.TransformOp{{Kind: style.Translate, X: style.Px(5), Y: style.Px(5)}}
		}),
		"cut on one axis": el(0, 0, 10, 10, func(s *style.Computed) { s.Overflow.Width = style.Hidden }),
		"a select":        tagged(dom.Select),
		"a dialog":        tagged(dom.Dialog),
		"on the way to the block that placed a box": escaping,
	} {
		row := el(0, 0, 100, 20, nil, b)
		beside := el(0, 20, 100, 20, nil)
		root := el(0, 0, 100, 40, nil, row, beside)
		bound(root)
		assert.True(t, b.Loose, name)
		assert.True(t, row.Loose, "%s: what holds it", name)
		assert.True(t, root.Loose, "%s: up to the root", name)
		assert.False(t, beside.Loose, "%s: not what lies beside it", name)
	}
	for name, b := range map[string]*box.Box{
		"static":   el(0, 0, 10, 10, nil),
		"relative": el(0, 0, 10, 10, func(s *style.Computed) { s.Position = style.Relative }),
		"floated":  el(0, 0, 10, 10, func(s *style.Computed) { s.Float = style.FloatLeft }),
		"cut on both axes": el(0, 0, 10, 10, func(s *style.Computed) {
			s.Overflow = style.Size[style.Overflow]{Width: style.OverflowAuto, Height: style.Scroll}
		}),
	} {
		bound(b)
		assert.False(t, b.Loose, name)
	}
}

// longPage is a 400×300 viewport scrolled to y over n sections 100 points
// high, as a long article's: each a heading and three paragraphs of text,
// every fifth one positioned and every seventh holding a float, so no
// run of rows is found by search; and how many boxes the viewport holds.
func longPage(n int, y float32) (*box.Tree, int) {
	body := el(0, 0, 400, float32(n*100), nil)
	boxes := 2
	for i := range n {
		section := el(0, float32(i*100), 400, 100, func(s *style.Computed) {
			if i%5 == 0 {
				s.Position = style.Relative
			}
		})
		head := el(0, 0, 400, 25, bg(blue))
		head.Append(text(0, 0, fmt.Sprintf("section %d", i), head.Style))
		section.Append(head)
		for k := range 3 {
			p := el(0, float32(25+k*25), 400, 25, nil)
			p.Append(text(0, 0, "a paragraph of the section", p.Style))
			section.Append(p)
		}
		boxes += 9
		if i%7 == 0 {
			section.Append(el(300, 30, 80, 60, func(s *style.Computed) { s.Float, s.Background = style.FloatRight, red }))
			boxes++
		}
		body.Append(section)
	}
	port := el(0, 0, 400, 300, func(s *style.Computed) {
		s.Overflow = style.Size[style.Overflow]{Width: style.OverflowAuto, Height: style.OverflowAuto}
	}, body)
	port.Kind = box.Viewport
	port.Content, port.Scroll = geom.V2(400, float32(n*100)), geom.V2(0, y)
	return tree(port), boxes - 1
}

// draws are what r drew, each with the clip and transform it was drawn
// under, without the clips pushed and popped around nothing.
func draws(r *recorder) []op {
	var out []op
	for _, o := range r.ops {
		if o.Kind != "push" && o.Kind != "push-round" && o.Kind != "pop" {
			out = append(out, o)
		}
	}
	return out
}

func TestAPaintWalksTheBoxesInViewNotThePage(t *testing.T) {
	const sections = 3000
	for _, y := range []float32{0, 1234, 150000, 299700} {
		tr, boxes := longPage(sections, y)
		whole, culled := newRecorder(), newRecorder()
		assert.GreaterOrEqual(t, paintOn(whole, tr, nil, true), boxes, "walking every box reaches the page's")
		reached := paintOn(culled, tr, nil, false)
		assert.Less(t, reached, 80, "scrolled to %v: the four sections in view, their boxes, and the body", y)
		require.NotEmpty(t, culled.of("text"))
		assert.Equal(t, draws(whole), draws(culled), "scrolled to %v: and draws the same", y)
	}
	short, _ := longPage(30, 0)
	long, _ := longPage(sections, 0)
	assert.Equal(t, paintOn(newRecorder(), short, nil, false), paintOn(newRecorder(), long, nil, false),
		"a page a hundred times longer costs the same")
}

func TestAHitTestWalksTheBoxesUnderThePointerNotThePage(t *testing.T) {
	tr, boxes := longPage(3000, 150000)
	for _, at := range []geom.Vec2{{X: 20, Y: 10}, {X: 350, Y: 150}, {X: 399, Y: 299}, {X: 500, Y: 100}} {
		want, all := hitTest(tr, at, nil, true)
		got, reached := hitTest(tr, at, nil, false)
		assert.GreaterOrEqual(t, all, boxes)
		assert.Less(t, reached, 20, "at %v: the body, the sections and the boxes under the point", at)
		assert.Same(t, want, got, "at %v", at)
	}
	hit, _ := hitTest(tr, geom.V2(20, 10), nil, false)
	require.NotNil(t, hit)
	assert.Equal(t, geom.Rect{Y: 0, W: 400, H: 25}, hit.Rect(), "the heading of section 1500")
}

func TestOnlyLooseBoxesAreWalkedLookingForAnOpenSelect(t *testing.T) {
	tr, _ := longPage(3000, 0)
	pt := scan(tr, nil, geom.Vec2{}, false, false)
	defer scanners.Put(pt)
	assert.Zero(t, pt.reached, "the page holds nothing loose")
}

func TestInkIsWorkedOutOnceForATreeAndAgainWhenItSaysSo(t *testing.T) {
	late := el(0, 1000, 100, 20, bg(red))
	root := el(0, 0, 400, 300, nil, el(0, 0, 100, 20, nil), late)
	tr := tree(root)
	r := newRecorder()
	Paint(r, tr)
	assert.True(t, tr.Inked)
	assert.Empty(t, r.of("rect"), "below the viewport")

	late.Style.Shadows = []style.Shadow{{Y: -900, Color: blue}}
	tr.Inked = false
	r = newRecorder()
	Paint(r, tr)
	assert.Equal(t, 1, drawn(r, "rect", blue), "its shadow now reaches into view")
}

func TestAScrollersBarsAreNotDrawnOutOfView(t *testing.T) {
	sc := list(0)
	sc.Pos = geom.V2(0, 1000)
	r := newRecorder()
	Paint(r, tree(el(0, 0, 400, 300, nil, sc)))
	assert.Empty(t, r.of("round"))

	sc.Pos = geom.V2(0, 250)
	r = newRecorder()
	Paint(r, tree(el(0, 0, 400, 300, nil, sc)))
	assert.Len(t, r.of("round"), 1, "its thumb once it shows")
}

func TestNothingShowsThroughClipsThatDoNotMeet(t *testing.T) {
	shadowed := el(0, 0, 80, 10, func(s *style.Computed) {
		s.Background = red
		s.Shadows = []style.Shadow{{Blur: 4, Color: color.Hex("#000000")}}
	})
	inner := el(0, 60, 100, 20, cutsBoth, shadowed)
	frame := el(0, 0, 100, 50, cutsBoth, inner)
	r := newRecorder()
	Paint(r, tree(el(0, 0, 400, 300, nil, frame)))
	assert.Empty(t, r.ops, "the inner clip lies below the frame's: what is in it is cut whole")
}

// BenchmarkInk is the pass that works the ink of a page of 27,000 boxes
// out, as a frame that restyled or laid out pays once.
func BenchmarkInk(b *testing.B) {
	tr, _ := longPage(3000, 0)
	b.ReportAllocs()
	for b.Loop() {
		bound(tr.Root)
	}
}

// BenchmarkLongPagePaint paints a page of 27,000 boxes scrolled to its
// middle, skipping the boxes out of view and walking every one.
func BenchmarkLongPagePaint(b *testing.B) {
	for name, whole := range map[string]bool{"culled": false, "whole": true} {
		b.Run(name, func(b *testing.B) {
			tr, _ := longPage(3000, 150000)
			r := &tally{xf: geom.Identity2D()}
			b.ReportAllocs()
			for b.Loop() {
				r.drawn = r.drawn[:0]
				paintOn(r, tr, nil, whole)
			}
		})
	}
}

// BenchmarkLongPageHitTest hit tests the same page.
func BenchmarkLongPageHitTest(b *testing.B) {
	for name, whole := range map[string]bool{"culled": false, "whole": true} {
		b.Run(name, func(b *testing.B) {
			tr, _ := longPage(3000, 150000)
			b.ReportAllocs()
			for b.Loop() {
				hitTest(tr, geom.V2(200, 150), nil, whole)
			}
		})
	}
}
