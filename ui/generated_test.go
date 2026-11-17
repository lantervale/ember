package ui_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/automation"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/align"
)

const generatedSheet = `
body { margin: 0; font-size: 10px; line-height: 20px; }
div { width: 200px; position: relative; }
#block::before { content: ""; display: block; height: 10px; }
#inline::before { content: "ab"; }
#atom::after { content: ""; display: inline-block; width: 30px; height: 12px; }
#abs::before { content: ""; position: absolute; left: -60%; top: 5px; width: 60%; height: 40px; }
#float::before { content: ""; float: right; width: 40px; height: 20px; }
#flex { display: flex; }
#flex::before { content: ""; width: 20px; }
#flex::after { content: ""; flex: 1; }
#grid { display: grid; grid-template-columns: 50px 60px 70px; }
#grid::before, #grid::after { content: ""; height: 5px; }
#none::before { content: none; display: block; height: 10px; }
#normal::before { display: block; height: 10px; }
#gone::before { content: "x"; display: none; }
`

func generatedDoc() (*ui.Document, map[string]*ui.Node) {
	d := ui.NewDocument(ui.CSS(generatedSheet))
	by := map[string]*ui.Node{}
	for _, id := range []string{"block", "inline", "atom", "abs", "float", "flex", "grid", "none", "normal", "gone"} {
		by[id] = ui.Div(ui.ID(id), ui.Span(ui.Text("text")))
		d.Body().Append(by[id])
	}
	d.Layout(geom.V2(400, 600))
	return d, by
}

func TestBeforeAndAfterMakeABoxOfEveryDisplay(t *testing.T) {
	d, by := generatedDoc()
	in := func(id string, after bool) geom.Rect {
		t.Helper()
		r, _, ok := ui.Generated(d, by[id], after)
		require.True(t, ok, id)
		e := by[id].Box()
		return geom.Rect{X: r.X - e.X, Y: r.Y - e.Y, W: r.W, H: r.H}
	}
	assert.Equal(t, geom.Rect{W: 200, H: 10}, in("block", false), "a block above the element's content")
	assert.Equal(t, float32(30), by["block"].Box().H, "which makes the element taller")

	inline := in("inline", false)
	text := by["inline"].Children()[0].Box()
	assert.Equal(t, float32(0), inline.X, "inline content starts the first line")
	assert.Greater(t, inline.W, float32(0))
	assert.InDelta(t, inline.W, text.X, 0.01, "and the element's own content follows it")

	atom := in("atom", true)
	assert.Equal(t, geom.V2(30, 12), geom.V2(atom.W, atom.H), "an inline-block of its own size")
	assert.InDelta(t, by["atom"].Children()[0].Box().W, atom.X, 0.01, "after the content")

	assert.Equal(t, geom.Rect{X: -120, Y: 5, W: 120, H: 40}, in("abs", false), "placed by its element, its containing block")
	assert.Equal(t, float32(20), by["abs"].Box().H, "out of the flow")
	assert.Equal(t, geom.Rect{X: 160, W: 40, H: 20}, in("float", false), "a float")

	before, after := in("flex", false), in("flex", true)
	span := by["flex"].Children()[0].Box()
	assert.Equal(t, geom.Rect{W: 20, H: 20}, before, "the first flex item")
	assert.InDelta(t, 20, span.X, 0.01)
	assert.InDelta(t, 200, after.X+after.W, 0.01, "the last item grows to the end")
	assert.InDelta(t, span.X+span.W, after.X, 0.01)

	assert.Equal(t, geom.Rect{W: 50, H: 5}, in("grid", false), "the first grid item")
	assert.Equal(t, geom.Rect{X: 110, W: 70, H: 5}, in("grid", true), "the last")

	for _, id := range []string{"none", "normal", "gone"} {
		_, _, ok := ui.Generated(d, by[id], false)
		assert.False(t, ok, "#%s has no box", id)
		assert.Equal(t, float32(20), by[id].Box().H)
	}
	assert.Empty(t, d.Warnings())
	assert.Empty(t, align.Check(d))
}

func TestGeneratedContentIsNoTextOfItsElement(t *testing.T) {
	b := ui.Button(ui.Text("Save"))
	l := liveDocWith(t, []ui.DocumentOption{ui.CSS(
		`button::before { content: "> "; } button::after { content: " <"; }`)}, b)
	r, text, ok := ui.Generated(l.doc, b, false)
	require.True(t, ok)
	assert.Equal(t, "> ", text)
	assert.Greater(t, r.W, float32(0))
	assert.Equal(t, "Save", b.Text(), "the element's text is its own")
	assert.Equal(t, []string{"Save"}, l.roles(automation.RoleButton), "and so is its name for automation")
	assert.Len(t, b.Children(), 1)
}

func TestAClickOnAGeneratedBoxIsItsElements(t *testing.T) {
	var got string
	tab := ui.Attr("tabindex", "0")
	inner := ui.Div(ui.ID("inner"), tab, ui.OnClick(func() { got += "inner" }), ui.Text("x"))
	outer := ui.Div(ui.ID("outer"), tab, ui.OnClick(func() { got += "outer" }), inner)
	l := liveDocWith(t, []ui.DocumentOption{ui.CSS(`
body { margin: 0; }
#outer { width: 300px; height: 100px; }
#inner { width: 100px; height: 20px; position: relative; }
#inner::before { content: "tag"; position: absolute; left: 150px; top: 40px; width: 50px; height: 30px; }
`)}, outer)
	r, _, ok := ui.Generated(l.doc, inner, false)
	require.True(t, ok)
	assert.Equal(t, geom.Rect{X: 150, Y: 40, W: 50, H: 30}, r)
	l.pressAt(geom.V2(170, 50))
	assert.Equal(t, "inner", got, "the press on ::before, outside #inner's own box, is #inner's")
	assert.Same(t, inner, ui.HoveredElement(l.doc), "and #inner is what the pointer is over")
	got = ""
	l.pressAt(geom.V2(250, 50))
	assert.Equal(t, "outer", got)
}

func TestAHoverMakesAndUnmakesAGeneratedBox(t *testing.T) {
	item := ui.Div(ui.ID("item"), ui.Text("x"))
	l := liveDocWith(t, []ui.DocumentOption{ui.CSS(`
body { margin: 0; font-size: 10px; line-height: 20px; }
#item { width: 100px; }
#item:hover::before { content: "hovered"; display: block; height: 30px; }
`)}, item)
	in := l.win.Input()
	for i, over := range []bool{false, true, false, true} {
		at := geom.V2(300, 300)
		if over {
			at = geom.V2(10, 10)
		}
		in.SetMouse(at)
		l.frames(3)
		r, text, ok := ui.Generated(l.doc, item, false)
		assert.Equal(t, over, ok, "step %d", i)
		if over {
			assert.Equal(t, "hovered", text)
			assert.Equal(t, geom.Rect{W: 100, H: 30}, r)
			assert.Equal(t, float32(50), item.Box().H)
		} else {
			assert.Equal(t, float32(20), item.Box().H, "step %d: the element is laid out without it", i)
		}
	}
	assert.Empty(t, l.doc.Warnings())
}

func TestAGeneratedBoxTransitions(t *testing.T) {
	item := ui.Div(ui.ID("item"), ui.Text("x"))
	l := liveDocWith(t, []ui.DocumentOption{ui.CSS(`
#item::before { content: ""; display: block; height: 10px; background-color: #000; transition: background-color 1s linear; }
#item.lit::before { background-color: #fff; }
`)}, item)
	l.w.Time().Scale = 0
	item.AddClass("lit")
	l.frames(30)
	mid := ui.GeneratedBackground(item)
	assert.Greater(t, mid.R, float32(0.05))
	assert.Less(t, mid.R, float32(0.95), "::before runs its own transition")
	l.frames(60)
	assert.InDelta(t, 1, ui.GeneratedBackground(item).R, 0.001)
}

func TestListsNumberThroughTheListItemCounter(t *testing.T) {
	li := func(args ...ui.Arg) *ui.Node { return ui.Li(append(args, ui.Text("x"))...) }
	a, b, c := li(), li(ui.Attr("value", "9")), li(ui.ID("c"))
	roman := li(ui.ID("roman"))
	own := li(ui.ID("own"))
	d := ui.NewDocument(ui.CSS(`
#roman { list-style-type: upper-roman; }
#own::marker { content: "→ "; color: #f00; }
#c::marker { font-size: 30px; }
`))
	list := ui.Ol(ui.Attr("start", "3"), a, b, c, roman, own)
	d.Body().Append(list)
	d.Layout(geom.V2(400, 300))
	text := func(n *ui.Node) string {
		t.Helper()
		s, _, ok := ui.MarkerText(d, n)
		require.True(t, ok)
		return s
	}
	assert.Equal(t, []string{"3. ", "9. ", "10. ", "XI. ", "→ "}, []string{text(a), text(b), text(c), text(roman), text(own)})
	_, small, _ := ui.MarkerText(d, b)
	_, big, _ := ui.MarkerText(d, c)
	assert.Less(t, big, small*2, "a ::marker's font sizes its box: three glyphs at 30px are wider than two lines of 16px")

	list.Remove(a)
	d.Layout(geom.V2(400, 300))
	assert.Equal(t, []string{"9. ", "10. ", "XI. "}, []string{text(b), text(c), text(roman)}, "value still sets the count")
	b.RemoveAttr("value")
	d.Layout(geom.V2(400, 300))
	assert.Equal(t, []string{"3. ", "4. ", "V. "}, []string{text(b), text(c), text(roman)},
		"the items after a change are numbered again")
}

func TestAQStandsBetweenTheQuotesOfItsDepth(t *testing.T) {
	inner := ui.Q(ui.Text("in"))
	outer := ui.Q(ui.Text("out "), inner)
	d := ui.NewDocument()
	d.Body().Append(ui.P(ui.Text("He said "), outer))
	d.Layout(geom.V2(400, 300))
	open, text, ok := ui.Generated(d, outer, false)
	require.True(t, ok)
	assert.Equal(t, "“", text)
	assert.InDelta(t, outer.Box().X, open.X, 0.01, "the opening mark starts the q's box")
	shut, text, _ := ui.Generated(d, outer, true)
	assert.Equal(t, "”", text)
	assert.InDelta(t, outer.Box().X+outer.Box().W, shut.X+shut.W, 0.01, "the closing one ends it")
	_, text, _ = ui.Generated(d, inner, false)
	assert.Equal(t, "‘", text)
	_, text, _ = ui.Generated(d, inner, true)
	assert.Equal(t, "’", text)
	assert.Equal(t, "out in", outer.Text())
}

func TestAGeneratedBoxIsSizedAsItsDocumentsElementsAre(t *testing.T) {
	const sheet = `body { margin: 0; }
div::before { content: ""; display: block; width: 28px; height: 20px; padding: 2px 6px; border: 1px solid; }`
	for _, c := range []struct {
		opts []ui.DocumentOption
		want geom.Vec2
	}{
		{nil, geom.V2(28, 20)},
		{[]ui.DocumentOption{ui.BrowserSheets()}, geom.V2(42, 26)},
	} {
		d := ui.NewDocument(append(c.opts, ui.CSS(sheet))...)
		div := ui.Div()
		d.Body().Append(div)
		d.Layout(geom.V2(300, 200))
		r, _, ok := ui.Generated(d, div, false)
		require.True(t, ok)
		assert.Equal(t, c.want, geom.V2(r.W, r.H))
	}
}

func TestGeneratedTextFollowsItsAttributeAndIsNotTranslated(t *testing.T) {
	tr := newFrench(saveWords())
	d := ui.NewDocument(ui.WithTranslator(tr), ui.CSS(`p::before { content: "Save"; } p::after { content: attr(data-n); }`))
	p := ui.P(ui.Attr("data-n", "1"), ui.Text("Save"))
	d.Body().Append(p)
	d.Layout(geom.V2(300, 200))
	_, before, _ := ui.Generated(d, p, false)
	_, after, _ := ui.Generated(d, p, true)
	assert.Equal(t, "Save", before)
	assert.Equal(t, "1", after)
	assert.Equal(t, 1, tr.asked["Save"], "the element's text alone is a message")

	narrow := p.Box().W
	p.SetAttr("data-n", "1234567")
	d.Layout(geom.V2(300, 200))
	_, after, _ = ui.Generated(d, p, true)
	assert.Equal(t, "1234567", after)
	assert.Equal(t, narrow, p.Box().W)
	assert.Empty(t, d.Warnings())
	assert.Empty(t, align.Check(d))
}
