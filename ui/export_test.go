package ui

import (
	"testing"

	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/ui/paint"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/internal/textedit"
	"github.com/lantervale/ember/ui/control"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

// LandImages waits for the pictures d is fetching, then lays it out at
// size to take them, until none is on its way.
func LandImages(d *Document, size geom.Vec2) {
	for {
		d.fetching.Wait()
		d.Layout(size)
		if len(d.fetching.onWay) == 0 {
			return
		}
	}
}

// Restyled is how many elements d has worked the style of out so far.
func Restyled(d *Document) int { return d.trans.Restyled() }

// ShownBackground is what paint draws as n's background now.
func ShownBackground(n *Node) color.Color { return dom.Data(n).Shown.Background }

// ShownInks are n's color, background-color and border-top-color as
// paint draws them now.
func ShownInks(n *Node) [3]color.Color {
	s := dom.Data(n).Shown
	return [3]color.Color{s.Color, s.Background, s.BorderColor.Top}
}

// ShownOpacity is n's opacity as paint draws it now.
func ShownOpacity(n *Node) float32 { return dom.Data(n).Shown.Opacity }

// ShownAccent is n's accent-color now, what paint fills a range with.
func ShownAccent(n *Node) color.Color { return dom.Data(n).Shown.AccentColor }

// ScrollSize is the size of n's scrollable overflow from the last layout,
// the viewport's for the root of a Document whose viewport scrolls.
func ScrollSize(d *Document, n *Node) geom.Vec2 { return d.scroller(n).Content }

// SetPlatform makes every Document follow p's key conventions until t ends.
func SetPlatform(t testing.TB, p textedit.Platform) {
	was := currentPlatform
	currentPlatform = func() textedit.Platform { return p }
	t.Cleanup(func() { currentPlatform = was })
}

// Keys runs d's keyboard handling for this frame, as interact does, without a canvas.
func Keys(d *Document, in *input.State) { d.keys(nil, in) }

// OptionTexts is what a select's list shows of its options.
func OptionTexts(d *Document, n *Node) []string {
	s, ok := d.machine(n).(*control.Select)
	if !ok {
		return nil
	}
	var out []string
	for _, o := range s.Options {
		out = append(out, o.Text())
	}
	return out
}

// HitAt is the element a click at p, in the Document's space, lands on
// after its last layout.
func HitAt(d *Document, p geom.Vec2) *Node {
	h := paint.HitTest(d.inked(), p, paint.Controls(d.machine), paint.Viewport(d.viewport), paint.Scale(d.fonts.Scale()))
	for b := h; b != nil; b = b.Parent {
		if b.Node != nil {
			return b.Node
		}
	}
	return nil
}

// Generated is the border box, in the Document's space, and the text of
// n's ::before box, or its ::after box when after; false while the
// pseudo-element has none.
func Generated(d *Document, n *Node, after bool) (geom.Rect, string, bool) {
	g := dom.Data(n).Before
	if after {
		g = dom.Data(n).After
	}
	if g == nil || d.tree == nil || d.tree.Of[g] == nil {
		return geom.Rect{}, "", false
	}
	return g.Box(), g.Text(), true
}

// GeneratedBackground is what paint draws as the background of n's
// ::before box now.
func GeneratedBackground(n *Node) color.Color { return dom.Data(dom.Data(n).Before).Shown.Background }

// MarkerText is the text of n's ::marker box and its left edge from n's
// border box, false without one.
func MarkerText(d *Document, n *Node) (string, float32, bool) {
	b := d.tree.Of[n]
	if b == nil {
		return "", 0, false
	}
	for _, c := range b.Children {
		if c.Kind == box.Marker {
			text := ""
			if len(c.Lines) > 0 {
				text = c.Lines[0].Text
			}
			return text, c.Pos.X, true
		}
	}
	return "", 0, false
}

// HoveredElement is the innermost element the pointer is over.
func HoveredElement(d *Document) *Node {
	if len(d.hovered) == 0 {
		return nil
	}
	return d.hovered[0]
}

// ScrollTo scrolls n, the root for the viewport of a Document whose
// viewport scrolls, to off, as a wheel or a dragged thumb does.
func ScrollTo(d *Document, n *Node, off geom.Vec2) { d.setScroll(d.scroller(n), off) }

// Placed counts the nodes d's layouts pointed at their boxes so far: it
// moves with each layout.
func Placed(d *Document) int { return d.placed }

// Painted is how many times d was painted, a replayed frame not being one.
func Painted(d *Document) uint64 { return d.painted }

// Framed reports whether d is tied to an iframe that shows it.
func Framed(d *Document) bool { return d.framed != nil }

// Bidi is n's computed direction and unicode-bidi, as CSS writes them.
func Bidi(n *Node) (string, string) {
	st := dom.Data(n).Style
	dir := "ltr"
	if st.Dir == style.RTL {
		dir = "rtl"
	}
	return dir, [...]string{"normal", "embed", "isolate", "bidi-override", "isolate-override", "plaintext"}[st.Bidi]
}

// ShownFrame is the frame d shows of the animation url names, -1 when it
// plays none there.
func ShownFrame(d *Document, url string) int {
	if p, ok := d.playing[url]; ok {
		return p.frame
	}
	return -1
}

// fieldLine is line l of field n's text run, laid out, in the document's
// space as paint places it.
func fieldLine(d *Document, n *Node, l textedit.Layout, line int) geom.Rect {
	b := d.tree.Of[n]
	at := n.Box()
	origin := geom.V2(at.X, at.Y).Sub(b.Scroll)
	for _, c := range b.Children {
		if c.Kind != box.TextRun {
			continue
		}
		if a, ok := l.(*area); ok {
			f := a.Row(line)
			return geom.Rect{X: origin.X + c.Pos.X + f.X, Y: origin.Y + c.Pos.Y + f.Y, W: f.W, H: f.H}
		}
		if line < len(c.Lines) {
			f := c.Lines[line].Rect
			for _, g := range c.Lines {
				f.X = min(f.X, g.Rect.X)
			}
			return geom.Rect{X: origin.X + c.Pos.X + f.X, Y: origin.Y + c.Pos.Y + f.Y, W: f.W, H: f.H}
		}
	}
	return geom.Rect{}
}

// FieldSelectionRects are the rectangles text field n's selection is drawn
// in, in the document's space, laid out as a frame would.
func FieldSelectionRects(d *Document, n *Node) []geom.Rect {
	d.layText()
	t := textOf(d.machine(n))
	from, to := t.Editor.Selection()
	if from == to {
		return nil
	}
	l := t.Layout
	var out []geom.Rect
	for line := l.Line(from); line <= l.Line(to); line++ {
		row := fieldLine(d, n, l, line)
		ranged, _ := l.(interface {
			Ranges(line, from, to int) [][2]float32
		})
		for _, r := range ranged.Ranges(line, from, to) {
			out = append(out, geom.Rect{X: row.X + r[0], Y: row.Y, W: r[1] - r[0], H: row.H})
		}
	}
	return out
}

// FieldCaretRect is where text field n's caret stands, in the document's
// space, laid out as a frame would.
func FieldCaretRect(d *Document, n *Node) geom.Rect {
	d.layText()
	t := textOf(d.machine(n))
	i := t.Editor.Caret()
	row := fieldLine(d, n, t.Layout, t.Layout.Line(i))
	return geom.Rect{X: row.X + t.Layout.X(i), Y: row.Y, W: 1, H: row.H}
}
