package paint //nolint:testpackage // white-box: builds box trees by hand

import (
	"testing"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/style"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// spaced paints s in 10-point Ahem with a word-spacing of 5 and a
// letter-spacing of letter, and returns the texts drawn.
func spaced(s string, letter float32, edit func(*style.Computed)) []op {
	p := el(0, 0, 400, 20, func(st *style.Computed) {
		st.Color = white
		st.Font.Size = 10
		st.LetterSpacing = letter
		st.WordSpacing = style.WordSpacing{Px: 5}
		if edit != nil {
			edit(st)
		}
	})
	p.Append(text(0, 0, s, p.Style))
	r := newRecorder()
	Paint(r, tree(p), Fonts(ahem{}))
	return r.of("text")
}

func places(ops []op) ([]string, []float32) {
	var texts []string
	var xs []float32
	for _, o := range ops {
		texts, xs = append(texts, o.Text), append(xs, o.At.X)
	}
	return texts, xs
}

func TestWordSpacedTextIsDrawnWordByWordEachAfterItsWidenedSpace(t *testing.T) {
	texts, xs := places(spaced("ab cd e", 0, nil))
	assert.Equal(t, []string{"ab", "cd", "e"}, texts)
	assert.Equal(t, []float32{0, 35, 70}, xs, "a space is 10 wide and 5 wider")
}

func TestTextWithoutWordSpacingIsDrawnWhole(t *testing.T) {
	texts, xs := places(spaced("ab cd e", 0, func(st *style.Computed) { st.WordSpacing = style.WordSpacing{} }))
	assert.Equal(t, []string{"ab cd e"}, texts)
	assert.Equal(t, []float32{0}, xs)
}

func TestANoBreakSpaceAndEachSpaceOfARunAreWidenedAsDrawn(t *testing.T) {
	texts, xs := places(spaced("  a\u00a0b ", 0, nil))
	assert.Equal(t, []string{"a", "b"}, texts)
	assert.Equal(t, []float32{30, 65}, xs, "two spaces of 15, then a letter and a no-break space of 20 and 5")
}

func TestWordSpacingAddsToLetterSpacingAsDrawn(t *testing.T) {
	_, xs := places(spaced("ab cd", 2, nil))
	assert.Equal(t, []float32{0, 41}, xs, "two letters of 12, a space of 12 and 5")
}

func TestAPercentageWordSpacingIsDrawnAtTheTextsFontSize(t *testing.T) {
	_, xs := places(spaced("a b", 0, func(st *style.Computed) { st.WordSpacing = style.WordSpacing{Percent: 50} }))
	assert.Equal(t, []float32{0, 25}, xs)
}

func TestATextShadowIsWordSpacedAsItsText(t *testing.T) {
	ops := spaced("a b", 0, func(st *style.Computed) {
		st.TextShadows = []style.Shadow{{X: 1, Y: 1, Color: color.RGBA(0, 0, 0, 1)}}
	})
	_, xs := places(ops)
	assert.Equal(t, []float32{1, 26, 0, 25}, xs, "the shadow's words, then the text's")
}

func TestSelectedWordSpacedTextIsDrawnSpacedInBothColours(t *testing.T) {
	p := el(0, 0, 400, 20, func(st *style.Computed) {
		st.Color = white
		st.Font.Size = 10
		st.WordSpacing = style.WordSpacing{Px: 5}
		st.HasSelectionText, st.SelectionText = true, red
	})
	run := text(0, 0, "ab cd", p.Style)
	p.Append(run)
	r := newRecorder()
	f := Frame{Selection: chosen{parts: map[*box.Box][]Selected{run: {{Rect: geom.Rect{X: 35, W: 20, H: 20}}}}}}
	Paint(r, tree(p), Fonts(ahem{}), Each(&f))
	var inks []color.Color
	var xs []float32
	for _, o := range r.of("text") {
		inks, xs = append(inks, o.Color), append(xs, o.At.X)
	}
	require.Len(t, xs, 4)
	assert.Equal(t, []float32{0, 35, 0, 35}, xs)
	assert.Equal(t, []color.Color{white, white, red, red}, inks)
}

func TestWordSpacedTextIsMeasuredByItsFaceWithoutFonts(t *testing.T) {
	p := el(0, 0, 400, 20, func(st *style.Computed) { st.Color = white; st.WordSpacing = style.WordSpacing{Px: 5} })
	p.Append(text(0, 0, "a b", p.Style))
	plain := el(0, 0, 400, 20, func(st *style.Computed) { st.Color = white; st.LetterSpacing = 5 })
	plain.Append(text(0, 0, "a b", plain.Style))
	r := newRecorder()
	Paint(r, tree(el(0, 0, 400, 40, nil, p, plain)))
	ts := r.of("text")
	require.Len(t, ts, 3)
	assert.Equal(t, []string{"a", "b", "a b"}, []string{ts[0].Text, ts[1].Text, ts[2].Text})
	assert.Greater(t, ts[1].At.X, float32(5), "the letter, the space and 5 more")
}
