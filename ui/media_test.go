package ui_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/ui"
)

const schemes = `.card { height: 10px }
@media (prefers-color-scheme: dark) { .card { height: 20px } }
@media (prefers-reduced-motion) { .card { width: 30px } }
@media (pointer: coarse) and (hover: none) { .card { width: 40px } }`

func mediaCard(doc *ui.Document) *ui.Node {
	c := ui.Div(ui.Class("card"))
	doc.Body().Append(c)
	return c
}

func TestADocumentAnswersLightWithNoPreferences(t *testing.T) {
	doc := ui.NewDocument(ui.CSS(schemes))
	c := mediaCard(doc)
	doc.Layout(geom.V2(320, 200))
	assert.Equal(t, platform.AppearanceLight, doc.ColorScheme())
	assert.False(t, doc.ReducedMotion())
	assert.Nil(t, doc.Preferences())
	assert.InDelta(t, 10, c.Box().H, 0.01)
}

func TestADocumentFollowsThePreferencesItIsGiven(t *testing.T) {
	prefs := platform.NewPreferences()
	doc := ui.NewDocument(ui.CSS(schemes), ui.Preferences(prefs))
	c := mediaCard(doc)
	doc.Layout(geom.V2(320, 200))
	assert.Equal(t, platform.AppearanceLight, doc.ColorScheme(), "unknown is light")
	assert.Same(t, prefs, doc.Preferences())

	prefs.SyncAppearance(platform.AppearanceDark)
	prefs.SyncMotion(platform.MotionReduced)
	doc.Layout(geom.V2(320, 200))
	assert.Equal(t, platform.AppearanceDark, doc.ColorScheme())
	assert.True(t, doc.ReducedMotion())
	assert.InDelta(t, 20, c.Box().H, 0.01, "restyled once the system is dark")
	assert.InDelta(t, 30, c.Box().W, 0.01, "and once it reduces motion")

	prefs.SyncAppearance(platform.AppearanceLight)
	prefs.SyncMotion(platform.MotionFull)
	doc.Layout(geom.V2(320, 200))
	assert.InDelta(t, 10, c.Box().H, 0.01)
	assert.False(t, doc.ReducedMotion())

	doc.SetPreferences(nil)
	prefs.SyncAppearance(platform.AppearanceDark)
	doc.Layout(geom.V2(320, 200))
	assert.InDelta(t, 10, c.Box().H, 0.01, "preferences taken away are no longer read")
}

func TestTheProgramsAnswersWinOverThePreferences(t *testing.T) {
	prefs := platform.NewPreferences()
	prefs.SyncAppearance(platform.AppearanceDark)
	prefs.SyncMotion(platform.MotionReduced)
	doc := ui.NewDocument(ui.CSS(schemes), ui.Preferences(prefs), ui.ColorScheme(platform.AppearanceLight))
	c := mediaCard(doc)
	doc.SetReducedMotion(false)
	doc.Layout(geom.V2(320, 200))
	assert.Equal(t, platform.AppearanceLight, doc.ColorScheme())
	assert.False(t, doc.ReducedMotion())
	assert.InDelta(t, 10, c.Box().H, 0.01)

	doc.SetColorScheme(platform.AppearanceUnknown)
	doc.Layout(geom.V2(320, 200))
	assert.Equal(t, platform.AppearanceDark, doc.ColorScheme(), "unknown follows the preferences again")
	assert.InDelta(t, 20, c.Box().H, 0.01)

	alone := ui.NewDocument(ui.CSS(schemes), ui.ColorScheme(platform.AppearanceDark), ui.ReducedMotion())
	other := mediaCard(alone)
	alone.Layout(geom.V2(320, 200))
	assert.InDelta(t, 20, other.Box().H, 0.01, "a scheme of the program's own, no preferences needed")
	assert.InDelta(t, 30, other.Box().W, 0.01)
}

func TestAChangeOfTheSystemRestylesTheNextFrame(t *testing.T) {
	win := platform.NewWindow(platform.Size(320, 200))
	prefs := platform.NewPreferences()
	doc := ui.NewDocument(ui.CSS(schemes), ui.Preferences(prefs))
	c := mediaCard(doc)
	win.Add(doc)
	w := ember.New().AddEntities(win, prefs)
	w.RunFrames(1)
	assert.InDelta(t, 10, c.Box().H, 0.01)
	prefs.SyncAppearance(platform.AppearanceDark)
	w.RunFrames(1)
	assert.InDelta(t, 20, c.Box().H, 0.01)
}

func TestADocumentAnswersContrastAndForcedColoursAsTheSystem(t *testing.T) {
	sheet := `.card { height: 10px; width: 10px }
@media (prefers-contrast: more) { .card { height: 12px } }
@media (prefers-contrast: custom) { .card { height: 16px } }
@media (forced-colors: active) { .card { width: 30px } }`
	prefs := platform.NewPreferences()
	doc := ui.NewDocument(ui.CSS(sheet), ui.Preferences(prefs))
	c := mediaCard(doc)
	doc.Layout(geom.V2(320, 200))
	assert.Equal(t, ui.ContrastNoPreference, doc.Contrast(), "unknown is no preference")
	assert.False(t, doc.ForcedColors())

	prefs.SyncContrast(platform.ContrastMore)
	prefs.SyncForcedColors(platform.ForcedColorsActive)
	doc.Layout(geom.V2(320, 200))
	assert.Equal(t, ui.ContrastMore, doc.Contrast())
	assert.True(t, doc.ForcedColors())
	assert.InDelta(t, 12, c.Box().H, 0.01, "restyled once the system asks for more contrast")
	assert.InDelta(t, 30, c.Box().W, 0.01, "and once it forces its colours")

	prefs.SyncContrast(platform.ContrastCustom)
	prefs.SyncForcedColors(platform.ForcedColorsNone)
	doc.Layout(geom.V2(320, 200))
	assert.InDelta(t, 16, c.Box().H, 0.01)
	assert.InDelta(t, 10, c.Box().W, 0.01)

	prefs.SyncContrast(platform.ContrastMore)
	prefs.SyncForcedColors(platform.ForcedColorsActive)
	doc.SetContrast(ui.ContrastNoPreference)
	doc.SetForcedColors(false)
	doc.Layout(geom.V2(320, 200))
	assert.Equal(t, ui.ContrastNoPreference, doc.Contrast(), "the program's answer wins, no preference too")
	assert.False(t, doc.ForcedColors())
	assert.InDelta(t, 10, c.Box().H, 0.01)
	assert.InDelta(t, 10, c.Box().W, 0.01)

	framed := ui.NewDocument(ui.CSS(sheet))
	host := ui.NewDocument(ui.Preferences(prefs))
	host.Body().Append(ui.Iframe(framed))
	win := platform.NewWindow(platform.Size(320, 200))
	win.Add(host)
	ember.New().AddEntities(win).RunFrames(3)
	assert.Equal(t, ui.ContrastMore, framed.Contrast(), "an iframe answers as its host's system")
	assert.True(t, framed.ForcedColors())
}

func TestAFingerOnTheScreenIsACoarsePointer(t *testing.T) {
	win := platform.NewWindow(platform.Size(320, 200))
	doc := ui.NewDocument(ui.CSS(schemes))
	c := mediaCard(doc)
	win.Add(doc)
	w := ember.New().AddEntities(win)
	w.RunFrames(1)
	assert.InDelta(t, 304, c.Box().W, 0.01, "a mouse until a finger touches")

	in := win.Input()
	in.AddTouch(1, input.Trackpad, geom.V2(0.5, 0.5))
	w.RunFrames(2)
	in.EndTouch(1)
	w.RunFrames(2)
	assert.InDelta(t, 304, c.Box().W, 0.01, "a trackpad moves a fine pointer")

	in.AddTouch(2, input.Touchscreen, geom.V2(100, 100))
	w.RunFrames(2)
	in.EndTouch(2)
	w.RunFrames(2)
	assert.InDelta(t, 40, c.Box().W, 0.01, "a finger on the screen: coarse, no hover")

	in.SetKey(input.KeyA, true)
	w.RunFrames(2)
	assert.InDelta(t, 40, c.Box().W, 0.01, "a key is no pointer")

	in.SetMouse(geom.V2(200, 150))
	w.RunFrames(2)
	assert.InDelta(t, 304, c.Box().W, 0.01, "the mouse moved: fine again")
}

func TestAnIframesDocumentAnswersAsItsHost(t *testing.T) {
	win := platform.NewWindow(platform.Size(320, 200))
	child := ui.NewDocument(ui.CSS(schemes))
	inner := mediaCard(child)
	host := ui.NewDocument(ui.ColorScheme(platform.AppearanceDark), ui.ReducedMotion())
	host.Body().Append(ui.Iframe(child))
	win.Add(host)
	w := ember.New().AddEntities(win)
	w.RunFrames(3)
	assert.Equal(t, platform.AppearanceDark, child.ColorScheme())
	assert.True(t, child.ReducedMotion())
	assert.InDelta(t, 20, inner.Box().H, 0.01)

	child.SetColorScheme(platform.AppearanceLight)
	w.RunFrames(2)
	assert.InDelta(t, 10, inner.Box().H, 0.01, "its own answer wins")
}
