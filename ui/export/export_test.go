package export

import (
	"encoding/base64"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/lantervale/ember/geom"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/css"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func written(t *testing.T, d *ui.Document, opts ...Option) string {
	var b strings.Builder
	require.NoError(t, Write(&b, d, opts...))
	return b.String()
}

func TestWritesTheTreeWithItsAttributes(t *testing.T) {
	d := ui.NewDocument()
	d.Body().Append(ui.Div(ui.ID("a"), ui.Class("x", "y"), ui.Style("width: 10px"),
		ui.P(ui.Text("1 < 2 & \"q\"")),
		ui.Input(ui.Type("text"), ui.Value("v")),
	))
	got := written(t, d)
	assert.Contains(t, got,
		`<body><div id="a" class="x y" style="width: 10px"><p>1 &lt; 2 &amp; &#34;q&#34;</p>`+
			`<input type="text" value="v"></div></body>`)
	assert.NotContains(t, got, "</input>")
}

func TestWritesATablesCaptionAndFooter(t *testing.T) {
	d := ui.NewDocument()
	d.Body().Append(ui.Table(ui.Caption(ui.Text("Scores")), ui.Tbody(ui.Tr(ui.Td(ui.Text("12")))),
		ui.Tfoot(ui.Tr(ui.Td(ui.Text("12"))))))
	assert.Contains(t, written(t, d),
		`<table><caption>Scores</caption><tbody><tr><td>12</td></tr></tbody><tfoot><tr><td>12</td></tr></tfoot></table>`)
}

func TestThePageCarriesItsFontsAndDefaults(t *testing.T) {
	got := written(t, ui.NewDocument(), Head("<script>x</script>"))
	assert.Contains(t, got,
		`@font-face { font-family: "Go"; font-weight: 700; font-style: italic; src: url(data:font/ttf;base64,`)
	assert.Contains(t, got, `font-family: "Go Mono"`)
	assert.Contains(t, got, "box-sizing: border-box")
	assert.Contains(t, got, "::-webkit-scrollbar { width: 15px; height: 15px; }")
	assert.Less(t, strings.Index(got, "<script>x</script>"), strings.Index(got, "</head>"))
}

func TestOverlayScrollbarsAreWrittenTakingNoRoom(t *testing.T) {
	got := written(t, ui.NewDocument(ui.OverlayScrollbars()))
	assert.Contains(t, got, "::-webkit-scrollbar { width: 0; height: 0; }")
	assert.NotContains(t, got, "15px; height: 15px")
}

func TestControlsAreWrittenAsTheyAreNow(t *testing.T) {
	d := ui.NewDocument()
	name := ui.Input(ui.Value("old"))
	solid := ui.Input(ui.Type("checkbox"))
	shadow := ui.Select(ui.Placeholder("cast (default)"), ui.Option(ui.Text("cast")), ui.Option(ui.Text("none")))
	shape := ui.Select(ui.Option(ui.Text("rectangle"), ui.Selected()), ui.Option(ui.Text("ellipse")))
	d.Body().Append(name, solid, shadow, shape)
	d.SetValue(name, "Heron")
	d.SetChecked(solid, true)
	d.SetSelected(shape, 1)
	got := written(t, d)
	assert.Contains(t, got, `<input value="Heron">`)
	assert.Contains(t, got, `<input type="checkbox" checked="">`)
	assert.Contains(t, got,
		`<select style="color: #757575"><option value="" disabled selected hidden data-ember-implied="">`+
			`cast (default)</option><option>cast</option><option>none</option></select>`)
	assert.Contains(t, got, `<select><option>rectangle</option><option selected="">ellipse</option></select>`)
}

func TestImagesTravelInThePageAsDataURLs(t *testing.T) {
	png := css.DefaultImages()["ember:select-arrow"]
	fsys := fstest.MapFS{
		"ui/tool.css":      {Data: []byte(".panel { background-image: url(img/panel.png); }")},
		"ui/img/panel.png": {Data: png},
	}
	d := ui.NewDocument(ui.Stylesheet(fsys, "ui/tool.css"))
	d.Body().Append(ui.Div(ui.Class("panel")))
	got := written(t, d)
	data := "url(data:image/png;base64," + base64.StdEncoding.EncodeToString(png) + ")"
	assert.Contains(t, got, ".panel { background-image: "+data)
	assert.Contains(t, got, "background-image: "+data, "the select's arrow")
	assert.NotContains(t, got, "url(ember:")
	assert.Empty(t, d.Warnings())
}

func TestASelectShowingItsPlaceholderTakesThePlaceholderColour(t *testing.T) {
	d := ui.NewDocument()
	d.Body().Append(ui.Select(ui.Placeholder("cast (default)"), ui.Option(ui.Text("cast"))),
		ui.Select(ui.Placeholder("p"), ui.Option(ui.Text("a"), ui.Selected())))
	got := written(t, d)
	assert.Contains(t, got,
		`<select style="color: #757575"><option value="" disabled selected hidden data-ember-implied="">`+
			`cast (default)</option>`)
	assert.Contains(t, got, `<select><option selected="">a</option></select>`)
}

func TestStatesTravelForChrome(t *testing.T) {
	sheet := "button:hover { color: #f00; } span:focus-visible, p:focus { color: #0f0; }"
	fsys := fstest.MapFS{"a.css": {Data: []byte(sheet)}}
	d := ui.NewDocument(ui.Stylesheet(fsys, "a.css"))
	b := ui.Button(ui.Text("x"))
	d.Body().Append(b)
	b.SetState(dom.Hover|dom.FocusVisible, true)
	got := written(t, d)
	assert.Contains(t, got, `<button data-ember-hover="" data-ember-focus-visible="">`)
	assert.Contains(t, got, `button:is(:hover, [data-ember-hover])`)
	assert.Contains(t, got, `:is(:focus-visible, [data-ember-focus-visible])`)
	assert.Contains(t, got, `:is(:focus, [data-ember-focus])`)
}

func TestTheTargetTheFocusWithinAndTheLanguageTravelForChrome(t *testing.T) {
	sheet := "form:focus-within p, :target { color: #f00; }"
	fsys := fstest.MapFS{"a.css": {Data: []byte(sheet)}}
	d := ui.NewDocument(ui.Stylesheet(fsys, "a.css"), ui.Language("fr-BE"))
	h := ui.H2(ui.Text("x"))
	f := ui.Form(h)
	d.Body().Append(f)
	d.SetTarget(h)
	f.SetState(dom.FocusWithin, true)
	got := written(t, d)
	assert.Contains(t, got, `<form data-ember-focus-within="">`)
	assert.Contains(t, got, `<h2 data-ember-target="">`)
	assert.Contains(t, got, `form:is(:focus-within, [data-ember-focus-within]) p, :is(:target, [data-ember-target])`)
	assert.Contains(t, got, `<meta http-equiv="content-language" content="fr-BE">`)
}

func TestAnImgSourceTravelsAsADataURL(t *testing.T) {
	png := css.DefaultImages()["ember:select-arrow"]
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"s.css": {Data: []byte("")}, "tile.png": {Data: png}}, "s.css"))
	d.Body().Append(ui.Img(ui.Attr("src", "tile.png")), ui.Img(ui.Attr("src", "missing.png")))
	got := written(t, d)
	assert.Contains(t, got, `src="data:image/png;base64,`+base64.StdEncoding.EncodeToString(png)+`"`)
	assert.Contains(t, got, `src="missing.png"`, "an unknown source is kept as written")
}

func TestAModalDialogOpensModalInChrome(t *testing.T) {
	d := ui.NewDocument()
	modal, plain, over := ui.Dialog(ui.Text("Delete?")), ui.Dialog(ui.Text("Note")), ui.Dialog(ui.Text("Sure?"))
	d.Body().Append(over, modal, plain)
	d.ShowModal(modal)
	d.ShowModal(over)
	d.Show(plain)
	got := written(t, d)
	assert.Less(t, layer(t, got, "Delete?"), layer(t, got, "Sure?"), "opened after, though first in the tree")
	assert.Contains(t, got, `<dialog open="">Note</dialog>`)
	assert.Contains(t, got, "dialog::backdrop")
	assert.NotContains(t, got, "rgba(0, 0, 0, 0.4)", "the user-agent sheet's backdrop, not one of export's own")
	assert.Contains(t, got, "@layer ember-user-agent {",
		"the user-agent sheet in a layer, so the page's rules win whatever their specificity, as over Chrome's own")
	assert.Contains(t, got, "d.showModal()")
}

func TestAModalShownAgainOpensOnTopInChrome(t *testing.T) {
	d := ui.NewDocument()
	first, second := ui.Dialog(ui.Text("first")), ui.Dialog(ui.Text("second"))
	d.Body().Append(first, second)
	d.ShowModal(first)
	d.ShowModal(second)
	d.Close(first, "")
	d.ShowModal(first)
	got := written(t, d)
	assert.Greater(t, layer(t, got, "first"), layer(t, got, "second"), "closed and shown modal again, the highest")
}

// layer is the data-ember-modal value of the dialog holding text in page.
func layer(t *testing.T, page, text string) uint64 {
	t.Helper()
	re := regexp.MustCompile(`<dialog data-ember-modal="(\d+)">` + regexp.QuoteMeta(text) + `</dialog>`)
	m := re.FindStringSubmatch(page)
	require.NotNil(t, m, "a modal dialog holding %q", text)
	n, err := strconv.ParseUint(m[1], 10, 64)
	require.NoError(t, err)
	return n
}

func TestTruncateStartCutsTheStartInChrome(t *testing.T) {
	d := ui.NewDocument()
	d.Body().Append(ui.P(ui.Class("truncate-start"), ui.Text("/assets/"), ui.Strong(ui.Text("crypt"))))
	got := written(t, d)
	assert.Contains(t, got,
		`<p class="truncate-start"><bdi dir="ltr" data-ember-implied="">/assets/<strong>crypt</strong></bdi></p>`,
		"its text kept left to right inside")
	assert.Contains(t, got, ".truncate-start { direction: rtl; text-align: left; }",
		"so Chrome's ellipsis takes the start")
}

// svgOf decodes the first base64 SVG in s.
func svgOf(t *testing.T, s string) string {
	const prefix = "data:image/svg+xml;base64,"
	i := strings.Index(s, prefix)
	require.GreaterOrEqual(t, i, 0, "an SVG in %s", s)
	rest := s[i+len(prefix):]
	raw, err := base64.StdEncoding.DecodeString(rest[:strings.IndexAny(rest, ")\"")])
	require.NoError(t, err)
	return string(raw)
}

func TestARangeIsDrawnForChromeAsEmbersConventionHasIt(t *testing.T) {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"a.css": {Data: []byte(
		`input { width: 100px; height: 20px; background-color: #000000; color: #ffffff; ` +
			`accent-color: #ff0000; }`)}}, "a.css"))
	r := ui.Input(ui.Type("range"), ui.Value("25"))
	d.Body().Append(r)
	d.Layout(geom.V2(400, 300))
	got := written(t, d)
	assert.Contains(t, got, ":where(input[type=\"range\"]) { appearance: none")
	assert.Contains(t, got, "border-image-outset: 0 7px")
	svg := svgOf(t, got)
	assert.Contains(t, svg, `width="114" height="20"`, "the box and the thumb's overhang each side")
	assert.Contains(t, svg, `<rect x="7" y="7" width="100" height="6" rx="3" fill="#000000"`, "the track")
	assert.Contains(t, svg, `<rect x="7" y="7" width="25" height="6" rx="3" fill="#ff0000"`, "the fill to the value")
	assert.Contains(t, svg, `<circle cx="32" cy="10" r="7" fill="#ffffff"`, "the thumb on the value")

	r.SetState(dom.Hover, true)
	assert.Contains(t, svgOf(t, written(t, d)), `<circle cx="32" cy="10" r="7" fill="#ff0000"`, "hovered: the accent")
}

func TestAProgressIsItsTrackAndAccentFillForChrome(t *testing.T) {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"a.css": {Data: []byte(
		`progress { width: 200px; height: 8px; background-color: #000000; accent-color: #ff0000; }`)}}, "a.css"))
	d.Body().Append(ui.Progress(ui.Attr("value", "30"), ui.Attr("max", "120")))
	d.Layout(geom.V2(400, 300))
	got := written(t, d)
	assert.Contains(t, got, ":where(progress)::-webkit-progress-bar")
	assert.Contains(t, got, "no-repeat content-box", "a background, so its border still shows")
	svg := svgOf(t, got)
	assert.Contains(t, svg, `<rect x="0" y="0" width="200" height="8" fill="#000000"`)
	assert.Contains(t, svg, `<rect x="0" y="0" width="50" height="8" fill="#ff0000"`)
}

func TestAScrolledBoxScrollsTheSameInChrome(t *testing.T) {
	d := ui.NewDocument()
	sc := ui.Div(ui.Style("height: 50px; overflow: auto"), ui.Div(ui.Style("height: 500px")))
	d.Body().Append(sc)
	dom.Data(sc).Scroll = geom.V2(0, 137.5)
	got := written(t, d)
	assert.Contains(t, got, `data-ember-scroll="0 137.5"`)
	assert.Contains(t, got, "e.scrollTop = y")
}

func TestAScrollingViewportIsWrittenForChromeToScroll(t *testing.T) {
	d := ui.NewDocument(ui.ScrollingViewport())
	d.Body().Append(ui.Div(ui.Style("height: 5000px")))
	dom.Data(d.HTML()).Scroll = geom.V2(0, 420)
	got := written(t, d)
	assert.Contains(t, got, `<html data-ember-scroll="0 420" `+ScrollingViewport+`>`)
	assert.NotContains(t, got, "html::-webkit-scrollbar", "the page shows its bar")
	assert.Contains(t, got, "::-webkit-scrollbar { width: 15px; height: 15px; }")

	still := written(t, ui.NewDocument())
	assert.Contains(t, still, "html::-webkit-scrollbar { display: none; }", "a window's Document shows none")
	assert.NotContains(t, still, ScrollingViewport)
}

type words map[string]string

func (m words) Tr(s string) string {
	if v, ok := m[s]; ok {
		return v
	}
	return s
}

func TestThePageCarriesWhatTheTranslatorSays(t *testing.T) {
	d := ui.NewDocument(ui.WithTranslator(words{"Save": "Enregistrer", "Search": "Rechercher", "Pick": "Choisir"}))
	d.Body().Append(
		ui.Button(ui.Text("Save")),
		ui.P(ui.Attr("translate", "no"), ui.Text("Save")),
		ui.Input(ui.Type("text"), ui.Attr("placeholder", "Search")),
		ui.Input(ui.Type("text"), ui.Value("Save")),
		ui.Select(ui.Attr("placeholder", "Pick"), ui.Option(ui.Text("Save"))),
	)
	d.Layout(geom.V2(400, 300))
	got := written(t, d)
	assert.Contains(t, got, "<button>Enregistrer</button>")
	assert.Contains(t, got, `<p translate="no">Save</p>`)
	assert.Contains(t, got, `placeholder="Rechercher"`)
	assert.Contains(t, got, `value="Save"`)
	assert.Contains(t, got, `hidden data-ember-implied="">Choisir</option><option>Enregistrer</option>`)
}

func TestACanvasCarriesItsSnapshotToChrome(t *testing.T) {
	d := ui.NewDocument()
	shot, bare := ui.Canvas(nil, ui.Attr("width", "40")), ui.Canvas(nil)
	d.Body().Append(shot, bare)
	got := written(t, d, Snapshots(func(n *ui.Node) ([]byte, bool) { return []byte("png"), n == shot }))
	assert.Contains(t, got,
		`<canvas width="40" data-ember-snapshot="data:image/png;base64,`+
			base64.StdEncoding.EncodeToString([]byte("png"))+`"></canvas>`)
	assert.Contains(t, got, `<canvas></canvas>`, "none without a snapshot")
	assert.Contains(t, got, "emberSnapshot", "and the script that draws them")
}

func TestACheckedRadioIsWrittenChecked(t *testing.T) {
	d := ui.NewDocument()
	on, off := ui.Input(ui.Type("radio"), ui.Attr("name", "mode"),
		ui.Attr("checked", "")), ui.Input(ui.Type("radio"), ui.Attr("name", "mode"))
	d.Body().Append(on, off)
	got := written(t, d)
	assert.Contains(t, got, `<input type="radio" name="mode" checked="">`)
	assert.Contains(t, got, `<input type="radio" name="mode">`)
}

func TestMediaRulesTravelForChromeToAnswer(t *testing.T) {
	d := ui.NewDocument(ui.CSS(`p { color: #100; } @media (max-width: 600px) { p { color: #200; colour: red; } }`))
	got := written(t, d)
	assert.Contains(t, got, "p { color: #100; }\n@media (max-width: 600px) { p { color: #200; } }\n")
}

func TestABrowsersSheetTravelsAsItWasWritten(t *testing.T) {
	site := `@layer base { p { -webkit-transform: rotate(3deg); -moz-transform: none; width: 100px } }
@supports (display: grid) { p:hover { color: red } }`
	d := ui.NewDocument(ui.BrowserSheets(), ui.CSS(site))
	got := written(t, d)
	assert.Contains(t, got, "@layer base { p { -webkit-transform: rotate(3deg); -moz-transform: none; width: 100px } }",
		"Chrome reads the sheet itself, not what ui made of it")
	assert.Contains(t, got, "p:is(:hover, [data-ember-hover]) { color: red }", "states still travel")
	agent := got[strings.Index(got, "@layer ember-user-agent {"):strings.Index(got, ".truncate-start")]
	assert.Contains(t, agent, "* { box-sizing: border-box; }")
	assert.Contains(t, agent, "* { box-sizing: content-box; }\nbutton, select, table,",
		"the user-agent layer ends as a browser's own sheet")
	assert.NotContains(t, written(t, ui.NewDocument(ui.CSS(site))), "box-sizing: content-box")
}

func TestASheetsURLsTravelFromItsAddress(t *testing.T) {
	d := ui.NewDocument(ui.BrowserSheets(),
		ui.CSS(`p { background-image: url(../img/wall.png); }`, ui.From("https://cdn.test/site/css/main.css")),
		ui.CSS(`a { background-image: url("door.png"); }`, ui.From("https://cdn.test/site/css/main.css")))
	got := written(t, d)
	assert.Contains(t, got, "url(https://cdn.test/site/img/wall.png)")
	assert.Contains(t, got, "url(https://cdn.test/site/css/door.png)")
}

func TestColumnsTravelForChromeToLayOut(t *testing.T) {
	d := ui.NewDocument(ui.CSS(`.m { columns: 2 120px; column-gap: 2em; column-rule: 1px solid #888; column-fill: auto; }
p { break-inside: avoid; orphans: 3; widows: 3; page-break-inside: avoid; break-before: column; }`))
	d.Body().Append(ui.Div(ui.Style("column-count: 3")))
	got := written(t, d)
	assert.Contains(t, got,
		".m { columns: 2 120px; column-gap: 2em; column-rule: 1px solid #888; column-fill: auto; }")
	assert.Contains(t, got, "p { break-inside: avoid; orphans: 3; widows: 3; break-inside: avoid; break-before: column; }",
		"the old page-break-inside under its name of today")
	assert.Contains(t, got, `<div style="column-count: 3">`)
}

func TestPseudoElementRulesTravelAsWritten(t *testing.T) {
	d := ui.NewDocument(ui.CSS(`li:nth-child(2)::before { content: counter(n) " " attr(title); }
li::marker { color: red; }`))
	d.Body().Append(ui.Ul(ui.Li(ui.Text("a")), ui.Li(ui.Text("b"))))
	got := written(t, d)
	assert.Contains(t, got, `li:nth-child(2)::before { content: counter(n) " " attr(title); }`)
	assert.Contains(t, got, "li::marker { color: red; }")
	assert.Contains(t, got, "::before, ::after { box-sizing: border-box; }")
	assert.Contains(t, got, "<ul><li>a</li><li>b</li></ul>", "generated boxes are Chrome's to make")
	assert.NotContains(t, written(t, ui.NewDocument(ui.BrowserSheets())), "::before, ::after { box-sizing")
}

func TestTheSelectionTravelsForChromeToSelect(t *testing.T) {
	d := ui.NewDocument(ui.CSS("body { user-select: text; }"))
	first := ui.P(ui.Text("one "), ui.Text("twó three"), ui.B(ui.Text("bold")), ui.Text("after"))
	second := ui.P(ui.Text("second"))
	d.Body().Append(first, second)
	assert.NotContains(t, written(t, d), "data-ember-anchor", "nothing selected, nothing written")

	d.Select(ui.Position{Node: first.Children()[1], Offset: 5}, ui.Position{Node: first.Children()[3], Offset: 2})
	got := written(t, d)
	assert.Contains(t, got, `<p data-ember-anchor="0 9" data-ember-focus="2 2">`,
		"text nodes side by side are one in Chrome, and its offsets count from that one's start")
	assert.Contains(t, got, "getSelection().setBaseAndExtent")

	d.Select(ui.Position{Node: second.Children()[0], Offset: 3}, ui.Position{Node: first.Children()[0], Offset: 1})
	got = written(t, d)
	assert.Contains(t, got, `<p data-ember-focus="0 1">`)
	assert.Contains(t, got, `<p data-ember-anchor="0 3">`, "backwards as it was made")
}
