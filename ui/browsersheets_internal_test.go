package ui

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/dom"
)

const browserPage = `
@layer reset, site;
@layer site { .card { width: 100px; padding: 10px; -webkit-transform: none; -moz-box-sizing: border-box; } }
@layer reset { #card { width: 40px; } }
@supports (display: grid) { .card { height: 30px; } }
@supports (display: -ms-grid) { .card { height: 99px; } }
button::-moz-focus-inner { border: 0; }
`

func TestBrowserSheetsReadsAPagesSheetAsABrowserDoes(t *testing.T) {
	card := Div(ID("card"), Class("card"), Style("-webkit-box-shadow: none; -o-transition: none"))
	button := Button(Style("width: 80px; padding: 10px"), Text("Go"))
	d := NewDocument(CSS(browserPage), BrowserSheets())
	d.Body().Append(card, button)
	d.Layout(geom.V2(400, 300))

	assert.True(t, d.BrowserSheets())
	assert.Empty(t, d.Warnings())
	assert.Equal(t, geom.V2(120, 50), geom.V2(card.Box().W, card.Box().H),
		"the later layer's width, the content's: padding adds to it")
	assert.InDelta(t, 80, button.Box().W, 0.01, "a button's width is its border box's, as in a browser")
}

func TestWithoutBrowserSheetsAGamesSheetKeepsItsWarnings(t *testing.T) {
	card := Div(ID("card"), Class("card"), Style("-webkit-box-shadow: none"))
	d := NewDocument(CSS(browserPage))
	d.Body().Append(card)
	d.Layout(geom.V2(400, 300))

	assert.False(t, d.BrowserSheets())
	var warns []string
	for _, w := range d.Warnings() {
		warns = append(warns, w.Error())
	}
	assert.Equal(t, []string{
		"style:2:1: @layer is not supported",
		"style:3:1: @layer is not supported",
		"style:4:1: @layer is not supported",
		"style:5:1: @supports is not supported",
		"style:6:1: @supports is not supported",
		"style:7:9: the vendor-prefixed pseudo-element ::-moz-focus-inner is not supported; rule dropped",
		`<div#card.card> style:1:1: vendor-prefixed property "-webkit-box-shadow" is not supported`,
	}, warns)
	assert.NotEqual(t, float32(120), card.Box().W)
}

func TestBrowserSheetsReadsEverySheetWhereverItStands(t *testing.T) {
	fsys := fstest.MapFS{"site.css": {Data: []byte(".card { -webkit-box-sizing: border-box; -moz-foo: 1; }")}}
	for name, opts := range map[string][]DocumentOption{
		"first": {BrowserSheets(), Stylesheet(fsys, "site.css"), CSS(browserPage)},
		"last":  {Stylesheet(fsys, "site.css"), CSS(browserPage), BrowserSheets()},
	} {
		d := NewDocument(opts...)
		assert.Empty(t, d.Warnings(), name)
		require.Len(t, d.Sheets(), 2, name)
		assert.Equal(t, "site.css", d.Sheets()[0].Name, "%s: the sheets keep their order", name)
		assert.True(t, d.Sheets()[0].Browser() && d.Sheets()[1].Browser(), name)
	}
}

func TestASheetsURLIsFetchedFromTheSheetsAddress(t *testing.T) {
	a := &asked{pictures: map[string][]byte{"https://cdn.test/site/img/wall.png": picture(t, 8, 8)}}
	d := NewDocument(FetchImages(a.fetch),
		CSS(`body { background-image: url(../img/wall.png); }`, From("https://cdn.test/site/css/main.css")),
		CSS(`p { background-image: url(img/own.png); }`))
	p := P(Style("background-image: url(inline.png)"))
	d.Body().Append(p)
	d.Layout(geom.V2(200, 100))

	assert.Equal(t, "https://cdn.test/site/css/main.css", d.Sheets()[0].From())
	assert.Empty(t, d.Sheets()[1].From())
	wall := dom.Data(d.Body()).Style.BackgroundLayer(0).Image.Source
	assert.Equal(t, "https://cdn.test/site/img/wall.png", wall, "resolved from the sheet's address")
	assert.Equal(t, "inline.png", dom.Data(p).Style.BackgroundLayer(0).Image.Source,
		"a style attribute's is the page's to resolve")
	d.image(wall)
	d.fetching.Wait()
	assert.Equal(t, []string{"https://cdn.test/site/img/wall.png"}, a.urls)
}

func TestMediaRulesFollowTheDocumentsViewport(t *testing.T) {
	d := NewDocument(CSS(`
.side { width: 200px; height: 10px; }
@media (max-width: 600px) { .side { width: 100%; } }
@media (orientation: portrait) { .side { height: 20px; } }
`))
	side, other := Div(Class("side")), Div(Style("height: 5px"))
	d.Body().Append(side, other)

	d.Layout(geom.V2(800, 400))
	assert.Empty(t, d.Warnings())
	assert.Equal(t, geom.V2(200, 10), geom.V2(side.Box().W, side.Box().H))

	d.Layout(geom.V2(700, 400))
	assert.Equal(t, geom.V2(200, 10), geom.V2(side.Box().W, side.Box().H), "no boundary crossed")

	d.Layout(geom.V2(500, 400))
	assert.Equal(t, geom.V2(484, 10), geom.V2(side.Box().W, side.Box().H), "under 600 wide it fills the body")

	d.Layout(geom.V2(500, 900))
	assert.Equal(t, geom.V2(484, 20), geom.V2(side.Box().W, side.Box().H), "taller than wide")

	d.Layout(geom.V2(800, 400))
	assert.Equal(t, geom.V2(200, 10), geom.V2(side.Box().W, side.Box().H), "and back")
}

func TestRulesUnderWhatChromeAsksTheMachineFollowFixedAnswers(t *testing.T) {
	d := NewDocument(CSS(`
div { height: 5px; }
@media (hover: hover) and (pointer: fine) { .mouse { height: 12px; } }
@media (hover: none), (pointer: coarse) { .touch { height: 12px; } }
@media (prefers-color-scheme: dark) { .dark { height: 12px; } }
@media (prefers-reduced-motion: reduce) { .still { height: 12px; } }
`))
	mouse, touch, dark, still := Div(Class("mouse")), Div(Class("touch")), Div(Class("dark")), Div(Class("still"))
	d.Body().Append(mouse, touch, dark, still)
	d.Layout(geom.V2(800, 400))
	assert.Empty(t, d.Warnings())
	assert.InDelta(t, 12, mouse.Box().H, 0.01, "a mouse, whatever the machine has")
	assert.InDelta(t, 5, touch.Box().H, 0.01, "never a touch screen")
	assert.InDelta(t, 5, dark.Box().H, 0.01, "a light scheme")
	assert.InDelta(t, 5, still.Box().H, 0.01, "no preference")
}

func TestMediaWidthsAreTheDocumentsOwnPixels(t *testing.T) {
	d := NewDocument(Zoom(2), CSS(`.side { width: 200px; } @media (max-width: 600px) { .side { width: 50px; } }`))
	side := Div(Class("side"))
	d.Body().Append(side)
	// A window 1000 points wide shows a Document zoomed twice 500 of its
	// pixels wide: Draw lays it out at that size.
	d.Layout(geom.V2(1000, 400).Scale(1 / d.Zoom()))
	assert.InDelta(t, 50, side.Box().W, 0.01)
}
