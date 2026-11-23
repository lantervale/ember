package paint

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/fx"
	"github.com/lantervale/ember/ui/internal/style"
)

func graphs(byID map[string]*fx.Graph) Option {
	return Filters(func(id string, _ geom.Vec2) *fx.Graph { return byID[id] })
}

func TestAFilterElementRunsPrimitiveByPrimitiveOnTheElementsPicture(t *testing.T) {
	region := geom.Rect{X: -10, Y: -10, W: 120, H: 120}
	g := &fx.Graph{Region: region, Primitives: []fx.Primitive{
		{Kind: fx.Offset, In: []fx.Input{fx.SourceGraphic}, Region: region, Offset: geom.V2(4, 2)},
		{Kind: fx.Composite, In: []fx.Input{fx.SourceGraphic, 0}, Region: region, Operator: fx.Xor},
	}}
	r := newPicturing()
	Paint(r, tree(el(0, 0, 400, 300, nil, el(10, 10, 100, 100, filtered(style.Filter{Kind: style.FilterURL,
		Ref: "g"})))), graphs(map[string]*fx.Graph{"g": g}), Scale(2))
	steps := r.steps()
	assert.Equal(t, "begin, rect, end 10, begin, show(), end 0, begin, push 0, show(?), pop, end 0, begin, show(), "+
		"end 0, begin, push 0, cut out 3, show(), pop, cut out 1, sum 3, pop, pop, end 0, begin, push 0, show(), pop, "+
		"end 0, show()", steps,
		"the pictures cut by first shown plain, each result cut to its region, the last to the filter's")
	at, _ := r.effects[0].Uniform("uOffset")
	assert.Equal(t, geom.V2(8, 4), at, "in pixels")
}

func TestAFilterNamingNoElementLeavesTheListOut(t *testing.T) {
	r := newPicturing()
	Paint(r, tree(el(0, 0, 400, 300, nil, el(10, 10, 100, 100, filtered(style.Filter{Kind: style.Blur, Amount: 2},
		style.Filter{Kind: style.FilterURL, Ref: "none"})))), graphs(nil))
	require.NotContains(t, r.steps(), "begin", "drawn unfiltered, as Chrome draws it")
}
