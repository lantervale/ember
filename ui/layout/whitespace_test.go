package layout //nolint:testpackage // tests the layout tree and its unexported helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

func padRight(px float32) func(*style.Computed) {
	return func(st *style.Computed) { st.Padding.Right = style.Px(px) }
}

func padLeft5(st *style.Computed) { st.Padding.Left = style.Px(5) }

func TestASpanAloneInItsBlockIsAsWideAsItsWords(t *testing.T) {
	a := el(dom.Span, nil, text("\n\t\t\tHH HH\t\t\t"))
	li := el(dom.Li, block, a, text("\n\t\t"))
	tr := lay(el(dom.Div, width(200), li), 200)
	assert.Equal(t, geom.Rect{W: 50, H: 10}, tr.Of[a].Rect(), "neither the space before HH nor the one after HH")
	assert.Equal(t, []string{"HH HH"}, lines(tr.Of[a]))
}

func TestSpacesAZeroWidthSpaceKeepsApartNeverCollapse(t *testing.T) {
	apart := el(dom.Span, nil, text("\u200b \u200b \u200b \u200b"))
	inside := el(dom.Span, nil, text(" \u200b \u200b "))
	tr := lay(el(dom.Div, block,
		el(dom.P, block, text("H"), apart, text("H")),
		el(dom.P, block, text("H"), inside, text("H")),
	), 200)
	assert.InDelta(t, 30, tr.Of[apart].Rect().W, 1e-3, "three spaces, a zero-width space between each")
	assert.InDelta(t, 30, tr.Of[inside].Rect().W, 1e-3)
}

func TestASpaceBeforeASpansEndIsDroppedAtTheLinesEnd(t *testing.T) {
	s := el(dom.Span, nil, text("HH  "))
	inner := el(dom.Span, nil, text(" HH "))
	outer := el(dom.Span, nil, inner, text(" "))
	blank := el(dom.Span, nil, text(" "))
	padded := el(dom.Span, padRight(5), text("HH "))
	empty := el(dom.Span, nil)
	tr := lay(el(dom.Div, block,
		el(dom.P, block, text("HH "), s),
		el(dom.P, block, outer),
		el(dom.P, block, text("HH"), blank),
		el(dom.P, block, text("HH "), padded, text(" "), empty),
	), 200)
	assert.Equal(t, float32(30), tr.Of[s].Rect().X)
	assert.Equal(t, float32(20), tr.Of[s].Size.X)
	assert.Equal(t, float32(20), tr.Of[inner].Size.X, "a nested span's too")
	assert.Equal(t, float32(20), tr.Of[outer].Size.X)
	assert.Equal(t, float32(20), tr.Of[blank].Rect().X, "a span of spaces alone ends where it starts")
	assert.Zero(t, tr.Of[blank].Size.X)
	assert.Equal(t, float32(25), tr.Of[padded].Size.X, "its padding follows its last word")
	assert.Equal(t, float32(55), tr.Of[empty].Rect().X, "what follows it on the line has no space before it either")
}

func TestSpacesInsideASpanStayInTheMiddleOfALine(t *testing.T) {
	s := el(dom.Span, nil, text(" H "))
	last := el(dom.Span, nil, text("H "))
	p := el(dom.P, block, text("H"), s, text("H "), last, text("H"))
	tr := lay(el(dom.Div, block, p), 200)
	assert.Equal(t, float32(10), tr.Of[s].Rect().X)
	assert.Equal(t, float32(30), tr.Of[s].Size.X)
	assert.Equal(t, float32(20), tr.Of[last].Size.X, "the space before its end, words following")
	assert.Equal(t, float32(90), tr.Of[p].Content.X)
}

func TestASpaceBeforeASpansEndAtAWrap(t *testing.T) {
	s := el(dom.Span, nil, text("HHH HH "))
	p := el(dom.P, width(50), text("HH "), s, text("HHH"))
	tr := lay(el(dom.Div, block, p), 200)
	assert.Equal(t, float32(40), tr.Of[p].Size.Y, "HH, HHH, HH and HHH: the span's end takes no line")
	frags := tr.Of[s].Fragments
	require.Len(t, frags, 2)
	assert.Equal(t, float32(20), frags[1].W, "HH without the space after it")

	fit := el(dom.Span, nil, text("HH "))
	p = el(dom.P, width(50), text("HH "), fit, text("HH"))
	tr = lay(el(dom.Div, block, p), 200)
	assert.Equal(t, float32(20), tr.Of[p].Size.Y, "HH HH fills the line, the space past its end")
	assert.Equal(t, float32(20), tr.Of[fit].Size.X)

	padded := el(dom.Span, padRight(5), text("HH "))
	p = el(dom.P, width(50), text("HH "), padded, text("HH"))
	tr = lay(el(dom.Div, block, p), 200)
	assert.Equal(t, float32(20), tr.Of[p].Size.Y, "HH HH and the padding past the line's end, then HH")
	assert.Equal(t, geom.Rect{X: 30, W: 25, H: 10}, tr.Of[padded].Rect(), "its end stays with its last word")
}

func TestASpanStartingWithASpaceAtAWrap(t *testing.T) {
	s := el(dom.Span, nil, text(" HHH"))
	p := el(dom.P, width(50), text("HHH"), s)
	tr := lay(el(dom.Div, block, p), 200)
	require.Len(t, tr.Of[s].Fragments, 1, "nothing of it on the first line")
	assert.Equal(t, geom.Rect{Y: 10, W: 30, H: 10}, tr.Of[s].Rect())

	fits := el(dom.Span, padLeft5, text(" HHH"))
	p = el(dom.P, width(50), text("HHH"), fits)
	tr = lay(el(dom.Div, block, p), 200)
	require.Len(t, tr.Of[fits].Fragments, 2)
	assert.Equal(t, geom.Rect{X: 30, W: 5, H: 10}, tr.Of[fits].Fragments[0], "its padding fits after HHH and stays")
	assert.Equal(t, geom.Rect{Y: 10, W: 30, H: 10}, tr.Of[fits].Fragments[1])

	s = el(dom.Span, padLeft5, text(" HHH"))
	p = el(dom.P, width(32), text("HHH"), s)
	tr = lay(el(dom.Div, block, p), 200)
	require.Len(t, tr.Of[s].Fragments, 1, "its padding has no room after HHH")
	assert.Equal(t, geom.Rect{Y: 10, W: 35, H: 10}, tr.Of[s].Rect())
	assert.Equal(t, float32(5), tr.Of[s].Children[0].Rect().X, "no space between its padding and its word")

	same := el(dom.Span, padLeft5, text(" H"))
	p = el(dom.P, width(60), text("HHH"), same)
	tr = lay(el(dom.Div, block, p), 200)
	assert.Equal(t, geom.Rect{X: 30, W: 25, H: 10}, tr.Of[same].Rect(), "on the same line the space stays")
}

func TestPreservedSpacesStayAtASpansEdges(t *testing.T) {
	for _, ws := range []style.WhiteSpace{style.Pre, style.PreWrap} {
		keep := func(st *style.Computed) { st.WhiteSpace = ws }
		s := el(dom.Span, keep, text(" H "))
		tr := lay(el(dom.Div, block, el(dom.P, func(st *style.Computed) { block(st); keep(st) }, s)), 200)
		assert.Equal(t, float32(30), tr.Of[s].Size.X)
	}
}

func TestContentWidthsLeaveOutSpacesAtASpansEdges(t *testing.T) {
	sized := func(w style.Length, kids ...*dom.Node) *dom.Node {
		return el(dom.Div, func(st *style.Computed) {
			st.Display = style.InlineBlock
			st.Size.Width = w
		}, kids...)
	}
	fit := sized(style.Length{}, el(dom.Span, nil, text(" HH HHH ")))
	minC := sized(style.Length{Kind: style.MinContent}, text("H "), el(dom.Span, padRight(5), text("HH HHH ")))
	lead := sized(style.Length{Kind: style.MinContent}, text("HHH"), el(dom.Span, padLeft5, text(" H")))
	tr := lay(el(dom.Div, block, el(dom.Div, block, fit), el(dom.Div, block, minC), el(dom.Div, block, lead)), 200)
	assert.Equal(t, float32(60), tr.Of[fit].Size.X, "shrunk to its words")
	assert.Equal(t, float32(35), tr.Of[minC].Size.X, "HHH and the padding at the span's end")
	assert.Equal(t, float32(35), tr.Of[lead].Size.X, "HHH and the padding at the span's start")
}

func TestSpacesAtASpansEdgesAlign(t *testing.T) {
	aligned := func(a style.TextAlign, kids ...*dom.Node) *dom.Node {
		return el(dom.P, func(st *style.Computed) { width(100)(st); st.TextAlign = a }, kids...)
	}
	right := el(dom.Span, nil, text(" HH "))
	centre := el(dom.Span, nil, text(" HH "))
	first := el(dom.Span, nil, text("HH HH "))
	tr := lay(el(dom.Div, block,
		aligned(style.Right, right),
		aligned(style.Center, centre),
		aligned(style.Justify, text("HH "), first, text("HHHHHH")),
	), 200)
	assert.Equal(t, float32(80), tr.Of[right].Rect().X)
	assert.Equal(t, float32(40), tr.Of[centre].Rect().X)
	assert.Equal(t, float32(60), tr.Of[first].Size.X, "the space inside it grows, the one before its end is no gap")
}
