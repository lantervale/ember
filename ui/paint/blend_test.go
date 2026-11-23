package paint //nolint:testpackage // white-box: builds box trees by hand and reads the painter's own state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/style"
)

func mixed(m style.BlendMode, edit func(s *style.Computed)) func(s *style.Computed) {
	return func(s *style.Computed) {
		s.Background = red
		s.BlendMode = m
		if edit != nil {
			edit(s)
		}
	}
}

func TestAMixedElementIsAPictureMixedWithWhatItsGroupDrewUnderIt(t *testing.T) {
	r := newPicturing()
	Paint(r, tree(el(0, 0, 400, 300, bg(blue),
		el(10, 10, 100, 100, mixed(style.BlendMultiply, nil)),
		el(10, 120, 100, 100, bg(blue)))))
	assert.Equal(t,
		"begin, rect, rect, end 0, begin, show(), begin, rect, end 0, mix(1), end 0, show()",
		r.steps(), "the page is the group, split where the element mixes, a stacking context after the flow")
}

func TestAnIsolatedElementIsTheGroupItsChildrenMixWithin(t *testing.T) {
	r := newPicturing()
	Paint(r, tree(el(0, 0, 400, 300, bg(blue),
		el(10, 10, 200, 200, func(s *style.Computed) { s.Isolate, s.Background = true, blue },
			el(10, 10, 100, 100, mixed(style.BlendScreen, nil))))))
	assert.Equal(t,
		"rect, begin, rect, end 0, begin, show(), begin, rect, end 0, mix(2), end 0, show()",
		r.steps(), "the page drawn straight, the isolated element a picture its child splits")
}

func TestATranslucentGroupIsFadedOnceShown(t *testing.T) {
	r := newPicturing()
	Paint(r, tree(el(0, 0, 400, 300, nil,
		el(10, 10, 200, 200, func(s *style.Computed) {
			s.Opacity, s.Filters = 0.5, []style.Filter{{Kind: style.Grayscale, Amount: 1}}
		}, el(10, 10, 20, 20, bg(red))))))
	assert.Equal(t, "begin, begin, rect, end 0, show(matrix), end 0, show(matrix)", r.steps(),
		"the filtered picture in one faded as a whole")
	require.Len(t, r.effects, 2)
	a, _ := r.effects[1].Uniform("uA")
	assert.Equal(t, geom.Vec4{W: 0.5}, a)
	assert.Equal(t, 1, int(r.ops[2].Color.A), "the content drawn opaque inside it")
}

func TestABackdropFilterShowsTheBackdropFilteredInTheElementsRoundedBox(t *testing.T) {
	r := newPicturing()
	Paint(r, tree(el(0, 0, 400, 300, bg(blue),
		el(20, 30, 100, 80, func(s *style.Computed) {
			s.BackdropFilters = []style.Filter{{Kind: style.Blur, Amount: 2}, {Kind: style.Invert, Amount: 1}}
			pct := style.Length{Kind: style.Fixed, Px: 10}
			s.Radius.TopLeft = style.Size[style.Length]{Width: pct, Height: pct}
		}))))
	assert.Equal(t, "begin, rect, end 0, begin, show(), mask 20, show(mirrored mirrored matrix), pop, "+
		"end 0, show()", r.steps())
	require.Len(t, r.effects, 3)
	sigma, _ := r.effects[0].Uniform("uSigma")
	assert.InDelta(t, 2, sigma, 1e-6)
}

func TestPaintGroupsFollowWhereTheyMixAndFilter(t *testing.T) {
	isolate := func(s *style.Computed) { s.Isolate = true }
	child := el(0, 0, 10, 10, mixed(style.BlendDarken, nil))
	iso := el(0, 0, 50, 50, isolate, child)
	root := el(0, 0, 100, 100, nil, iso)
	tr := tree(root)
	inked(tr)
	assert.True(t, iso.Group, "the isolated parent")
	assert.False(t, root.Group)
	assert.False(t, tr.Grouped)

	back := el(0, 0, 10, 10, func(s *style.Computed) { s.BackdropFilters = []style.Filter{{Kind: style.Blur}} })
	positioned := el(0, 0, 50, 50, func(s *style.Computed) { s.Position, s.ZAuto = style.Relative, false }, back)
	tr = tree(el(0, 0, 100, 100, nil, positioned))
	inked(tr)
	assert.False(t, positioned.Group, "a stacking context is no backdrop root")
	assert.True(t, tr.Grouped, "the backdrop reaches the page")
}

func TestABackgroundsMixedLayersAreDrawnAsOnePicture(t *testing.T) {
	r := newPicturing()
	Paint(r, tree(el(0, 0, 400, 300, nil, el(10, 10, 100, 100, func(s *style.Computed) {
		s.Background = blue
		s.BackgroundImages = []style.Image{{Gradient: &style.Gradient{}}, {Gradient: &style.Gradient{}}}
		s.BackgroundBlends = []style.BlendMode{style.BlendNormal, style.BlendScreen}
	}))))
	assert.Equal(t, "push 10, begin, rect, end 0, begin, show(), begin, end 0, mix(2), end 0, show(), pop", r.steps(),
		"the colour, the bottom layer screened over it, the top one drawn straight, inside the box's clip")
}
