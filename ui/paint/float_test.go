package paint //nolint:testpackage // white-box: builds box trees by hand and reads the painter's own state

import (
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
	"github.com/stretchr/testify/assert"
)

func floated(children ...*box.Box) *box.Box {
	return el(0, 0, 40, 40, func(s *style.Computed) { s.Background, s.Float = red, style.FloatLeft }, children...)
}

func TestAFloatPaintsOverTheBlocksAfterItAndUnderTheirText(t *testing.T) {
	para := el(0, 0, 100, 60, bg(blue))
	para.Append(text(40, 0, "beside", para.Style))
	r := newRecorder()
	Paint(r, tree(el(0, 0, 100, 100, bg(white), floated(el(0, 0, 10, 10, bg(green))), para)))
	assert.Equal(t, "white blue red green", r.order(names), "the paragraph's background never covers the float")
	assert.Equal(t, "text", r.ops[len(r.ops)-1].Kind, "text comes after the floats")
}

func TestAnInlineBlockPaintsWholeWithItsLineOverTheBlocksAfterIt(t *testing.T) {
	inner := el(0, 0, 10, 10, bg(black))
	ib := el(0, 0, 40, 20, func(s *style.Computed) { s.Background, s.Display = green, style.InlineBlock }, inner)
	line := el(0, 0, 100, 20, nil, ib)
	after := el(0, 0, 100, 20, bg(red))
	tr := tree(el(0, 0, 100, 100, bg(white), line, after))
	r := newRecorder()
	Paint(r, tr)
	assert.Equal(t, "white red green black", r.order(names), "the next block's background goes under it")
	assert.Same(t, inner, HitTest(tr, geom.V2(5, 5)), "hit as it is painted")
}

func TestAFlexItemPaintsWholeAsAnInlineBlockDoes(t *testing.T) {
	first := el(0, 0, 40, 20, bg(green), el(0, 0, 60, 10, bg(black)))
	flex := el(0, 0, 100, 20, func(s *style.Computed) { s.Display = style.FlexDisplay }, first, el(40, 0, 40, 20, bg(red)))
	r := newRecorder()
	Paint(r, tree(el(0, 0, 100, 100, bg(white), flex)))
	assert.Equal(t, "white green black red", r.order(names), "what the first holds goes under the second")
}

func TestAFloatIsUnderThePositionedBoxesAndHitBeforeTheBlockBehindIt(t *testing.T) {
	f := floated()
	para := el(0, 0, 100, 60, bg(blue))
	over := el(0, 0, 20, 20, func(s *style.Computed) { s.Background, s.Position, s.ZAuto = black, style.Relative, true })
	tr := tree(el(0, 0, 100, 100, bg(white), over, f, para))
	r := newRecorder()
	Paint(r, tr)
	assert.Equal(t, "white blue red black", r.order(names))
	assert.Same(t, f, HitTest(tr, geom.V2(30, 30)), "the float is on top of the paragraph")
	assert.Same(t, over, HitTest(tr, geom.V2(10, 10)))
}

func TestListMarkersDrawTheirShapeInTheItemsColour(t *testing.T) {
	marker := func(ls style.ListStyle, text string) *recorder {
		m := &box.Box{Kind: box.Marker, Pos: geom.V2(-13, 4), Size: geom.V2(5, 5)}
		if text != "" {
			m.Lines = []box.Fragment{{Text: text, Baseline: 12}}
		}
		li := el(40, 0, 100, 20, func(s *style.Computed) { s.ListItem, s.ListStyle, s.Color = true, ls, red })
		m.Style = li.Style
		li.Append(m)
		r := newRecorder()
		Paint(r, tree(el(0, 0, 200, 100, nil, li)))
		return r
	}
	disc := marker(style.ListDisc, "").of("circle")
	if assert.Len(t, disc, 1) {
		assert.Equal(t, red, disc[0].Color)
	}
	assert.Len(t, marker(style.ListSquare, "").of("rect"), 1)
	assert.Len(t, marker(style.ListCircle, "").of("border"), 1, "a ring")
	num := marker(style.ListCounter, "3. ").of("text")
	if assert.Len(t, num, 1) {
		assert.Equal(t, "3. ", num[0].Text)
	}
	assert.Empty(t, marker(style.ListNone, "").ops)
}

func TestAFieldsetsBorderLeavesItsLegendClear(t *testing.T) {
	legend := el(10, 0, 30, 10, nil)
	legend.Node = dom.NewElement(dom.Legend)
	set := el(0, 0, 100, 40, func(s *style.Computed) {
		s.Border = style.Sides[float32]{Top: 2, Right: 2, Bottom: 2, Left: 2}
		s.BorderStyle = style.Sides[style.BorderStyle]{Top: style.BorderSolid, Right: style.BorderSolid,
			Bottom: style.BorderSolid, Left: style.BorderSolid}
	}, legend)
	set.Node = dom.NewElement(dom.Fieldset)
	set.Kind, legend.Kind = box.Element, box.Element
	set.Wrap.Top = 4
	r := newRecorder()
	Paint(r, tree(el(0, 0, 200, 100, nil, set)))
	var clips []geom.Rect
	for _, o := range r.of("push") {
		clips = append(clips, o.Rect)
	}
	assert.Equal(t, []geom.Rect{{X: 0, Y: 4, W: 10, H: 36}, {X: 40, Y: 4, W: 60, H: 36}, {X: 10, Y: 10, W: 30, H: 30}},
		clips, "the border is drawn left of, right of and below the legend")
}

func TestAStyledMarkerDrawsItsContentInItsOwnColour(t *testing.T) {
	item := dom.NewElement(dom.Li)
	li := el(40, 0, 100, 20, func(s *style.Computed) { s.ListItem, s.ListStyle, s.Color = true, style.ListDisc, red })
	li.Node = item
	own := *li.Style
	own.Color = blue
	dom.Data(item).Style, dom.Data(item).Shown, dom.Data(item).Marker = li.Style, li.Style, &own
	li.Append(&box.Box{Kind: box.Marker, Node: item, Style: &own, Pos: geom.V2(-13, 4), Size: geom.V2(9, 12),
		Lines: []box.Fragment{{Text: "+ ", Baseline: 12}}})
	r := newRecorder()
	Paint(r, tree(el(0, 0, 200, 100, nil, li)))
	texts := r.of("text")
	if assert.Len(t, texts, 1) {
		assert.Equal(t, "+ ", texts[0].Text)
		assert.Equal(t, blue, texts[0].Color)
	}
	assert.Empty(t, r.of("circle"), "content stands in place of the disc")
}
