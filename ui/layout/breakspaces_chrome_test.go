//go:build chrome

package layout_test

import (
	"fmt"
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
)

func TestBreakSpacesBreakAsChrome(t *testing.T) {
	bs := func(style, text string) *ui.Node {
		return ui.P(ui.Style("white-space: break-spaces; margin: 0; "+style), ui.Text(text))
	}
	cases := map[string]*ui.Document{
		"spaces at a line's end": page(
			bs("width: 40px", "abc   def"),
			bs("width: 30px", "abc def"),
			bs("width: 60px", "abc  def ghi    jkl mno"),
			bs("width: 0", "ab  cd"),
			bs("width: 20px", "   ab   "),
		),
		"a sweep of widths": func() *ui.Document {
			var ps []*ui.Node
			for w := 10; w < 200; w += 9 {
				ps = append(ps, bs(fmt.Sprintf("width: %dpx", w), "The  lantern  swings   over the gate,    all night  long. "))
			}
			return page(ps...)
		}(),
		"tabs and newlines": page(
			bs("width: 41px", "ab  \t cd"),
			bs("width: 30px", "ab\t\tcd"),
			bs("width: 50px", "ab\n  cd  \nef"),
			bs("width: 50px", "abc   \n\n  de"),
		),
		"aligned": page(
			bs("width: 60px; text-align: right", "abc def ghi jkl"),
			bs("width: 60px; text-align: center", "abc def ghi jkl"),
			bs("width: 70px; text-align: right", "abc    def"),
			bs("width: 60px; text-align: end; direction: rtl", "abc def ghi jkl"),
			bs("width: 60px; text-align: left; direction: rtl", "abc def ghi jkl"),
			bs("width: 60px; text-align: justify", "abc def ghi jkl"),
			bs("width: 70px; text-align: justify", "ab cd  ef gh ij kl"),
			bs("width: 90px; text-align: justify", "The  lantern  swings   over the gate,    all night  long. "),
		),
		"indented": page(bs("width: 50px; text-indent: 10px", "abc  defg hi")),
		"spans": page(
			ui.P(ui.Style("white-space: break-spaces; width: 60px"), ui.Text("abc"), ui.B(ui.Text(" ")),
				ui.Text("def"), ui.B(ui.Text("  ghi"))),
			ui.P(ui.Style("width: 70px"), ui.Text("one two "),
				ui.Span(ui.Style("white-space: break-spaces"), ui.Text("three   four  ")), ui.Text("five six")),
		),
		"breaking words": page(
			bs("width: 20px; overflow-wrap: anywhere", "abcdef  ghij"),
			bs("width: 20px; word-break: break-all", "abcdef  ghij"),
			bs("width: 30px; line-break: anywhere", "abc  defg"),
		),
		"nowrap": page(
			bs("width: 30px; text-wrap-mode: nowrap", "abc   def  "),
			ui.P(ui.Style("white-space-collapse: break-spaces; width: 30px"), ui.Text("abc   def")),
		),
		"content sizes": page(
			ui.Div(ui.Style("display: inline-block; white-space: break-spaces"), ui.Text("ab   ")),
			ui.Div(ui.Style("width: min-content; white-space: break-spaces"), ui.Text("ab  cd")),
			ui.Div(ui.Style("width: min-content; white-space: break-spaces"), ui.Text("ab cdef  ")),
			ui.Div(ui.Style("width: min-content; white-space: break-spaces"), ui.Text("   ab")),
			ui.Div(ui.Style("width: min-content; white-space: break-spaces"), ui.Text("ab\t\tcd")),
			ui.Div(ui.Style("width: max-content; white-space: break-spaces"), ui.Text("ab  cd  ")),
			ui.Div(ui.Style("display: flex"), ui.Div(ui.Style("white-space: break-spaces"), ui.Text("one   two  "))),
		),
	}
	for name, d := range cases {
		t.Run(name, func(t *testing.T) {
			r := chrome.Match(t, d, geom.V2(800, 600))
			t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
			matchText(t, d, geom.V2(800, 600))
			matchSpanFragments(t, d, geom.V2(800, 600))
		})
	}
}

func TestACarriageReturnBreaksNoLineAsChrome(t *testing.T) {
	cr := func(ws string) *ui.Node {
		return ui.P(ui.Style("width: 80px; white-space: "+ws), ui.Text("ab\rcd\r\nef gh\r\rij"))
	}
	d := page(cr("pre"), cr("pre-wrap"), cr("pre-line"), cr("normal"), cr("nowrap"), cr("break-spaces"),
		ui.Div(ui.Style("display: inline-block; white-space: pre"), ui.Text("abc\r")),
		ui.Div(ui.Style("display: inline-block; white-space: pre"), ui.Text("abc\r\r\rd")),
		ui.Div(ui.Style("width: min-content; white-space: pre-wrap"), ui.Text("ab\rcd")),
	)
	r := chrome.Match(t, d, geom.V2(800, 600))
	t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
	matchText(t, d, geom.V2(800, 600))
}

func TestBreakAllBreaksPunctuationAndElementsAsChrome(t *testing.T) {
	all := func(width, text string) *ui.Node {
		return ui.P(ui.Style("word-break: break-all; margin: 0; width: "+width), ui.Text(text))
	}
	cases := map[string]*ui.Document{
		"punctuation": page(
			all("0", "a(b)c[d]e{f}g"),
			all("0", "a.b,c;d:e!f?g"),
			all("0", "a'b\"c/d\\e%f$g#h@i&j*k=m"),
			all("0", "a-b‐c–d—e_f"),
			all("0", "1.5 2,3 4%5 $6 a1"),
			all("0", "(ab) [cd] «ef» “gh”"),
			all("0", "x).(y a…b x·y a~b a^b"),
			all("30px", "Open assets/tilesets/(dungeon_walls).tsj, now!"),
			all("45px", "“Lanterns” [brass], (iron) & glass: 12.5% off $30."),
		),
		"elements": page(
			ui.P(ui.Style("width: 0"), ui.Text("ab"), ui.B(ui.Style("word-break: break-all"), ui.Text("cd")),
				ui.Text("ef")),
			ui.P(ui.Style("width: 0; word-break: break-all"), ui.Text("ab"),
				ui.B(ui.Style("word-break: normal"), ui.Text("cd")), ui.Text("ef")),
			ui.P(ui.Style("width: 0; word-break: break-all"), ui.Text("ab"), ui.I(ui.Text("(cd)")), ui.Text("ef")),
			ui.P(ui.Style("width: 30px; word-break: break-all"), ui.Text("lantern"), ui.B(ui.Text("making")),
				ui.Text("folk")),
		),
	}
	for name, d := range cases {
		t.Run(name, func(t *testing.T) {
			r := chrome.Match(t, d, geom.V2(800, 600))
			t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
			matchText(t, d, geom.V2(800, 600))
			matchSpanFragments(t, d, geom.V2(800, 600))
		})
	}
}

func TestPreservedSpacesEndingTheContentTakeRoomAsChrome(t *testing.T) {
	pre := func(s string) *ui.Node {
		return ui.Pre(ui.Style("float: left; clear: left; margin: 0; background: #ddd"), ui.Text(s))
	}
	d := page(pre("ab   "), pre("   "), pre("ab\n   "), pre("ab\n   \n"), pre("ab\t"),
		ui.Div(ui.Style("float: left; clear: left; white-space: break-spaces"), ui.Text("ab   ")))
	r := chrome.Match(t, d, geom.V2(800, 600))
	t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
}
