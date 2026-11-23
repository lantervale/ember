//go:build chrome

package layout_test

import (
	"bytes"
	"image"
	"image/png"
	"testing"
	"testing/fstest"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
)

func smallPNG() []byte {
	var b bytes.Buffer
	if err := png.Encode(&b, image.NewNRGBA(image.Rect(0, 0, 100, 50))); err != nil {
		panic(err)
	}
	return b.Bytes()
}

func pictured(sheet string, nodes ...*ui.Node) *ui.Document {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(sheet)}, "small.png": {Data: smallPNG()},
		"wide.png": {Data: widePNG()}}, "page.css"))
	d.Body().Append(nodes...)
	return d
}

func TestReplacedBoxesInShrunkAndGridBoxesMatchChrome(t *testing.T) {
	small, wide, missing := ui.Attr("src", "small.png"), ui.Attr("src", "wide.png"), ui.Attr("src", "missing.png")
	const grid = `.grid { display: grid; width: 300px; height: 200px; grid-template-columns: 1fr 1fr; margin-bottom: 8px }`
	cases := map[string]*ui.Document{
		"a picture in a grid cell keeps its size": pictured(grid,
			ui.Div(ui.Class("grid"), ui.Img(small), ui.Img(wide))),
		"a picture in a grid cell stretched on one axis": pictured(grid+` .h img { justify-self: stretch }
.v img { align-self: stretch } .b img { place-self: stretch }`,
			ui.Div(ui.Class("grid", "h"), ui.Img(small), ui.Img(wide)),
			ui.Div(ui.Class("grid", "v"), ui.Img(small), ui.Img(wide)),
			ui.Div(ui.Class("grid", "b"), ui.Img(small), ui.Img(wide))),
		"a picture with a set width in a grid cell": pictured(grid+` img { width: 80px }`,
			ui.Div(ui.Class("grid"), ui.Img(small), ui.Img(wide))),
		"an img's width and height attributes as its proportions": pictured(`div { width: 300px }
img { width: 100%; height: auto } .half img { width: 50% } .broken img { display: block }`,
			ui.Div(ui.Class("broken"), ui.Img(missing, ui.Attr("width", "250"), ui.Attr("height", "167"))),
			ui.Div(ui.Class("broken"), ui.Img(missing, ui.Attr("width", "250"), ui.Attr("height", "167"),
				ui.Attr("alt", "a picture"))),
			ui.Div(ui.Img(ui.Attr("width", "250"), ui.Attr("height", "167"))),
			ui.Div(ui.Img(small, ui.Attr("width", "250"), ui.Attr("height", "167"))),
			ui.Div(ui.Class("half", "broken"), ui.Img(missing, ui.Attr("width", "40"), ui.Attr("height", "80px"))),
			ui.Div(ui.Class("broken"), ui.Img(missing, ui.Attr("width", "50%"), ui.Attr("height", "80")))),
		"a block picture of a percentage width in a float": pictured(`.f { float: left; margin-right: 8px }
img { display: block } .full { width: 100% } .most { max-width: 100% }`,
			ui.Div(ui.Style("width: 600px"),
				ui.Div(ui.Class("f"), ui.Img(small, ui.Class("full")), ui.Text("caption")),
				ui.Div(ui.Class("f"), ui.Img(wide, ui.Class("most")), ui.Text("a longer caption")),
				ui.Div(ui.Style("display: inline-block"), ui.Img(small, ui.Class("full")), ui.Text("in an inline block")))),
		"a block picture of a percentage width at a shrunk box's narrowest": pictured(`img { display: block }
.full { width: 100% } .most { max-width: 100% } .narrow { width: 60px } .f { float: left }
.row { display: flex; width: 120px } .grid { display: grid; grid-template-columns: min-content auto; width: 400px }`,
			ui.Div(ui.Class("narrow"), ui.Div(ui.Class("f"), ui.Img(small, ui.Class("full")), ui.Text("a few words"))),
			ui.Div(ui.Style("clear: both")),
			ui.Div(ui.Class("narrow"), ui.Div(ui.Class("f"), ui.Img(wide, ui.Class("most")), ui.Text("a few words"))),
			ui.Div(ui.Style("clear: both")),
			ui.Div(ui.Class("row"), ui.Div(ui.Img(wide, ui.Class("full")), ui.Text("one")), ui.Div(ui.Text("two words here"))),
			ui.Div(ui.Class("grid"), ui.Div(ui.Img(small, ui.Class("full")), ui.Text("one")), ui.Div(ui.Text("two")))),
		"a block picture of a percentage width in a flex item and a grid track": pictured(`img { display: block; width: 100% }
.row { display: flex; width: 600px } .grid { display: grid; grid-template-columns: auto auto; width: 600px }`,
			ui.Div(ui.Class("row"), ui.Div(ui.Img(small), ui.Text("one")), ui.Div(ui.Text("two words"))),
			ui.Div(ui.Class("grid"), ui.Div(ui.Img(wide), ui.Text("one")), ui.Div(ui.Text("two words")))),
	}
	for name, d := range cases {
		t.Run(name, func(t *testing.T) {
			r := chrome.Match(t, d, geom.V2(800, 600))
			t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
		})
	}
}

func TestAFloatAndItsTextInAShrunkBoxMatchChrome(t *testing.T) {
	const page = `.f { float: left; width: 50px; height: 20px; background: #888 }
.r { float: right; width: 30px; height: 10px }`
	cases := map[string]*ui.Document{
		"in a float": styled(page,
			ui.Div(ui.Style("float: left"), ui.Div(ui.Class("f")), ui.Text("text beside"))),
		"in an inline block": styled(page,
			ui.Div(ui.Style("display: inline-block"), ui.Div(ui.Class("f")), ui.Text("text beside"))),
		"in a table": styled(page,
			ui.Div(ui.Style("display: table"), ui.Div(ui.Class("f")), ui.Text("text beside"))),
		"two floats and text": styled(page,
			ui.Div(ui.Style("float: left"), ui.Div(ui.Class("f")), ui.Div(ui.Class("r")), ui.Text("text between"))),
		"a float in a paragraph": styled(page,
			ui.Div(ui.Style("display: inline-block"), ui.P(ui.Span(ui.Class("f")), ui.Text("some words")),
				ui.P(ui.Text("short")))),
		"a float before a block": styled(page,
			ui.Div(ui.Style("float: left"), ui.Div(ui.Class("f")), ui.Div(ui.Text("a block beside")))),
		"a float wider than the room": styled(page,
			ui.Div(ui.Style("width: 200px"),
				ui.Div(ui.Style("float: left"), ui.Div(ui.Class("f")), ui.Text(sentence)))),
	}
	for name, d := range cases {
		t.Run(name, func(t *testing.T) {
			r := chrome.Match(t, d, geom.V2(800, 600))
			t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
		})
	}
}

func TestInlinesCutByBlocksAndAfterBreaksMatchChrome(t *testing.T) {
	const page = `.wide { width: 300px } .b { background: #ccd } .f { border: 2px solid #a00; padding: 3px 5px }`
	cases := map[string]*ui.Document{
		"an empty inline after an ending br": styled(page,
			ui.Div(ui.Class("wide"), ui.Text("one"), ui.Br(), ui.Span())),
		"an empty framed inline after an ending br": styled(page,
			ui.Div(ui.Class("wide"), ui.Text("one"), ui.Br(), ui.Span(ui.Class("f")))),
		"a framed inline with no text before its block": styled(page,
			ui.Div(ui.Class("wide"), ui.Span(ui.Class("f"), ui.Div(ui.Class("b"), ui.Text("The block.")), ui.Text("after")))),
		"a framed inline with no text after its block": styled(page,
			ui.Div(ui.Class("wide"), ui.Span(ui.Class("f"), ui.Text("before"), ui.Div(ui.Class("b"), ui.Text("The block."))))),
		"a framed inline holding only a block": styled(page,
			ui.Div(ui.Class("wide"), ui.Span(ui.Class("f"), ui.Div(ui.Class("b"), ui.Text("The block."))))),
		"a padded inline with no text round its block": styled(`.p { padding: 4px }`,
			ui.Div(ui.Span(ui.Class("p"), ui.Div(ui.Text("The block."))))),
		"an inline holding blocks moved by position relative": styled(page+`
.m { position: relative; left: 20px; top: 10px }`,
			ui.Div(ui.Class("wide"), ui.Span(ui.Class("m"), ui.Text("before"), ui.Div(ui.Class("b"), ui.Text("The block.")),
				ui.Text("after")))),
		"inlines moved by position relative, nested and in percentages": styled(page+`
.m { position: relative; left: 10%; bottom: 6px } .n { position: relative; right: 5px; top: 3px }`,
			ui.Div(ui.Class("wide"), ui.Text("Head "), ui.Span(ui.Class("m"), ui.Text("moved "),
				ui.Span(ui.Class("n", "f"), ui.Text("more"), ui.Div(ui.Class("b"), ui.Text("The block."))), ui.Text(" tail")))),
	}
	for name, d := range cases {
		t.Run(name, func(t *testing.T) {
			r := chrome.Match(t, d, geom.V2(800, 600))
			t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
		})
	}
}
