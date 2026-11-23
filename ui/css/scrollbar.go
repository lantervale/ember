package css

import (
	"slices"

	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

// barGutter is the room ui's own bars take beside a scroll container's
// content, and a styled bar whose thickness is auto.
const barGutter = 15

// scrollbars gives the scroll container n, of style c, the parts of its
// bars its sheets style, as Chrome does once a rule styles its
// ::-webkit-scrollbar and neither scrollbar-color nor scrollbar-width
// asks for the system's bars: their thicknesses are its gutters, and a
// bar of display: none takes none. It also gives a box the user may
// resize its ::-webkit-resizer, and the root and body, of parent's
// style, the viewport's bars as their rules style them.
func (r *restyler) scrollbars(c *style.Computed, n *dom.Node, rem float32, viewport bool, parent *style.Computed) {
	data := dom.Data(n)
	own := data.BarsOwn
	if c.ScrollbarWidth > 0 && !c.ScrollbarColored && !c.ScrollbarThin && r.may(n, webkitScrollbar) {
		own = r.part(n, c, rem, webkitScrollbar, 0)
	}
	whole := r.restylesBars(data, c) || !own.Equal(data.BarsOwn)
	data.BarsOwn = own
	if c.Resizable() {
		c.Resizer = r.part(n, c, rem, webkitResizer, 0)
		if !whole && data.Style != nil {
			c.Resizer = data.Style.Resizer
		}
	}
	if viewport {
		r.viewportBars(c, n, rem, parent)
	}
	if c.ScrollbarWidth == 0 || c.ScrollbarColored || c.ScrollbarThin || !r.may(n, webkitScrollbar) {
		data.BarsShown = nil
		return
	}
	p := r.barParts(c, c, n, rem, data.BarHover, data.BarPressed)
	if !whole {
		p = data.BarsStale.Kept(data.BarsShown, p)
	}
	data.BarsShown, data.BarsStale, data.BarsInactive = p, style.BarStale{}, r.env.inactive
	if p != nil {
		c.Scrollbar, c.ScrollbarWidth, c.ScrollbarHeight = p.Parts, p.Width, p.Height
		c.ScrollbarNone = p.Parts == nil
	}
}

// restylesBars reports whether the parts of the bars of n, whose pipeline
// is data, restyle whole as its style becomes c: on the first, once its
// own style changes, the parts aside, or once the window's focus does,
// as Chrome's (decision 0207); its ::-webkit-scrollbar in no part's state is the
// caller's.
func (r *restyler) restylesBars(data *dom.Pipeline, c *style.Computed) bool {
	old := data.Style
	if old == nil || data.BarsShown == nil || data.BarsInactive != r.env.inactive {
		return true
	}
	was := *old
	was.Scrollbar, was.ScrollbarWidth, was.ScrollbarHeight = c.Scrollbar, c.ScrollbarWidth, c.ScrollbarHeight
	was.ScrollbarNone, was.ViewportBars, was.Resizer = c.ScrollbarNone, c.ViewportBars, c.Resizer
	return !was.Equal(c)
}

// viewportBars gives the root or body n, of style c under parent's, the
// viewport's bars as n's rules style them, while the viewport shows
// bars of its own: the hovered and pressed parts are kept on the root,
// the viewport's node.
func (r *restyler) viewportBars(c *style.Computed, n *dom.Node, rem float32, parent *style.Computed) {
	view, root := c.Viewport, n
	if p := n.Parent(); p != nil {
		root = p
		if view == nil {
			view = parent.Viewport
		}
	}
	if view == nil || view.ScrollbarWidth == 0 || view.ScrollbarColored || view.ScrollbarThin ||
		!r.may(n, webkitScrollbar) {
		return
	}
	data := dom.Data(root)
	c.ViewportBars = r.barParts(view, c, n, rem, data.BarHover, data.BarPressed)
}

// barParts styles the bars of a scroller of style box from the rules for
// n, of style el, the element their parts are of, with hover the part
// under the pointer and pressed the one held: nil when no rule styles
// n's ::-webkit-scrollbar, Parts nil when both bars take no room.
func (r *restyler) barParts(box, el *style.Computed, n *dom.Node, rem float32,
	hover, pressed style.BarPart) *style.StyledBars {
	corner := partState(0)
	if box.Overflow.Width == style.Scroll && box.Overflow.Height == style.Scroll ||
		box.Resizable() && (box.Overflow.Width == style.Scroll || box.Overflow.Height == style.Scroll) {
		corner = partCornerPresent
	}
	held := func(horizontal bool, part style.BarPart) partState {
		var s partState
		if partHeld(hover, horizontal, part) {
			s |= partHover
		}
		if partHeld(pressed, horizontal, part) {
			s |= partActive
		}
		return s
	}
	const always = partEnabled
	orient := func(horizontal bool) partState {
		if horizontal {
			return partHorizontal | always | corner
		}
		return partVertical | always | corner
	}
	v := r.part(n, el, rem, webkitScrollbar, orient(false)|held(false, style.BarBackground))
	if v == nil {
		return nil
	}
	h := r.part(n, el, rem, webkitScrollbar, orient(true)|held(true, style.BarBackground))
	thickness := func(st *style.Computed, l style.Length) float32 {
		if st.Display == style.DisplayNone {
			return 0
		}
		if px, ok := l.Resolve(0); ok && l.Kind != style.Auto {
			return max(px, 0)
		}
		return barGutter
	}
	out := &style.StyledBars{Width: thickness(v, v.Size.Width), Height: thickness(h, h.Size.Height)}
	if out.Width == 0 && out.Height == 0 {
		return out
	}
	bar := func(st *style.Computed, horizontal bool) style.BarParts {
		s := orient(horizontal)
		part := func(name string, p style.BarPart, more partState) *style.Computed {
			return r.part(n, el, rem, name, s|more|held(horizontal, p))
		}
		b := style.BarParts{Bar: st,
			Track: part(scrollbarTrack, barTrack, 0),
			Start: part(scrollbarTrackPiece, style.BarBackTrack, partStart|partDecrement|partNoButton),
			End:   part(scrollbarTrackPiece, style.BarForwardTrack, partEnd|partIncrement|partNoButton),
			Thumb: part(scrollbarThumb, style.BarThumb, 0),
		}
		for i, at := range [4]partState{partStart | partDecrement, partStart | partIncrement,
			partEnd | partDecrement, partEnd | partIncrement} {
			b.Buttons[i] = part(scrollbarButton, style.BarBackStart+style.BarPart(i), at)
		}
		return b
	}
	out.Parts = &style.ScrollbarParts{V: bar(v, false), H: bar(h, true),
		Corner: r.part(n, el, rem, scrollbarCorner, 0)}
	return out
}

// barTrack stands for ::-webkit-scrollbar-track among the parts, which
// the pointer is never on alone: it is on a piece or the thumb.
const barTrack = style.BarHorizontal - 1

// partHeld reports whether the part of a bar, horizontal or not, is in the
// state of at, the part the pointer is on or presses, as Chrome reads
// :hover and :active on a scrollbar's parts: the bar while any part of
// it is, the track while a piece or the thumb is, another part while it
// is itself.
func partHeld(at style.BarPart, horizontal bool, part style.BarPart) bool {
	if at == style.NoBarPart || (at&style.BarHorizontal != 0) != horizontal {
		return false
	}
	on := at.Part()
	if part == barTrack {
		return on == style.BarBackTrack || on == style.BarThumb || on == style.BarForwardTrack
	}
	return part == style.BarBackground || on == part
}

// part is the style of n's scrollbar part name in the states state,
// inheriting from n's style el; nil when no rule sets any property of it.
func (r *restyler) part(n *dom.Node, el *style.Computed, rem float32, name string, state partState) *style.Computed {
	if !r.may(n, name) {
		return nil
	}
	r.pm.part = state
	casc := r.pm.cascadeFor(r.pm.matchedFor(n, r.sheets, nil, name), name, el)
	r.pm.part = 0
	if !slices.ContainsFunc(casc.longhands, func(d *Declaration) bool { return d != nil }) && len(casc.custom) == 0 {
		return nil
	}
	c := initial
	x := &ctx{c: &c, parent: el, em: el.Font.Size, rem: rem, env: r.env, overlay: r.overlay, node: n, self: true}
	x.cascaded(casc, false)
	return &c
}
