package paint //nolint:testpackage // white-box: builds box trees by hand and reads the painter's own state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/ui/internal/style"
)

func kinds(ops []op) []string {
	out := make([]string, 0, len(ops))
	for _, o := range ops {
		out = append(out, o.Kind)
	}
	return out
}

func TestABackgroundClippedToTextShowsThroughItsGlyphs(t *testing.T) {
	b := el(10, 10, 120, 40, func(s *style.Computed) {
		s.Background = blue
		s.Color = color.Color{}
		s.BackgroundClips = []style.BackgroundBox{style.UnderText}
	})
	inner := el(0, 20, 120, 20, func(s *style.Computed) { s.Color = red })
	b.Append(text(4, 2, "Vale", b.Style), inner)
	inner.Append(text(0, 0, "held", inner.Style))
	r := newPicturing()
	Paint(r, tree(el(0, 0, 400, 300, nil, b)))
	require.Equal(t, []string{"begin", "text", "text", "end", "cut", "rect", "pop", "text"}, kinds(r.ops))
	assert.Equal(t, color.White, r.ops[1].Color, "the glyphs opaque, though the text is clear")
	assert.Equal(t, "held", r.ops[2].Text, "and those of what it holds")
	assert.InDelta(t, 30+15, r.ops[2].At.Y, 0.01, "where they are")
	assert.Equal(t, "cut 1", r.ops[4].Text)
	assert.Equal(t, blue, r.ops[5].Color)
	assert.Equal(t, red, r.ops[7].Color, "the child's text drawn over in its colour")
}

func TestOnlyTheLayersClippedToTextAreCut(t *testing.T) {
	b := el(0, 0, 100, 30, func(s *style.Computed) {
		s.Background = blue
		s.Color = color.Color{}
		s.BackgroundImages = []style.Image{
			{Gradient: &style.Gradient{Stops: stops(red, red)}}, {Gradient: &style.Gradient{Stops: stops(red, red)}},
		}
		s.BackgroundClips = []style.BackgroundBox{style.UnderText, style.InitialBox}
	})
	b.Append(text(0, 0, "Layers", b.Style))
	r := newPicturing()
	Paint(r, tree(el(0, 0, 400, 300, nil, b)))
	assert.Equal(t, []string{"push", "begin", "text", "end", "rect", "gradient", "cut", "gradient", "pop", "pop"},
		kinds(r.ops), "the colour and the layer under, in the border box; the top layer in the text")
}

func TestWithoutPicturesWhatIsClippedToTextIsLeftOut(t *testing.T) {
	b := el(0, 0, 100, 30, func(s *style.Computed) {
		s.Background = blue
		s.BackgroundClips = []style.BackgroundBox{style.UnderText}
	})
	b.Append(text(0, 0, "Vale", b.Style))
	r := newRecorder()
	Paint(r, tree(el(0, 0, 400, 300, nil, b)))
	assert.Empty(t, r.of("rect"))
	assert.Len(t, r.of("text"), 1)
}
