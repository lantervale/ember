// A page taller than its window, whose viewport scrolls, at its end: the
// rows the scroll leaves in view, a sticky head stuck to the window's top,
// a fixed badge where it was, and body's picture at the bottom right of
// the page, where the canvas has it.
package main

import (
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/pages"
)

func main() {
	d := pages.TallPage()
	dom.Data(d.HTML()).Scroll = geom.V2(0, 1e6)
	pages.Run(d, 320, 260)
}
