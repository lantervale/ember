package ui_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/kit"
)

// hovering is a menu of an item and a label before a paragraph, each
// box 100×20, and the frames that move the pointer over one of them.
type hovering struct {
	*live
	menu, item, label, after, inside *ui.Node
}

const boxes = `div, p, span { display: block; margin: 0; width: 100px; height: 20px; }
.menu { height: 40px; }
`

func hoverDoc(t *testing.T, sheet string) *hovering {
	t.Helper()
	h := &hovering{item: ui.Span(ui.Class("item")), label: ui.Span(ui.Class("label")), inside: ui.Span(ui.Class("item"))}
	h.menu = ui.Div(ui.Class("menu"), h.item, h.label)
	h.after = ui.P(ui.Class("after"), h.inside)
	h.live = liveDocWith(t, []ui.DocumentOption{ui.CSS(boxes + sheet)}, h.menu, h.after)
	return h
}

func (h *hovering) over(n *ui.Node) {
	at := geom.V2(500, 300)
	if n != nil {
		at = n.Box().Center()
	}
	h.win.Input().SetMouse(at)
	h.frames(2)
}

func red(n *ui.Node) bool { return ui.ShownBackground(n) == color.Hex("#ff0000") }

func TestThePointerOverAnElementRestylesWhatItsSheetReaches(t *testing.T) {
	h := hoverDoc(t, `.menu:hover .item, .menu:hover + .after, .item:hover ~ .label { background-color: #f00; }
.menu:not(:hover) .label { background-color: #f00; }`)
	reds := func() [5]bool { return [5]bool{red(h.menu), red(h.item), red(h.label), red(h.after), red(h.inside)} }
	require.Equal(t, [5]bool{false, false, true, false, false}, reds(), "the pointer elsewhere")
	h.over(h.label)
	assert.Equal(t, [5]bool{false, true, false, true, false}, reds(), "over the menu's label")
	h.over(h.item)
	assert.Equal(t, [5]bool{false, true, true, true, false}, reds(), "over its item")
	h.over(h.inside)
	assert.Equal(t, [5]bool{false, false, true, false, false}, reds(), "over what follows the menu")
	h.over(h.item)
	h.over(nil)
	assert.Equal(t, [5]bool{false, false, true, false, false}, reds(), "gone")
	assert.Empty(t, h.doc.Warnings())
}

func TestASheetReadingHoverAboveIsFollowedFromTheFrameItComes(t *testing.T) {
	h := hoverDoc(t, `.menu:hover { outline: 1px solid #000; }`)
	h.over(h.item)
	require.False(t, red(h.item))

	s := h.doc.AddSheet(`.menu:hover .item { background-color: #f00; }`)
	h.frames(1)
	assert.True(t, red(h.item), "a sheet added under the pointer")
	h.over(nil)
	assert.False(t, red(h.item), "its hover is read when the pointer leaves")
	h.over(h.label)
	assert.True(t, red(h.item), "and when it comes back")

	s.SetMedia("(min-width: 5000px)")
	h.frames(1)
	assert.False(t, red(h.item), "the sheet's media no longer holds")
	h.over(nil)
	h.over(h.label)
	assert.False(t, red(h.item))
	s.SetMedia("")
	h.frames(1)
	assert.True(t, red(h.item), "it holds again")

	require.NoError(t, s.DeleteRule(0))
	h.frames(1)
	assert.False(t, red(h.item), "the rule deleted")
	require.NoError(t, s.InsertRule(0, `.menu:hover + p > .item { background-color: #f00; }`))
	h.frames(1)
	assert.True(t, red(h.inside), "another put in its place")
	h.over(nil)
	assert.False(t, red(h.inside))
	h.over(h.menu)
	assert.True(t, red(h.inside))

	s.Remove()
	h.frames(1)
	assert.False(t, red(h.inside), "the sheet removed")
	before := ui.Restyled(h.doc)
	h.over(nil)
	assert.Equal(t, 4, ui.Restyled(h.doc)-before, "nothing reads hover above now: the four that were under the pointer")
}

func TestAMediaRuleTurnsAHoverReadAboveOnAndOff(t *testing.T) {
	item := ui.Span(ui.Class("item"))
	menu := ui.Div(ui.Class("menu"), item)
	d := ui.NewDocument(ui.CSS(boxes + `@media (min-width: 500px) { .menu:hover .item { background-color: #f00; } }`))
	d.Body().Append(menu)
	hover := func(on bool) {
		for n := menu; n != nil; n = n.Parent() {
			n.SetState(dom.Hover, on)
		}
	}
	d.Layout(geom.V2(400, 300))
	hover(true)
	d.Layout(geom.V2(400, 300))
	assert.False(t, red(item), "a narrow viewport")
	d.Layout(geom.V2(600, 300))
	assert.True(t, red(item), "widened under the pointer")
	hover(false)
	d.Layout(geom.V2(600, 300))
	assert.False(t, red(item))
	hover(true)
	d.Layout(geom.V2(600, 300))
	assert.True(t, red(item))
	d.Layout(geom.V2(400, 300))
	assert.False(t, red(item), "narrow again")
}

func TestThePointerOverABigContainerRestylesOnlyWhatIsUnderIt(t *testing.T) {
	list := ui.Div(ui.Class("list"))
	for range 1500 {
		list.Append(ui.Div(ui.Class("row"), ui.Span(ui.Class("item"), ui.Text("row")), ui.Span(ui.Class("label"))))
	}
	side := ui.Div(ui.Class("side"), ui.Span(ui.Class("item")))
	l := liveDocWith(t, []ui.DocumentOption{ui.CSS(`.side, .row { height: 20px; }
a:hover, .row:hover { background-color: #eee; }
.side:hover .item, .tab:hover + .tab, li:hover > ul { background-color: #f00; }`)}, side, list)
	move := func(at geom.Vec2) int {
		before := ui.Restyled(l.doc)
		l.win.Input().SetMouse(at)
		l.frames(2)
		return ui.Restyled(l.doc) - before
	}
	row := list.Children()[3]
	assert.Equal(t, 4, move(row.Box().Center()), "the row, the list, body and html, of 4,500 elements")
	assert.Equal(t, 2, move(list.Children()[5].Box().Center()), "the row left and the row entered")
	assert.Equal(t, 4, move(side.Box().Center()), "the row and the list left, the side and its item entered or reached")
	assert.True(t, red(side.Children()[0]))
	assert.Equal(t, 4, move(geom.V2(-10, -10)), "the side, body and html left, and the side's item")
}

func TestThePointerOverAKitScreenRestylesOnlyWhatIsUnderIt(t *testing.T) {
	list := kit.Column()
	for range 300 {
		list.Append(kit.Row(kit.Label("name"), kit.Button("set"), kit.Check("on")))
	}
	l := liveDocWith(t, []ui.DocumentOption{kit.Theme()}, list)
	move := func(at geom.Vec2) int {
		before := ui.Restyled(l.doc)
		l.win.Input().SetMouse(at)
		l.frames(2)
		return ui.Restyled(l.doc) - before
	}
	rows := list.Children()
	move(rows[0].Children()[0].Box().Center())
	const chain = 8
	assert.LessOrEqual(t, move(geom.V2(-10, -10)), chain, "the pointer gone: what was under it, of 1,800 elements")
	assert.LessOrEqual(t, move(rows[1].Children()[2].Box().Center()), chain, "over a check: what is under it, and its box")
	assert.LessOrEqual(t, move(rows[2].Children()[0].Box().Center()), 2*chain, "over another row: those left and entered")
}
