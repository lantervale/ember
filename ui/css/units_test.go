package css //nolint:testpackage // reads computed styles off dom with the package's test helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

// measured is a Transitions whose fonts say an x-height of 0.4em, a zero
// 0.6em wide, capitals 0.7em high, an ideograph 1.1em wide and lines
// 1.5em apart, in a viewport of 800 by 600.
func measured() *Transitions {
	t := NewTransitions()
	t.MeasureFonts(func(f style.Font) FontUnits {
		return FontUnits{Ex: 0.4 * f.Size, Ch: 0.6 * f.Size, Cap: 0.7 * f.Size, Ic: 1.1 * f.Size, Line: 1.5 * f.Size}
	})
	t.SetMedia(Media{Width: 800, Height: 600})
	return t
}

func widthOf(t *testing.T, n *dom.Node) float32 {
	t.Helper()
	w := styleOf(n).Size.Width
	require.Equal(t, style.Fixed, w.Kind)
	require.Zero(t, w.Calc)
	require.Zero(t, w.Percent)
	return w.Px
}

func TestUnitsOfTheFontReadTheElementsOwnFont(t *testing.T) {
	cases := map[string]float32{
		"10ex":                  10 * 0.4 * 20,
		"10ch":                  10 * 0.6 * 20,
		"10cap":                 10 * 0.7 * 20,
		"10ic":                  10 * 1.1 * 20,
		"10lh":                  10 * 1.5 * 20,
		"10rex":                 10 * 0.4 * 10,
		"10rch":                 10 * 0.6 * 10,
		"10RCAP":                10 * 0.7 * 10,
		"10ric":                 10 * 1.1 * 10,
		"10rlh":                 10 * 30,
		"calc(1ex + 1ch + 2px)": 0.4*20 + 0.6*20 + 2,
	}
	for value, want := range cases {
		t.Run(value, func(t *testing.T) {
			box := el(dom.Div, "box")
			root := page(box)
			tr := measured()
			warns := tr.Compute(root, []*Sheet{sheet(t, `html { font-size: 10px; line-height: 30px; }
#box { font-size: 20px; line-height: normal; width: `+value+`; }`)})
			require.Empty(t, warns)
			assert.InDelta(t, want, widthOf(t, box), 0.001)
		})
	}
}

func TestUnitsOfTheFontInItsOwnSizeAndLineHeightReadTheParents(t *testing.T) {
	inner := el(dom.Div, "in")
	root := page(with(el(dom.Div, "out"), inner))
	tr := measured()
	tr.Compute(root, []*Sheet{sheet(t, `html { font-size: 2ex; line-height: 2lh; }
#out { font-size: 10px; line-height: 3; }
#in { font-size: 5ex; line-height: 2lh; width: 1lh; height: 1ex; }`)})
	html := styleOf(root)
	assert.InDelta(t, 2*0.4*16, html.Font.Size, 0.001, "the root's ex is the initial font's")
	assert.InDelta(t, 2*1.5*16, html.LineHeight.Px, 0.001, "the root's lh is the initial line height")
	s := styleOf(inner)
	assert.InDelta(t, 5*0.4*10, s.Font.Size, 0.001)
	assert.InDelta(t, 2*30, s.LineHeight.Px, 0.001)
	assert.InDelta(t, 60, s.Size.Width.Px, 0.001, "lh elsewhere is its own line height")
	assert.InDelta(t, 0.4*20, s.Size.Height.Px, 0.001, "ex elsewhere is its own font's")
}

func TestUnitsOfTheFontWithoutFontsFallBackOnTheFontSize(t *testing.T) {
	box := el(dom.Div, "box")
	root := page(box)
	Compute(root, []*Sheet{sheet(t, `#box { font-size: 20px; width: 1ex; height: 1ch; min-width: 1cap; min-height: 1ic;
max-width: 1lh; }`)})
	s := styleOf(box)
	assert.Equal(t, style.Px(10), s.Size.Width)
	assert.Equal(t, style.Px(10), s.Size.Height)
	assert.Equal(t, style.Px(20), s.MinSize.Width)
	assert.Equal(t, style.Px(20), s.MinSize.Height)
	assert.Equal(t, style.Px(24), s.MaxSize.Width)
}

func TestUnitsOfTheViewport(t *testing.T) {
	cases := map[string]float32{
		"10vw": 80, "10vh": 60, "10vmin": 60, "10vmax": 80, "10vi": 80, "10vb": 60,
		"10svw": 80, "10svh": 60, "10lvw": 80, "10lvh": 60, "10dvw": 80, "10DVH": 60,
		"10svmin": 60, "10lvmax": 80, "10dvi": 80, "10svb": 60,
		"calc(50vw - 10px)": 390,
	}
	for value, want := range cases {
		t.Run(value, func(t *testing.T) {
			box := el(dom.Div, "box")
			root := page(box)
			tr := measured()
			require.Empty(t, tr.Compute(root, []*Sheet{sheet(t, `#box { width: `+value+`; }`)}))
			assert.InDelta(t, want, widthOf(t, box), 0.001)
		})
	}
}

func TestAViewportThatChangesRestylesOnlyWhereItsUnitsAreUsed(t *testing.T) {
	box := el(dom.Div, "box")
	root := page(box)
	tr := measured()
	sheets := []*Sheet{sheet(t, `#box { width: 10px; }`)}
	tr.Compute(root, sheets)
	assert.False(t, tr.SetMedia(Media{Width: 400, Height: 600}), "no rule reads the viewport")

	sheets = []*Sheet{sheet(t, `#box { width: 10vw; }`)}
	tr.Compute(root, sheets)
	assert.InDelta(t, 40, widthOf(t, box), 0.001)
	assert.True(t, tr.SetMedia(Media{Width: 1000, Height: 600}))
	dom.Data(root).Dirty |= dom.DirtyStyle
	tr.Compute(root, sheets)
	assert.InDelta(t, 100, widthOf(t, box), 0.001)
}

func TestFontsThatChangeRestyleOnlyWhereTheirUnitsAreUsed(t *testing.T) {
	box := el(dom.Div, "box")
	root := page(box)
	tr := measured()
	sheets := []*Sheet{sheet(t, `#box { width: 10em; }`)}
	tr.Compute(root, sheets)
	assert.False(t, tr.FontsChanged())

	sheets = []*Sheet{sheet(t, `#box { width: 10ex; }`)}
	tr.Compute(root, sheets)
	tr.MeasureFonts(func(f style.Font) FontUnits { return FontUnits{Ex: f.Size} })
	assert.True(t, tr.FontsChanged())
	dom.Data(root).Dirty |= dom.DirtyStyle
	tr.Compute(root, sheets)
	assert.InDelta(t, 160, widthOf(t, box), 0.001)
}

func TestMinMaxAndClampOfLengthsAreOneLength(t *testing.T) {
	cases := map[string]float32{
		"min(100px, 5em)":                  80,
		"max(100px, 5em)":                  100,
		"MIN(100px, 5em, 3rem)":            48,
		"clamp(10px, 5em, 60px)":           60,
		"clamp(100px, 5em, 200px)":         100,
		"clamp(10px, 5em, 200px)":          80,
		"clamp(100px, 5em, 60px)":          100,
		"min(10vw, 100px)":                 80,
		"calc(min(10px, 1em) * 2 + 1px)":   21,
		"max(min(10px, 20px), 15px)":       15,
		"min(calc(1em + 4px), 30px)":       20,
		"calc(100px - max(2em, 10px) / 2)": 84,
		"min(10px)":                        10,
		"min(2 * 10px, 30px)":              20,
	}
	for value, want := range cases {
		t.Run(value, func(t *testing.T) {
			box := el(dom.Div, "box")
			root := page(box)
			tr := measured()
			require.Empty(t, tr.Compute(root, []*Sheet{sheet(t, `#box { width: `+value+`; }`)}))
			assert.InDelta(t, want, widthOf(t, box), 0.001)
		})
	}
}

func TestMinMaxAndClampOverPercentagesAreWorkedOutAgainstTheirBasis(t *testing.T) {
	cases := map[string][3]float32{
		"min(100%, 500px)":              {300, 500, 500},
		"max(50%, 200px)":               {200, 300, 600},
		"clamp(100px, 50%, 300px)":      {150, 300, 300},
		"calc(min(100%, 500px) - 20px)": {280, 480, 480},
		"min(100% - 40px, 30em)":        {260, 480, 480},
		"max(10%, min(50%, 200px))":     {150, 200, 200},
	}
	for value, want := range cases {
		t.Run(value, func(t *testing.T) {
			box := el(dom.Div, "box")
			root := page(box)
			tr := measured()
			sheets := []*Sheet{sheet(t, `#box { width: `+value+`; }`)}
			require.Empty(t, tr.Compute(root, sheets))
			w := styleOf(box).Size.Width
			require.Equal(t, style.Fixed, w.Kind)
			assert.True(t, w.IsPercent)
			for i, basis := range []float32{300, 600, 1200} {
				got, ok := w.Resolve(basis)
				require.True(t, ok)
				assert.InDelta(t, want[i], got, 0.001, "against %v", basis)
			}
			before := w
			dom.Data(root).Dirty |= dom.DirtyStyle
			tr.Compute(root, sheets)
			assert.Equal(t, before, styleOf(box).Size.Width, "styled again, the same length")
		})
	}
}

func TestMinMaxAndClampOfNumbers(t *testing.T) {
	box := el(dom.Div, "box")
	root := page(box)
	require.Empty(t, Compute(root, []*Sheet{sheet(t, `#box { flex-grow: max(1, 2); opacity: min(0.5, 0.8);
line-height: clamp(1, 3, 2); }`)}))
	s := styleOf(box)
	assert.InDelta(t, 2, s.Grow, 0.001)
	assert.InDelta(t, 0.5, s.Opacity, 0.001)
	assert.InDelta(t, 2, s.LineHeight.Factor, 0.001)
}

func TestMinMaxAndClampThatAreNotValuesAreRefused(t *testing.T) {
	for _, value := range []string{
		"min()", "max(1px, 2)", "clamp(1px, 2px)", "clamp(1px, 2px, 3px, 4px)", "min(1px 2px)", "min(1px,, 2px)",
		"max(1px, red)", "min(1px, 1fr)",
	} {
		t.Run(value, func(t *testing.T) {
			_, warns := Parse("t.css", `#box { width: `+value+`; }`)
			assert.NotEmpty(t, warns)
		})
	}
}
