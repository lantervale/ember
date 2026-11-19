//go:build chrome

package layout_test

import (
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
)

const hlistSheet = `
.hlist ul { margin: 0; padding: 0; }
.hlist li { margin: 0; display: inline; }
.hlist ul ul { display: inline; }
.hlist li::after { content: "\a0 · "; font-weight: bold; }
.hlist li:last-child::after { content: none; }
.hlist li li:first-child::before { content: " ("; font-weight: normal; }
.hlist li li:last-child::after { content: ")"; font-weight: normal; }
.nowraplinks a { white-space: nowrap; }
table { width: 100%; } th { white-space: nowrap; } td { width: 100%; }
`

func hlistItem(text string, nested ...string) *ui.Node {
	li := ui.Li(ui.A(ui.Attr("href", "#"), ui.Text(text)))
	if len(nested) > 0 {
		ul := ui.Ul()
		for _, n := range nested {
			ul.Append(ui.Li(ui.A(ui.Attr("href", "#"), ui.Text(n))))
		}
		li.Append(ul)
	}
	return li
}

func TestHorizontalListsOfNowrapLinksBreakBetweenItemsAsChrome(t *testing.T) {
	items := func(n int, nestedAt int) *ui.Node {
		ul := ui.Ul()
		for i := range n {
			if i == nestedAt {
				ul.Append(hlistItem("Philips circle pattern", "PM 5538", "PM 5540", "PM 5544"))
				continue
			}
			ul.Append(hlistItem("Snell and Wilcox test card"))
		}
		return ul
	}
	navbox := func(ul *ui.Node) *ui.Node {
		tr := ui.Tr()
		tr.Append(ui.Th(ui.Text("Television")), ui.Td(ui.Div(ul)))
		return ui.Table(ui.Class("nowraplinks", "hlist"), ui.Tbody(tr))
	}
	cases := map[string]*ui.Document{
		"plain items":            styled(hlistSheet, ui.Div(ui.Class("nowraplinks", "hlist"), items(8, -1))),
		"a nested list first":    styled(hlistSheet, ui.Div(ui.Class("nowraplinks", "hlist"), items(8, 0))),
		"a nested list between":  styled(hlistSheet, ui.Div(ui.Class("nowraplinks", "hlist"), items(8, 2))),
		"in a cell":              styled(hlistSheet, navbox(items(8, -1))),
		"in a cell, nested":      styled(hlistSheet, navbox(items(8, 1))),
		"in a cell, nested last": styled(hlistSheet, navbox(items(8, 7))),
	}
	for name, d := range cases {
		t.Run(name, func(t *testing.T) { chrome.Match(t, d, geom.V2(640, 400)) })
	}
}

// TestANavboxOfTheWebWrapsItsListsAsChrome has the rows of Wikipedia's navboxes as its
// pages write them: a line break between the items, an item ending in text, or holding
// a list of its own.
func TestANavboxOfTheWebWrapsItsListsAsChrome(t *testing.T) {
	link := func(text string) *ui.Node { return ui.A(ui.Attr("href", "#"), ui.Text(text)) }
	list := func(items ...*ui.Node) *ui.Node {
		var args []ui.Arg
		for i, li := range items {
			if i > 0 {
				args = append(args, ui.Text("\n"))
			}
			args = append(args, li)
		}
		return ui.Ul(args...)
	}
	plain := func(names ...string) []*ui.Node {
		var out []*ui.Node
		for _, n := range names {
			out = append(out, ui.Li(link(n)))
		}
		return out
	}
	row := func(group string, ul *ui.Node) *ui.Node {
		tr := ui.Tr()
		tr.Append(ui.Th(ui.Style("width: 1%"), ui.Text(group)),
			ui.Td(ui.Class("hlist"), ui.Style("width: 100%; padding: 0"),
				ui.Div(ui.Style("padding: 0 0.25em"), ui.Text("\n"), ul, ui.Text("\n"))))
		return tr
	}
	navbox := func(rows ...*ui.Node) *ui.Node {
		body := ui.Tbody()
		body.Append(rows...)
		return ui.Table(ui.Class("nowraplinks"), body)
	}
	television := append(plain("SMPTE color bars", "EBU colour bars", "Indian-head test pattern", "ETP-1"),
		ui.Li(link("Philips circle pattern"), ui.Text(" ("), link("PM 5538"), ui.Text(", "), link("PM 5540"), ui.Text(", "),
			link("PM 5544"), ui.Text(")")))
	television = append(television, plain("Snell & Wilcox SW2/SW4", "Telefunken FuBK", "TVE test card", "UEIT")...)
	other := append([]*ui.Node{ui.Li(ui.Text("Acid\n"), list(plain("1", "2", "3")...))},
		plain(`"Bad Apple!!"`, "Big Buck Bunny", "Doom", "EICAR test file", "Functions for optimization", "GTUBE",
			"Harvard sentences", `"The North Wind and the Sun"`, "SMPTE universal leader", "Webdriver Torso")...)
	cases := map[string]*ui.Document{
		"an item ending in text": styled(hlistSheet, navbox(row("Television", list(television...)))),
		"an item holding a list": styled(hlistSheet, navbox(row("Other", list(other...)))),
		"both": styled(hlistSheet,
			navbox(row("Television", list(television...)), row("Other", list(other...)))),
	}
	for name, d := range cases {
		for _, w := range []float32{420, 700} {
			t.Run(name, func(t *testing.T) { chrome.Match(t, d, geom.V2(w, 400)) })
		}
	}
}
