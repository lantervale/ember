package style_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/ui/internal/style"
)

func TestABackgroundHasALayerAPicture(t *testing.T) {
	var l style.Look
	assert.Equal(t, 1, l.BackgroundLayers(), "none is one layer, under the colour")
	assert.False(t, l.HasBackgroundImage())
	l.BackgroundImages = []style.Image{{Source: "a.png"}, {}, {Source: "c.png"}}
	assert.Equal(t, 3, l.BackgroundLayers())
	assert.True(t, l.HasBackgroundImage())
}

func TestALayerTakesItsLonghandsRepeated(t *testing.T) {
	l := style.Look{
		BackgroundImages:    []style.Image{{Source: "a.png"}, {Source: "b.png"}, {Source: "c.png"}},
		BackgroundPositions: []style.BackgroundPosition{{X: style.Px(1)}, {X: style.Px(2)}},
		BackgroundRepeats:   []style.BackgroundRepeat{{X: style.TileNoRepeat, Y: style.TileSpace}},
		BackgroundClips: []style.BackgroundBox{
			style.UnderPadding, style.UnderContent, style.UnderBorder, style.UnderPadding,
		},
		BackgroundAttachments: []style.BackgroundAttachment{style.AttachFixed, style.AttachLocal},
	}
	third := l.BackgroundLayer(2)
	assert.Equal(t, "c.png", third.Image.Source)
	assert.Equal(t, style.Px(1), third.Position.X, "a list shorter than the pictures starts again")
	assert.Equal(t, style.BackgroundRepeat{X: style.TileNoRepeat, Y: style.TileSpace}, third.Repeat)
	assert.Equal(t, style.UnderBorder, third.Clip, "a longer one is cut")
	assert.Equal(t, style.AttachFixed, third.Attachment)
	assert.Equal(t, style.AttachLocal, l.BackgroundLayer(1).Attachment)
	assert.Equal(t, style.InitialBox, third.Origin, "no list is the initial value")
	assert.Equal(t, style.BackgroundSize{}, third.Size)
}

func TestAnUnsetPositionIsTheTopLeftCorner(t *testing.T) {
	var l style.Look
	p := l.BackgroundLayer(0).Position
	assert.True(t, p.X.IsPercent && p.Y.IsPercent, "0% 0%, as CSS computes it")
	assert.Zero(t, p.X.At(100))
}
