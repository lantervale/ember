package ui_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/world"
)

// rightHeard counts what a Document says of the right button.
type rightHeard struct {
	contexts []*ui.Node
	selected []string
}

func hearRight(doc *ui.Document) *rightHeard {
	h := &rightHeard{}
	ui.OnContextMenu(func(e ui.Event) { h.contexts = append(h.contexts, e.Node) }).Apply(doc.Body())
	ui.OnRightDragSelect(func() { h.selected = append(h.selected, doc.Selection().Text) }).Apply(doc.Body())
	return h
}

func rightDrag(in *input.State, frames func(int), from geom.Vec2, to ...geom.Vec2) {
	in.SetMouse(from)
	frames(1)
	in.SetMouseButton(input.MouseRight, true)
	frames(1)
	for _, at := range to {
		in.SetMouse(at)
		frames(1)
	}
	in.SetMouseButton(input.MouseRight, false)
	frames(2)
}

func rightDragDoc(t *testing.T, opts []ui.DocumentOption, nodes ...*ui.Node) (*live, *rightHeard) {
	t.Helper()
	l := liveDocWith(t, append([]ui.DocumentOption{ui.CSS(selectable)}, opts...), nodes...)
	return l, hearRight(l.doc)
}

func TestARightDragSelectsNothingByDefault(t *testing.T) {
	p := ui.P(ui.Text("The quick brown fox"))
	l, heard := rightDragDoc(t, nil, p)
	require.False(t, l.doc.RightDragSelects())

	rightDrag(l.win.Input(), l.frames, l.at(t, p, 4), l.at(t, p, 9))
	assert.True(t, l.doc.Selection().Empty(), "as Chrome, the right button selects nothing")
	assert.Equal(t, []*ui.Node{p}, heard.contexts, "pressed and released on the paragraph, it is a right click")
	assert.Empty(t, heard.selected)
}

func TestARightDragSelectsTheTextBetweenItsEnds(t *testing.T) {
	first, second := ui.P(ui.Text("The quick brown fox")), ui.P(ui.Text("jumps over the dog"))
	l, heard := rightDragDoc(t, []ui.DocumentOption{ui.RightDragSelects()}, first, second)
	require.True(t, l.doc.RightDragSelects())
	var r world.Reader[ui.RightDragSelected]
	events := 0
	frames := func(n int) {
		for range n {
			l.frames(1)
			for e := range r.Read(l.w) {
				assert.Same(t, l.doc.Body(), e.Node)
				events++
			}
		}
	}

	rightDrag(l.win.Input(), frames, l.at(t, first, 4), l.at(t, first, 9))
	s := l.doc.Selection()
	assert.Equal(t, "quick", s.Text)
	assert.Equal(t, ui.Position{Node: textOf(first), Offset: 4}, s.Anchor, "the anchor is where the press fell")
	assert.Equal(t, ui.Position{Node: textOf(first), Offset: 9}, s.Focus)
	assert.Empty(t, heard.contexts, "a drag that selected is no right click")
	assert.Equal(t, []string{"quick"}, heard.selected)
	assert.Equal(t, 1, events)

	rightDrag(l.win.Input(), frames, l.at(t, second, 5), l.at(t, second, 2), l.at(t, first, 10))
	assert.Equal(t, "brown fox\n\njumps", l.doc.Selection().Text, "backwards, across two paragraphs")
	assert.Empty(t, heard.contexts)
	assert.Equal(t, []string{"quick", "brown fox\n\njumps"}, heard.selected, "said once, when the button is let go")
	assert.Equal(t, 2, events)
}

func TestARightClickStaysARightClickWhereARightDragSelects(t *testing.T) {
	p := ui.P(ui.Text("The quick brown fox"))
	l, heard := rightDragDoc(t, []ui.DocumentOption{ui.RightDragSelects()}, p)
	in := l.win.Input()

	l.dragged(l.at(t, p, 4), l.at(t, p, 9))
	require.Equal(t, "quick", l.doc.Selection().Text)
	assert.Empty(t, heard.selected, "the left button's selection is not the right one's")

	rightDrag(in, l.frames, l.at(t, p, 12))
	assert.Equal(t, "quick", l.doc.Selection().Text, "a right click leaves the selection as it is")
	assert.Equal(t, []*ui.Node{p}, heard.contexts)
	assert.Empty(t, heard.selected)

	at := l.at(t, p, 12)
	rightDrag(in, l.frames, at, at.Add(geom.V2(2, 1)))
	assert.Equal(t, "quick", l.doc.Selection().Text, "nor does a hand that shook")
	assert.Len(t, heard.contexts, 2)
	assert.Empty(t, heard.selected)

	rightDrag(in, l.frames, at, l.at(t, p, 16), at)
	assert.True(t, l.doc.Selection().Empty(), "dragged back to where it began")
	assert.Len(t, heard.contexts, 3, "a drag that selected nothing is a right click")
	assert.Empty(t, heard.selected)
}

func TestARightDragBeginsOnlyWhereALeftOneWould(t *testing.T) {
	p := ui.P(ui.Text("The quick brown fox"))
	link := ui.A(ui.Attr("href", "#"), ui.Text("a link to follow"))
	fixed := ui.P(ui.Style("user-select: none"), ui.Text("not to be selected"))
	l, heard := rightDragDoc(t, []ui.DocumentOption{ui.RightDragSelects()}, p, ui.P(link), fixed)
	in := l.win.Input()

	box := link.Box()
	rightDrag(in, l.frames, geom.V2(box.X+2, box.Y+box.H/2), geom.V2(box.X+60, box.Y+box.H/2))
	assert.True(t, l.doc.Selection().Empty(), "a link keeps the pointer")
	assert.Equal(t, []*ui.Node{link}, heard.contexts)

	box = fixed.Box()
	rightDrag(in, l.frames, geom.V2(box.X+2, box.Y+box.H/2), geom.V2(box.X+60, box.Y+box.H/2))
	assert.True(t, l.doc.Selection().Empty())
	assert.Equal(t, []*ui.Node{link, fixed}, heard.contexts)

	in.SetMouse(l.at(t, p, 4))
	l.frames(1)
	in.SetMouseButton(input.MouseRight, true)
	l.frames(1)
	in.SetMouse(l.at(t, p, 9))
	l.frames(1)
	in.SetMouseButton(input.MouseLeft, true)
	l.frames(1)
	in.SetMouseButton(input.MouseLeft, false)
	in.SetMouseButton(input.MouseRight, false)
	l.frames(2)
	assert.Empty(t, heard.selected, "a left press takes the selection from the right button")
	assert.True(t, l.doc.Selection().Empty())
}

func TestARightDragPastAScrollersEdgeScrollsIt(t *testing.T) {
	var lines []ui.Arg
	for range 30 {
		lines = append(lines, ui.P(ui.Text("a line of text")))
	}
	scroller := ui.Div(append(lines, ui.Style("height: 100px; overflow: auto"))...)
	l, heard := rightDragDoc(t, []ui.DocumentOption{ui.RightDragSelects()}, scroller)
	first := scroller.Children()[0]
	in := l.win.Input()

	in.SetMouse(l.at(t, first, 2))
	l.frames(1)
	in.SetMouseButton(input.MouseRight, true)
	l.frames(1)
	in.SetMouse(geom.V2(40, 160))
	l.frames(30)
	in.SetMouseButton(input.MouseRight, false)
	l.frames(2)
	assert.Greater(t, scroller.Box().Y-first.Box().Y, float32(20), "held below the scroller, it scrolls")
	assert.Greater(t, len(l.doc.Selection().Text), len("a line of text\n\na line"), "and the selection follows")
	assert.Len(t, heard.selected, 1)
	assert.Empty(t, heard.contexts)
}

// rightFrames is a page with a paragraph and an iframe showing a
// Document where a right drag selects, an iframe of that one showing a
// third.
type rightFrames struct {
	host, child, inner      *ui.Document
	slot, nested            *ui.Node
	words, childWords, deep *ui.Node
	heard, inChild, inDeep  *rightHeard
	w                       *world.World
	in                      *input.State
}

func newRightFrames(t *testing.T, host ...ui.DocumentOption) *rightFrames {
	t.Helper()
	const sheet = `body { margin: 0; user-select: text; font-size: 16px; line-height: 20px } p { margin: 0 }
iframe { display: block; border: 0; width: 300px; height: 100px; margin: 20px 0 0 40px }`
	p := &rightFrames{}
	p.inner = ui.NewDocument(ui.CSS(sheet), ui.RightDragSelects())
	p.deep = ui.P(ui.Text("words of the innermost frame"))
	p.inner.Body().Append(p.deep)
	p.child = ui.NewDocument(ui.CSS(sheet), ui.RightDragSelects())
	p.childWords = ui.P(ui.Text("words of the frame"))
	p.nested = ui.Iframe(p.inner, ui.ID("nested"))
	p.child.Body().Append(p.childWords, p.nested)
	p.host = ui.NewDocument(append([]ui.DocumentOption{ui.CSS(sheet), ui.CSS("#slot { height: 200px }")}, host...)...)
	p.words = ui.P(ui.Text("words of the page"))
	p.slot = ui.Iframe(p.child, ui.ID("slot"))
	p.host.Body().Append(p.words, p.slot)
	p.heard, p.inChild, p.inDeep = hearRight(p.host), hearRight(p.child), hearRight(p.inner)
	win := platform.NewWindow(platform.Size(800, 500))
	win.Add(p.host)
	p.w = ember.New().AddEntities(win)
	p.in = win.Input()
	p.w.RunFrames(2)
	return p
}

func (p *rightFrames) frames(n int) { p.w.RunFrames(n) }

// childPoint is the window's point of a point of the child Document,
// innerPoint of one of the innermost.
func (p *rightFrames) childPoint(x, y float32) geom.Vec2 {
	r, _ := p.host.BoxOf(p.slot)
	return geom.V2(r.X+x, r.Y+y)
}

func (p *rightFrames) innerPoint(x, y float32) geom.Vec2 {
	r, _ := p.child.BoxOf(p.nested)
	return p.childPoint(r.X+x, r.Y+y)
}

func TestARightDragInAnIframeSelectsThereAndIsNoRightClickOfThePage(t *testing.T) {
	p := newRightFrames(t)

	rightDrag(p.in, p.frames, p.childPoint(60, 10))
	assert.Equal(t, []*ui.Node{p.slot}, p.heard.contexts, "a right click in the frame is one on the iframe")
	assert.Equal(t, []*ui.Node{p.childWords}, p.inChild.contexts)
	assert.True(t, p.child.Selection().Empty())

	rightDrag(p.in, p.frames, p.childPoint(1, 10), p.childPoint(60, 10))
	assert.NotEmpty(t, p.child.Selection().Text)
	assert.Contains(t, p.childWords.Text(), p.child.Selection().Text)
	assert.Equal(t, []string{p.child.Selection().Text}, p.inChild.selected)
	assert.Len(t, p.heard.contexts, 1, "the page hears no right click for a drag that selected in its frame")
	assert.Len(t, p.inChild.contexts, 1)
	assert.Empty(t, p.heard.selected, "the selection is the frame's")

	rightDrag(p.in, p.frames, p.childPoint(60, 10))
	assert.Len(t, p.heard.contexts, 2, "the next right click is one again")
	assert.Len(t, p.inChild.contexts, 2)
	assert.Len(t, p.inChild.selected, 1)

	rightDrag(p.in, p.frames, p.childPoint(1, 10), p.childPoint(60, 10), p.childPoint(400, 10))
	assert.Equal(t, []string{"words of", "words of the frame"}, p.inChild.selected,
		"let go over the page, the drag is still the frame's")
	assert.Len(t, p.heard.contexts, 2)
	assert.True(t, p.host.Selection().Empty())

	words, _ := p.host.BoxOf(p.words)
	rightDrag(p.in, p.frames, geom.V2(words.X+1, words.Y+10), geom.V2(words.X+60, words.Y+10))
	assert.True(t, p.host.Selection().Empty(), "the page has not the option")
	assert.Len(t, p.heard.contexts, 3)
}

func TestARightDragInAnIframeOfAnIframe(t *testing.T) {
	p := newRightFrames(t, ui.RightDragSelects())

	rightDrag(p.in, p.frames, p.innerPoint(1, 10), p.innerPoint(60, 10))
	assert.NotEmpty(t, p.inner.Selection().Text)
	assert.Len(t, p.inDeep.selected, 1)
	assert.Empty(t, p.inDeep.contexts)
	assert.Empty(t, p.inChild.contexts, "neither Document round it hears a right click")
	assert.Empty(t, p.heard.contexts)
	assert.Empty(t, p.inChild.selected)

	rightDrag(p.in, p.frames, p.innerPoint(60, 10))
	assert.Equal(t, []*ui.Node{p.deep}, p.inDeep.contexts)
	assert.Equal(t, []*ui.Node{p.nested}, p.inChild.contexts)
	assert.Equal(t, []*ui.Node{p.slot}, p.heard.contexts)

	words, _ := p.host.BoxOf(p.words)
	rightDrag(p.in, p.frames, geom.V2(words.X+1, words.Y+10), geom.V2(words.X+60, words.Y+10))
	assert.NotEmpty(t, p.host.Selection().Text)
	assert.True(t, p.inner.Selection().Empty(), "one page has one selection")
	assert.Len(t, p.heard.selected, 1)
	assert.Len(t, p.heard.contexts, 1)
}
