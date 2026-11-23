package twod //nolint:testpackage // white-box: shares twod's test helpers, reaches unexported state

import (
	"slices"
	"testing"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestACanvasReplaysWhatItRecorded(t *testing.T) {
	w, _, _ := app()
	w.RunFrames(1)
	c := CanvasFor(w, nil)
	start := len(c.list.Vertices)
	m := c.Mark()
	c.Rect(geom.Rect{X: 10, Y: 10, W: 40, H: 20}, color.White)
	c.Text(geom.V2(12, 25), "Save", FontSize(15), Baseline())
	drawn := slices.Clone(c.list.Vertices[start:])
	var r Recording
	require.True(t, c.Keep(m, &r))
	n := len(c.list.Vertices)
	require.True(t, c.Replay(&r))
	assert.Equal(t, drawn, c.list.Vertices[n:])

	c.SetTransform(c.Transform().Mul(geom.Transform2D{X: geom.V2(2, 0), Y: geom.V2(0, 2)}))
	assert.False(t, c.Replay(&r), "not under another transform")
}

func TestACanvasReplaysAroundTheGapsItLeft(t *testing.T) {
	w, _, _ := app()
	w.RunFrames(1)
	c := CanvasFor(w, nil)
	start := len(c.list.Vertices)
	m := c.Mark()
	c.Rect(geom.Rect{X: 10, Y: 10, W: 40, H: 20}, color.White)
	from := c.Mark()
	c.Rect(geom.Rect{X: 20, Y: 20, W: 8, H: 8}, color.RGB(1, 0, 0))
	gap := Gap{From: from, To: c.Mark()}
	c.Text(geom.V2(12, 25), "Save", FontSize(15), Baseline())
	drawn := slices.Clone(c.list.Vertices[start:])
	var r Recording
	require.True(t, c.KeepAround(m, []Gap{gap}, &r))
	assert.False(t, c.Replay(&r), "a recording with gaps replays only around them")

	n := len(c.list.Vertices)
	require.True(t, c.ReplayAround(&r, func(i int) {
		assert.Equal(t, 0, i)
		c.Rect(geom.Rect{X: 20, Y: 20, W: 8, H: 8}, color.RGB(1, 0, 0))
	}))
	assert.Equal(t, drawn, c.list.Vertices[n:])

	require.True(t, c.Keep(m, &r))
	assert.False(t, c.ReplayAround(&r, func(int) {}), "nor one without them around")
}
