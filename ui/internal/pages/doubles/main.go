// Double borders and outlines as Chrome draws them: two lines a third of
// the width each, solid where too thin for a gap, round corners, sides of
// other colours and widths mitered, sides set by border-inline and
// border-block, and an outline round a rounded box.
package main

import (
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #f4ede4; }
body { padding: 12px; margin: 0; }
.row { display: flex; gap: 16px; align-items: flex-start; margin-bottom: 16px; }
.row > div { width: 48px; height: 32px; background-color: #ffffff; }
.d1 { border: 1px double #2a2530; }
.d2 { border: 2px double #2a2530; }
.d3 { border: 3px double #2a2530; }
.d5 { border: 5px double #3421e6; }
.d6 { border: 6px double #3421e6; }
.d10 { border: 10px double #e03c31; }
.r { border: 9px double #2a9d4a; border-radius: 14px; }
.r2 { border: 6px double #2a2530; border-radius: 20px 4px; }
.mix { border: 9px double; border-color: #e03c31 #3421e6 #2a9d4a #e0a030; }
.uneven { border-style: double; border-width: 3px 12px 6px 9px; border-color: #2a2530; }
.half { border: 9px solid #3421e6; border-top-style: double; border-left-style: double; }
.inl { border-inline: 9px double #e03c31; border-block: 3px solid #2a2530; }
.blk { border-block: 6px double #3421e6; border-inline-width: 2px; border-inline-style: solid; }
.o1 { outline: 6px double #e03c31; margin: 8px; }
.o2 { outline: 9px double #3421e6; outline-offset: 3px; border-radius: 10px; margin: 12px; }
.o3 { outline: 3px double #2a2530; outline-offset: -8px; margin: 8px; }
`

func main() {
	d := ui.NewDocument(ui.CSS(sheet))
	row := func(classes ...string) *ui.Node {
		r := ui.Div(ui.Class("row"))
		for _, c := range classes {
			r.Append(ui.Div(ui.Class(c)))
		}
		return r
	}
	d.Body().Append(
		row("d1", "d2", "d3", "d5", "d6", "d10"),
		row("r", "r2", "mix", "uneven", "half"),
		row("inl", "blk", "o1", "o2", "o3"),
	)
	pages.Run(d, 560, 230)
}
