package paint

import (
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/style"
)

// emFonts draws every character an em wide, upright or turned.
type emFonts struct{}

func (emFonts) Metrics(f style.Font) box.Metrics { return box.Metrics{Ascent: f.Size} }

func (emFonts) Advance(f style.Font, text string) float32 {
	return f.Size * float32(utf8.RuneCountInString(text))
}

func TestAVerticalLinesPartsTakeTheirSpacing(t *testing.T) {
	st := style.Initial()
	st.Mode, st.Font.Size = style.VerticalRL, 20
	p := &painter{fonts: emFonts{}}
	f := box.Fragment{Text: "漢」「AB漢", Rect: geom.Rect{W: 115, H: 20},
		Spacing: []box.Spacing{{At: 6, Shift: -10, Space: -10}, {At: 9, Space: 2.5}, {At: 10, Space: 2.5}}}
	parts := p.orientedParts(f, &st)
	require.Len(t, parts, 5)
	var at []geom.Rect
	for _, part := range parts {
		at = append(at, part.Rect)
	}
	assert.Equal(t, []geom.Rect{{W: 20, H: 20}, {X: 20, W: 20, H: 20}, {X: 30, W: 20, H: 20},
		{X: 50, W: 45, H: 20}, {X: 95, W: 20, H: 20}}, at,
		"the trimmed bracket drawn half an em up its room, the turned letters as long as their room")
	assert.Equal(t, []box.Spacing{{At: 0, Space: 2.5}, {At: 1, Space: 2.5}}, parts[3].Spacing)
	assert.Nil(t, parts[2].Spacing)
}
