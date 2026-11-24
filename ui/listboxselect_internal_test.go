package ui

import (
	"testing"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/automation"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/world"
	"github.com/stretchr/testify/assert"
)

type listed struct {
	doc     *Document
	sel     *Node
	win     *platform.Window
	w       *world.World
	auto    *automation.Automation
	changed int
}

func newListed(t *testing.T) *listed {
	t.Helper()
	l := &listed{doc: NewDocument(CSS(`body { margin: 0; }`))}
	l.sel = Select([]Arg{
		ID("gear"), Attr("multiple", ""),
		Option(Text("Sword"), Value("sword")),
		Optgroup("Magic", Option(Text("Wand")), Option(Text("Staff"), Selected())),
		Option(Text("Bow"), Disabled()),
		Option(Text("Shield")),
		OnChange(func() { l.changed++ }),
	}...)
	l.doc.Body().Append(l.sel)
	l.win = platform.NewWindow(platform.Size(300, 300))
	l.win.Add(l.doc)
	l.auto = automation.New(automation.Recording())
	l.w = ember.New(ember.Automation(l.auto)).AddEntities(l.win)
	l.win.Input().SetMouse(geom.V2(290, 290))
	l.w.RunFrames(3)
	return l
}

func TestAListBoxsOptionsAreRowsOfThePage(t *testing.T) {
	l := newListed(t)
	r := l.sel.Box()
	opts := dom.Options(l.sel)
	first := opts[0].Box()
	assert.InDelta(t, r.Y+1, first.Y, 0.01, "the first row under the border, never centred")
	wand := opts[1].Box()
	label := wand.Y - first.Y - first.H
	assert.Greater(t, label, first.H, "the group's label a row of its own, content-box under the game's border-box")
	assert.InDelta(t, 4*label+2, r.H, 0.01, "four rows as tall as the tallest, and the border")
	assert.True(t, opts[2].State().Has(dom.Checked), "a selected option matches :checked")
}

func TestAListBoxReadsAndSetsEverySelectedOption(t *testing.T) {
	l := newListed(t)
	assert.Equal(t, []int{2}, l.doc.SelectedOptions(l.sel))
	l.doc.SetSelectedOptions(l.sel, 0, 3, 4)
	assert.Equal(t, []int{0, 3, 4}, l.doc.SelectedOptions(l.sel))
	assert.Equal(t, 0, l.doc.Selected(l.sel), "selectedIndex is the first")
	assert.Equal(t, "sword", l.doc.Value(l.sel))
	assert.Equal(t, []string{"sword", "Shield"}, l.doc.Values(l.sel), "a disabled option is never submitted")
	l.doc.SetValue(l.sel, "Wand")
	assert.Equal(t, []int{1}, l.doc.SelectedOptions(l.sel), "a value selects its option alone")
	l.doc.SetSelected(l.sel, -1)
	assert.Empty(t, l.doc.SelectedOptions(l.sel))
	assert.Equal(t, 0, l.changed, "the page's changes fire nothing")
}

func TestAListBoxShownAsADropdownAgainKeepsItsFirstOption(t *testing.T) {
	l := newListed(t)
	l.doc.SetSelectedOptions(l.sel, 1, 2)
	l.sel.RemoveAttr("multiple")
	l.w.RunFrames(2)
	assert.Equal(t, []int{1}, l.doc.SelectedOptions(l.sel))
	assert.NotNil(t, dom.Data(l.sel).Shadow, "a dropdown shows its choice")
	assert.False(t, dom.ListBox(l.sel))
}

func TestAutomationSelectsAListBoxsOptionsAtOnce(t *testing.T) {
	l := newListed(t)
	l.auto.Queue(automation.Slug("gear"), automation.ActSelect, []any{"sword", "Shield"})
	l.w.RunFrames(2)
	assert.Equal(t, []int{0, 4}, l.doc.SelectedOptions(l.sel))
	assert.Equal(t, 1, l.changed, "one change")
	l.auto.Queue(automation.Slug("gear"), automation.ActSelect, "Wand")
	l.w.RunFrames(2)
	assert.Equal(t, []int{1}, l.doc.SelectedOptions(l.sel), "one name selects one alone")
	_, all := l.auto.Snapshot("", "")
	for _, e := range all {
		if e.ID == "gear" {
			assert.Equal(t, []any{1}, e.Value, "its value lists what it selects")
		}
	}
}

func TestEnterGoesPastAListBox(t *testing.T) {
	l := newListed(t)
	l.doc.Focus(l.sel)
	l.w.RunFrames(2)
	in := l.win.Input()
	in.SetKey(input.KeyEnter, true)
	l.w.RunFrames(1)
	in.SetKey(input.KeyEnter, false)
	l.w.RunFrames(1)
	assert.Equal(t, []int{2}, l.doc.SelectedOptions(l.sel), "Enter toggles nothing")
	in.SetKey(input.KeyDown, true)
	l.w.RunFrames(1)
	in.SetKey(input.KeyDown, false)
	l.w.RunFrames(1)
	assert.Equal(t, []int{4}, l.doc.SelectedOptions(l.sel), "Down from Staff to Shield, past the disabled Bow")
	assert.Equal(t, 1, l.changed)
}
