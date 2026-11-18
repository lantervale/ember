package svg_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/export"
)

const disc = `<circle cx="12" cy="12" r="10" fill="currentColor"/>`

func TestAnSvgElementIsSizedByItsAttributesItsViewBoxAndTheSheet(t *testing.T) {
	d := ui.NewDocument(ui.BrowserSheets(), ui.CSS(`body { margin: 0; font-size: 20px; }
		.row svg { height: 30px; } svg[icon='search'] { width: 20px; height: 20px; } .tall { height: 64px; }`))
	box := ui.Attr("viewBox", "0 0 48 24")
	shaped, sized, relative, none := ui.Svg(disc, box), ui.Svg(disc, ui.Attr("width", "32"), ui.Attr("height", "16")),
		ui.Svg(disc, box, ui.Attr("width", "3em")), ui.Svg(disc)
	ruled, named, classed := ui.Svg(disc, box), ui.Svg(disc, box, ui.Attr("icon", "search")),
		ui.Svg(disc, box, ui.Class("tall"), ui.Attr("height", "10"))
	d.Body().Append(ui.Div(ui.Style("width: 400px"), shaped, sized, relative, none),
		ui.Div(ui.Class("row"), ui.Style("width: 400px"), ruled, named), ui.Div(classed))
	d.Layout(geom.V2(800, 2000))
	assert.Equal(t, geom.V2(400, 200), size(shaped), "a view box alone: the width it is given, by its ratio")
	assert.Equal(t, geom.V2(32, 16), size(sized))
	assert.Equal(t, geom.V2(60, 30), size(relative), "a width in em, the height by the view box")
	assert.Equal(t, geom.V2(300, 150), size(none))
	assert.Equal(t, geom.V2(60, 30), size(ruled), "a rule naming svg")
	assert.Equal(t, geom.V2(20, 20), size(named), "a rule naming one of its attributes")
	assert.Equal(t, geom.V2(128, 64), size(classed), "a rule wins over the attribute")
	assert.Empty(t, d.Warnings())
}

func TestAnSvgElementIsDrawnInTheColourOfItsTextAndAgainWhenItsMarkupChanges(t *testing.T) {
	d := ui.NewDocument(ui.CSS(`body { color: #ff0000; } .blue { color: rgb(0 0 255 / 50%); }`))
	icon := ui.Svg(disc, ui.Attr("viewBox", "0 0 24 24"), ui.Attr("width", "24"))
	d.Body().Append(icon)
	d.Layout(geom.V2(200, 200))
	picture := func() string {
		address, ok := ui.PictureOf(icon)
		require.True(t, ok)
		text, err := url.PathUnescape(address)
		require.NoError(t, err)
		return text
	}
	src := picture()
	assert.Contains(t, src, `color="rgba(255,0,0,1)"`)
	assert.Contains(t, src, `viewBox="0 0 24 24"`)
	assert.NotContains(t, src, `width=`, "its size is the element's, not the picture's")
	assert.Equal(t, disc, icon.Markup())

	icon.AddClass("blue")
	icon.SetMarkup(`<rect width="24" height="24"/>`)
	d.Layout(geom.V2(200, 200))
	src = picture()
	assert.Contains(t, src, `color="rgba(0,0,255,0.5)"`)
	assert.Contains(t, src, `<rect width="24" height="24"/>`)
	icon.SetAttr("viewBox", "0 0 12 12")
	assert.Contains(t, picture(), `viewBox="0 0 12 12"`)
	assert.Empty(t, d.Warnings())
}

func TestAnSvgElementIsExportedWithItsMarkup(t *testing.T) {
	d := ui.NewDocument()
	d.Body().Append(ui.Svg(disc, ui.Attr("viewBox", "0 0 24 24")))
	d.Layout(geom.V2(200, 200))
	var page strings.Builder
	require.NoError(t, export.Write(&page, d))
	assert.Contains(t, page.String(), `<svg viewBox="0 0 24 24">`+disc+`</svg>`)
}
