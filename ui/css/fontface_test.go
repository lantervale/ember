package css //nolint:testpackage // uses the package's unexported test helpers (warnings)

import (
	"reflect"
	"testing"

	"github.com/lantervale/ember/ui/internal/style"
)

func faceOf(t *testing.T, src string, opts ...ParseOption) (FontFace, string) {
	t.Helper()
	s, warns := Parse("ui/tool.css", src, opts...)
	if len(s.FontFaces()) != 1 {
		t.Fatalf("%d faces, want 1; warnings: %s", len(s.FontFaces()), warnings(warns))
	}
	return s.FontFaces()[0], warnings(warns)
}

func TestAFontFaceKeepsItsWholeSrcList(t *testing.T) {
	f, warns := faceOf(t, `@font-face {
	font-family: FontAwesome;
	src: url('fonts/fa.eot?v=4.7.0');
	src: url('fonts/fa.eot?#iefix&v=4.7.0') format('embedded-opentype'),
		url(fonts/fa.woff2?v=4.7.0) format("woff2") tech(variations),
		local(Font Awesome), local("FontAwesome"),
		url('fonts/fa.svg#fa') format(svg),
		url(fonts/fa.ttf);
}`)
	if warns != "" {
		t.Errorf("warnings: %s", warns)
	}
	want := []FontSource{
		{URL: "ui/fonts/fa.eot?#iefix&v=4.7.0", Format: "embedded-opentype"},
		{URL: "ui/fonts/fa.woff2?v=4.7.0", Format: "woff2", Tech: []string{"variations"}},
		{Local: "Font Awesome"},
		{Local: "FontAwesome"},
		{URL: "ui/fonts/fa.svg#fa", Format: "svg"},
		{URL: "ui/fonts/fa.ttf"},
	}
	if !reflect.DeepEqual(f.Sources, want) {
		t.Errorf("sources:\n%+v\nwant\n%+v", f.Sources, want)
	}
	if f.Src != "ui/fonts/fa.woff2?v=4.7.0" {
		t.Errorf("Src is %q, want the first file ui reads", f.Src)
	}
}

func TestTheSourcesUIReads(t *testing.T) {
	for _, tc := range []struct {
		src  FontSource
		want bool
	}{
		{FontSource{URL: "a.eot"}, false},
		{FontSource{URL: "a.eot?#iefix"}, false},
		{FontSource{URL: "a.svg#name"}, false},
		{FontSource{URL: "a.ttf"}, true},
		{FontSource{URL: "a.OTF"}, true},
		{FontSource{URL: "a.woff?v=2"}, true},
		{FontSource{URL: "a.woff2"}, true},
		{FontSource{URL: "https://use.typekit.net/af/1/l?primer=7c"}, true},
		{FontSource{URL: "a.bin", Format: "woff2"}, true},
		{FontSource{URL: "a.woff2", Format: "embedded-opentype"}, false},
		{FontSource{URL: "a", Format: "truetype"}, true},
		{FontSource{URL: "a", Format: "opentype"}, true},
		{FontSource{URL: "a", Format: "woff"}, true},
		{FontSource{URL: "a", Format: "woff2-variations"}, true},
		{FontSource{URL: "a", Format: "truetype-variations"}, true},
		{FontSource{URL: "a", Format: "collection"}, false},
		{FontSource{URL: "a", Format: "svg"}, false},
		{FontSource{URL: "a.woff2", Format: "woff2", Tech: []string{"variations"}}, true},
		{FontSource{URL: "a.woff2", Format: "woff2", Tech: []string{"color-colrv1"}}, false},
		{FontSource{Local: "Arial"}, true},
		{FontSource{URL: "data:font/woff2;base64,d09GMg=="}, true},
	} {
		if got := tc.src.Reads(); got != tc.want {
			t.Errorf("%+v: Reads() = %v, want %v", tc.src, got, tc.want)
		}
	}
}

func TestAFontFaceSrcIsResolvedFromItsSheet(t *testing.T) {
	f, _ := faceOf(t,
		`@font-face { font-family: M; src: url(../fonts/m.woff2), url(/abs/m.woff), url(//cdn.example/m.ttf) }`,
		From("https://lantervale.example/css/site/main.css"))
	want := []string{
		"https://lantervale.example/css/fonts/m.woff2", "https://lantervale.example/abs/m.woff",
		"https://cdn.example/m.ttf",
	}
	for i, s := range f.Sources {
		if s.URL != want[i] {
			t.Errorf("source %d is %q, want %q", i, s.URL, want[i])
		}
	}
}

func TestAFontFaceWithNoSourceUIReadsIsDropped(t *testing.T) {
	s, warns := Parse("ui/tool.css",
		`@font-face { font-family: Old; src: url(old.eot), url(old.svg#old) format("svg") }`)
	if len(s.FontFaces()) != 0 {
		t.Errorf("faces: %+v", s.FontFaces())
	}
	want := `ui/tool.css:1:1: @font-face "Old": no source ui reads (TrueType, OpenType, WOFF, WOFF2); dropped`
	if got := warnings(warns); got != want {
		t.Errorf("warnings: %s\nwant %s", got, want)
	}
}

func TestUnicodeRange(t *testing.T) {
	f, warns := faceOf(t, `@font-face { font-family: M; src: url(m.woff2);
	unicode-range: U+0000-00FF, U+0131, u+4??, U+2B0C-2Bff, U+1F00-1FFF, U+A720-A7FF,U+E0FF,U+0-7F,
		U+1E00-1EFF, U+2E80-2E99, U+00E9, U+1E??, U+F002, U+10FFFF }`)
	if warns != "" {
		t.Errorf("warnings: %s", warns)
	}
	want := []UnicodeRange{
		{0x0, 0xff}, {0x131, 0x131}, {0x400, 0x4ff}, {0x2b0c, 0x2bff}, {0x1f00, 0x1fff}, {0xa720, 0xa7ff},
		{0xe0ff, 0xe0ff}, {0x0, 0x7f}, {0x1e00, 0x1eff}, {0x2e80, 0x2e99}, {0xe9, 0xe9}, {0x1e00, 0x1eff},
		{0xf002, 0xf002}, {0x10ffff, 0x10ffff},
	}
	if !reflect.DeepEqual(f.Ranges, want) {
		t.Errorf("ranges:\n%v\nwant\n%v", f.Ranges, want)
	}
	if !f.Covers('é') || !f.Covers(0x4a0) || f.Covers(0x100) || f.Covers(0x2000) {
		t.Errorf("Covers: é %v, U+4A0 %v, U+100 %v, U+2000 %v",
			f.Covers('é'), f.Covers(0x4a0), f.Covers(0x100), f.Covers(0x2000))
	}
	all, _ := faceOf(t, `@font-face { font-family: M; src: url(m.woff2) }`)
	if all.Ranges != nil || !all.Covers(0x10ffff) {
		t.Errorf("a face with no unicode-range covers every character; ranges %v", all.Ranges)
	}
}

func TestABadUnicodeRangeIsAWarningAndTheFaceCoversAll(t *testing.T) {
	for _, bad := range []string{"U+", "U+110000", "U+20-10", "U+1?2", "U+??-FF", "U+1234567", "0-FF", "U+00 FF"} {
		f, warns := faceOf(t, `@font-face { font-family: M; src: url(m.woff2); unicode-range: `+bad+` }`)
		if warns == "" {
			t.Errorf("%s: no warning", bad)
		}
		if f.Ranges != nil {
			t.Errorf("%s: ranges %v", bad, f.Ranges)
		}
	}
}

func TestAFontFaceWeightStyleAndDisplay(t *testing.T) {
	f, warns := faceOf(t, `@font-face { font-family: V; src: url(v.woff2) format("woff2-variations");
	font-weight: 100 900; font-style: oblique 0deg 10deg; font-display: swap }`)
	if warns != "" {
		t.Errorf("warnings: %s", warns)
	}
	if f.Weight != 100 || f.WeightMax != 900 || !f.Italic || f.Display != "swap" {
		t.Errorf("face: %+v", f)
	}
	one, _ := faceOf(t, `@font-face { font-family: V; src: url(v.ttf); font-weight: bold }`)
	if one.Weight != 700 || one.WeightMax != 700 || one.Display != "auto" {
		t.Errorf("face: %+v", one)
	}
	swapped, _ := faceOf(t, `@font-face { font-family: V; src: url(v.ttf); font-weight: 900 normal }`)
	if swapped.Weight != 400 || swapped.WeightMax != 900 {
		t.Errorf("a range written backwards: %+v", swapped)
	}
	if f.AutoWeight || one.AutoWeight || !f.AutoStretch {
		t.Errorf("auto: a face that gives its weight is not, one that gives no width is: %+v, %+v", f, one)
	}
	auto, warns := faceOf(t, `@font-face { font-family: V; src: url(v.ttf); font-weight: auto; font-stretch: auto }`)
	if warns != "" || !auto.AutoWeight || !auto.AutoStretch || auto.Weight != 400 || auto.Stretch != 100 {
		t.Errorf("auto: %+v, warnings %s", auto, warns)
	}
	none, _ := faceOf(t, `@font-face { font-family: V; src: url(v.ttf) }`)
	if !none.AutoWeight || !none.AutoStretch {
		t.Errorf("a face that gives neither: %+v", none)
	}
}

func TestFontVariationSettingsAreTagsAndNumbers(t *testing.T) {
	for _, c := range []struct {
		in   string
		want []style.FontVariation
		ok   bool
	}{
		{`normal`, nil, true},
		{`"wght" 650`, []style.FontVariation{{Tag: "wght", Value: 650}}, true},
		{`"wght" 650.5, 'wdth' -2`, []style.FontVariation{{Tag: "wdth", Value: -2}, {Tag: "wght", Value: 650.5}}, true},
		{`"wght" 1, "XXXX" 2, "wght" 3`, []style.FontVariation{{Tag: "XXXX", Value: 2}, {Tag: "wght", Value: 3}}, true},
		{`"wght"`, nil, false},
		{`"wghtt" 1`, nil, false},
		{`wght 1`, nil, false},
		{`"wght" 1 2`, nil, false},
		{`"wght" 1,`, nil, false},
	} {
		got, ok := fontVariations(componentsOf(c.in))
		if ok != c.ok || !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: %+v %v, want %+v %v", c.in, got, ok, c.want, c.ok)
		}
	}
}

func TestDescriptorsUIDoesNotUseAreSilentForABrowsersSheet(t *testing.T) {
	const src = `@font-face { font-family: M; src: url(m.woff2); font-stretch: 75% 125%; size-adjust: 90%;
	ascent-override: 90%; descent-override: 20%; line-gap-override: 0%; font-feature-settings: "liga" 0;
	font-variation-settings: "wght" 400; font-variant: normal; font-named-instance: auto;
	font-language-override: normal; font-width: normal }`
	if _, warns := faceOf(t, src, Browser()); warns != "" {
		t.Errorf("a browser's sheet warns: %s", warns)
	}
	_, warns := faceOf(t, `@font-face { font-family: M; src: url(m.woff2); font-variation-settings: "wght" 400 }`)
	if want := `ui/tool.css:1:49: @font-face: "font-variation-settings" is not supported`; warns != want {
		t.Errorf("a game's sheet: %s\nwant %s", warns, want)
	}
	_, warns = faceOf(t, `@font-face { font-family: M; src: url(m.woff2); font-colour: red }`, Browser())
	if want := `ui/tool.css:1:49: @font-face: "font-colour" is not supported`; warns != want {
		t.Errorf("a descriptor CSS has not: %s\nwant %s", warns, want)
	}
}

func TestAFacesWidthMetricsAndFeaturesAreRead(t *testing.T) {
	f, warns := faceOf(t, `@font-face { font-family: M; src: url(m.woff2); font-stretch: 125% condensed;
	size-adjust: 90%; ascent-override: 80%; descent-override: 0%; line-gap-override: normal;
	font-feature-settings: "kern" off, "liga", "salt" 3 }`)
	if warns != "" {
		t.Fatalf("warns: %s", warns)
	}
	if f.Stretch != 75 || f.StretchMax != 125 {
		t.Errorf("stretch %v to %v, want 75%% to 125%%", f.Stretch, f.StretchMax)
	}
	if f.SizeAdjust != 0.9 || f.Ascent != 0.8 || f.Descent != 0 || f.LineGap != -1 {
		t.Errorf("size-adjust %v, ascent %v, descent %v, line gap %v", f.SizeAdjust, f.Ascent, f.Descent, f.LineGap)
	}
	want := []FontFeature{{"kern", 0}, {"liga", 1}, {"salt", 3}}
	if !reflect.DeepEqual(f.Features, want) {
		t.Errorf("features %v, want %v", f.Features, want)
	}
	if kern, ok := f.Feature("kern"); !ok || kern != 0 {
		t.Errorf("kern %d, %v", kern, ok)
	}
	plain, _ := faceOf(t, `@font-face { font-family: M; src: url(m.woff2) }`)
	if plain.Stretch != 100 || plain.StretchMax != 100 || plain.SizeAdjust != 1 || plain.Ascent != -1 ||
		plain.Features != nil {
		t.Errorf("a face that says none: %+v", plain)
	}
	for _, bad := range []string{`font-stretch: 10px`, `size-adjust: 1.2`, `ascent-override: -5%`,
		`font-feature-settings: liga`, `font-feature-settings: "ligature"`, `font-feature-settings: "kern" -1`} {
		if _, bw := faceOf(t, `@font-face { font-family: M; src: url(m.woff2); `+bad+` }`); bw == "" {
			t.Errorf("%s: no warning", bad)
		}
	}
}
