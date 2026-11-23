//go:build chrome

package layout_test

import (
	"fmt"
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
)

// pairsPage is every pair of printable ASCII characters whose first is
// from firsts, each in a grid cell of no width: its lines break wherever
// a line may.
func pairsPage(firsts string, wordBreak string) *ui.Document {
	cells := []ui.Arg{ui.Class("pairs")}
	for _, a := range firsts {
		for b := '!'; b <= '~'; b++ {
			cells = append(cells, ui.Div(ui.Text(string(a)+string(b))))
		}
	}
	d := page(ui.Div(cells...))
	d.AddSheet(fmt.Sprintf(`.pairs { display: grid; grid-template-columns: repeat(32, 24px);
word-break: %s; }
.pairs > div { width: 0; }`, wordBreak))
	return d
}

// TestASCIIPairsBreakAsChrome compares where a line may break between
// every two printable ASCII characters with Chrome's.
func TestASCIIPairsBreakAsChrome(t *testing.T) {
	const all = "!\"#$%&'()*+,-./0123456789:;<=>?@ABCDEFGHIJKLMNOPQRSTUVWXYZ[\\]^_`abcdefghijklmnopqrstuvwxyz{|}~"
	for i := 0; i < len(all); i += 16 {
		firsts := all[i:min(i+16, len(all))]
		t.Run(firsts, func(t *testing.T) { matchPairs(t, pairsPage(firsts, "normal")) })
	}
	t.Run("break-all", func(t *testing.T) { matchPairs(t, pairsPage("!$%-?)}|a1", "break-all")) })
	t.Run("keep-all", func(t *testing.T) { matchPairs(t, pairsPage("!$%-?)}|a1", "keep-all")) })
}

func matchPairs(t *testing.T, d *ui.Document) {
	t.Helper()
	size := geom.V2(800, 2000)
	r := chrome.Match(t, d, size)
	t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
	matchText(t, d, size)
}

// TestASCIIBreaksInTextAsChrome compares the breaks of running Latin
// text, URLs, code and punctuation with Chrome's at several widths, a
// hyphen before a digit after a letter, a digit and punctuation, pairs
// across elements and beside other characters, and with word-break and
// line-break.
func TestASCIIBreaksInTextAsChrome(t *testing.T) {
	texts := []string{
		"See https://example.com/path/to/page?query=a&b=c#frag (or www.example.org/x_y) now!",
		"if(a<b&&c>d){return f(x[i],y[j]);}else{x+=1;y-=2;} // a|b, a^b, ~c, $d, 50% off",
		"Wait...what?!No way!!It's \"quoted\" and 'single' (paren)[bracket]{brace}<angle>",
		"Pay $$100, 99%% or #tag @user; a=b+c*d/e-f; 1-2 x-3 !-4 --5 ab-12 2025-11-23",
		"C:\\Users\\me\\file.txt; ~/bin/run.sh --flag=value -x; ./a.out|grep -v foo>out.txt",
	}
	widths := []string{"0", "40px", "73px", "120px"}
	cases := map[string]func() *ui.Document{
		"running text": func() *ui.Document {
			var ps []*ui.Node
			for _, w := range widths {
				for _, s := range texts {
					ps = append(ps, ui.P(ui.Style("width: "+w), ui.Text(s)))
				}
			}
			return page(ps...)
		},
		"across elements": func() *ui.Document {
			return page(
				ui.P(ui.Style("width: 0"), ui.Text("ab!"), ui.Span(ui.Text("cd")), ui.Text("}ef")),
				ui.P(ui.Style("width: 0"), ui.Text("ab,"), ui.B(ui.Text("(cd)")), ui.Text("ef")),
				ui.P(ui.Style("width: 0"), ui.Text("ab-"), ui.Span(ui.Text("12")), ui.Text(" x"), ui.Span(ui.Text("-3"))),
				ui.P(ui.Style("width: 0"), ui.Text("$"), ui.Span(ui.Style("padding: 0 2px"), ui.Text("$")), ui.Text("%%")),
			)
		},
		"beside other characters": func() *ui.Document {
			return page(
				ui.P(ui.Style("width: 0"), ui.Text("é!a ab!é é-1 a-é «a!b» a\u00a0!b a!\u00a0b a/é é/a")),
			)
		},
		"min-content": func() *ui.Document {
			cell := func(s string) *ui.Node {
				return ui.Div(ui.Style("display: inline-block; width: min-content; border: 1px solid; margin: 2px; "+
					"vertical-align: top"), ui.Text(s))
			}
			return page(cell("ab!cd"), cell("ab-12"), cell("ab,(cd)"), cell("$$$$"), cell("what?yes"),
				cell("a=b+c"), cell("x->y"), cell("f(x)[y]{z}<w>"))
		},
	}
	for name, d := range cases {
		t.Run(name, func(t *testing.T) { matchBreaks(t, d()) })
	}
	t.Run("beside ideographs", func(t *testing.T) {
		matchBreaks(t, hanPage(t, "p { width: 0; }", hanP("", ui.Text("漢!a a!漢 a(漢 漢(a 1-漢 a-1漢 漢$$漢"))))
	})
	for _, st := range []string{"word-break: break-all", "word-break: keep-all", "word-break: break-word",
		"overflow-wrap: anywhere", "line-break: strict", "line-break: loose", "line-break: anywhere"} {
		t.Run(st, func(t *testing.T) {
			var ps []*ui.Node
			for _, w := range []string{"0", "73px"} {
				for _, s := range texts {
					ps = append(ps, ui.P(ui.Style(st+"; width: "+w), ui.Text(s)))
				}
			}
			matchBreaks(t, page(ps...))
		})
	}
}
