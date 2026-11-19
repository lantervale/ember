// UI kit: each ui kit component alone in a window, with -element and its
// page's name, and the events it sends printed as they come.
//
//	go run ./examples/92-ui-kit -element button
package main

import (
	_ "embed"
	"flag"
	"log"
	"slices"
	"strings"
	"testing/fstest"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/examples/internal/lucide"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/kit"
	"github.com/lantervale/ember/world"
)

// panel.png is a framed 24×24 panel, the image and nine-patch elements' picture.
//
//go:embed panel.png
var panelPNG []byte

// sheet lays an element out on the kit's window colour, with room round it.
const sheet = `body { background-color: var(--window-color); padding: 16px; }
#contents { width: 380px; padding: 12px; border: 1px solid #8a7a5a; border-radius: 8px; background-color: #2b2f3a;
	color: #d8ceb4; font-size: 13px; line-height: 18px; position: relative; counter-reset: chapter; }
#contents::before { content: ""; position: absolute; top: 0; right: 0; width: 6px; height: 100%;
	background-color: #e0a040; border-radius: 0 8px 8px 0; }
#contents h4 { margin: 0 0 4px; font-size: 14px; color: #f2e6c9; counter-increment: chapter; }
#contents h4::before { content: counter(chapter, upper-roman) ". "; color: #e0a040; }
#contents h4[data-pages]::after { content: " · " attr(data-pages) " pages"; font-weight: 400; color: #8a7a5a; }
#contents ol { margin: 0 0 8px; padding-left: 28px; counter-reset: part; list-style: none; }
#contents li { counter-increment: part; }
#contents li::before { content: counter(chapter) "." counter(part) " "; color: #8a7a5a; }
#contents ul { margin: 0; padding-left: 28px; }
#contents ul li::before { content: none; }
#contents ul li::marker { content: "+ "; color: #e0a040; }
#contents q { color: #f2e6c9; quotes: "«" "»"; }
#letter { width: 380px; padding: 12px; border: 1px solid #8a7a5a; border-radius: 8px; background-color: #2b2f3a;
	color: #d8ceb4; font-size: 13px; line-height: 18px; user-select: text; }
#letter ::selection { background-color: #e0a040; color: #1d2129; }
#letter p { margin: 0 0 8px; }
#letter code { user-select: all; color: #f2e6c9; font-size: 12px; }
#letter .seal { margin: 0; user-select: none; color: #8a7a5a; }`

func main() {
	only := flag.String("element", "button", "the element to show, by its page's name: "+strings.Join(names(), ", "))
	flag.Parse()
	build, ok := elements()[*only]
	if !ok {
		log.Fatalf("no element %q; try one of %s", *only, strings.Join(names(), ", "))
	}

	g, err := page(build)
	if err != nil {
		log.Fatal(err)
	}
	win := platform.NewWindow(platform.Title("ember · "+*only), platform.Size(480, 320), platform.FixedSize(),
		platform.Background(color.Hex("#1d2129")))
	win.Add(g.doc)

	ember.New().AddEntities(win).AddSystems(append([]world.System{&Pointer{doc: g.doc}}, g.systems...)...).MustRun()
}

// gallery is what an element's page is built with: its Document, the
// Lucide icons whose Sheet it has, and the systems the page needs beside it.
type gallery struct {
	doc     *ui.Document
	icons   *kit.Icons
	systems []world.System
}

// page is a gallery showing what build makes, with the kit's look and
// icons; a picture its files don't hold comes late, from a fetcher.
func page(build func(*gallery) *ui.Node) (*gallery, error) {
	icons, err := lucide.Kit()
	if err != nil {
		return nil, err
	}
	files := fstest.MapFS{"page.css": {Data: []byte(sheet)}, "panel.png": {Data: panelPNG}}
	pictures := newLate()
	g := &gallery{
		doc: ui.NewDocument(kit.Theme(), icons.Sheet(), ui.Stylesheet(files, "page.css"),
			ui.FetchImages(pictures.fetch)),
		icons:   icons,
		systems: []world.System{pictures},
	}
	shown := build(g)
	logEvents(shown)
	g.doc.Body().Append(shown)
	return g, nil
}

// logEvents prints what the player does with each element under root,
// through its callbacks.
func logEvents(root *ui.Node) {
	root.Walk(func(n *ui.Node) bool {
		ui.On(n, ui.EventClick, func(e ui.Event) { log.Printf("clicked %s %q", e.Node.Tag(), e.Node.Text()) })
		ui.On(n, ui.EventInput, func(e ui.Event) { log.Printf("edited %s", e.Node.Tag()) })
		ui.On(n, ui.EventChange, func(e ui.Event) { log.Printf("changed %s", e.Node.Tag()) })
		ui.On(n, ui.EventSubmit, func(e ui.Event) { log.Printf("submitted %s", e.Node.Tag()) })
		ui.On(n, ui.EventClose, func(e ui.Event) { log.Printf("closed %s %q", e.Node.Tag(), e.Value) })
		return true
	})
}

func names() []string {
	var all []string
	for name := range elements() {
		all = append(all, name)
	}
	slices.Sort(all)
	return all
}
