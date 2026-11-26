package css //nolint:testpackage // reads the imports and the rules as the cascade does

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseReadsTheImportsABrowserFollows(t *testing.T) {
	s, warns := Parse("site.css", `@charset "utf-8";
@layer base;
@import url(reset.css);
@import "theme/dark.css" layer(theme) screen and (min-width: 600px);
@import url("print.css") print;
@import url(https://cdn.test/grid.css) layer supports(display: grid);
@import url(flex.css) supports((display: flex) and (not (display: bogus)));
@import url(never.css) supports(display: bogus);
p { color: #f00 }
@import url(late.css);
@media screen { @import url(inner.css); }`, Browser(), From("https://site.test/css/site.css"))
	got := s.Imports()
	require.Len(t, got, 5)
	var urls []string
	for _, imp := range got {
		urls = append(urls, imp.URL)
	}
	assert.Equal(t, []string{
		"https://site.test/css/reset.css", "https://site.test/css/theme/dark.css", "https://site.test/css/print.css",
		"https://cdn.test/grid.css", "https://site.test/css/flex.css",
	}, urls)
	assert.Equal(t, Pos{3, 1}, got[0].Pos)
	assert.Empty(t, got[0].media)
	require.Len(t, got[1].media, 1)
	assert.Equal(t, "screen and (min-width: 600px)", s.media[got[1].media[0]].text)
	assert.Equal(t, []string{"base", "theme", anonymous + "2"}, s.layers)
	assert.Equal(t, 2, got[1].layer)
	assert.Equal(t, 3, got[3].layer)
	assert.Zero(t, got[0].layer)
	var said []string
	for _, w := range warns {
		if strings.Contains(w.Msg, "@import") {
			said = append(said, w.Error())
		}
	}
	assert.Equal(t, []string{
		"site.css:10:1: @import after other rules is ignored, as browsers ignore it",
		"site.css:11:17: @import inside a block is ignored, as browsers ignore it",
	}, said)
}

func TestAGamesSheetImportsWithoutLayersOrSupports(t *testing.T) {
	s, warns := Parse("ui/screen.css", `@import "parts/hud.css";
@import url(menu.css) (min-width: 600px);
@import url(theme.css) layer(theme);
@import url(grid.css) supports(display: grid);`)
	require.Len(t, s.Imports(), 2)
	assert.Equal(t, "ui/parts/hud.css", s.Imports()[0].URL, "beside the sheet, as its url()s")
	assert.Equal(t, "ui/menu.css", s.Imports()[1].URL)
	require.Len(t, warns, 2)
	assert.Contains(t, warns[0].Error(), "@import: layer() is read only as a browser reads a sheet")
	assert.Contains(t, warns[1].Error(), "@import: supports() is read only as a browser reads a sheet")
}

func TestOnlyAValidRuleEndsTheImports(t *testing.T) {
	s, _ := Parse("site.css", "@badat-rule foo;\n@import url(after-unknown.css);\n.a ! {}\n@import url(after-bad.css);"+
		"\n@media;\n@page;\n@charset;\n@import url(after-blockless.css);\n@layer base;\n@import url(after-layers.css);",
		Browser())
	require.Len(t, s.Imports(), 4,
		"an unknown at-rule, a rule dropped as badly written and an at-rule without its block are no rules")
	s, _ = Parse("site.css", "@media print {}\n@import url(late.css);", Browser())
	assert.Empty(t, s.Imports(), "an empty block is a rule")
	s, _ = Parse("site.css", "foo {}\n@import url(late.css);", Browser())
	assert.Empty(t, s.Imports(), "a rule naming an element ui has not is a rule all the same")
}

func TestParseWarnsOfAnImportItCannotRead(t *testing.T) {
	for src, want := range map[string]string{
		`@import;`:                          "@import names no sheet",
		`@import 12px;`:                     "@import names no sheet",
		`@import url(a.css) layer(1);`:      `@import: "1" is no layer name`,
		`@import url(a.css) { p { } }`:      "@import takes no { block }",
		`@import url(a.css) (min-width: ;`:  "",
		`@import url(a.css) screen layer;`:  "",
		`@import url(a.css) layer(a, b);`:   `@import: "a, b" is no layer name`,
		`@import url(a.css) supports(x y);`: "",
	} {
		s, warns := Parse("t.css", src, Browser())
		if want == "" {
			continue
		}
		assert.Empty(t, s.Imports(), src)
		require.NotEmpty(t, warns, src)
		assert.Contains(t, warns[0].Error(), want, src)
	}
}

// imported is site's text with each @import's sheet in its place, read
// as a browser reads them.
func imported(t *testing.T, site string, sheets map[string]string) *Sheet {
	t.Helper()
	s, warns := Parse("site.css", site, Browser(), From("https://site.test/site.css"))
	require.Empty(t, warns)
	var got []*Sheet
	for _, imp := range s.Imports() {
		text, ok := sheets[imp.URL]
		if !ok {
			got = append(got, nil)
			continue
		}
		in, inWarns := Parse(imp.URL, text, Browser(), From(imp.URL))
		require.Empty(t, inWarns)
		got = append(got, in)
	}
	return s.Imported(got)
}

func TestImportedRulesStandWhereTheImportIs(t *testing.T) {
	s := imported(t, `@import url(a.css); @import url(gone.css); @import url(b.css); p { width: 3px }`,
		map[string]string{
			"https://site.test/a.css": `p { width: 1px } @media (min-width: 9px) { p { height: 1px } }`,
			"https://site.test/b.css": `p { width: 2px }`,
		})
	var widths []string
	for _, r := range s.Rules {
		widths = append(widths, r.Decls[0].String())
	}
	assert.Equal(t, []string{"width: 1px", "height: 1px", "width: 2px", "width: 3px"}, widths)
	require.Len(t, s.Rules[1].media, 1)
	assert.Equal(t, "(min-width: 9px)", s.media[s.Rules[1].media[0]].text)
	assert.Equal(t, "https://site.test/site.css", s.From())
	assert.Empty(t, s.Imports(), "what is imported is in the sheet")
}

func TestAnImportsMediaAndLayerHoldItsRules(t *testing.T) {
	s := imported(t, `@layer early; @import url(a.css) layer(theme) (min-width: 600px); @layer late { p { width: 4px } }`,
		map[string]string{
			"https://site.test/a.css": `p { width: 1px } @layer inner { p { width: 2px } } @layer { p { width: 3px } }
				@media (min-height: 5px) { p { height: 1px } }`,
		})
	require.Len(t, s.Rules, 5)
	nameless := "theme." + anonymous + "import0-1"
	assert.Equal(t, []string{"early", "theme", "theme.inner", nameless, "late"}, s.layers)
	for i, want := range []string{"theme", "theme.inner", nameless, "theme"} {
		assert.Equal(t, want, s.layers[s.Rules[i].layer-1], "rule %d", i)
	}
	assert.Equal(t, "late", s.layers[s.Rules[4].layer-1])
	narrow := mediaOff([]*Sheet{s}, Media{Width: 500, Height: 500})
	wide := mediaOff([]*Sheet{s}, Media{Width: 700, Height: 500})
	assert.Equal(t, [][]bool{{true, true, true, true, false}}, narrow)
	assert.Equal(t, [][]bool{{false, false, false, false, false}}, wide)
	ranks := layerRanks([]*Sheet{s})[0]
	assert.Greater(t, ranks[0], ranks[1], "early is named first, and loses to theme")
	assert.Greater(t, ranks[1], ranks[4], "theme loses to late")
}

func TestAnImportWithoutALayerNamesItsLayersWhereItStands(t *testing.T) {
	s := imported(t, `@layer a; @import url(x.css); @layer c { p { width: 3px } }`, map[string]string{
		"https://site.test/x.css": `@layer b { p { width: 1px } } @layer a { p { width: 2px } }`,
	})
	assert.Equal(t, []string{"a", "b", "c"}, s.layers)
	assert.Equal(t, "a", s.layers[s.Rules[1].layer-1], "a layer two sheets name is one layer")
}

func TestAnImportedSheetsImportsStandInItsLayer(t *testing.T) {
	inner, _ := Parse("https://site.test/in.css", `p { width: 1px }`, Browser())
	middle, _ := Parse("https://site.test/mid.css", `@import url(in.css) layer(deep); p { width: 2px }`, Browser(),
		From("https://site.test/mid.css"))
	require.Len(t, middle.Imports(), 1)
	middle = middle.Imported([]*Sheet{inner})
	outer, _ := Parse("site.css", `@import url(mid.css) layer(top);`, Browser(), From("https://site.test/site.css"))
	s := outer.Imported([]*Sheet{middle})
	require.Len(t, s.Rules, 2)
	assert.Equal(t, "top.deep", s.layers[s.Rules[0].layer-1])
	assert.Equal(t, "top", s.layers[s.Rules[1].layer-1])
}

func TestAnImportedSheetIsWrittenInItsPlace(t *testing.T) {
	s := imported(t, `@layer early; @import url(a.css) layer(theme) (min-width: 600px); @import url(b.css) layer;
p { width: 3px }`, map[string]string{
		"https://site.test/a.css": `p { width: 1px }`,
		"https://site.test/b.css": `p { width: 2px }`,
	})
	assert.Equal(t, `@layer early, theme;
@layer theme {
@media (min-width: 600px) {
p { width: 1px }
}
}
@layer {
p { width: 2px }
}
@layer early; @import url(a.css) layer(theme) (min-width: 600px); @import url(b.css) layer;
p { width: 3px }`, s.String())
}

func TestJoinKeepsEachSheetsImports(t *testing.T) {
	parts, warns := ParseRules("", `@import url(a.css) (min-width: 9px); p { width: 3px }`, Browser())
	require.Empty(t, warns)
	joined, _ := Join("style", "(min-height: 5px)", parts)
	require.Len(t, joined.Imports(), 1)
	a, _ := Parse("a.css", `p { width: 1px }`, Browser())
	s := joined.Imported([]*Sheet{a})
	require.Len(t, s.Rules, 2)
	assert.Equal(t, "width: 1px", s.Rules[0].Decls[0].String())
	var media []string
	for _, m := range s.Rules[0].media {
		media = append(media, s.media[m].text)
	}
	assert.Equal(t, []string{"(min-height: 5px)", "(min-width: 9px)"}, media)
}
