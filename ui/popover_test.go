package ui_test

import (
	"testing"
	"testing/fstest"

	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/world"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func popoverDoc(t *testing.T, nodes ...*ui.Node) *live {
	t.Helper()
	sheet := `.b { position: absolute; width: 80px; height: 24px }
.p { position-area: bottom; width: 100px; height: 40px }`
	return liveDocWith(t, []ui.DocumentOption{ui.Stylesheet(fstest.MapFS{"p.css": {Data: []byte(sheet)}}, "p.css")},
		nodes...)
}

func TestAPopovertargetButtonTogglesItsPopoverUnderIt(t *testing.T) {
	open := ui.Button(ui.Class("b"), ui.Style("left: 100px; top: 40px"), ui.Attr("popovertarget", "menu"))
	menu := ui.Div(ui.ID("menu"), ui.Class("p"), ui.Attr("popover", ""), ui.Text("menu"))
	l := popoverDoc(t, open, menu)
	require.Zero(t, menu.Box().W, "hidden, it is not displayed")

	l.clickAt(open)
	assert.True(t, l.doc.PopoverOpen(menu))
	assert.True(t, menu.State().Has(dom.PopoverOpen), ":popover-open")
	assert.Same(t, open, l.doc.PopoverSource(menu))
	b := menu.Box()
	assert.InDelta(t, 90, b.X, 0.5, "centred under the button that showed it")
	assert.InDelta(t, 64, b.Y, 0.5)

	l.clickAt(open)
	assert.False(t, l.doc.PopoverOpen(menu), "the button toggles it")
	assert.Zero(t, menu.Box().W)
}

func TestPopovertargetactionShowsOrHides(t *testing.T) {
	show := ui.Button(ui.Class("b"), ui.Style("left: 0"), ui.Attr("popovertarget", "p"),
		ui.Attr("popovertargetaction", "show"))
	hide := ui.Button(ui.Class("b"), ui.Style("left: 200px"), ui.Attr("popovertarget", "p"),
		ui.Attr("popovertargetaction", "hide"))
	p := ui.Div(ui.ID("p"), ui.Attr("popover", "manual"), ui.Text("p"))
	l := popoverDoc(t, show, hide, p)
	l.clickAt(show)
	l.clickAt(show)
	assert.True(t, l.doc.PopoverOpen(p), "show twice leaves it shown")
	l.clickAt(hide)
	assert.False(t, l.doc.PopoverOpen(p))
}

func TestAutoPopoversLightDismissAndEscape(t *testing.T) {
	outside := ui.Button(ui.Class("b"), ui.Style("left: 500px; top: 300px"))
	inner := ui.Button(ui.Text("in"))
	a := ui.Div(ui.ID("a"), ui.Attr("popover", "auto"), inner,
		ui.Button(ui.Attr("popovertarget", "nested"), ui.Text("more")))
	nested := ui.Div(ui.ID("nested"), ui.Attr("popover", ""), ui.Text("nested"))
	manual := ui.Div(ui.ID("m"), ui.Attr("popover", "manual"), ui.Style("inset: auto; left: 0; top: 0"),
		ui.Text("manual"))
	other := ui.Div(ui.ID("o"), ui.Attr("popover", ""), ui.Text("other"))
	l := popoverDoc(t, outside, a, nested, manual, other)

	l.doc.ShowPopover(a, nil)
	l.doc.ShowPopover(manual, nil)
	l.frames(2)
	l.clickAt(inner)
	assert.True(t, l.doc.PopoverOpen(a), "a press inside keeps it")
	l.doc.ShowPopover(nested, inner)
	l.frames(2)
	assert.True(t, l.doc.PopoverOpen(a), "a popover shown from inside another is its child")
	l.key(input.KeyEscape)
	assert.False(t, l.doc.PopoverOpen(nested), "Escape hides the newest")
	assert.True(t, l.doc.PopoverOpen(a))

	l.doc.ShowPopover(other, nil)
	l.frames(2)
	assert.False(t, l.doc.PopoverOpen(a), "another auto popover hides it")
	l.clickAt(outside)
	assert.False(t, l.doc.PopoverOpen(other), "a press outside hides it")
	assert.True(t, l.doc.PopoverOpen(manual), "a manual popover stays")
}

func TestAHintLeavesAutoPopoversButNotOtherHints(t *testing.T) {
	a := ui.Div(ui.Attr("popover", ""), ui.Text("a"))
	h1, h2 := ui.Div(ui.Attr("popover", "hint"), ui.Text("h1")), ui.Div(ui.Attr("popover", "hint"), ui.Text("h2"))
	l := popoverDoc(t, a, h1, h2)
	l.doc.ShowPopover(a, nil)
	l.doc.ShowPopover(h1, nil)
	assert.True(t, l.doc.PopoverOpen(a))
	l.doc.ShowPopover(h2, nil)
	assert.False(t, l.doc.PopoverOpen(h1), "one hint at a time")
	assert.True(t, l.doc.PopoverOpen(a))
	l.doc.ShowPopover(ui.Div(), nil)
	assert.Equal(t, []*ui.Node{a, h2}, l.doc.OpenPopovers(), "a non-popover shows nothing")

	l.frames(2)
	var toggled []*ui.Node
	l.doc.TogglePopover(a, nil)
	l.w.RunFrames(1)
	var r world.Reader[ui.Toggled]
	for e := range r.Read(l.w) {
		toggled = append(toggled, e.Node)
	}
	assert.Equal(t, []*ui.Node{h2, a}, toggled, "hiding an auto popover hides the hints over it, each told")
}
