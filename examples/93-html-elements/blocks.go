package main

import "github.com/lantervale/ember/ui"

func blocks() map[string]element {
	return map[string]element{
		"div":     {div, 240},
		"section": {section, 280},
		"header":  {header, 280},
		"footer":  {footer, 280},
		"nav":     {nav, 340},
		"main":    {mainElement, 280},
		"aside":   {aside, 322},
		"article": {article, 300},

		"blockquote": {blockquote, 340},
		"figure":     {figure, 240},
		"figcaption": {figure, 240},
		"form":       {formElement, 316},
		"address":    {address, 296},
		"hgroup":     {hgroup, 260},
		"search":     {search, 290},
		"center":     {center, 280},
		"fieldset":   {fieldset, 240},
		"legend":     {fieldset, 240},
	}
}

// div is a hero's stats: a box holding a box per line, stacked.
func div(*ui.Document) *ui.Node {
	return ui.Div(ui.Style("border: 1px dashed #999; padding: 8px"),
		ui.Div(ui.Text("Health 80 / 100")),
		ui.Div(ui.Text("Mana 35 / 50")),
		ui.Div(ui.Text("Gold 120")),
	)
}

// section is a settings screen in two themed parts, each with its heading.
func section(*ui.Document) *ui.Node {
	return ui.Div(
		ui.Section(ui.ID("audio"), ui.Style("border: 1px dashed #999; padding: 8px; margin-bottom: 8px"),
			ui.H3(ui.Text("Audio")),
			ui.P(ui.Text("Music 80%, effects 60%")),
		),
		ui.Section(ui.ID("video"), ui.Style("border: 1px dashed #999; padding: 8px"),
			ui.H3(ui.Text("Video")),
			ui.P(ui.Text("1920 × 1080, fullscreen")),
		),
	)
}

// header is the top of an inventory screen: its title and what it holds.
func header(*ui.Document) *ui.Node {
	return ui.Div(
		ui.Header(ui.Style("border-bottom: 1px dashed #999; padding-bottom: 8px; margin-bottom: 8px"),
			ui.H2(ui.Text("Inventory")),
			ui.P(ui.Text("12 of 20 slots used")),
		),
		ui.P(ui.Text("Iron sword, 3 potions, a map")),
	)
}

// footer is the bottom of an inventory screen: the keys it takes.
func footer(*ui.Document) *ui.Node {
	return ui.Div(
		ui.H2(ui.Text("Inventory")),
		ui.P(ui.Text("Iron sword, 3 potions, a map")),
		ui.Footer(ui.Style("border-top: 1px dashed #999; padding-top: 8px; margin-top: 8px"),
			ui.Text("Enter to use, Esc to close"),
		),
	)
}

// nav is a pause menu's way to the game's other screens.
func nav(*ui.Document) *ui.Node {
	return ui.Nav(ui.ID("menu"), ui.Style("border: 1px dashed #999; padding: 8px"),
		ui.Button(ui.Text("Resume")),
		ui.Text(" "),
		ui.Button(ui.Text("Map")),
		ui.Text(" "),
		ui.Button(ui.Text("Settings")),
		ui.Text(" "),
		ui.Button(ui.Text("Quit")),
	)
}

// mainElement is a quest log's own content, between its title and its keys.
func mainElement(*ui.Document) *ui.Node {
	return ui.Div(
		ui.H2(ui.Text("Quest log")),
		ui.Main(ui.Style("border: 1px dashed #999; padding: 8px; margin: 8px 0"),
			ui.P(ui.Text("Find the lost key")),
			ui.P(ui.Text("Light the three beacons")),
			ui.P(ui.Text("Talk to the smith")),
		),
		ui.Footer(ui.Text("Esc to close")),
	)
}

// aside is a tip beside a shop's goods, apart from them.
func aside(*ui.Document) *ui.Node {
	return ui.Div(ui.Style("display: flex; gap: 12px"),
		ui.Div(
			ui.H3(ui.Text("Shop")),
			ui.P(ui.Text("Potion, 10 gold")),
			ui.P(ui.Text("Rope, 4 gold")),
			ui.P(ui.Text("Lantern, 25 gold")),
		),
		ui.Aside(ui.Style("width: 160px; border: 1px dashed #999; padding: 8px; background-color: #f2f2f2"),
			ui.P(ui.Text("Tip: lanterns light the crypt for ten minutes.")),
		),
	)
}

// article is a quest log's entries, each one whole on its own.
func article(*ui.Document) *ui.Node {
	return ui.Div(
		ui.Article(ui.ID("key"), ui.Style("border: 1px dashed #999; padding: 8px; margin-bottom: 8px"),
			ui.H3(ui.Text("The lost key")),
			ui.P(ui.Text("The smith dropped it in the well.")),
		),
		ui.Article(ui.ID("beacons"), ui.Style("border: 1px dashed #999; padding: 8px"),
			ui.H3(ui.Text("The three beacons")),
			ui.P(ui.Text("Two are lit; one is left.")),
		),
	)
}

// blockquote is what the innkeeper said, set in from the line before it.
func blockquote(*ui.Document) *ui.Node {
	return ui.Div(
		ui.P(ui.Text("The innkeeper lowers her voice:")),
		ui.Blockquote(ui.Text("Nobody who went down to the crypt after dark came back the same.")),
	)
}

// figure is the panel picture with its caption under it.
func figure(*ui.Document) *ui.Node {
	return ui.Figure(
		ui.Img(ui.Attr("src", "panel.png"), ui.Attr("alt", "A panel"), ui.Style("width: 72px; image-rendering: pixelated")),
		ui.Figcaption(ui.Text("The crypt's seal")),
	)
}

// formElement is a new hero's name and the button that makes it.
func formElement(*ui.Document) *ui.Node {
	return ui.Form(ui.ID("new-hero"),
		ui.Label(ui.For("hero"), ui.Text("Hero")),
		ui.Text(" "),
		ui.Input(ui.ID("hero"), ui.Value("Aldric")),
		ui.Text(" "),
		ui.Button(ui.Text("Create")),
	)
}

// address is who made the game, one line each.
func address(*ui.Document) *ui.Node {
	return ui.Address(
		ui.Text("Lantervale studio"),
		ui.Br(),
		ui.Text("Made by two leads and their agents"),
	)
}

// hgroup is a screen's title with the line that goes with it.
func hgroup(*ui.Document) *ui.Node {
	return ui.Hgroup(ui.Style("border: 1px dashed #999; padding: 0 8px"),
		ui.H2(ui.Text("The Crypt")),
		ui.P(ui.Text("Floor 3, the flooded halls")),
	)
}

// search is the field and button that look through the bestiary.
func search(*ui.Document) *ui.Node {
	return ui.Search(ui.Style("border: 1px dashed #999; padding: 8px"),
		ui.Input(ui.Type("search"), ui.Placeholder("Search the bestiary")),
		ui.Text(" "),
		ui.Button(ui.Text("Find")),
	)
}

// center is a game over screen's two lines, centred.
func center(*ui.Document) *ui.Node {
	return ui.Center(ui.Style("border: 1px dashed #999; padding: 8px"),
		ui.Div(ui.Text("Game over")),
		ui.Div(ui.Text("Press Enter to try again")),
	)
}

// fieldset is the sound settings in a frame, its legend on the frame's top line.
func fieldset(*ui.Document) *ui.Node {
	return ui.Fieldset(
		ui.Legend(ui.Text("Sound")),
		ui.Div(ui.Label(ui.Input(ui.Type("checkbox"), ui.Checked(true)), ui.Text(" Music"))),
		ui.Div(ui.Label(ui.Input(ui.Type("checkbox")), ui.Text(" Effects"))),
	)
}
