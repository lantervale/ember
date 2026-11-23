package ui

import (
	"testing"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/ui/control"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/world"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type grouped struct {
	doc     *Document
	sel     *Node
	win     *platform.Window
	w       *world.World
	changed int
}

func newGrouped(t *testing.T, opts ...Arg) *grouped {
	t.Helper()
	g := &grouped{doc: NewDocument(CSS(`body { margin: 0; } select { position: absolute; left: 10px; top: 10px; }`))}
	g.sel = Select(append([]Arg{
		Option(Text("Sword")),
		Optgroup("Magic", Option(Text("Wand")), Option(Text("Staff"), Value("staff"))),
		Optgroup("Cursed", Disabled(), Option(Text("Skull"))),
		Option(Text("Bow")),
		OnChange(func() { g.changed++ }),
	}, opts...)...)
	g.doc.Body().Append(g.sel)
	g.win = platform.NewWindow(platform.Size(300, 300))
	g.win.Add(g.doc)
	g.w = ember.New().AddEntities(g.win)
	g.win.Input().SetMouse(geom.V2(290, 290))
	g.w.RunFrames(3)
	return g
}

func (g *grouped) click(at geom.Vec2) {
	in := g.win.Input()
	in.SetMouse(at)
	g.w.RunFrames(2)
	in.SetMouseButton(input.MouseLeft, true)
	g.w.RunFrames(1)
	in.SetMouseButton(input.MouseLeft, false)
	g.w.RunFrames(2)
}

func (g *grouped) open(t *testing.T) *control.Select {
	t.Helper()
	r, ok := g.doc.BoxOf(g.sel)
	require.True(t, ok)
	g.click(geom.V2(r.X+r.W/2, r.Y+r.H/2))
	m, _ := g.doc.machine(g.sel).(*control.Select)
	require.True(t, m.Open)
	return m
}

// rows are the middles of the open list's n rows, inside its 1-point
// border, and the option each picks, -1 for a group's label.
func (g *grouped) rows(n int) ([]int, []geom.Vec2) {
	r, _ := g.doc.BoxOf(g.sel)
	x := r.X + 20
	var top, bottom float32
	for y := r.Y + r.H; y < 300; y += 0.25 {
		if g.doc.pickAt(geom.V2(x, y)).List {
			if top == 0 {
				top = y
			}
			bottom = y + 0.25
		}
	}
	row := (bottom - top - 2) / float32(n)
	var picks []int
	var at []geom.Vec2
	for i := range n {
		p := geom.V2(x, top+1+(float32(i)+0.5)*row)
		at = append(at, p)
		picks = append(picks, g.doc.pickAt(p).Index)
	}
	return picks, at
}

func TestASelectsOptionsRunAcrossItsGroups(t *testing.T) {
	g := newGrouped(t, Value("x"))
	m, _ := g.doc.machine(g.sel).(*control.Select)
	assert.Len(t, m.Options, 5)
	assert.Equal(t, 0, g.doc.Selected(g.sel))
	g.doc.SetValue(g.sel, "staff")
	assert.Equal(t, 2, g.doc.Selected(g.sel), "as selectedIndex, counted across groups")
	assert.Equal(t, "staff", g.doc.Value(g.sel))
	assert.True(t, m.Options[3].Disabled, "a disabled group disables its options")
	assert.False(t, m.Options[4].Disabled)
}

func TestAnOpenListShowsGroupLabelsThatAreNeverPicked(t *testing.T) {
	g := newGrouped(t)
	m := g.open(t)
	picks, at := g.rows(7)
	assert.Equal(t, []int{0, -1, 1, 2, -1, 3, 4}, picks, "each label a row of its own")
	g.click(at[1])
	assert.True(t, m.Open, "a click on a label chooses nothing and keeps the list open")
	assert.Zero(t, g.changed)
	g.click(at[2])
	assert.False(t, m.Open)
	assert.Equal(t, 1, g.doc.Selected(g.sel), "Wand")
	assert.Equal(t, 1, g.changed)
}

func TestTheArrowsSkipGroupLabelsAndADisabledGroupsOptions(t *testing.T) {
	g := newGrouped(t)
	g.doc.Focus(g.sel)
	tap := func(k input.Key) {
		in := g.win.Input()
		in.SetKey(k, true)
		g.w.RunFrames(1)
		in.SetKey(k, false)
		g.w.RunFrames(1)
	}
	tap(input.KeyEnter)
	for range 3 {
		tap(input.KeyDown)
	}
	tap(input.KeyEnter)
	assert.Equal(t, 4, g.doc.Selected(g.sel), "Sword, Wand, Staff, then Bow past Cursed's Skull")
}

func TestAnOptgroupAndADisabledOnesOptionsMatchDisabled(t *testing.T) {
	g := newGrouped(t)
	groups := []*Node{g.sel.Children()[1], g.sel.Children()[2]}
	assert.False(t, groups[0].State().Has(dom.Disabled))
	assert.True(t, groups[1].State().Has(dom.Disabled))
	assert.True(t, groups[1].Children()[0].State().Has(dom.Disabled), "Skull")
	assert.False(t, groups[0].Children()[0].State().Has(dom.Disabled))
}
