// Masks (decision 0133): a bordered, padded, shadowed box whose child
// spills past it, masked by a picture, gradients, several layers joined
// by each compositing word, luminance, the boxes mask-origin and
// mask-clip name, under a transform, a clip-path, opacity and a filter,
// a mask inside another, Chrome's -webkit-mask with xor, and an icon of
// a colour and a mask.
package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"strings"
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #fff; }
body { margin: 0; padding: 10px; display: grid; grid-template-columns: repeat(5, 116px); gap: 12px; }
.cell { height: 70px; border: 6px solid #345; padding: 8px; background: linear-gradient(135deg, #fde9c8, #9bd);
  box-shadow: 0 0 0 4px #e07a5f; }
.cell > div { height: 100px; margin: -14px 10px; background-color: #36c; }
.picture { mask: url(ring.png) center / contain no-repeat; }
.linear { mask-image: linear-gradient(#000, transparent); }
.radial { mask-image: radial-gradient(circle, #000 40%, transparent 70%); }
.tiled { mask: url(ring.png) 0 0 / 24px 24px; }
.space { mask: url(ring.png) 0 0 / 30px 30px space; }
.add { mask: url(ring.png) left / 60px no-repeat, linear-gradient(90deg, transparent 50%, #000 50%); }
.intersect { mask: url(ring.png) left / 90px no-repeat intersect, linear-gradient(90deg, transparent 30%, #000 30%); }
.subtract { mask: url(ring.png) center / 60px no-repeat subtract, linear-gradient(#000, #000); }
.exclude { mask: linear-gradient(#000, #000) content-box exclude, linear-gradient(#000, #000); }
.luminance { mask: linear-gradient(90deg, #fff, #000) luminance; }
.lumpicture { mask: url(ring.png) center / contain no-repeat luminance; }
.content { mask: url(ring.png) 0 0 / 40px no-repeat content-box; }
.padding { mask-image: linear-gradient(#000, #000); mask-clip: padding-box; }
.noclip { mask-image: linear-gradient(#000 0 0); mask-clip: no-clip; }
.round { border-radius: 30px; mask-image: linear-gradient(#000, rgba(0, 0, 0, 0.3)); }
.turned { transform: rotate(15deg); mask-image: linear-gradient(90deg, #000, transparent); }
.clipped { clip-path: circle(45%); mask-image: linear-gradient(#000, transparent); }
.faded { opacity: 0.6; mask-image: radial-gradient(#000 50%, transparent 70%); }
.filtered { filter: grayscale(1); mask-image: linear-gradient(transparent, #000); }
.nested { mask-image: linear-gradient(90deg, #000 60%, transparent); }
.nested > div { mask-image: linear-gradient(transparent 20%, #000); }
.webkit { -webkit-mask: linear-gradient(#000 0 0) content-box, linear-gradient(#000 0 0);
  -webkit-mask-composite: xor; padding: 12px; }
.icons { display: flex; gap: 8px; align-items: center; }
.icon { width: 24px; height: 24px; background-color: #c33; mask: url(ring.png) center / contain no-repeat; }
.icon + .icon { background-color: #36c; mask-size: 16px; mask-repeat: repeat; }
`

// ring is a picture 32 by 32: a coloured ring, solid round its middle,
// its edges fading out.
func ring() []byte {
	m := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	for y := range 32 {
		for x := range 32 {
			d := math.Hypot(float64(x)-15.5, float64(y)-15.5)
			a := 1 - math.Abs(d-10)/6
			if a <= 0 {
				continue
			}
			m.Set(x, y, color.NRGBA{R: uint8(x * 8), G: 0xc0, B: uint8(255 - y*8), A: uint8(math.Min(a*1.5, 1) * 255)})
		}
	}
	var b bytes.Buffer
	if err := png.Encode(&b, m); err != nil {
		panic(err)
	}
	return b.Bytes()
}

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"masks.css": {Data: []byte(sheet)}, "ring.png": {Data: ring()}},
		"masks.css"), ui.BrowserSheets())
	for i, class := range []string{
		"picture", "linear", "radial", "tiled", "space", "add", "intersect", "subtract", "exclude", "luminance",
		"lumpicture", "content", "padding", "noclip", "round", "turned", "clipped", "faded", "filtered", "nested",
		"webkit",
	} {
		d.Body().Append(ui.Div(ui.Class(append([]string{"cell"}, strings.Fields(class)...)...),
			ui.Div(ui.Text(fmt.Sprint(i+1)))))
	}
	d.Body().Append(ui.Div(ui.Class("icons"), ui.Span(ui.Class("icon")), ui.Span(ui.Class("icon"))))
	pages.Run(d, 680, 560)
}
