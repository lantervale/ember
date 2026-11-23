package layout //nolint:testpackage // tests the layout tree and its unexported helpers

import (
	"testing"
	"unicode"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

// hanFonts is Ahem with a halt feature: full-width brackets and dots set
// half an em wide, an opening one moved half an em left.
type hanFonts struct{ ahemFonts }

func (hanFonts) HanGlyph(f style.Font, r rune) box.HanGlyph {
	g := box.HanGlyph{Halt: true, Full: true}
	switch {
	case unicode.Is(unicode.Ps, r):
		g.Shift, g.Narrow = -f.Size/2, -f.Size/2
	case unicode.Is(unicode.Pe, r):
		g.Narrow = -f.Size / 2
	case r == '。' || r == '、':
		g.Narrow, g.Ink = -f.Size/2, box.InkLeft
	}
	return g
}

func hanText(edit func(*style.Computed), w float32, s string) (*box.Box, []box.Fragment) {
	p := el(dom.P, func(st *style.Computed) {
		width(w)(st)
		if edit != nil {
			edit(st)
		}
	}, text(s))
	tr := Layout(el(dom.Div, block, p), geom.V2(200, 600), hanFonts{})
	b := tr.Of[p]
	return b, b.Children[0].Lines
}

func TestFullWidthPunctuationAfterPunctuationIsSetHalfWide(t *testing.T) {
	_, frags := hanText(nil, 200, "」「。」」」")
	require.Len(t, frags, 1)
	assert.Equal(t, float32(40), frags[0].Rect.W,
		"the opening bracket after a closing one, the dot before a closing bracket, a closing bracket before another")
	assert.Equal(t, []box.Spacing{{At: 3, Shift: -5, Space: -5}, {At: 6, Space: -5}, {At: 9, Space: -5},
		{At: 12, Space: -5}}, frags[0].Spacing)

	_, frags = hanText(func(st *style.Computed) { st.SpacingTrim = style.SpaceAll }, 200, "」「。」」」")
	assert.Equal(t, float32(60), frags[0].Rect.W, "space-all trims none")
	assert.Nil(t, frags[0].Spacing)
}

func TestALineStartsWithItsOpeningBracketAsTextSpacingTrimSays(t *testing.T) {
	const s = "漢字漢字」「漢字漢字漢"
	for _, c := range []struct {
		trim  style.SpacingTrim
		lines []string
		width []float32
	}{
		{style.SpacingTrimNormal, []string{"漢字漢字」", "「漢字漢字", "漢"}, []float32{50, 50, 10}},
		{style.SpaceFirst, []string{"漢字漢字」", "「漢字漢字", "漢"}, []float32{50, 45, 10}},
		{style.TrimStartSpacing, []string{"漢字漢字」", "「漢字漢字", "漢"}, []float32{50, 45, 10}},
	} {
		b, frags := hanText(func(st *style.Computed) { st.SpacingTrim = c.trim }, 50, s)
		assert.Equal(t, c.lines, lines(b), "trim %d", c.trim)
		for i, f := range frags {
			assert.Equal(t, c.width[i], f.Rect.W, "trim %d line %d", c.trim, i)
		}
	}
	_, frags := hanText(func(st *style.Computed) { st.SpacingTrim = style.TrimStartSpacing }, 50, "「漢」")
	assert.Equal(t, float32(25), frags[0].Rect.W, "trim-start trims the block's first line too")
	_, frags = hanText(func(st *style.Computed) { st.SpacingTrim = style.SpaceFirst }, 50, "「漢」")
	assert.Equal(t, float32(30), frags[0].Rect.W, "space-first leaves it")
}

func TestAClosingBracketALineWouldOverflowIsTrimmedToFit(t *testing.T) {
	b, frags := hanText(nil, 45, "字字字字」字")
	assert.Equal(t, []string{"字字字字」", "字"}, lines(b))
	assert.Equal(t, float32(45), frags[0].Rect.W)
	assert.Equal(t, []box.Spacing{{At: 12, Space: -5}}, frags[0].Spacing)

	b, _ = hanText(nil, 45, "字字字字。字")
	assert.Equal(t, []string{"字字字", "字。字"}, lines(b), "a dot is not")
	b, _ = hanText(func(st *style.Computed) { st.SpacingTrim = style.SpaceAll }, 45, "字字字字」字")
	assert.Equal(t, []string{"字字字", "字」字"}, lines(b), "nor under space-all")
	b, frags = hanText(nil, 45, "字字字字」")
	assert.Equal(t, []string{"字字字字」"}, lines(b), "the content's end fits it, as Chrome measures it")
	assert.Equal(t, float32(50), frags[0].Rect.W, "and draws it whole")
}

func TestAutospaceAddsAnEighthOfTheSizeBetweenIdeographsAndLetters(t *testing.T) {
	auto := func(st *style.Computed) { st.Autospace = true }
	_, frags := hanText(auto, 200, "漢A1漢 A")
	assert.Equal(t, float32(62.5), frags[0].Rect.W)
	assert.Equal(t, []box.Spacing{{At: 0, Space: 1.25}, {At: 4, Space: 1.25}}, frags[0].Spacing)

	_, frags = hanText(nil, 200, "漢A1漢 A")
	assert.Equal(t, float32(60), frags[0].Rect.W, "no-autospace adds none")

	b, frags := hanText(auto, 21, "漢漢A")
	assert.Equal(t, []string{"漢漢", "A"}, lines(b), "a line ending between them drops the room")
	assert.Equal(t, float32(20), frags[0].Rect.W)
	assert.Nil(t, frags[0].Spacing)
}

func TestTextSpacingClasses(t *testing.T) {
	for r, want := range map[rune]spaceClass{
		'漢': ideograph, 'か': ideograph, 'ー': ideograph, 'ㄅ': ideograph, '〆': ideograph, '々': ideograph,
		'・': spaceOther, '゠': spaceOther, 'ｶ': spaceOther, 'A': alphanumeric, '1': alphanumeric, 'é': alphanumeric,
		'Ａ': spaceOther, '%': spaceOther, '²': spaceOther,
	} {
		assert.Equal(t, want, spaceClassOf(r), "%c", r)
	}
	full, narrow := box.HanGlyph{Full: true}, box.HanGlyph{}
	for _, c := range []struct {
		r    rune
		g    box.HanGlyph
		want hanClass
	}{
		{'「', narrow, hanOpen}, {'」', narrow, hanClose}, {'(', narrow, hanOpenNarrow}, {'｣', narrow, hanCloseNarrow},
		{'“', narrow, hanOpenNarrow}, {'“', full, hanOpen}, {'”', full, hanClose},
		{'。', box.HanGlyph{Ink: box.InkLeft}, hanDot}, {'：', box.HanGlyph{Ink: box.InkCentred}, hanMiddle},
		{'。', narrow, hanOther}, {'・', narrow, hanMiddle}, {'　', narrow, hanMiddle}, {'！', narrow, hanOther},
	} {
		assert.Equal(t, c.want, hanClassOf(c.r, c.g), "%c", c.r)
	}
	assert.True(t, mayHan("a漢"))
	assert.True(t, mayHan("“x”"))
	assert.True(t, mayHan("a·b"))
	assert.False(t, mayHan("plain text — with a dash"))
}

func TestALineBreaksAroundCurlyQuotesBetweenIdeographs(t *testing.T) {
	var at []int
	quoteBreaks("字“漢”字 a“b” 「“漢”」", func(pos int) { at = append(at, pos) })
	assert.Equal(t, []int{3, 12}, at, "before the opening quote and after the closing one, between ideographs alone")
}

func TestAnElementsLastCharacterKeepsItsAutospaceAtALinesEnd(t *testing.T) {
	auto := func(st *style.Computed) { st.Autospace = true }
	span := el(dom.Span, auto, text("字"))
	p := el(dom.P, func(st *style.Computed) { width(50)(st); auto(st) }, text("字字字字"), span, text("A漢"))
	tr := Layout(el(dom.Div, block, p), geom.V2(200, 600), hanFonts{})
	assert.Equal(t, []string{"字字字字", "A漢"}, lines(tr.Of[p]),
		"Chrome keeps the room where its element ends, which the first line has no room for")
	require.Len(t, tr.Of[span].Children[0].Lines, 1)
	assert.Equal(t, float32(11.25), tr.Of[span].Children[0].Lines[0].Rect.W)
}
