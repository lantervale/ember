package layout //nolint:testpackage // tests the bidi pieces of inline layout

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

func rtl(edit func(*style.Computed)) func(*style.Computed) {
	return func(st *style.Computed) {
		if edit != nil {
			edit(st)
		}
		st.Dir = style.RTL
	}
}

func inlineEl(children ...*dom.Node) *dom.Node { return el(dom.Span, nil, children...) }

func TestABreakEndsItsLineOnTheLineEndSide(t *testing.T) {
	br := el(dom.Br, nil)
	p := el(dom.P, rtl(width(100)), inlineEl(text("HHH")), br, inlineEl(text("HH")))
	tr := lay(el(dom.Div, block, p), 200)
	assert.Equal(t, float32(70), tr.Of[br].Pos.X, "left of the first line's text, at its end")
}

func TestARightToLeftItemsMarkerStandsOnItsRight(t *testing.T) {
	a := el(dom.Li, rtl(listItem(style.ListDisc)), text("HH"))
	b := el(dom.Li, rtl(listItem(style.ListDisc)), text("HH"))
	dom.Data(b).MarkerText, dom.Data(b).HasMarkerText = "12. ", true
	tr := lay(el(dom.Div, width(100), el(dom.Ol, rtl(func(st *style.Computed) {
		st.Display = style.Block
		st.Padding.Right = style.Px(40)
	}), a, b)), 200)
	disc := markerOfBox(tr.Of[a])
	assert.Greater(t, disc.Pos.X, float32(60), "past the content's right")
	assert.Less(t, disc.Pos.X+disc.Size.X, float32(100), "within the list's padding")
	num := markerOfBox(tr.Of[b])
	assert.Equal(t, float32(60), num.Pos.X, "a number starts where the content ends")
	assert.Equal(t, " .12", num.Lines[0].Text, "the number in a right-to-left line")
}

func TestVisualText(t *testing.T) {
	assert.Equal(t, "abc", visualText("abc", 0))
	assert.Equal(t, "גבא", visualText("אבג", 0))
	assert.Equal(t, "12 בא", visualText("אב 12", 1))
	assert.Equal(t, "(א)", visualText("(א)", 1), "brackets mirrored")
}

func TestABlockOverridingBidiShowsItsTextInItsDirection(t *testing.T) {
	override := func(dir style.Direction) func(*style.Computed) {
		return func(st *style.Computed) {
			st.Dir = dir
			st.Bidi = style.BidiOverride
			st.Display = style.Block
		}
	}
	shown := func(b *box.Box) []string {
		var out []string
		for _, c := range b.Children {
			for _, f := range c.Lines {
				out = append(out, f.Shown())
			}
		}
		return out
	}
	rtlP := el(dom.P, override(style.RTL), text("txeT relliF"), el(dom.Br, nil), text("dc"))
	ltrP := el(dom.P, override(style.LTR), text("ab אבג"))
	tr := lay(el(dom.Div, block, rtlP, ltrP), 400)
	assert.Equal(t, []string{"Filler Text", "cd"}, shown(tr.Of[rtlP]), "each line, past the break too")
	assert.Equal(t, []string{"ab אבג"}, shown(tr.Of[ltrP]), "Hebrew left to right, as written")
}

func TestASpanBidiSplitsIsInPartsEachWithItsOwnEdges(t *testing.T) {
	framed := func(st *style.Computed) {
		st.Border, st.BorderStyle = style.All[float32](2), style.All(style.BorderSolid)
	}
	bdo := func(st *style.Computed) { st.Dir, st.Bidi = style.RTL, style.BidiOverride }
	span := el(dom.Span, framed, text("ab"), el(dom.Span, bdo, text("cd")))
	tr := lay(el(dom.P, func(st *style.Computed) { block(st); width(400)(st) }, span, el(dom.Span, bdo, text("ef"))), 400)
	b := tr.Of[span]
	require.Len(t, b.Fragments, 2, "ab, then fe of the other span, then dc")
	assert.Nil(t, b.Ends, "its start before ab on the first, its end after dc on the last")
	assert.Equal(t, float32(2*10+2), b.Fragments[0].W, "ab and its left border")
	assert.Equal(t, b.Fragments[0].X+b.Fragments[0].W+2*10, b.Fragments[1].X, "fe stands between them")

	rtlSpan := el(dom.Span, func(st *style.Computed) { framed(st); st.Dir = style.RTL }, text("aaaa bbbb"))
	tr = lay(el(dom.P, func(st *style.Computed) { block(st); width(50)(st) }, rtlSpan), 400)
	b = tr.Of[rtlSpan]
	require.Len(t, b.Fragments, 2)
	assert.Equal(t, []box.Ends{box.RightEnd, box.LeftEnd}, b.Ends,
		"right to left, its start on the right of its first line, its end on the left of its last")
}

func TestLineAndParagraphSeparatorsEndTheirLines(t *testing.T) {
	p := el(dom.P, func(st *style.Computed) { block(st); width(400)(st) }, text("ab\u2028cd \u2029 ef"))
	assert.Equal(t, []string{"ab", "cd", "ef"}, lines(lay(p, 400).Of[p]), "a break each, the spaces after them dropped")
}

func TestIsolatesSideBySideAreOrderedApart(t *testing.T) {
	bdo := func(st *style.Computed) { st.Dir, st.Bidi = style.RTL, style.BidiIsolateOverride }
	cd, ef := el(dom.Span, bdo, text("cd")), el(dom.Span, bdo, text("ef"))
	tr := lay(el(dom.P, func(st *style.Computed) { block(st); width(400)(st) }, text("ab "), cd, ef), 400)
	assert.Less(t, tr.Of[cd].Pos.X, tr.Of[ef].Pos.X,
		"each its own isolate, in the order they come, as Chrome's bdo")
}
