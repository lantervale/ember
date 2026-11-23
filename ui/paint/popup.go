package paint

import (
	"fmt"
	"sync"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/ui/control"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/cull"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
	"github.com/lantervale/ember/ui/layout"
)

// An open select's list draws over everything, after the whole tree, like
// a modal's top layer: under the select, above it when it doesn't fit,
// kept inside the root, in the select's look: its background, border and
// colour, the highlighted option in its accent-color with its text in the
// select's colour, or Chrome's blue and white while the accent is auto.

//nolint:gochecknoglobals // fixed values every paint reads; Go has no constant colours, tables or shapes
var (
	highlight     = color.Hex("#0075ff")
	highlightText = color.White
	popupShadow   = color.RGBA(0, 0, 0, 0.25)
)

const (
	// optionPad is the room either side of an option's label, and dimmed
	// what a disabled option's colour keeps.
	optionPad = 6
	dimmed    = 0.4
)

// popup is an open select met while painting.
type popup struct {
	b  *box.Box
	at geom.Vec2
	m  *control.Select
	xf geom.Transform2D
}

// menu is where an open select's options lie, in the root's space: its
// lines, each group's label before its options, rows of them shown from
// first.
type menu struct {
	rect        geom.Rect
	border      float32
	row         float32
	lines       []control.Row
	first, rows int
}

// rowHeight is an option row's height: the font's line, and 2 points
// above and below.
func rowHeight(fonts box.Fonts, f style.Font) float32 {
	line := f.Size * 1.25
	if fonts != nil {
		m := fonts.Metrics(f)
		line = m.Ascent + m.Descent + m.LineGap
	}
	return line + 4
}

func (p *painter) open(e entry) {
	if m, ok := p.machine(e.b.Node).(*control.Select); ok && m.Open {
		p.popups = append(p.popups, popup{b: e.b, at: e.at, m: m, xf: p.s.Transform()})
	}
}

// place lays an open select's list out within the viewport.
func (p *painter) place(pp popup) menu {
	b, st := pp.b, shown(pp.b)
	lines := pp.m.Rows()
	n := len(lines)
	l := menu{border: b.Border.Top, row: rowHeight(p.fonts, st.Font), lines: lines, rows: n}
	w := b.Size.X
	if p.fonts != nil {
		for _, r := range lines {
			f, text := lineText(st, pp.m, r)
			w = max(w, p.fonts.Advance(f, text)+st.Spacing(text)+hanRoom(p.hanSpacing(st, f, text))+
				2*optionPad+2*l.border)
		}
	}
	vp := p.viewport
	if p.lists {
		vp = p.listsIn
	}
	h := float32(n)*l.row + 2*l.border
	if h > vp.H {
		l.rows = max(int((vp.H-2*l.border)/l.row), 1)
		h = float32(l.rows)*l.row + 2*l.border
	}
	y := pp.at.Y + b.Size.Y
	switch {
	case y+h <= vp.Y+vp.H:
	case pp.at.Y-h >= vp.Y:
		y = pp.at.Y - h
	default:
		y = vp.Y + vp.H - h
	}
	x := min(max(pp.at.X, vp.X), vp.X+vp.W-w)
	l.rect = geom.Rect{X: x, Y: y, W: w, H: h}
	l.first = max(0, min(l.line(pp.m.Highlight)-l.rows+1, n-l.rows))
	return l
}

// line is the line of option i, -1 for none.
func (l menu) line(i int) int {
	for k, r := range l.lines {
		if r.Option == i {
			return k
		}
	}
	return -1
}

// groupIndent is what Chrome sets before an option of an optgroup in
// its list, as it measures a select's width.
const groupIndent = "    "

// lineText is the font and the text of a line of the list: a group's
// label in bold, an option of a group indented.
func lineText(st *style.Computed, m *control.Select, r control.Row) (style.Font, string) {
	switch {
	case r.Option >= 0 && r.Group > 0:
		return st.Font, groupIndent + m.Options[r.Option].Text()
	case r.Option >= 0:
		return st.Font, m.Options[r.Option].Text()
	}
	f := st.Font
	f.Weight = bolder(f.Weight)
	return f, m.Groups[r.Group-1].Text()
}

// bolder is font-weight bolder of w.
func bolder(w float32) float32 {
	switch {
	case w < 350:
		return 400
	case w < 550:
		return 700
	case w < 900:
		return 900
	}
	return w
}

// rowRect is line i's row, if shown.
func (l menu) rowRect(i int) (geom.Rect, bool) {
	if i < l.first || i >= l.first+l.rows {
		return geom.Rect{}, false
	}
	return geom.Rect{
		X: l.rect.X + l.border, Y: l.rect.Y + l.border + float32(i-l.first)*l.row,
		W: l.rect.W - 2*l.border, H: l.row,
	}, true
}

func (l menu) inner() geom.Rect { return l.rect.Inset(l.border) }

// popups draws the open selects' lists, on top of everything.
func (p *painter) drawPopups() {
	saved, apart := p.s.Transform(), p.apart
	if p.listsApart && p.live != nil {
		p.apart, p.live.listsApart = true, true
	}
	for _, pp := range p.popups {
		p.s.SetTransform(pp.xf)
		if p.record(livePart{kind: liveList, list: pp}, geom.Rect{}) && !p.listsApart {
			p.drawList(pp, p.place(pp))
			p.endGap()
		}
	}
	p.apart = apart
	p.s.SetTransform(saved)
}

// DrawLists draws the open selects' lists a paint with Lists apart left
// out of set, kept inside within as Lists says, over what is drawn: the
// page round the Document in an iframe. It appends where each lies, in
// the space of the surface's drawing, to rects and returns them.
func DrawLists(s Surface, set *LiveSet, within geom.Rect, time float32, rects []geom.Rect,
	opts ...Option,
) []geom.Rect {
	if !set.listsApart {
		return rects
	}
	p := pooled[painter](&painters)
	defer painters.Put(p)
	*p = painter{s: s, base: s.Transform(), scratch: p.scratch[:0], chain: p.chain[:0]}
	for _, o := range opts {
		o(p)
	}
	p.time, p.timed = time, true
	p.lists, p.listsIn = true, within
	saved := s.Transform()
	p.use(nil)
	for _, part := range set.parts {
		if part.kind != liveList {
			continue
		}
		s.SetTransform(part.xf)
		l := p.place(part.list)
		p.drawList(part.list, l)
		rects = append(rects, boundsIn(part.xf, l.rect))
	}
	s.SetTransform(saved)
	return rects
}

func (p *painter) drawList(pp popup, l menu) {
	st := shown(pp.b)
	radius := radiiOf(st, geom.V2(l.rect.W, l.rect.H))
	p.s.Glow(l.rect, max(radius[0], radius[1], radius[2], radius[3]), 8, popupShadow)
	bg := st.Background
	if bg.A <= 0 {
		bg = color.White
	}
	p.fill(l.rect, radius, bg)
	p.border(l.rect, pp.b.Border, radius, st, 1)
	p.s.PushClip(l.inner())
	for i, r := range l.lines {
		row, ok := l.rowRect(i)
		if !ok {
			continue
		}
		ink := st.Color
		if r.Option >= 0 && r.Option == pp.m.Highlight {
			mark, text := highlight, highlightText
			if !st.AccentAuto {
				mark, text = st.AccentColor, st.Color
			}
			p.s.Rect(row, mark)
			ink = text
		}
		if r.Option >= 0 && pp.m.Options[r.Option].Disabled || r.Option < 0 && pp.m.Groups[r.Group-1].Disabled {
			ink = fade(ink, dimmed)
		}
		f, text := lineText(st, pp.m, r)
		look := st
		if r.Option < 0 {
			bold := *st
			bold.Font = f
			look = &bold
		}
		p.spaced(geom.V2(row.X+optionPad, row.Y+2), text, p.hanSpacing(st, f, text), look, smoothed([]twod.TextOption{
			twod.Typeface(p.faceOf(f)), twod.FontSize(f.Size), twod.Color(ink), twod.Tracking(st.LetterGap()),
		}, smoothingOf(st)))
	}
	p.s.PopClip()
}

// hanSpacing is the room text-spacing-trim and text-autospace give a row
// of an open select's list, text of style st in font f.
func (p *painter) hanSpacing(st *style.Computed, f style.Font, text string) []box.Spacing {
	if p.fonts == nil {
		return nil
	}
	row := *st
	row.Font = f
	return layout.LineSpacing(p.fonts, &row, text)
}

func hanRoom(spacing []box.Spacing) float32 {
	var w float32
	for _, s := range spacing {
		w += s.Space
	}
	return w
}

// Pick is what is under the pointer while a select is open: the select,
// the option row under it (-1 for none), and whether it is over the list
// or over the select itself. Neither is a click outside, which closes it.
type Pick struct {
	Select     *dom.Node
	Index      int
	List, Self bool
}

// PickOption is the open select's Pick at p, in the root's space; a Pick
// without a Select when none is open.
func PickOption(t *box.Tree, p geom.Vec2, opts ...Option) Pick {
	mode := cull.Now()
	pick := pickOption(t, p, opts, mode == cull.Off)
	if mode == cull.Check {
		if whole := pickOption(t, p, opts, true); whole != pick {
			cull.Differ(fmt.Sprintf("the option picked at %v: %+v, %+v walking every box", p, pick, whole))
		}
	}
	return pick
}

func pickOption(t *box.Tree, p geom.Vec2, opts []Option, whole bool) Pick {
	pt := scan(t, opts, p, false, whole)
	if pt == nil {
		return Pick{Index: -1}
	}
	defer scanners.Put(pt)
	if len(pt.popups) == 0 {
		return Pick{Index: -1}
	}
	pp := pt.popups[len(pt.popups)-1]
	l := pt.place(pp)
	pick := Pick{Select: pp.b.Node, Index: -1}
	pick.Self = geom.Rect{X: pp.at.X, Y: pp.at.Y, W: pp.b.Size.X, H: pp.b.Size.Y}.Contains(p)
	if l.rect.Contains(p) {
		pick.List, pick.Self = true, false
		for i, line := range l.lines {
			if r, ok := l.rowRect(i); ok && r.Contains(p) {
				pick.Index = line.Option
			}
		}
	}
	return pick
}

// scan walks t in paint order without drawing, keeping the open selects
// it meets and, hitting, the last element box holding at (found). Only
// boxes whose ink holds at are walked to, and not hitting, none but the
// loose ones: the walk then only finds the open selects. whole walks
// every box. The caller puts the painter back in scanners.
func scan(t *box.Tree, opts []Option, at geom.Vec2, hitting, whole bool) *painter {
	if t == nil || t.Root == nil {
		return nil
	}
	inked(t)
	pt := pooled[painter](&scanners)
	s := pt.probe
	if s == nil {
		s = &probe{}
	}
	*s = probe{xf: geom.Identity2D(), clips: s.clips[:0]}
	*pt = painter{s: s, probe: s, probing: true, point: at, pointless: !hitting, whole: whole,
		scratch: pt.scratch[:0], chain: pt.chain[:0], popups: pt.popups[:0], top: pt.top[:0], slabs: pt.slabs}
	for _, o := range opts {
		o(pt)
	}
	root := t.Root
	pt.start(root)
	pt.page(root)
	pt.use(nil)
	return pt
}

// scanners keeps scan's painters, each with its probe, between walks.
//
//nolint:gochecknoglobals // a sync.Pool every paint shares
var scanners = sync.Pool{New: func() any { return new(painter) }}
