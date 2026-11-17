package main

import "github.com/lantervale/ember/ui"

func texts() map[string]element {
	return map[string]element{
		"p":      {p, 360},
		"h1":     {h1, 260},
		"h2":     {h2, 240},
		"h3":     {h3, 280},
		"h4":     {h4, 190},
		"span":   {span, 120},
		"strong": {strong, 190},
		"em":     {em, 220},
		"code":   {code, 310},
		"label":  {label, 260},
		"kbd":    {kbd, 260},
		"samp":   {samp, 300},
		"pre":    {pre, 160},

		"h5":    {h5, 160},
		"h6":    {h6, 230},
		"a":     {a, 308},
		"b":     {b, 260},
		"i":     {i, 216},
		"u":     {u, 250},
		"s":     {s, 212},
		"small": {small, 256},
		"big":   {big, 212},
		"sub":   {sub, 242},
		"sup":   {sup, 196},
		"abbr":  {abbr, 140},
		"cite":  {cite, 340},
		"q":     {q, 430},
		"mark":  {mark, 304},
		"time":  {timeElement, 260},
		"var":   {varElement, 266},
		"del":   {del, 192},
		"ins":   {ins, 192},
		"dfn":   {dfn, 340},
		"bdi":   {bdi, 160},
		"bdo":   {bdo, 200},
		"data":  {data, 160},
		"font":  {font, 120},
		"br":    {br, 188},
	}
}

// p is two paragraphs of a quest log, wrapping and stacked without a gap.
func p(*ui.Document) *ui.Node {
	return ui.Div(
		ui.P(ui.Text("The miller's daughter has not come home. Her lantern was found by the old bridge.")),
		ui.P(ui.Text("Search the riverbank before nightfall.")),
	)
}

// h1 is a screen's title over a line of body text.
func h1(*ui.Document) *ui.Node {
	return ui.Div(
		ui.H1(ui.Text("Inventory")),
		ui.P(ui.Text("Twelve slots, four of them free.")),
	)
}

// h2 is a section's title over a line of body text.
func h2(*ui.Document) *ui.Node {
	return ui.Div(
		ui.H2(ui.Text("Settings")),
		ui.P(ui.Text("Sound, controls and display.")),
	)
}

// h3 is a group's title over a line of body text.
func h3(*ui.Document) *ui.Node {
	return ui.Div(
		ui.H3(ui.Text("Controls")),
		ui.P(ui.Text("Keyboard and gamepad bindings.")),
	)
}

// h4 is a small title over a line of body text, the same size.
func h4(*ui.Document) *ui.Node {
	return ui.Div(
		ui.H4(ui.Text("Gamepad")),
		ui.P(ui.Text("Hold South to sprint.")),
	)
}

// span is a score whose number a span colours.
func span(*ui.Document) *ui.Node {
	return ui.P(
		ui.Text("Gold: "),
		ui.Span(ui.Style("color: #b8860b"), ui.Text("1,250")),
	)
}

// strong is a hint whose key is bold.
func strong(*ui.Document) *ui.Node {
	return ui.P(
		ui.Text("Press "),
		ui.Strong(ui.Text("Space")),
		ui.Text(" to jump."),
	)
}

// em is a message whose state is italic.
func em(*ui.Document) *ui.Node {
	return ui.P(
		ui.Text("The cellar door is "),
		ui.Em(ui.Text("locked")),
		ui.Text("."),
	)
}

// code is a chat hint whose command is monospace.
func code(*ui.Document) *ui.Node {
	return ui.P(
		ui.Text("Type "),
		ui.Code(ui.Text("/invite name")),
		ui.Text(" to add a player."),
	)
}

// label is a player's name field, its label naming the input by For.
func label(*ui.Document) *ui.Node {
	return ui.Div(
		ui.Label(ui.For("name"), ui.Text("Name")),
		ui.Text(" "),
		ui.Input(ui.ID("name"), ui.Value("Ayla")),
	)
}

// kbd is a key hint: the keys the player presses.
func kbd(*ui.Document) *ui.Node {
	return ui.P(
		ui.Text("Press "),
		ui.Kbd(ui.Text("Ctrl")),
		ui.Text("+"),
		ui.Kbd(ui.Text("S")),
		ui.Text(" to save."),
	)
}

// samp is what the game answered to a command.
func samp(*ui.Document) *ui.Node {
	return ui.P(
		ui.Text("The console says "),
		ui.Samp(ui.Text("player not found")),
		ui.Text("."),
	)
}

// pre is a save file's lines, kept as typed.
func pre(*ui.Document) *ui.Node {
	return ui.Pre(ui.Text("level:  3\nhero:   Ayla\ngold:   1250"))
}

// h5 is a smaller title over a line of body text.
func h5(*ui.Document) *ui.Node {
	return ui.Div(
		ui.H5(ui.Text("Left stick")),
		ui.P(ui.Text("Moves the hero.")),
	)
}

// h6 is the smallest title over a line of body text.
func h6(*ui.Document) *ui.Node {
	return ui.Div(
		ui.H6(ui.Text("Deadzone")),
		ui.P(ui.Text("A tenth of the stick's travel.")),
	)
}

// a is a link to the patch notes, and an a without an href, plain text.
func a(*ui.Document) *ui.Node {
	return ui.P(
		ui.Text("Read the "),
		ui.A(ui.Attr("href", "notes"), ui.Text("patch notes")),
		ui.Text(" or "),
		ui.A(ui.Text("the old ones")),
		ui.Text("."),
	)
}

// b is a pickup message whose item is bold.
func b(*ui.Document) *ui.Node {
	return ui.P(
		ui.Text("You picked up the "),
		ui.B(ui.Text("Iron sword")),
		ui.Text("."),
	)
}

// i is a line whose ship's name is italic.
func i(*ui.Document) *ui.Node {
	return ui.P(
		ui.Text("The "),
		ui.I(ui.Text("Heron")),
		ui.Text(" sails at dawn."),
	)
}

// u is a chat line whose misspelt word is marked.
func u(*ui.Document) *ui.Node {
	return ui.P(
		ui.Text("Meet me at the "),
		ui.U(ui.Text("brigde")),
		ui.Text(" tonight."),
	)
}

// s is a shop price that no longer holds, before the one that does.
func s(*ui.Document) *ui.Node {
	return ui.P(
		ui.Text("Lantern: "),
		ui.S(ui.Text("25 gold")),
		ui.Text(" 18 gold"),
	)
}

// small is a save slot's fine print after its name.
func small(*ui.Document) *ui.Node {
	return ui.P(
		ui.Text("Slot 1, Aldric "),
		ui.Small(ui.Text("saved two hours ago")),
	)
}

// big is a hit's damage, larger than the words round it.
func big(*ui.Document) *ui.Node {
	return ui.P(
		ui.Text("Critical hit: "),
		ui.Big(ui.Text("128")),
		ui.Text(" damage"),
	)
}

// sub is a potion's formula, its count a subscript.
func sub(*ui.Document) *ui.Node {
	return ui.P(
		ui.Text("Brewed from H"),
		ui.Sub(ui.Text("2")),
		ui.Text("O and moss."),
	)
}

// sup is a leaderboard place, its ordinal a superscript.
func sup(*ui.Document) *ui.Node {
	return ui.P(
		ui.Text("You finished 3"),
		ui.Sup(ui.Text("rd")),
		ui.Text(" of 40."),
	)
}

// abbr is a stat's short name, its title saying it in full.
func abbr(*ui.Document) *ui.Node {
	return ui.P(
		ui.Abbr(ui.Attr("title", "Hit points"), ui.Text("HP")),
		ui.Text(" 80, "),
		ui.Abbr(ui.Attr("title", "Mana points"), ui.Text("MP")),
		ui.Text(" 35"),
	)
}

// cite is the book a hint comes from.
func cite(*ui.Document) *ui.Node {
	return ui.P(
		ui.Text("As told in "),
		ui.Cite(ui.Text("The Lantern Keeper's Almanac")),
		ui.Text("."),
	)
}

// q is a short quotation inside a line.
func q(*ui.Document) *ui.Node {
	return ui.P(
		ui.Text("The smith said "),
		ui.Q(ui.Text("bring me the key")),
		ui.Text(" and turned away."),
	)
}

// mark is the word a search found in a quest's line.
func mark(*ui.Document) *ui.Node {
	return ui.P(
		ui.Text("Light the three "),
		ui.Mark(ui.Text("beacons")),
		ui.Text(" before dawn."),
	)
}

// timeElement is when a save was made, its datetime the machine's form.
func timeElement(*ui.Document) *ui.Node {
	return ui.P(
		ui.Text("Saved on "),
		ui.Time(ui.Attr("datetime", "2026-11-17T21:40"), ui.Text("17 November, 21:40")),
		ui.Text("."),
	)
}

// varElement is a damage formula's variables.
func varElement(*ui.Document) *ui.Node {
	return ui.P(
		ui.Text("Damage is "),
		ui.Var(ui.Text("strength")),
		ui.Text(" times "),
		ui.Var(ui.Text("level")),
		ui.Text("."),
	)
}

// del is a patch note's old value, taken out.
func del(*ui.Document) *ui.Node {
	return ui.P(
		ui.Text("Torch radius: "),
		ui.Del(ui.Text("200")),
		ui.Text(" 300"),
	)
}

// ins is a patch note's new value, put in.
func ins(*ui.Document) *ui.Node {
	return ui.P(
		ui.Text("Torch radius: 200 "),
		ui.Ins(ui.Text("300")),
	)
}

// dfn is the term a tutorial line defines.
func dfn(*ui.Document) *ui.Node {
	return ui.P(
		ui.Text("A "),
		ui.Dfn(ui.Text("beacon")),
		ui.Text(" is a fire that lights a whole floor."),
	)
}

// bdi is a player's name kept apart from the line round it.
func bdi(*ui.Document) *ui.Node {
	return ui.P(
		ui.Bdi(ui.Text("Ayla")),
		ui.Text(": 4210 points"),
	)
}

// bdo is a rune word with its direction attribute, drawn as written.
func bdo(*ui.Document) *ui.Node {
	return ui.P(
		ui.Text("The rune reads "),
		ui.Bdo(ui.Attr("dir", "rtl"), ui.Text("NEPO")),
		ui.Text("."),
	)
}

// data is a shop item's name, its value the item's id.
func data(*ui.Document) *ui.Node {
	return ui.P(
		ui.Text("In stock: "),
		ui.Data(ui.Value("item-042"), ui.Text("Lantern")),
	)
}

// font is the old font element, its attributes styling nothing.
func font(*ui.Document) *ui.Node {
	return ui.P(
		ui.Text("Gold: "),
		ui.Font(ui.Attr("color", "#b8860b"), ui.Attr("size", "5"), ui.Text("1,250")),
	)
}

// br is a sign's two lines in one paragraph.
func br(*ui.Document) *ui.Node {
	return ui.P(
		ui.Text("North: the mill"),
		ui.Br(),
		ui.Text("South: the old bridge"),
	)
}
