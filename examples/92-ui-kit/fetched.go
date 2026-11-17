package main

import (
	"fmt"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/kit"
	"github.com/lantervale/ember/world"
)

// The frames a late picture is held back: a second and two at 60 a second.
const (
	portraitFrame = 60
	wallFrame     = 120
)

// late is a slow source of pictures, the page's ui.ImageFetcher: it
// holds each picture back until the page has drawn its frames, as a
// network would, then answers with panel.png. A game's fetcher reads
// its own cache or the web (web.Images) the same way.
type late struct {
	frames int
	after  map[string]chan struct{}
}

func newLate() *late {
	return &late{after: map[string]chan struct{}{"portrait.png": make(chan struct{}), "wall.png": make(chan struct{})}}
}

// fetch runs off the frame, so it can wait as long as its picture takes.
func (l *late) fetch(url string) ([]byte, error) {
	ready, ok := l.after[url]
	if !ok {
		return nil, fmt.Errorf("no picture %q", url)
	}
	<-ready
	return panelPNG, nil
}

func (l *late) Update(world.Frame) {
	l.frames++
	switch l.frames {
	case portraitFrame:
		close(l.after["portrait.png"])
	case wallFrame:
		close(l.after["wall.png"])
	}
}

// fetchedImages is two pictures no file of the Document holds, so its
// fetcher gets them: an img, sized once its picture lands, and a tiled
// background. Each stands on a plain colour until then.
func fetchedImages(*gallery) *ui.Node {
	const waiting = "background-color: #3a3f4b; image-rendering: pixelated"
	return kit.Panel(ui.Style("width: 380px; padding: 20px 24px"),
		ui.H2(ui.Text("Fetched pictures"), ui.Style("margin: 0 0 12px")),
		kit.Column(kit.Gap(12),
			kit.Row(kit.Gap(16),
				ui.Div(ui.Style("width: 72px; height: 72px; "+waiting),
					ui.Img(ui.ID("portrait"), ui.Attr("src", "portrait.png"), ui.Attr("alt", ""),
						ui.Style("display: block; width: 72px; image-rendering: pixelated"))),
				ui.Div(ui.ID("wall"), ui.Style("flex: 1; height: 72px; background-image: url(wall.png); "+waiting)),
			),
			kit.Label("An img after a second, a background after two."),
		),
	)
}
