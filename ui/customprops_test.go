package ui_test

import (
	"testing"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/ui"
	"github.com/stretchr/testify/assert"
)

func TestAClassOnTheRootRedefiningCustomPropertiesRepaintsWhatReadsThem(t *testing.T) {
	win := platform.NewWindow(platform.Size(320, 200))
	doc := ui.NewDocument(ui.ScrollingViewport(), ui.CSS(`
:root { --bg: #f7f4ee; --card: #ffffff }
:root.dark { --bg: #0b0c0f; --card: #15171b }
body { background: var(--bg) }
.card { background: var(--card); transition: box-shadow .6s ease }
:root.dark .card { box-shadow: 0 4px 8px #000 }
`))
	card := ui.Div(ui.Class("card"), ui.Style("height: 20px"))
	inner := ui.Div(ui.Style("height: 10px; background: var(--bg)"))
	card.Append(inner)
	doc.Body().Append(ui.Div(card))
	win.Add(doc)
	w := ember.New().AddEntities(win)
	w.RunFrames(2)

	for _, dark := range []bool{true, false, true} {
		bg, kept := color.Hex("#f7f4ee"), color.Hex("#ffffff")
		class := ""
		if dark {
			bg, kept, class = color.Hex("#0b0c0f"), color.Hex("#15171b"), "dark"
		}
		doc.HTML().SetAttr("class", class)
		w.RunFrames(2)
		assert.Equal(t, bg, ui.ShownBackground(doc.Body()), "body, dark %v", dark)
		assert.Equal(t, kept, ui.ShownBackground(card), "the card, dark %v", dark)
		assert.Equal(t, bg, ui.ShownBackground(inner), "inside the card, dark %v", dark)
	}
}

func TestComputedStyleReadsAClassTheRootTookSinceTheLastFrame(t *testing.T) {
	win := platform.NewWindow(platform.Size(320, 200))
	doc := ui.NewDocument(ui.CSS(`
:root { --bg: #f7f4ee } :root.dark { --bg: #0b0c0f }
body { background: var(--bg) } p { color: var(--bg) }
`))
	p := ui.P(ui.Text("hi"))
	doc.Body().Append(ui.Div(p))
	win.Add(doc)
	w := ember.New().AddEntities(win)
	w.RunFrames(2)

	doc.HTML().SetAttr("class", "dark")
	bg, _ := doc.ComputedValue(doc.Body(), "background-color")
	ink, _ := doc.ComputedValue(p, "color")
	v, _ := doc.ComputedValue(p, "--bg")
	assert.Equal(t, []string{"#0b0c0f", "#0b0c0f", "#0b0c0f"}, []string{bg, ink, v},
		"body's background, p's colour and the custom property under the root")
	var all string
	for _, prop := range doc.ComputedStyle(doc.Body()) {
		if prop.Name == "background-color" {
			all = prop.Value
		}
	}
	assert.Equal(t, "#0b0c0f", all, "body's background in its whole computed style")
}
