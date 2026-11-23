//go:build chrome

package ui_test

import (
	"testing"

	"github.com/lantervale/ember/ui"
)

func TestVerticalBaselinesAsChrome(t *testing.T) {
	matchVertical(t, map[string]func(t *testing.T) *ui.Document{
		"an inline table beside an inline-block on a vertical-rl line": func(_ *testing.T) *ui.Document {
			d := ui.NewDocument(ui.CSS(`.c { border: solid 1px black; writing-mode: vertical-rl; }
				.b { display: inline-block; background: aqua; }
				.t { display: inline-table; font-size: 40px; background: tan; margin-block-start: 50px; }
				.t td { padding: 0; } .t { border-spacing: 0; }`))
			d.Body().Append(ui.Div(ui.Class("c"), ui.Div(ui.Class("b"), ui.Text("aaa")), ui.Text(" "),
				ui.Table(ui.Class("t"), ui.Tr(ui.Td(ui.Text("bbb"))))))
			return d
		},
		"a positioned box of last baseline in a vertical-rl grid": func(_ *testing.T) *ui.Document {
			return staticGrid(``)
		},
		"a horizontal positioned box of last baseline in a vertical-rl grid": func(_ *testing.T) *ui.Document {
			return staticGrid(`.c > * { writing-mode: horizontal-tb; }`)
		},
		"positioned boxes of a vertical-rl grid aligned at its end and centred": func(_ *testing.T) *ui.Document {
			return staticGrid(`.c > * { align-self: end; justify-self: center; }`)
		},
		"positioned boxes of a vertical-lr grid aligned at its end": func(_ *testing.T) *ui.Document {
			return staticGrid(`.c { writing-mode: vertical-lr; } .c > * { align-self: end; justify-self: end; }`)
		},
	})
}

func TestPositionedBoxesLeavingVerticalBoxesAsChrome(t *testing.T) {
	page := func(sheet string) func(t *testing.T) *ui.Document {
		return func(_ *testing.T) *ui.Document {
			d := ui.NewDocument(ui.CSS(`.v { width: 120px; height: 90px; padding: 3px 7px 5px 2px; border: 2px solid;
				margin: 10px 20px; } .v > p { margin: 0; inline-size: 30px; block-size: 20px; }
				.a { position: absolute; width: 14px; height: 9px; background: teal; margin: 1px 4px 2px 3px; }` + sheet))
			d.Body().Append(ui.Div(ui.Class("v"), ui.P(), ui.Div(ui.Class("a")), ui.P()))
			return d
		}
	}
	matchVertical(t, map[string]func(t *testing.T) *ui.Document{
		"at its static place in vertical-rl": page(`.v { writing-mode: vertical-rl; }`),
		"at its static place in vertical-lr": page(`.v { writing-mode: vertical-lr; }`),
		"at its static place in sideways-lr": page(`.v { writing-mode: sideways-lr; }`),
		"by its insets in vertical-rl":       page(`.v { writing-mode: vertical-rl; } .a { top: 6px; right: 9px; }`),
		"by its logical insets in vertical-lr": page(`.v { writing-mode: vertical-lr; }
			.a { inset-block-start: 6px; inset-inline-end: 9px; }`),
		"of horizontal lines in vertical-rl": page(`.v { writing-mode: vertical-rl; } .a { writing-mode: horizontal-tb;
			inset-inline-start: 5px; }`),
		"with text in vertical-rl": page(`.v { writing-mode: vertical-rl; } .a { width: auto; height: auto; }
			.a::before { content: "Some words"; }`),
		"in a horizontal box of a vertical page": page(`html { writing-mode: vertical-rl; }
			.v { writing-mode: horizontal-tb; } .a { bottom: 4px; }`),
	})
}

func staticGrid(extra string) *ui.Document {
	d := ui.NewDocument(ui.CSS(`.c { display: grid; writing-mode: vertical-rl; padding: 2px 1px;
		border: 1px solid black; background: yellow; margin-bottom: 5px; margin-right: 5px; float: left; }
		.big > .c { width: 30px; height: 22px; grid: 2px 20px 2px / 3px 14px 3px; }
		.small > .c { grid: 0px 2px 0px / 3px 2px 3px; width: 2px; height: 4px; margin-right: 10px; }
		.c > * { position: absolute; grid-area: 2 / 2 / 3 / 3; background: teal; width: 6px; height: 8px;
		align-self: last baseline; }` + extra))
	for _, size := range []string{"big", "small"} {
		d.Body().Append(ui.Div(ui.Class(size), ui.Div(ui.Class("c"), ui.Div())))
	}
	return d
}
