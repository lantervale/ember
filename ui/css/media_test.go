package css //nolint:testpackage // reads a sheet's media lists and the package's unexported test helpers

import (
	"slices"
	"strings"
	"testing"

	"github.com/lantervale/ember/ui/internal/dom"
)

// holds answers query for m, as a sheet's @media rule would, with what
// the sheet warned of.
func holds(query string, m Media, opts ...ParseOption) (bool, string) {
	s, warns := Parse("t.css", "@media "+query+" { p { color: #f00 } }", opts...)
	return s.media[0].matches(m), warnings(warns)
}

// chromeViewport is the viewport headless Chrome 157 answered the
// queries of TestMediaQueriesAreAnsweredAsChromeAnswers in.
//
//nolint:gochecknoglobals // a test's fixture
var chromeViewport = Media{Width: 800, Height: 513}

// TestMediaQueriesAreAnsweredAsChromeAnswers holds Chrome's own answers,
// from matchMedia in a window 800 by 513; the chrome suite asks Chrome
// again (TestChromeAnswersMediaQueriesTheSame).
func TestMediaQueriesAreAnsweredAsChromeAnswers(t *testing.T) {
	for _, c := range mediaCases {
		got, _ := holds(c.query, chromeViewport)
		if got != c.want {
			t.Errorf("@media %s: %v, want %v", c.query, got, c.want)
		}
	}
}

// TestWhatChromeAsksTheMachineFollowsTheMedium holds ui's answers where
// Chrome's are the machine's, its pointing devices and its user's
// preferences, for each medium the chrome suite has Chrome emulate
// (TestChromeAnswersTheMachinesQueriesTheSame).
func TestWhatChromeAsksTheMachineFollowsTheMedium(t *testing.T) {
	for _, medium := range machineMedia {
		m := medium.m
		m.Width, m.Height = chromeViewport.Width, chromeViewport.Height
		for _, q := range machineQueries {
			got, warns := holds(q, m)
			if want := slices.Contains(medium.holding, q); got != want || warns != "" {
				t.Errorf("@media %s for %+v: %v, want %v %s", q, medium.m, got, want, warns)
			}
		}
	}
}

//nolint:gochecknoglobals // a test's table
var machineQueries = []string{
	"(hover: hover)", "(hover: none)", "(hover)", "(any-hover: hover)", "(any-hover: none)", "(any-hover)",
	"(pointer: fine)", "(pointer: coarse)", "(pointer: none)", "(pointer)",
	"(any-pointer: fine)", "(any-pointer: coarse)", "(any-pointer: none)", "(any-pointer)",
	"(prefers-reduced-motion: reduce)", "(prefers-reduced-motion: no-preference)", "(prefers-reduced-motion)",
	"(prefers-contrast: no-preference)", "(prefers-contrast: more)", "(prefers-contrast)",
	"(forced-colors: none)", "(forced-colors: active)", "(forced-colors)",
	"(prefers-color-scheme: light)", "(prefers-color-scheme: dark)", "(prefers-color-scheme)",
	"not (prefers-color-scheme: dark)", "(prefers-color-scheme: dark) and (prefers-reduced-motion)",
}

// machineMedia are the media the machine's queries are answered for,
// each with the queries that hold.
//
//nolint:gochecknoglobals // a test's table
var machineMedia = []struct {
	m       Media
	holding []string
}{
	{Media{}, []string{
		"(hover: hover)", "(hover)", "(any-hover: hover)", "(any-hover)", "(pointer: fine)", "(pointer)",
		"(any-pointer: fine)", "(any-pointer)", "(prefers-reduced-motion: no-preference)",
		"(prefers-contrast: no-preference)", "(forced-colors: none)", "(prefers-color-scheme: light)",
		"(prefers-color-scheme)", "not (prefers-color-scheme: dark)",
	}},
	{Media{Dark: true, ReducedMotion: true}, []string{
		"(hover: hover)", "(hover)", "(any-hover: hover)", "(any-hover)", "(pointer: fine)", "(pointer)",
		"(any-pointer: fine)", "(any-pointer)", "(prefers-reduced-motion: reduce)", "(prefers-reduced-motion)",
		"(prefers-contrast: no-preference)", "(forced-colors: none)", "(prefers-color-scheme: dark)",
		"(prefers-color-scheme)", "(prefers-color-scheme: dark) and (prefers-reduced-motion)",
	}},
	{Media{Touch: true}, []string{
		"(hover: none)", "(any-hover: hover)", "(any-hover)", "(pointer: coarse)", "(pointer)",
		"(any-pointer: fine)", "(any-pointer: coarse)", "(any-pointer)", "(prefers-reduced-motion: no-preference)",
		"(prefers-contrast: no-preference)", "(forced-colors: none)", "(prefers-color-scheme: light)",
		"(prefers-color-scheme)", "not (prefers-color-scheme: dark)",
	}},
}

//nolint:gochecknoglobals // a test's table
var mediaCases = []struct {
	query string
	want  bool
}{
	{"screen", true}, {"print", false}, {"all", true}, {"tv", false}, {"handheld", false}, {"speech", false},
	{"(max-width: calc(801px - 1px))", true}, {"(max-width: calc(800px - 1px))", false},
	{"(min-width: calc(40em + 160px))", true}, {"(min-width: calc(2 * 25rem + 1px))", false},
	{"(width >= calc(1in * 8))", true}, {"(calc(900px - 100px) = width)", true},
	{"(min-width: calc(1px + 1%))", false}, {"(min-width: calc(2))", false},
	{"only screen", true}, {"not screen", false}, {"not print", true}, {"", true}, {"screen, print", true},
	{"print, (unknown-thing), screen", true}, {"not all", false}, {"SCREEN AND (COLOR)", true},
	{"(color)", true}, {"(min-color: 8)", true}, {"(color: 8)", true}, {"(monochrome)", false},
	{"(monochrome: 0)", true}, {"(grid)", false}, {"(grid: 0)", true}, {"(orientation)", true}, {"(width)", true},
	{"(unknown-thing)", false}, {"not (unknown-thing)", false}, {"(unknown-thing) or (color)", true},
	{"(unknown-thing) and (color)", false}, {"not ((unknown-thing) and (color))", false},
	{"screen and (unknown-thing)", false}, {"not screen and (unknown-thing)", false},
	{"(min-resolution: 1dppx)", true}, {"(min-resolution: 2dppx)", false}, {"(resolution: 96dpi)", true},
	{"(min-width: 0em)", true}, {"(min-width: 50em)", true}, {"(min-width: 50.1em)", false},
	{"(min-width: 50rem)", true}, {"(min-width: 600pt)", true}, {"(min-width: 8in)", true},
	{"(min-width: 8.5in)", false}, {"(min-width: 20cm)", true}, {"(min-width: 22cm)", false},
	{"(min-width: 1Q)", true}, {"(min-width: 10pc)", true}, {"(min-width: 10mm)", true},
	{"(width >= 800px)", true}, {"(width > 800px)", false}, {"(width = 800px)", true}, {"(800px <= width)", true},
	{"(700px < width < 900px)", true}, {"(900px > width > 700px)", true}, {"(700px < width > 900px)", false},
	{"(width: 800px)", true}, {"(min-width: 800px)", true}, {"(max-width: 800px)", true},
	{"(max-width: 799.9px)", false}, {"(min-width: 100vw)", true}, {"(min-width: 1e2px)", true},
	{"(min-width: 50vw)", true}, {"(max-width: 99vw)", false}, {"(min-height: 100vh)", true}, {"(width >= 155vh)", true},
	{"(width >= 156vh)", false}, {"(max-width: 100vmax)", true}, {"(min-width: 155vmin)", true},
	{"(min-width: 156vmin)", false}, {"(400px < width < 101vw)", true}, {"(min-width: calc(50vw + 1px))", true},
	{"(min-width: calc(100vw + 1px))", false}, {"(min-height: 10dvh)", true}, {"(min-height: 10svh)", true},
	{"(min-width: 10lvw)", true}, {"(min-aspect-ratio: 1vw)", false}, {"(min-width: min(50vw, 1000px))", true},
	{"(min-width: max(101vw, 10px))", false}, {"(min-width: clamp(10px, 99vw, 2000px))", true},
	{"(width < 800.5px)", true}, {"(MIN-WIDTH: 800PX)", true},
	{"(min-aspect-ratio: 4/3)", true}, {"(aspect-ratio: 4/3)", false}, {"(max-aspect-ratio: 1/1)", false},
	{"(min-aspect-ratio: 1)", true}, {"(aspect-ratio > 1.3)", true},
	{"(orientation: landscape)", true}, {"(orientation: portrait)", false},
	{"(min-width: 0)", true}, {"(min-width: 5)", false}, {"(min-width: 800)", false}, {"(min-width: -1px)", true},
	{"(height >= 600px)", false}, {"(min-height: 601px)", false}, {"(height: 600px)", false},
	{"screen and (color) and (min-width: 1px)", true}, {"screen and (color) or (min-width: 1px)", false},
	{"(color) and (min-width:1px) or (grid)", false}, {"screen and ((color) or (grid))", true},
	{"only (color)", false}, {"not", false}, {"screen and", false}, {"and (color)", false},
	{"screen and not (grid)", true}, {"not screen and not (grid)", false},
	{"(min-width:800px)and (color)", true}, {"(color)and(grid)", false}, {"screen and(color)", false},
	{"(min-width: 12.5em) and (max-width: 99999px)", true},
	// ui knows no screen: the device is the viewport, as in a headless Chrome.
	{"(device-width: 800px)", true}, {"(min-device-width: 100px)", true}, {"(max-device-width: 480px)", false},
}

func TestAMediaQueryFollowsTheViewport(t *testing.T) {
	for _, c := range []struct {
		query string
		m     Media
		want  bool
	}{
		{"(min-width: 600px)", Media{Width: 599.5, Height: 400}, false},
		{"(min-width: 600px)", Media{Width: 600, Height: 400}, true},
		{"(max-height: 400px)", Media{Width: 600, Height: 400}, true},
		{"(orientation: portrait)", Media{Width: 400, Height: 400}, true},
		{"(orientation: landscape)", Media{Width: 401, Height: 400}, true},
		{"(min-width: 50vh)", Media{Width: 300, Height: 700}, false},
		{"(min-width: 50vmin)", Media{Width: 300, Height: 700}, true},
		{"(max-width: 50vmax)", Media{Width: 300, Height: 700}, true},
		{"(min-resolution: 2dppx)", Media{Width: 300, Height: 700, Resolution: 2}, true},
		{"(min-resolution: 192dpi)", Media{Width: 300, Height: 700, Resolution: 1.5}, false},
		{"(resolution >= 50dpcm)", Media{Width: 300, Height: 700, Resolution: 2}, true},
		{"(prefers-color-scheme: light)", Media{}, true},
		{"(prefers-color-scheme: dark)", Media{}, false},
		{"(prefers-reduced-motion: reduce)", Media{ReducedMotion: true}, true},
		{"(prefers-reduced-motion)", Media{ReducedMotion: true}, true},
		{"(prefers-reduced-motion: no-preference)", Media{ReducedMotion: true}, false},
	} {
		if got, warns := holds(c.query, c.m); got != c.want || warns != "" {
			t.Errorf("@media %s at %+v: %v, want %v %s", c.query, c.m, got, c.want, warns)
		}
	}
}

func TestWhatAMediaQueryCannotAnswerIsAWarning(t *testing.T) {
	for query, want := range map[string]string{
		"(unknown-thing)":   "t.css:1:8: @media: the unknown-thing feature is not supported; its query never holds",
		"(min-width: 40ch)": "t.css:1:20: @media: min-width can't take 40ch; its query never holds",
		"(min-width: calc(1px + 2%))": "t.css:1:20: @media: min-width can't take calc(1px + 2%); its query never " +
			"holds",
		"(hover: sometimes)": "t.css:1:16: @media: hover takes none, hover; its query never holds",
		"(min-hover: hover)": "t.css:1:8: @media: the min-hover feature is not supported; its query never holds",
		"(min-width)":        "t.css:1:8: @media: the min-width feature is not supported; its query never holds",
		"(display-mode: browser)": "t.css:1:8: @media: the display-mode feature is not supported; its query never " +
			"holds",
		"(-webkit-min-device-pixel-ratio: 2)": "t.css:1:8: @media: the -webkit-min-device-pixel-ratio feature is not " +
			"supported; its query never holds",
		"screen and":              `t.css:1:8: @media: "screen and" is no media query, and never holds`,
		"print, , tv":             `t.css:1:7: @media: "" is no media query, and never holds`,
		"layer(base)":             `t.css:1:8: @media: layer() is not supported; its query never holds`,
		"(700px < width > 900px)": "t.css:1:8: @media: (700px < width > 900px) is no media feature; its query never holds",
	} {
		if got, warns := holds(query, chromeViewport); got || warns != want {
			t.Errorf("@media %s: %v\n got %s\nwant %s", query, got, warns, want)
		}
	}
	if _, warns := Parse("t.css", "@media screen;"); warnings(warns) != "t.css:1:1: @media has no { block }" {
		t.Errorf("no block: %s", warnings(warns))
	}
}

func TestABrowsersSheetReadsChromesPixelRatio(t *testing.T) {
	for _, c := range []struct {
		query string
		dppx  float32
		want  bool
		warns string
	}{
		{"(-webkit-min-device-pixel-ratio: 0)", 1, true, ""},
		{"(-webkit-min-device-pixel-ratio: 2)", 1, false, ""},
		{"(-webkit-min-device-pixel-ratio: 1.5)", 2, true, ""},
		{"(-webkit-max-device-pixel-ratio: 1)", 2, false, ""},
		{"(-webkit-device-pixel-ratio: 2)", 2, true, ""},
		{"screen and (-webkit-min-device-pixel-ratio:0)", 1, true, ""},
		{"(-webkit-animation)", 1, false, ""},
		{"not (-webkit-animation)", 1, false, ""},
		{"(-webkit-transform-3d)", 1, false,
			"t.css:1:8: @media: the -webkit-transform-3d feature is not supported; its query never holds"},
	} {
		got, warns := holds(c.query, Media{Width: 800, Height: 600, Resolution: c.dppx}, Browser())
		if got != c.want || warns != c.warns {
			t.Errorf("@media %s at %gx: %v %s, want %v %s", c.query, c.dppx, got, warns, c.want, c.warns)
		}
	}
}

func TestRulesInAMediaRuleApplyWhileItHolds(t *testing.T) {
	sheets := []*Sheet{sheet(t, `
p { color: #100; }
@media (min-width: 600px) {
  p { color: #200; }
  @media (min-height: 500px) { p { color: #300; } }
  #p::placeholder { color: #400; }
}
@media print { p { color: #500; } }
p { opacity: 0.5; }
`)}
	p := el(dom.P, "p")
	p.SetAttr("placeholder", "name")
	root := page(p)
	tr := NewTransitions()
	at := func(w, h float32) (string, string, bool) {
		var colour, placeholder string
		changed := tr.SetMedia(Media{Width: w, Height: h})
		if changed {
			dom.Data(root).Dirty |= dom.DirtyStyle
		}
		tr.Compute(root, sheets)
		s := styleOf(p)
		for _, c := range []string{"#100", "#200", "#300", "#400", "#500"} {
			if s.Color == hex(c) {
				colour = c
			}
			if s.HasPlaceholder && s.PlaceholderColor == hex(c) {
				placeholder = c
			}
		}
		return colour, placeholder, changed
	}
	for _, c := range []struct {
		w, h                float32
		colour, placeholder string
		changed             bool
	}{
		{500, 400, "#100", "", false},
		{599, 300, "#100", "", false},
		{600, 300, "#200", "#400", true},
		{900, 499, "#200", "#400", false},
		{900, 500, "#300", "#400", true},
		{600, 500, "#300", "#400", false},
		{400, 800, "#100", "", true},
	} {
		before := tr.computed
		colour, placeholder, changed := at(c.w, c.h)
		if colour != c.colour || placeholder != c.placeholder || changed != c.changed {
			t.Errorf("%g by %g: colour %s, placeholder %q, changed %v; want %s, %q, %v", c.w, c.h, colour, placeholder,
				changed, c.colour, c.placeholder, c.changed)
		}
		if restyled := tr.computed - before; !c.changed && c.w != 500 && restyled != 0 {
			t.Errorf("%g by %g crosses no boundary and restyled %d elements", c.w, c.h, restyled)
		}
		if styleOf(p).Opacity != 0.5 {
			t.Errorf("%g by %g: the rule after the @media rules is lost", c.w, c.h)
		}
	}
}

func TestACrossedBoundaryLaysOutOnlyWhatChanged(t *testing.T) {
	sheets := []*Sheet{sheet(t, `@media (min-width: 600px) { #a { width: 10px; } #b { color: #f00; } }`)}
	a, b, c := el(dom.Div, "a"), el(dom.Div, "b"), el(dom.Div, "c")
	root := page(a, b, c)
	tr := NewTransitions()
	tr.SetMedia(Media{Width: 500, Height: 400})
	tr.Compute(root, sheets)
	root.Walk(func(n *dom.Node) bool { dom.Data(n).Dirty = 0; return true })
	if !tr.SetMedia(Media{Width: 700, Height: 400}) {
		t.Fatal("600 is crossed")
	}
	dom.Data(root).Dirty |= dom.DirtyStyle
	tr.Compute(root, sheets)
	if dom.Data(a).Dirty&dom.DirtyLayout == 0 {
		t.Error("a's width changed: it lays out again")
	}
	if dom.Data(b).Dirty&dom.DirtyLayout != 0 || dom.Data(c).Dirty&dom.DirtyLayout != 0 {
		t.Error("b only changed colour and c nothing: neither lays out again")
	}
	if styleOf(b).Color != hex("#f00") {
		t.Errorf("b: %v", styleOf(b).Color)
	}
}

func TestASheetWritesItsMediaRulesForChrome(t *testing.T) {
	s := sheet(t, `p { color: #100; }
@media screen  and (min-width: 600px) , print { p { color: #200; } @media (hover) { a { color: #300; } } }
a { color: #400; }`)
	want := `p { color: #100; }
@media screen and (min-width: 600px) , print { p { color: #200; } }
@media screen and (min-width: 600px) , print { @media (hover) { a { color: #300; } } }
a { color: #400; }
`
	if got := s.String(); got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
	again, warns := Parse("t.css", s.String())
	if len(warns) > 0 || again.String() != want {
		t.Errorf("read again: %s\n%s", warnings(warns), again.String())
	}
}

func TestMediaTextKeepsItsCase(t *testing.T) {
	s := sheet(t, "@media (MIN-WIDTH: 10PX) { p { color: #f00 } }")
	if !strings.Contains(s.String(), "@media (MIN-WIDTH: 10PX) {") {
		t.Errorf("got %s", s.String())
	}
}
