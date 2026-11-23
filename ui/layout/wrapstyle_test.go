package layout //nolint:testpackage // tests the line breakers

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

func wrapped(w float32, ws style.WrapStyle) func(*style.Computed) {
	return func(st *style.Computed) {
		width(w)(st)
		st.WrapStyle = ws
	}
}

func spaceless(ls []string) []string {
	for i, l := range ls {
		ls[i] = strings.TrimSpace(l)
	}
	return ls
}

func TestBalanceEvensTheLines(t *testing.T) {
	p := el(dom.P, wrapped(150, style.WrapBalance), text("aa bb cc dd ee ff gg hh"))
	tr := lay(el(dom.Div, block, p), 200)
	assert.Equal(t, []string{"aa bb cc dd", "ee ff gg hh"}, spaceless(lines(tr.Of[p])),
		"greedy breaks after ee, two lines of 140 and 80; balanced, two of 110")
}

func TestBalanceBisectsAFewBreaks(t *testing.T) {
	p := el(dom.P, wrapped(150, style.WrapBalance), text("aaaaaa bbbbbb cc"))
	tr := lay(el(dom.Div, block, p), 200)
	assert.Equal(t, []string{"aaaaaa", "bbbbbb cc"}, spaceless(lines(tr.Of[p])),
		"two places to break are too few to score: the narrowest width keeping two lines")
}

func TestBalanceLeavesLongParagraphs(t *testing.T) {
	words := strings.Repeat("aa bb ", 12)
	p := el(dom.P, wrapped(80, style.WrapBalance), text(words))
	q := el(dom.P, width(80), text(words))
	tr := lay(el(dom.Div, block, p, q), 200)
	assert.Equal(t, lines(tr.Of[q]), lines(tr.Of[p]), "more than six lines are broken greedily")
}

func TestPrettyAvoidsAShortLastLine(t *testing.T) {
	p := el(dom.P, wrapped(100, style.WrapPretty), text("aaaaa bbb cccc ddd ee"))
	tr := lay(el(dom.Div, block, p), 200)
	assert.Equal(t, []string{"aaaaa", "bbb cccc", "ddd ee"}, spaceless(lines(tr.Of[p])),
		"greedy leaves ee alone on the last line")
}

func TestPrettyKeepsALongEnoughLastLine(t *testing.T) {
	p := el(dom.P, wrapped(100, style.WrapPretty), text("aaaa bbbb cccc ddd"))
	tr := lay(el(dom.Div, block, p), 200)
	assert.Equal(t, []string{"aaaa bbbb", "cccc ddd"}, spaceless(lines(tr.Of[p])))
}

func TestSoftHyphensBreakWithAHyphen(t *testing.T) {
	p := el(dom.P, width(50), text("aaa\u00adbbb\u00adc"))
	tr := lay(el(dom.Div, block, p), 200)
	ls := tr.Of[p].Children[0].Lines
	assert.Equal(t, []string{"aaa", "bbbc"}, lines(tr.Of[p]), "the soft hyphens take no room and show nothing")
	assert.Equal(t, "\u2010", ls[0].Hyphen)
	assert.InDelta(t, 40, ls[0].Rect.W, 0.01, "the hyphen takes its room at the line's end")
	assert.Empty(t, ls[1].Hyphen)
}

func TestHyphensNoneKeepsWordsWhole(t *testing.T) {
	p := el(dom.P, func(st *style.Computed) { width(50)(st); st.Hyphens = style.HyphensNone }, text("aaa\u00adbbb"))
	tr := lay(el(dom.Div, block, p), 200)
	assert.Equal(t, []string{"aaabbb"}, lines(tr.Of[p]))
}

func exToBaseline(st *style.Computed) {
	st.TextBoxTrim = style.TrimBoth
	st.TextBoxEdge = style.TextBoxEdge{Over: style.EdgeEx, Under: style.EdgeAlphabetic, Named: true}
}

func TestTextBoxTrimCutsTheLinesToTheirEdges(t *testing.T) {
	p := el(dom.P, func(st *style.Computed) { block(st); exToBaseline(st) }, text("HH"))
	q := el(dom.P, func(st *style.Computed) { block(st); st.TextBoxTrim = style.TrimEnd }, text("HH"))
	tr := lay(el(dom.Div, block, p, q), 200)
	assert.InDelta(t, 8, tr.Of[p].Size.Y, 0.001, "an x-height of 8 above the baseline, nothing below")
	assert.InDelta(t, 10, tr.Of[q].Size.Y, 0.001, "auto trims to the text's edges: no leading to trim")
	run := tr.Of[p].Children[0]
	assert.InDelta(t, 8, run.Pos.Y+run.Lines[0].Baseline, 0.001, "the baseline at the trimmed bottom")
}

func TestTextBoxTrimReachesTheFirstAndLastBlocks(t *testing.T) {
	capToBaseline := func(st *style.Computed) {
		exToBaseline(st)
		st.TextBoxEdge.Over = style.EdgeCap
	}
	first, middle, last := el(dom.P, block, text("HH")), el(dom.P, block, text("HH")), el(dom.P, block, text("HH"))
	padded := el(dom.P, func(st *style.Computed) { block(st); st.Padding.Top = style.Px(1) }, text("HH"))
	div := el(dom.Div, func(st *style.Computed) { block(st); capToBaseline(st) }, first, middle, last)
	other := el(dom.Div, func(st *style.Computed) { block(st); st.TextBoxTrim = style.TrimStart }, padded)
	tr := lay(el(dom.Div, block, div, other), 200)
	assert.InDelta(t, 2, tr.Of[first].Size.Y, 0.001, "the first line, in the first block, trimmed to its cap height")
	assert.InDelta(t, 10, tr.Of[middle].Size.Y, 0.001)
	assert.InDelta(t, 8, tr.Of[last].Size.Y, 0.001, "the last line, in the last block, trimmed to its baseline")
	assert.InDelta(t, 11, tr.Of[padded].Size.Y, 0.001, "the padding keeps the first block's top")
}

// spaciousEx trims to the x-height and the baseline lines of 20 whose baseline stands
// 13 down: 5 off a line's top, 7 off its bottom.
func spaciousEx(st *style.Computed) {
	exToBaseline(st)
	st.LineHeight = style.LineHeight{Factor: 2}
}

func TestAnonymousBlocksTrimOnlyAsTheirContainersFirstOrLastLines(t *testing.T) {
	first, last := el(dom.P, block, text("HH")), el(dom.P, block, text("HH"))
	div := el(dom.Div, func(st *style.Computed) { block(st); spaciousEx(st) }, first, text("HH"), last)
	flex := el(dom.Div, func(st *style.Computed) { st.Display = style.FlexDisplay; spaciousEx(st) }, text("HH"))
	tr := lay(el(dom.Div, block, div, flex), 200)
	assert.InDelta(t, 10+20+8, tr.Of[div].Size.Y, 0.001, "the lines between the blocks keep their room")
	assert.InDelta(t, 20, tr.Of[flex].Size.Y, 0.001, "a flex container's trim reaches none of its items")
}

func TestFloatsRiseWithTheTrimmedLineTheyStandOn(t *testing.T) {
	float := el(dom.Div, func(st *style.Computed) {
		st.Display, st.Float = style.Block, style.FloatLeft
		st.Size.Width, st.Size.Height = style.Px(10), style.Px(10)
	})
	before := el(dom.Div, func(st *style.Computed) {
		st.Display, st.Float = style.Block, style.FloatLeft
		st.Size.Width, st.Size.Height = style.Px(10), style.Px(10)
	})
	div := el(dom.Div, func(st *style.Computed) { block(st); spaciousEx(st) }, float, text("HH"))
	other := el(dom.Div, func(st *style.Computed) { block(st); spaciousEx(st) }, before,
		el(dom.P, block, text("HH")))
	tr := lay(el(dom.Div, block, div, other), 200)
	assert.InDelta(t, -5, tr.Of[float].Pos.Y, 0.001, "raised with the first line, by what the trim takes off it")
	assert.InDelta(t, 8, tr.Of[div].Size.Y, 0.001)
	assert.InDelta(t, 0, tr.Of[before].Pos.Y, 0.001, "a float before a block stands where the block starts")
}

func TestColumnsTrimTheLinesEachStartsAndEndsWith(t *testing.T) {
	p := para(4, func(st *style.Computed) { st.LineHeight = style.LineHeight{Factor: 2} })
	m := multicol(2, spaciousEx, p)
	tr := lay(m, 600)
	assert.InDelta(t, 34, tr.Of[m].Size.Y, 0.001,
		"balanced at the trimmed flow's half: two lines a column, the second line ends 7 above its bottom")
	assert.Equal(t, []geom.Rect{{X: 0, Y: 0, W: 190, H: 34}, {X: 210, Y: 0, W: 190, H: 28}}, parts(tr.Of[p]),
		"the second column shows the flow from 5 into its first line")
	run := tr.Of[p].Children[0]
	require.Len(t, run.Lines, 4)
	assert.InDelta(t, -5, run.Pos.Y+run.Lines[2].Rect.Y, 0.001, "the third line raised above the column's top")
}

func TestHyphenateCharacterIsShown(t *testing.T) {
	p := el(dom.P, func(st *style.Computed) {
		width(50)(st)
		st.HyphenChar, st.HyphenChosen = "=>", true
	}, text("aaa\u00adbbb"))
	tr := lay(el(dom.Div, block, p), 200)
	l := tr.Of[p].Children[0].Lines[0]
	assert.Equal(t, "=>", l.Hyphen)
	assert.InDelta(t, 50, l.Rect.W, 0.01)
}
