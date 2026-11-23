package svg

import (
	"math"
	"strings"
)

// cssGradient is the gradient a CSS linear-gradient() or
// radial-gradient(), repeating or not, paints over box, in the box's
// space, its lengths of st's font; ok is false for another image or one
// this does not read.
func cssGradient(v string, box Box, st *style) (*Gradient, bool) {
	em := st.fontSize
	open := strings.IndexByte(v, '(')
	if open <= 0 || !strings.HasSuffix(v, ")") || box.Empty() {
		return nil, false
	}
	name := strings.ToLower(strings.TrimSpace(v[:open]))
	repeating := strings.HasPrefix(name, "repeating-")
	name = strings.TrimPrefix(name, "repeating-")
	if name != "linear-gradient" && name != "radial-gradient" {
		return nil, false
	}
	args := splitCommas(v[open+1 : len(v)-1])
	if len(args) == 0 {
		return nil, false
	}
	g := &Gradient{Transform: Identity()}
	var line float64
	var ok bool
	if name == "linear-gradient" {
		line, ok = linearLine(g, args[0], box)
	} else {
		g.Radial = true
		line, ok = radialShape(g, args[0], box, em)
	}
	if ok {
		args = args[1:]
	} else if line, ok = defaultLine(g, box); !ok {
		return nil, false
	}
	stops, ok := cssStops(args, line, st)
	if !ok {
		return nil, false
	}
	fitLine(g, stops, repeating)
	g.Stops = stops
	return g, true
}

// splitCommas splits s at the commas outside brackets.
func splitCommas(s string) []string {
	var out []string
	depth, start := 0, 0
	for i := range len(s) {
		switch s[i] {
		case '(':
			depth++
		case ')':
			depth--
		case ',':
			if depth == 0 {
				out = append(out, strings.TrimSpace(s[start:i]))
				start = i + 1
			}
		}
	}
	return append(out, strings.TrimSpace(s[start:]))
}

// defaultLine is a gradient's line when it gives none: to the bottom of
// a linear one, the farthest corner of an ellipse about the middle.
func defaultLine(g *Gradient, box Box) (float64, bool) {
	if g.Radial {
		return radialShape(g, "", box, 0)
	}
	return linearLine(g, "to bottom", box)
}

// linearLine sets a linear gradient's vector from its angle or its side
// or corner, the line through the box's middle long enough for its ends
// to touch the corners, and is its length.
func linearLine(g *Gradient, v string, box Box) (float64, bool) {
	words := strings.Fields(strings.ToLower(v))
	var angle float64
	switch {
	case len(words) >= 2 && words[0] == "to" && len(words) <= 3:
		var sx, sy float64
		for _, w := range words[1:] {
			switch w {
			case "left":
				sx = -1
			case "right":
				sx = 1
			case "top":
				sy = -1
			case "bottom":
				sy = 1
			default:
				return 0, false
			}
		}
		if len(words) == 3 && (sx == 0 || sy == 0) {
			return 0, false
		}
		d := Point{sx * box.H, sy * box.W}
		if sx == 0 || sy == 0 {
			d = Point{sx, sy}
		}
		angle = math.Atan2(d.X, -d.Y)
	case len(words) == 1:
		deg, ok := angleValue(words[0])
		if !ok || strings.TrimLeft(words[0], "+-.0123456789") == "" && deg != 0 {
			return 0, false
		}
		angle = deg * math.Pi / 180
	default:
		return 0, false
	}
	sin, cos := math.Sincos(angle)
	line := math.Abs(box.W*sin) + math.Abs(box.H*cos)
	c := Point{box.X + box.W/2, box.Y + box.H/2}
	d := Point{sin, -cos}.mul(line / 2)
	g.Start, g.End = c.sub(d), c.add(d)
	return line, true
}

// radialShape sets a radial gradient's circle or ellipse from its shape,
// size and centre, and is the length of its ray.
func radialShape(g *Gradient, v string, box Box, em float64) (float64, bool) {
	words := shapeArgs(strings.ToLower(v))
	var at []string
	for i, w := range words {
		if w == "at" {
			words, at = words[:i], words[i+1:]
			break
		}
	}
	c, ok := position(at, box, em)
	if !ok {
		return 0, false
	}
	shape, size := "", "farthest-corner"
	var lengths []string
	for _, w := range words {
		switch w {
		case elemCircle, elemEllipse:
			if shape != "" {
				return 0, false
			}
			shape = w
		case "closest-side", "farthest-side", "closest-corner", "farthest-corner":
			size = w
		default:
			lengths = append(lengths, w)
		}
	}
	if shape == "" {
		shape = elemEllipse
		if len(lengths) == 1 {
			shape = elemCircle
		}
	}
	dx := [2]float64{math.Abs(c.X - box.X), math.Abs(box.X + box.W - c.X)}
	dy := [2]float64{math.Abs(c.Y - box.Y), math.Abs(box.Y + box.H - c.Y)}
	var rx, ry float64
	switch {
	case len(lengths) == 1 && shape == elemCircle:
		if rx, ok = length(lengths[0], lengthContext{em: em}); !ok || rx < 0 || strings.HasSuffix(lengths[0], "%") {
			return 0, false
		}
		ry = rx
	case len(lengths) == 2 && shape == elemEllipse:
		rx, okX := length(lengths[0], lengthContext{whole: box.W, em: em})
		ry, okY := length(lengths[1], lengthContext{whole: box.H, em: em})
		if !okX || !okY || rx < 0 || ry < 0 {
			return 0, false
		}
		return setRadial(g, c, rx, ry), true
	case len(lengths) != 0:
		return 0, false
	default:
		rx, ry = radialSize(shape == elemCircle, size, dx, dy)
	}
	return setRadial(g, c, rx, ry), true
}

// radialSize is the radii of a circle or ellipse sized by a keyword,
// from the distances of its centre to the box's sides.
func radialSize(circle bool, size string, dx, dy [2]float64) (float64, float64) {
	near, far := strings.HasPrefix(size, "closest"), strings.HasSuffix(size, "corner")
	pick := func(d [2]float64) float64 {
		if near {
			return math.Min(d[0], d[1])
		}
		return math.Max(d[0], d[1])
	}
	x, y := pick(dx), pick(dy)
	switch {
	case circle && far:
		r := math.Hypot(x, y)
		return r, r
	case circle && near:
		r := math.Min(x, y)
		return r, r
	case circle:
		r := math.Max(x, y)
		return r, r
	case far:
		return x * math.Sqrt2, y * math.Sqrt2
	}
	return x, y
}

// setRadial makes g an ellipse of radii rx and ry about c, and is the
// length of its ray along x.
func setRadial(g *Gradient, c Point, rx, ry float64) float64 {
	g.Radius = rx
	if rx > 0 && ry > 0 {
		g.Transform = translate(c.X, c.Y).Mul(scale(1, ry/rx))
	} else {
		g.Transform = translate(c.X, c.Y)
	}
	return rx
}

// cssStops reads a gradient's colour stops along a line so long,
// placed as CSS places those without a position; hints are passed over.
func cssStops(args []string, line float64, st *style) ([]Stop, bool) {
	em := st.fontSize
	var stops []Stop
	var at []float64
	for _, a := range args {
		words := shapeArgs(a)
		if len(words) == 0 || len(words) > 3 {
			return nil, false
		}
		c, ok := parseColor(words[0])
		if strings.EqualFold(words[0], "currentColor") {
			c, ok = st.color, true
		}
		if !ok {
			if len(words) == 1 {
				if _, hint := length(words[0], lengthContext{whole: line, em: em}); hint {
					continue
				}
			}
			return nil, false
		}
		if len(words) == 1 {
			stops, at = append(stops, Stop{Color: c}), append(at, math.NaN())
			continue
		}
		for _, w := range words[1:] {
			v, valid := length(w, lengthContext{whole: line, em: em})
			if !valid {
				return nil, false
			}
			if line > 0 {
				v /= line
			}
			stops, at = append(stops, Stop{Color: c}), append(at, v)
		}
	}
	if len(stops) < 1 {
		return nil, false
	}
	if math.IsNaN(at[0]) {
		at[0] = 0
	}
	if last := len(at) - 1; math.IsNaN(at[last]) {
		at[last] = math.Max(1, at[0])
	}
	most := at[0]
	for i := range at {
		if !math.IsNaN(at[i]) {
			most = math.Max(most, at[i])
			at[i] = most
		}
	}
	for i := 0; i < len(at); i++ {
		if !math.IsNaN(at[i]) {
			continue
		}
		j := i
		for math.IsNaN(at[j]) {
			j++
		}
		for k := i; k < j; k++ {
			at[k] = at[i-1] + (at[j]-at[i-1])*float64(k-i+1)/float64(j-i+1)
		}
	}
	for i := range stops {
		stops[i].Offset = at[i]
	}
	return stops, true
}

// fitLine places g's stops along its line from 0 to 1: the line cut to
// them, from the first to the last, for a repeating gradient, which then
// repeats them, and grown to them for another.
func fitLine(g *Gradient, stops []Stop, repeating bool) {
	first, last := stops[0].Offset, stops[len(stops)-1].Offset
	if !repeating {
		first, last = math.Min(first, 0), math.Max(last, 1)
	}
	if g.Radial {
		first = math.Max(first, 0)
	}
	span := last - first
	if !(span > 0) || first == 0 && last == 1 {
		for i := range stops {
			stops[i].Offset = min(max(stops[i].Offset, 0), 1)
		}
		return
	}
	if repeating {
		g.Spread = Repeat
	}
	for i := range stops {
		stops[i].Offset = min(max((stops[i].Offset-first)/span, 0), 1)
	}
	if g.Radial {
		g.FocalRadius, g.Radius = first*g.Radius, last*g.Radius
		return
	}
	d := g.End.sub(g.Start)
	g.Start, g.End = g.Start.add(d.mul(first)), g.Start.add(d.mul(last))
}
