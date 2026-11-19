package ui_test

import (
	"fmt"
	"testing"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/internal/draw"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/cull"
	"github.com/lantervale/ember/world"
)

const longSheet = `
html { scrollbar-width: none; }
body { margin: 0 24px; font-size: 14px; }
h2 { margin: 12px 0 4px; border-bottom: 1px solid #a2a9b1; }
p { margin: 6px 0; }
a { color: #3366cc; }
a:hover { text-decoration: underline; }
.thumb { float: right; width: 180px; height: 120px; margin: 0 0 8px 12px; background: #eaecf0;
  border: 1px solid #c8ccd1; }
.note { position: relative; top: -2px; font-size: 11px; }
`

// longPage is an article of sections, each a heading, a floated thumbnail
// and three paragraphs of text with links and raised notes, in a window of
// 1000×700 whose viewport scrolls: some 16,000 boxes for 1,000 sections,
// as a long encyclopedia article has.
func longPage(tb testing.TB, sections int) (*ui.Document, *world.World, *platform.Window) {
	tb.Helper()
	doc := ui.NewDocument(ui.ScrollingViewport(), ui.CSS(longSheet))
	for i := range sections {
		doc.Body().Append(ui.H2(ui.Text(fmt.Sprintf("Section %d", i))), ui.Div(ui.Class("thumb")))
		for range 3 {
			doc.Body().Append(ui.P(
				ui.Text("A paragraph of the article, long enough to wrap over a few lines of the window, with "),
				ui.A(ui.Attr("href", "#"), ui.Text("a link")), ui.Text(" in it and a note"),
				ui.Span(ui.Class("note"), ui.Text("[1]")),
				ui.Text(" after which the text goes on for a while, as an article's does, to the end of its paragraph.")))
		}
	}
	win := platform.NewWindow(platform.Size(1000, 700))
	win.Add(doc)
	w := ember.New().AddEntities(win)
	w.RunFrames(3)
	return doc, w, win
}

// eachWalk runs bench with paint skipping the boxes out of view, as it
// does, and walking every box, as it did before decision 0101.
func eachWalk(b *testing.B, bench func(b *testing.B)) {
	for _, m := range []struct {
		name string
		mode cull.Mode
	}{{"culled", cull.On}, {"whole", cull.Off}} {
		b.Run(m.name, func(b *testing.B) {
			defer cull.Set(m.mode, nil)()
			bench(b)
		})
	}
}

// BenchmarkLongPageScrollFrame is a whole frame of a long page whose
// viewport scrolled: input, the layout's refresh, the hit test, the paint
// and the draw list, with the vertices the frame made.
func BenchmarkLongPageScrollFrame(b *testing.B) {
	eachWalk(b, func(b *testing.B) {
		doc, w, win := longPage(b, 1000)
		high := ui.ScrollSize(doc, doc.HTML()).Y - 700
		vertices := 0
		b.ReportAllocs()
		b.ResetTimer()
		for i := range b.N {
			ui.ScrollTo(doc, doc.HTML(), geom.V2(0, float32(i*37%int(high))))
			w.RunFrames(1)
			vertices += len(draw.FrontFor(w, win).Vertices)
		}
		b.ReportMetric(float64(vertices)/float64(b.N), "vertices/frame")
	})
}

// BenchmarkLongPagePointerFrame is a whole frame of a long page, scrolled
// to its middle, in which the pointer moved.
func BenchmarkLongPagePointerFrame(b *testing.B) {
	eachWalk(b, func(b *testing.B) {
		doc, w, win := longPage(b, 1000)
		ui.ScrollTo(doc, doc.HTML(), geom.V2(0, ui.ScrollSize(doc, doc.HTML()).Y/2))
		w.RunFrames(2)
		b.ReportAllocs()
		b.ResetTimer()
		for i := range b.N {
			win.Input().SetMouse(geom.V2(float32(40+i*13%900), float32(30+i*7%640)))
			w.RunFrames(1)
		}
	})
}

// BenchmarkLongPageElementAt is one hit test of a long page scrolled to
// its middle.
func BenchmarkLongPageElementAt(b *testing.B) {
	eachWalk(b, func(b *testing.B) {
		doc, w, _ := longPage(b, 1000)
		ui.ScrollTo(doc, doc.HTML(), geom.V2(0, ui.ScrollSize(doc, doc.HTML()).Y/2))
		w.RunFrames(2)
		b.ReportAllocs()
		b.ResetTimer()
		for i := range b.N {
			doc.ElementAt(geom.V2(float32(40+i*13%900), float32(30+i*7%640)))
		}
	})
}
