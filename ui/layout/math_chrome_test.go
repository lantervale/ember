//go:build chrome

package layout_test

import (
	"strconv"
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
)

func TestMathFunctionsLayOutAsInChrome(t *testing.T) {
	long := "calc(0px"
	for i := range 24 {
		long += " + min(1%, " + strconv.Itoa(i) + "px)"
	}
	long += ")"
	widths := []string{"round(50%, 7px)", "round(up, 33%, 10px)", "round(down, 33%, 10px)",
		"round(to-zero, calc(20px - 40%), 15px)", "calc(-1 * round(to-zero, calc(20px - 40%), 15px))",
		"mod(50%, 70px)", "mod(calc(-1 * 50%), 70px)", "rem(50%, 70px)", "abs(calc(20px - 50%))",
		"calc(100px * sign(50% - 100px))", "calc(100px + 50px * sign(25% - 100px))", "hypot(30%, 40px)",
		"clamp(none, 50%, 120px)", "clamp(60px, 10%, none)", "clamp(none, 70%, none)", "calc(1px * pow(2, 7))",
		"calc(200px * sin(30deg))", "calc(100px * cos(0.25turn - 60deg))", "calc(1px * atan2(1em, 10px) / 1deg)",
		"calc(10px / 1px * 3px)", "calc(1em * 1em / 1px)", "calc(30px * e)", "calc(20px * pi)",
		"calc(1px * log(1024, 2) * 10)", "calc(1px * sqrt(2) * 100)", "calc(1px * exp(4))", long,
		"round(nearest, 50% / 3, 1em)"}
	cases := map[string]*ui.Document{
		"widths": styled(unitsSheet, bars("outer", widths...)),
		"widths in a narrower box": styled(unitsSheet,
			ui.Div(ui.Style("width: 180px"), bars("", widths...))),
		"margins and paddings": styled(unitsSheet,
			ui.Div(ui.Class("outer"), ui.Style("padding: mod(10%, 13px) rem(7%, 6px) abs(-2%) round(up, 3%, 4px)"),
				ui.Div(ui.Class("b"), ui.Style("margin: hypot(1%, 2px) calc(10px * sign(10% - 30px))")))),
		"infinities": styled(unitsSheet,
			ui.Div(ui.Class("outer"),
				ui.Div(ui.Class("b"), ui.Style("width: 50%; max-width: calc(infinity * 1px)")),
				ui.Div(ui.Class("b"), ui.Style("width: calc(NaN * 1px + 40px)")),
				ui.Div(ui.Class("b"), ui.Style("width: 120px; margin-left: max(calc(-infinity * 1px), 30px)")),
				ui.Div(ui.Class("b"), ui.Style("width: calc(1px / sign(-0) + 1px); min-width: 15px")),
				ui.Div(ui.Class("b"), ui.Style("width: min(calc(infinity * 1%), 90px)")))),
		"line heights": styled(unitsSheet,
			ui.Div(ui.Style("font-size: 20px"),
				ui.Div(ui.Style("line-height: min(150%, 40px)"), ui.Text("Lines a and b")),
				ui.Div(ui.Style("line-height: max(120%, 10px)"), ui.Text("Lines c")),
				ui.Div(ui.Style("line-height: clamp(10px, 200%, 50px)"), ui.Text("Lines d")),
				ui.Div(ui.Style("line-height: round(150%, 4px)"), ui.Text("Lines e")),
				ui.Div(ui.Style("line-height: calc(1em / 10px)"), ui.Text("Lines f")),
				ui.Div(ui.Style("line-height: calc(2 * sign(50% - 1px))"), ui.Text("Lines g")),
				ui.Div(ui.Style("line-height: abs(-80%)"), ui.Text("Lines h")))),
		"word spacings": styled(unitsSheet,
			ui.Div(ui.Style("width: 600px"),
				ui.Span(ui.Style("word-spacing: max(50%, 2px)"), ui.Text("one two three four")), ui.Br(),
				ui.Span(ui.Style("word-spacing: min(100%, 3px)"), ui.Text("one two three four")), ui.Br(),
				ui.Span(ui.Style("word-spacing: round(30%, 2px)"), ui.Text("one two three four")), ui.Br(),
				ui.Span(ui.Style("word-spacing: calc(10px * sign(10% - 1px))"), ui.Text("one two three four")))),
		"vertical aligns": styled(unitsSheet,
			ui.Div(ui.Style("line-height: 30px"), ui.Text("Base "),
				ui.Span(ui.Style("vertical-align: min(50%, 4px)"), ui.Text("up")), ui.Text(" "),
				ui.Span(ui.Style("vertical-align: max(50%, 4px)"), ui.Text("upper")), ui.Text(" "),
				ui.Span(ui.Style("vertical-align: round(up, 30%, 4px)"), ui.Text("rounded")), ui.Text(" "),
				ui.Span(ui.Style("vertical-align: calc(-1 * max(20%, 2px))"), ui.Text("down")))),
	}
	for name, d := range cases {
		t.Run(name, func(t *testing.T) { chrome.Match(t, d, geom.V2(640, 400)) })
	}
}
