package main

import (
	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/kit"
	"github.com/lantervale/ember/world"
	"log"
)

// settings is a small settings screen: a heading, a form and two buttons.
func settings(*gallery) *ui.Node {
	return kit.Panel(ui.Style("width: 380px; padding: 20px 24px"),
		ui.H2(ui.Text("Settings"), ui.Style("margin: 0 0 12px")),
		kit.Form(
			kit.Field("Name", kit.TextInput("Ayla", ui.ID("name"))),
			kit.Field("Volume", kit.Slider(60, ui.ID("volume"))),
			kit.Field("Music", kit.Switch("", ui.ID("music"), ui.Checked(true))),
		),
		kit.Row(kit.Gap(8), kit.JustifyEnd(), ui.Style("margin-top: 12px"),
			kit.Button("Cancel"),
			kit.Button("Save", kit.Primary(), ui.ID("save")),
		),
	)
}

// styled is plain elements given their look by the style attribute.
func styled(*gallery) *ui.Node {
	return ui.Div(
		ui.Style(
			"display: grid; grid-template-columns: 48px 1fr; gap: 4px 12px; align-items: center; padding: 12px; "+
				"border: 1px solid #8a7a5a; border-radius: 8px; background-color: #2b2f3a; width: 280px",
		),
		ui.Div(ui.Style("grid-row: span 2; width: 48px; height: 48px; border-radius: 24px; background-color: #e0a040")),
		ui.Strong(ui.Text("Ember Quest"), ui.Style("color: #f2e6c9; font-size: 18px")),
		ui.Span(ui.Text("Area 2 · 1.5k to home"), ui.Style("color: #9a9384; font-size: 13px")),
	)
}

// floated is a quest's text going round what floats: a portrait on the
// left, a tip on the right, each paragraph's first line indented, and a
// reward that clears them both.
func floated(*gallery) *ui.Node {
	return ui.Div(
		ui.Style("display: flow-root; width: 380px; padding: 12px; border: 1px solid #8a7a5a; border-radius: 8px; "+
			"background-color: #2b2f3a; color: #d8ceb4; font-size: 13px; line-height: 18px; text-indent: 16px"),
		ui.Div(ui.ID("portrait"), ui.Style("float: left; width: 64px; height: 64px; margin: 2px 12px 6px 0; "+
			"border-radius: 8px; background-color: #e0a040")),
		ui.P(ui.ID("first"), ui.Style("margin: 0 0 6px"), ui.Text("The lamplighter of Lantervale has lost her ember. "+
			"She last had it at the mill, before the fog came down from the hills and the road went dark. "+
			"Find it before the night bell, and the lamps of the old bridge will burn again.")),
		ui.Div(ui.ID("tip"), ui.Style("float: right; width: 96px; margin: 2px 0 6px 12px; padding: 6px 8px; "+
			"border-radius: 6px; background-color: #3a4152; color: #9a9384; font-size: 12px; text-indent: 0"),
			ui.Text("Tip: the miller only talks after dusk, and never for free.")),
		ui.P(ui.ID("second"), ui.Style("margin: 0 0 6px"), ui.Text("Ask at the mill first.")),
		ui.Div(ui.ID("reward"), ui.Style("clear: both; padding-top: 6px; border-top: 1px solid #4a4f5c; "+
			"color: #f2e6c9; text-indent: 0"), ui.Strong(ui.Text("Reward")), ui.Text(" · 40 gold and a lit lantern")),
	)
}

// columned is a traveller's journal in two columns with a rule between
// them: the entries are cut between lines, the one with break-inside:
// avoid kept whole.
func columned(*gallery) *ui.Node {
	entry := func(id, style, day, text string) *ui.Node {
		return ui.P(ui.ID(id), ui.Style("margin: 0 0 8px; "+style), ui.Strong(ui.Text(day), ui.Style("color: #f2e6c9")),
			ui.Text(" "+text))
	}
	return ui.Div(
		ui.ID("journal"),
		ui.Style("width: 380px; padding: 12px; border: 1px solid #8a7a5a; border-radius: 8px; "+
			"background-color: #2b2f3a; color: #d8ceb4; font-size: 13px; line-height: 18px; "+
			"column-count: 2; column-gap: 24px; column-rule: 1px solid #4a4f5c"),
		entry("monday", "", "Monday.", "Left the mill at dawn with the lamplighter's map. The fog sat low on the "+
			"road, and the old bridge was dark from end to end."),
		entry("tuesday", "", "Tuesday.", "The miller would not talk before dusk. Waited by the weir, counted "+
			"herons, and paid him two gold for a story about the hills."),
		entry("wednesday", "break-inside: avoid", "Wednesday.", "Found the ember under the third arch, still "+
			"warm. The lamps burn again."),
	)
}

// generated is a journal's contents drawn by pseudo-elements: chapters and
// their parts numbered by counters, a page count from an attribute, a
// bar down the side from an empty ::before, markers of the sheet's own
// and a quotation in the marks its quotes property names.
func generated(*gallery) *ui.Node {
	return ui.Div(
		ui.ID("contents"),
		ui.H4(ui.ID("road"), ui.Attr("data-pages", "12"), ui.Text("The road")),
		ui.Ol(ui.Li(ui.ID("mill"), ui.Text("Leaving the mill")), ui.Li(ui.Text("The dark bridge"))),
		ui.H4(ui.ID("hills"), ui.Text("The hills")),
		ui.Ol(ui.Li(ui.ID("miller"), ui.Text("The miller's story")), ui.Li(ui.Text("Under the third arch"))),
		ui.Ul(ui.Li(ui.ID("note"), ui.Text("He said "), ui.Q(ui.ID("said"), ui.Text("never for free")), ui.Text("."))),
	)
}

// selection is a letter the player selects and copies: its text takes a
// selection where the kit's takes none, in the sheet's ::selection
// colours, the password whole or not at all, the seal never. It opens
// with a part selected by the game.
func selection(g *gallery) *ui.Node {
	first := ui.P(ui.ID("greeting"), ui.Text("Aldric, the crypt under the mill is open again."))
	second := ui.P(ui.Text("Say "), ui.Code(ui.ID("password"), ui.Text("ember and ash")),
		ui.Text(" at the third arch and the keeper lets you through."))
	g.doc.Select(ui.Position{Node: first.Children()[0], Offset: 12}, ui.Position{Node: second.Children()[2], Offset: 18})
	ui.OnSelectionChange(func() { log.Printf("selected %q", g.doc.Selection().Text) }).Apply(g.doc.Body())
	return ui.Div(ui.ID("letter"), first, second, ui.P(ui.Class("seal"), ui.Text("Sealed by the miller")))
}

// mapCanvas is the canvas element: a map of rooms drawn by the game, and a
// marker where the pointer is, which Pointer redraws each frame.
func mapCanvas(g *gallery) *ui.Node {
	d := g.doc
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
	}, ui.ID("minimap"), ui.Attr("width", "200"), ui.Attr("height", "150"),
		ui.Style("border: 1px solid #8a7a5a; border-radius: 6px"))
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
