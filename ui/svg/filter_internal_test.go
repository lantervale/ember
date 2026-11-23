package svg

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/fx"
)

func TestAFilterElementIsHandedToUIInItsOwnModel(t *testing.T) {
	v, err := parse([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="0" height="0"><filter id="f">
<feFlood flood-color="#ff0000" flood-opacity="0.5"/><feComposite in2="SourceAlpha" operator="in"/>
<feComponentTransfer><feFuncA type="table" tableValues="0 1"/></feComponentTransfer></filter></svg>`))
	require.NoError(t, err)
	pic, isPicture := v.(picture)
	require.True(t, isPicture)
	g, ok := pic.FilterGraph("f", 50, 20)
	require.True(t, ok)
	assert.Equal(t, geom.Rect{X: -5, Y: -2, W: 60, H: 24}, g.Region)
	require.Len(t, g.Primitives, 3)
	flood := g.Primitives[0]
	assert.Equal(t, fx.Flood, flood.Kind)
	assert.InDelta(t, 1, flood.Color.R, 1e-6)
	assert.InDelta(t, 0.5, flood.Color.A, 0.01)
	assert.Equal(t, []fx.Input{0, fx.SourceAlpha}, g.Primitives[1].In)
	assert.Equal(t, fx.In, g.Primitives[1].Operator)
	assert.Equal(t, []float32{0, 1}, g.Primitives[2].Transfer[3].Table)
	_, ok = pic.FilterGraph("g", 50, 20)
	assert.False(t, ok)
}
