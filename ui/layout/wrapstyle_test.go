package layout //nolint:testpackage // tests the line breakers

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

func wrapped(w float32, ws style.WrapStyle) func(*style.Computed) {
	return func(st *style.Computed) {
		width(w)(st)
		st.WrapStyle = ws
	}
}

func trimmed(ls []string) []string {
	for i, l := range ls {
		ls[i] = strings.TrimSpace(l)
	}
	return ls
}

func TestBalanceEvensTheLines(t *testing.T) {
	p := el(dom.P, wrapped(150, style.WrapBalance), text("aa bb cc dd ee ff gg hh"))
	tr := lay(el(dom.Div, block, p), 200)
	assert.Equal(t, []string{"aa bb cc dd", "ee ff gg hh"}, trimmed(lines(tr.Of[p])),
		"greedy breaks after ee, two lines of 140 and 80; balanced, two of 110")
}

func TestBalanceBisectsAFewBreaks(t *testing.T) {
	p := el(dom.P, wrapped(150, style.WrapBalance), text("aaaaaa bbbbbb cc"))
	tr := lay(el(dom.Div, block, p), 200)
	assert.Equal(t, []string{"aaaaaa", "bbbbbb cc"}, trimmed(lines(tr.Of[p])),
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
	assert.Equal(t, []string{"aaaaa", "bbb cccc", "ddd ee"}, trimmed(lines(tr.Of[p])),
		"greedy leaves ee alone on the last line")
}

func TestPrettyKeepsALongEnoughLastLine(t *testing.T) {
	p := el(dom.P, wrapped(100, style.WrapPretty), text("aaaa bbbb cccc ddd"))
	tr := lay(el(dom.Div, block, p), 200)
	assert.Equal(t, []string{"aaaa bbbb", "cccc ddd"}, trimmed(lines(tr.Of[p])))
}

func TestSoftHyphensBreakWithAHyphen(t *testing.T) {
	p := el(dom.P, width(50), text("aaa­bbb­c"))
	tr := lay(el(dom.Div, block, p), 200)
	ls := tr.Of[p].Children[0].Lines
	assert.Equal(t, []string{"aaa", "bbbc"}, lines(tr.Of[p]), "the soft hyphens take no room and show nothing")
	assert.Equal(t, "‐", ls[0].Hyphen)
	assert.InDelta(t, 40, ls[0].Rect.W, 0.01, "the hyphen takes its room at the line's end")
	assert.Empty(t, ls[1].Hyphen)
}

func TestHyphensNoneKeepsWordsWhole(t *testing.T) {
	p := el(dom.P, func(st *style.Computed) { width(50)(st); st.Hyphens = style.HyphensNone }, text("aaa­bbb"))
	tr := lay(el(dom.Div, block, p), 200)
	assert.Equal(t, []string{"aaabbb"}, lines(tr.Of[p]))
}

func TestHyphenateCharacterIsShown(t *testing.T) {
	p := el(dom.P, func(st *style.Computed) {
		width(50)(st)
		st.HyphenChar, st.HyphenChosen = "=>", true
	}, text("aaa­bbb"))
	tr := lay(el(dom.Div, block, p), 200)
	l := tr.Of[p].Children[0].Lines[0]
	assert.Equal(t, "=>", l.Hyphen)
	assert.InDelta(t, 50, l.Rect.W, 0.01)
}
