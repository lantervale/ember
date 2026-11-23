package css

import (
	"strings"

	"github.com/lantervale/ember/ui/internal/dom"
)

const (
	kwScrollState = "scroll-state"
	kwBlockStart  = "block-start"
	kwBlockEnd    = "block-end"
	kwInlineStart = "inline-start"
	kwInlineEnd   = "inline-end"
	kwBlock       = "block"
	kwInline      = "inline"
)

// Sides are the physical sides of a box, a bit each.
type Sides uint8

const (
	SideTop Sides = 1 << iota
	SideRight
	SideBottom
	SideLeft
)

// SnapAxes are the axes a snap area is snapped to.
type SnapAxes uint8

const (
	SnappedX SnapAxes = 1 << iota
	SnappedY
)

// ScrollState is what a scroll-state() query asks of a container, as the
// last layout or scroll left it: the sides it is stuck to as a sticky
// box; those toward which it can scroll, and those its last relative
// scroll went toward on each axis, as a scroll container; the axes its
// snap container is snapped to it on, as a snap area.
type ScrollState struct {
	Stuck, Scrollable, Scrolled Sides
	Snapped                     SnapAxes
}

// scrollStateFeatures are the words each scroll-state() feature takes.
//
//nolint:gochecknoglobals // lookup table
var scrollStateFeatures = map[string][]string{
	"stuck": {kwNone, kwTop, kwRight, kwBottom, kwLeft, kwBlockStart, kwBlockEnd, kwInlineStart, kwInlineEnd},
	"scrollable": {kwNone, kwTop, kwRight, kwBottom, kwLeft, kwBlockStart, kwBlockEnd, kwInlineStart, kwInlineEnd,
		"x", "y", kwBlock, kwInline},
	"snapped": {kwNone, "x", "y", kwBlock, kwInline, kwBoth},
	"scrolled": {kwNone, kwTop, kwRight, kwBottom, kwLeft, kwBlockStart, kwBlockEnd, kwInlineStart, kwInlineEnd,
		"x", "y", kwBlock, kwInline},
}

// scrollStateQuery reads scroll-state(): one feature, or conditions on
// several in parentheses.
func (p *mediaParser) scrollStateQuery(fn Component) mediaCond {
	inner := items(fn.Args)
	if len(inner) > 0 && (inner[0].Kind == LParen || isWord(inner[0], "not")) {
		p.inScrollState = true
		cond, ok := p.condition(inner, true)
		p.inScrollState = false
		if !ok {
			p.warn(fn.Pos, "@container: %s is no scroll-state query; its query never holds", fn.String())
			return mediaCond{op: mediaUnknown}
		}
		return *cond
	}
	return p.scrollStateFeature(fn, inner)
}

// scrollStateFeature reads stuck, snapped or scrollable, alone or with
// one of its words.
func (p *mediaParser) scrollStateFeature(at Component, toks []Component) mediaCond {
	unknown := func() mediaCond {
		p.warn(at.Pos, "@container: %s is no scroll-state feature ui answers; the condition never holds",
			strings.TrimSpace(join(toks)))
		return mediaCond{op: mediaUnknown}
	}
	if len(toks) == 0 || toks[0].Kind != Ident {
		return unknown()
	}
	name := strings.ToLower(toks[0].Text)
	allowed, ok := scrollStateFeatures[name]
	if !ok {
		return unknown()
	}
	c := mediaCond{op: mediaScrollState, name: name}
	switch {
	case len(toks) == 1:
		return c
	case len(toks) == 3 && toks[1].Kind == Colon && toks[2].Kind == Ident:
		c.word = strings.ToLower(toks[2].Text)
		for _, w := range allowed {
			if w == c.word {
				return c
			}
		}
	}
	return unknown()
}

// scrollEval answers c, a scroll-state() feature, for s.
func (c *mediaCond) scrollEval(s *subject) tri {
	if s.scroll == nil {
		return unknown
	}
	var on Sides
	switch c.name {
	case "stuck":
		on = s.scroll.Stuck
	case "scrollable":
		on = s.scroll.Scrollable
	case "scrolled":
		on = s.scroll.Scrolled
	case "snapped":
		return c.snappedEval(s.scroll.Snapped)
	}
	switch c.word {
	case "":
		return triOf(on != 0)
	case kwNone:
		return triOf(on == 0)
	}
	return triOf(on&sidesOf(c.word, s.rtl) != 0)
}

// snappedEval answers scroll-state(snapped) for an area snapped on axes.
func (c *mediaCond) snappedEval(axes SnapAxes) tri {
	switch c.word {
	case "":
		return triOf(axes != 0)
	case kwNone:
		return triOf(axes == 0)
	case "x", kwInline:
		return triOf(axes&SnappedX != 0)
	case "y", kwBlock:
		return triOf(axes&SnappedY != 0)
	}
	return triOf(axes == SnappedX|SnappedY)
}

// sidesOf are the physical sides a scroll-state() word names on a
// horizontal container, right to left when rtl.
func sidesOf(word string, rtl bool) Sides {
	start, end := SideLeft, SideRight
	if rtl {
		start, end = end, start
	}
	switch word {
	case kwTop, kwBlockStart:
		return SideTop
	case kwBottom, kwBlockEnd:
		return SideBottom
	case kwLeft:
		return SideLeft
	case kwRight:
		return SideRight
	case kwInlineStart:
		return start
	case kwInlineEnd:
		return end
	case "x", kwInline:
		return SideLeft | SideRight
	case "y", kwBlock:
		return SideTop | SideBottom
	}
	return 0
}

// scrollOf is a's scroll state as the last layout or scroll left it.
func (m *measures) scrollOf(a *dom.Node) ScrollState {
	if m == nil {
		return ScrollState{}
	}
	return m.scroll[a]
}

// readsScrollState reports whether c asks scroll-state().
func (c *mediaCond) readsScrollState() bool {
	if c.op == mediaScrollState {
		return true
	}
	for i := range c.kids {
		if c.kids[i].readsScrollState() {
			return true
		}
	}
	return false
}

// RequeryScroll tells the cascade the scroll state of each scroll-state
// container, as state reports it after a layout or a scroll, and reports
// whether that changes the answer of some @container rule's condition:
// the caller then styles and lays out again.
func (t *Transitions) RequeryScroll(state func(n *dom.Node) ScrollState) bool {
	env := &t.env
	changed := false
	for n := range env.containers {
		st := dom.Data(n).Style
		if st == nil || !st.ContainerScrollState || !t.attached(n) {
			delete(env.scroll, n)
			continue
		}
		now := state(n)
		if old, had := env.scroll[n]; had && old == now {
			continue
		}
		before := env.subjectOf(n)
		if env.scroll == nil {
			env.scroll = map[*dom.Node]ScrollState{}
		}
		env.scroll[n] = now
		if t.answersMoved(before, env.subjectOf(n)) {
			restyle(n)
			if env.requery == nil {
				env.requery = map[*dom.Node]bool{}
			}
			env.requery[n] = true
			changed = true
		}
	}
	return changed
}
