package svg

import (
	"image/color"
	"math"
	"strings"
)

const (
	elemFilter = "filter"
	// maxPrimitives is the most primitives one filter runs; those past it
	// are dropped.
	maxPrimitives = 64
	// maxTable is the most values a transfer function's table keeps.
	maxTable = 256
)

// blendModes are mix-blend-mode's and feBlend's names.
var blendModes = map[string]BlendMode{ //nolint:gochecknoglobals // a constant table
	valNormal: BlendNormal, "multiply": BlendMultiply, "screen": BlendScreen, "overlay": BlendOverlay,
	"darken": BlendDarken, "lighten": BlendLighten, "color-dodge": BlendColorDodge, "color-burn": BlendColorBurn,
	"hard-light": BlendHardLight, "soft-light": BlendSoftLight, "difference": BlendDifference,
	"exclusion": BlendExclusion, "hue": BlendHue, "saturation": BlendSaturation, "color": BlendColor,
	"luminosity": BlendLuminosity, "plus-lighter": BlendPlusLighter,
}

// filterContext is what a filter element's lengths are read against: the
// filtered element's bounding box, and whether its primitives' units are
// that box's.
type filterContext struct {
	bbox  Box
	boxed bool
}

// scaled is p in the filter's space: as it is, or of the box when the
// units are the box's.
func (c filterContext) scaled(p Point) Point {
	if c.boxed {
		return Point{p.X * c.bbox.W, p.Y * c.bbox.H}
	}
	return p
}

// pair reads one or two numbers, the second the first when absent, each
// scaled to the box along its axis when the units are the box's.
func pair(s string, c filterContext, def float64) (Point, bool) {
	sc := &scanner{s: s}
	v := sc.list(3)
	if !sc.done() || len(v) > 2 {
		return Point{def, def}, strings.TrimSpace(s) == ""
	}
	p := Point{def, def}
	switch len(v) {
	case 1:
		p = Point{v[0], v[0]}
	case 2:
		p = Point{v[0], v[1]}
	}
	return c.scaled(p), true
}

// filters are the filters of an element's filter property: those of the
// filter elements it names and of its CSS functions. m takes the
// element's user space to the picture's, where its items' bounding box
// is bbox; shown is false when the element is not drawn at all.
func (b *builder) filters(v string, st *style, m Matrix, bbox Box) ([]*Filter, bool) {
	v = strings.TrimSpace(v)
	var out []*Filter
	var fns *Filter
	// linear is the colour space of the last primitive: functions after a
	// filter element work in it, as Chrome runs them, a shadow's colour
	// taken as it is in that space.
	linear := false
	for v != "" {
		open := strings.IndexByte(v, '(')
		end := closing(v, open)
		if open <= 0 || end < 0 {
			b.unsupported("filter")
			return nil, true
		}
		name, args := strings.ToLower(strings.TrimSpace(v[:open])), strings.TrimSpace(v[open+1:end])
		v = strings.TrimSpace(v[end+1:])
		if name == "url" {
			fns = nil
			n := b.target(strings.Trim(args, `'" `))
			if n == nil || n.name != elemFilter {
				return nil, true
			}
			f, ok := b.filterElement(n, m, bbox)
			if !ok {
				return nil, false
			}
			out = append(out, f)
			linear = f.Primitives[len(f.Primitives)-1].LinearRGB
			continue
		}
		p, ok := cssFunction(name, args, st)
		if !ok {
			b.unsupported("filter")
			return nil, true
		}
		if p.LinearRGB = linear; linear {
			p.Color = encoded(p.Color)
		}
		if fns == nil {
			fns = &Filter{Transform: m}
			out = append(out, fns)
		}
		if p.Kind != Clear {
			p.In = []Input{SourceGraphic}
			if len(fns.Primitives) > 0 {
				p.In[0] = Input(len(fns.Primitives) - 1)
			}
			fns.Primitives = append(fns.Primitives, p)
		}
	}
	return out, true
}

// encoded is the colour that is c in linear light: what a primitive in
// linear light paints when it takes c's values as they are.
func encoded(c color.NRGBA) color.NRGBA {
	at := func(v uint8) uint8 { return round8(255 * float64(toSRGB(float32(v)/255))) }
	return color.NRGBA{at(c.R), at(c.G), at(c.B), c.A}
}

// closing is the index of the bracket closing the one at open in s, -1
// for none.
func closing(s string, open int) int {
	if open < 0 {
		return -1
	}
	depth := 0
	for i := open; i < len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			if depth--; depth == 0 {
				return i
			}
		}
	}
	return -1
}

// amount reads a CSS filter function's number or percentage, def when
// there is none.
func amount(s string, def float64) (float64, bool) {
	if strings.TrimSpace(s) == "" {
		return def, true
	}
	sc := &scanner{s: s}
	v, ok := sc.number()
	if !ok {
		return 0, false
	}
	switch strings.TrimSpace(sc.s[sc.i:]) {
	case "":
	case "%":
		v /= 100
	default:
		return 0, false
	}
	return v, v >= 0
}

func identityMatrix() [20]float64 {
	return [20]float64{0: 1, 6: 1, 12: 1, 18: 1}
}

// mixMatrix is the colour matrix to (a fraction 1-keep of the way to) to.
func mixMatrix(to [9]float64, keep float64) [20]float64 {
	id := [9]float64{1, 0, 0, 0, 1, 0, 0, 0, 1}
	out := identityMatrix()
	for r := range 3 {
		for c := range 3 {
			out[r*5+c] = to[r*3+c] + (id[r*3+c]-to[r*3+c])*keep
		}
	}
	return out
}

//nolint:gochecknoglobals // constant matrices
var (
	greyMatrix  = [9]float64{0.2126, 0.7152, 0.0722, 0.2126, 0.7152, 0.0722, 0.2126, 0.7152, 0.0722}
	sepiaMatrix = [9]float64{0.393, 0.769, 0.189, 0.349, 0.686, 0.168, 0.272, 0.534, 0.131}
)

func saturateMatrix(s float64) [20]float64 {
	out := identityMatrix()
	copy(out[0:3], []float64{0.213 + 0.787*s, 0.715 - 0.715*s, 0.072 - 0.072*s})
	copy(out[5:8], []float64{0.213 - 0.213*s, 0.715 + 0.285*s, 0.072 - 0.072*s})
	copy(out[10:13], []float64{0.213 - 0.213*s, 0.715 - 0.715*s, 0.072 + 0.928*s})
	return out
}

func hueMatrix(deg float64) [20]float64 {
	sin, cos := math.Sincos(deg * math.Pi / 180)
	out := identityMatrix()
	copy(out[0:3], []float64{
		0.213 + cos*0.787 - sin*0.213, 0.715 - cos*0.715 - sin*0.715, 0.072 - cos*0.072 + sin*0.928,
	})
	copy(out[5:8], []float64{
		0.213 - cos*0.213 + sin*0.143, 0.715 + cos*0.285 + sin*0.140, 0.072 - cos*0.072 - sin*0.283,
	})
	copy(out[10:13], []float64{
		0.213 - cos*0.213 - sin*0.787, 0.715 - cos*0.715 + sin*0.715, 0.072 + cos*0.928 + sin*0.072,
	})
	return out
}

func linearTransfer(slope, intercept float64) Transfer {
	return Transfer{Kind: TransferLinear, Slope: slope, Intercept: intercept}
}

// cssFunction is the primitive of one CSS filter function, in sRGB as
// CSS filters work; a function that changes nothing is Clear.
func cssFunction(name, args string, st *style) (Primitive, bool) {
	p := Primitive{}
	switch name {
	case "blur":
		r, ok := 0.0, true
		if strings.TrimSpace(args) != "" {
			r, ok = length(args, lengthContext{em: st.fontSize})
		}
		if !ok || r < 0 {
			return p, false
		}
		if r > 0 {
			p.Kind, p.StdDev = GaussianBlur, Point{r, r}
		}
		return p, true
	case "drop-shadow":
		return dropShadowFunction(args, st)
	case "hue-rotate":
		deg := 0.0
		if strings.TrimSpace(args) != "" {
			v, ok := angleValue(args)
			if !ok {
				return p, false
			}
			deg = v
		}
		p.Kind, p.Matrix = ColorMatrix, hueMatrix(deg)
		return p, true
	}
	a, ok := amount(args, 1)
	if !ok {
		return p, false
	}
	switch name {
	case "grayscale":
		p.Kind, p.Matrix = ColorMatrix, mixMatrix(greyMatrix, 1-min(a, 1))
	case "sepia":
		p.Kind, p.Matrix = ColorMatrix, mixMatrix(sepiaMatrix, 1-min(a, 1))
	case "saturate":
		p.Kind, p.Matrix = ColorMatrix, saturateMatrix(a)
	case "invert":
		a = min(a, 1)
		t := linearTransfer(1-2*a, a)
		p.Kind, p.Transfer = ComponentTransfer, [4]Transfer{t, t, t, {}}
	case "opacity":
		p.Kind, p.Transfer = ComponentTransfer, [4]Transfer{3: linearTransfer(min(a, 1), 0)}
	case "brightness":
		t := linearTransfer(a, 0)
		p.Kind, p.Transfer = ComponentTransfer, [4]Transfer{t, t, t, {}}
	case "contrast":
		t := linearTransfer(a, 0.5-0.5*a)
		p.Kind, p.Transfer = ComponentTransfer, [4]Transfer{t, t, t, {}}
	default:
		return p, false
	}
	return p, true
}

// dropShadowFunction reads drop-shadow(): a colour and two or three
// lengths, the colour before or after them.
func dropShadowFunction(args string, st *style) (Primitive, bool) {
	p := Primitive{Kind: DropShadow, Color: st.color}
	var lengths []float64
	for args = strings.TrimSpace(args); args != ""; args = strings.TrimSpace(args) {
		word := args
		if open := strings.IndexByte(args, '('); open >= 0 && strings.IndexFunc(args[:open], isSpace) < 0 {
			if end := closing(args, open); end > open {
				word = args[:end+1]
			}
		} else if i := strings.IndexFunc(args, isSpace); i >= 0 {
			word = args[:i]
		}
		args = args[len(word):]
		if v, ok := length(word, lengthContext{em: st.fontSize}); ok {
			lengths = append(lengths, v)
		} else if c, isColor := parseColor(word); isColor {
			p.Color = c
		} else if strings.EqualFold(word, "currentColor") {
			p.Color = st.color
		} else {
			return p, false
		}
	}
	if len(lengths) < 2 || len(lengths) > 3 || len(lengths) == 3 && lengths[2] < 0 {
		return p, false
	}
	p.Offset = Point{lengths[0], lengths[1]}
	if len(lengths) == 3 {
		p.StdDev = Point{lengths[2], lengths[2]}
	}
	return p, true
}

func isSpace(r rune) bool { return r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == '\f' }

// filterElement is the filter n on an element whose user space m takes to
// the picture's, with its bounding box bbox; ok is false when the
// element is not drawn.
func (b *builder) filterElement(n *node, m Matrix, bbox Box) (*Filter, bool) {
	props := b.props(n)
	userRegion := strings.TrimSpace(props["filterUnits"]) == unitsUser
	if !userRegion && bbox.Empty() {
		return nil, false
	}
	def := Box{-0.1, -0.1, 1.2, 1.2}
	if userRegion {
		def = Box{-0.1 * b.vw, -0.1 * b.vh, 1.2 * b.vw, 1.2 * b.vh}
	}
	f := &Filter{Region: b.region(props, "filterUnits", bbox, def), Transform: m}
	if f.Region.Empty() {
		return nil, false
	}
	c := filterContext{bbox: bbox, boxed: strings.TrimSpace(props["primitiveUnits"]) == unitsBox}
	results := map[string]Input{}
	for _, k := range n.kids {
		if !strings.HasPrefix(k.name, "fe") || len(f.Primitives) >= maxPrimitives {
			continue
		}
		f.Primitives = append(f.Primitives, b.primitive(k, f, c, results))
	}
	if len(f.Primitives) == 0 {
		return nil, false
	}
	return f, true
}

// input is what a primitive's in or in2 names: a standard input, an
// earlier result, or, unnamed or unknown, the result before it. Chrome
// draws neither the background nor the fill and stroke paints, and
// reads BackgroundImage, BackgroundAlpha, FillPaint and StrokePaint so
// too.
func (b *builder) input(v string, f *Filter, results map[string]Input) Input {
	switch v = strings.TrimSpace(v); v {
	case "SourceGraphic":
		return SourceGraphic
	case "SourceAlpha":
		return SourceAlpha
	}
	if in, ok := results[v]; ok && v != "" {
		return in
	}
	if len(f.Primitives) == 0 {
		return SourceGraphic
	}
	return Input(len(f.Primitives) - 1)
}

// subregion is the region of a primitive reading ins, from its x, y,
// width and height: each, when absent, from the union of its inputs'
// regions, or the filter's when one of them is a standard input.
func (b *builder) subregion(props map[string]string, f *Filter, c filterContext, ins []Input) Box {
	r := Box{}
	standard := len(ins) == 0
	for _, in := range ins {
		if in < 0 {
			standard = true
			break
		}
		r = union(r, f.Primitives[in].Region)
	}
	if standard {
		r = f.Region
	}
	at := func(k string, whole, origin float64, set *float64) {
		v, ok := props[k]
		if !ok {
			return
		}
		if c.boxed {
			if s, valid := length(v, lengthContext{whole: 1}); valid {
				*set = origin + s*whole
			}
			return
		}
		if s, valid := length(v, lengthContext{whole: whole, em: 16}); valid {
			*set = s
		}
	}
	boxW, boxH, boxX, boxY := b.vw, b.vh, 0.0, 0.0
	if c.boxed {
		boxW, boxH, boxX, boxY = c.bbox.W, c.bbox.H, c.bbox.X, c.bbox.Y
	}
	at("x", boxW, boxX, &r.X)
	at("y", boxH, boxY, &r.Y)
	at("width", boxW, 0, &r.W)
	at("height", boxH, 0, &r.H)
	return r
}

func union(a, b Box) Box {
	if a.Empty() {
		return b
	}
	if b.Empty() {
		return a
	}
	x0, y0 := math.Min(a.X, b.X), math.Min(a.Y, b.Y)
	x1, y1 := math.Max(a.X+a.W, b.X+b.W), math.Max(a.Y+a.H, b.Y+b.H)
	return Box{x0, y0, x1 - x0, y1 - y0}
}

// floodColor is a primitive's flood-color through its flood-opacity.
func floodColor(props map[string]string, st *style) color.NRGBA {
	c := color.NRGBA{A: 255}
	if v, ok := props["flood-color"]; ok {
		if strings.EqualFold(strings.TrimSpace(v), "currentColor") {
			c = st.color
		} else if parsed, valid := parseColor(v); valid {
			c = parsed
		}
	}
	if o, ok := fraction(props["flood-opacity"]); ok {
		c.A = round8(float64(c.A) * o)
	}
	return c
}

func (b *builder) primitive(n *node, f *Filter, c filterContext, results map[string]Input) Primitive {
	props := b.props(n)
	st := b.inherited(n)
	p := Primitive{LinearRGB: st.linearFilters, In: []Input{b.input(props["in"], f, results)}}
	b.primitiveKind(&p, n, props, &st, f, c, results)
	p.Region = b.subregion(props, f, c, p.In)
	if p.Kind == Tile {
		p.Region = b.subregion(props, f, c, nil)
	}
	if p.Region.Empty() {
		p.Kind = Clear
	}
	if name := strings.TrimSpace(props["result"]); name != "" {
		results[name] = Input(len(f.Primitives))
	}
	return p
}

// primitiveKind sets what the primitive n does, and what it reads
// beyond its in.
func (b *builder) primitiveKind(p *Primitive, n *node, props map[string]string, st *style, f *Filter, c filterContext,
	results map[string]Input,
) {
	if b.combining(p, n, props, f, results) {
		return
	}
	num := func(k string, def float64) float64 {
		if v, ok := number(props[k]); ok {
			return v
		}
		return def
	}
	switch n.name {
	case "feGaussianBlur":
		p.Kind = GaussianBlur
		dev, ok := pair(props["stdDeviation"], c, 0)
		if ok && dev.X >= 0 && dev.Y >= 0 {
			p.StdDev = dev
		}
	case "feOffset":
		p.Kind = Offset
		p.Offset = c.scaled(Point{num("dx", 0), num("dy", 0)})
	case "feDropShadow":
		p.Kind = DropShadow
		p.Offset = c.scaled(Point{num("dx", 2), num("dy", 2)})
		if dev, ok := pair(props["stdDeviation"], c, 2); ok && dev.X >= 0 && dev.Y >= 0 {
			p.StdDev = dev
		}
		p.Color = floodColor(props, st)
	case "feFlood":
		p.Kind, p.In = Flood, nil
		p.Color = floodColor(props, st)
	case "feColorMatrix":
		p.Kind = ColorMatrix
		p.Matrix = colorMatrix(props)
	case "feComponentTransfer":
		p.Kind = ComponentTransfer
		for _, k := range n.kids {
			channel := strings.Index("RGBA", strings.TrimPrefix(k.name, "feFunc"))
			if len(k.name) == len("feFuncR") && channel >= 0 {
				p.Transfer[channel] = transfer(b.props(k))
			}
		}
	case "feMorphology":
		p.Kind = Morphology
		p.Dilate = strings.TrimSpace(props["operator"]) == "dilate"
		if r, ok := pair(props["radius"], c, 0); ok {
			p.Radius = r
		}
	case "feTile":
		p.Kind = Tile
	default:
		p.Kind, p.In = Clear, nil
		b.unsupported(n.name)
	}
}

// combining sets the primitives that put pictures together, and is
// false for another.
func (b *builder) combining(p *Primitive, n *node, props map[string]string, f *Filter, results map[string]Input) bool {
	switch n.name {
	case "feComposite":
		p.Kind = Composite
		p.In = append(p.In, b.input(props["in2"], f, results))
		ops := map[string]CompositeOperator{
			"over": CompositeOver, "in": CompositeIn, "out": CompositeOut, "atop": CompositeAtop,
			"xor": CompositeXor, "lighter": CompositeLighter, "arithmetic": CompositeArithmetic,
		}
		p.Operator = ops[strings.TrimSpace(props["operator"])]
		for i, k := range [...]string{"k1", "k2", "k3", "k4"} {
			p.K[i], _ = number(props[k])
		}
	case "feBlend":
		p.Kind = Blend
		p.In = append(p.In, b.input(props["in2"], f, results))
		p.Mode = blendModes[strings.TrimSpace(props["mode"])]
	case "feMerge":
		p.Kind, p.In = Merge, nil
		for _, k := range n.kids {
			if k.name == "feMergeNode" {
				p.In = append(p.In, b.input(b.props(k)["in"], f, results))
			}
		}
	default:
		return false
	}
	return true
}

// colorMatrix is an feColorMatrix's matrix: its type's, from its values.
func colorMatrix(props map[string]string) [20]float64 {
	sc := &scanner{s: props["values"]}
	values := sc.list(21)
	given := strings.TrimSpace(props["values"]) != "" && sc.done()
	switch strings.TrimSpace(props["type"]) {
	case "saturate":
		if given && len(values) == 1 {
			return saturateMatrix(values[0])
		}
	case "hueRotate":
		if given && len(values) == 1 {
			return hueMatrix(values[0])
		}
	case "luminanceToAlpha":
		return [20]float64{15: 0.2125, 16: 0.7154, 17: 0.0721}
	default:
		if given && len(values) == 20 {
			var m [20]float64
			copy(m[:], values)
			return m
		}
	}
	return identityMatrix()
}

// transfer is the function of an feFuncR, G, B or A.
func transfer(props map[string]string) Transfer {
	num := func(k string, def float64) float64 {
		if v, ok := number(props[k]); ok {
			return v
		}
		return def
	}
	t := Transfer{
		Slope: num("slope", 1), Intercept: num("intercept", 0),
		Amplitude: num("amplitude", 1), Exponent: num("exponent", 1), Shift: num("offset", 0),
	}
	sc := &scanner{s: props["tableValues"]}
	t.Table = sc.list(maxTable)
	switch strings.TrimSpace(props["type"]) {
	case "table":
		t.Kind = TransferTable
	case "discrete":
		t.Kind = TransferDiscrete
	case "linear":
		t.Kind = TransferLinear
	case "gamma":
		t.Kind = TransferGamma
	}
	if (t.Kind == TransferTable || t.Kind == TransferDiscrete) && len(t.Table) == 0 {
		t.Kind = TransferIdentity
	}
	return t
}
