package layout

import (
	"strings"

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
	if t, _ := e.Attr("type"); e.Tag() == dom.Input && strings.EqualFold(strings.TrimSpace(t), "range") {
		return sliderPart
	}
	return noPart
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
	flow = append(flow, item{part: &node{box: &box.Box{Kind: box.ControlPart, Style: st}, st: st, parent: n,
		fresh: true}})
	if d.After != nil {
		flow = append(flow, t.flatten(n, []*dom.Node{d.After})...)
	}
	n.slider = k == sliderPart
	t.buildItems(n, t.lift(n, flow))
}
