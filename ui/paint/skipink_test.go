package paint //nolint:testpackage // white-box: builds box trees by hand and reads the painter's own state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/cull"
	"github.com/lantervale/ember/ui/internal/style"
)

// inky is ahem whose first glyph's ink crosses every band from 3 to 5.
type inky struct{ ahem }

func (inky) Intercepts(style.Font, string, float32, float32, float32, float32, bool) [][2]float32 {
	return [][2]float32{{3, 5}}
}

func TestAnUnderlineSkipsTheInkItCrosses(t *testing.T) {
	r := newRecorder()
	p := el(0, 0, 100, 20, func(s *style.Computed) {
		s.Color = white
		s.Decorations = []style.Decoration{decoration(style.Underline, red)}
	})
	p.Append(text(0, 0, "hi", p.Style))
	Paint(r, tree(p), Fonts(inky{}))
	pushes := r.of("push")
	require.Len(t, pushes, 2, "the line drawn twice, either side of the glyph")
	assert.InDelta(t, 3-1.6, pushes[0].Rect.X+pushes[0].Rect.W, 1e-4, "the gap dilated by the line's thickness")
	assert.InDelta(t, 5+1.6, pushes[1].Rect.X, 1e-4)
	assert.Len(t, r.of("rect"), 2)
}

func TestSkipInkNoneDrawsTheLineThrough(t *testing.T) {
	r := newRecorder()
	p := el(0, 0, 100, 20, func(s *style.Computed) {
		s.Color = white
		s.SkipInk = style.SkipInkNone
		s.Decorations = []style.Decoration{decoration(style.Underline|style.LineThrough, red)}
	})
	p.Append(text(0, 0, "hi", p.Style))
	Paint(r, tree(p), Fonts(inky{}))
	assert.Empty(t, r.of("push"))
	assert.Len(t, r.of("rect"), 2)
}

func TestADecorationCastsTheTextsShadow(t *testing.T) {
	r := newRecorder()
	p := el(0, 0, 100, 20, func(s *style.Computed) {
		s.Color = white
		s.TextShadows = []style.Shadow{{X: 2, Y: 3, Color: blue}}
		s.Decorations = []style.Decoration{decoration(style.Underline|style.LineThrough, red)}
	})
	p.Append(text(0, 0, "hi", p.Style))
	Paint(r, tree(p), Fonts(ahem{}))
	var kinds []string
	for _, o := range r.of("rect", "text") {
		kinds = append(kinds, o.Kind)
	}
	assert.Equal(t, []string{"rect", "rect", "text", "text", "rect", "rect"}, kinds,
		"the underline's shadow and the underline, the text's, the line through's")
	ops := r.of("rect")
	assert.Equal(t, blue, ops[0].Color)
	assert.Equal(t, geom.Rect{X: 2, Y: 19, W: 16, H: 1}, ops[0].Rect)
	assert.Equal(t, red, ops[1].Color)
	assert.Equal(t, blue, ops[2].Color)
}

// inkyAll is inky that records whether each call asks for every glyph's ink.
type inkyAll struct {
	inky
	all *[]bool
}

func (i inkyAll) Intercepts(f style.Font, s string, letter, word, top, bottom float32, all bool) [][2]float32 {
	*i.all = append(*i.all, all)
	return i.inky.Intercepts(f, s, letter, word, top, bottom, all)
}

func TestSkipInkAllAsksForEveryGlyphsInkAndLeavesLinesThroughWhole(t *testing.T) {
	defer cull.Set(cull.On, nil)()
	for _, c := range []struct {
		skip style.SkipInk
		all  bool
	}{{style.SkipInkAuto, false}, {style.SkipInkAll, true}} {
		r := newRecorder()
		p := el(0, 0, 100, 20, func(s *style.Computed) {
			s.Color = white
			s.SkipInk = c.skip
			s.Decorations = []style.Decoration{decoration(style.Underline|style.Overline|style.LineThrough, red)}
		})
		p.Append(text(0, 0, "hi", p.Style))
		var asked []bool
		Paint(r, tree(p), Fonts(inkyAll{all: &asked}))
		assert.Equal(t, []bool{c.all, c.all}, asked, "the underline and the overline")
		assert.Len(t, r.of("push"), 4, "each cut round the glyph; the line through whole")
	}
}
