//go:build chrome

package layout_test

import (
	"strings"
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
)

// boxes are n divs, each with style.
func boxes(n int, style string) []ui.Arg {
	out := make([]ui.Arg, n)
	for i := range out {
		out[i] = ui.Div(ui.Style(style))
	}
	return out
}

// sized are divs of the widths or heights given, in px, each with style.
func sized(prop, style string, lengths ...string) []ui.Arg {
	out := make([]ui.Arg, len(lengths))
	for i, l := range lengths {
		out[i] = ui.Div(ui.Style(prop + ": " + l + "px; " + style))
	}
	return out
}

func inline(style ui.Arg, kids ...ui.Arg) *ui.Node {
	return ui.Div(append([]ui.Arg{style}, kids...)...)
}

func flexBox(style string, kids []ui.Arg) *ui.Node {
	return ui.Div(append([]ui.Arg{ui.Style("display: flex; margin-bottom: 8px; background: #eee; " + style)},
		kids...)...)
}

// TestFlexWrapBalanceMatchesChrome lays out balanced flex containers, WPT
// css/css-flexbox/balance's: lines as long as each other, a minimum of lines, zero-sized
// and negative-margined items, an item alone on an overflowing line, and the room the
// line count leaves each item across.
func TestFlexWrapBalanceMatchesChrome(t *testing.T) {
	const sheet = `body { width: 600px } .s > div { width: 25px; height: 25px; background: green }`
	square := "width: 25px; height: 25px; background: green"
	sixteen := append(boxes(7, square), ui.Div(ui.Style("width: 100px; height: 25px")))
	sixteen = append(sixteen, boxes(8, square)...)
	text := func(n int) []ui.Arg {
		out := make([]ui.Arg, n)
		for i := range out {
			out[i] = ui.Div(ui.Style("background: green; line-height: 0"),
				ui.Span(ui.Style("display: inline-block; width: 100px; height: 50px")), ui.Text(" "),
				ui.Span(ui.Style("display: inline-block; width: 100px; height: 50px")))
		}
		return out
	}
	cases := map[string]*ui.Document{
		"four in two lines of two": styled(sheet,
			flexBox("flex-wrap: balance; width: 100px; padding: 10px; gap: 10px", boxes(4, square)),
			flexBox("flex-wrap: wrap-reverse balance; width: 100px; padding: 10px; gap: 10px", boxes(5, square)),
			flexBox("flex-flow: column balance; height: 100px; width: 200px; gap: 10px", boxes(5, square)),
			flexBox("flex-wrap: balance; width: 120px; padding: 10px; gap: 10px", sixteen)),
		"a minimum of lines": styled(sheet,
			flexBox("flex-wrap: balance; flex-line-count: 4; width: 200px", boxes(4, "width: 100px; height: 25px")),
			flexBox("flex-wrap: balance; flex-line-count: 4; gap: 20px", boxes(9, "width: 20px; height: 20px")),
			flexBox("flex-wrap: balance; flex-line-count: 6; gap: 20px", boxes(9, "width: 20px; height: 20px")),
			flexBox("flex-wrap: balance; flex-line-count: 3; width: 150px", boxes(4, "width: 50px; height: 50px")),
			flexBox("flex-wrap: balance; flex-line-count: 5; width: 300px", boxes(3, "width: 50px; height: 20px")),
			flexBox("flex-wrap: wrap; flex-line-count: 3; width: 300px", boxes(4, "width: 50px; height: 20px"))),
		"items that measure nothing": styled(sheet,
			flexBox("flex-wrap: balance; width: 100px",
				sized("width", "height: 50px; flex-grow: 1", "150", "0")),
			flexBox("flex-wrap: balance; width: 100px; height: 100px",
				[]ui.Arg{ui.Div(ui.Style("width: 150px; height: 50px; flex-grow: 1")),
					ui.Div(ui.Style("width: 0; height: 50px; flex-grow: 1; margin: 0 -50px"))}),
			flexBox("flex-wrap: balance; width: 100px; column-gap: 10px; row-gap: 10px",
				sized("width", "height: 20px; flex-shrink: 0", "50", "300", "20", "20")),
			flexBox("flex-wrap: balance; width: 100px", sized("width", "height: 20px", "60", "40", "0", "30", "0"))),
		"flex-grow after balancing": styled(sheet,
			flexBox("flex-wrap: balance; width: 100px; gap: 10px", boxes(5, "width: 30px; height: 20px; flex-grow: 1"))),
		"the room across from the line count": styled(sheet,
			flexBox("flex-flow: column balance; flex-line-count: 2; gap: 20px; width: 220px", text(3)),
			flexBox("flex-flow: column wrap; flex-line-count: 2; gap: 20px; width: 220px", text(2)),
			flexBox("flex-flow: column balance; flex-line-count: 2; gap: 20px; width: 100px",
				boxes(4, "width: 60%; aspect-ratio: 1"))),
		"intrinsic sizes with a line count": styled(sheet,
			inline(ui.Style("display: inline-flex; gap: 20px; flex-wrap: balance; flex-line-count: 2; height: 100px"),
				sized("width", "", "40", "40", "40")...),
			inline(ui.Style("display: inline-flex; gap: 20px; flex-wrap: balance; flex-line-count: 2; height: 100px"),
				sized("width", "", "100", "30", "30")...),
			inline(ui.Style("display: inline-flex; width: min-content; gap: 20px; flex-wrap: balance; "+
				"flex-line-count: 2; height: 100px"), sized("width", "", "100", "100", "100")...),
			inline(ui.Style("display: inline-flex; gap: 20px; flex-direction: column; flex-wrap: balance; "+
				"flex-line-count: 2"), sized("height", "width: 40px", "40", "40", "40")...),
			inline(ui.Style("display: inline-flex; gap: 20px; flex-direction: column; flex-wrap: balance; "+
				"flex-line-count: 2"), sized("height", "width: 40px", "100", "30", "30")...),
			inline(ui.Style("display: inline-flex; gap: 20px; flex-wrap: balance; flex-line-count: 3; height: 50px"),
				sized("width", "", strings.Fields("10 20 30 40 50 60")...)...)),
	}
	matchPages(t, geom.Vec2{X: 800, Y: 600}, cases)
}
