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
