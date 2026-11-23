package paint

import (
	"math"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/style"
)

// barSize is the room a bar takes along its edge and barWidth how wide it
// draws, as the old ember/ui's scrollbars.
const (
	barSize  = 10
	barWidth = 6
)

//nolint:gochecknoglobals // fixed values every paint reads; Go has no constant colours, tables or shapes
var (
	thumbColor = color.RGBA(1, 1, 1, 0.25)
	// An overlay thumb lies on content of any colour: dark, with a light
	// edge a point out, as macOS draws its own.
	overlayThumb = color.RGBA(0, 0, 0, 0.5)
	overlayEdge  = color.RGBA(1, 1, 1, 0.35)
)

// Bars are where a scroll container's bars lie, in the root's space: the
// view they scroll, the tracks and thumbs of those showing, and the
// content's size.
type Bars struct {
	View                           geom.Rect
	VTrack, VThumb, HTrack, HThumb geom.Rect
	V, H                           bool
	Content                        geom.Vec2
	// Scroll is the offset the thumbs show, from Start, the least one: below
	// zero for a box that scrolls from its end.
	Scroll, Start geom.Vec2
	// Overlay bars lie over the content, not in a gutter beside it.
	Overlay bool
	// Parts are the parts its sheets style, drawn in VBar, HBar and Corner
	// in place of ui's own look; nil for that look.
	Parts              *style.ScrollbarParts
	VBar, HBar, Corner geom.Rect
	// VButtons and HButtons are the bars' styled buttons, in the order of
	// style.BarParts.Buttons, empty where none shows.
	VButtons, HButtons [4]geom.Rect
	// Resizer is the ::-webkit-resizer of a box the user may resize,
	// drawn in ResizerAt, the corner Chrome keeps for it.
	Resizer   *style.Computed
	ResizerAt geom.Rect
	// Thumb and Track are scrollbar-color's, Thumb nil for auto: the
	// thumbs drawn in it, the gutters filled with Track.
	Thumb *color.Color
	Track color.Color
}

// Clamp is off kept within what the bars scroll.
func (l Bars) Clamp(off geom.Vec2) geom.Vec2 {
	return box.ClampScroll(off, l.Start, l.Content, geom.V2(l.View.W, l.View.H))
}

// A thumb is sized and placed on the whole bar, as Chrome sizes the box of
// export's bordered thumb, then drawn inset by thumbInset at both ends, as
// that box's borders inset what Chrome paints; minThumb is the least box.
const (
	thumbInset = 4
	minThumb   = 16 + 2*thumbInset
)

// BarsOf is where b's bars lie, its border box's top-left at at: along its
// padding box's right and bottom edges, the vertical one along its left
// under direction rtl, as Chrome's, in the gutter layout keeps beside
// the scrollport, overflow: scroll always and auto when the content
// overflows that way; none for scrollbar-width: none, which still scrolls
// by the wheel. Overlay bars lie in the same place with no gutter under
// them, over the content's edge. A box the user may resize has its styled
// resizer in the corner.
func BarsOf(b *box.Box, at geom.Vec2) Bars {
	return barsOf(b, at).resizer(shown(b), b.Scrollbar.X, b.Scrollbar.Y)
}

func barsOf(b *box.Box, at geom.Vec2) Bars {
	view := padding(b, at)
	st := shown(b)
	l := Bars{View: view, Content: b.Content, Scroll: b.Scroll, Start: b.ScrollStart, Overlay: st.ScrollbarOverlay}
	o := st.Overflow
	l.V = !st.ScrollbarNone && (o.Height == style.Scroll || o.Height == style.OverflowAuto && b.Content.Y > view.H)
	l.H = !st.ScrollbarNone && (o.Width == style.Scroll || o.Width == style.OverflowAuto && b.Content.X > view.W)
	if st.Scrollbar != nil && !l.Overlay {
		return l.styled(b, st)
	}
	if st.ScrollbarColored {
		l.Thumb, l.Track = &st.ScrollbarThumb, st.ScrollbarTrack
		if !l.Overlay {
			l.VBar = geom.Rect{X: view.X + view.W - b.Scrollbar.X, Y: view.Y, W: b.Scrollbar.X, H: view.H - b.Scrollbar.Y}
			l.HBar = geom.Rect{X: view.X, Y: view.Y + view.H - b.Scrollbar.Y, W: view.W - b.Scrollbar.X, H: b.Scrollbar.Y}
			l.Corner = geom.Rect{X: l.VBar.X, Y: l.HBar.Y, W: b.Scrollbar.X, H: b.Scrollbar.Y}
			if st.Dir == style.RTL {
				l.VBar.X, l.Corner.X, l.HBar.X = view.X, view.X, view.X+b.Scrollbar.X
			}
		}
	}
	corner := func(on bool) float32 {
		if on {
			return barSize
		}
		return 0
	}
	rtl := st.Dir == style.RTL
	if l.V {
		l.VTrack = geom.Rect{X: view.X + view.W - barSize + 2, Y: view.Y + 4, W: barWidth, H: view.H - 8 - corner(l.H)}
		if rtl {
			l.VTrack.X = view.X + barSize - 2 - barWidth
		}
		if b.Content.Y > view.H {
			bar := view.H - corner(l.H)
			h := max(bar*view.H/b.Content.Y, minThumb)
			y := view.Y + (bar-h)*(b.Scroll.Y-b.ScrollStart.Y)/(b.Content.Y-view.H)
			l.VThumb = geom.Rect{X: l.VTrack.X, Y: y + thumbInset, W: barWidth, H: h - 2*thumbInset}
		}
	}
	if l.H {
		l.HTrack = geom.Rect{X: view.X + 4, Y: view.Y + view.H - barSize + 2, W: view.W - 8 - corner(l.V), H: barWidth}
		if rtl {
			l.HTrack.X += corner(l.V)
		}
		if b.Content.X > view.W {
			bar := view.W - corner(l.V)
			w := max(bar*view.W/b.Content.X, minThumb)
			x := l.HTrack.X - 4 + (bar-w)*(b.Scroll.X-b.ScrollStart.X)/(b.Content.X-view.W)
			l.HThumb = geom.Rect{X: x + thumbInset, Y: l.HTrack.Y, W: w - 2*thumbInset, H: barWidth}
		}
	}
	return l
}

// minStyledThumb is the shortest a styled thumb draws whose min-height
// (min-width on a horizontal bar) is auto, measured from Chrome 157.
const minStyledThumb = 26

// styled lays l out as Chrome lays out a scrollbar its sheets style: each
// bar as thick as its ::-webkit-scrollbar, along the padding box's edge
// less the corner, its buttons at its ends, its track what they leave
// less the track's margins along it, and the thumb sized by the share of
// the content in view, no shorter than its min-height, placed along the
// track by the offset; none where it doesn't fit the track. A box the
// user may resize keeps the corner beside one bar too.
func (l Bars) styled(b *box.Box, st *style.Computed) Bars {
	parts, view := st.Scrollbar, l.View
	w, h := st.ScrollbarWidth, st.ScrollbarHeight
	l.Parts = parts
	l.V, l.H = l.V && w > 0, l.H && h > 0
	port := geom.V2(view.W-b.Scrollbar.X, view.H-b.Scrollbar.Y)
	rtl := st.Dir == style.RTL
	cw, ch := w, h
	switch {
	case l.V && l.H:
	case l.V && st.Resizable:
		ch = w
	case l.H && st.Resizable:
		cw = h
	default:
		cw, ch = 0, 0
	}
	margin := func(m style.Length) float32 { v, _ := m.Resolve(0); return v }
	if l.V {
		l.VBar = geom.Rect{X: view.X + view.W - w, Y: view.Y, W: w, H: view.H - ch}
		if rtl {
			l.VBar.X = view.X
		}
		buttons, before, after, fits := buttonsAlong(parts.V.Buttons, l.VBar.Y, l.VBar.H,
			func(t *style.Computed) style.Length { return t.Size.Height },
			func(t *style.Computed) (style.Length, style.Length) { return t.MinSize.Height, t.MaxSize.Height })
		for i, r := range buttons {
			if r.H > 0 {
				l.VButtons[i] = geom.Rect{X: l.VBar.X, Y: r.Y, W: w, H: r.H}
			}
		}
		l.VTrack = l.VBar
		l.VTrack.Y, l.VTrack.H = l.VTrack.Y+before, max(l.VTrack.H-before-after, 0)
		if t := parts.V.Track; t != nil && fits {
			top, bottom := margin(t.Margin.Top), margin(t.Margin.Bottom)
			l.VTrack.Y, l.VTrack.H = l.VTrack.Y+top, max(l.VTrack.H-top-bottom, 0)
		}
		if at, n, ok := thumbAlong(l.VTrack.Y, l.VTrack.H, port.Y, b.Content.Y, b.Scroll.Y-b.ScrollStart.Y,
			minThumbOf(parts.V.Thumb, parts.V.Thumb != nil && parts.V.Thumb.MinSize.Height.Kind != style.Auto,
				func(t *style.Computed) style.Length { return t.MinSize.Height })); ok {
			l.VThumb = geom.Rect{X: l.VBar.X, Y: at, W: w, H: n}
		}
	}
	if l.H {
		l.HBar = geom.Rect{X: view.X, Y: view.Y + view.H - h, W: view.W - cw, H: h}
		if rtl {
			l.HBar.X += cw
		}
		buttons, before, after, fits := buttonsAlong(parts.H.Buttons, l.HBar.X, l.HBar.W,
			func(t *style.Computed) style.Length { return t.Size.Width },
			func(t *style.Computed) (style.Length, style.Length) { return t.MinSize.Width, t.MaxSize.Width })
		for i, r := range buttons {
			if r.H > 0 {
				l.HButtons[i] = geom.Rect{X: r.Y, Y: l.HBar.Y, W: r.H, H: h}
			}
		}
		l.HTrack = l.HBar
		l.HTrack.X, l.HTrack.W = l.HTrack.X+before, max(l.HTrack.W-before-after, 0)
		if t := parts.H.Track; t != nil && fits {
			left, right := margin(t.Margin.Left), margin(t.Margin.Right)
			l.HTrack.X, l.HTrack.W = l.HTrack.X+left, max(l.HTrack.W-left-right, 0)
		}
		if at, n, ok := thumbAlong(l.HTrack.X, l.HTrack.W, port.X, b.Content.X, b.Scroll.X-b.ScrollStart.X,
			minThumbOf(parts.H.Thumb, parts.H.Thumb != nil && parts.H.Thumb.MinSize.Width.Kind != style.Auto,
				func(t *style.Computed) style.Length { return t.MinSize.Width })); ok {
			l.HThumb = geom.Rect{X: at, Y: l.HBar.Y, W: n, H: h}
		}
	}
	if cw > 0 && ch > 0 {
		l.Corner = geom.Rect{X: view.X + view.W - cw, Y: view.Y + view.H - ch, W: cw, H: ch}
		if rtl {
			l.Corner.X = view.X
		}
	}
	return l
}

// resizer gives l the resizer of a box of style st the user may resize,
// its bars w and h thick, in the corner Chrome keeps for it: where the
// bars meet, square beside a lone bar, and as thick as ui's own bars
// with none.
func (l Bars) resizer(st *style.Computed, w, h float32) Bars {
	if !st.Resizable || st.Resizer == nil {
		return l
	}
	switch {
	case l.V && l.H:
	case l.V:
		h = w
	case l.H:
		w = h
	default:
		w, h = 15, 15
	}
	l.Resizer = st.Resizer
	l.ResizerAt = geom.Rect{X: l.View.X + l.View.W - w, Y: l.View.Y + l.View.H - h, W: w, H: h}
	if st.Dir == style.RTL {
		l.ResizerAt.X = l.View.X
	}
	return l
}

// autoButton is the length of a styled button whose height (width on a
// horizontal bar) is auto, measured from Chrome 157.
const autoButton = 16

// buttonsAlong places a bar's buttons, of the styles buttons, along it
// from start, length long, as Chrome places them, in whole points: their spans, as Y and
// H of each rect, the room they take at the bar's start and end, and
// whether they fit: when they are longer than the bar, none shows and
// the track is the whole bar, its margins left out. A button shows only
// with display: block, as on macOS, whose own bars have none.
func buttonsAlong(buttons [4]*style.Computed, start, length float32, size func(*style.Computed) style.Length,
	limits func(*style.Computed) (style.Length, style.Length)) ([4]geom.Rect, float32, float32, bool) {
	var n [4]float32
	for i, st := range buttons {
		if st == nil || st.Display != style.Block {
			continue
		}
		v := float32(autoButton)
		if l := size(st); l.Kind != style.Auto {
			v, _ = l.Resolve(length)
		}
		least, most := limits(st)
		if least.Kind != style.Auto {
			m, _ := least.Resolve(length)
			v = max(v, m)
		}
		if most.Kind != style.None && most.Kind != style.Auto {
			m, _ := most.Resolve(length)
			v = min(v, m)
		}
		n[i] = float32(int(max(v, 0)))
	}
	var out [4]geom.Rect
	before, after := n[0]+n[1], n[2]+n[3]
	if before+after > length {
		return out, 0, 0, false
	}
	out[0] = geom.Rect{Y: start, H: n[0]}
	out[1] = geom.Rect{Y: start + n[0], H: n[1]}
	out[3] = geom.Rect{Y: start + length - n[3], H: n[3]}
	out[2] = geom.Rect{Y: start + length - n[3] - n[2], H: n[2]}
	return out, before, after, true
}

// minThumbOf is a styled thumb's least length: its min-height or
// min-width when set, else minStyledThumb.
func minThumbOf(t *style.Computed, set bool, get func(*style.Computed) style.Length) float32 {
	if !set {
		return minStyledThumb
	}
	v, _ := get(t).Resolve(0)
	return max(v, 0)
}

// thumbAlong is where a styled thumb starts along a track from start, of
// length track, and its length, in whole points as Chrome's, no
// longer than the track; not ok when the content fits the port.
func thumbAlong(start, track, port, content, scroll, least float32) (float32, float32, bool) {
	if content <= port || track <= 0 {
		return 0, 0, false
	}
	n := min(max(round(track*port/content), least), track)
	return start + float32(int((track-n)*min(max(scroll, 0), content-port)/(content-port))), n, true
}

func round(v float32) float32 { return float32(math.Round(float64(v))) }

// bars paints l: its styled parts, else ui's own look, then its styled
// resizer.
func (p *painter) bars(l Bars, alpha float32) {
	if l.Parts != nil {
		p.styledBars(l, alpha)
	} else {
		l.paint(p.s, alpha)
	}
	p.part(l.Resizer, l.ResizerAt, alpha)
}

// styledBars paints l's parts as boxes of their styles: each bar's own
// box, its buttons, its track, the track's pieces before and after the
// thumb's middle, as Chrome splits the track, the thumb, then the
// corner, white where no rule styles it, as Chrome's.
func (p *painter) styledBars(l Bars, alpha float32) {
	if p.probing {
		return
	}
	bar := func(on bool, parts style.BarParts, whole, track, thumb geom.Rect, buttons [4]geom.Rect, vertical bool) {
		if !on {
			return
		}
		p.part(parts.Bar, whole, alpha)
		for i, r := range buttons {
			p.part(parts.Buttons[i], r, alpha)
		}
		p.part(parts.Track, track, alpha)
		before, after := track, track
		switch {
		case thumb.W <= 0 || thumb.H <= 0:
			before = geom.Rect{}
		case vertical:
			before.H = thumb.Y - track.Y + float32(int(thumb.H/2))
			after.Y, after.H = track.Y+before.H, track.H-before.H
		default:
			before.W = thumb.X - track.X + float32(int(thumb.W/2))
			after.X, after.W = track.X+before.W, track.W-before.W
		}
		p.part(parts.Start, before, alpha)
		p.part(parts.End, after, alpha)
		p.part(parts.Thumb, thumb, alpha)
	}
	bar(l.V, l.Parts.V, l.VBar, l.VTrack, l.VThumb, l.VButtons, true)
	bar(l.H, l.Parts.H, l.HBar, l.HTrack, l.HThumb, l.HButtons, false)
	if l.Parts.Corner == nil && l.Corner.W > 0 && l.Corner.H > 0 {
		p.s.Rect(l.Corner, fade(color.White, alpha))
		return
	}
	p.part(l.Parts.Corner, l.Corner, alpha)
}

// part paints a scrollbar's or a control's part of style st in r, as an
// element's box: its shadows, backgrounds and borders. A hit test meets no
// part: the pointer is on the element the part is of.
func (p *painter) part(st *style.Computed, r geom.Rect, alpha float32) {
	if p.probing || st == nil || r.W <= 0 || r.H <= 0 || st.Display == style.DisplayNone {
		return
	}
	b := &box.Box{Kind: box.Element, Style: st, Size: geom.V2(r.W, r.H),
		Border: box.Edges{Top: st.Border.Top, Right: st.Border.Right, Bottom: st.Border.Bottom, Left: st.Border.Left}}
	p.decorate(entry{b: b, at: geom.V2(r.X, r.Y)}, alpha)
}

func (l Bars) paint(s Surface, alpha float32) {
	ink := thumbColor
	if l.Thumb != nil {
		ink = *l.Thumb
		for _, r := range []geom.Rect{l.VBar, l.HBar, l.Corner} {
			if r.W > 0 && r.H > 0 {
				s.Rect(r, fade(l.Track, alpha))
			}
		}
	}
	thumb := func(r geom.Rect) {
		if !l.Overlay {
			s.RoundRect(r, 3, fade(ink, alpha))
			return
		}
		s.RoundRect(r.Inset(-1), 4, fade(overlayEdge, alpha))
		if l.Thumb != nil {
			s.RoundRect(r, 3, fade(ink, alpha))
			return
		}
		s.RoundRect(r, 3, fade(overlayThumb, alpha))
	}
	if l.VThumb.H > 0 {
		thumb(l.VThumb)
	}
	if l.HThumb.W > 0 {
		thumb(l.HThumb)
	}
}

// Over reports whether p is on a bar showing: a styled one's whole bar,
// ui's own track.
func (l Bars) Over(p geom.Vec2) bool {
	if l.Parts != nil {
		return l.V && l.VBar.Contains(p) || l.H && l.HBar.Contains(p)
	}
	return l.V && l.VTrack.Contains(p) || l.H && l.HTrack.Contains(p)
}

// PartAt is the part of l's styled bars at p, as Chrome tells it: a
// button, the thumb, the track before or after it, or the bar off them;
// none off the bars, and on ui's own bars, which no part styles.
func (l Bars) PartAt(p geom.Vec2) style.BarPart {
	if l.Parts == nil {
		return style.NoBarPart
	}
	part := func(track, thumb geom.Rect, buttons [4]geom.Rect, vertical bool) style.BarPart {
		for i, r := range buttons {
			if r.Contains(p) {
				return style.BarBackStart + style.BarPart(i)
			}
		}
		switch {
		case thumb.Contains(p):
			return style.BarThumb
		case !track.Contains(p):
			return style.BarBackground
		case vertical && p.Y < thumb.Y || !vertical && p.X < thumb.X || thumb.W <= 0 || thumb.H <= 0:
			return style.BarBackTrack
		}
		return style.BarForwardTrack
	}
	switch {
	case l.V && l.VBar.Contains(p):
		return part(l.VTrack, l.VThumb, l.VButtons, true)
	case l.H && l.HBar.Contains(p):
		return part(l.HTrack, l.HThumb, l.HButtons, false) | style.BarHorizontal
	}
	return style.NoBarPart
}

// Pointer is the mouse over a scroll container this frame.
type Pointer struct {
	At geom.Vec2
	// Pressed is the left button going down this frame, Down it held.
	Pressed, Down bool
	// Wheel is the wheel's lines this frame; Shift turns a vertical wheel
	// sideways.
	Wheel geom.Vec2
	Shift bool
	// Hover is the pointer over the container, nothing covering it.
	Hover bool
	// Page is how far a click on a track pages on each axis, the view's
	// length where zero, and Line how far a click on a button scrolls.
	Page geom.Vec2
	Line float32
}

// Scrolling is a scroll container's drag of its thumbs, kept between
// frames by its node.
type Scrolling struct {
	dragV, dragH bool
	grab         float32
}

// Input is the offset l's container scrolls to for p, step points a wheel
// line, as the old ember/ui's bars do: the wheel turns it, a thumb drags it, a
// click on a track pages it, and one on a styled button scrolls a line. It reports whether the bars took the mouse,
// so what is under them doesn't also take the click.
func (s *Scrolling) Input(l Bars, p Pointer, step float32) (geom.Vec2, bool) {
	offset, took := l.Scroll, false
	if w := p.Wheel; p.Hover && (w.X != 0 || w.Y != 0) {
		if p.Shift && w.X == 0 {
			w = geom.V2(w.Y, 0)
		}
		offset = offset.Add(w.Scale(step * 3))
	}
	if p.Pressed {
		switch {
		case l.V && l.VTrack.Contains(p.At):
			took = true
			if l.VThumb.Contains(p.At) {
				s.dragV, s.grab = true, p.At.Y-l.VThumb.Y
			} else if l.VThumb.H > 0 {
				offset.Y += page(p.At.Y < l.VThumb.Y, orView(p.Page.Y, l.View.H))
			}
		case l.H && l.HTrack.Contains(p.At):
			took = true
			if l.HThumb.Contains(p.At) {
				s.dragH, s.grab = true, p.At.X-l.HThumb.X
			} else if l.HThumb.W > 0 {
				offset.X += page(p.At.X < l.HThumb.X, orView(p.Page.X, l.View.W))
			}
		case l.Over(p.At):
			took = true
			by, part := p.Line, l.PartAt(p.At)
			switch part.Part() {
			case style.BarBackStart, style.BarBackEnd:
				by = -by
			case style.BarForwardStart, style.BarForwardEnd:
			default:
				by = 0
			}
			if part&style.BarHorizontal != 0 {
				offset.X += by
			} else {
				offset.Y += by
			}
		}
	}
	switch {
	case !p.Down:
		s.dragV, s.dragH = false, false
	case s.dragV:
		if travel := l.VTrack.H - l.VThumb.H; travel > 0 {
			offset.Y = l.Start.Y + (p.At.Y-s.grab-l.VTrack.Y)/travel*(l.Content.Y-l.View.H)
		}
		took = true
	case s.dragH:
		if travel := l.HTrack.W - l.HThumb.W; travel > 0 {
			offset.X = l.Start.X + (p.At.X-s.grab-l.HTrack.X)/travel*(l.Content.X-l.View.W)
		}
		took = true
	}
	return l.Clamp(offset), took
}

// Dragging reports whether a thumb is being dragged.
func (s *Scrolling) Dragging() bool { return s.dragV || s.dragH }

func orView(v, otherwise float32) float32 {
	if v == 0 {
		return otherwise
	}
	return v
}

func page(back bool, by float32) float32 {
	if back {
		return -by
	}
	return by
}
