package main

import "github.com/lantervale/ember/ui"

func lists() map[string]element {
	return map[string]element{
		"ul":   {ul, 240},
		"li":   {ul, 240},
		"ol":   {ol, 228},
		"menu": {menu, 160},
		"dl":   {dl, 308},
		"dt":   {dl, 308},
		"dd":   {dl, 308},
	}
}

// ul is a pack's contents, the potions listed a level down, their flasks
// one more.
func ul(*ui.Document) *ui.Node {
	return ui.Ul(
		ui.Li(ui.Text("Iron sword")),
		ui.Li(ui.Text("Potions"),
			ui.Ul(
				ui.Li(ui.Text("Healing"),
					ui.Ul(ui.Li(ui.Text("Two flasks"))),
				),
				ui.Li(ui.Text("Mana")),
			),
		),
		ui.Li(ui.Text("A map of the crypt")),
	)
}

// ol is a quest's steps in order, and its last two numbered from 4.
func ol(*ui.Document) *ui.Node {
	return ui.Div(
		ui.Ol(
			ui.Li(ui.Text("Find the lost key")),
			ui.Li(ui.Text("Open the cellar door")),
			ui.Li(ui.Text("Light the beacon")),
		),
		ui.Ol(ui.Attr("start", "4"),
			ui.Li(ui.Text("Return to the smith")),
			ui.Li(ui.Text("Claim the reward")),
		),
	)
}

// menu is what a held item can do, a button per item.
func menu(*ui.Document) *ui.Node {
	return ui.Menu(
		ui.Li(ui.Button(ui.Text("Use"))),
		ui.Li(ui.Button(ui.Text("Drop"))),
		ui.Li(ui.Button(ui.Text("Inspect"))),
	)
}

// dl is a bestiary: each creature's name over what it is.
func dl(*ui.Document) *ui.Node {
	return ui.Dl(
		ui.Dt(ui.Text("Slime")),
		ui.Dd(ui.Text("Slow, and splits in two when hit.")),
		ui.Dt(ui.Text("Wraith")),
		ui.Dd(ui.Text("Only seen in the dark.")),
	)
}
