package layout //nolint:testpackage // tests the layout tree and its unexported helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

func wordSpaced(px float32, edits ...func(*style.Computed)) func(*style.Computed) {
	return func(st *style.Computed) {
		st.WordSpacing = style.Spacing{Px: px}
		for _, e := range edits {
			e(st)
		}
	}
}

func whiteSpace(ws style.WhiteSpace, edits ...func(*style.Computed)) func(*style.Computed) {
	return func(st *style.Computed) {
		st.WhiteSpace = ws
		for _, e := range edits {
			e(st)
		}
	}
}

func TestWordSpacingWidensEachSpaceBetweenWords(t *testing.T) {
	s := el(dom.Span, wordSpaced(5), text("HH HH H"))
	tr := lay(el(dom.Div, block, el(dom.P, block, s)), 200)
	assert.Equal(t, float32(80), tr.Of[s].Size.X, "seven glyphs and two spaces 5 wider")
	frags := tr.Of[s].Children[0].Lines
	require.Len(t, frags, 1)
	assert.Equal(t, "HH HH H", frags[0].Text)
	assert.Equal(t, float32(80), frags[0].Rect.W)
}

func TestWordSpacingMovesWhereALineWraps(t *testing.T) {
	plain := el(dom.P, width(50), text("HH HH HHH H"))
	spaced := el(dom.P, wordSpaced(5, width(50)), text("HH HH HHH H"))
	tight := el(dom.P, wordSpaced(-10, width(50)), text("HH HH HHH H"))
	tr := lay(el(dom.Div, block, plain, spaced, tight), 200)
	assert.Equal(t, []string{"HH HH", "HHH H"}, lines(tr.Of[plain]))
	assert.Equal(t, []string{"HH", "HH", "HHH", "H"}, lines(tr.Of[spaced]), "HH HH is 55 wide in 50")
	assert.Equal(t, []string{"HH HH", "HHH H"}, lines(tr.Of[tight]), "HH HH HHH would be 70 wide still")
	assert.Equal(t, float32(40), tr.Of[tight].Children[0].Lines[0].Rect.W, "the spaces take no room at -10")
}

func TestASpaceThatEndsALineHangsWithItsWordSpacing(t *testing.T) {
	p := el(dom.P, wordSpaced(30, width(50)), text("HH HHH "))
	right := el(dom.P, wordSpaced(30, width(50), func(st *style.Computed) { st.TextAlign = style.Right }), text("HH HHH"))
	tr := lay(el(dom.Div, block, p, right), 200)
	assert.Equal(t, []string{"HH", "HHH"}, lines(tr.Of[p]))
	assert.Zero(t, tr.Of[p].Content.X-50, "nothing overflows")
	run := tr.Of[right].Children[0]
	for _, f := range run.Lines {
		assert.Equal(t, float32(50), run.Pos.X+f.Rect.X+f.Rect.W, "%q ends at the right edge", f.Text)
	}
}

func TestWordSpacingCountsInMinAndMaxContent(t *testing.T) {
	hug := el(dom.Div, wordSpaced(5, inlineBlock), text("HH HHH H "))
	narrow := el(dom.Div, wordSpaced(5, func(st *style.Computed) {
		st.Display = style.InlineBlock
		st.Size.Width = style.Length{Kind: style.MinContent}
	}), text("HH HHH H"))
	tr := lay(el(dom.Div, block, el(dom.P, block, hug), el(dom.P, block, narrow)), 200)
	assert.Equal(t, float32(90), tr.Of[hug].Size.X, "six glyphs, two spaces of 15, the last space collapsed away")
	assert.Equal(t, float32(30), tr.Of[narrow].Size.X, "the longest word: a space that ends a line hangs")
}

func TestEachPreservedSpaceTakesWordSpacingAndCollapsedOnesTakeItOnce(t *testing.T) {
	pre := el(dom.Span, wordSpaced(5, whiteSpace(style.Pre)), text("H  H "))
	wrap := el(dom.Span, wordSpaced(5, whiteSpace(style.PreWrap)), text(" H  H"))
	normal := el(dom.Span, wordSpaced(5), text("H \n\t H"))
	tr := lay(el(dom.Div, block,
		el(dom.P, whiteSpace(style.Pre, block), pre),
		el(dom.P, whiteSpace(style.PreWrap, block), wrap),
		el(dom.P, block, normal),
	), 200)
	assert.Equal(t, float32(65), tr.Of[pre].Size.X, "two glyphs and three spaces of 15")
	assert.Equal(t, float32(65), tr.Of[wrap].Size.X, "a line's first space too")
	assert.Equal(t, float32(35), tr.Of[normal].Size.X, "one space left of the run, widened once")
}

func TestANoBreakSpaceTakesWordSpacingAndBreaksNothing(t *testing.T) {
	p := el(dom.P, wordSpaced(5, width(30)), text("H\u00a0H\u00a0H H"))
	tr := lay(el(dom.Div, block, p), 200)
	frags := tr.Of[p].Children[0].Lines
	require.Len(t, frags, 2)
	assert.Equal(t, "H\u00a0H\u00a0H", frags[0].Text)
	assert.Equal(t, float32(60), frags[0].Rect.W, "five glyphs and 5 after each no-break space")
}

func TestOtherSpacesTakeNoWordSpacing(t *testing.T) {
	for _, sep := range []string{"\u2003", "\u2009", "\u202f", "\u3000", "\u1361"} {
		s := el(dom.Span, wordSpaced(5), text("H"+sep+"H"))
		tr := lay(el(dom.Div, block, el(dom.P, block, s)), 200)
		assert.Equal(t, float32(30), tr.Of[s].Size.X, "%U", []rune(sep)[0])
	}
}

func TestJustifyingAddsToWordSpacing(t *testing.T) {
	p := el(dom.P, wordSpaced(5, width(80), func(st *style.Computed) { st.TextAlign = style.Justify }),
		text("H H HH HH H"))
	tr := lay(el(dom.Div, block, p), 200)
	var first []float32
	for _, f := range tr.Of[p].Children[0].Lines {
		if f.Rect.Y == 0 {
			first = append(first, f.Rect.X)
		}
	}
	assert.Equal(t, []float32{0, 30, 60}, first, "H H HH is 70 wide in 80 with its spaces of 15: 5 more a space")
}

func TestASpaceTakesTheWordSpacingOfTheElementItIsIn(t *testing.T) {
	inside := el(dom.Span, wordSpaced(5), text("HH "))
	after := el(dom.Span, nil, text("HH"))
	before := el(dom.Span, nil, text("HH"))
	leading := el(dom.Span, wordSpaced(5), text(" HH"))
	plain := el(dom.Span, nil, text("HH "))
	dropped := el(dom.Span, wordSpaced(5), text(" HH"))
	tr := lay(el(dom.Div, block,
		el(dom.P, block, inside, after),
		el(dom.P, block, before, leading),
		el(dom.P, block, plain, dropped),
	), 200)
	assert.Equal(t, float32(35), tr.Of[inside].Size.X, "the space before the span's end, widened")
	assert.Equal(t, float32(35), tr.Of[after].Rect().X)
	assert.Equal(t, float32(20), tr.Of[leading].Rect().X)
	assert.Equal(t, float32(35), tr.Of[leading].Size.X, "the space after the span's start")
	assert.Equal(t, float32(30), tr.Of[plain].Size.X, "the first of two collapsed spaces stays, unwidened")
	assert.Equal(t, float32(20), tr.Of[dropped].Size.X)
}

func TestAChildKeepsItsParentsWordSpacingLengthAndTakesAPercentageOfItsOwnFont(t *testing.T) {
	big := func(st *style.Computed) { st.Font.Size = 20 }
	length := el(dom.Span, wordSpaced(5, big), text("H H"))
	pct := el(dom.Span, func(st *style.Computed) { big(st); st.WordSpacing = style.Spacing{Percent: 50} },
		text("H H"))
	tr := lay(el(dom.Div, block, el(dom.P, block, length), el(dom.P, block, pct)), 200)
	assert.Equal(t, float32(65), tr.Of[length].Size.X, "three glyphs of 20 and 5")
	assert.Equal(t, float32(70), tr.Of[pct].Size.X, "three glyphs of 20 and half of 20")
}

func TestWordsBrokenAnywhereKeepTheirWordSpacing(t *testing.T) {
	p := el(dom.P, wordSpaced(5, width(200), func(st *style.Computed) { st.BreakWord = true }), text("H\u00a0HH H"))
	tr := lay(el(dom.Div, block, p), 200)
	assert.Equal(t, float32(70), tr.Of[p].Children[0].Lines[0].Rect.W, "six glyphs, 5 after each of the two spaces")
}

func TestAnEllipsisCutsWordSpacedTextWhereItFits(t *testing.T) {
	p := el(dom.P, wordSpaced(5, func(st *style.Computed) {
		width(55)(st)
		st.WhiteSpace = style.NoWrapSpace
		st.Overflow = style.Size[style.Overflow]{Width: style.Hidden, Height: style.Hidden}
		st.Ellipsis = true
	}), text("H H H H H"))
	tr := lay(el(dom.Div, block, p), 200)
	frags := tr.Of[p].Children[0].Lines
	require.Len(t, frags, 1)
	assert.Equal(t, "H H…", frags[0].Text, "H H and its space are 50 wide, the ellipsis 10 more")
}

func TestANumberedMarkerIsAsWideAsItsWordSpacedText(t *testing.T) {
	li := el(dom.Li, wordSpaced(5, listItem(style.ListCounter)), text("HH"))
	dom.Data(li).MarkerText, dom.Data(li).HasMarkerText = "1. ", true
	tr := lay(el(dom.Div, width(100), el(dom.Ol, block, li)), 200)
	m := markerOfBox(tr.Of[li])
	require.NotNil(t, m)
	assert.Equal(t, geom.Vec2{X: -35, Y: 0}, m.Pos, "three glyphs and the space's 5")
}

func TestTextIndentIsNotWordSpaced(t *testing.T) {
	p := el(dom.P, wordSpaced(5, block, func(st *style.Computed) { st.TextIndent = style.Px(20) }), text("HH HH"))
	tr := lay(el(dom.Div, block, p), 200)
	run := tr.Of[p].Children[0]
	assert.Equal(t, float32(20), run.Pos.X)
	assert.Equal(t, float32(55), run.Lines[0].Rect.W)
}

func TestSpacingFarBelowZeroLeavesNoNegativeWidth(t *testing.T) {
	s := el(dom.Span, wordSpaced(-100), text("H H H"))
	hug := el(dom.Div, inlineBlock, s)
	tight := el(dom.Span, func(st *style.Computed) { st.LetterSpacing = style.Spacing{Px: -30} }, text("HHH"))
	none := el(dom.Div, inlineBlock, tight)
	wrapped := el(dom.P, wordSpaced(-100, width(15)), text("H H H"))
	tr := lay(el(dom.Div, block, el(dom.P, block, hug), el(dom.P, block, none), wrapped), 200)
	assert.Equal(t, float32(10), tr.Of[hug].Size.X, "the first glyph, as Chrome's max-content")
	assert.Zero(t, tr.Of[s].Size.X, "a span of what is left of it: nothing")
	assert.Zero(t, tr.Of[s].Children[0].Lines[0].Rect.W)
	assert.Zero(t, tr.Of[none].Size.X)
	assert.Zero(t, tr.Of[tight].Size.X)
	assert.Equal(t, []string{"H H H"}, lines(tr.Of[wrapped]), "nothing to wrap: the line never grows")
	assert.Equal(t, geom.Vec2{X: 15, Y: 10}, tr.Of[wrapped].Size)
}
