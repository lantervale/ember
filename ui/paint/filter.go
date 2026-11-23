package paint

import (
	"math"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/render"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/ui/internal/fx"
	"github.com/lantervale/ember/ui/internal/style"
)

// Pictures is a Surface that draws a part apart, as one picture, shows
// it through effects and cuts by its alpha: *twod.Canvas is one. On a
// Surface that isn't, filters and masks are not drawn.
type Pictures interface {
	BeginPicture()
	EndPicture(margin float32) *twod.Picture
	DrawPicture(p *twod.Picture, effects ...*render.Effect)
	SumPicture(p *twod.Picture, effects ...*render.Effect)
	PushClipPicture(p *twod.Picture, opts ...twod.MaskOption)
	DrawPictureOver(p, backdrop *twod.Picture, m twod.Mix, effects ...*render.Effect)
	Effect(shader string) *render.Effect
}

// The filters' shaders read and write premultiplied colours in sRGB, as
// Chrome filters them.
const (
	// matrixShader multiplies a colour, unpremultiplied, by a 4×5 matrix:
	// one row each for red, green, blue and alpha, and the offsets.
	matrixShader = `fn effect(vUV: vec2f) -> vec4f {
	let c = color(vUV);
	if (c.a <= 0.0) {
		return vec4f(0.0);
	}
	let u = vec4f(c.rgb / c.a, c.a);
	let o = clamp(vec4f(dot(uR, u), dot(uG, u), dot(uB, u), dot(uA, u)) + uO, vec4f(0.0), vec4f(1.0));
	return vec4f(o.rgb * o.a, o.a);
}`
	// blurShader blurs along uStep, a pixel of the picture one way, by a
	// Gaussian of deviation uSigma pixels; past the picture is clear.
	blurShader = `fn sampled(uv: vec2f) -> vec4f {
	if (any(uv < vec2f(0.0)) || any(uv > vec2f(1.0))) {
		return vec4f(0.0);
	}
	return color(uv);
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
	// shadowShader is the picture's alpha moved by uOffset pixels in the
	// premultiplied colour uShadow.
	shadowShader = `fn effect(vUV: vec2f) -> vec4f {
	let uv = vUV - uOffset / uSize;
	if (any(uv < vec2f(0.0)) || any(uv > vec2f(1.0))) {
		return vec4f(0.0);
	}
	return uShadow * color(uv).a;
}`
)

// filtering is a filtered element's pictures being drawn: the filters,
// how many pictures are open round its content, and the clips its
// content is drawn above.
type filtering struct {
	on      Pictures
	filters []style.Filter
	open    int
	floor   *clip
	spread  float32
	// graphs are the filter elements the filters name, by their index,
	// read for the element's border box at at, size large.
	graphs   []*fx.Graph
	at, size geom.Vec2
}

// beginFilters opens the pictures st's filters draw an element in, its
// content's and one round it for each drop shadow other filters follow,
// none when the surface draws no pictures or a scan is walking.
func (p *painter) beginFilters(e entry, st *style.Computed) *filtering {
	on, ok := p.s.(Pictures)
	if !ok || len(st.Filters) == 0 || p.probing {
		return nil
	}
	f := &filtering{on: on, filters: st.Filters, floor: p.floor, spread: p.spread}
	if urlFilters(st.Filters) {
		graphs, ok := p.graphs(e, st)
		if !ok {
			return nil
		}
		f.graphs, f.at, f.size = graphs, e.at, e.b.Size
		p.use(nil)
		on.BeginPicture()
		p.pictures++
		p.floor, p.spread = e.clip, p.spread+graphReach(st.Filters, graphs, 0, e.b.Size)
		return f
	}
	for i, fl := range st.Filters {
		if fl.Kind == style.DropShadow && i < len(st.Filters)-1 {
			f.open++
		}
	}
	p.use(nil)
	for range f.open + 1 {
		on.BeginPicture()
	}
	p.pictures += f.open + 1
	p.floor, p.spread = e.clip, p.spread+filterReach(st.Filters)
	return f
}

// endFilters shows what f's pictures hold through its filters, in order,
// in the clip c: each drop shadow under the picture so far, into the
// picture round it when more filters follow.
func (p *painter) endFilters(f *filtering, c *clip) {
	if f == nil {
		return
	}
	p.use(nil)
	p.floor, p.spread = f.floor, f.spread
	p.pictures -= f.open + 1
	if f.graphs != nil {
		p.endGraphs(f, c)
		return
	}
	xf := p.s.Transform()
	scale := float32(math.Sqrt(math.Abs(float64(xf.X.X*xf.Y.Y - xf.X.Y*xf.Y.X))))
	from := 0
	pic := f.on.EndPicture(segmentReach(f.filters, from) * scale)
	show := func(pic *twod.Picture, fx []*render.Effect, last bool) {
		if last {
			p.use(c)
		}
		f.on.DrawPicture(pic, fx...)
	}
	var pending []*render.Effect
	for i, fl := range f.filters {
		if fl.Kind != style.DropShadow {
			pending = p.filterEffects(f.on, pending, fl, xf)
			continue
		}
		last := i == len(f.filters)-1
		show(pic, p.filterEffects(f.on, pending, fl, xf), last)
		show(pic, pending, last)
		pending = nil
		if !last {
			from = i + 1
			pic = f.on.EndPicture(segmentReach(f.filters, from) * scale)
		}
	}
	if len(f.filters) > 0 && f.filters[len(f.filters)-1].Kind != style.DropShadow {
		show(pic, pending, true)
	}
}

// segmentReach is how far, in the element's points, the filters from
// from up to the next drop shadow, it included, draw past what they
// filter.
func segmentReach(filters []style.Filter, from int) float32 {
	for i := from; i < len(filters); i++ {
		if filters[i].Kind == style.DropShadow {
			return filterReach(filters[from : i+1])
		}
	}
	return filterReach(filters[from:])
}

// filterReach is how far, in the element's points, filters draw past
// what they filter: three deviations of each blur, and each drop shadow's
// offset and blur.
func filterReach(filters []style.Filter) float32 {
	var d float32
	for _, f := range filters {
		switch f.Kind {
		case style.Blur:
			d += 3 * f.Amount
		case style.DropShadow:
			d += max(abs(f.Shadow.X), abs(f.Shadow.Y)) + 1.5*f.Shadow.Blur
		case style.Brightness, style.Contrast, style.Grayscale, style.HueRotate, style.Invert, style.OpacityFilter,
			style.Saturate, style.Sepia, style.FilterURL:
		}
	}
	return d
}

// filterInk is r, what an element of size and its content draw, as
// filters leave it: grown by each blur, joined by each drop shadow and
// by the default region of a filter element, a tenth of size round it.
func filterInk(r geom.Rect, filters []style.Filter, size geom.Vec2) geom.Rect {
	for _, f := range filters {
		switch f.Kind {
		case style.Blur:
			r = grow(r, 3*f.Amount)
		case style.DropShadow:
			s := grow(geom.Rect{X: r.X + f.Shadow.X, Y: r.Y + f.Shadow.Y, W: r.W, H: r.H}, 1.5*f.Shadow.Blur)
			r = union(r, s)
		case style.FilterURL:
			r = union(r, geom.Rect{X: -0.1 * size.X, Y: -0.1 * size.Y, W: 1.2 * size.X, H: 1.2 * size.Y})
		case style.Brightness, style.Contrast, style.Grayscale, style.HueRotate, style.Invert, style.OpacityFilter,
			style.Saturate, style.Sepia:
		}
	}
	return r
}

// filterEffects appends to fx the effects that draw f under xf, the
// element's transform, in the surface's pixels.
func (p *painter) filterEffects(
	on Pictures, fx []*render.Effect, f style.Filter, xf geom.Transform2D,
) []*render.Effect {
	scale := p.scale
	if scale <= 0 {
		scale = 1
	}
	pixels := scale * float32(math.Sqrt(math.Abs(float64(xf.X.X*xf.Y.Y-xf.X.Y*xf.Y.X))))
	switch f.Kind {
	case style.Blur:
		return blurEffects(on, fx, f.Amount*pixels)
	case style.DropShadow:
		e := on.Effect(shadowShader)
		at := geom.V2(xf.X.X*f.Shadow.X+xf.Y.X*f.Shadow.Y, xf.X.Y*f.Shadow.X+xf.Y.Y*f.Shadow.Y).Scale(scale)
		e.SetVec2("uOffset", at)
		c, s := f.Shadow.Color, f.Shadow.Color.SRGB()
		e.SetVec4("uShadow", geom.Vec4{X: s[0] * c.A, Y: s[1] * c.A, Z: s[2] * c.A, W: c.A})
		return blurEffects(on, append(fx, e), f.Shadow.Blur/2*pixels)
	case style.Brightness, style.Contrast, style.Grayscale, style.HueRotate, style.Invert, style.OpacityFilter,
		style.Saturate, style.Sepia, style.FilterURL:
	}
	m := colorMatrix(f)
	e := on.Effect(matrixShader)
	for i, name := range []string{"uR", "uG", "uB", "uA"} {
		e.SetVec4(name, geom.Vec4{X: m[i][0], Y: m[i][1], Z: m[i][2], W: m[i][3]})
	}
	e.SetVec4("uO", geom.Vec4{X: m[0][4], Y: m[1][4], Z: m[2][4], W: m[3][4]})
	return append(fx, e)
}

// blurEffects appends a Gaussian blur of deviation sigma pixels, across
// then down, to fx; none for a blur under a hundredth of a pixel.
func blurEffects(on Pictures, fx []*render.Effect, sigma float32) []*render.Effect {
	if sigma < 0.01 {
		return fx
	}
	for _, step := range []geom.Vec2{geom.V2(1, 0), geom.V2(0, 1)} {
		e := on.Effect(blurShader)
		e.SetVec2("uStep", step)
		e.SetFloat("uSigma", sigma)
		fx = append(fx, e)
	}
	return fx
}

// colorMatrix is the 4×5 matrix of a filter on colours, as Filter Effects
// defines it: rows of red, green, blue and alpha, the last column added.
func colorMatrix(f style.Filter) [4][5]float32 {
	a := f.Amount
	m := [4][5]float32{{1}, {0, 1}, {0, 0, 1}, {0, 0, 0, 1}}
	switch f.Kind {
	case style.Brightness:
		m[0][0], m[1][1], m[2][2] = a, a, a
	case style.Contrast:
		for i := range 3 {
			m[i][i], m[i][4] = a, 0.5-0.5*a
		}
	case style.Invert:
		for i := range 3 {
			m[i][i], m[i][4] = 1-2*a, a
		}
	case style.OpacityFilter:
		m[3][3] = a
	case style.Grayscale:
		k := 1 - a
		m[0] = [5]float32{0.2126 + 0.7874*k, 0.7152 - 0.7152*k, 0.0722 - 0.0722*k}
		m[1] = [5]float32{0.2126 - 0.2126*k, 0.7152 + 0.2848*k, 0.0722 - 0.0722*k}
		m[2] = [5]float32{0.2126 - 0.2126*k, 0.7152 - 0.7152*k, 0.0722 + 0.9278*k}
	case style.Sepia:
		k := 1 - a
		m[0] = [5]float32{0.393 + 0.607*k, 0.769 - 0.769*k, 0.189 - 0.189*k}
		m[1] = [5]float32{0.349 - 0.349*k, 0.686 + 0.314*k, 0.168 - 0.168*k}
		m[2] = [5]float32{0.272 - 0.272*k, 0.534 - 0.534*k, 0.131 + 0.869*k}
	case style.Saturate:
		m[0] = [5]float32{0.213 + 0.787*a, 0.715 - 0.715*a, 0.072 - 0.072*a}
		m[1] = [5]float32{0.213 - 0.213*a, 0.715 + 0.285*a, 0.072 - 0.072*a}
		m[2] = [5]float32{0.213 - 0.213*a, 0.715 - 0.715*a, 0.072 + 0.928*a}
	case style.HueRotate:
		s, c := math.Sincos(float64(a) * math.Pi / 180)
		cos, sin := float32(c), float32(s)
		m[0] = [5]float32{0.213 + 0.787*cos - 0.213*sin, 0.715 - 0.715*cos - 0.715*sin, 0.072 - 0.072*cos + 0.928*sin}
		m[1] = [5]float32{0.213 - 0.213*cos + 0.143*sin, 0.715 + 0.285*cos + 0.140*sin, 0.072 - 0.072*cos - 0.283*sin}
		m[2] = [5]float32{0.213 - 0.213*cos - 0.787*sin, 0.715 - 0.715*cos + 0.715*sin, 0.072 + 0.928*cos + 0.072*sin}
	case style.Blur, style.DropShadow, style.FilterURL:
	}
	return m
}
