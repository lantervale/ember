package ui_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/css"
)

const inspectedSheet = `:root { --accent: #336699; }
body { margin: 0; color: #222222; font-size: 20px; }
.card { padding: 1em; color: var(--accent); border: 2px solid red; }
@media (min-width: 100px) {
  #one { color: green; margin: 4px 8px; }
}
@media (min-width: 5000px) {
  #one { color: blue; }
}
div { display: flex; color: black; }
#one { padding-left: 3px !important; }`

func inspected(t *testing.T) (*ui.Document, *ui.Node, *ui.Node) {
	t.Helper()
	inner := ui.Span(ui.Text("text"))
	one := ui.Div(ui.ID("one"), ui.Class("card"), ui.Style("width: 100px; padding-left: 9px"), inner)
	win := platform.NewWindow(platform.Size(640, 360))
	doc := ui.NewDocument(ui.CSS(inspectedSheet))
	doc.Body().Append(one)
	win.Add(doc)
	ember.New().AddEntities(win).RunFrames(2)
	require.Empty(t, doc.Warnings())
	return doc, one, inner
}

func matchNamed(matches []css.Match, selector string) *css.Match {
	for i := range matches {
		if matches[i].Selector == selector {
			return &matches[i]
		}
	}
	return nil
}

func declared(m *css.Match, name string) *css.MatchedDeclaration {
	for i := range m.Declarations {
		if m.Declarations[i].Name == name {
			return &m.Declarations[i]
		}
	}
	return nil
}

func TestMatchedRulesAreTheRulesOfAnElementTheStrongestFirst(t *testing.T) {
	doc, one, _ := inspected(t)
	matches := doc.MatchedRules(one)
	require.NotEmpty(t, matches)

	var selectors []string
	for _, m := range matches {
		if m.Origin != css.Agent {
			selectors = append(selectors, m.Selector)
		}
	}
	assert.Equal(t, []string{"#one", "", "#one", ".card", "div"}, selectors,
		"the important one, the style attribute, then by specificity; a rule whose @media is off is no match")

	inline := matches[1]
	assert.Equal(t, css.Inline, inline.Origin)
	assert.Nil(t, inline.Sheet)
	assert.Equal(t, "100px", declared(&inline, "width").Value)
	assert.True(t, declared(&inline, "padding-left").Overridden, "an !important declaration wins over the style attribute")

	important := matches[0]
	assert.True(t, declared(&important, "padding-left").Important)
	assert.False(t, declared(&important, "padding-left").Overridden)
	assert.Equal(t, 11, important.Pos.Line, "where its rule opens in its sheet")
	assert.Same(t, doc.Sheets()[0], important.Sheet)

	media := matches[2]
	assert.Equal(t, []string{"(min-width: 100px)"}, media.Media)
	assert.False(t, declared(&media, "color").Overridden)
	assert.False(t, declared(&media, "margin").Overridden)

	card := matchNamed(matches, ".card")
	assert.True(t, declared(card, "color").Overridden, "a more specific rule sets the colour")
	assert.False(t, declared(card, "padding").Overridden, "a shorthand stands while one of its longhands does")
	assert.Equal(t, "var(--accent)", declared(card, "color").Value, "as written")

	plain := matchNamed(matches, "div")
	assert.Empty(t, plain.Layer)
	assert.False(t, declared(plain, "display").Overridden)
	assert.True(t, declared(plain, "color").Overridden)

	agent := matches[len(matches)-1]
	assert.Equal(t, css.Agent, agent.Origin, "ember's own sheet is the weakest")
}

func TestMatchedRulesNameTheLayerOfARule(t *testing.T) {
	box := ui.Div(ui.ID("box"))
	l := liveDocWith(t, []ui.DocumentOption{ui.BrowserSheets(),
		ui.CSS("@layer base { #box { color: red; } }\n@layer { #box { color: blue; } }\ndiv { color: green; }")}, box)
	var layers []string
	var lost []bool
	for _, m := range l.doc.MatchedRules(box) {
		if m.Origin == css.Author {
			layers = append(layers, m.Layer)
			lost = append(lost, m.Declarations[0].Overridden)
		}
	}
	assert.Equal(t, []string{"", "", "base"}, layers, "no layer wins over any, a later layer over an earlier one")
	assert.Equal(t, []bool{false, true, true}, lost)
}

func TestMatchedRulesOfWhatHasNoneAreEmpty(t *testing.T) {
	doc, _, inner := inspected(t)
	assert.Empty(t, doc.MatchedRules(inner.Children()[0]), "a text node matches no rule")
	assert.Empty(t, doc.MatchedRules(ui.Div()), "nor an element outside the Document")
	assert.Empty(t, doc.MatchedRules(nil))
	for _, m := range doc.MatchedRules(inner) {
		assert.Equal(t, css.Agent, m.Origin, "a span has only the agent's rules here: %s", m.Selector)
	}
}

func valueOf(props []css.Property, name string) (css.Property, bool) {
	for _, p := range props {
		if p.Name == name {
			return p, true
		}
	}
	return css.Property{}, false
}

func TestComputedStyleIsEveryLonghandAfterTheCascade(t *testing.T) {
	doc, one, inner := inspected(t)
	props := doc.ComputedStyle(one)
	require.NotEmpty(t, props)
	for i := 1; i < len(props); i++ {
		if props[i].Name[:2] != "--" && props[i-1].Name[:2] != "--" {
			assert.Less(t, props[i-1].Name, props[i].Name, "by name")
		}
	}
	want := map[string]string{
		"color":            "green",
		"display":          "flex",
		"width":            "100px",
		"padding-left":     "3px",
		"padding-top":      "20px",
		"margin-top":       "4px",
		"margin-right":     "8px",
		"border-top-width": "2px",
		"border-top-color": "red",
		"font-size":        "20px",
		"position":         "static",
		"--accent":         "#336699",
	}
	for name, value := range want {
		p, ok := valueOf(props, name)
		assert.True(t, ok, name)
		assert.Equal(t, value, p.Value, name)
	}
	size, _ := valueOf(props, "font-size")
	assert.True(t, size.Declared, "inherited from a declaration")
	position, _ := valueOf(props, "position")
	assert.False(t, position.Declared, "the initial value")

	child := doc.ComputedStyle(inner)
	colour, _ := valueOf(child, "color")
	assert.Equal(t, "green", colour.Value, "inherited")
	padding, _ := valueOf(child, "padding-left")
	assert.Equal(t, "0", padding.Value, "padding is not")

	assert.Empty(t, doc.ComputedStyle(ui.Div()), "an element outside the Document has none")
}

func TestComputedValueIsOnePropertyOfComputedStyle(t *testing.T) {
	doc, one, inner := inspected(t)
	for _, n := range []*ui.Node{one, inner, doc.Body()} {
		for _, p := range doc.ComputedStyle(n) {
			got, ok := doc.ComputedValue(n, p.Name)
			assert.True(t, ok, p.Name)
			assert.Equal(t, p.Value, got, p.Name)
		}
	}
	_, ok := doc.ComputedValue(one, "no-such-property")
	assert.False(t, ok)
	_, ok = doc.ComputedValue(ui.Div(), "display")
	assert.False(t, ok)
}

func TestBoxesOfAreAnElementsFourBoxes(t *testing.T) {
	doc, one, _ := inspected(t)
	boxes, ok := doc.BoxesOf(one)
	require.True(t, ok)
	content, _ := doc.BoxOf(one)
	assert.Equal(t, content, boxes.Content, "the content box is BoxOf's")
	assert.Equal(t, geom.Rect{X: 8, Y: 4, W: 100, H: boxes.Border.H}, boxes.Border, "inside its margin")
	assert.Equal(t, geom.Rect{X: 10, Y: 6, W: 96, H: boxes.Border.H - 4}, boxes.Padding, "inside its border")
	assert.Equal(t, geom.Rect{X: 0, Y: 0, W: 116, H: boxes.Border.H + 8}, boxes.Margin)
	assert.InDelta(t, 13, boxes.Content.X, 0.01, "past the important padding")

	_, ok = doc.BoxesOf(ui.Div())
	assert.False(t, ok, "an element outside the Document has no box")
}
