//go:build chrome

package layout_test

import (
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
)

func TestAnIframesBoxMatchesChromes(t *testing.T) {
	frame := func(args ...ui.Arg) *ui.Node { return ui.Iframe(nil, args...) }
	cases := map[string]*ui.Document{
		"default": styled("", ui.P(ui.Text("before "), frame(), ui.Text(" after"))),
		"attributes": styled("",
			ui.P(frame(ui.Attr("width", "120"), ui.Attr("height", "40")), ui.Text(" beside "),
				frame(ui.Attr("width", "50%")), frame(ui.Attr("height", "60"))),
			ui.P(frame(ui.Attr("frameborder", "0")), frame(ui.Attr("frameborder", "no")),
				frame(ui.Attr("frameborder", "1"), ui.Attr("width", "80")))),
		"css sizes": styled(`.w { width: 400px } .h { height: 80px } .both { width: 90px; height: 70px }
			.box { box-sizing: border-box; width: 200px; height: 100px; padding: 10px; border: 5px solid }
			.max { max-width: 100px } .min { min-height: 200px }`,
			ui.Div(frame(ui.Class("w")), frame(ui.Class("h")), frame(ui.Class("both"))),
			ui.Div(frame(ui.Class("box")), frame(ui.Class("max")), frame(ui.Class("min")))),
		"percentages": styled(`.outer { width: 500px; height: 300px } .w { width: 50% }
			.h { display: block; height: 50% } .full { display: block; width: 100%; height: 100%; box-sizing: border-box }`,
			ui.Div(ui.Class("outer"), frame(ui.Class("w")), frame(ui.Class("h"))),
			ui.Div(ui.Class("outer"), frame(ui.Class("full")))),
		"aspect-ratio": styled(`.r { aspect-ratio: 16 / 9; width: 320px } .rh { aspect-ratio: 2 / 1; height: 60px }
			.block { display: block; aspect-ratio: 16 / 9; width: 100%; height: auto }`,
			ui.Div(frame(ui.Class("r")), frame(ui.Class("rh"))),
			ui.Div(ui.Style("width: 480px"), frame(ui.Class("block")))),
		"display": styled(`.none { display: none } .block { display: block } .flex { display: flex; gap: 8px }
			.grow { flex: 1 } .abs { position: absolute; right: 10px; bottom: 10px; width: 100px; height: 50px }`,
			ui.P(ui.Text("gone "), frame(ui.Class("none")), ui.Text(" here")),
			frame(ui.Class("block")),
			ui.Div(ui.Class("flex"), frame(), frame(ui.Class("grow"))),
			frame(ui.Class("abs"))),
		"baseline": styled(`p { font-size: 20px; line-height: 30px } .mid { vertical-align: middle }
			.top { vertical-align: top } .small { width: 60px; height: 20px }`,
			ui.P(ui.Text("text "), frame(ui.Class("small")), ui.Text(" on the line "),
				frame(ui.Class("small"), ui.Class("mid")), ui.Text(" and "), frame(ui.Class("small"), ui.Class("top"))),
			ui.P(ui.Span(ui.Text("a span ")), frame(), ui.Text(" then words that wrap round the frame and go on"))),
	}
	for name, d := range cases {
		t.Run(name, func(t *testing.T) {
			matchesChromesOwn(t, d, geom.V2(800, 900), "iframe")
		})
	}
}
