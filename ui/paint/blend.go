package paint

import (
	"math"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/render"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/outline"
	"github.com/lantervale/ember/ui/internal/style"
)

// isolates reports whether an element of style st is the group its
// descendants' blend modes mix within: a stacking context, or isolated.
func isolates(st *style.Computed) bool { return startsContext(st) || st.Isolate }

// backdropRoot reports whether an element of style st is where the
// backdrop its descendants' backdrop filters read stops, as Chrome's
// Backdrop Root: filtered, translucent, clipped by a clip-path, masked,
// mixed, or filtering its own backdrop.
func backdropRoot(st *style.Computed) bool {
	return len(st.Filters) > 0 || st.Opacity < 1 || !st.ClipPath.None() || st.Masked() ||
		st.BlendMode != style.BlendNormal || len(st.BackdropFilters) > 0
}

// grouping is an element drawn as a picture of its own, whole: shown at
// alpha, mixed by mix with backdrop, what its group drew under it.
type grouping struct {
	on       Pictures
	whole    bool
	alpha    float32
	mix      twod.Mix
	backdrop *twod.Picture
	floor    *clip
	// filters is set while the backdrop is still to be shown filtered.
	filters bool
}

// beginGroup opens what e needs drawn apart before its filters do: for a
// blend mode or a backdrop filter, the picture of its group so far, the
// backdrop, ended and shown again at the start of the group's next
// picture, to be shown filtered in e's border box by showBackdrop; and one
// picture round e and all it holds when e mixes, is a group, or is
// translucent and drawn as a picture, which then shows at alpha, its
// content at 1. None when the surface draws no pictures or a scan is
// walking.
func (p *painter) beginGroup(e entry, st *style.Computed, alpha *float32) *grouping {
	on, ok := p.s.(Pictures)
	if !ok || p.probing || e.b.Kind != box.Element {
		return nil
	}
	mixes := st.BlendMode != style.BlendNormal && p.pictures > 0
	filters := len(st.BackdropFilters) > 0 && p.pictures > 0
	whole := mixes || e.b.Group && len(st.Filters) == 0 ||
		*alpha < 1 && (len(st.Filters) > 0 || e.b.Group || filters || e.b.Canvas)
	if !whole && !filters {
		return nil
	}
	g := &grouping{on: on, whole: whole, alpha: 1, floor: p.floor}
	p.use(nil)
	xf := p.s.Transform()
	defer p.s.SetTransform(xf)
	if mixes || filters {
		g.backdrop = on.EndPicture(0)
		on.BeginPicture()
		on.DrawPicture(g.backdrop)
	}
	if mixes {
		g.mix = twod.Mix(st.BlendMode)
	}
	if whole {
		on.BeginPicture()
		p.pictures++
		g.alpha, *alpha = *alpha, 1
		p.floor = e.clip
	}
	g.filters = filters
	return g
}

// showBackdrop shows the backdrop g ended, filtered by st's backdrop
// filters, in e's border box: after beginMask, so e's mask cuts it too.
func (p *painter) showBackdrop(g *grouping, e entry, st *style.Computed) {
	if g == nil || !g.filters {
		return
	}
	g.filters = false
	p.use(nil)
	p.backdropFilter(g.on, e, st, g.backdrop)
}

// endGroup shows the picture g drew e in, in the clip c.
func (p *painter) endGroup(g *grouping, c *clip) {
	if g == nil || !g.whole {
		return
	}
	p.use(nil)
	p.floor = g.floor
	p.pictures--
	pic := g.on.EndPicture(0)
	var fx []*render.Effect
	if g.alpha < 1 {
		fx = p.filterEffects(g.on, fx, style.Filter{Kind: style.OpacityFilter, Amount: g.alpha}, p.s.Transform())
	}
	p.use(c)
	if g.mix == twod.MixNormal {
		g.on.DrawPicture(pic, fx...)
		return
	}
	g.on.DrawPictureOver(pic, g.backdrop, g.mix, fx...)
}

// backdropFilter shows back, what e's group drew under it, through st's
// backdrop filters in e's border box, rounded as its background is: its
// edges mirrored where a blur reaches past the box, as Chrome's are.
func (p *painter) backdropFilter(on Pictures, e entry, st *style.Computed, back *twod.Picture) {
	if back == nil {
		return
	}
	xf := p.s.Transform()
	r := geom.Rect{X: e.at.X, Y: e.at.Y, W: e.b.Size.X, H: e.b.Size.Y}
	box := boundsIn(xf, r)
	at := back.Rect()
	at.W, at.H = max(at.W, 1e-3), max(at.H, 1e-3)
	edges := geom.Vec4{
		X: (box.X - at.X) / at.W, Y: (box.Y - at.Y) / at.H,
		Z: (box.X + box.W - at.X) / at.W, W: (box.Y + box.H - at.Y) / at.H,
	}
	if urlFilters(st.BackdropFilters) {
		if back = p.backdropGraphs(on, e, st.BackdropFilters, back, xf); back == nil {
			return
		}
	}
	var fx []*render.Effect
	for _, f := range st.BackdropFilters {
		if f.Kind == style.FilterURL {
			fx = nil
			break
		}
		if f.Kind == style.Blur {
			fx = mirroredBlur(on, fx, f.Amount*p.pixelsUnder(xf), edges)
			continue
		}
		fx = p.filterEffects(on, fx, f, xf)
	}
	shape := clipShape{path: outline.RoundedRect(r, ellipticRadii(st, e.b.Size)), boxed: true}
	shape.bounds, _ = shape.path.Bounds()
	p.use(p.newClip(clip{rect: shape.bounds, xf: xf, parent: e.clip, shape: &shape}))
	on.DrawPicture(back, fx...)
}

// ellipticRadii are the radii, across and down, of the corners of a border
// box of size, top-left, top-right, bottom-right, bottom-left: all
// scaled down together where two on one side would overlap, as CSS does.
func ellipticRadii(st *style.Computed, size geom.Vec2) [4]geom.Vec2 {
	var r [4]geom.Vec2
	for i, c := range [4]style.Size[style.Length]{
		st.Radius.TopLeft, st.Radius.TopRight, st.Radius.BottomRight, st.Radius.BottomLeft,
	} {
		w, okW := c.Width.Resolve(size.X)
		h, okH := c.Height.Resolve(size.Y)
		if okW && okH && w > 0 && h > 0 {
			r[i] = geom.V2(w, h)
		}
	}
	f := float32(1)
	for _, side := range [][3]float32{
		{size.X, r[0].X, r[1].X}, {size.X, r[3].X, r[2].X}, {size.Y, r[0].Y, r[3].Y}, {size.Y, r[1].Y, r[2].Y},
	} {
		if sum := side[1] + side[2]; sum > side[0] && sum > 0 {
			f = min(f, side[0]/sum)
		}
	}
	for i := range r {
		r[i] = r[i].Scale(f)
	}
	return r
}

// pixelsUnder is how many of the surface's pixels a point under xf takes.
func (p *painter) pixelsUnder(xf geom.Transform2D) float32 {
	scale := p.scale
	if scale <= 0 {
		scale = 1
	}
	return scale * float32(math.Sqrt(math.Abs(float64(xf.X.X*xf.Y.Y-xf.X.Y*xf.Y.X))))
}

// boundsIn is the rectangle round r under xf.
func boundsIn(xf geom.Transform2D, r geom.Rect) geom.Rect {
	lo, hi := xf.Xform(geom.V2(r.X, r.Y)), xf.Xform(geom.V2(r.X, r.Y))
	for _, c := range []geom.Vec2{{X: r.X + r.W, Y: r.Y}, {X: r.X, Y: r.Y + r.H}, {X: r.X + r.W, Y: r.Y + r.H}} {
		v := xf.Xform(c)
		lo, hi = geom.V2(min(lo.X, v.X), min(lo.Y, v.Y)), geom.V2(max(hi.X, v.X), max(hi.Y, v.Y))
	}
	return geom.Rect{X: lo.X, Y: lo.Y, W: hi.X - lo.X, H: hi.Y - lo.Y}
}

// mirroredBlurShader blurs as blurShader does, reading the picture only
// inside uEdges (left, top, right, bottom, 0–1), mirrored at them.
const mirroredBlurShader = `fn sampled(uv: vec2f) -> vec4f {
	let lo = uEdges.xy;
	let hi = uEdges.zw;
	var p = select(uv, 2.0 * lo - uv, uv < lo);
	p = select(p, 2.0 * hi - p, p > hi);
	return color(clamp(p, lo, hi));
}

fn effect(vUV: vec2f) -> vec4f {
	let r = min(i32(ceil(uSigma * 3.0)), 512);
	if (r < 1) {
		return color(vUV);
	}
	let step = uStep / uSize;
	var sum = vec4f(0.0);
	var weight = 0.0;
	for (var i = -r; i <= r; i++) {
		let x = f32(i);
		let k = exp(-x * x / (2.0 * uSigma * uSigma));
		sum += sampled(vUV + step * x) * k;
		weight += k;
	}
	return sum / weight;
}`

// mirroredBlur appends a Gaussian blur of deviation sigma pixels, across
// then down, reading inside edges, to fx.
func mirroredBlur(on Pictures, fx []*render.Effect, sigma float32, edges geom.Vec4) []*render.Effect {
	if sigma < 0.01 {
		return fx
	}
	for _, step := range []geom.Vec2{geom.V2(1, 0), geom.V2(0, 1)} {
		e := on.Effect(mirroredBlurShader)
		e.SetVec2("uStep", step)
		e.SetFloat("uSigma", sigma)
		e.SetVec4("uEdges", edges)
		fx = append(fx, e)
	}
	return fx
}

// layering is an element's background drawn as a picture of its own, its
// layers mixed with the colour and the layers under them by
// background-blend-mode, as Chrome isolates them.
type layering struct {
	on Pictures
	p  *painter
	xf geom.Transform2D
}

// beginLayers opens the picture st's background is drawn in when one of
// its layers mixes; none, drawing straight, otherwise.
func (p *painter) beginLayers(st *style.Computed) *layering {
	on, ok := p.s.(Pictures)
	if !ok || p.probing || !st.BackgroundBlended() {
		return nil
	}
	l := &layering{on: on, p: p, xf: p.s.Transform()}
	on.BeginPicture()
	p.s.SetTransform(l.xf)
	return l
}

// layer draws one layer by draw, mixed by m with what the background drew
// under it.
func (l *layering) layer(m style.BlendMode, draw func()) {
	if l == nil || m == style.BlendNormal {
		draw()
		return
	}
	back := l.on.EndPicture(0)
	l.on.BeginPicture()
	l.on.DrawPicture(back)
	l.on.BeginPicture()
	l.p.s.SetTransform(l.xf)
	draw()
	front := l.on.EndPicture(0)
	l.on.DrawPictureOver(front, back, twod.Mix(m))
	l.p.s.SetTransform(l.xf)
}

// end shows the background's picture.
func (l *layering) end() {
	if l == nil {
		return
	}
	l.on.DrawPicture(l.on.EndPicture(0))
	l.p.s.SetTransform(l.xf)
}
