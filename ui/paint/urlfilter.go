package paint

import (
	"math"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/render"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/ui/internal/fx"
	"github.com/lantervale/ember/ui/internal/style"
)

// The shaders of a filter element's primitives, on premultiplied colours.
const (
	// toLinearShader and toSRGBShader take colours, unpremultiplied, to
	// linear light and back, for a primitive working in linearRGB.
	toLinearShader = `fn effect(vUV: vec2f) -> vec4f {
	let c = color(vUV);
	if (c.a <= 0.0) {
		return vec4f(0.0);
	}
	let s = c.rgb / c.a;
	let l = select(pow((s + 0.055) / 1.055, vec3f(2.4)), s / 12.92, s <= vec3f(0.04045));
	return vec4f(l * c.a, c.a);
}`
	toSRGBShader = `fn effect(vUV: vec2f) -> vec4f {
	let c = color(vUV);
	if (c.a <= 0.0) {
		return vec4f(0.0);
	}
	let l = clamp(c.rgb / c.a, vec3f(0.0), vec3f(1.0));
	let s = select(1.055 * pow(l, vec3f(1.0 / 2.4)) - 0.055, l * 12.92, l <= vec3f(0.0031308));
	return vec4f(s * c.a, c.a);
}`
	// offsetShader moves the picture by uOffset pixels.
	offsetShader = `fn effect(vUV: vec2f) -> vec4f {
	let uv = vUV - uOffset / uSize;
	if (any(uv < vec2f(0.0)) || any(uv > vec2f(1.0))) {
		return vec4f(0.0);
	}
	return color(uv);
}`
	// alphaShader is the picture's alpha in black.
	alphaShader = `fn effect(vUV: vec2f) -> vec4f {
	return vec4f(0.0, 0.0, 0.0, color(vUV).a);
}`
	// scaleShader multiplies the premultiplied colour by uK, up to one.
	scaleShader = `fn effect(vUV: vec2f) -> vec4f {
	return clamp(color(vUV) * uK, vec4f(0.0), vec4f(1.0));
}`
	// morphShader takes the least (or, uDilate set, the most) of each
	// channel within uRadius pixels along uStep, clear past the picture.
	morphShader = `fn effect(vUV: vec2f) -> vec4f {
	let r = min(i32(uRadius), 256);
	let step = uStep / uSize;
	var out = color(vUV);
	for (var i = -r; i <= r; i++) {
		let uv = vUV + step * f32(i);
		var c = vec4f(0.0);
		if (all(uv >= vec2f(0.0)) && all(uv <= vec2f(1.0))) {
			c = color(uv);
		}
		if (uDilate > 0.5) {
			out = max(out, c);
		} else {
			out = min(out, c);
		}
	}
	return out;
}`
	// transferShader maps each channel, unpremultiplied, through its
	// function: uKind per channel 0 identity, 1 table, 2 discrete, 3
	// linear, 4 gamma; uN the table's length, its values in uTable, one
	// vec4 an entry; uA slope or amplitude, uB intercept or exponent, uC
	// the gamma's offset.
	transferShader = `fn channel(k: i32, c: f32, n: i32, a: f32, b: f32, o: f32, t: i32) -> f32 {
	switch k {
		case 1: {
			if (n < 2) { return c; }
			let x = clamp(c, 0.0, 1.0) * f32(n - 1);
			let i = min(i32(floor(x)), n - 2);
			return mix(uTable[i][t], uTable[i + 1][t], x - f32(i));
		}
		case 2: {
			if (n < 1) { return c; }
			let i = min(i32(floor(clamp(c, 0.0, 1.0) * f32(n))), n - 1);
			return uTable[i][t];
		}
		case 3: { return a * c + b; }
		case 4: { return a * pow(c, b) + o; }
		default: { return c; }
	}
}

fn effect(vUV: vec2f) -> vec4f {
	let c = color(vUV);
	var u = vec4f(0.0);
	if (c.a > 0.0) {
		u = vec4f(c.rgb / c.a, c.a);
	}
	var o = vec4f(0.0);
	for (var t = 0; t < 4; t++) {
		o[t] = channel(i32(uKind[t]), u[t], i32(uN[t]), uA[t], uB[t], uC[t], t);
	}
	o = clamp(o, vec4f(0.0), vec4f(1.0));
	return vec4f(o.rgb * o.a, o.a);
}`
)

// maxTable is the most entries of a transfer table the shader reads; a
// longer table is sampled at as many points.
const maxTable = 32

// urlFilters reports whether filters name a filter element.
func urlFilters(filters []style.Filter) bool {
	for _, f := range filters {
		if f.Kind == style.FilterURL {
			return true
		}
	}
	return false
}

// graphs are the filter elements st's filters name, read for e's box;
// ok is false when one names none, and the whole list is then left out,
// as Chrome leaves it.
func (p *painter) graphs(e entry, st *style.Computed) ([]*fx.Graph, bool) {
	out := make([]*fx.Graph, len(st.Filters))
	for i, f := range st.Filters {
		if f.Kind != style.FilterURL {
			continue
		}
		if p.graph == nil {
			return nil, false
		}
		if out[i] = p.graph(f.Ref, e.b.Size); out[i] == nil {
			return nil, false
		}
	}
	return out, true
}

// graphReach is how far, in the element's points, what the filters from
// i on draw may reach past the element of size: each function's reach,
// and each filter element's region past the box.
func graphReach(filters []style.Filter, graphs []*fx.Graph, i int, size geom.Vec2) float32 {
	var d float32
	for k := i; k < len(filters); k++ {
		if g := graphs[k]; g != nil {
			r := g.Region
			d += max(0, -r.X, -r.Y, r.X+r.W-size.X, r.Y+r.H-size.Y)
			continue
		}
		d += filterReach(filters[k : k+1])
	}
	return d
}

// endGraphs shows what f's picture holds through its filters in order,
// a filter element's run primitive by primitive, the last result in the
// clip c.
func (p *painter) endGraphs(f *filtering, c *clip) {
	xf := p.s.Transform()
	scale := float32(math.Sqrt(math.Abs(float64(xf.X.X*xf.Y.Y - xf.X.Y*xf.Y.X))))
	pic := f.on.EndPicture(graphReach(f.filters, f.graphs, 0, f.size) * scale)
	for i, fl := range f.filters {
		next := graphReach(f.filters, f.graphs, i+1, f.size) * scale
		if g := f.graphs[i]; g != nil {
			pic = p.runGraph(f, pic, g, xf, next)
			continue
		}
		f.on.BeginPicture()
		if fl.Kind == style.DropShadow {
			f.on.DrawPicture(pic, p.filterEffects(f.on, nil, fl, xf)...)
			f.on.DrawPicture(pic)
		} else {
			f.on.DrawPicture(pic, p.filterEffects(f.on, nil, fl, xf)...)
		}
		pic = f.on.EndPicture(next)
	}
	p.use(c)
	if pic != nil {
		f.on.DrawPicture(pic)
	}
}

// backdropGraphs is back, what lies under e, through filters, a list
// naming a filter element: each filter in turn, nil when one names none.
func (p *painter) backdropGraphs(on Pictures, e entry, filters []style.Filter, back *twod.Picture,
	xf geom.Transform2D) *twod.Picture {
	f := &filtering{on: on, filters: filters, at: e.at, size: e.b.Size}
	for _, fl := range filters {
		if fl.Kind != style.FilterURL {
			on.BeginPicture()
			on.DrawPicture(back, p.filterEffects(on, nil, fl, xf)...)
			back = on.EndPicture(0)
			continue
		}
		if p.graph == nil {
			return nil
		}
		g := p.graph(fl.Ref, e.b.Size)
		if g == nil {
			return nil
		}
		back = p.runGraph(f, back, g, xf, 0)
	}
	return back
}

// runGraph is src, the picture of f's element, through g: each
// primitive's result a picture cut to its region, the last one's cut to
// the filter's, with margin points round it for the filters after.
func (p *painter) runGraph(f *filtering, src *twod.Picture, g *fx.Graph, xf geom.Transform2D,
	margin float32) *twod.Picture {
	on := f.on
	at := func(r geom.Rect) geom.Rect { return geom.Rect{X: f.at.X + r.X, Y: f.at.Y + r.Y, W: r.W, H: r.H} }
	results := make([]*twod.Picture, len(g.Primitives))
	// A picture is first shown in its own pass, through the effects it
	// is shown through then, and a clip reads that pass: those a
	// composite cuts by are first shown plain, in a picture of no use.
	clips := make(map[fx.Input]bool)
	for _, pr := range g.Primitives {
		if pr.Kind == fx.Composite && pr.Operator >= fx.In && pr.Operator <= fx.Xor {
			for _, in := range pr.In {
				clips[in] = true
			}
		}
	}
	plain := func(in fx.Input, pic *twod.Picture) {
		if pic != nil && clips[in] {
			on.BeginPicture()
			on.DrawPicture(pic)
			on.EndPicture(0)
		}
	}
	plain(fx.SourceGraphic, src)
	var alpha *twod.Picture
	input := func(in fx.Input) *twod.Picture {
		switch {
		case in == fx.SourceGraphic:
			return src
		case in == fx.SourceAlpha:
			if alpha == nil && src != nil {
				on.BeginPicture()
				on.DrawPicture(src, on.Effect(alphaShader))
				alpha = on.EndPicture(0)
				plain(fx.SourceAlpha, alpha)
			}
			return alpha
		case in >= 0 && int(in) < len(results):
			return results[in]
		}
		return nil
	}
	for i := range g.Primitives {
		pr := &g.Primitives[i]
		if pr.Region.W <= 0 || pr.Region.H <= 0 || pr.Kind == fx.Clear || pr.Kind == fx.Tile {
			continue
		}
		ins := make([]*twod.Picture, len(pr.In))
		for k, in := range pr.In {
			ins[k] = input(in)
		}
		on.BeginPicture()
		p.s.PushClip(at(pr.Region))
		p.primitive(on, pr, ins, at(pr.Region), xf)
		p.s.PopClip()
		results[i] = on.EndPicture(0)
		plain(fx.Input(i), results[i])
	}
	on.BeginPicture()
	if n := len(results); n > 0 && results[n-1] != nil {
		p.s.PushClip(at(g.Region))
		on.DrawPicture(results[n-1])
		p.s.PopClip()
	}
	return on.EndPicture(margin)
}

// primitive draws pr of ins, its inputs, into the picture open, over r,
// its region where the element's transform is xf.
func (p *painter) primitive(on Pictures, pr *fx.Primitive, ins []*twod.Picture, r geom.Rect, xf geom.Transform2D) {
	in := func(k int) *twod.Picture {
		if k < len(ins) {
			return ins[k]
		}
		return nil
	}
	draw := func(pic *twod.Picture, fx ...*render.Effect) {
		if pic != nil {
			on.DrawPicture(pic, fx...)
		}
	}
	linear := func(fx []*render.Effect) []*render.Effect {
		if !pr.LinearRGB {
			return fx
		}
		return append(append([]*render.Effect{on.Effect(toLinearShader)}, fx...), on.Effect(toSRGBShader))
	}
	pixels := p.pixelsOf(xf)
	switch pr.Kind {
	case fx.GaussianBlur:
		draw(in(0), linear(blurXY(on, nil, pr.StdDev.Scale(pixels)))...)
	case fx.Offset:
		draw(in(0), p.offsetEffect(on, pr.Offset, xf))
	case fx.Flood:
		p.s.Rect(r, pr.Color)
	case fx.ColorMatrix:
		e := on.Effect(matrixShader)
		m := pr.Matrix
		for i, name := range []string{"uR", "uG", "uB", "uA"} {
			e.SetVec4(name, geom.Vec4{X: m[5*i], Y: m[5*i+1], Z: m[5*i+2], W: m[5*i+3]})
		}
		e.SetVec4("uO", geom.Vec4{X: m[4], Y: m[9], Z: m[14], W: m[19]})
		draw(in(0), linear([]*render.Effect{e})...)
	case fx.ComponentTransfer:
		draw(in(0), linear([]*render.Effect{transferEffect(on, pr.Transfer)})...)
	case fx.Composite:
		p.composite(on, pr, in(0), in(1))
	case fx.Merge:
		for _, pic := range ins {
			draw(pic)
		}
	case fx.Blend:
		blend(on, pr, in(0), in(1))
	case fx.DropShadow:
		if src := in(0); src != nil {
			on.DrawPicture(src, p.dropShadow(on, pr, xf)...)
			on.DrawPicture(src)
		}
	case fx.Morphology:
		draw(in(0), morphology(on, pr, pixels)...)
	case fx.Clear, fx.Tile:
	}
}

// blend draws a over b through pr's mode, either nil for nothing.
func blend(on Pictures, pr *fx.Primitive, a, b *twod.Picture) {
	switch {
	case a == nil && b == nil:
	case a == nil:
		on.DrawPicture(b)
	case b == nil:
		on.DrawPicture(a)
	case pr.LinearRGB:
		a, b = linearOf(on, a), linearOf(on, b)
		on.BeginPicture()
		on.DrawPicture(b)
		on.DrawPictureOver(a, b, twod.Mix(pr.Mode))
		on.DrawPicture(on.EndPicture(0), on.Effect(toSRGBShader))
	default:
		on.DrawPicture(b)
		on.DrawPictureOver(a, b, twod.Mix(pr.Mode))
	}
}

// dropShadow are the effects that draw pr's shadow of a picture under
// an element's transform xf.
func (p *painter) dropShadow(on Pictures, pr *fx.Primitive, xf geom.Transform2D) []*render.Effect {
	e := on.Effect(shadowShader)
	e.SetVec2("uOffset", p.moved(pr.Offset, xf))
	c := pr.Color
	rgb := c.SRGB()
	if pr.LinearRGB {
		rgb = [3]float32{c.R, c.G, c.B}
	}
	e.SetVec4("uShadow", geom.Vec4{X: rgb[0] * c.A, Y: rgb[1] * c.A, Z: rgb[2] * c.A, W: c.A})
	fxs := blurXY(on, []*render.Effect{e}, pr.StdDev.Scale(p.pixelsOf(xf)))
	if pr.LinearRGB {
		fxs = append(fxs, on.Effect(toSRGBShader))
	}
	return fxs
}

// morphology are the effects that thin or thicken a picture by pr's
// radius, pixels to a point.
func morphology(on Pictures, pr *fx.Primitive, pixels float32) []*render.Effect {
	var fxs []*render.Effect
	for _, axis := range []struct {
		step geom.Vec2
		r    float32
	}{{geom.V2(1, 0), pr.Radius.X}, {geom.V2(0, 1), pr.Radius.Y}} {
		e := on.Effect(morphShader)
		e.SetVec2("uStep", axis.step)
		e.SetFloat("uRadius", float32(math.Round(float64(axis.r*pixels))))
		dilate := float32(0)
		if pr.Dilate {
			dilate = 1
		}
		e.SetFloat("uDilate", dilate)
		fxs = append(fxs, e)
	}
	return fxs
}

// composite draws a with b by pr's operator, either nil for nothing.
func (p *painter) composite(on Pictures, pr *fx.Primitive, a, b *twod.Picture) {
	draw := func(pic *twod.Picture, sum bool, fxs ...*render.Effect) {
		switch {
		case pic == nil:
		case sum:
			on.SumPicture(pic, fxs...)
		default:
			on.DrawPicture(pic, fxs...)
		}
	}
	// cut draws pic where by is opaque, or clear when inverted.
	cut := func(pic, by *twod.Picture, inverted, sum bool) {
		switch {
		case pic == nil:
		case by == nil && inverted:
			draw(pic, sum)
		case by == nil:
		case inverted:
			on.PushClipPicture(by, twod.Inverted())
			draw(pic, sum)
			p.s.PopClip()
		default:
			on.PushClipPicture(by)
			draw(pic, sum)
			p.s.PopClip()
		}
	}
	switch pr.Operator {
	case fx.Over:
		draw(b, false)
		draw(a, false)
	case fx.In:
		cut(a, b, false, false)
	case fx.Out:
		cut(a, b, true, false)
	case fx.Atop:
		cut(b, a, true, false)
		cut(a, b, false, true)
	case fx.Xor:
		cut(a, b, true, false)
		cut(b, a, true, true)
	case fx.Lighter:
		draw(a, false)
		draw(b, true)
	case fx.Arithmetic:
		arithmetic(on, pr, a, b)
	}
}

// arithmetic draws k2·a + k3·b of pr's K, either nil for nothing; k1·a·b
// and k4 are left out.
func arithmetic(on Pictures, pr *fx.Primitive, a, b *twod.Picture) {
	scaled := func(pic *twod.Picture, by float32, sum bool) {
		if pic == nil || by == 0 {
			return
		}
		e := on.Effect(scaleShader)
		e.SetFloat("uK", by)
		fxs := []*render.Effect{e}
		if pr.LinearRGB {
			fxs = []*render.Effect{on.Effect(toLinearShader), e, on.Effect(toSRGBShader)}
		}
		if sum {
			on.SumPicture(pic, fxs...)
		} else {
			on.DrawPicture(pic, fxs...)
		}
	}
	scaled(a, pr.K[1], false)
	scaled(b, pr.K[2], true)
}

// linearOf is pic in linear light.
func linearOf(on Pictures, pic *twod.Picture) *twod.Picture {
	on.BeginPicture()
	on.DrawPicture(pic, on.Effect(toLinearShader))
	return on.EndPicture(0)
}

// pixelsOf is how many of the surface's pixels a point of an element
// under xf takes.
func (p *painter) pixelsOf(xf geom.Transform2D) float32 {
	scale := p.scale
	if scale <= 0 {
		scale = 1
	}
	return scale * float32(math.Sqrt(math.Abs(float64(xf.X.X*xf.Y.Y-xf.X.Y*xf.Y.X))))
}

// moved is an offset of an element under xf in the surface's pixels.
func (p *painter) moved(d geom.Vec2, xf geom.Transform2D) geom.Vec2 {
	scale := p.scale
	if scale <= 0 {
		scale = 1
	}
	return geom.V2(xf.X.X*d.X+xf.Y.X*d.Y, xf.X.Y*d.X+xf.Y.Y*d.Y).Scale(scale)
}

// offsetEffect moves a picture by d, an element's offset under xf.
func (p *painter) offsetEffect(on Pictures, d geom.Vec2, xf geom.Transform2D) *render.Effect {
	e := on.Effect(offsetShader)
	e.SetVec2("uOffset", p.moved(d, xf))
	return e
}

// blurXY appends a Gaussian blur of deviation sigma pixels, across by
// its X and down by its Y, to fx.
func blurXY(on Pictures, fxs []*render.Effect, sigma geom.Vec2) []*render.Effect {
	for _, axis := range []struct {
		step geom.Vec2
		s    float32
	}{{geom.V2(1, 0), sigma.X}, {geom.V2(0, 1), sigma.Y}} {
		if axis.s < 0.01 {
			continue
		}
		e := on.Effect(blurShader)
		e.SetVec2("uStep", axis.step)
		e.SetFloat("uSigma", axis.s)
		fxs = append(fxs, e)
	}
	return fxs
}

// transferEffect is the effect of a ComponentTransfer's functions.
func transferEffect(on Pictures, ts [4]fx.Transfer) *render.Effect {
	e := on.Effect(transferShader)
	var kind, n, a, b, c [4]float32
	table := make([]geom.Vec4, maxTable)
	for t, f := range ts {
		kind[t] = float32(f.Kind)
		vals := sampled(f)
		n[t] = float32(len(vals))
		for i, v := range vals {
			setLane(&table[i], t, v)
		}
		switch f.Kind {
		case fx.Linear:
			a[t], b[t] = f.Slope, f.Intercept
		case fx.Gamma:
			a[t], b[t], c[t] = f.Amplitude, f.Exponent, f.Shift
		case fx.Identity, fx.Table, fx.Discrete:
		}
	}
	vec := func(v [4]float32) geom.Vec4 { return geom.Vec4{X: v[0], Y: v[1], Z: v[2], W: v[3]} }
	e.SetVec4("uKind", vec(kind))
	e.SetVec4("uN", vec(n))
	e.SetVec4("uA", vec(a))
	e.SetVec4("uB", vec(b))
	e.SetVec4("uC", vec(c))
	e.SetVec4s("uTable", table)
	return e
}

// sampled is a transfer's table, at most maxTable long: a longer one
// read at as many points through its own function.
func sampled(f fx.Transfer) []float32 {
	if f.Kind != fx.Table && f.Kind != fx.Discrete || len(f.Table) <= maxTable {
		return f.Table
	}
	out := make([]float32, maxTable)
	n := len(f.Table)
	for i := range out {
		x := float32(i) / float32(maxTable-1)
		if f.Kind == fx.Discrete {
			out[i] = f.Table[min(int(float32(i)/float32(maxTable)*float32(n)), n-1)]
			continue
		}
		at := x * float32(n-1)
		k := min(int(at), n-2)
		out[i] = f.Table[k] + (at-float32(k))*(f.Table[k+1]-f.Table[k])
	}
	return out
}

func setLane(v *geom.Vec4, t int, x float32) {
	switch t {
	case 0:
		v.X = x
	case 1:
		v.Y = x
	case 2:
		v.Z = x
	default:
		v.W = x
	}
}
