package svg

import (
	"math"
	"strings"
)

// shapeArgs splits a CSS function's arguments at commas and white space
// outside brackets and quotes; commas are kept as words of their own.
func shapeArgs(s string) []string {
	var out []string
	start, depth := -1, 0
	var quote byte
	flush := func(i int) {
		if start >= 0 {
			out = append(out, s[start:i])
			start = -1
		}
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case quote != 0:
			if c == quote {
				quote = 0
			}
			continue
		case c == '"' || c == '\'':
			quote = c
		case c == '(':
			depth++
		case c == ')':
			depth--
		case depth == 0 && (isSpace(rune(c)) || c == ','):
			flush(i)
			if c == ',' {
				out = append(out, ",")
			}
			continue
		}
		if start < 0 {
			start = i
		}
	}
	flush(len(s))
	return out
}

// referenceBox is the box a basic shape of clip-path is in: the
// element's fill box, its stroke box, or its viewport's.
func (b *builder) referenceBox(name string, items []Item, bbox Box) (Box, bool) {
	switch name {
	case "fill-box", "content-box", "padding-box":
		return bbox, true
	case "stroke-box", "border-box", "margin-box":
		half := 0.0
		for _, s := range shapesOf(items, nil) {
			if s.Stroke.Kind != NoPaint {
				half = math.Max(half, s.StrokeWidth/2)
			}
		}
		return Box{bbox.X - half, bbox.Y - half, bbox.W + 2*half, bbox.H + 2*half}, true
	case "view-box":
		return Box{0, 0, b.vw, b.vh}, true
	}
	return Box{}, false
}

// shapeClip is the clip of a clip-path given as a basic shape or a box,
// on an element whose user space m takes to the picture's, drawing
// items within bbox there; nil when it is no shape this reads.
func (b *builder) shapeClip(v string, st *style, m Matrix, items []Item, bbox Box) *Clip {
	words := shapeArgs(v)
	var fn string
	box := "border-box"
	for _, w := range words {
		if open := strings.IndexByte(w, '('); open > 0 && strings.HasSuffix(w, ")") && fn == "" {
			fn = w
		} else if _, ok := b.referenceBox(w, nil, Box{}); ok {
			box = w
		} else {
			return nil
		}
	}
	ref, _ := b.referenceBox(box, items, bbox)
	var path Path
	evenOdd := false
	if fn == "" {
		path = rectPath(ref.X, ref.Y, ref.W, ref.H, 0, 0)
	} else {
		open := strings.IndexByte(fn, '(')
		var ok bool
		path, evenOdd, ok = basicShape(strings.ToLower(fn[:open]), fn[open+1:len(fn)-1], ref, st.fontSize)
		if !ok {
			return nil
		}
	}
	s := &Shape{Path: path, Transform: m, Fill: Paint{Kind: Solid, Opacity: 1}}
	if evenOdd {
		s.FillRule = EvenOdd
	}
	return &Clip{Shapes: []*Shape{s}}
}

// basicShape is the outline of a CSS basic shape in the reference box
// ref, and its fill rule.
func basicShape(name, args string, ref Box, em float64) (Path, bool, bool) {
	words := shapeArgs(args)
	switch name {
	case elemCircle, elemEllipse:
		return roundShape(name == elemCircle, words, ref, em)
	case "inset", "xywh", elemRect:
		p, ok := rectShape(name, words, ref, em)
		return p, false, ok
	case elemPolygon:
		return polygonShape(words, ref, em)
	case elemPath:
		evenOdd, rest := fillRule(words)
		if len(rest) != 1 {
			return Path{}, false, false
		}
		d, quoted := unquote(rest[0])
		if !quoted {
			return Path{}, false, false
		}
		p := parsePath(d, MaxSegments)
		for i := range p.Points {
			p.Points[i] = p.Points[i].add(Point{ref.X, ref.Y})
		}
		return p, evenOdd, len(p.Verbs) > 0
	}
	return Path{}, false, false
}

// unquote is s within its quotes; ok is false when it has none.
func unquote(s string) (string, bool) {
	if len(s) < 2 || s[0] != s[len(s)-1] || s[0] != '"' && s[0] != '\'' {
		return "", false
	}
	return s[1 : len(s)-1], true
}

// lengths reads four lengths, each a share of the box's width or its
// height as across says of its place; auto stands for autos'.
func lengths(words []string, ref Box, em float64, across [4]bool, autos *[4]float64) ([4]float64, bool) {
	var v [4]float64
	if len(words) != 4 {
		return v, false
	}
	for i, w := range words {
		if w == valAuto && autos != nil {
			v[i] = autos[i]
			continue
		}
		whole := ref.H
		if across[i] {
			whole = ref.W
		}
		var ok bool
		if v[i], ok = length(w, lengthContext{whole: whole, em: em}); !ok {
			return v, false
		}
	}
	return v, true
}

// rectShape is inset(), xywh() or rect(): a rectangle in ref, rounded as
// its round asks.
func rectShape(name string, words []string, ref Box, em float64) (Path, bool) {
	edges := words
	var round []string
	for i, w := range words {
		if w == valRound {
			edges, round = words[:i], words[i+1:]
			break
		}
	}
	r, ok := radius(round, ref, em)
	if !ok {
		return Path{}, false
	}
	var x0, y0, x1, y1 float64
	switch name {
	case "inset":
		t, valid := sides(edges, ref, em)
		if !valid {
			return Path{}, false
		}
		x0, y0, x1, y1 = ref.X+t[3], ref.Y+t[0], ref.X+ref.W-t[1], ref.Y+ref.H-t[2]
	case "xywh":
		v, valid := lengths(edges, ref, em, [4]bool{true, false, true, false}, nil)
		if !valid {
			return Path{}, false
		}
		x0, y0, x1, y1 = ref.X+v[0], ref.Y+v[1], ref.X+v[0]+v[2], ref.Y+v[1]+v[3]
	default:
		v, valid := lengths(edges, ref, em, [4]bool{false, true, false, true}, &[4]float64{0, ref.W, ref.H, 0})
		if !valid {
			return Path{}, false
		}
		x0, y0, x1, y1 = ref.X+v[3], ref.Y+v[0], ref.X+v[1], ref.Y+v[2]
	}
	if x1 < x0 {
		x0, x1 = (x0+x1)/2, (x0+x1)/2
	}
	if y1 < y0 {
		y0, y1 = (y0+y1)/2, (y0+y1)/2
	}
	return rectPath(x0, y0, x1-x0, y1-y0, r.X, r.Y), true
}

// polygonShape is polygon(): its points in ref, and its fill rule.
func polygonShape(words []string, ref Box, em float64) (Path, bool, bool) {
	evenOdd, rest := fillRule(words)
	var pt []float64
	for _, w := range rest {
		if w == "," {
			continue
		}
		whole := ref.W
		if len(pt)%2 == 1 {
			whole = ref.H
		}
		v, ok := length(w, lengthContext{whole: whole, em: em})
		if !ok {
			return Path{}, false, false
		}
		pt = append(pt, v)
	}
	if len(pt) < 6 || len(pt)%2 != 0 {
		return Path{}, false, false
	}
	var p Path
	p.moveTo(Point{ref.X + pt[0], ref.Y + pt[1]})
	for i := 2; i < len(pt); i += 2 {
		p.lineTo(Point{ref.X + pt[i], ref.Y + pt[i+1]})
	}
	p.close()
	return p, evenOdd, true
}

// fillRule takes a leading fill rule and its comma off words.
func fillRule(words []string) (bool, []string) {
	if len(words) > 0 && (words[0] == valEvenOdd || words[0] == valNonZero) {
		rest := words[1:]
		if len(rest) > 0 && rest[0] == "," {
			rest = rest[1:]
		}
		return words[0] == valEvenOdd, rest
	}
	return false, words
}

// sides reads one to four lengths, top, right, bottom and left as CSS
// repeats them.
func sides(words []string, ref Box, em float64) ([4]float64, bool) {
	var v [4]float64
	if len(words) == 0 || len(words) > 4 {
		return v, false
	}
	var got []float64
	for i, w := range words {
		whole := ref.H
		if i%2 == 1 {
			whole = ref.W
		}
		l, ok := length(w, lengthContext{whole: whole, em: em})
		if !ok {
			return v, false
		}
		got = append(got, l)
	}
	switch len(got) {
	case 1:
		return [4]float64{got[0], got[0], got[0], got[0]}, true
	case 2:
		return [4]float64{got[0], got[1], got[0], got[1]}, true
	case 3:
		return [4]float64{got[0], got[1], got[2], got[1]}, true
	}
	return [4]float64{got[0], got[1], got[2], got[3]}, true
}

// radius reads a round's radius: one length, or a horizontal and a
// vertical one about a slash; corners of their own are read as the
// first.
func radius(words []string, ref Box, em float64) (Point, bool) {
	if len(words) == 0 {
		return Point{}, true
	}
	x, ok := length(words[0], lengthContext{whole: ref.W, em: em})
	if !ok {
		return Point{}, false
	}
	y := x
	for i, w := range words {
		if w == "/" && i+1 < len(words) {
			if y, ok = length(words[i+1], lengthContext{whole: ref.H, em: em}); !ok {
				return Point{}, false
			}
		}
	}
	return Point{x, y}, true
}

// roundShape is circle() or ellipse(): its radii, closest-side by
// default, about its centre, the box's unless at says.
func roundShape(circle bool, words []string, ref Box, em float64) (Path, bool, bool) {
	var radii, at []string
	for i, w := range words {
		if w == "at" {
			at = words[i+1:]
			break
		}
		radii = append(radii, w)
	}
	c, ok := position(at, ref, em)
	if !ok || circle && len(radii) > 1 || !circle && len(radii) != 0 && len(radii) != 2 {
		return Path{}, false, false
	}
	dx := [2]float64{math.Abs(c.X - ref.X), math.Abs(ref.X + ref.W - c.X)}
	dy := [2]float64{math.Abs(c.Y - ref.Y), math.Abs(ref.Y + ref.H - c.Y)}
	side := func(word string, d [2]float64, whole float64) (float64, bool) {
		switch word {
		case "", "closest-side":
			return math.Min(d[0], d[1]), true
		case "farthest-side":
			return math.Max(d[0], d[1]), true
		}
		v, valid := length(word, lengthContext{whole: whole, em: em})
		return v, valid && v >= 0
	}
	if circle {
		word := ""
		if len(radii) == 1 {
			word = radii[0]
		}
		var r float64
		switch word {
		case "", "closest-side":
			r = math.Min(math.Min(dx[0], dx[1]), math.Min(dy[0], dy[1]))
		case "farthest-side":
			r = math.Max(math.Max(dx[0], dx[1]), math.Max(dy[0], dy[1]))
		default:
			if r, ok = side(word, dx, math.Hypot(ref.W, ref.H)/math.Sqrt2); !ok {
				return Path{}, false, false
			}
		}
		return ellipsePath(c, r, r), false, true
	}
	words2 := [2]string{}
	copy(words2[:], radii)
	rx, okX := side(words2[0], dx, ref.W)
	ry, okY := side(words2[1], dy, ref.H)
	if !okX || !okY {
		return Path{}, false, false
	}
	return ellipsePath(c, rx, ry), false, true
}

// position reads a CSS position of one or two words in ref, its centre
// when there are none.
func position(words []string, ref Box, em float64) (Point, bool) {
	at := Point{ref.X + ref.W/2, ref.Y + ref.H/2}
	if len(words) == 0 {
		return at, true
	}
	if len(words) > 2 {
		return at, false
	}
	keyword := map[string][2]float64{
		"left": {0, -1}, "right": {1, -1}, "top": {-1, 0}, "bottom": {-1, 1}, "center": {0.5, 0.5},
	}
	xSet, ySet := false, false
	for i, w := range words {
		if k, ok := keyword[w]; ok {
			switch {
			case w == "center":
				continue
			case k[0] >= 0:
				at.X, xSet = ref.X+k[0]*ref.W, true
			default:
				at.Y, ySet = ref.Y+k[1]*ref.H, true
			}
			continue
		}
		horizontal := i == 0 && !xSet || ySet && !xSet
		whole, origin := ref.H, ref.Y
		if horizontal {
			whole, origin = ref.W, ref.X
		}
		v, ok := length(w, lengthContext{whole: whole, em: em})
		if !ok {
			return at, false
		}
		if horizontal {
			at.X, xSet = origin+v, true
		} else {
			at.Y, ySet = origin+v, true
		}
	}
	return at, true
}
