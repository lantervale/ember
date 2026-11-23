package layout

import (
	"strings"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

// partKind is the box of its own a control lays its ::before and ::after
// out round, as Chrome's shadow tree does.
type partKind uint8

const (
	noPart partKind = iota
	// innerPart is a progress's or a meter's inner element: a block as
	// tall as the control's content box, between the lines of its
	// ::before and its ::after.
	innerPart
	// sliderPart is a range's slider container, taking the room its
	// ::before and its ::after leave in the row they make.
	sliderPart
)

//nolint:gochecknoglobals // the parts' styles, fixed as Chrome's sheet gives them
var partStyles = [...]*style.Computed{innerPart: innerStyle(), sliderPart: sliderStyle()}

func innerStyle() *style.Computed {
	s := style.Initial()
	s.Display = style.Block
	s.Size.Height = style.Percent(100)
	return &s
}

func sliderStyle() *style.Computed {
	s := style.Initial()
	s.Display = style.Block
	s.Grow, s.Shrink, s.Basis = 1, 1, style.Percent(0)
	s.MinSize.Width = style.Px(0)
	return &s
}

// controlPartOf is the part a progress, a meter or a range with a ::before or
// an ::after lays them out round, noPart for any other element.
func controlPartOf(e *dom.Node) partKind {
	d := dom.Data(e)
	if d.Before == nil && d.After == nil {
		return noPart
	}
	if e.Tag() == dom.Progress || e.Tag() == dom.Meter {
		return innerPart
	}
	if isRange(e) {
		return sliderPart
	}
	return noPart
}

func isRange(e *dom.Node) bool {
	t, _ := e.Attr("type")
	return e.Tag() == dom.Input && strings.EqualFold(strings.TrimSpace(t), "range")
}

// ranged reports a range, which a block shrinks to its natural width as Chrome does a
// form control.
func (n *node) ranged() bool { return n.box.Node != nil && isRange(n.box.Node) }

// Chrome's range: its track's length, its own thumb's side.
const trackLength, nativeThumb = 129, 16

// RangeNatural is the content size of a range of style st on the page: its natural
// one, which its sizes left auto take.
func RangeNatural(st *style.Computed) geom.Vec2 {
	s := rangeNatural(st)
	if st.Mode.Vertical() {
		s = s.across()
	}
	return geom.Vec2{X: s.w, Y: s.h}
}

// rangeNatural is the content size of a range of physical style st in its own frame,
// as Chrome's shadow tree gives it: its track's length along it, and across it its own
// thumb's side, the track's length for one of appearance slider-vertical in horizontal
// lines, which runs down it, and nothing for one of appearance none (its parts then
// give it its min-size).
func rangeNatural(st *style.Computed) size {
	switch {
	case !st.Appearance.Native():
		return size{trackLength, 0}
	case st.Appearance == style.AppearanceSliderVertical && !st.Mode.Vertical():
		return size{trackLength, trackLength}
	}
	return size{trackLength, nativeThumb}
}

// buildParts gives n, a control of part k, its ::before, the part and
// its ::after: a range laid out as a flex row, a progress or a meter as
// a block of them. The control's children are never shown, as in Chrome.
func (t *tree) buildParts(n *node, k partKind) {
	d := dom.Data(n.box.Node)
	var flow []item
	if d.Before != nil {
		flow = append(flow, t.flatten(n, []*dom.Node{d.Before})...)
	}
	st := partStyles[k]
	part := &node{box: &box.Box{Kind: box.ControlPart, Style: st}, st: st, parent: n, fresh: true}
	if k == sliderPart {
		// the slider container holds the track, whose thickness the row takes
		part.natural, n.natural = size{0, n.natural.h}, size{}
	}
	flow = append(flow, item{part: part})
	if d.After != nil {
		flow = append(flow, t.flatten(n, []*dom.Node{d.After})...)
	}
	n.slider = k == sliderPart
	t.buildItems(n, t.lift(n, flow))
}
