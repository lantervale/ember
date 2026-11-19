//go:build chrome

package ui_test

import (
	"fmt"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
)

func TestGeneratedBoxesOfEveryDisplayMatchChrome(t *testing.T) {
	d, _ := generatedDoc()
	r := chrome.Match(t, d, geom.V2(400, 600))
	assert.Equal(t, 22+9, r.Boxes, "html, body, ten divs and spans, and the nine boxes generated")
	t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
}

// The elements Chrome gives no ::before box, and those ui draws whole or
// lays out as table parts, take none here; an hr, a button, a details and
// its summary, a table's cell and caption do. The fields are blocks of a
// size the sheet gives: left to themselves they are as wide as Chrome's
// system font makes them, which changes with the machine.
func TestGeneratedBoxesStandOnTheElementsChromeGivesThem(t *testing.T) {
	d := ui.NewDocument(ui.CSS(`
body { margin: 0; }
input, textarea, select { display: block; width: 120px; height: 24px; }
img::before, br::before, input::before, textarea::before, select::before, canvas::before { content: "x"; }
hr::before, button::before, details::after, td::before, caption::after, li::before,
legend::before, fieldset::after, a::before, dialog::before, label::after {
  content: "+"; display: inline-block; width: 12px; height: 12px; }
`))
	d.Body().Append(
		ui.Img(ui.Style("width: 20px; height: 20px")), ui.Div(ui.Text("a"), ui.Br(), ui.Text("b")),
		ui.Input(), ui.Textarea(), ui.Select(ui.Option(ui.Text("one"))), ui.Canvas(func(*twod.Canvas) {}),
		ui.Hr(), ui.Button(ui.Text("go")),
		ui.Details(ui.Attr("open", ""), ui.Summary(ui.Text("more")), ui.P(ui.Text("less"))),
		ui.Table(ui.Caption(ui.Text("cap")), ui.Tr(ui.Td(ui.Text("cell")))),
		ui.Ul(ui.Li(ui.Text("item"))), ui.Fieldset(ui.Legend(ui.Text("legend")), ui.Label(ui.Text("label"))),
		ui.P(ui.A(ui.Attr("href", "#"), ui.Text("link"))),
	)
	r := chrome.Match(t, d, geom.V2(600, 800))
	t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
}

func TestGeneratedBoxesAreSizedAsChromesInBothSheets(t *testing.T) {
	const sheet = `body { margin: 0; } div { position: relative; padding: 6px; }
.a::before { content: ""; position: absolute; top: 0; right: 0; width: 28px; height: 28px; padding: 2px 6px; }
.b::before { content: "x"; display: block; width: 50%; padding: 4px; border: 2px solid; }
.c::after { content: ""; display: inline-block; width: 28px; height: 28px; padding: 2px 6px; }`
	for _, opts := range [][]ui.DocumentOption{nil, {ui.BrowserSheets()}} {
		d := ui.NewDocument(append(opts, ui.CSS(sheet))...)
		d.Body().Append(ui.Div(ui.Class("a"), ui.Text("x")), ui.Div(ui.Class("b"), ui.Text("x")),
			ui.Div(ui.Class("c"), ui.Text("x")))
		chrome.Match(t, d, geom.V2(600, 400))
	}
}

func TestCountersListsAndQuotesMatchChrome(t *testing.T) {
	d := ui.NewDocument(ui.CSS(`
body { margin: 8px; font-size: 16px; counter-reset: chapter; }
h2 { counter-increment: chapter; counter-reset: section; margin: 4px 0; font-size: 20px; }
h2::before { content: "Chapter " counter(chapter, upper-roman) ": "; }
h3 { counter-increment: section; margin: 2px 0; font-size: 16px; }
h3::before { content: counter(chapter) "." counter(section, decimal-leading-zero) " "; }
h3::after { content: " [" attr(data-tag) "]"; }
.toc { counter-reset: item; list-style-type: none; padding-left: 0; margin: 4px 0; }
.toc li { counter-increment: item; }
.toc li::before { content: counters(item, ".") " — "; display: inline-block; }
.toc .toc { padding-left: 20px; }
.set { counter-reset: n 4; }
.set span { counter-increment: n 3; }
.set span::after {
	content: counter(n) counter(n, lower-alpha) counter(n, upper-alpha) counter(n, lower-roman) counter(none-such) ",";
}
.set .jump { counter-set: n 40; }
ol, ul { margin: 4px 0; }
.alpha { list-style-type: lower-alpha; }
.fr { quotes: "«" "»" "‹" "›"; }
.noq { quotes: none; }
q { display: inline-block; }
blockquote { margin: 4px 0; }
blockquote::before { content: open-quote; }
blockquote::after { content: no-close-quote; }
.image::before { content: url(dot.png) " then " url(dot.png); }
`), ui.Images(fstest.MapFS{"dot.png": {Data: pngOf(12, 9)}}))
	toc := func(args ...ui.Arg) *ui.Node { return ui.Ol(append([]ui.Arg{ui.Class("toc")}, args...)...) }
	d.Body().Append(
		ui.H2(ui.Text("One")), ui.H3(ui.Attr("data-tag", "new"), ui.Text("First")), ui.H3(ui.Text("Second")),
		ui.H2(ui.Text("Two")), ui.H3(ui.Attr("data-tag", "x"), ui.Text("Third")),
		toc(ui.Li(ui.Text("a")),
			ui.Li(ui.Text("b"), toc(ui.Li(ui.Text("b1")), ui.Li(ui.Text("b2"), toc(ui.Li(ui.Text("deep")))))),
			ui.Li(ui.Text("c"))),
		ui.P(ui.Class("set"), ui.Span(ui.Text("x")), ui.Span(ui.Class("jump"), ui.Text("y")), ui.Span(ui.Text("z"))),
		ui.P(ui.Text("He said "), ui.Q(ui.Text("one "), ui.Q(ui.Text("two "), ui.Q(ui.Text("three")))), ui.Text(" and "),
			ui.Q(ui.Text("again"))),
		ui.P(ui.Class("fr"), ui.Q(ui.Text("un "), ui.Q(ui.Text("deux "), ui.Q(ui.Text("trois"))))),
		ui.P(ui.Class("noq"), ui.Q(ui.Text("bare"))),
		ui.Blockquote(ui.Text("left open "), ui.Q(ui.Text("inside"))),
		ui.P(ui.Q(ui.Text("after it"))),
		ui.P(ui.Class("image"), ui.Text("pictures")),
	)
	r := chrome.Match(t, d, geom.V2(700, 900))
	t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
}

// zenGarden is the two uses of ::before in CSS Zen Garden's design 221,
// "Mid Century Modern": a numeral above each design's name, from
// li:nth-child(n)::before in a font seven times the item's, and a
// photograph beside the requirements, an empty absolutely positioned
// ::before with a cover background, left of its element.
const zenGarden = `
body { margin: 0; font-size: 12px; line-height: 1.5; }
.sidebar .design-selection nav { width: 80%; margin-left: 20%; padding: 1em 0 2em 0; }
.sidebar .design-selection nav ul { padding: 0; margin: 12px 0; }
.sidebar .design-selection nav ul li { display: inline-block; list-style: none; color: #ffffff; width: 25%; float: left;
  padding: 10em 1em 2em 1em; box-sizing: border-box; text-align: left; }
.sidebar .design-selection nav ul li a { color: #ffffff; text-decoration: none; display: block; }
.sidebar .design-selection nav ul li:nth-child(1) { background: #0d2c40; margin-top: -9em; padding: 19em 1em 2em 1em; }
.sidebar .design-selection nav ul li:nth-child(2) { background: #f15a30; }
.sidebar .design-selection nav ul li:nth-child(3) { background: #31c5da; }
.sidebar .design-selection nav ul li:nth-child(4) { background: #f15a30; }
.sidebar .design-selection nav ul li:nth-child(5) { background: #0d2c40; }
.sidebar .design-selection nav ul li:nth-child(1)::before { content: "1"; font-size: 700%; }
.sidebar .design-selection nav ul li:nth-child(2)::before { content: "2"; font-size: 700%; }
.sidebar .design-selection nav ul li:nth-child(3)::before { content: "3"; font-size: 700%; }
.sidebar .design-selection nav ul li:nth-child(4)::before { content: "4"; font-size: 700%; }
.sidebar .design-selection nav ul li:nth-child(5)::before { content: "5"; font-size: 700%; }
@media (max-width: 53em) {
  .sidebar .design-selection nav ul li { width: 100%; padding: 5em 1em 2em 1em; }
  .sidebar .design-selection nav ul li:nth-child(1) { margin-top: 0; padding: 5em 1em 2em 1em; }
  .sidebar .design-selection nav ul li:nth-child(1)::before,
  .sidebar .design-selection nav ul li:nth-child(2)::before { font-size: 300%; display: block; padding: 0 0 0.25em 0; }
}
.main { position: relative; height: 44em; margin-top: 10em; clear: both; }
.main .requirements { position: absolute; right: 0; z-index: 2; margin: 2em 0 0 0; width: 60%; padding: 1em 0 0 0;
  box-sizing: border-box; column-count: 2; column-gap: 2em; }
.main .requirements p { margin: 0 1em 1em 1em; }
.main .requirements::before { content: ""; position: absolute; top: 0; left: -60%;
  background: url(photo.png) no-repeat left center; background-size: cover; z-index: 2; width: 60%; height: 40em;
  margin-top: 0; padding: 1em; box-sizing: border-box; }
`

func zenGardenDoc() *ui.Document {
	d := ui.NewDocument(ui.CSS(zenGarden), ui.Images(fstest.MapFS{"photo.png": {Data: pngOf(40, 30)}}))
	list := ui.Ul()
	for i, name := range []string{"Mid Century Modern", "Garments", "Steel", "Apothecary", "Screen Filler"} {
		list.Append(ui.Li(ui.A(ui.Attr("href", "#"), ui.Text(name)), ui.Text(fmt.Sprint(" by designer ", i+1))))
	}
	reqs := ui.Div(ui.Class("requirements"))
	for range 4 {
		reqs.Append(ui.P(ui.Text("Where possible, we would like to see mostly CSS 1 & 2 usage. CSS 3 & 4 should be " +
			"limited to widely-supported elements only, or strong fallbacks should be provided.")))
	}
	d.Body().Append(ui.Div(ui.Class("sidebar"), ui.Div(ui.Class("design-selection"), ui.Nav(list))),
		ui.Div(ui.Class("main"), reqs))
	return d
}

func TestTheZenGardensNumeralsAndPhotoMatchChrome(t *testing.T) {
	for _, width := range []float32{1000, 600} {
		t.Run(fmt.Sprint(width), func(t *testing.T) {
			r := chrome.Match(t, zenGardenDoc(), geom.V2(width, 900))
			t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
		})
	}
}
