//go:build chrome

package ui_test

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
)

// dropCapWithin is how far a drop cap's box may stand from Chrome's: on
// Linux and Windows Chrome rounds a letter's ink out to whole pixels.
func dropCapWithin() float32 {
	if runtime.GOOS != "darwin" {
		return 2
	}
	return chrome.Tolerance
}

// Chrome reads initial-letter on an element, the first inline child of
// its block included, and lays it out as if it were normal: only a
// ::first-letter is a drop cap.
func TestInitialLetterOnAnElementIsChromes(t *testing.T) {
	d := ui.NewDocument(ui.CSS(`
body { margin: 8px; font-size: 16px; width: 360px; }
p { margin: 0 0 10px; }
.il { initial-letter: 3; }
.ib { display: inline-block; }
.bl { display: block; }
.fl { float: left; }
img.il { width: 20px; height: 20px; }
`))
	first := func(lead ui.Arg) *ui.Node {
		return ui.P(append([]ui.Arg{lead, ui.Text(" ")}, words(prose)...)...)
	}
	d.Body().Append(
		first(ui.Span(ui.Class("il"), ui.Text("Down"))),
		first(ui.Span(ui.Class("il", "ib"), ui.Text("D"))),
		first(ui.Span(ui.Class("il", "bl"), ui.Text("D"))),
		first(ui.Span(ui.Class("il", "fl"), ui.Text("D"))),
		first(ui.Img(ui.Class("il"))),
	)
	r := chrome.Match(t, d, geom.V2(400, 600))
	t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
	out, err := chrome.Eval(d, geom.V2(400, 600),
		`return getComputedStyle(document.querySelector(".il")).fontSize`)
	require.NoError(t, err)
	assert.JSONEq(t, `"16px"`, string(out), "Chrome sizes the element as without initial-letter")
}

// Chrome has no initial-letter-align: it drops the declaration, as ui
// does with a warning.
func TestInitialLetterAlignIsRefusedAsChromeRefusesIt(t *testing.T) {
	d := ui.NewDocument(ui.CSS(`p::first-letter { initial-letter: 3; initial-letter-align: alphabetic; }`))
	out, err := chrome.Eval(d, geom.V2(200, 100), `return ["alphabetic", "ideographic", "hanging", "leading",
	"border-box alphabetic"].some(v => CSS.supports("initial-letter-align", v))`)
	require.NoError(t, err)
	assert.JSONEq(t, "false", string(out))
	require.Len(t, d.Warnings(), 1)
	assert.Contains(t, d.Warnings()[0].Error(), "initial-letter-align")
}

// A drop cap in vertical text stands at the start of its lines as in
// horizontal text turned: its capitals across the size's lines, the lines
// beside it going round it.
func TestVerticalInitialLettersMatchChrome(t *testing.T) {
	for _, mode := range []string{"vertical-rl", "vertical-lr", "sideways-rl", "sideways-lr"} {
		t.Run(mode, func(t *testing.T) {
			d := ui.NewDocument(ui.CSS(`
body { margin: 8px; font-size: 16px; }
p { margin: 0 12px 0 0; writing-mode: ` + mode + `; height: 300px; display: inline-block; }
.i3::first-letter { initial-letter: 3; }
.i2::first-letter { initial-letter: 2; color: #a00; }
.raise::first-letter { initial-letter: 3 2; }
.boxed::first-letter { initial-letter: 2; padding: 2px 4px; margin-right: 4px; border: 2px solid #357; }
.upright { text-orientation: upright; }
.upright::first-letter { initial-letter: 2; }
`))
			for _, c := range []string{"i3", "i2", "raise", "boxed", "upright"} {
				d.Body().Append(ui.P(append([]ui.Arg{ui.Class(c), ui.Text("Down ")}, words(prose)...)...))
			}
			r := chrome.MatchWithin(t, d, geom.V2(800, 400), dropCapWithin())
			t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
		})
	}
}

// An ideograph standing upright as a drop cap takes down its lines what
// its ink takes down its room.
func TestVerticalIdeographInitialLettersMatchChrome(t *testing.T) {
	d := cjkDocument(t, `
body { margin: 8px; font-size: 16px; line-height: 20px; }
p { margin: 0 12px 0 0; writing-mode: vertical-rl; height: 240px; display: inline-block; }
.lr { writing-mode: vertical-lr; }
p::first-letter { initial-letter: 2; }
.i3::first-letter { initial-letter: 3; }
`)
	text := []string{"丄丅丆", "万丈三上下", "不与丏丐丑", "丒专且丕世", "丗丘丙一丁", "丂七丄丅丆"}
	for _, c := range [][]string{{"rl"}, {"lr"}, {"rl", "i3"}} {
		d.Body().Append(ui.P(append([]ui.Arg{ui.Class(c...), ui.Text("一丁丂七 ")}, wordSpans(text...)...)...))
	}
	landed(t, d, geom.V2(800, 300))
	r := chrome.MatchWithin(t, d, geom.V2(800, 300), dropCapWithin())
	t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
}
