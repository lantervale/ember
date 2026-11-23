package layout //nolint:testpackage // tests the layout tree and its unexported helpers

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

func breaking(wb style.WordBreak, lb style.LineBreak, edits ...func(*style.Computed)) func(*style.Computed) {
	return func(st *style.Computed) {
		st.WordBreak, st.LineBreak = wb, lb
		for _, e := range edits {
			e(st)
		}
	}
}

func shrunk(st *style.Computed) {
	st.Display = style.InlineBlock
	st.Size.Width = style.Length{Kind: style.MinContent}
}

func TestBreakAllFillsEachLineBeforeBreakingAWord(t *testing.T) {
	plain := el(dom.P, width(50), text("HH HHHHH"))
	all := el(dom.P, breaking(style.BreakAll, style.LineBreakAuto, width(50)), text("HH HHHHH"))
	tr := lay(el(dom.Div, block, plain, all), 200)
	assert.Equal(t, []string{"HH", "HHHHH"}, lines(tr.Of[plain]))
	assert.Equal(t, []string{"HH HH", "HHH"}, lines(tr.Of[all]), "the long word is cut where the line ends")
}

func TestBreakAllAndAnywhereMakeMinContentOneGrapheme(t *testing.T) {
	all := el(dom.Div, breaking(style.BreakAll, style.LineBreakAuto, shrunk), text("HHH HH"))
	anywhere := el(dom.Div, breaking(style.WordBreakNormal, style.LineBreakAnywhere, shrunk), text("HHH HH"))
	word := el(dom.Div, breaking(style.WordBreakBreakWord, style.LineBreakAuto, shrunk), text("HHH HH"))
	keep := el(dom.Div, breaking(style.KeepAll, style.LineBreakAuto, shrunk), text("HHH HH"))
	tr := lay(el(dom.Div, block, el(dom.P, block, all), el(dom.P, block, anywhere), el(dom.P, block, word),
		el(dom.P, block, keep)), 200)
	assert.Equal(t, float32(10), tr.Of[all].Size.X)
	assert.Equal(t, float32(10), tr.Of[anywhere].Size.X)
	assert.Equal(t, float32(10), tr.Of[word].Size.X, "break-word is overflow-wrap: anywhere")
	assert.Equal(t, float32(30), tr.Of[keep].Size.X, "keep-all leaves words of letters whole")
}

// rows is the text of each line of b's runs, those of the elements it holds
// included, from the top.
func rows(b *box.Box) []string {
	at := map[float32]string{}
	var tops []float32
	var walk func(b *box.Box)
	walk = func(b *box.Box) {
		for _, f := range b.Lines {
			y := b.Rect().Y + f.Rect.Y
			if _, ok := at[y]; !ok {
				tops = append(tops, y)
			}
			at[y] += f.Text
		}
		for _, k := range b.Children {
			walk(k)
		}
	}
	walk(b)
	slices.Sort(tops)
	out := make([]string, len(tops))
	for i, y := range tops {
		out[i] = at[y]
	}
	return out
}

func TestBreakAllBreaksBetweenElementsByTheTextBefore(t *testing.T) {
	all := func(st *style.Computed) { st.WordBreak = style.BreakAll }
	normal := func(st *style.Computed) { st.WordBreak = style.WordBreakNormal }
	inner := el(dom.P, width(0), text("HI"), el(dom.Span, all, text("JK")), text("LM"))
	outer := el(dom.P, breaking(style.BreakAll, style.LineBreakAuto, width(0)), text("HI"),
		el(dom.Span, normal, text("JK")), text("LM"))
	tr := lay(el(dom.Div, block, inner, outer), 200)
	assert.Equal(t, []string{"HIJ", "K", "LM"}, rows(tr.Of[inner]), "a break after the span's text, none before")
	assert.Equal(t, []string{"H", "I", "JKL", "M"}, rows(tr.Of[outer]))
}

func TestBreakAllBreaksAsUAX14WithLettersAsIdeographs(t *testing.T) {
	for _, c := range []struct {
		a, b   string
		breaks bool
	}{
		{"a", "b", true}, {"a", "1", true}, {"1", "a", true}, {"a", "(", true}, {"(", "a", false},
		{"a", ")", false}, {")", "a", true}, {"a", ".", false}, {".", "b", true}, {"a", "-", true},
		{"-", "b", true}, {"a", "‐", false}, {"‐", "b", true}, {"a", "—", true},
		{"a", "%", false}, {"%", "b", true}, {"$", "b", false}, {"a", "$", true}, {"a", "'", false},
		{"'", "b", false}, {"a", "/", false}, {"/", "b", true}, {"a", "_", true}, {"_", "b", true},
		{"a", "。", false}, {"é́", "c", true}, {"a", "!", false},
	} {
		assert.Equal(t, c.breaks, breaksAll(c.a, c.b, style.StrictnessNormal), "%q %q", c.a, c.b)
	}
}
