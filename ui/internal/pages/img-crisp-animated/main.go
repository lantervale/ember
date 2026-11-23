// Animated GIFs scaled up, as an img and as a background: each plays its
// two frames once and stays on its last, a checker of other colours.
// Pixelated, its texels stay sharp as the first frame's would; crisp-edges
// smooths them as Chrome does, as auto does beside them.
package main

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"os"
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
	_ "github.com/lantervale/ember/ui/web"
)

const sheet = `
html { background-color: #ffffff; }
body { margin: 0; padding: 8px; display: grid; grid-template-columns: repeat(3, 48px); gap: 8px; }
img, div { display: block; width: 48px; height: 48px; }
div { background-image: url(flip.gif); background-size: 48px 48px; }
.pixelated { image-rendering: pixelated; }
.crisp { image-rendering: crisp-edges; }
`

// flip is an 8×8 GIF played once: a red and blue checker for 20 ms,
// then a green and yellow one.
func flip() []byte {
	pal := color.Palette{color.RGBA{255, 0, 0, 255}, color.RGBA{0, 0, 255, 255},
		color.RGBA{0, 160, 0, 255}, color.RGBA{255, 220, 0, 255}}
	frame := func(a, b uint8) *image.Paletted {
		p := image.NewPaletted(image.Rect(0, 0, 8, 8), pal)
		for y := range 8 {
			for x := range 8 {
				p.Pix[y*8+x] = a
				if (x+y)%2 == 1 {
					p.Pix[y*8+x] = b
				}
			}
		}
		return p
	}
	var b bytes.Buffer
	_ = gif.EncodeAll(&b, &gif.GIF{Image: []*image.Paletted{frame(0, 1), frame(2, 3)}, Delay: []int{2, 2},
		LoopCount: -1, Config: image.Config{ColorModel: pal, Width: 8, Height: 8}})
	return b.Bytes()
}

func main() {
	// An animation begins once drawn, as Chrome's, which drew it long
	// before its shot: every frame before the captured one is drawn.
	_ = os.Setenv("EMBER_CAPTURE_DRAW", "all")
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(sheet)}, "flip.gif": {Data: flip()}},
		"page.css"))
	for _, look := range []string{"pixelated", "crisp", "smooth"} {
		d.Body().Append(ui.Img(ui.Attr("src", "flip.gif"), ui.Class(look)))
	}
	for _, look := range []string{"pixelated", "crisp", "smooth"} {
		d.Body().Append(ui.Div(ui.Class(look)))
	}
	pages.Run(d, 176, 120)
}
