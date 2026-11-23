//go:build chrome

package layout_test

import (
	"testing"
	"testing/fstest"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
)

// aligning is a page of sheet with a picture to show.
func aligning(sheet string, nodes ...*ui.Node) *ui.Document {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(sheet)}, "tile.png": {Data: tilePNG()}},
		"page.css"))
	d.Body().Append(nodes...)
	return d
}

func TestVerticalAlignMatchesChrome(t *testing.T) {
	src := ui.Attr("src", "tile.png")
	cases := map[string]*ui.Document{}
	for _, v := range []string{"baseline", "sub", "super", "text-top", "text-bottom", "middle", "top", "bottom",
		"7px", "-5px", "50%", "-25%", "calc(10% + 3px)", "-webkit-baseline-middle"} {
		at := "vertical-align: " + v
		var ps []*ui.Node
		for _, lh := range []string{"1.2", "40px", "normal", "8px"} {
			p := "width: 400px; line-height: " + lh
			ps = append(ps,
				ui.P(ui.Style(p), ui.Text("Hxg "), ui.Span(ui.Style(at+"; font-size: 24px"), ui.Text("big span")),
					ui.Text(" then "), ui.Span(ui.Style(at+"; font-size: 9px; line-height: 30px"), ui.Text("small")),
					ui.Text(" end")),
				ui.P(ui.Style(p), ui.Text("Hxg "),
					ui.Label(ui.Style(at+"; display: inline-block; width: 20px; height: 34px; background: #888")),
					ui.Text(" then "),
					ui.Label(ui.Style(at+"; display: inline-block; width: 40px; font-size: 10px; margin: 3px 0 5px"),
						ui.Text("two lines")),
					ui.Text(" end")),
				ui.P(ui.Style(p), ui.Text("Hxg "), ui.Img(src, ui.Style(at+"; width: 16px; height: 30px")),
					ui.Text(" then "), ui.Img(src, ui.Style(at+"; width: 6px; height: 6px; margin: 2px")), ui.Text(" end")),
			)
		}
		cases[v] = aligning("", ps...)
	}
	slab := func(h, at string) *ui.Node {
		return ui.Label(ui.Style("display: inline-block; width: 10px; background: #888; height: " + h +
			"; vertical-align: " + at))
	}
	cases["top and bottom"] = aligning("p { font-size: 20px; line-height: 30px; }",
		ui.P(ui.Text("x"), slab("80px", "top")),
		ui.P(ui.Text("x"), slab("80px", "bottom")),
		ui.P(ui.Text("x"), slab("80px", "top"), slab("50px", "bottom")),
		ui.P(ui.Text("x"), slab("50px", "top"), slab("80px", "bottom")),
		ui.P(ui.Text("x"), slab("80px", "bottom"), slab("50px", "top")),
		ui.P(ui.Text("x"), slab("80px", "top"), slab("80px", "bottom")),
		ui.P(ui.Text("x"), slab("60px", "top"), slab("80px", "bottom"), slab("70px", "top"), slab("40px", "middle")),
		ui.P(ui.Text("x"), slab("10px", "top"), slab("10px", "bottom")),
		ui.P(slab("80px", "top")),
		ui.P(slab("80px", "bottom"), ui.Br(), ui.Text("y"), slab("10px", "top")),
		ui.P(ui.Text("x"), slab("50px", "text-top"), slab("80px", "bottom"), slab("44px", "text-bottom")),
	)
	cases["nested"] = aligning("p { font-size: 20px; line-height: 30px; }",
		ui.P(ui.Text("x"), ui.Span(ui.Style("vertical-align: super"), ui.Text("y"), slab("40px", "middle"))),
		ui.P(ui.Text("x"), ui.Span(ui.Style("vertical-align: top"), ui.Text("y"), slab("60px", "baseline"))),
		ui.P(ui.Text("x"), ui.Span(ui.Style("vertical-align: top"), ui.Text("y"), slab("60px", "top"))),
		ui.P(ui.Text("x"), ui.Span(ui.Style("vertical-align: top"), ui.Text("y"), slab("60px", "bottom"))),
		ui.P(ui.Text("x"), ui.Span(ui.Style("vertical-align: bottom"), ui.Text("y"), slab("60px", "top"),
			ui.Span(ui.Text("z"), slab("70px", "bottom")))),
		ui.P(ui.Text("x"), ui.Span(ui.Style("vertical-align: 10px"), ui.Text("y"), slab("70px", "bottom"))),
		ui.P(ui.Text("x"), ui.Span(ui.Style("vertical-align: sub"), ui.Text("y"), slab("70px", "top"))),
		ui.P(ui.Text("x"), ui.Span(ui.Style("vertical-align: -10px"), ui.Text("y"), slab("20px", "text-top"))),
		ui.P(ui.Text("x"), ui.Span(ui.Style("vertical-align: middle"), ui.Text("y"), slab("60px", "baseline"))),
		ui.P(ui.Text("x"), ui.Span(ui.Style("vertical-align: text-top"), ui.Text("y"), slab("60px", "baseline"))),
		ui.P(ui.Text("x"), ui.Span(ui.Style("vertical-align: sub"), ui.Text("y"),
			ui.Span(ui.Style("vertical-align: sub; font-size: 12px"), ui.Text("z"),
				ui.Span(ui.Style("vertical-align: 50%"), ui.Text("w"))))),
		ui.P(ui.Text("x"), ui.Span(ui.Style("vertical-align: text-bottom; font-size: 30px"), ui.Text("y"),
			ui.Span(ui.Style("vertical-align: text-top; font-size: 10px"), ui.Text("z")))),
	)
	cases["lines of their own"] = aligning("p { width: 150px; font-size: 14px; }",
		ui.P(ui.Text("Words before "), ui.Span(ui.Style("vertical-align: middle; font-size: 22px"),
			ui.Text("a middle span that wraps to more lines "), slab("40px", "top"), ui.Text(" with a tall box on one")),
			ui.Text(" and after.")),
		ui.P(ui.Text("One "), ui.Span(ui.Style("vertical-align: 6px; border: 1px solid; padding: 2px"),
			ui.Text("raised and boxed over a break")), ui.Text(" end")),
	)
	cases["empty boxes"] = aligning("p { font-size: 20px; line-height: 30px; }",
		ui.P(ui.Text("x"), ui.Span(ui.Style("font-size: 60px; line-height: 90px"))),
		ui.P(ui.Text("x"), ui.Span(ui.Style("font-size: 60px; line-height: 90px"), slab("10px", "baseline"))),
		ui.P(ui.Style("width: 100px"), ui.Span(ui.Style("line-height: 50px"),
			ui.Span(ui.Style("line-height: 20px"), ui.Text("an inner span that wraps inside a taller one")))),
		ui.P(ui.Text("x"), ui.Span(ui.Style("vertical-align: top; line-height: 70px"))),
	)
	cases["baselines of inline-blocks"] = aligning(
		"p { font-size: 20px; line-height: 30px; } p label { display: inline-block; font-size: 10px; line-height: 14px; }",
		ui.P(ui.Text("x"), ui.Label(ui.Text("a"), ui.Br(), ui.Text("b"))),
		ui.P(ui.Text("x"), ui.Label(ui.Style("overflow: hidden"), ui.Text("a"), ui.Br(), ui.Text("b"))),
		ui.P(ui.Text("x"), ui.Label(ui.Style("width: 10px; height: 20px; margin: 3px 0 5px"))),
		ui.P(ui.Text("x"), ui.Label(ui.Style("width: 10px; height: 60px; margin: 3px 0 5px; vertical-align: top"))),
		ui.P(ui.Text("x"), ui.Label(ui.Style("width: 10px; height: 60px; margin: 3px 0 5px; vertical-align: bottom"))),
		ui.P(ui.Text("x"), ui.Label(ui.Style("vertical-align: middle; padding: 4px; border: 2px solid"), ui.Text("a"),
			ui.Br(), ui.Text("b"))),
		ui.P(ui.Text("x"), ui.Label(ui.Style("display: inline-flex; vertical-align: text-top"), ui.Label(ui.Text("flex")))),
		ui.P(ui.Text("x"), ui.Input(ui.Style("vertical-align: middle; width: 120px")),
			ui.Button(ui.Style("vertical-align: top"), ui.Text("go")),
			ui.Input(ui.Attr("type", "checkbox"), ui.Style("vertical-align: -3px"))),
	)
	cases["generated, markers and floats"] = aligning(`
p, li { font-size: 16px; line-height: 24px; width: 220px; }
.a::before { content: "top"; vertical-align: top; font-size: 10px; }
.a::after { content: "low"; vertical-align: -6px; font-size: 10px; }
.f { display: block; float: left; width: 40px; height: 50px; }`,
		ui.P(ui.Class("a"), ui.Text(" between ")),
		ui.Ul(ui.Li(ui.Text("an item "), slab("40px", "top")), ui.Li(slab("40px", "middle"), ui.Text(" one more"))),
		ui.P(ui.Label(ui.Class("f")), ui.Text("Words beside a float with "), slab("30px", "middle"),
			ui.Text(" a box in the middle and "), slab("36px", "bottom"), ui.Text(" one at the bottom of its line.")),
	)
	cases["cells"] = aligning(`
td { width: 70px; padding: 2px; }
.t { vertical-align: top; } .m { vertical-align: middle; } .b { vertical-align: bottom; }
.s { vertical-align: sub; } .l { vertical-align: 10px; }`,
		ui.Table(
			ui.Tr(ui.Td(block("20px", "80px")), ui.Td(ui.Text("plain")), ui.Td(ui.Class("t"), ui.Text("top")),
				ui.Td(ui.Class("m"), ui.Text("middle")), ui.Td(ui.Class("b"), ui.Text("bottom")),
				ui.Td(ui.Class("b"), block("20px", "20px")), ui.Td(ui.Class("t"), block("20px", "20px"))),
			ui.Tr(ui.Class("b"), ui.Td(block("20px", "60px")), ui.Td(ui.Text("row")),
				ui.Td(ui.Class("t"), ui.Text("own")), ui.Th(ui.Text("head"))),
			ui.Tr(ui.Td(block("20px", "60px")), ui.Td(ui.Attr("valign", "bottom"), ui.Class("t"), ui.Text("sheet wins")),
				ui.Td(ui.Class("s"), ui.Text("sub")), ui.Td(ui.Class("l"), ui.Text("length"))),
		),
		ui.Table(ui.Tbody(ui.Class("t"), ui.Tr(ui.Td(block("20px", "60px")), ui.Td(ui.Text("group")))),
			ui.Tfoot(ui.Tr(ui.Td(block("20px", "60px")), ui.Td(ui.Text("foot"))))),
	)
	for name, d := range cases {
		t.Run(name, func(t *testing.T) {
			r := chrome.Match(t, d, geom.V2(800, 1400))
			t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
			matchText(t, d, geom.V2(800, 1400))
			matchSpanFragments(t, d, geom.V2(800, 1400))
		})
	}
}
