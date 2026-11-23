package paint //nolint:testpackage // white-box: builds box trees by hand and reads the painter's own state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

// chosen is a selection of given parts of runs and of whole elements.
type chosen struct {
	parts map[*box.Box][]Selected
	holds *dom.Node
}

func (c chosen) Parts(run *box.Box) []Selected { return c.parts[run] }
func (c chosen) Holds(n *dom.Node) bool        { return n != nil && n == c.holds }

func selected(root *box.Box, c chosen) *recorder {
	r := newRecorder()
	f := Frame{Selection: c}
	Paint(r, tree(root), Each(&f), Images(func(string) *twod.Texture { return arrow }))
	return r
}

func TestSelectedTextDrawsOverChromesHighlightInItsOwnColour(t *testing.T) {
	p := el(5, 5, 100, 20, func(s *style.Computed) { s.Color = red })
	run := text(2, 0, "hello", p.Style)
	p.Append(run)
	r := selected(p, chosen{parts: map[*box.Box][]Selected{run: {{Rect: geom.Rect{X: 8, W: 16, H: 20}}}}})
	rects := r.of("rect")
	require.Len(t, rects, 1)
	assert.Equal(t, geom.Rect{X: 15, Y: 5, W: 16, H: 20}, rects[0].Rect, "the part, from the run's corner")
	assert.Equal(t, color.Hex("#80bcfd99"), rects[0].Color, "Chrome's highlight, #b3d7fe over white")
	ts := r.of("text")
	require.Len(t, ts, 1, "the text drawn once, over it")
	assert.Equal(t, red, ts[0].Color)

	assert.Empty(t, selected(p, chosen{}).of("rect"), "no part, no highlight")
}

func TestTheSystemsHighlightIsBlendedWithWhiteAsChromeOnMacOSBlendsIt(t *testing.T) {
	for in, want := range map[string]string{
		"#b3d7fe":   "#80bcfd99",
		"#dcdcdc":   "#c4c4c499",
		"#ffffff":   "#fefefe99",
		"#0078d7":   "#0056cdcc",
		"#3390ff80": "#3390ff80",
	} {
		assert.Equal(t, color.Hex(want), blendWithWhite(color.Hex(in)), in)
	}
}

func TestASelectionWithNoRuleTakesTheSystemsHighlight(t *testing.T) {
	p := el(0, 0, 100, 20, func(s *style.Computed) { s.Color = red })
	run := text(0, 0, "hello", p.Style)
	p.Append(run)
	paint := func(f Frame) *recorder {
		r := newRecorder()
		f.Selection = chosen{parts: map[*box.Box][]Selected{run: {{Rect: geom.Rect{W: 44, H: 20}}}}}
		Paint(r, tree(p), Each(&f))
		return r
	}
	sys := Highlight{Background: color.Hex("#0078d7"), Inactive: color.Hex("#f0f0f0"), Text: color.White}
	r := paint(Frame{Highlight: sys})
	require.Len(t, r.of("rect"), 1)
	assert.Equal(t, color.Hex("#0056cdcc"), r.of("rect")[0].Color)
	assert.Equal(t, color.White, r.of("text")[0].Color, "the system's text colour over it")

	r = paint(Frame{Highlight: sys, Inactive: true})
	assert.Equal(t, color.Hex("#e5e5e599"), r.of("rect")[0].Color, "its inactive one in an inactive window")
	assert.Equal(t, color.White, r.of("text")[0].Color)

	r = paint(Frame{Highlight: Highlight{Background: color.Hex("#ffd0a0")}, Inactive: true})
	assert.Equal(t, color.Hex("#c4c4c499"), r.of("rect")[0].Color, "Chrome's grey where the system says none")
	assert.Equal(t, red, r.of("text")[0].Color, "and the text keeps its colour where it says none")
}

func TestASelectionRuleSetsTheHighlightAndTheTextsColour(t *testing.T) {
	p := el(0, 0, 100, 20, func(s *style.Computed) {
		s.Color = red
		s.SelectionColor, s.HasSelection = green, true
		s.SelectionText, s.HasSelectionText = blue, true
	})
	run := text(0, 0, "hello", p.Style)
	p.Append(run)
	r := selected(p, chosen{parts: map[*box.Box][]Selected{run: {{Rect: geom.Rect{X: 8, W: 16, H: 20}}}}})
	rects := r.of("rect")
	require.Len(t, rects, 1)
	assert.Equal(t, green, rects[0].Color)
	ts := r.of("text")
	require.Len(t, ts, 3, "before, inside and after the part")
	assert.Equal(t, []color.Color{red, blue, red}, []color.Color{ts[0].Color, ts[1].Color, ts[2].Color})
	assert.InDelta(t, 8, ts[1].Clip.X, 1e-4, "each clipped to its side of the part")
	assert.InDelta(t, 16, ts[1].Clip.W, 1e-4)
	assert.InDelta(t, 24, ts[2].Clip.X, 1e-4)

	r = selected(p, chosen{parts: map[*box.Box][]Selected{run: {{Rect: geom.Rect{W: 44, H: 20}}}}})
	ts = r.of("text")
	require.Len(t, ts, 1, "a fragment wholly selected is drawn once")
	assert.Equal(t, blue, ts[0].Color)
	assert.Less(t, ts[0].Clip.X, float32(0), "with what its glyphs draw outside it")

	only := el(0, 0, 100, 20, func(s *style.Computed) { s.SelectionText, s.HasSelectionText = blue, true })
	run = text(0, 0, "hello", only.Style)
	only.Append(run)
	r = selected(only, chosen{parts: map[*box.Box][]Selected{run: {{Rect: geom.Rect{W: 40, H: 20}}}}})
	assert.Empty(t, r.of("rect"), "a rule with a colour alone has no background, as Chrome")
}

func TestAPictureInsideASelectionIsTinted(t *testing.T) {
	img := ctl(dom.Img, 100, 50, "src", "arrow")
	r := selected(el(0, 0, 400, 300, nil, img), chosen{holds: img.Node})
	rects := r.of("rect")
	require.Len(t, rects, 1)
	assert.Equal(t, geom.Rect{X: 10, Y: 10, W: 100, H: 50}, rects[0].Rect)
	assert.Equal(t, pageSelection, rects[0].Color)
	assert.Empty(t, selected(el(0, 0, 400, 300, nil, img), chosen{}).of("rect"))
}
