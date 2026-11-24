package css

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

func TestInitialLetterTakesASizeAndASink(t *testing.T) {
	for v, want := range map[string]style.InitialLetter{
		"normal": {}, "3": {Size: 3, Sink: 3}, "2.5": {Size: 2.5, Sink: 2}, "3 2": {Size: 3, Sink: 2},
		"2.7 drop": {Size: 2.7, Sink: 2}, "3 drop": {Size: 3, Sink: 3}, "2 raise": {Size: 2, Sink: 1},
		"1.5 4": {Size: 1.5, Sink: 4},
	} {
		_, warns := Parse("t.css", "p { initial-letter: "+v+" }")
		require.Empty(t, warns, v)
		assert.Equal(t, want, initialLetterOf(nil, items(componentsOf(v))), v)
	}
	for _, v := range []string{"0.5", "-2", "2 1.5", "2 0", "drop", "2 3 4", "2px"} {
		_, warns := Parse("t.css", "p { initial-letter: "+v+" }")
		assert.Len(t, warns, 1, v)
	}
}

func TestAnInitialLetterIsAFloatAsTallAsItsLines(t *testing.T) {
	p := with(el(dom.P, ""), txt("Down the stairs"))
	raised := with(el(dom.P, "", "up"), txt("Raised"))
	half := with(el(dom.P, "", "half"), txt("Half"))
	root := page(p, raised, half)
	Compute(root, []*Sheet{sheet(t, `p { font-size: 10px; line-height: 20px; }
		p::first-letter { initial-letter: 3; } .up::first-letter { initial-letter: 3 1; padding-top: 2px; }
		.half::first-letter { initial-letter: 2.5; }`)})
	st := dom.Data(dom.Data(p).FirstLetter).Style
	// without fonts, a cap is an em tall and the ascent an em
	assert.Equal(t, style.FloatLeft, st.Float)
	assert.InDelta(t, 50, st.Font.Size, 1e-4, "two lines and a cap of the paragraph")
	assert.Equal(t, style.Px(50), st.Size.Height, "trimmed to its capitals")
	assert.InDelta(t, 2*20+(20-12)/2+10-50, st.Margin.Top.Px, 1e-4, "its baseline on the third line's")
	assert.Zero(t, st.InitialLetter.Raise)

	up := dom.Data(dom.Data(raised).FirstLetter).Style
	assert.Equal(t, float32(40), up.InitialLetter.Raise, "two lines over the first")
	assert.InDelta(t, (20-12)/2+10-50-2, up.Margin.Top.Px, 1e-4)

	h := dom.Data(dom.Data(half).FirstLetter).Style
	assert.Equal(t, float32(10), h.InitialLetter.Raise, "half a line over the first, sunk two")
	assert.InDelta(t, 20+(20-12)/2+10-40, h.Margin.Top.Px, 1e-4, "its baseline on the second line's")
}

func TestAFontLandingSizesADropCapAgain(t *testing.T) {
	p := with(el(dom.P, ""), txt("Down the stairs"))
	root := page(p)
	tr := measured()
	sheets := []*Sheet{sheet(t, `p { font-size: 10px; line-height: 20px; }`)}
	tr.Compute(root, sheets)
	assert.False(t, tr.FontsChanged())

	sheets = []*Sheet{sheet(t, `p { font-size: 10px; line-height: 20px; } p::first-letter { initial-letter: 2; }`)}
	tr.Compute(root, sheets)
	assert.True(t, tr.FontsChanged(), "the drop cap is sized by its capitals")
}

func TestAVerticalInitialLetterIsTheHorizontalOneTurned(t *testing.T) {
	rl := with(el(dom.P, "", "rl"), txt("Down the stairs"))
	lr := with(el(dom.P, "", "lr"), txt("Down the stairs"))
	Compute(page(rl, lr), []*Sheet{sheet(t, `p { font-size: 10px; line-height: 20px; }
		p::first-letter { initial-letter: 3; padding: 1px 2px 3px 4px; }
		.rl { writing-mode: vertical-rl; } .lr { writing-mode: vertical-lr; }`)})
	top := float32(2*20 + (20-12)/2 + 10 - 50)
	st := dom.Data(dom.Data(rl).FirstLetter).Style
	assert.Equal(t, style.Px(50), st.Size.Width, "its capitals across the lines")
	assert.Zero(t, st.Size.Height.Px)
	assert.InDelta(t, top-2, st.Margin.Right.Px, 1e-4, "its baseline on the third line's, from the right")
	assert.Zero(t, st.Margin.Top.Px)

	st = dom.Data(dom.Data(lr).FirstLetter).Style
	assert.Equal(t, style.Px(50), st.Size.Width)
	assert.InDelta(t, top-4, st.Margin.Left.Px, 1e-4, "from the left")
}
