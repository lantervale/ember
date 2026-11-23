//go:build chrome

package ui_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
)

const faceProse = "Handgloves, AVATAR Type: a face made bold or slanted where its family has no such face."

func TestABoldOrItalicAFamilyLacksIsMadeUpAsInChrome(t *testing.T) {
	d := ui.NewDocument(ui.FetchFonts(foundry(t)), ui.CSS(`
@font-face { font-family: Plain; src: url(go-regular-latin.woff2); }
@font-face { font-family: Weighted; src: url(go-regular-latin.woff2); }
@font-face { font-family: Weighted; src: url(heavy.ttf); font-weight: 700; }
p { margin: 0 0 6px; width: 300px; font: 15px Plain, monospace; }
span { display: inline-block; font: 22px Plain, monospace; }
.b { font-weight: bold; }
.semi { font-weight: 600; }
.medium { font-weight: 500; }
.i { font-style: italic; }
.o { font-style: oblique; }
.none { font-synthesis: none; }
.style-only { font-synthesis: style; }
.weighted { font-family: Weighted, monospace; }`))
	d.Body().Append(
		ui.P(ui.Class("b"), ui.Text(faceProse)),
		ui.P(ui.Class("i"), ui.Text(faceProse)),
		ui.P(ui.Class("b", "i"), ui.Text(faceProse)),
		ui.Div(ui.Span(ui.Class("b"), ui.Text("AVATAR bold"))),
		ui.Div(ui.Span(ui.Class("semi"), ui.Text("AVATAR 600"))),
		ui.Div(ui.Span(ui.Class("medium"), ui.Text("AVATAR 500"))),
		ui.Div(ui.Span(ui.Class("o"), ui.Text("AVATAR oblique"))),
		ui.Div(ui.Span(ui.Class("b", "none"), ui.Text("AVATAR none"))),
		ui.Div(ui.Span(ui.Class("b", "i", "style-only"), ui.Text("AVATAR style only"))),
		ui.Div(ui.Span(ui.Class("b", "weighted"), ui.Text("AVATAR its own bold"))),
	)
	size := geom.V2(600, 700)
	landed(t, d, size)
	r := chrome.Match(t, d, size)
	assert.Empty(t, d.Warnings())
	t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
}

func TestAFacesMetricsOverridesAndSizeAdjustAreChromes(t *testing.T) {
	d := ui.NewDocument(ui.FetchFonts(foundry(t)), ui.CSS(`
@font-face { font-family: Tall; src: url(go-regular-latin.woff2); ascent-override: 120%; descent-override: 40%; }
@font-face { font-family: Gapped; src: url(go-regular-latin.woff2); line-gap-override: 50%; }
@font-face { font-family: Small; src: url(go-regular-latin.woff2); size-adjust: 80%; }
@font-face { font-family: Big; src: url(heavy.ttf); size-adjust: 125.5%; ascent-override: 90%; }
@font-face { font-family: Flat; src: url(go-regular-latin.woff2); ascent-override: normal; descent-override: 0%; }
@font-face { font-family: Plain; src: url(go-regular-latin.woff2); }
div { margin: 0 0 4px; }
span { font-size: 18px; }
p { margin: 0 0 6px; width: 260px; font: 14px Small, monospace; }
.tall { font-family: Tall, monospace; }
.gapped { font-family: Gapped, monospace; }
.small { font-family: Small, monospace; }
.big { font-family: Big, monospace; }
.flat { font-family: Flat, monospace; }
.mixed { font-family: Plain, monospace; }`))
	d.Body().Append(
		ui.Div(ui.Class("tall"), ui.Span(ui.Text("Ascent 120%, descent 40%"))),
		ui.Div(ui.Class("gapped"), ui.Span(ui.Text("A gap of half the size"))),
		ui.Div(ui.Class("small"), ui.Span(ui.Text("Eighty percent of its size"))),
		ui.Div(ui.Class("big"), ui.Span(ui.Text("AVATAR, heavier and larger"))),
		ui.Div(ui.Class("flat"), ui.Span(ui.Text("No descent at all"))),
		ui.Div(ui.Class("mixed"), ui.Span(ui.Text("mono, ")), ui.Span(ui.Class("tall"), ui.Text("tall")),
			ui.Span(ui.Class("small"), ui.Text(" and small"))),
		ui.P(ui.Text(fetchedProse)),
	)
	size := geom.V2(600, 500)
	landed(t, d, size)
	r := chrome.Match(t, d, size)
	assert.Empty(t, d.Warnings())
	t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
}

func TestAFaceIsChosenByItsStretchAsInChrome(t *testing.T) {
	d := ui.NewDocument(ui.FetchFonts(foundry(t)), ui.CSS(`
@font-face { font-family: Wide; src: url(go-regular-latin.woff2); }
@font-face { font-family: Wide; src: url(mono.ttf); font-stretch: condensed; }
@font-face { font-family: Wide; src: url(heavy.ttf); font-stretch: 150%; }
@font-face { font-family: Ranged; src: url(mono.ttf); font-stretch: 50% 80%; }
@font-face { font-family: Ranged; src: url(heavy.ttf); font-stretch: 110% 200%; }
span { display: inline-block; font: 18px Wide, monospace; }
.ranged { font-family: Ranged, monospace; }`))
	for _, stretch := range []string{"normal", "condensed", "semi-condensed", "ultra-condensed", "semi-expanded",
		"expanded", "ultra-expanded", "62.5%", "95%", "105%", "175%"} {
		d.Body().Append(ui.Div(
			ui.Span(ui.Style("font-stretch: "+stretch), ui.Text("AVATAR "+stretch)),
			ui.Span(ui.Class("ranged"), ui.Style("font-stretch: "+stretch), ui.Text(" ranged")),
		))
	}
	d.Body().Append(ui.Div(ui.Span(ui.Style("font: condensed 18px Wide, monospace"), ui.Text("by the shorthand"))))
	size := geom.V2(600, 500)
	landed(t, d, size)
	r := chrome.Match(t, d, size)
	assert.Empty(t, d.Warnings())
	t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
}

func TestAFacesFeatureSettingsTurnItsKerningOffAsInChrome(t *testing.T) {
	d := ui.NewDocument(ui.FetchFonts(foundry(t)), ui.CSS(`
@font-face { font-family: Kerned; src: url(kerned.ttf); }
@font-face { font-family: Unkerned; src: url(kerned.ttf); font-feature-settings: "kern" 0, "liga"; }
@font-face { font-family: Off; src: url(kerned.ttf); font-feature-settings: "kern" off; }
span { display: inline-block; font: 40px Kerned, monospace; }
.unkerned { font-family: Unkerned, monospace; }
.off { font-family: Off, monospace; }`))
	d.Body().Append(
		ui.Div(ui.Span(ui.ID("kerned"), ui.Text("AVAVAV"))),
		ui.Div(ui.Span(ui.ID("unkerned"), ui.Class("unkerned"), ui.Text("AVAVAV"))),
		ui.Div(ui.Span(ui.Class("off"), ui.Text("AVAVAV"))),
	)
	size := geom.V2(600, 300)
	landed(t, d, size)
	r := chrome.Match(t, d, size)
	assert.Empty(t, d.Warnings())
	assert.Greater(t, d.ByID("unkerned").Box().W, d.ByID("kerned").Box().W)
	t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
}

func TestAFaceGoesWithItsSheetAsInChrome(t *testing.T) {
	d := ui.NewDocument(ui.FetchFonts(foundry(t)), ui.CSS(`
span { display: inline-block; font: 20px Heavy, Packed, monospace; }`))
	d.Body().Append(
		ui.Div(ui.Span(ui.ID("heavy"), ui.Text("AVATAR, heavy until its sheet goes"))),
		ui.Div(ui.Span(ui.Style("font-family: Inserted, Packed, monospace"), ui.Text("AVATAR, from a rule inserted alone"))),
		ui.Div(ui.Span(ui.Style("font-family: Disabled, Packed, monospace"), ui.Text("AVATAR, of a disabled sheet"))),
		ui.Div(ui.Span(ui.Style("font-family: Replaced, Packed, monospace"), ui.Text("AVATAR, of a replaced text"))),
	)
	heavy := d.AddSheet(`@font-face { font-family: Heavy; src: url(heavy.ttf); }`)
	d.AddSheet(`@font-face { font-family: Packed; src: url(go-regular-latin.woff); }`)
	rules := d.AddSheet(`p { color: red; }`)
	require.NoError(t, rules.InsertRule(1, `@font-face { font-family: Inserted; src: url(mono.ttf); }`))
	disabled := d.AddSheet(`@font-face { font-family: Disabled; src: url(mono.ttf); }`)
	replaced := d.AddSheet(`@font-face { font-family: Replaced; src: url(heavy.ttf); }`)
	size := geom.V2(600, 300)
	landed(t, d, size)
	wide := d.ByID("heavy").Box().W

	heavy.Remove()
	disabled.SetDisabled(true)
	replaced.Replace(`@font-face { font-family: Replaced; src: url(mono.ttf); }`)
	landed(t, d, size)
	r := chrome.Match(t, d, size)
	assert.Empty(t, d.Warnings())
	assert.Less(t, d.ByID("heavy").Box().W, wide, "Packed, the next family, is lighter")
	t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
}
