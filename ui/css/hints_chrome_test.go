//go:build chrome

package css_test

import (
	"strings"
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

// hintedTags are the elements whose presentational attributes Chrome reads under its own
// user-agent sheet: ember's, sent as the page's, would win over them.
const hintedTags = "table|thead|tbody|tfoot|tr|td|th|col|colgroup|hr|font|br|img"

// ownSheet drops from the page every rule of ember's user-agent sheet that names one of
// hintedTags, so Chrome styles them by its own.
const ownSheet = `
const tags = /(^|[\s,>+~(])(` + hintedTags + `)(?![\w-])/;
for (const sheet of document.styleSheets) {
	for (const layer of sheet.cssRules) {
		if (!(layer instanceof CSSLayerBlockRule)) continue;
		for (let i = layer.cssRules.length - 1; i >= 0; i--) {
			if (tags.test(layer.cssRules[i].selectorText || "")) layer.deleteRule(i);
		}
	}
}`

//nolint:gochecknoglobals // the properties compared, a test table
var hinted = map[string]func(s *style.Computed) string{
	"color":            func(s *style.Computed) string { return rgb(s.Color) },
	"background-color": func(s *style.Computed) string { return rgb(s.Background) },
	"background-image": func(s *style.Computed) string {
		if len(s.BackgroundImages) == 0 {
			return "none"
		}
		return image(s.BackgroundImages[0].Source)
	},
	"border-top-width":    func(s *style.Computed) string { return px(s.Border.Top) },
	"border-right-width":  func(s *style.Computed) string { return px(s.Border.Right) },
	"border-bottom-width": func(s *style.Computed) string { return px(s.Border.Bottom) },
	"border-left-width":   func(s *style.Computed) string { return px(s.Border.Left) },
	"border-top-style":    func(s *style.Computed) string { return sideStyle(s, 0) },
	"border-right-style":  func(s *style.Computed) string { return sideStyle(s, 1) },
	"border-bottom-style": func(s *style.Computed) string { return sideStyle(s, 2) },
	"border-left-style":   func(s *style.Computed) string { return sideStyle(s, 3) },
	"border-top-color":    func(s *style.Computed) string { return rgb(s.BorderColor.Top) },
	"border-left-color":   func(s *style.Computed) string { return rgb(s.BorderColor.Left) },
	"border-collapse": func(s *style.Computed) string {
		return map[bool]string{true: "collapse", false: "separate"}[s.BorderCollapse]
	},
	"float": func(s *style.Computed) string { return []string{"none", "left", "right"}[s.Float] },
	"clear": func(s *style.Computed) string { return []string{"none", "left", "right", "both"}[s.Clear] },
	"vertical-align": func(s *style.Computed) string {
		return []string{"baseline", "sub", "super", "text-top", "text-bottom", "middle", "top", "bottom",
			px(s.VerticalShift.Px), "-webkit-baseline-middle"}[s.VerticalAlign]
	},
	"font-size":   func(s *style.Computed) string { return px(s.Font.Size) },
	"font-family": func(s *style.Computed) string { return strings.Join(s.Font.Family, ", ") },
	"text-align":  compared["text-align"],
}

func sideStyle(s *style.Computed, side int) string {
	st := [4]style.BorderStyle{s.BorderStyle.Top, s.BorderStyle.Right, s.BorderStyle.Bottom, s.BorderStyle.Left}
	hidden := [4]bool{s.BorderHidden.Top, s.BorderHidden.Right, s.BorderHidden.Bottom, s.BorderHidden.Left}
	if hidden[side] {
		return "hidden"
	}
	return borderStyle(st[side])
}

func TestPresentationalAttributesStyleAsChromes(t *testing.T) {
	a := ui.Attr
	cell := func(args ...ui.Arg) *ui.Node { return ui.Td(append(args, ui.Text("x"))...) }
	table := func(args ...ui.Arg) *ui.Node {
		return ui.Table(append(args, ui.Tbody(ui.Tr(cell(), ui.Th(ui.Text("y")))))...)
	}
	const tile = "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8DwHwAFBQIAX8jx0g" +
		"AAAABJRU5ErkJggg=="
	img := func(args ...ui.Arg) *ui.Node { return ui.Img(append(args, ui.Attr("src", tile))...) }
	d := ui.NewDocument()
	d.Body().Append(
		table(a("border", "2")), table(a("border", "")), table(a("border", "0")), table(a("border", "3px")),
		table(a("border", "1"), a("bordercolor", "red")), table(a("bordercolor", "#00f")),
		table(a("rules", "all")), table(a("rules", "rows"), a("border", "1")), table(a("rules", "COLS")),
		table(a("rules", "none"), a("border", "3")), table(a("rules", "bogus")),
		ui.Table(a("rules", "groups"), ui.Colgroup(ui.Col()), ui.Thead(ui.Tr(cell())), ui.Tbody(ui.Tr(cell()))),
		table(a("frame", "box")), table(a("frame", "above")), table(a("frame", "below")), table(a("frame", "hsides")),
		table(a("frame", "lhs")), table(a("frame", "rhs")), table(a("frame", "vsides")), table(a("frame", "void")),
		table(a("frame", "border"), a("border", "4")), table(a("border", "4"), a("frame", "lhs")),
		table(a("frame", "box"), a("rules", "rows")),
		table(a("align", "left")), table(a("align", "RIGHT")), table(a("background", tile)),
		ui.Table(ui.Colgroup(a("align", "center"), a("valign", "top"), ui.Col(a("align", "right"), a("valign", "bottom"))),
			ui.Tbody(a("background", tile), ui.Tr(a("background", tile), cell(a("background", tile))))),
		img(a("align", "left")), img(a("align", "right")), img(a("align", "top")), img(a("align", "bottom")),
		img(a("align", "texttop")), img(a("align", "absmiddle")), img(a("align", "absbottom")),
		img(a("align", "baseline")), img(a("align", "center")), img(a("border", "3")),
		ui.Br(a("clear", "all")), ui.Br(a("clear", "left")), ui.Br(a("clear", "Right")), ui.Br(a("clear", "both")),
		ui.Br(a("clear", "none")),
		ui.Hr(a("noshade", "")), ui.Hr(a("size", "1")), ui.Hr(a("size", "6")), ui.Hr(a("noshade", ""), a("size", "4")),
		ui.Hr(a("color", "red")), ui.Hr(a("color", "red"), a("noshade", "")),
		ui.Font(a("size", "1"), ui.Text("a")), ui.Font(a("size", "2"), ui.Text("a")), ui.Font(a("size", "4"), ui.Text("a")),
		ui.Font(a("size", "5"), ui.Text("a")), ui.Font(a("size", "6"), ui.Text("a")), ui.Font(a("size", "7"), ui.Text("a")),
		ui.Font(a("size", "+1"), ui.Text("a")), ui.Font(a("size", "-2"), ui.Text("a")),
		ui.Font(a("size", "+9"), ui.Text("a")),
		ui.Font(a("size", "0"), ui.Text("a")), ui.Font(a("size", "x"), ui.Text("a")),
		ui.Font(a("face", "Go Mono, monospace"), ui.Text("a")), ui.Font(a("color", "#0a0"), ui.Text("a")),
		ui.Font(a("color", "green"), ui.Text("a"), ui.Span(ui.Text("b"))),
		ui.Font(a("face", "Go, sans-serif"), a("size", "+2"), ui.Text("a")),
		ui.Table(ui.Tbody(ui.Tr(
			cell(a("bgcolor", "fff")), cell(a("bgcolor", "0a0")), cell(a("bgcolor", "red; width: 9px")),
			cell(a("bgcolor", "chucknorris")), cell(a("bgcolor", "transparent")), cell(a("bgcolor", "#abcd")),
			cell(a("bgcolor", "RED")), cell(a("bgcolor", " #ff8000 ")), cell(a("bgcolor", "#1234567890ab")),
			cell(a("bgcolor", "rgb(1, 2, 3)"))))),
	)
	d.Layout(geom.V2(800, 600))
	matchChromeShown(t, d, hinted, 0, ownSheet, nil, func(n *dom.Node) *style.Computed { return dom.Data(n).Style })
}
