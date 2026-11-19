package svg

import (
	"image/color"
	"math"
	"strings"
)

// gradientDef is a gradient element with what it takes from the ones it
// refers to, before a shape gives it a bounding box.
type gradientDef struct {
	radial    bool
	userSpace bool
	attr      map[string]string
	stops     []Stop
	spread    Spread
	transform Matrix
}

func isGradient(n *node) bool {
	return n != nil && (n.name == "linearGradient" || n.name == "radialGradient")
}

func (b *builder) gradientDef(n *node) *gradientDef {
	if def, ok := b.gradients[n]; ok {
		return def
	}
	def := &gradientDef{radial: n.name == "radialGradient", attr: map[string]string{}, transform: Identity()}
	b.gradients[n] = def
	chain := []*node{n}
	for at := n; len(chain) < 16; {
		next := b.target(at.attr["href"])
		if !isGradient(next) {
			break
		}
		looped := false
		for _, seen := range chain {
			looped = looped || seen == next
		}
		if looped {
			break
		}
		chain = append(chain, next)
		at = next
	}
	for i := len(chain) - 1; i >= 0; i-- {
		for k, v := range b.props(chain[i]) {
			def.attr[k] = v
		}
	}
	for _, c := range chain {
		for _, k := range c.kids {
			if k.name != "stop" || len(def.stops) >= maxStops {
				continue
			}
			p := b.props(k)
			stop := Stop{Color: color.NRGBA{A: 255}}
			if v := strings.TrimSpace(p["stop-color"]); strings.EqualFold(v, "currentColor") {
				stop.Color = b.color
			} else if col, ok := parseColor(v); ok {
				stop.Color = col
			}
			if o, ok := fraction(p["stop-opacity"]); ok {
				stop.Color.A = round8(float64(stop.Color.A) * o)
			}
			stop.Offset, _ = fraction(p["offset"])
			if len(def.stops) > 0 {
				stop.Offset = math.Max(stop.Offset, def.stops[len(def.stops)-1].Offset)
			}
			def.stops = append(def.stops, stop)
		}
		if len(def.stops) > 0 {
			break
		}
	}
	def.userSpace = strings.TrimSpace(def.attr["gradientUnits"]) == "userSpaceOnUse"
	switch strings.TrimSpace(def.attr["spreadMethod"]) {
	case "reflect":
		def.spread = Reflect
	case "repeat":
		def.spread = Repeat
	}
	if t, ok := parseTransform(def.attr["gradientTransform"]); ok {
		def.transform = t
	}
	return def
}

// gradient is the paint of the gradient element n on a shape whose
// path is within bounds; ok is false when n is no gradient, or one that
// paints nothing.
func (b *builder) gradient(n *node, opacity float64, bounds Box) (Paint, bool) {
	if n != nil && n.name == "pattern" {
		b.unsupported("pattern")
		return Paint{}, false
	}
	if !isGradient(n) || opacity == 0 {
		return Paint{}, false
	}
	def := b.gradientDef(n)
	if len(def.stops) == 0 {
		return Paint{}, false
	}
	last := def.stops[len(def.stops)-1].Color
	solid := Paint{Kind: Solid, Color: last, Opacity: opacity}
	if last.A == 0 {
		solid = Paint{}
	}
	if len(def.stops) == 1 {
		return solid, true
	}
	g := &Gradient{Radial: def.radial, Stops: def.stops, Spread: def.spread}
	w, h, diag := 1.0, 1.0, 1.0
	space := Identity()
	if def.userSpace {
		w, h = b.vw, b.vh
		diag = math.Sqrt((w*w + h*h) / 2)
	} else {
		if bounds.Empty() {
			return Paint{}, true
		}
		space = Matrix{bounds.W, 0, 0, bounds.H, bounds.X, bounds.Y}
	}
	get := func(name string, whole, def2 float64) float64 {
		if v, ok := length(def.attr[name], lengthContext{whole: whole, em: 16}); ok {
			return v
		}
		return def2
	}
	if g.Radial {
		g.End = Point{get("cx", w, w/2), get("cy", h, h/2)}
		g.Radius = get("r", diag, diag/2)
		g.Start = Point{get("fx", w, g.End.X), get("fy", h, g.End.Y)}
		g.FocalRadius = math.Max(get("fr", diag, 0), 0)
		if !(g.Radius > 0) {
			return solid, true
		}
	} else {
		g.Start = Point{get("x1", w, 0), get("y1", h, 0)}
		g.End = Point{get("x2", w, w), get("y2", h, 0)}
		if g.Start == g.End {
			return solid, true
		}
	}
	g.Transform = space.Mul(def.transform)
	if _, ok := g.Transform.Inverse(); !ok || !g.Transform.finite() {
		return Paint{}, true
	}
	return Paint{Kind: GradientPaint, Gradient: g, Opacity: opacity}, true
}
