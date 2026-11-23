package layout

import (
	"math"

	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

// A select with multiple or a size above 1 is a listbox, as Chrome lays
// it out: a block of its options and optgroups, each group's label a row
// of its own before its options, Chrome's shadow element, and as tall as
// its size (4 by default) in rows as tall as its tallest option or label.

// inListBox reports an optgroup of a listbox, which shows its label.
func inListBox(e *dom.Node) bool {
	return e.Tag() == dom.Optgroup && dom.ListBox(e.Parent())
}

// buildOptgroup gives an optgroup of a listbox its label row, then its
// options.
func (t *tree) buildOptgroup(n *node) {
	e := n.box.Node
	st := t.labelStyle(n.st)
	part := &node{box: &box.Box{Kind: box.ControlPart, Style: st}, st: st, parent: n, fresh: true}
	if shadow := dom.Data(e).Shadow; shadow != nil {
		text := Translated(shadow, t.translate)
		run := *st
		run.WhiteSpace = style.Pre
		b := &box.Box{Kind: box.TextRun, Node: shadow, Style: styleOf(shadow)}
		c := &inlineContent{owner: part, runs: []*textRun{{box: b, parent: part.box, st: &run, span: -1, text: text}},
			unit: unitsPerPoint(t.fonts), height: none}
		c.cut(t.fonts)
		part.inline = c
	}
	flow := append([]item{{part: part}}, t.flatten(n, e.Flow())...)
	t.buildItems(n, t.lift(n, flow))
}

// labelStyle is the style of the label row of an optgroup of style g: its
// text, an option's padding and least height.
func (t *tree) labelStyle(g *style.Computed) *style.Computed {
	if st := t.labelStyles[g]; st != nil {
		return st
	}
	s := style.Initial()
	s.Display, s.BoxSizing = style.Block, style.ContentBox
	s.Text = g.Text
	s.Visible = g.Visible
	s.Padding = style.Sides[style.Length]{Top: style.Px(0), Right: style.Px(2), Bottom: style.Px(1), Left: style.Px(2)}
	s.MinSize.Height = style.Px(1.2 * g.Font.Size)
	if t.labelStyles == nil {
		t.labelStyles = map[*style.Computed]*style.Computed{}
	}
	t.labelStyles[g] = &s
	return &s
}

// listBox is in with the height of a listbox n where in leaves it open:
// its rows, each as tall as its tallest option or label, else as its
// font and a point, within its own min and max.
func (t *tree) listBox(n *node, in input) input {
	e := n.box.Node
	if some(in.known.h) || e == nil || !dom.ListBox(e) || n.st.Display == style.DisplayNone {
		return in
	}
	var row float32
	measure := func(k *node) {
		if k.st.Display == style.DisplayNone {
			return
		}
		row = max(row, t.measureChildSize(k, noSize, in.parent, spaces{w: maxContentSpace, h: maxContentSpace},
			inherentSize, vertical, [2]bool{}))
	}
	for _, k := range n.kids {
		switch tag := tagOf(k); {
		case tag == dom.Option:
			measure(k)
		case tag == dom.Optgroup && k.st.Display != style.DisplayNone:
			for _, o := range k.kids {
				if o.box.Kind == box.ControlPart || tagOf(o) == dom.Option {
					measure(o)
				}
			}
		}
	}
	if row == 0 && t.fonts != nil {
		m := t.fonts.Metrics(n.st.Font)
		row = float32(math.Round(float64(m.Ascent))+math.Round(float64(m.Descent))) + 1
	}
	st := n.st
	pb := edges(st.Padding, in.parent.w).add(n.borders())
	inset := pb.sum().add(size{n.ownGutter().x, n.ownGutter().y})
	h := float32(dom.ListBoxRows(e))*row + inset.h
	if in.sizing == inherentSize {
		adj := boxSizingAdjustment(st, pb.sum())
		lo := stretched(resolveSize(st.MinSize, in.parent).mAdd(adj), st.MinSize, st, in.parent)
		hi := stretched(resolveMax(st.MaxSize, in.parent).mAdd(adj), st.MaxSize, st, in.parent)
		if styled := resolveSize(st.Size, in.parent).mAdd(adj); some(styled.h) {
			h = styled.h
		}
		h = size{0, h}.mClamp(lo, hi).h
	}
	in.known.h = max(h, inset.h)
	return in
}

func tagOf(n *node) dom.Tag {
	if n.box.Node == nil || n.box.Kind != box.Element {
		return dom.TextTag
	}
	return n.box.Node.Tag()
}
