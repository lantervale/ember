// Pseudo-elements against Chrome's: ::before and ::after as inline text,
// blocks, badges and an absolutely positioned ribbon, attr() and a
// picture as content, counters numbering headings and a nested list,
// quotes by depth, ::marker colours and content over the list styles,
// and the two uses of CSS Zen Garden 221: a numeral above each design's
// name and a photograph beside the requirements.
package main

import (
	_ "embed"
	"fmt"
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

//go:embed beach.png
var beach []byte

const sheet = `
html { background-color: #fff; }
body { margin: 0; padding: 10px; font-size: 12px; line-height: 16px; color: #2b2722; counter-reset: chapter; }
p, h4, ul, ol { margin: 0 0 6px; }
.row { display: flex; gap: 10px; margin-bottom: 8px; }
.row > div { flex: 1; padding: 6px; background-color: #f4f1ec; position: relative; }
.note::before { content: "Note: "; font-weight: 700; color: #b0493c; }
.note::after { content: " (" attr(data-by) ")"; font-style: italic; color: #3c6fb0; }
.block::before { content: "before"; display: block; padding: 2px 6px; margin-bottom: 4px; background-color: #3c6fb0;
	color: #fff; border-radius: 4px; }
.block::after { content: ""; display: block; height: 4px; margin-top: 4px;
	background-image: linear-gradient(to right, #b0493c, #e6dcc8); }
.badge::after { content: "new"; display: inline-block; margin-left: 6px; padding: 0 5px; border: 1px solid #3c8a4f;
	border-radius: 8px; color: #3c8a4f; font-size: 10px; transform: rotate(-8deg); }
.ribbon::before { content: ""; position: absolute; top: 0; right: 0; width: 28px; height: 28px;
	background-color: #7a4fb0; opacity: 0.6; }
.pic::before { content: url(beach.png); margin-right: 6px; }
h4 { counter-increment: chapter; counter-reset: section; font-size: 13px; }
h4::before { content: counter(chapter, upper-roman) ". "; color: #b0493c; }
h5 { counter-increment: section; margin: 0 0 4px; font-size: 12px; font-weight: 400; }
h5::before { content: counter(chapter) "." counter(section, decimal-leading-zero) " "; color: #8a8378; }
.toc, .toc ol { counter-reset: item; list-style: none; padding-left: 14px; }
.toc li { counter-increment: item; }
.toc li::before { content: counters(item, ".") " "; color: #3c6fb0; }
.fr { quotes: "«" "»" "‹" "›"; }
q { color: #3c8a4f; }
.marks { padding-left: 30px; }
.marks li::marker { color: #b0493c; font-weight: 700; }
.marks .star::marker { content: "+ "; color: #7a4fb0; }
.roman { list-style-type: upper-roman; }
.alpha { list-style-type: lower-alpha; }
.zero { list-style-type: decimal-leading-zero; }
.designs { margin: 0 0 8px; padding: 0; overflow: hidden; }
.designs li { display: inline-block; list-style: none; color: #fff; width: 25%; float: left; padding: 1em 1em 1em 1em;
	box-sizing: border-box; }
.designs li:nth-child(1) { background: #0d2c40; }
.designs li:nth-child(2) { background: #f15a30; }
.designs li:nth-child(3) { background: #31c5da; }
.designs li:nth-child(4) { background: #f15a30; }
.designs li:nth-child(1)::before { content: "1"; font-size: 400%; line-height: 1; }
.designs li:nth-child(2)::before { content: "2"; font-size: 400%; line-height: 1; }
.designs li:nth-child(3)::before { content: "3"; font-size: 400%; line-height: 1; }
.designs li:nth-child(4)::before { content: "4"; font-size: 400%; line-height: 1; }
.designs a { display: block; color: #fff; text-decoration: none; }
.main { position: relative; height: 96px; background-color: #ece7df; }
.requirements { position: absolute; right: 0; width: 60%; padding: 1em 0 0 0; box-sizing: border-box; }
.requirements p { margin: 0 1em 1em 1em; }
.requirements::before { content: ""; position: absolute; top: 0; left: -60%;
	background: url(beach.png) no-repeat left center; background-size: cover; width: 60%; height: 8em;
	padding: 1em; box-sizing: border-box; }
`

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(sheet)}, "beach.png": {Data: beach}},
		"page.css"))
	designs := ui.Ul(ui.Class("designs"))
	for i, name := range []string{"Mid Century", "Garments", "Steel", "Apothecary"} {
		designs.Append(ui.Li(ui.A(ui.Attr("href", "#"), ui.Text(name)), ui.Text(fmt.Sprint("by designer ", i+1))))
	}
	d.Body().Append(
		ui.Div(ui.Class("row"),
			ui.Div(ui.P(ui.Class("note"), ui.Attr("data-by", "the editor"), ui.Text("generated around text")),
				ui.P(ui.Class("badge"), ui.Text("An inline-block"))),
			ui.Div(ui.Class("block", "ribbon"), ui.Text("Blocks before and after, a ribbon over the corner")),
			ui.Div(ui.P(ui.Class("pic"), ui.Text("a picture")),
				ui.P(ui.Text("She said "), ui.Q(ui.Text("one "), ui.Q(ui.Text("two"))), ui.Text(".")),
				ui.P(ui.Class("fr"), ui.Q(ui.Text("un "), ui.Q(ui.Text("deux"))))),
		),
		ui.Div(ui.Class("row"),
			ui.Div(ui.H4(ui.Text("Light")), ui.H5(ui.Text("Torches")), ui.H5(ui.Text("Lanterns")),
				ui.H4(ui.Text("Dark")), ui.H5(ui.Text("Crypts"))),
			ui.Div(ui.Ol(ui.Class("toc"), ui.Li(ui.Text("Walls")),
				ui.Li(ui.Text("Floors"), ui.Ol(ui.Li(ui.Text("Stone")), ui.Li(ui.Text("Moss"), ui.Ol(ui.Li(ui.Text("Wet")))))),
				ui.Li(ui.Text("Doors")))),
			ui.Div(ui.Ol(ui.Class("marks", "roman"), ui.Attr("start", "3"), ui.Li(ui.Text("three")),
				ui.Li(ui.Class("star"), ui.Text("starred")), ui.Li(ui.Attr("value", "9"), ui.Text("nine"))),
				ui.Ol(ui.Class("marks", "alpha"), ui.Attr("reversed", ""), ui.Li(ui.Text("b")), ui.Li(ui.Text("a"))),
				ui.Ol(ui.Class("marks", "zero"), ui.Li(ui.Text("one")), ui.Li(ui.Text("two")))),
		),
		designs,
		ui.Div(ui.Class("main"), ui.Div(ui.Class("requirements"),
			ui.P(ui.Text("Where possible, we would like to see mostly CSS 1 & 2 usage.")),
			ui.P(ui.Text("We ask that you submit original artwork.")))),
	)
	pages.Run(d, 640, 440)
}
