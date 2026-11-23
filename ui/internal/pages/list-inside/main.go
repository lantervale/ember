// Lists against Chrome's: markers inside their items, the first box of
// their first line (bullets, numbers, strings, ::marker's content,
// pictures and gradients, in an empty item and before a block), gradients
// outside, and counter styles symbols() makes.
package main

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #fff; }
body { margin: 0; padding: 10px; font-size: 13px; line-height: 17px; color: #2b2722; display: grid;
	grid-template-columns: 1fr 1fr 1fr; gap: 6px 10px; }
ol, ul { margin: 0; padding-left: 24px; background-color: #f4f1ec; list-style-position: inside; }
.c { list-style-type: circle; }
.s { list-style-type: square; color: #3c6fb0; }
.d { list-style-type: decimal; }
.r { list-style-type: upper-roman; }
.str { list-style-type: "-> "; }
.big li:nth-child(2) { font-size: 20px; line-height: 24px; }
.m li::marker { content: "§ "; color: #b0493c; font-weight: 700; }
.img { list-style-image: url(dot.png); }
.g { list-style-image: linear-gradient(#b0493c, #3c6fb0); }
.g li:nth-child(2) { font-size: 24px; line-height: 28px; }
.rg { list-style-image: radial-gradient(#3c8a4f, #f4f1ec); }
.out { list-style: outside linear-gradient(to right, #3c8a4f, #b0493c); padding-left: 34px; }
.blk p { margin: 0; color: #3c6fb0; }
.narrow { width: 110px; }
.sym { list-style: outside symbols(cyclic "*" "+" "#"); padding-left: 34px; }
.alpha { list-style: inside symbols(alphabetic "x" "y"); }
.fix { list-style-type: symbols(fixed "a." "b."); }
`

func picture(w, h int, c color.NRGBA) []byte {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: c}, image.Point{}, draw.Src)
	var b bytes.Buffer
	_ = png.Encode(&b, img)
	return b.Bytes()
}

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"list-inside.css": {Data: []byte(sheet)}}, "list-inside.css"),
		ui.Images(fstest.MapFS{
			"dot.png": {Data: picture(8, 8, color.NRGBA{R: 0x3c, G: 0x8a, B: 0x4f, A: 0xff})},
		}))
	list := func(tag func(...ui.Arg) *ui.Node, n int, args ...ui.Arg) *ui.Node {
		for range n {
			args = append(args, ui.Li(ui.Text("item")))
		}
		return tag(args...)
	}
	d.Body().Append(
		list(ui.Ul, 3, ui.Class("big")),
		list(ui.Ul, 3, ui.Class("c")),
		list(ui.Ul, 3, ui.Class("s")),
		list(ui.Ol, 3, ui.Class("d"), ui.Attr("start", "9")),
		list(ui.Ol, 3, ui.Class("r")),
		list(ui.Ul, 2, ui.Class("str")),
		list(ui.Ul, 2, ui.Class("m")),
		list(ui.Ul, 2, ui.Class("img")),
		list(ui.Ul, 2, ui.Class("g")),
		list(ui.Ul, 2, ui.Class("rg")),
		list(ui.Ul, 2, ui.Class("out")),
		ui.Ol(ui.Class("d", "blk"), ui.Li(ui.P(ui.Text("a block"))), ui.Li(), ui.Li(ui.Text("lead"),
			ui.P(ui.Text("a block")))),
		ui.Ol(ui.Class("d", "narrow"), ui.Li(ui.Text("words that wrap past the first line"))),
		list(ui.Ul, 3, ui.Class("sym")),
		list(ui.Ol, 3, ui.Class("alpha")),
		list(ui.Ol, 3, ui.Class("fix")),
	)
	pages.Run(d, 520, 360)
}
