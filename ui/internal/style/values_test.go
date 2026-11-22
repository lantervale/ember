package style_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/ui/internal/style"
)

func TestResolveAddsPxAndPercent(t *testing.T) {
	v, ok := style.Length{Kind: style.Fixed, Px: -20, Percent: 100}.Resolve(300)
	assert.True(t, ok)
	assert.Equal(t, float32(280), v)
	_, ok = style.Length{}.Resolve(300)
	assert.False(t, ok, "auto")
}

func TestInitialIsCSSs(t *testing.T) {
	c := style.Initial()
	assert.Equal(t, style.Inline, c.Display)
	assert.Equal(t, style.BorderBox, c.BoxSizing)
	assert.Equal(t, style.Auto, c.Size.Width.Kind)
	assert.Equal(t, style.None, c.MaxSize.Width.Kind)
	assert.Equal(t, float32(1), c.Shrink)
	assert.True(t, c.LineHeight.Normal)
	assert.Equal(t, float32(1), c.Opacity)
	assert.True(t, c.ZAuto)
	assert.Equal(t, style.Static, c.Position)
}

func TestMixIsASumOfTheTwoScaled(t *testing.T) {
	m := style.Px(100).Mix(style.Percent(50), 0.35)
	c := m.Calc.Value()
	for ref, want := range map[float32]float32{0: 65, 200: 65 + 35} {
		assert.InDelta(t, want, c.At(ref), 1e-4, "against %v", ref)
	}
	assert.True(t, m.IsPercent)
}
