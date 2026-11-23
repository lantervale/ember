package css //nolint:testpackage // reads computed styles off dom and the package's unexported test helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

func TestAnchorFunctionsGoOnlyWhereChromeTakesThem(t *testing.T) {
	for _, ok := range []struct{ name, value string }{
		{"top", "anchor(bottom)"}, {"left", "anchor(--a right, 10px)"}, {"right", "calc(anchor(50%) + 4px)"},
		{"bottom", "anchor(--a inside)"}, {"top", "anchor-size(height)"}, {"width", "anchor-size(width)"},
		{"max-height", "anchor-size(--a self-block, 100px)"}, {"margin-left", "anchor-size()"},
		{"inset", "anchor(top) auto"},
	} {
		assert.NoError(t, validate(ok.name, componentsOf(ok.value)), "%s: %s", ok.name, ok.value)
	}
	for _, bad := range []struct{ name, value string }{
		{"width", "anchor(left)"}, {"padding-left", "anchor-size(width)"}, {"margin-top", "anchor(top)"},
		{"top", "anchor()"}, {"top", "anchor(--a)"}, {"top", "anchor(--a --b top)"}, {"top", "anchor(middle)"},
		{"width", "anchor-size(left)"}, {"top", "anchor(top, red)"},
	} {
		assert.Error(t, validate(bad.name, componentsOf(bad.value)), "%s: %s", bad.name, bad.value)
	}
}

func TestAnchorPropertiesCompute(t *testing.T) {
	a, b, c := el(dom.Div, "a"), el(dom.Div, "b"), el(dom.Div, "c")
	Compute(page(a, b, c), []*Sheet{sheet(t, `
#a { anchor-name: --x, --y; anchor-scope: all; position-anchor: --x; position-area: span-left bottom;
  position-try: most-height flip-block, --t flip-inline flip-start, top; position-visibility: no-overflow }
#b { position: absolute; position-try-fallbacks: --t, --missing; anchor-scope: --q; position-anchor: auto }
#c { position-area: start; justify-self: anchor-center }
@position-try --t { top: anchor(bottom); width: 90px }`)})
	sa, sb, sc := styleOf(a).Anchoring, styleOf(b).Anchoring, styleOf(c).Anchoring
	assert.Equal(t, []string{"--x", "--y"}, sa.Names)
	assert.Equal(t, style.ScopeAll, sa.Scope)
	assert.Equal(t, style.AnchorNamed, sa.Anchor)
	assert.Equal(t, "--x", sa.AnchorName)
	assert.Equal(t, style.PositionArea{Set: true, X: style.AreaSpan{From: 0, To: 1}, Y: style.AreaSpan{From: 2, To: 2}},
		sa.Area)
	assert.Equal(t, style.TryMostHeight, sa.Order)
	require.Len(t, sa.Fallbacks, 3)
	assert.Equal(t, []style.Flip{style.FlipBlock}, sa.Fallbacks[0].Flips)
	assert.Equal(t, "--t", sa.Fallbacks[1].Name)
	assert.Equal(t, []style.Flip{style.FlipInline, style.FlipStart}, sa.Fallbacks[1].Flips)
	assert.True(t, sa.Fallbacks[2].Area.Set)
	assert.Equal(t, style.NoOverflow, sa.Visibility)
	assert.Empty(t, sa.Tries, "no tries for a box in the flow")

	assert.Equal(t, style.AnchorAuto, sb.Anchor)
	assert.Equal(t, []string{"--q"}, sb.ScopeNames)
	assert.Equal(t, style.AnchorsVisible, sb.Visibility, "anchors-visible by default")
	require.Len(t, sb.Tries, 1, "a fallback naming no rule is left out")
	try := sb.Tries[0]
	assert.True(t, try.Inset.Top.Anchored(), "the rule's insets")
	assert.Equal(t, style.Px(90), try.Size.Width)

	assert.Equal(t, style.AreaSpan{From: 0, To: 0}, sc.Area.X, "start alone is both axes")
	assert.Equal(t, style.AlignAnchorCenter, styleOf(c).JustifySelf)
}

func TestFlipsMoveDeclarationsAndTurnTheirAnchorSides(t *testing.T) {
	p := el(dom.Div, "p")
	Compute(page(p), []*Sheet{sheet(t, `#p { position: absolute; top: anchor(bottom); left: anchor(25%);
  margin-top: 4px; align-self: start; width: anchor-size(height);
  position-try-fallbacks: flip-block, flip-inline, flip-start }`)})
	tries := styleOf(p).Anchoring.Tries
	require.Len(t, tries, 3)
	side := func(l style.Length) style.AnchorFn {
		var fn style.AnchorFn
		c := l.Calc.Value()
		c.Anchors(func(f style.AnchorFn) { fn = f })
		return fn
	}
	block := tries[0]
	assert.Equal(t, style.Length{}, block.Inset.Top)
	assert.Equal(t, style.AnchorTop, side(block.Inset.Bottom).Side)
	assert.Equal(t, style.Px(4), block.Margin.Bottom)
	assert.Equal(t, style.AlignEnd, block.AlignSelf)

	inline := tries[1]
	assert.Equal(t, style.Length{}, inline.Inset.Left)
	assert.InDelta(t, 75, side(inline.Inset.Right).Percent, 0.001, "a percentage from the other end")

	start := tries[2]
	assert.Equal(t, style.AnchorRight, side(start.Inset.Left).Side)
	assert.Equal(t, style.AnchorWidth, side(start.Size.Height).Side)
	assert.Equal(t, style.AlignStart, start.JustifySelf)
}
