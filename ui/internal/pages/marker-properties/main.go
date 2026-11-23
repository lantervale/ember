// ::marker's text properties against Chrome's: a line-height of its own,
// white-space collapsing the spaces of a string marker, overflow-wrap on
// a long marker, its item's quotes giving its content's open-quote and
// close-quote (Chrome 156 ignores quotes set on ::marker itself, 157 takes
// them, as ui does), and a text-shadow.
package main

import (
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #fff; }
body { margin: 0; padding: 10px; font-size: 15px; line-height: 20px; color: #2b2722; }
ol, ul { margin: 0 0 8px; padding-left: 60px; background-color: #f4f1ec; }
.tall li::marker { line-height: 44px; color: #b0493c; }
.spaces { list-style-type: "a   b   "; }
.spaces li:nth-child(2)::marker { white-space: normal; }
.spaces li:nth-child(3)::marker { white-space: pre-line; }
.wrap li::marker { content: "averyveryverylongmarker "; overflow-wrap: anywhere; }
.wrap li:nth-child(2)::marker { overflow-wrap: normal; white-space: normal; }
.quoted li { quotes: "<<" ">>"; }
.quoted li::marker { content: open-quote "q" close-quote " "; }
.quoted li:nth-child(2) { quotes: "[" "]"; }
.shadow li::marker { text-shadow: 2px 2px #3c6fb0; font-weight: 700; }
`

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"m.css": {Data: []byte(sheet)}}, "m.css"))
	items := func(n int) []ui.Arg {
		var out []ui.Arg
		for i := range n {
			out = append(out, ui.Li(ui.Text([]string{"first item", "second item", "third item"}[i])))
		}
		return out
	}
	d.Body().Append(
		ui.Ol(append([]ui.Arg{ui.Class("tall")}, items(2)...)...),
		ui.Ul(append([]ui.Arg{ui.Class("spaces")}, items(3)...)...),
		ui.Ul(append([]ui.Arg{ui.Class("wrap")}, items(2)...)...),
		ui.Ul(append([]ui.Arg{ui.Class("quoted")}, items(2)...)...),
		ui.Ol(append([]ui.Arg{ui.Class("shadow")}, items(2)...)...),
	)
	pages.Run(d, 420, 360)
}
