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
