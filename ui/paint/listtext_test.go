package paint //nolint:testpackage // white-box: builds box trees by hand

import (
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/ui/control"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func widelySpaced(st *style.Computed) {
	st.Font.Size = 10
	st.LetterSpacing = style.Spacing{Px: 1}
	st.WordSpacing = style.Spacing{Px: 5}
	st.ExactText = true
}

func TestASelectsListSpacesAndSmoothsItsOptionsAsTheSelect(t *testing.T) {
	tr, sel, m, find := selectPage(20, 50, control.Option{Label: "ab cdefghijklm"})
	widelySpaced(sel.Style)
	m.Do(control.Activate)
	r := newRecorder()
	Paint(r, tr, find, Fonts(ahem{}))
	bg := listBg(r)
	assert.InDelta(t, 140+14*1+5+2*optionPad+2, bg.Rect.W, 1e-3,
		"as wide as its widest option with its letters and words spaced")
	texts := r.of("text")
	require.Len(t, texts, 2, "word by word, as word-spacing widens the space between")
	assert.Equal(t, []string{"ab", "cdefghijklm"}, []string{texts[0].Text, texts[1].Text})
	assert.InDelta(t, texts[0].At.X+3*(10+1)+5, texts[1].At.X, 1e-3, "after 'ab ' and the word gap")
	for _, o := range texts {
		assert.True(t, o.Exact, "-webkit-font-smoothing: antialiased")
	}
}

// hanAhem is Ahem with a halt feature: full-width brackets set half an
// em wide, an opening one moved half an em back.
type hanAhem struct{ ahem }

func (hanAhem) HanGlyph(f style.Font, r rune, _ bool) box.HanGlyph {
	g := box.HanGlyph{Halt: true, Full: true}
	switch r {
	case '「':
		g.Shift, g.Narrow = -f.Size/2, -f.Size/2
	case '」':
		g.Narrow, g.Ink = -f.Size/2, box.InkLeft
	}
	return g
}

func TestASelectsListTrimsAndAutospacesItsOptionsAsTheSelect(t *testing.T) {
	tr, sel, m, find := selectPage(20, 50, control.Option{Label: "漢」「漢A"})
	sel.Style.Font.Size, sel.Style.Autospace = 10, true
	m.Do(control.Activate)
	r := newRecorder()
	Paint(r, tr, find, Fonts(hanAhem{}))
	bg := listBg(r)
	// this Ahem draws a character as wide as its bytes: an ideograph or a bracket 30
	assert.InDelta(t, 130-5+1.25+2*optionPad+2, bg.Rect.W, 1e-3,
		"as wide as its option, the opening bracket half wide and an eighth of an em before the letter")
	require.Equal(t, []string{"漢」", "「", "漢", "A"}, textsOf(r))
	at := r.of("text")
	assert.InDelta(t, at[0].At.X+60-5, at[1].At.X, 1e-3, "the bracket moved back")
	assert.InDelta(t, at[0].At.X+85, at[2].At.X, 1e-3)
	assert.InDelta(t, at[0].At.X+116.25, at[3].At.X, 1e-3, "an eighth of an em before the letter")
}

func TestAltTextSpacesAndSmoothsAsItsImage(t *testing.T) {
	b := ctl(dom.Img, 300, 50, "src", "missing", "alt", "ab cd")
	widelySpaced(b.Style)
	r := newRecorder()
	Paint(r, tree(el(0, 0, 400, 300, nil, b)), Images(func(string) *twod.Texture { return nil }), Fonts(ahem{}))
	texts := r.of("text")
	require.Len(t, texts, 2)
	assert.Equal(t, geom.V2(10+BrokenIcon, 10), texts[0].At)
	assert.InDelta(t, texts[0].At.X+3*(10+1)+5, texts[1].At.X, 1e-3, "after 'ab ' and the word gap")
	for _, o := range texts {
		assert.True(t, o.Exact)
	}

	narrow := ctl(dom.Img, BrokenIcon+55, 50, "src", "missing", "alt", "ab cd")
	widelySpaced(narrow.Style)
	r = newRecorder()
	Paint(r, tree(el(0, 0, 400, 300, nil, narrow)), Images(func(string) *twod.Texture { return nil }), Fonts(ahem{}))
	texts = r.of("text")
	require.Len(t, texts, 2, "'ab cd' fits 55 unspaced, not spaced: it wraps")
	assert.Equal(t, float32(10), texts[1].At.X, "the second word under the icon")
}

func textsOf(r *recorder) []string {
	var out []string
	for _, o := range r.of("text") {
		out = append(out, o.Text)
	}
	return out
}

func TestASelectsListDrawsItsOptionsInSmallCapitals(t *testing.T) {
	tr, sel, m, find := selectPage(20, 50, control.Option{Label: "Ab"})
	sel.Style.Font.Caps = style.SmallCaps
	m.Do(control.Activate)
	r := newRecorder()
	Paint(r, tr, find, Fonts(ahem{}))
	texts := textsOf(r)
	require.GreaterOrEqual(t, len(texts), 2)
	assert.Equal(t, []string{"A", "B"}, texts[len(texts)-2:], "the lowercase as small capitals")
}

func TestAltTextIsDrawnInSmallCapitals(t *testing.T) {
	b := ctl(dom.Img, 300, 50, "src", "missing", "alt", "Ab")
	b.Style.Font.Caps = style.SmallCaps
	r := newRecorder()
	Paint(r, tree(el(0, 0, 400, 300, nil, b)), Images(func(string) *twod.Texture { return nil }), Fonts(ahem{}))
	assert.Equal(t, []string{"A", "B"}, textsOf(r))
}
