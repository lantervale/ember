package paint //nolint:testpackage // white-box: builds box trees by hand and reads the painter's own state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/control"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

func TestAnInactiveWindowGreysTheDefaultHighlightOnly(t *testing.T) {
	paint := func(st func(*style.Computed)) []op {
		p := el(0, 0, 100, 20, st)
		run := text(0, 0, "hello", p.Style)
		p.Append(run)
		r := newRecorder()
		f := Frame{Selection: chosen{parts: map[*box.Box][]Selected{run: {{Rect: geom.Rect{W: 16, H: 20}}}}}, Inactive: true}
		Paint(r, tree(p), Each(&f))
		return r.of("rect")
	}
	rects := paint(nil)
	require.Len(t, rects, 1)
	assert.Equal(t, color.Hex("#c4c4c499"), rects[0].Color, "macOS's #dcdcdc, made translucent as Chrome makes it")

	rects = paint(func(s *style.Computed) { s.SelectionColor, s.HasSelection = green, true })
	require.Len(t, rects, 1)
	assert.Equal(t, green, rects[0].Color, "a ::selection's own colour stays")
}

func TestASelectionsShadowsReplaceTheTextsOwnInItsPart(t *testing.T) {
	p := el(0, 0, 100, 20, func(s *style.Computed) {
		s.Color = red
		s.TextShadows = []style.Shadow{{X: 1, Y: 1, Color: green}}
		s.SelectionShadows, s.HasSelectionShadows = []style.Shadow{{X: 2, Y: 2, Color: blue}}, true
	})
	run := text(0, 0, "hello", p.Style)
	p.Append(run)
	r := selected(p, chosen{parts: map[*box.Box][]Selected{run: {{Rect: geom.Rect{X: 8, W: 16, H: 20}}}}})
	var shadows []op
	for _, o := range r.of("text") {
		if o.Color != red {
			shadows = append(shadows, o)
		}
	}
	require.Len(t, shadows, 3, "before, inside and after the part")
	assert.Equal(t, []color.Color{green, blue, green}, []color.Color{shadows[0].Color, shadows[1].Color, shadows[2].Color})
	assert.InDelta(t, 2, shadows[1].At.X, 1e-4, "the selection's offset")
	assert.InDelta(t, 8, shadows[1].Clip.X, 1e-4, "clipped to the part")
	assert.InDelta(t, 16, shadows[1].Clip.W, 1e-4)
}

func TestASelectionsDecorationIsDrawnOnItsPartOnly(t *testing.T) {
	p := el(0, 0, 100, 20, func(s *style.Computed) {
		s.Color = red
		s.SelectionDecoration = style.TextDecoration{Line: style.Underline, Color: blue}
		s.HasSelectionDecoration = true
	})
	run := text(0, 0, "hello", p.Style)
	p.Append(run)
	r := selected(p, chosen{parts: map[*box.Box][]Selected{run: {{Rect: geom.Rect{X: 8, W: 16, H: 20}}}}})
	var lines []op
	for _, o := range r.of("rect") {
		if o.Color == blue {
			lines = append(lines, o)
		}
	}
	require.Len(t, lines, 1)
	assert.Greater(t, lines[0].Rect.Y, float32(15), "under the baseline")
	assert.InDelta(t, 8, lines[0].Clip.X, 1e-4, "clipped to the part")
	assert.InDelta(t, 16, lines[0].Clip.W, 1e-4)
	assert.Empty(t, selected(p, chosen{}).of("rect"), "nothing selected, no line")
}

func TestABreakAloneOnItsLineDrawsItsMark(t *testing.T) {
	br := ctl(dom.Br, 0, 20)
	r := selected(el(0, 0, 400, 300, nil, br),
		chosen{parts: map[*box.Box][]Selected{br: {{Rect: geom.Rect{Y: -3, W: 5, H: 26}}}}})
	rects := r.of("rect")
	require.Len(t, rects, 1)
	assert.Equal(t, geom.Rect{X: 10, Y: 7, W: 5, H: 26}, rects[0].Rect)
	assert.Equal(t, pageSelection, rects[0].Color)
}

func TestAFieldsSelectedTextTakesItsSelectionColour(t *testing.T) {
	b := ctl(dom.Input, 100, 20, "type", "text")
	b.Style.SelectionText, b.Style.HasSelectionText = red, true
	b.Append(text(0, 0, "hello", b.Style))
	m := control.NewText("hello")
	m.Layout = line{5}
	m.Editor.Select(1, 3)
	b.Node.SetState(dom.Focus, true)
	texts := paintOne(b, Controls(func(*dom.Node) any { return m })).of("text")
	require.Len(t, texts, 3, "before, inside and after the selection")
	assert.Equal(t, []color.Color{black, red, black}, []color.Color{texts[0].Color, texts[1].Color, texts[2].Color})
	assert.InDelta(t, 10+8, texts[1].Clip.X, 1e-4)
	assert.InDelta(t, 16, texts[1].Clip.W, 1e-4)
}
