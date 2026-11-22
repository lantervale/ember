package css //nolint:testpackage // computes styles off dom with the package's unexported test helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

// schemeOf styles a page holding p under sheet for a Document that
// prefers dark or not.
func schemeOf(t *testing.T, src string, dark bool, kids ...*dom.Node) *dom.Node {
	t.Helper()
	root := page(kids...)
	tr := NewTransitions()
	tr.SetMedia(Media{Width: 400, Height: 300, Dark: dark})
	tr.Compute(root, []*Sheet{sheet(t, src)})
	return root
}

func TestColorSchemeReadsItsNamesAndOnly(t *testing.T) {
	for src, want := range map[string]style.ColorScheme{
		"normal":            0,
		"light":             style.SchemeLight,
		"dark":              style.SchemeDark,
		"light dark":        style.SchemeLight | style.SchemeDark,
		"dark light":        style.SchemeLight | style.SchemeDark,
		"only dark":         style.SchemeDark | style.SchemeOnly,
		"light only":        style.SchemeLight | style.SchemeOnly,
		"sepia dark":        style.SchemeDark,
		"sepia":             0,
		"only light custom": style.SchemeLight | style.SchemeOnly,
	} {
		p := el(dom.P, "p")
		schemeOf(t, "#p { color-scheme: "+src+" }", false, p)
		assert.Equal(t, want, styleOf(p).ColorScheme, src)
	}
}

func TestColorSchemeRefusesWhatChromeRefuses(t *testing.T) {
	for _, src := range []string{"only", "light only dark", "only only light", "normal dark", "light none", "#fff",
		"light, dark"} {
		_, warns := Parse("t.css", "p { color-scheme: "+src+" }")
		assert.NotEmpty(t, warns, src)
	}
}

func TestColorSchemeIsInherited(t *testing.T) {
	span := el(dom.Span, "s")
	div := with(el(dom.Div, "d"), span)
	schemeOf(t, "#d { color-scheme: dark }", false, div)
	assert.Equal(t, style.SchemeDark, styleOf(span).ColorScheme)
	assert.True(t, styleOf(span).Dark)
}

func TestTheUsedSchemeIsThePreferredOneWhereBothAreNamed(t *testing.T) {
	for _, c := range []struct {
		scheme string
		dark   bool
		want   bool
	}{
		{"normal", true, false},
		{"light", true, false},
		{"dark", false, true},
		{"light dark", false, false},
		{"light dark", true, true},
		{"only dark", false, true},
		{"sepia dark", false, true},
	} {
		p := el(dom.P, "p")
		schemeOf(t, "#p { color-scheme: "+c.scheme+" }", c.dark, p)
		assert.Equal(t, c.want, styleOf(p).Dark, "%s, preferring dark %v", c.scheme, c.dark)
	}
}

func TestSystemColoursTakeTheirSchemesValues(t *testing.T) {
	src := `#l { color-scheme: light } #d { color-scheme: dark }
p { color: CanvasText; background-color: Canvas; border-top-color: buttonborder; outline-color: LinkText }`
	l, d := el(dom.P, "l"), el(dom.P, "d")
	schemeOf(t, src, false, l, d)
	for _, c := range []struct {
		n                   *dom.Node
		fg, bg, border, out string
	}{
		{l, "#000", "#fff", "#000", "#0000ee"},
		{d, "#fff", "#121212", "#fff", "#9e9eff"},
	} {
		s := styleOf(c.n)
		assert.Equal(t, hex(c.fg), s.Color, c.n.ID())
		assert.Equal(t, hex(c.bg), s.Background, c.n.ID())
		assert.Equal(t, hex(c.border), s.BorderColor.Top, c.n.ID())
		assert.Equal(t, hex(c.out), s.OutlineColor, c.n.ID())
	}
}

func TestLightDarkPicksByTheElementsScheme(t *testing.T) {
	src := `p { color: light-dark(#f00, #00f); background: light-dark(rgb(0 255 0), Canvas);
  box-shadow: 1px 1px light-dark(#100, #200); }
#d { color-scheme: dark }`
	l, d := el(dom.P, "l"), el(dom.P, "d")
	schemeOf(t, src, false, l, d)
	assert.Equal(t, hex("#f00"), styleOf(l).Color)
	assert.Equal(t, hex("#0f0"), styleOf(l).Background)
	require.Len(t, styleOf(l).Shadows, 1)
	assert.Equal(t, hex("#100"), styleOf(l).Shadows[0].Color)
	assert.Equal(t, hex("#00f"), styleOf(d).Color)
	assert.Equal(t, hex("#121212"), styleOf(d).Background)
	assert.Equal(t, hex("#200"), styleOf(d).Shadows[0].Color)
}

func TestLightDarkInADropShadowFilter(t *testing.T) {
	p := el(dom.P, "p")
	schemeOf(t, `#p { color-scheme: dark; filter: drop-shadow(1px 1px light-dark(#100, #200)) }`, false, p)
	require.Len(t, styleOf(p).Filters, 1)
	assert.Equal(t, hex("#200"), styleOf(p).Filters[0].Shadow.Color)
}

func TestLightDarkInAGradientAndThroughAVariable(t *testing.T) {
	src := `:root { --ink: light-dark(#f00, #00f) }
p { color: var(--ink); background-image: linear-gradient(light-dark(#100, #200), #300) }
#d { color-scheme: dark }`
	l, d := el(dom.P, "l"), el(dom.P, "d")
	schemeOf(t, src, false, l, d)
	assert.Equal(t, hex("#f00"), styleOf(l).Color, "the variable resolves where it is used")
	assert.Equal(t, hex("#00f"), styleOf(d).Color)
	assert.NotEqual(t, styleOf(l).BackgroundLayers(), 0)
	lg, dg := styleOf(l).BackgroundLayer(0), styleOf(d).BackgroundLayer(0)
	assert.NotEqual(t, lg, dg, "the gradient's first stop follows the scheme")
}

func TestASchemedColourIsInheritedAsResolved(t *testing.T) {
	span := el(dom.Span, "s")
	schemeOf(t, `#d { color-scheme: dark; color: CanvasText } #s { color-scheme: light }`, false,
		with(el(dom.Div, "d"), span))
	assert.Equal(t, hex("#fff"), styleOf(span).Color, "as Chrome: the parent's white, not the child's black")
}

func TestTheRootsTextFollowsItsScheme(t *testing.T) {
	p := el(dom.P, "p")
	root := schemeOf(t, `:root { color-scheme: light dark }`, true, p)
	assert.Equal(t, hex("#fff"), styleOf(root).Color)
	assert.Equal(t, hex("#fff"), styleOf(p).Color)
	p = el(dom.P, "p")
	root = schemeOf(t, `:root { color-scheme: light dark }`, false, p)
	assert.Equal(t, hex("#000"), styleOf(root).Color)
	p = el(dom.P, "p")
	schemeOf(t, `#p { color-scheme: dark; color: initial }`, false, p)
	assert.Equal(t, hex("#fff"), styleOf(p).Color, "initial is canvastext, in the element's scheme")
}

func TestASystemColoursNameIsAWordOutsideColours(t *testing.T) {
	p := el(dom.P, "p")
	schemeOf(t, `@keyframes mark { from { opacity: 0 } } #p { color-scheme: dark; animation-name: mark;
  grid-area: menu }`, false, p)
	require.Len(t, styleOf(p).Animations, 1)
	assert.Equal(t, "mark", styleOf(p).Animations[0].Name)
}

func TestAPreferenceFlipRestylesOnlyWhereAnElementFollowsIt(t *testing.T) {
	p := el(dom.P, "p")
	root := page(p)
	tr := NewTransitions()
	sheets := []*Sheet{sheet(t, `#p { color: light-dark(#f00, #00f) }`)}
	tr.SetMedia(Media{Width: 400, Height: 300})
	tr.Compute(root, sheets)
	assert.False(t, tr.SetMedia(Media{Width: 400, Height: 300, Dark: true}),
		"nothing follows the preference: every element is light")
	sheets = []*Sheet{sheet(t, `#p { color-scheme: light dark; color: light-dark(#f00, #00f) }`)}
	dom.Data(root).Dirty |= dom.DirtyStyle
	tr.Compute(root, sheets)
	assert.Equal(t, hex("#00f"), styleOf(p).Color)
	require.True(t, tr.SetMedia(Media{Width: 400, Height: 300}), "p follows the preference")
	dom.Data(root).Dirty |= dom.DirtyStyle
	tr.Compute(root, sheets)
	assert.Equal(t, hex("#f00"), styleOf(p).Color)
}

func TestAnUnknownLightDarkIsRefused(t *testing.T) {
	for _, src := range []string{"light-dark(#fff)", "light-dark(#fff, #000, #111)", "light-dark(#fff #000)",
		"light-dark(#fff, nonsense)"} {
		_, warns := Parse("t.css", "p { color: "+src+" }")
		assert.NotEmpty(t, warns, src)
	}
	_, warns := Parse("t.css", "p { color: light-dark(currentcolor, Canvas) }")
	assert.Empty(t, warns)
}
