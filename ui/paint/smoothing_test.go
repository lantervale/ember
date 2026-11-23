package paint //nolint:testpackage // white-box: builds box trees by hand

import (
	"testing"

	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/style"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTextIsDrawnWithItsFontSmoothing(t *testing.T) {
	cases := []struct {
		name         string
		edit         func(*style.Computed)
		pixel, exact bool
	}{
		{"auto", func(*style.Computed) {}, false, false},
		{"none", func(st *style.Computed) { st.PixelText = true }, true, false},
		{"antialiased", func(st *style.Computed) { st.ExactText = true }, false, true},
	}
	for _, c := range cases {
		p := el(0, 0, 400, 40, func(st *style.Computed) {
			st.Color = white
			st.Font.Size = 10
			st.WordSpacing = style.Spacing{Px: 5}
			st.ListItem, st.ListStyle = true, style.ListDecimal
			st.TextShadows = []style.Shadow{{X: 1, Y: 1, Color: white}}
			c.edit(st)
		})
		p.Append(text(0, 0, "ab cd", p.Style), &box.Box{Kind: box.Marker, Style: p.Style, Size: p.Size})
		r := newRecorder()
		Paint(r, tree(p), Fonts(ahem{}))
		texts := r.of("text")
		require.NotEmpty(t, texts, c.name)
		for _, o := range texts {
			assert.Equal(t, c.pixel, o.Pixelated, "%s: %q", c.name, o.Text)
			assert.Equal(t, c.exact, o.Exact, "%s: %q", c.name, o.Text)
		}
	}
}
