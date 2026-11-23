package ui_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
)

// sheetsDoc is a box the tests' sheets size, under one sheet of the
// Document's own.
func sheetsDoc(opts ...ui.DocumentOption) (*ui.Document, *ui.Node) {
	box := ui.Div(ui.ID("box"), ui.Class("box"))
	d := ui.NewDocument(append(opts, ui.CSS(`.box { width: 100px; height: 10px; }`))...)
	d.Body().Append(box)
	return d, box
}

func width(d *ui.Document, n *ui.Node) float32 {
	d.Layout(geom.V2(800, 600))
	return n.Box().W
}

func TestASheetAddedLaterWinsByOrderAndLosesBySpecificity(t *testing.T) {
	d, box := sheetsDoc()
	require.InDelta(t, 100, width(d, box), 0.01)

	d.AddSheet(`.box { width: 200px; }`)
	assert.InDelta(t, 200, width(d, box), 0.01, "the same specificity, later in the order")

	d.AddSheet(`div { width: 300px; }`)
	assert.InDelta(t, 200, width(d, box), 0.01, "a weaker selector loses wherever it stands")
	assert.Empty(t, d.Warnings())
}

func TestASheetInsertedBeforeAnotherLosesToIt(t *testing.T) {
	d, box := sheetsDoc()
	first := d.StyleSheets()[0]
	early := d.InsertSheet(first, `.box { width: 200px; }`)
	assert.InDelta(t, 100, width(d, box), 0.01)
	assert.Equal(t, []*ui.Sheet{early, first}, d.StyleSheets())

	first.Remove()
	assert.InDelta(t, 200, width(d, box), 0.01, "a sheet the options gave is removed as any other")
}

func TestRemovingASheetRestoresTheLookBeforeIt(t *testing.T) {
	d, box := sheetsDoc()
	s := d.AddSheet(`.box { width: 200px; }`)
	require.InDelta(t, 200, width(d, box), 0.01)
	s.Remove()
	assert.InDelta(t, 100, width(d, box), 0.01)
	s.Replace(`.box { width: 300px; }`)
	assert.InDelta(t, 100, width(d, box), 0.01, "a removed sheet does nothing more")
}

func TestASheetIsReplacedDisabledAndPutUnderAMedia(t *testing.T) {
	d, box := sheetsDoc()
	s := d.AddSheet(`.box { width: 200px; }`)
	s.Replace(`.box { width: 250px; }`)
	assert.InDelta(t, 250, width(d, box), 0.01)
	assert.Equal(t, `.box { width: 250px; }`, s.Text())

	s.SetDisabled(true)
	assert.True(t, s.Disabled())
	assert.InDelta(t, 100, width(d, box), 0.01)
	assert.Len(t, d.Sheets(), 1)
	s.SetDisabled(false)
	assert.InDelta(t, 250, width(d, box), 0.01)

	s.SetMedia("(min-width: 900px)")
	assert.Equal(t, "(min-width: 900px)", s.Media())
	assert.InDelta(t, 100, width(d, box), 0.01, "the viewport is 800 wide")
	d.Layout(geom.V2(1000, 600))
	assert.InDelta(t, 250, box.Box().W, 0.01, "a resize answers the sheet's media as a rule's")

	wide := d.AddSheet(`.box { height: 44px; }`, ui.Media("(max-width: 500px)"))
	d.Layout(geom.V2(400, 600))
	assert.InDelta(t, 44, box.Box().H, 0.01)
	assert.Equal(t, "(max-width: 500px)", wide.Media())
}

func TestRulesAreInsertedAndDeletedOneAtATime(t *testing.T) {
	d, box := sheetsDoc()
	s := d.AddSheet(`/* two rules */ .box { width: 200px; } @media (min-width: 1px) { .box { height: 20px; } }`)
	assert.Equal(t, []string{`.box { width: 200px; }`, `@media (min-width: 1px) { .box { height: 20px; } }`}, s.Rules())

	require.NoError(t, s.InsertRule(2, `.box { width: 210px; }`))
	assert.InDelta(t, 210, width(d, box), 0.01)
	require.NoError(t, s.InsertRule(0, `.box { width: 220px; height: 30px }`))
	assert.InDelta(t, 210, width(d, box), 0.01, "a rule put first loses to those after it")
	assert.InDelta(t, 20, box.Box().H, 0.01)

	require.NoError(t, s.DeleteRule(3))
	assert.InDelta(t, 200, width(d, box), 0.01)
	require.NoError(t, s.DeleteRule(1))
	assert.InDelta(t, 220, width(d, box), 0.01)
	assert.Equal(t, []string{`.box { width: 220px; height: 30px }`, `@media (min-width: 1px) { .box { height: 20px; } }`},
		s.Rules())
	assert.Equal(t, ".box { width: 220px; height: 30px }\n@media (min-width: 1px) { .box { height: 20px; } }", s.Text())
}

func TestANestedRuleIsInsertedAndDeletedWhole(t *testing.T) {
	d, box := sheetsDoc()
	inner := ui.Div(ui.Class("in"))
	box.Append(inner)
	s := d.AddSheet(`.box { width: 200px; .in { height: 2px; } }`)
	require.NoError(t, s.InsertRule(1, `.box { @media (min-width: 1px) { width: 230px; } &.box { height: 40px; } }`))
	require.NoError(t, s.InsertRule(0, `@scope (.box) { :scope { width: 240px; } .in { height: 3px; } }`))
	assert.InDelta(t, 240, width(d, box), 0.01, "a scoped rule wins a tie of weight by proximity, before the order")
	assert.InDelta(t, 40, box.Box().H, 0.01)
	assert.InDelta(t, 2, inner.Box().H, 0.01, "an @scope's root weighs nothing in its selectors")
	matched := d.MatchedRules(box)
	require.Greater(t, len(matched), 3)
	assert.Equal(t, []string{".box.box", ":scope", ".box"},
		[]string{matched[0].Selector, matched[1].Selector, matched[2].Selector}, "the strongest first, nested or not")
	assert.Equal(t, []string{"(.box)"}, matched[1].Scopes, "a tool sees the scope a rule stands in")
	assert.Equal(t, []string{"(min-width: 1px)"}, matched[2].Media)
	assert.Equal(t, []string{`@scope (.box) { :scope { width: 240px; } .in { height: 3px; } }`,
		`.box { width: 200px; .in { height: 2px; } }`,
		`.box { @media (min-width: 1px) { width: 230px; } &.box { height: 40px; } }`}, s.Rules())

	require.NoError(t, s.DeleteRule(0))
	assert.InDelta(t, 230, width(d, box), 0.01, "the nested @media's declarations stand after the rule's own")
	require.NoError(t, s.DeleteRule(1))
	assert.InDelta(t, 200, width(d, box), 0.01)
	assert.InDelta(t, 10, box.Box().H, 0.01)
	assert.Empty(t, d.Warnings())
}

func TestARuleUIReadsPartOfIsTakenAndOneThatIsNoRuleIsAnError(t *testing.T) {
	d, box := sheetsDoc()
	s := d.AddSheet(``)
	require.NoError(t, s.InsertRule(0, `.box { width: 120px; no-such-property: 1; }`))
	assert.InDelta(t, 120, width(d, box), 0.01)
	require.Len(t, d.Warnings(), 1)
	assert.ErrorContains(t, d.Warnings()[0], "no-such-property")

	for _, bad := range []string{``, `.box {`, `.box`, `.a { } .b { }`, `width: 1px`} {
		assert.Error(t, s.InsertRule(0, bad), "%q", bad)
	}
	require.ErrorIs(t, s.InsertRule(5, `.box { }`), ui.ErrNoRule)
	require.ErrorIs(t, s.InsertRule(-1, `.box { }`), ui.ErrNoRule)
	require.ErrorIs(t, s.DeleteRule(1), ui.ErrNoRule)
	assert.Len(t, s.Rules(), 1)
	require.NoError(t, s.InsertRule(1, `@layer one, two;`), "a statement at-rule is one rule")
}

func TestARunTimeSheetIsReadAsTheDocumentsOthers(t *testing.T) {
	d, box := sheetsDoc(ui.BrowserSheets())
	s := d.AddSheet(`@supports (display: grid) { .box { width: -webkit-calc(50px + 20px); } }`,
		ui.From("https://cdn.test/css/site.css"))
	assert.InDelta(t, 70, width(d, box), 0.01)
	assert.Empty(t, d.Warnings())
	assert.True(t, d.Sheets()[1].Browser())
	assert.Equal(t, "https://cdn.test/css/site.css", d.Sheets()[1].From())
	assert.Equal(t, "https://cdn.test/css/site.css", s.From())

	require.NoError(t, s.InsertRule(1, `@layer low { .box { width: 80px; min-height: 33px; } }`))
	assert.InDelta(t, 70, width(d, box), 0.01, "a layer's rule loses to one in none")
	assert.InDelta(t, 33, box.Box().H, 0.01)
	assert.Equal(t, "https://cdn.test/art/a.png", d.Sheets()[1].URL("../art/a.png"))

	plain, _ := sheetsDoc()
	plain.AddSheet(`@supports (display: grid) { .box { width: 1px; } } @font-face { font-family: "X"; src: url(x.ttf) }`)
	plain.Layout(geom.V2(800, 600))
	require.Len(t, plain.Warnings(), 2)
	assert.ErrorContains(t, plain.Warnings()[0], "@supports is not supported")
	assert.ErrorContains(t, plain.Warnings()[1], "@font-face")
}

func TestTheInspectorNamesARunTimeSheetsRule(t *testing.T) {
	d, box := sheetsDoc()
	s := d.AddSheet("\n.box { width: 200px; }", ui.From("https://cdn.test/a.css"))
	require.NoError(t, s.InsertRule(1, `@media (min-width: 1px) { #box { height: 20px; } }`))
	d.Layout(geom.V2(800, 600))
	matches := d.MatchedRules(box)
	require.GreaterOrEqual(t, len(matches), 3)
	assert.Equal(t, "#box", matches[0].Selector)
	assert.Equal(t, []string{"(min-width: 1px)"}, matches[0].Media)
	assert.Equal(t, "https://cdn.test/a.css", matches[0].Sheet.From())
	assert.Equal(t, "style", matches[0].Sheet.Name)
	assert.Equal(t, ".box", matches[1].Selector)
	assert.Equal(t, 2, matches[1].Pos.Line)
	value, ok := d.ComputedValue(box, "width")
	assert.True(t, ok)
	assert.Equal(t, "200px", value)

	require.NoError(t, s.InsertRule(2, `#box { width: 77px; color: #f00; }`))
	value, _ = d.ComputedValue(box, "width")
	assert.Equal(t, "77px", value, "a sheet changed since the last frame is read as it now stands")
	assert.Equal(t, "#box", d.MatchedRules(box)[0].Selector)
	assert.InDelta(t, 77, width(d, box), 0.01)
}

func TestManyChangesBetweenTwoFramesAreOneRestyle(t *testing.T) {
	d, box := sheetsDoc()
	for i := range 400 {
		d.Body().Append(ui.Div(ui.Class(fmt.Sprint("c", i)), ui.Text("row")))
	}
	d.Layout(geom.V2(800, 600))
	s := d.AddSheet(``)
	before := ui.Restyled(d)
	began := time.Now()
	for i := range 1000 {
		require.NoError(t, s.InsertRule(i, fmt.Sprintf(`.c%d { color: #%03x; padding-left: %dpx; }`, i, i, i%7)))
	}
	inserted := time.Since(began)
	require.NoError(t, s.InsertRule(1000, `.box { width: 333px; }`))
	began = time.Now()
	assert.InDelta(t, 333, width(d, box), 0.01)
	frame := time.Since(began)
	assert.LessOrEqual(t, ui.Restyled(d)-before, 410, "every element is styled once, not once a rule")
	t.Logf("1000 rules inserted in %v, the frame after in %v", inserted, frame)
	assert.Less(t, inserted, 500*time.Millisecond)
	assert.Less(t, frame, 500*time.Millisecond)

	before = ui.Restyled(d)
	d.Layout(geom.V2(800, 600))
	assert.Equal(t, before, ui.Restyled(d), "a frame after no change styles nothing")
}
