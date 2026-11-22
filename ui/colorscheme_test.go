package ui_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/ui"
)

const followsScheme = `:root { color-scheme: light dark }
.card { background-color: light-dark(#ff0000, #0000ff); color: FieldText }`

func schemeCard(doc *ui.Document) *ui.Node {
	c := ui.Div(ui.Class("card"), ui.Text("Heron"))
	doc.Body().Append(c)
	doc.Layout(geom.V2(320, 200))
	return c
}

func inks(t *testing.T, doc *ui.Document, n *ui.Node) [3]string {
	t.Helper()
	root, _ := doc.ComputedValue(doc.HTML(), "color")
	bg, _ := doc.ComputedValue(n, "background-color")
	fg, _ := doc.ComputedValue(n, "color")
	return [3]string{root, bg, fg}
}

func TestADocumentWithoutPreferencesDrawsItsSchemesLight(t *testing.T) {
	doc := ui.NewDocument(ui.CSS(followsScheme))
	c := schemeCard(doc)
	assert.Equal(t, color.Hex("#ff0000"), ui.ShownBackground(c))
	assert.Equal(t, [3]string{"#000000", "#ff0000", "#000000"}, inks(t, doc, c))
}

func TestTheProgramsSchemeRestylesWhatFollowsIt(t *testing.T) {
	doc := ui.NewDocument(ui.CSS(followsScheme))
	c := schemeCard(doc)
	doc.SetColorScheme(platform.AppearanceDark)
	doc.Layout(geom.V2(320, 200))
	assert.Equal(t, color.Hex("#0000ff"), ui.ShownBackground(c))
	assert.Equal(t, [3]string{"#ffffff", "#0000ff", "#ffffff"}, inks(t, doc, c))
	doc.SetColorScheme(platform.AppearanceLight)
	doc.Layout(geom.V2(320, 200))
	assert.Equal(t, color.Hex("#ff0000"), ui.ShownBackground(c))
}

func TestTheSystemsSchemeRestylesADocumentGivenThePreferences(t *testing.T) {
	prefs := platform.NewPreferences()
	doc := ui.NewDocument(ui.CSS(followsScheme), ui.Preferences(prefs))
	c := schemeCard(doc)
	assert.Equal(t, color.Hex("#ff0000"), ui.ShownBackground(c))
	prefs.SyncAppearance(platform.AppearanceDark)
	doc.Layout(geom.V2(320, 200))
	assert.Equal(t, color.Hex("#0000ff"), ui.ShownBackground(c))
	prefs.SyncAppearance(platform.AppearanceLight)
	doc.Layout(geom.V2(320, 200))
	assert.Equal(t, color.Hex("#ff0000"), ui.ShownBackground(c))
}

func TestAPageThatNamesOneSchemeIgnoresThePreference(t *testing.T) {
	doc := ui.NewDocument(ui.CSS(`.card { color-scheme: dark; background-color: Canvas }`),
		ui.ColorScheme(platform.AppearanceLight))
	c := schemeCard(doc)
	assert.Equal(t, color.Hex("#121212"), ui.ShownBackground(c))
	doc.SetColorScheme(platform.AppearanceDark)
	doc.Layout(geom.V2(320, 200))
	assert.Equal(t, color.Hex("#121212"), ui.ShownBackground(c))
	before := ui.Restyled(doc)
	doc.SetColorScheme(platform.AppearanceLight)
	doc.Layout(geom.V2(320, 200))
	assert.Equal(t, before, ui.Restyled(doc), "nothing follows the preference, so nothing is styled again")
}

func TestAColourFollowingTheSchemeTransitions(t *testing.T) {
	win := platform.NewWindow(platform.Size(320, 200))
	doc := ui.NewDocument(ui.CSS(`:root { color-scheme: light dark }
.card { height: 20px; background-color: light-dark(#000000, #ffffff); transition: background-color 1s linear }`))
	card := ui.Div(ui.Class("card"))
	doc.Body().Append(card)
	win.Add(doc)
	w := ember.New().AddEntities(win)
	w.RunFrames(2)
	w.Time().Scale = 0
	assert.Equal(t, color.Hex("#000000"), ui.ShownBackground(card))

	doc.SetColorScheme(platform.AppearanceDark)
	w.RunFrames(30)
	mid := ui.ShownBackground(card)
	assert.Greater(t, mid.R, float32(0.05))
	assert.Less(t, mid.R, float32(0.95))
	w.RunFrames(60)
	assert.InDelta(t, 1, ui.ShownBackground(card).R, 0.001)
}

func TestABrowsersControlsTakeTheirSchemesText(t *testing.T) {
	doc := ui.NewDocument(ui.BrowserSheets(), ui.CSS(`body { color: #ff0000 } .dark { color-scheme: dark }`))
	in, btn := ui.Input(), ui.Button(ui.Text("Go"))
	dark := ui.Input(ui.Class("dark"))
	doc.Body().Append(in, btn, dark)
	doc.Layout(geom.V2(320, 200))
	for n, want := range map[*ui.Node]string{in: "#000000", btn: "#000000", dark: "#ffffff"} {
		got, _ := doc.ComputedValue(n, "color")
		assert.Equal(t, want, got)
	}
	bg, _ := doc.ComputedValue(dark, "background-color")
	assert.Equal(t, "#3b3b3b", bg)

	game := ui.NewDocument(ui.CSS(`body { color: #ff0000 }`))
	field := ui.Input()
	game.Body().Append(field)
	game.Layout(geom.V2(320, 200))
	got, _ := game.ComputedValue(field, "color")
	assert.NotEqual(t, "#000000", got, "a game's field takes its parent's colour")
}
