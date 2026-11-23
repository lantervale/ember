package paint

import (
	"sync"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/ui/control"
	"github.com/lantervale/ember/ui/internal/box"
)

// LiveSet is what a paint drew live (decision 0052): a focused editable
// field's caret, an indeterminate progress's slide, an open select's list,
// each with its place, clips and transform, so PaintLive can draw them
// again over a replay of the rest; and whether anything painted after one
// covered it, when they can't be drawn on top.
type LiveSet struct {
	parts   []livePart
	covered bool
	// rects are where the parts lie in the root's space, for the check.
	rects []geom.Rect
	// owners are the boxes the parts belong to, whose own outlines ring
	// them without covering them.
	owners []*box.Box
	// viewport is the paint's, which places an open list.
	viewport geom.Rect
	// slabs hold the parts' clip chains, copied from the paint's, which
	// the next paint reuses.
	slabs [][]clip
	used  int
	// gaps are where the parts drawn in their place lie in the surface's
	// drawing, and fills what each gap draws: a part, or the stacking
	// context round the pictures parts were drawn in. gapless is set when
	// one could not be marked; open while a part's gap is being drawn,
	// pictured while a part lies in a picture no gap holds yet.
	gaps                    []twod.Gap
	fills                   []livePart
	gapless, open, pictured bool
	// listsApart is set when the paint left the open lists out for
	// DrawLists.
	listsApart bool
	// frame is what the paint took of its Frame, for a gap's stacking
	// context drawn again.
	frame liveFrame
}

// liveFrame is what a stacking context drawn again in its gap takes of
// the paint's Frame and walk.
type liveFrame struct {
	keep     *box.Box
	sel      Selection
	inactive bool
	system   Highlight
	canvas   *box.Box
}

type liveKind uint8

const (
	liveCaret liveKind = iota
	liveSlide
	liveList
	liveCanvas
	// liveStack is a stacking context holding live parts in a picture:
	// drawn again whole, as the picture must be.
	liveStack
)

type livePart struct {
	kind liveKind
	e    entry
	xf   geom.Transform2D
	ink  color.Color
	rect geom.Rect
	list popup
	// timed is a canvas drawn each frame (control.Canvas.Live).
	timed bool
	// alpha is the opacity a stacking context is drawn in, pseudo whether
	// it is a float or a positioned box holding only its flow.
	alpha  float32
	pseudo bool
}

// ListsApart reports whether the paint left open lists out for
// DrawLists.
func (l *LiveSet) ListsApart() bool { return l.listsApart }

// Any reports whether the paint drew anything live.
func (l *LiveSet) Any() bool { return len(l.parts) > 0 }

// Timed reports whether a part moves with the clock: all but canvases,
// which change only when invalidated, unless they are drawn each frame.
func (l *LiveSet) Timed() bool {
	for _, p := range l.parts {
		if p.kind != liveCanvas || p.timed {
			return true
		}
	}
	return false
}

// Covered reports whether something painted after a live part overlaps
// it, so the parts can't go on top of a replay.
func (l *LiveSet) Covered() bool { return l.covered }

func (l *LiveSet) reset() {
	clear(l.parts)
	l.parts, l.rects, l.owners, l.covered, l.used = l.parts[:0], l.rects[:0], l.owners[:0], false, 0
	clear(l.fills)
	l.gaps, l.fills, l.gapless, l.open, l.pictured = l.gaps[:0], l.fills[:0], false, false, false
	l.listsApart = false
	l.frame = liveFrame{}
}

// Gaps are where a paint drew its parts in their place in the surface's
// drawing (decision 0166): a part's own, or for parts in a picture (an
// opacity, a filter or a mask round them) the outermost stacking context
// drawing one. A recording kept round them replays with each drawn again
// by FillGap. None when the parts were drawn apart, or on a surface
// that does not mark its drawing, or in a picture of the whole page.
func (l *LiveSet) Gaps() []twod.Gap {
	if l.gapless || l.pictured {
		return nil
	}
	return l.gaps
}

// marker is a surface whose drawing can be marked, to be recorded.
type marker interface{ Mark() twod.Mark }

// Live has the paint keep what it draws live in set, set emptied first.
// Apart, it draws the tree without them, for PaintLive to draw on top.
func Live(set *LiveSet, apart bool) Option {
	return func(p *painter) {
		set.reset()
		p.live, p.apart = set, apart
	}
}

// record keeps part, reporting whether it is to be drawn now: not when
// the paint keeps live parts apart. A part under another transform than
// the root's counts as covered: it can't be checked; so does one inside a
// group picture, which an opacity or a filter of the group must reach.
func (p *painter) record(part livePart, area geom.Rect) bool {
	if p.live == nil {
		return true
	}
	part.xf = p.s.Transform()
	part.e.clip = p.live.keep(part.e.clip)
	p.live.parts = append(p.live.parts, part)
	p.live.viewport = p.viewport
	if part.xf != p.base || p.pictures > 0 && !p.paged || p.pictures > 1 {
		p.live.covered = true
	} else if part.kind != liveList {
		p.live.rects = append(p.live.rects, area)
		p.live.owners = append(p.live.owners, part.e.b)
	}
	if !p.apart {
		m, ok := p.s.(marker)
		switch {
		case !ok:
			p.live.gapless = true
		case p.pictures > 0:
			p.live.pictured = true
		default:
			p.live.gaps = append(p.live.gaps, twod.Gap{From: m.Mark()})
			p.live.fills = append(p.live.fills, part)
			p.live.open = true
		}
	}
	return !p.apart
}

// endGap ends the gap of the part record last let draw in its place.
func (p *painter) endGap() {
	if p.live == nil || !p.live.open {
		return
	}
	p.live.open = false
	if m, ok := p.s.(marker); ok {
		p.live.gaps[len(p.live.gaps)-1].To = m.Mark()
	}
}

// stackGap is where a stacking context's drawing begins, for a gap round
// it should a part be drawn in a picture inside it: only outside every
// picture, while live parts are drawn in their place.
func (p *painter) stackGap() (twod.Mark, bool) {
	if p.live == nil || p.apart || p.probing || p.pictures > 0 || p.live.pictured {
		return twod.Mark{}, false
	}
	m, ok := p.s.(marker)
	if !ok {
		return twod.Mark{}, false
	}
	return m.Mark(), true
}

// endStackGap has the stacking context e, begun at from under xf in
// alpha, be a gap when a part was drawn in a picture inside it: its
// drawing is no part of the surface's, the whole context drawn again.
func (p *painter) endStackGap(from twod.Mark, e entry, xf geom.Transform2D, alpha float32, pseudo bool) {
	if !p.live.pictured {
		return
	}
	m, _ := p.s.(marker)
	p.live.pictured = false
	e.clip = p.live.keep(e.clip)
	p.live.gaps = append(p.live.gaps, twod.Gap{From: from, To: m.Mark()})
	p.live.fills = append(p.live.fills, livePart{kind: liveStack, e: e, xf: xf, alpha: alpha, pseudo: pseudo})
	p.live.frame = liveFrame{keep: p.keep, sel: p.sel, inactive: p.inactive, system: p.system, canvas: p.canvas}
}

// keep is a copy of the clip chain c in l's slabs.
func (l *LiveSet) keep(c *clip) *clip {
	if c == nil {
		return nil
	}
	parent := l.keep(c.parent)
	i, j := l.used/slab, l.used%slab
	if i == len(l.slabs) {
		l.slabs = append(l.slabs, make([]clip, slab))
	}
	l.used++
	l.slabs[i][j] = *c
	l.slabs[i][j].parent = parent
	return &l.slabs[i][j]
}

// cover notes ink drawn now for b: it covers a live part it overlaps,
// but an element's own outline, a ring round its box, never covers the
// parts inside the box.
func (p *painter) cover(b *box.Box, ink geom.Rect, ring bool) {
	if p.live == nil || p.live.covered {
		return
	}
	for i, r := range p.live.rects {
		if ring && p.live.owners[i] == b {
			continue
		}
		if overlaps(ink, r) {
			p.live.covered = true
			return
		}
	}
}

// painters keeps painters between paints, with their scratch room.
//
//nolint:gochecknoglobals // a sync.Pool every paint shares
var painters = sync.Pool{New: func() any { return new(painter) }}

// PaintLive draws set's live parts on s at the clock's time seconds, each
// under its own clips and transform, as Paint with opts drew them: over a
// replay of the rest of the frame.
//
//nolint:revive // Live is already the option that keeps the parts; PaintLive pairs with Paint.
func PaintLive(s Surface, set *LiveSet, time float32, opts ...Option) {
	paintLive(s, set, -1, time, opts)
}

// FillGap draws set's part i alone, as PaintLive draws it: in the
// gap a recording round set.Gaps() left for it.
func FillGap(s Surface, set *LiveSet, i int, time float32, opts ...Option) {
	paintLive(s, set, i, time, opts)
}

// drawPart draws part as paint drew it, under its own clips and
// transform.
func (p *painter) drawPart(part livePart) {
	p.s.SetTransform(part.xf)
	switch part.kind {
	case liveCaret:
		p.use(part.e.clip)
		p.drawCaret(part.e, part.ink)
	case liveSlide:
		p.use(part.e.clip)
		p.slide(part.rect, part.ink)
	case liveList:
		p.use(nil)
		p.drawList(part.list, p.place(part.list))
	case liveCanvas:
		p.use(part.e.clip)
		if m, ok := p.machine(part.e.b.Node).(*control.Canvas); ok {
			p.drawCanvas(m, part.rect)
		}
	case liveStack:
		p.use(part.e.clip)
		p.stack(part.e, part.alpha, part.pseudo)
	}
}

func paintLive(s Surface, set *LiveSet, gap int, time float32, opts []Option) {
	p := pooled[painter](&painters)
	defer painters.Put(p)
	*p = painter{
		s: s, base: s.Transform(),
		scratch: p.scratch[:0], chain: p.chain[:0], popups: p.popups[:0], top: p.top[:0], slabs: p.slabs,
	}
	for _, o := range opts {
		o(p)
	}
	p.time, p.timed = time, true
	p.viewport = set.viewport
	saved := s.Transform()
	if gap < 0 {
		for _, part := range set.parts {
			if part.kind != liveList || !set.listsApart {
				p.drawPart(part)
			}
		}
	} else {
		f := set.frame
		p.keep, p.sel, p.inactive, p.system, p.canvas = f.keep, f.sel, f.inactive, f.system, f.canvas
		p.drawPart(set.fills[gap])
	}
	p.use(nil)
	s.SetTransform(saved)
}
