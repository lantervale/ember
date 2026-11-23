// Gradients and pictures turned, skewed and scaled with their box: linear,
// radial and conic backgrounds, a gradient tiled by background-size, a
// picture repeated and alone, a rounded border, a gradient border-image,
// a gradient shown through text and a box masked by a gradient, each under
// rotate, a skew and a scale with a turn.
package main

import (
	_ "embed"
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

//go:embed beach.png
var beach []byte

const sheet = `
html { background-color: #fff; }
body { margin: 0; padding: 20px; display: grid; grid-template-columns: repeat(6, 80px); gap: 34px 26px; }
.b { height: 50px; background-color: #ddd; }
.turn { rotate: 30deg; }
.skew { transform: skewX(-20deg); }
.scale { transform: scale(0.8, 1.3) rotate(-15deg); }
.linear { background: linear-gradient(90deg, #c33, #f0b545 50%, #36c); }
.corner { background: linear-gradient(to bottom right, #2a8, #345); }
.radial { background: radial-gradient(circle at 30% 40%, #f0b545, #c33 50%, #345); }
.conic { background: conic-gradient(from 45deg, #c33, #f0b545, #2a8, #36c, #c33); }
.tiled { background: linear-gradient(45deg, #36c 50%, #f0b545 50%) 0 0 / 20px 20px; }
.picture { background: #345 url(beach.png) 0 0 / 40px auto; }
.alone { background: #cde url(beach.png) center / 50px auto no-repeat; }
.round { border: 4px solid #345; border-radius: 16px; background: linear-gradient(#c33, #36c); }
.frame { height: 30px; border: 10px solid; border-image: linear-gradient(90deg, #c33, #36c) 10;
  background: #fde9c8; }
.text { height: auto; font: bold 36px sans-serif; background: linear-gradient(90deg, #c33, #36c) text;
  color: transparent; }
.masked { background: #36c; mask-image: linear-gradient(90deg, #000, transparent); }
.clipped { border: 6px solid #345; background: linear-gradient(#f0b545, #2a8) padding-box, #c33; }
`

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{
		"t.css":     {Data: []byte(sheet)},
		"beach.png": {Data: beach},
	}, "t.css"))
	kinds := []string{"linear", "corner", "radial", "conic", "tiled", "picture", "alone", "round", "frame", "text",
		"masked", "clipped"}
	for _, turn := range []string{"turn", "skew", "scale"} {
		for _, k := range kinds {
			args := []ui.Arg{ui.Class("b", k, turn)}
			if k == "text" {
				args = append(args, ui.Text("Ab"))
			}
			d.Body().Append(ui.Div(args...))
		}
	}
	pages.Run(d, 640, 560)
}
