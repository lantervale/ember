package main

import (
	_ "embed"
	"testing/fstest"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/world"
)

func structure() map[string]element {
	return map[string]element{
		"details": {details, 320},
		"summary": {summary, 240},
		"dialog":  {dialog, 232},
		"table":   {scores, 224},
		"thead":   {scores, 224},
		"tbody":   {scores, 224},
		"tr":      {scores, 224},
		"th":      {scores, 224},
		"td":      {scores, 224},
		"caption": {scores, 224},
		"tfoot":   {scores, 224},
		"img":     {img, 136},
		"svg":     {svg, 96},
		"hr":      {hr, 152},
		"canvas":  {canvas, 232},
		"iframe":  {iframe, 336},
	}
}

// panel.png is a framed 24×24 panel, the img element's picture.
//
//go:embed panel.png
var panelPNG []byte

// pictures are the files a page's img elements name.
func pictures() fstest.MapFS {
	return fstest.MapFS{"panel.png": {Data: panelPNG}}
}

// details is a quest log: one quest open, one closed.
func details(*ui.Document) *ui.Node {
	return ui.Div(
		ui.Details(ui.Open(),
			ui.Summary(ui.Text("The old well")),
			ui.P(ui.Text("Bring three buckets of water to the mill.")),
		),
		ui.Details(
			ui.Summary(ui.Text("Lost keys")),
			ui.P(ui.Text("Find the keeper's keys in the crypt.")),
		),
	)
}

// summary is a closed inventory, its summary holding a count in emphasis.
func summary(*ui.Document) *ui.Node {
	return ui.Details(
		ui.Summary(ui.Text("Inventory "), ui.Em(ui.Text("12 of 20 slots"))),
		ui.P(ui.Text("A rope, a lantern, three apples.")),
	)
}

// dialog is a quit dialog shown open over the page.
func dialog(d *ui.Document) *ui.Node {
	quit := ui.Dialog(ui.ID("quit"),
		ui.P(ui.Text("Quit to the title screen?")),
		ui.Button(ui.Text("Stay"), ui.Attr("formmethod", "dialog"), ui.Value("stay")),
		ui.Text(" "),
		ui.Button(ui.Text("Quit"), ui.Attr("formmethod", "dialog"), ui.Value("quit")),
	)
	d.Show(quit)
	return quit
}

// scores is a score table: a grid, sorted by a press on a header, by Score
// now, under its caption, Player spanning both header rows and Run its two
// columns, with the totals in its footer.
func scores(*ui.Document) *ui.Node {
	row := func(name, score, time string) *ui.Node {
		return ui.Tr(ui.Td(ui.Text(name)), ui.Td(ui.Text(score)), ui.Td(ui.Text(time)))
	}
	return ui.Table(ui.Attr("role", "grid"),
		ui.Caption(ui.Text("Arena, week 12")),
		ui.Thead(
			ui.Tr(
				ui.Th(ui.Text("Player"), ui.Attr("rowspan", "2")),
				ui.Th(ui.Text("Run"), ui.Attr("colspan", "2")),
			),
			ui.Tr(
				ui.Th(ui.Text("Score"), ui.Attr("aria-sort", "descending")),
				ui.Th(ui.Text("Time"), ui.Attr("aria-sort", "none")),
			),
		),
		ui.Tbody(
			row("Ayla", "4210", "12:40"),
			row("Bram", "3875", "11:02"),
			row("Corin", "2950", "9:18"),
		),
		ui.Tfoot(row("Total", "11035", "33:00")),
	)
}

// img is the panel picture at its own size and drawn three times larger.
func img(*ui.Document) *ui.Node {
	return ui.Div(
		ui.Img(ui.Attr("src", "panel.png"), ui.Attr("alt", "A panel")),
		ui.Text(" "),
		ui.Img(ui.Attr("src", "panel.png"), ui.Style("width: 72px; image-rendering: pixelated")),
	)
}

// svg is a compass drawn in the page, in the colour of its text, small
// and large.
func svg(*ui.Document) *ui.Node {
	const compass = `<circle cx="12" cy="12" r="10" fill="none" stroke="currentColor" stroke-width="2"/>` +
		`<path d="M12 4l3 8-3 8-3-8z" fill="currentColor"/>`
	return ui.Div(ui.Style("color: #e0a040"),
		ui.Svg(compass, ui.Attr("viewBox", "0 0 24 24"), ui.Attr("width", "24")),
		ui.Text(" "),
		ui.Svg(compass, ui.Attr("viewBox", "0 0 24 24"), ui.Style("width: 72px; color: #7fb0d8")),
	)
}

// iframe is a help page in a frame: a Document of its own, laid out in
// the frame's box and scrolled there.
func iframe(*ui.Document) *ui.Node {
	help := ui.NewDocument(ui.ScrollingViewport(), ui.CSS(`
		html { background-color: #1d2129; color: #d8d2c4; }
		body { margin: 12px; }
		h3 { margin: 0 0 8px; color: #e0a040; }
		p { margin: 0 0 8px; }`))
	help.Body().Append(
		ui.H3(ui.Text("How to play")),
		ui.P(ui.Text("Light the lantern before the stairs. The dark takes a heart a minute.")),
		ui.P(ui.Text("Slimes fear the light: keep them at the edge of it and they stay there.")),
		ui.P(ui.Text("Doors with a rune open to the key of the same colour, found a floor above.")),
		ui.Button(ui.Text("Got it")),
	)
	return ui.Iframe(help, ui.ID("help"), ui.Attr("height", "160"))
}

// hr is a pause menu split in two by a rule.
func hr(*ui.Document) *ui.Node {
	return ui.Div(ui.Style("width: 120px"),
		ui.Div(ui.Text("Resume")),
		ui.Div(ui.Text("Settings")),
		ui.Hr(),
		ui.Div(ui.Text("Quit")),
	)
}

// canvas is a minimap: rooms drawn by the game, a marker where the pointer
// is, drawn again each frame by Pointer.
func canvas(d *ui.Document) *ui.Node {
	var minimap *ui.Node
	minimap = ui.Canvas(func(c *twod.Canvas) {
		c.Rect(geom.Rect{W: c.Size().X, H: c.Size().Y}, color.Hex("#1d2129"))
		rooms := []geom.Rect{
			{X: 20, Y: 20, W: 70, H: 50}, {X: 110, Y: 30, W: 60, H: 90}, {X: 30, Y: 90, W: 60, H: 40},
		}
		for _, room := range rooms {
			c.Rect(room, color.Hex("#5a4a30"))
		}
		if at, ok := d.PointerOn(minimap); ok {
			c.Circle(at, 5, color.Hex("#e0a040"))
		}
	}, ui.ID("minimap"), ui.Attr("width", "200"), ui.Attr("height", "150"))
	return minimap
}

// Pointer has the minimap drawn again each frame, so its marker follows
// the pointer; a canvas left alone keeps its last drawing.
type Pointer struct{ doc *ui.Document }

func (p *Pointer) Update(world.Frame) {
	if minimap := p.doc.ByID("minimap"); minimap != nil {
		p.doc.Invalidate(minimap)
	}
}
