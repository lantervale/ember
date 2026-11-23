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
// bar of display: none takes none.
func (r *restyler) scrollbars(c *style.Computed, n *dom.Node, rem float32) {
	if c.ScrollbarWidth == 0 || c.ScrollbarColored || !r.may(n, webkitScrollbar) {
		return
	}
	const always = partEnabled | partNoButton
	corner := partState(0)
	if c.Overflow.Width == style.Scroll && c.Overflow.Height == style.Scroll {
		corner = partCornerPresent
	}
	v := r.part(n, c, rem, webkitScrollbar, partVertical|always|corner)
	if v == nil {
		return
	}
	h := r.part(n, c, rem, webkitScrollbar, partHorizontal|always|corner)
	thickness := func(st *style.Computed, l style.Length) float32 {
		if st.Display == style.DisplayNone {
			return 0
		}
		if px, ok := l.Resolve(0); ok && l.Kind != style.Auto {
			return max(px, 0)
		}
		return barGutter
	}
	c.ScrollbarWidth, c.ScrollbarHeight = thickness(v, v.Size.Width), thickness(h, h.Size.Height)
	if c.ScrollbarWidth == 0 && c.ScrollbarHeight == 0 {
		c.ScrollbarNone = true
		return
	}
	bar := func(st *style.Computed, s partState) style.BarParts {
		s |= always | corner
		return style.BarParts{Bar: st,
			Track: r.part(n, c, rem, scrollbarTrack, s),
			Start: r.part(n, c, rem, scrollbarTrackPiece, s|partStart),
			End:   r.part(n, c, rem, scrollbarTrackPiece, s|partEnd),
			Thumb: r.part(n, c, rem, scrollbarThumb, s),
		}
	}
	c.Scrollbar = &style.ScrollbarParts{V: bar(v, partVertical), H: bar(h, partHorizontal),
		Corner: r.part(n, c, rem, scrollbarCorner, always|corner)}
}

// part is the style of n's scrollbar part name in the states state,
// inheriting from n's style el; nil when no rule sets any property of it.
func (r *restyler) part(n *dom.Node, el *style.Computed, rem float32, name string, state partState) *style.Computed {
	if !r.may(n, name) {
		return nil
	}
	r.pm.part = state
	casc := r.pm.cascadeFor(r.pm.matchedFor(n, r.sheets, nil, name), name, el.Dir)
	r.pm.part = 0
	if !slices.ContainsFunc(casc.longhands, func(d *Declaration) bool { return d != nil }) && len(casc.custom) == 0 {
		return nil
	}
	c := initial
	x := &ctx{c: &c, parent: el, em: el.Font.Size, rem: rem, env: r.env, overlay: r.overlay, node: n, self: true}
	x.cascaded(casc, false)
	return &c
}
