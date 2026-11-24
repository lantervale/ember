//go:build chrome

package layout_test

import (
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
)

func TestBreaksClearingFloatsMatchChrome(t *testing.T) {
	const sheet = `.l { float: left } .r { float: right }
.box { width: 60px; height: 50px; background: #888 }
.tall { height: 90px }
.wide { width: 260px }
br.both { clear: both } br.left { clear: left } br.right { clear: right }
br.start { clear: inline-start } br.end { clear: inline-end }`
	box := func(class ...string) *ui.Node { return ui.Div(ui.Class(append(class, "box")...)) }
	inBox := func(class ...string) *ui.Node { return ui.Span(ui.Class(append(class, "box")...)) }
	word := func(s string) *ui.Node { return ui.Span(ui.Text(s)) }
	cases := map[string]*ui.Document{
		"at the end of a line": styled(sheet, ui.Div(ui.Class("wide"), box("l"),
			word("One"), ui.Br(ui.Class("both")), word("Two"))),
		"the presentational attribute": styled(sheet, ui.Div(ui.Class("wide"), box("l"), box("r", "tall"),
			word("One"), ui.Br(ui.Attr("clear", "left")), word("Two"), ui.Br(ui.Attr("clear", "all")), word("Three"),
			box("r"), ui.Br(ui.Attr("clear", "right")), word("Four"))),
		"one side": styled(sheet, ui.Div(ui.Class("wide"), box("l"), box("r", "tall"),
			word("One"), ui.Br(ui.Class("left")), word("Two"), ui.Br(ui.Class("right")), word("Three"))),
		"logical sides right to left": styled(sheet, ui.Div(ui.Class("wide"), ui.Style("direction: rtl"),
			box("l"), box("r", "tall"), word("One"), ui.Br(ui.Class("start")), word("Two"),
			ui.Br(ui.Class("end")), word("Three"))),
		"inside a span": styled(sheet, ui.Div(ui.Class("wide"), box("l"),
			ui.Span(word("One"), ui.Br(ui.Class("both"))), word("Two"))),
		"inside nested spans with padding": styled(sheet, ui.Div(ui.Class("wide"), box("l"),
			ui.Span(ui.Style("padding: 0 6px; border: 1px solid"),
				ui.Span(ui.Style("padding-right: 4px"), word("One"), ui.Br(ui.Class("both")))), word("Two"))),
		"at the end of the block": styled(sheet, ui.Div(ui.Class("wide"), box("l"),
			word("One"), ui.Br(ui.Class("both"))), ui.P(ui.Style("margin: 0"), word("After"))),
		"at the end of the block inside a span": styled(sheet, ui.Div(ui.Class("wide"), box("l"),
			ui.Span(word("One"), ui.Br(ui.Class("both")))), ui.P(ui.Style("margin: 0"), word("After"))),
		"at the end of a bordered block": styled(sheet, ui.Div(ui.Class("wide"),
			ui.Style("border: 2px solid; padding: 3px; margin-bottom: 7px"), box("l"),
			word("One"), ui.Br(ui.Class("both"))), ui.P(ui.Style("margin: 4px 0"), word("After"))),
		"two at the end": styled(sheet, ui.Div(ui.Class("wide"), box("l"),
			word("One"), ui.Br(ui.Class("both")), ui.Br(ui.Class("both"))), ui.P(ui.Style("margin: 0"), word("After"))),
		"a float of its own line": styled(sheet, ui.Div(ui.Class("wide"), word("One"), box("r"),
			ui.Br(ui.Class("both")), word("Two"))),
		"a float after it on its line": styled(sheet, ui.Div(ui.Class("wide"), word("One"),
			ui.Br(ui.Class("both")), box("r"), word("Two"))),
		"floats already above": styled(sheet, ui.Div(ui.Class("wide"), ui.Style("line-height: 60px"), box("l"),
			word("One"), ui.Br(ui.Class("both")), word("Two"))),
		"no clear inside a span": styled(sheet, ui.Div(ui.Class("wide"),
			ui.Span(ui.Style("padding-right: 5px; border: 1px solid"), word("One"), ui.Br()), word("Two"))),
		"no clear inside nested spans": styled(sheet, ui.Div(ui.Class("wide"), ui.Span(ui.Style("padding: 0 3px"),
			ui.Span(ui.Style("padding: 0 4px"), word("One"), ui.Br())), ui.Span(ui.Style("padding: 0 2px"), word("Two")))),
		"no floats": styled(sheet, ui.Div(ui.Class("wide"), word("One"), ui.Br(ui.Class("both")), word("Two"))),
		"a float from a block before": styled(sheet, ui.Div(ui.Class("wide"), box("l"),
			ui.P(ui.Style("margin: 0"), word("One"), ui.Br(ui.Class("both")), word("Two")),
			ui.P(ui.Style("margin: 0"), word("Three")))),
		"a float from a block before, at the end": styled(sheet, ui.Div(ui.Class("wide"), box("l"),
			ui.P(ui.Style("margin: 0 0 6px"), word("One"), ui.Br(ui.Class("both"))),
			ui.P(ui.Style("margin: 4px 0"), word("Two")))),
		"a tall line": styled(sheet, ui.Div(ui.Class("wide"), box("l"),
			ui.Span(ui.Style("font-size: 40px"), word("One")), ui.Br(ui.Class("both")), word("Two"))),
		"vertical": styled(sheet, ui.Div(ui.Style("writing-mode: vertical-rl; height: 260px"), box("l"),
			word("One"), ui.Br(ui.Class("both")), word("Two"))),
		"vertical at the end of the block": styled(sheet, ui.Div(ui.Style("writing-mode: vertical-rl; height: 260px"),
			ui.Div(box("r"), word("One"), ui.Br(ui.Class("both"))), word("After"))),
		"an empty span after it": styled(sheet, ui.Div(ui.Class("wide"), box("l"),
			word("One"), ui.Br(ui.Class("both")), ui.Span(ui.Style("border: 1px solid"))),
			ui.P(ui.Style("margin: 0"), word("After"))),
		"a flow root at its end": styled(sheet, ui.Div(ui.Class("wide"), ui.Style("display: flow-root"), box("l", "tall"),
			word("One"), ui.Br(ui.Class("both"))), ui.P(ui.Style("margin: 0"), word("After"))),
		"alone between paragraphs whose margins leave the body": styled(sheet,
			ui.P(word("One"), inBox("l")), ui.Br(ui.Attr("clear", "all")), ui.P(word("Two"))),
		"a block clearing the floats of a paragraph whose margins leave the body": styled(sheet,
			ui.P(word("One"), inBox("l")), ui.Div(ui.Style("clear: both"), word("Two")), ui.P(word("Three"))),
		"lines beside the floats of a paragraph whose margins leave a block": styled(sheet,
			ui.Div(ui.Style("margin-top: 4px"), ui.P(word("One"), inBox("l", "tall")), ui.Div(ui.Class("wide"),
				word("Two"), ui.Br(), word("Three"), ui.Br(), word("Four"), ui.Br(), word("Five"), ui.Br(),
				ui.Br(ui.Class("left")), word("Six")))),
		"vertical at the end": styled(sheet, ui.Div(ui.Style("writing-mode: vertical-lr; height: 260px"),
			ui.Div(box("l"), ui.Span(word("One"), ui.Br(ui.Class("both")))), word("After"))),
	}
	matchPages(t, geom.V2(800, 600), cases)
}
