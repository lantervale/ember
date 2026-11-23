package ui_test

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/dom"
)

func TestAMarkerIsItsCounterStylesStringOrItsListStyleString(t *testing.T) {
	d := ui.NewDocument(ui.CSS(`
@counter-style paren { system: extends lower-alpha; prefix: "("; suffix: ") "; }
@counter-style dots { system: cyclic; symbols: "◆" "◇"; suffix: " "; }
#paren { list-style-type: paren; }
#dots { list-style-type: dots; }
#star { list-style-type: "★ "; }
#greek { list-style-type: lower-greek; }
#lost { list-style-type: no-such-style; }
`))
	li := func(id string) *ui.Node { return ui.Li(ui.ID(id), ui.Text("x")) }
	items := []*ui.Node{li("paren"), li("dots"), li("star"), li("greek"), li("lost")}
	d.Body().Append(ui.Ol(ui.Attr("start", "2"), items[0], items[1], items[2], items[3], items[4]))
	d.Layout(geom.V2(400, 300))
	var got []string
	for _, n := range items {
		s, _, ok := ui.MarkerText(d, n)
		require.True(t, ok, n.ID())
		got = append(got, s)
	}
	assert.Equal(t, []string{"(b) ", "◆ ", "★ ", "ε. ", "6. "}, got)
	assert.Empty(t, d.Warnings())
}

func TestAListsTypeAttributeNumbersItsItems(t *testing.T) {
	d := ui.NewDocument()
	a, b := ui.Li(ui.Text("x")), ui.Li(ui.Attr("type", "i"), ui.Text("x"))
	list := ui.Ol(ui.Attr("type", "A"), a, b)
	d.Body().Append(list)
	d.Layout(geom.V2(400, 300))
	text := func(n *ui.Node) string { s, _, _ := ui.MarkerText(d, n); return s }
	assert.Equal(t, []string{"A. ", "ii. "}, []string{text(a), text(b)})
	list.SetAttr("type", "a")
	d.Layout(geom.V2(400, 300))
	assert.Equal(t, "a. ", text(a), "the attribute changed")
}

func TestABeforeOrAfterThatIsAListItemHasItsOwnMarker(t *testing.T) {
	d := ui.NewDocument(ui.CSS(`
div { margin-left: 40px; }
#a::before { content: "a"; display: list-item; list-style-type: upper-roman; }
#b::before { content: "b"; display: list-item; }
#b::before::marker { content: "M> "; }
#c::after { content: "c"; display: list-item; list-style-type: "* "; }
#c::after::marker { color: #00f; }
#d::before { content: "d"; display: list-item; }
`))
	a, b, c, e := ui.Div(ui.ID("a")), ui.Div(ui.ID("b")), ui.Div(ui.ID("c")), ui.Div(ui.ID("d"))
	d.Body().Append(a, b, c, e)
	d.Layout(geom.V2(400, 300))
	marker := func(n *ui.Node, after bool) string {
		t.Helper()
		g := dom.Data(n).Before
		if after {
			g = dom.Data(n).After
		}
		require.NotNil(t, g)
		s, _, ok := ui.MarkerText(d, g)
		require.True(t, ok)
		return s
	}
	assert.Equal(t, "0. ", marker(a, false), "a pseudo-element counts no list item: 0, upper-roman's fallback's")
	assert.Equal(t, "M> ", marker(b, false))
	assert.Equal(t, "* ", marker(c, true))
	assert.Equal(t, color.Hex("#00f"), dom.Data(dom.Data(c).After).Marker.Color)
	assert.Empty(t, marker(e, false), "a bullet")
	assert.Empty(t, d.Warnings())
}

func TestAListStyleImageStandsForTheMarkerOnceFound(t *testing.T) {
	d := ui.NewDocument(ui.CSS(`
ul { list-style: square url(dot.png); }
.lost { list-style-image: url(lost.png); }
.own::marker { content: "> "; }
`), ui.Images(fstest.MapFS{"dot.png": {Data: pngOf(20, 8)}}))
	a, lost, own := ui.Li(ui.Text("x")), ui.Li(ui.Class("lost"), ui.Text("x")), ui.Li(ui.Class("own"), ui.Text("x"))
	d.Body().Append(ui.Ul(a, lost, own))
	d.Layout(geom.V2(400, 300))
	text, left, ok := ui.MarkerText(d, a)
	require.True(t, ok)
	assert.Empty(t, text)
	assert.Equal(t, float32(-27), left, "the picture's right edge 7 points before the item's content")
	_, left, ok = ui.MarkerText(d, lost)
	require.True(t, ok, "a picture not found leaves the square")
	assert.Greater(t, left, float32(-27))
	text, _, _ = ui.MarkerText(d, own)
	assert.Equal(t, "> ", text, "the ::marker's content stands in place of the picture")
	assert.Len(t, d.Warnings(), 1, "the picture not found")
}

func TestContentURLReplacesAnElementByItsPicture(t *testing.T) {
	d := ui.NewDocument(ui.CSS(".pic { content: url(pic.png); } .wide { width: 60px; }"),
		ui.Images(fstest.MapFS{"pic.png": {Data: pngOf(30, 20)}}))
	pic, wide := ui.Div(ui.Class("pic"), ui.Text("gone")), ui.Div(ui.Class("pic", "wide"), ui.Text("gone"))
	d.Body().Append(pic, wide)
	d.Layout(geom.V2(400, 300))
	assert.Equal(t, geom.V2(30, 20), geom.V2(pic.Box().W, pic.Box().H))
	assert.Equal(t, geom.V2(60, 40), geom.V2(wide.Box().W, wide.Box().H), "sized as an img, its ratio kept")
	src, ok := ui.PictureOf(pic)
	assert.True(t, ok)
	assert.Equal(t, "pic.png", src)
}
