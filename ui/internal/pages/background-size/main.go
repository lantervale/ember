// background-size against Chrome's: a 64×44 picture sized by percentages,
// by one side and auto, by cover and contain, placed by
// background-position and tiled each way background-repeat has, under a
// rounded border, pixelated, and through the background shorthand; then
// gradients, which have no size of their own.
package main

import (
	_ "embed"
	"strings"
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

//go:embed beach.png
var beach []byte

const sheet = `
html { background-color: #1b1714; }
body { margin: 0; padding: 10px; display: grid; grid-template-columns: repeat(5, 120px); gap: 10px; }
div { height: 70px; background-color: #3a332c; background-image: url(beach.png); background-repeat: no-repeat; }
.half { background-size: 50% 50%; }
.width { background-size: 50% auto; background-position: center; }
.height { background-size: auto 100%; background-position: right top; }
.cover { background-size: cover; background-position: center; }
.contain { background-size: contain; background-position: center; }
.tiled-contain { background-size: contain; background-position: right; background-repeat: repeat; }
.tiled { background-size: 25% 40%; background-repeat: repeat; }
.across { background-size: 30% auto; background-position: 10px bottom; background-repeat: repeat-x; }
.down { background-size: auto 50%; background-position: right 10px top 5px; background-repeat: repeat-y; }
.round { background-size: cover; border: 3px solid #f0b545; border-radius: 16px; box-sizing: border-box; }
.pixelated { background-size: contain; image-rendering: pixelated; }
.tall { width: 50px; background-size: cover; background-position: 25% 0; }
.calc { background-size: calc(100% - 20px) 30px; background-position: 10px 10px; }
.short { background: #5a4630 url(beach.png) no-repeat left 6px bottom 4px / 80% auto; }
.gradient { background-image: linear-gradient(90deg, #d9534f, #337ab7); }
.gradient.part { background-size: 50% 50%; background-position: center; }
.gradient.stripes { background-size: 25% 100%; background-repeat: repeat; }
.gradient.band { background-size: auto 20px; background-position: bottom; }
.gradient.whole { background-size: cover; }
.gradient.dots { background-image: radial-gradient(circle, #f0b545 30%, #2b2722 32%); background-size: 20px 20px;
  background-repeat: repeat; border-radius: 12px; }
`

func main() {
	files := fstest.MapFS{"page.css": {Data: []byte(sheet)}, "beach.png": {Data: beach}}
	d := ui.NewDocument(ui.Stylesheet(files, "page.css"))
	for _, class := range []string{
		"half", "width", "height", "cover", "contain", "tiled-contain", "tiled", "across", "down", "round", "pixelated",
		"tall", "calc", "short", "gradient part", "gradient stripes", "gradient band", "gradient whole", "gradient dots",
	} {
		d.Body().Append(ui.Div(ui.Class(strings.Fields(class)...)))
	}
	pages.Run(d, 660, 330)
}
