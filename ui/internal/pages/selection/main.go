// A selection against Chrome's: made as the page starts, here by
// Document.Select and there by the script the export carries, from the
// middle of a paragraph to the middle of another, over wrapped lines, a
// br, an empty line's end, inline elements with their own ::selection, a
// rule that sets the text's colour alone, text user-select: none leaves
// out, two columns, a table and a scrolled box. Every ::selection colour
// is the page's own, so no machine's highlight colour shows.
package main

import (
	"strings"
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #fff; user-select: text; }
body { margin: 0; padding: 12px; font-size: 14px; line-height: 20px; color: #2b2722; }
::selection { background-color: #ffd54a; color: #1a1a1a; }
p { margin: 0 0 10px; }
h2 { margin: 0 0 8px; font-size: 20px; line-height: 26px; }
.own::selection { background-color: #3c6fb0; color: #ffffff; }
.ink::selection { color: #b0493c; }
.none { user-select: none; color: #8a8378; }
.big { font-size: 22px; }
.cols { column-count: 2; column-gap: 24px; column-rule: 1px solid #8a8378; margin-bottom: 10px; }
.cols p { margin: 0 0 6px; }
.just { text-align: justify; }
table { border-collapse: collapse; margin-bottom: 10px; }
td { border: 1px solid #8a8378; padding: 2px 8px; }
.scroller { height: 50px; overflow: auto; border: 1px solid #8a8378; padding: 2px 6px; margin-bottom: 10px; }
`

const sentence = "The quick brown fox jumps over the lazy dog, then naps in the warm afternoon sun."

func prose(n int) string { return strings.TrimSpace(strings.Repeat(sentence+" ", n)) }

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"selection.css": {Data: []byte(sheet)}}, "selection.css"))
	first := ui.P(ui.Text(prose(2)))
	last := ui.P(ui.Text(prose(1)))
	d.Body().Append(
		ui.H2(ui.Text("Before the selection")),
		first,
		ui.H2(ui.Text("A heading inside it")),
		ui.P(ui.Text("A line ends here,"), ui.Br(), ui.Text("the next starts, with "),
			ui.Strong(ui.Class("own"), ui.Text("its own colours")), ui.Text(" and "),
			ui.Span(ui.Class("big"), ui.Text("taller text")), ui.Text(" on a line.")),
		ui.P(ui.Class("ink"), ui.Text("Only the text's colour changes here, "),
			ui.Span(ui.Class("none"), ui.Text("this takes no selection, ")), ui.Text("and this does again.")),
		ui.Div(ui.Class("cols"), ui.P(ui.Class("just"), ui.Text(prose(2))), ui.P(ui.Text(prose(1)))),
		ui.Table(ui.Tbody(
			ui.Tr(ui.Td(ui.Text("one")), ui.Td(ui.Text("two")), ui.Td(ui.Text("three"))),
			ui.Tr(ui.Td(ui.Text("four")), ui.Td(ui.Text("five")), ui.Td(ui.Text("six"))))),
		ui.Div(ui.Class("scroller"), ui.Text(prose(3))),
		last,
		ui.P(ui.Text("After the selection.")),
	)
	d.Select(ui.Position{Node: first.Children()[0], Offset: 30}, ui.Position{Node: last.Children()[0], Offset: 43})
	pages.Run(d, 560, 560)
}
