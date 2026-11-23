package svg_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/svg"
)

func TestAFilterIsReadForAnElementOutsideThePicture(t *testing.T) {
	p, err := svg.NewPicture([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="0" height="0">
<filter id="soft"><feGaussianBlur stdDeviation="2"/><feOffset dx="3" result="moved"/>
<feMerge><feMergeNode in="moved"/><feMergeNode in="SourceGraphic"/></feMerge></filter>
<filter id="own" filterUnits="userSpaceOnUse" x="0" y="0" width="50%" height="20"
  primitiveUnits="objectBoundingBox"><feFlood flood-color="red" width="0.5"/></filter>
<rect id="r" width="5" height="5"/></svg>`))
	require.NoError(t, err)
	f, ok := p.Filter("soft", svg.Box{W: 100, H: 50})
	require.True(t, ok)
	assert.Equal(t, svg.Box{X: -10, Y: -5, W: 120, H: 60}, f.Region, "10% round the box")
	require.Len(t, f.Primitives, 3)
	assert.Equal(t, svg.GaussianBlur, f.Primitives[0].Kind)
	assert.Equal(t, []svg.Input{svg.Input(1), svg.SourceGraphic}, f.Primitives[2].In)

	f, ok = p.Filter("own", svg.Box{X: 0, Y: 0, W: 200, H: 40})
	require.True(t, ok)
	assert.Equal(t, svg.Box{W: 100, H: 20}, f.Region, "percentages of the element's box")
	assert.InDelta(t, 100, f.Primitives[0].Region.W, 1e-9, "half the box")

	_, ok = p.Filter("r", svg.Box{W: 10, H: 10})
	assert.False(t, ok, "no filter element")
	_, ok = p.Filter("none", svg.Box{W: 10, H: 10})
	assert.False(t, ok)
}
