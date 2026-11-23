package paint

import (
	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/style"
)

// luminanceShader turns a picture into the mask its colours make: their
// luminance times their alpha, in sRGB as Chrome takes it, as alpha.
const luminanceShader = `fn effect(vUV: vec2f) -> vec4f {
	let c = color(vUV);
	let l = clamp(dot(c.rgb, vec3f(0.2126, 0.7152, 0.0722)), 0.0, 1.0);
	return vec4f(l, l, l, l);
}`

// masking is a masked element's picture being drawn, with the clips it
// is drawn above, the clip it is shown in and the alpha it is shown at.
type masking struct {
	on      Pictures
	floor   *clip
	show    *clip
	opacity float32
}

// silhouetted reports whether b, of style st, paints its mask as its
// background colour's shape (decision 0093): its one layer is a picture
// whose alpha it shows, and the colour is all b paints, so the shape
// tinted is the element masked, with no picture of its own.
func silhouetted(b *box.Box, st *style.Computed) bool {
	if b.Kind != box.Element || !st.Masked() || st.MaskLayers() != 1 {
		return false
	}
	l := st.MaskLayer(0)
	inContent := func(c style.BackgroundBox) bool {
		return b.Padding != (box.Edges{}) && (c == style.UnderContent || c == style.FillBox)
	}
	return l.Image.Source != "" && l.Mode != style.MaskLuminance && l.Clip != style.NoClip && !inContent(l.Clip) &&
		!inContent(st.BackgroundLayer(0).Clip) && len(b.Children) == 0 && !marked(b) && !tracked(b) &&
		!outlined(b) && !paintsCells(st) && len(st.Shadows) == 0 && b.Border == (box.Edges{}) &&
		st.BorderImage.Source == "" && st.BorderImage.Gradient == nil && !st.HasBackgroundImage()
}

// maskedFill paints st's background colour through its mask's one
// picture over r, the border box with borders of edges: the picture's
// alpha, sized, placed and tiled as a background image is, in the colour
// (decision 0093). Nothing is painted until the picture is there.
func (p *painter) maskedFill(b *box.Box, r geom.Rect, edges box.Edges, radius radii, st *style.Computed,
	alpha float32) {
	l := st.MaskLayer(0)
	area, _ := p.maskBox(b, r, edges, radius, l.Origin)
	p.tiled(r, area, radius, &l.BackgroundLayer, st, alpha, fade(st.Background, alpha), true)
}

// maskBox is the box of b, whose border box is r with borders of edges
// and corners of radius, that a mask layer names, and its corners: an SVG
// shape's fill box is the content box, its others the border box, and
// no-clip whatever b and its content draw.
func (p *painter) maskBox(b *box.Box, r geom.Rect, edges box.Edges, radius radii, which style.BackgroundBox) (
	geom.Rect, radii) {
	switch which {
	case style.NoClip:
		ink := geom.Rect{X: b.Ink.X + r.X, Y: b.Ink.Y + r.Y, W: b.Ink.W, H: b.Ink.H}
		return union(r, ink), radii{}
	case style.FillBox:
		which = style.UnderContent
	case style.InitialBox, style.StrokeBox, style.ViewBox, style.UnderText:
		which = style.UnderBorder
	case style.UnderBorder, style.UnderPadding, style.UnderContent:
	}
	return backgroundBox(b, r, edges, radius, which)
}

// beginMask opens the picture a masked element and its content are drawn
// in, at 1, its alpha left for the picture: its clip-path's clip and
// those inside around them, or inside the clip of g, the picture of its
// group it is drawn in whole, which shows it in its clip-path. None for an
// element masked as a silhouette, on a surface that draws no pictures or
// in a scan, as masks change nothing hit.
func (p *painter) beginMask(e entry, st *style.Computed, around *clip, g *grouping, alpha *float32) *masking {
	on, ok := p.s.(Pictures)
	if !ok || p.probing || !st.Masked() || silhouetted(e.b, st) {
		return nil
	}
	m := &masking{on: on, floor: p.floor, show: around, opacity: *alpha}
	if g != nil && g.whole {
		m.show = e.clip
	}
	*alpha = 1
	p.use(nil)
	on.BeginPicture()
	p.pictures++
	p.floor = m.show
	return m
}

// endMask shows what m's picture holds through the element's mask, in
// its clip, at its alpha: the mask is a picture of its layers, drawn as
// backgrounds are, whose alpha cuts the element's.
func (p *painter) endMask(m *masking, e entry, st *style.Computed) {
	if m == nil {
		return
	}
	p.use(nil)
	p.floor = m.floor
	p.pictures--
	content := m.on.EndPicture(0)
	m.on.BeginPicture()
	p.maskFrom(m.on, e, st, 0)
	mask := m.on.EndPicture(0)
	p.use(m.show)
	m.on.PushClipPicture(mask)
	if m.opacity < 1 {
		m.on.DrawPicture(content, p.filterEffects(m.on, nil, style.Filter{Kind: style.OpacityFilter, Amount: m.opacity},
			p.s.Transform())...)
	} else {
		m.on.DrawPicture(content)
	}
	p.s.PopClip()
}

// maskFrom draws st's mask layers from i to the last, i on top, into the
// picture open: each joined to those under it by its mask-composite,
// the last's left aside and a layer of no picture left out, as Chrome
// does. Each layer is drawn into the picture it joins, or the one it is
// cut by into its own, so a picture always holds those it shows.
func (p *painter) maskFrom(on Pictures, e entry, st *style.Computed, i int) {
	l := st.MaskLayer(i)
	if i == st.MaskLayers()-1 {
		p.maskLayer(on, e, st, &l)
		return
	}
	if l.Image.Source == "" && l.Image.Gradient == nil {
		p.maskFrom(on, e, st, i+1)
		return
	}
	composite := l.Composite.Plain()
	switch composite {
	case style.CompositeAdd:
		p.maskFrom(on, e, st, i+1)
		p.maskLayer(on, e, st, &l)
	case style.CompositeDestinationOver:
		p.maskLayer(on, e, st, &l)
		p.maskFrom(on, e, st, i+1)
	case style.CompositeCopy:
		p.maskLayer(on, e, st, &l)
	case style.CompositeClear:
	case style.CompositeSubtract, style.CompositeIntersect, style.CompositeExclude, style.CompositeSourceOver,
		style.CompositeSourceIn, style.CompositeSourceOut, style.CompositeXor, style.CompositeSourceAtop,
		style.CompositeDestinationIn, style.CompositeDestinationOut, style.CompositeDestinationAtop,
		style.CompositePlusLighter:
		on.BeginPicture()
		p.maskFrom(on, e, st, i+1)
		p.maskJoin(on, e, st, &l, composite, on.EndPicture(0))
	}
}

// maskJoin draws l, a layer of st's mask, joined by composite to under,
// the picture of the layers under it.
func (p *painter) maskJoin(on Pictures, e entry, st *style.Computed, l *style.MaskLayer, composite style.MaskComposite,
	under *twod.Picture) {
	layer := func() *twod.Picture {
		on.BeginPicture()
		p.maskLayer(on, e, st, l)
		return on.EndPicture(0)
	}
	switch composite {
	case style.CompositeIntersect:
		on.PushClipPicture(under)
		p.maskLayer(on, e, st, l)
		p.s.PopClip()
	case style.CompositeSubtract:
		on.PushClipPicture(under, twod.Inverted())
		p.maskLayer(on, e, st, l)
		p.s.PopClip()
	case style.CompositeExclude:
		on.BeginPicture()
		p.maskLayer(on, e, st, l)
		over := on.EndPicture(0)
		on.PushClipPicture(under, twod.Inverted())
		on.DrawPicture(over)
		p.s.PopClip()
		on.PushClipPicture(over, twod.Inverted())
		on.SumPicture(under)
		p.s.PopClip()
	case style.CompositeSourceAtop:
		over := layer()
		on.PushClipPicture(over, twod.Inverted())
		on.DrawPicture(under)
		p.s.PopClip()
		on.PushClipPicture(under)
		on.SumPicture(over)
		p.s.PopClip()
	case style.CompositeDestinationIn, style.CompositeDestinationOut:
		over := layer()
		if composite == style.CompositeDestinationIn {
			on.PushClipPicture(over)
		} else {
			on.PushClipPicture(over, twod.Inverted())
		}
		on.DrawPicture(under)
		p.s.PopClip()
	case style.CompositeDestinationAtop:
		over := layer()
		on.PushClipPicture(over)
		on.DrawPicture(under)
		p.s.PopClip()
		on.PushClipPicture(under, twod.Inverted())
		on.SumPicture(over)
		p.s.PopClip()
	case style.CompositePlusLighter:
		on.DrawPicture(layer())
		on.SumPicture(under)
	case style.CompositeAdd, style.CompositeSourceOver, style.CompositeSourceIn, style.CompositeSourceOut,
		style.CompositeXor, style.CompositeClear, style.CompositeCopy, style.CompositeDestinationOver:
	}
}

// maskLayer draws l, a layer of st's mask, on each piece of e's box, as a
// background layer is drawn in the boxes its mask-origin and mask-clip
// name: in its own colours, its luminance made alpha through a picture
// of its own when its mode says so.
func (p *painter) maskLayer(on Pictures, e entry, st *style.Computed, l *style.MaskLayer) {
	if l.Image.Source == "" && l.Image.Gradient == nil {
		return
	}
	lum := l.Mode == style.MaskLuminance
	if lum {
		on.BeginPicture()
	}
	b := e.b
	ps := p.pieces(b, e.at)
	radius := radiiOf(st, framed(b))
	if len(ps) > 1 {
		radius = radii{}
	}
	total, laid := float32(0), float32(0)
	if len(ps) > 1 && !b.Broken {
		for _, pc := range ps {
			total += pc.rect.W
		}
	}
	for _, pc := range ps {
		r := p.snap(pc.rect)
		strip := r
		if total > 0 {
			strip = geom.Rect{X: r.X - laid, Y: r.Y, W: total, H: r.H}
			laid += pc.rect.W
		}
		view, rounding := p.maskBox(b, r, pc.edges, radius, l.Clip)
		area, _ := p.maskBox(b, strip, pc.edges, radius, l.Origin)
		p.tiled(view, area, rounding, &l.BackgroundLayer, st, 1, color.White, false)
	}
	if lum {
		on.DrawPicture(on.EndPicture(0), on.Effect(luminanceShader))
	}
}
