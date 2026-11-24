package layout //nolint:testpackage // tests the layout tree and its unexported helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

func valign(v style.VerticalAlign, edits ...func(*style.Computed)) func(*style.Computed) {
	return func(st *style.Computed) {
		st.VerticalAlign = v
		for _, e := range edits {
			e(st)
		}
	}
}

func raised(px, percent float32) func(*style.Computed) {
	return func(st *style.Computed) {
		st.VerticalAlign = style.AlignLength
		st.VerticalShift = style.Length{Px: px, Percent: percent}
	}
}

func slab(h float32, edits ...func(*style.Computed)) *dom.Node {
	return el(dom.Span, func(st *style.Computed) {
		st.Display = style.InlineBlock
		st.Size.Width, st.Size.Height = style.Px(10), style.Px(h)
		for _, e := range edits {
			e(st)
		}
	})
}

func lineHeight(px float32) func(*style.Computed) {
	return func(st *style.Computed) { st.LineHeight = style.LineHeight{Px: px} }
}

func fontSize(px float32) func(*style.Computed) {
	return func(st *style.Computed) { st.Font.Size = px }
}

// baselineOf is where the text s stands in b, from b's top.
func baselineOf(t *testing.T, b *box.Box, s string) float32 {
	t.Helper()
	found, at := false, float32(0)
	var walk func(b *box.Box, y float32)
	walk = func(b *box.Box, y float32) {
		for _, f := range b.Lines {
			if f.Text == s {
				found, at = true, y+f.Baseline
			}
		}
		for _, c := range b.Children {
			walk(c, y+c.Pos.Y)
		}
	}
	walk(b, 0)
	assert.True(t, found, "no text %q", s)
	return at
}

// topIn is where the box of n starts in b, from b's top.
func topIn(t *testing.T, tr *box.Tree, b *box.Box, n *dom.Node) float32 {
	t.Helper()
	found, at := false, float32(0)
	var walk func(b *box.Box, y float32)
	walk = func(b *box.Box, y float32) {
		if b == tr.Of[n] {
			found, at = true, y
		}
		for _, c := range b.Children {
			walk(c, y+c.Pos.Y)
		}
	}
	walk(b, 0)
	assert.True(t, found, "no box")
	return at
}

// aligned lays a 10px paragraph of kids out, 200 wide: a strut 8 above its baseline, 2 below.
func aligned(edit func(*style.Computed), kids ...*dom.Node) (*box.Tree, *dom.Node) {
	p := el(dom.P, func(st *style.Computed) {
		block(st)
		if edit != nil {
			edit(st)
		}
	}, kids...)
	return lay(el(dom.Div, width(200), p), 200), p
}

func TestTopAndBottomAlignToTheLineOnceTheRestIsSized(t *testing.T) {
	top, bottom := slab(30, valign(style.AlignTop)), slab(20, valign(style.AlignBottom))
	tr, p := aligned(nil, text("HH"), top, bottom)
	assert.InDelta(t, 30, tr.Of[p].Size.Y, 0.001, "the line grows below for the top box")
	assert.InDelta(t, 8, baselineOf(t, tr.Of[p], "HH"), 0.001)
	assert.InDelta(t, 0, topIn(t, tr, tr.Of[p], top), 0.001)
	assert.InDelta(t, 10, topIn(t, tr, tr.Of[p], bottom), 0.001)

	top, bottom = slab(20, valign(style.AlignTop)), slab(30, valign(style.AlignBottom))
	tr, p = aligned(nil, text("HH"), top, bottom)
	assert.InDelta(t, 30, tr.Of[p].Size.Y, 0.001)
	assert.InDelta(t, 28, baselineOf(t, tr.Of[p], "HH"), 0.001, "the taller bottom box grows the line above")
	assert.InDelta(t, 0, topIn(t, tr, tr.Of[p], top), 0.001)
	assert.InDelta(t, 0, topIn(t, tr, tr.Of[p], bottom), 0.001)

	top, bottom = slab(30, valign(style.AlignTop)), slab(30, valign(style.AlignBottom))
	tr, p = aligned(nil, text("HH"), top, bottom)
	assert.InDelta(t, 8, baselineOf(t, tr.Of[p], "HH"), 0.001, "as tall as each other, the top one wins")

	top, bottom = slab(4, valign(style.AlignTop)), slab(4, valign(style.AlignBottom))
	tr, p = aligned(nil, text("HH"), top, bottom)
	assert.InDelta(t, 10, tr.Of[p].Size.Y, 0.001)
	assert.InDelta(t, 0, topIn(t, tr, tr.Of[p], top), 0.001)
	assert.InDelta(t, 6, topIn(t, tr, tr.Of[p], bottom), 0.001)
}

func TestMiddlePutsABoxsMiddleHalfAnXHeightAboveTheBaseline(t *testing.T) {
	ib := slab(30, valign(style.AlignMiddle))
	tr, p := aligned(nil, text("HH"), ib)
	assert.InDelta(t, 30, tr.Of[p].Size.Y, 0.001)
	assert.InDelta(t, 19, baselineOf(t, tr.Of[p], "HH"), 0.001)
	assert.InDelta(t, 0, topIn(t, tr, tr.Of[p], ib), 0.001)

	big := el(dom.Span, valign(style.AlignMiddle, fontSize(20)), text("XX"))
	tr, p = aligned(nil, text("HH"), big)
	// the span's line-height box is 16 above its baseline and 4 below: 6 down, 4 up
	assert.InDelta(t, 2, baselineOf(t, tr.Of[p], "XX")-baselineOf(t, tr.Of[p], "HH"), 0.001)
	assert.InDelta(t, 14+6, tr.Of[p].Size.Y, 0.001)
}

func TestTextTopAndTextBottomAlignToTheParentsContentArea(t *testing.T) {
	ib := slab(30, valign(style.AlignTextTop))
	tr, p := aligned(lineHeight(20), text("HH"), ib)
	assert.InDelta(t, 13, baselineOf(t, tr.Of[p], "HH"), 0.001)
	assert.InDelta(t, 5, topIn(t, tr, tr.Of[p], ib), 0.001)
	assert.InDelta(t, 35, tr.Of[p].Size.Y, 0.001)

	ib = slab(30, valign(style.AlignTextBottom))
	tr, p = aligned(lineHeight(20), text("HH"), ib)
	assert.InDelta(t, 28, baselineOf(t, tr.Of[p], "HH"), 0.001)
	assert.InDelta(t, 0, topIn(t, tr, tr.Of[p], ib), 0.001)
	assert.InDelta(t, 35, tr.Of[p].Size.Y, 0.001)
}

func TestALengthRaisesABoxAndAPercentageIsOfItsLineHeight(t *testing.T) {
	ib := slab(10, raised(5, 0))
	tr, p := aligned(nil, text("HH"), ib)
	assert.InDelta(t, 15, baselineOf(t, tr.Of[p], "HH"), 0.001)
	assert.InDelta(t, 0, topIn(t, tr, tr.Of[p], ib), 0.001)

	ib = slab(10, raised(0, 50), lineHeight(20))
	tr, p = aligned(nil, text("HH"), ib)
	assert.InDelta(t, 20, baselineOf(t, tr.Of[p], "HH"), 0.001)

	ib = slab(10, raised(-5, 0))
	tr, p = aligned(nil, text("HH"), ib)
	assert.InDelta(t, 8, baselineOf(t, tr.Of[p], "HH"), 0.001)
	assert.InDelta(t, 3, topIn(t, tr, tr.Of[p], ib), 0.001)
	assert.InDelta(t, 13, tr.Of[p].Size.Y, 0.001)

	span := el(dom.Span, raised(4, 0), text("XX"))
	tr, p = aligned(nil, text("HH"), span)
	assert.InDelta(t, -4, baselineOf(t, tr.Of[p], "XX")-baselineOf(t, tr.Of[p], "HH"), 0.001)
	assert.InDelta(t, 0, topIn(t, tr, tr.Of[p], span), 0.001, "its box moves with its text")
}

func TestAlignmentNestsFromEachBoxsParent(t *testing.T) {
	ib := slab(20, valign(style.AlignMiddle))
	sup := el(dom.Span, valign(style.AlignSuper), text("XX"), ib)
	tr, p := aligned(nil, text("HH"), sup)
	up := float32(277.0 / 64)
	// the box's middle 4 above the sup's baseline: 14 above it, 6 below
	assert.InDelta(t, 14+up, baselineOf(t, tr.Of[p], "HH"), 0.001)
	assert.InDelta(t, -up, baselineOf(t, tr.Of[p], "XX")-baselineOf(t, tr.Of[p], "HH"), 0.001)
	assert.InDelta(t, 0, topIn(t, tr, tr.Of[p], ib), 0.001)
}

func TestATopAlignedBoxAlignsWithAllItHolds(t *testing.T) {
	child := slab(30)
	span := el(dom.Span, valign(style.AlignTop), text("XX"), child)
	tr, p := aligned(nil, text("HH"), span)
	assert.InDelta(t, 32, tr.Of[p].Size.Y, 0.001)
	assert.InDelta(t, 8, baselineOf(t, tr.Of[p], "HH"), 0.001)
	assert.InDelta(t, 30, baselineOf(t, tr.Of[p], "XX"), 0.001)
	assert.InDelta(t, 0, topIn(t, tr, tr.Of[p], child), 0.001)

	child = slab(30, valign(style.AlignBottom))
	span = el(dom.Span, valign(style.AlignTop), text("XX"), child)
	tr, p = aligned(nil, text("HH"), span)
	assert.InDelta(t, 30, tr.Of[p].Size.Y, 0.001)
	assert.InDelta(t, 28, baselineOf(t, tr.Of[p], "XX"), 0.001, "a bottom box in it aligns to its bottom, as in Chrome")
	assert.InDelta(t, 0, topIn(t, tr, tr.Of[p], child), 0.001)
}

func TestAnInlineBoxWithNothingOnTheLineStillHoldsItOpen(t *testing.T) {
	empty := el(dom.Span, lineHeight(40))
	tr, p := aligned(nil, text("HH"), empty)
	assert.InDelta(t, 40, tr.Of[p].Size.Y, 0.001)
	assert.InDelta(t, 23, baselineOf(t, tr.Of[p], "HH"), 0.001)

	inner := el(dom.Span, nil, text("XX"))
	tr, p = aligned(nil, el(dom.Span, lineHeight(40), inner))
	assert.InDelta(t, 40, tr.Of[p].Size.Y, 0.001, "the outer box counts where only the inner one has text")
}

func TestABoxWithNoBaselineStandsOnItsBottomMargin(t *testing.T) {
	ib := slab(20, func(st *style.Computed) { st.Margin.Top, st.Margin.Bottom = style.Px(3), style.Px(5) })
	tr, p := aligned(nil, text("HH"), ib)
	assert.InDelta(t, 28, baselineOf(t, tr.Of[p], "HH"), 0.001)
	assert.InDelta(t, 3, topIn(t, tr, tr.Of[p], ib), 0.001)
	assert.InDelta(t, 30, tr.Of[p].Size.Y, 0.001)
}

func TestEachLineAlignsItsOwnBoxes(t *testing.T) {
	a, b := slab(30, valign(style.AlignTop)), slab(4, valign(style.AlignBottom))
	span := el(dom.Span, valign(style.AlignMiddle, fontSize(20)), text("XX"), el(dom.Br, nil), text("YY"), b)
	tr, p := aligned(nil, a, span)
	assert.InDelta(t, 30+20, tr.Of[p].Size.Y, 0.001)
	assert.InDelta(t, 30+16+2, topIn(t, tr, tr.Of[p], b), 0.001,
		"on the second line's bottom, then moved as far as the span holding it, as in Chrome")
}
