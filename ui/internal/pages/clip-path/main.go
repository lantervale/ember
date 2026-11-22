// clip-path (decision 0119): each basic shape cut out of a bordered,
// padded box whose child spills past it, in each reference box, under a
// transform, round a scroller, and a clip-path holding another.
package main

import (
	"fmt"
	"strings"
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #fff; }
body { margin: 0; padding: 10px; display: grid; grid-template-columns: repeat(5, 116px); gap: 12px; }
.cell { height: 70px; border: 6px solid #345; padding: 8px; background-color: #fde9c8;
  box-shadow: 0 0 0 4px #e07a5f; }
.cell > div { height: 100px; margin: -14px; background: linear-gradient(135deg, #36c, #9bd 60%, #c33); }
.circle { clip-path: circle(40%); }
.farthest { clip-path: circle(farthest-side at 20% 30%); }
.ellipse { clip-path: ellipse(closest-side farthest-side at 30% 60%) content-box; }
.inset { clip-path: inset(10px 20% 15px round 12px 4px / 20px); }
.xywh { clip-path: xywh(10px 10% 60% 40px round 8px); }
.rect { clip-path: rect(5px auto 70% 15px); }
.star { clip-path: polygon(50% 0, 79% 90%, 2% 35%, 98% 35%, 21% 90%); }
.star-hole { clip-path: polygon(evenodd, 50% 0, 79% 90%, 2% 35%, 98% 35%, 21% 90%); }
.path { clip-path: path(evenodd, "M10 10 h100 v70 h-100 z M35 30 h50 v30 h-50 z"); }
.curve { clip-path: path("M0 50 C 30 -20, 90 120, 128 30 L 128 98 L 0 98 Z"); }
.padding { clip-path: padding-box; border-radius: 18px; }
.content { clip-path: content-box; }
.margin { margin: 6px; clip-path: inset(-12px) margin-box; }
.turned { transform: rotate(20deg); clip-path: polygon(0 0, 100% 0, 0 100%); }
.scroller { overflow: auto; clip-path: circle(50% at 50% 50%); }
.scroller > div { height: 160px; }
.nested { clip-path: inset(6px round 30px); }
.nested > div { clip-path: circle(45px at 60% 50%); margin: 0; }
.faded { opacity: 0.6; clip-path: ellipse(45% 35%); }
.empty { clip-path: circle(0); }
.over { clip-path: circle(60px at 0 0); }
`

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"clip-path.css": {Data: []byte(sheet)}}, "clip-path.css"))
	for i, class := range []string{
		"circle", "farthest", "ellipse", "inset", "xywh", "rect", "star", "star-hole", "path", "curve",
		"padding", "content", "margin", "turned", "scroller", "nested", "faded", "empty", "over",
	} {
		d.Body().Append(ui.Div(ui.Class(append([]string{"cell"}, strings.Fields(class)...)...),
			ui.Div(ui.Text(fmt.Sprint(i+1)))))
	}
	pages.Run(d, 680, 470)
}
