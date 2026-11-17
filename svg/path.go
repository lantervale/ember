package svg

import (
	"math"
	"strconv"
	"strings"
)

func (p *Path) moveTo(q Point) {
	p.Verbs, p.Points = append(p.Verbs, MoveTo), append(p.Points, q)
}

func (p *Path) lineTo(q Point) {
	p.Verbs, p.Points = append(p.Verbs, LineTo), append(p.Points, q)
}

func (p *Path) quadTo(c, q Point) {
	p.Verbs, p.Points = append(p.Verbs, QuadTo), append(p.Points, c, q)
}

func (p *Path) cubicTo(c1, c2, q Point) {
	p.Verbs, p.Points = append(p.Verbs, CubicTo), append(p.Points, c1, c2, q)
}

func (p *Path) close() { p.Verbs = append(p.Verbs, Close) }

// arc appends the arc of an ellipse centred on c and turned by phi, from
// angle a0 over sweep, as cubic curves of a quarter turn at most; the arc's
// first point is where the path already is.
func (p *Path) arc(c Point, rx, ry, phi, a0, sweep float64) {
	n := int(math.Ceil(math.Abs(sweep) / (math.Pi / 2) * (1 - 1e-9)))
	n = max(n, 1)
	step := sweep / float64(n)
	k := 4.0 / 3 * math.Tan(step/4)
	sinPhi, cosPhi := math.Sincos(phi)
	at := func(x, y float64) Point {
		x, y = x*rx, y*ry
		return Point{c.X + x*cosPhi - y*sinPhi, c.Y + x*sinPhi + y*cosPhi}
	}
	for i := range n {
		s0, c0 := math.Sincos(a0 + step*float64(i))
		s1, c1 := math.Sincos(a0 + step*float64(i+1))
		p.cubicTo(at(c0-k*s0, s0+k*c0), at(c1+k*s1, s1-k*c1), at(c1, s1))
	}
}

// arcTo is the path A command from cur: SVG 1.1 implementation notes
// F.6.5 and F.6.6.
func (p *Path) arcTo(cur Point, rx, ry, rot float64, large, sweep bool, to Point) {
	rx, ry = math.Abs(rx), math.Abs(ry)
	if cur == to {
		return
	}
	if rx == 0 || ry == 0 {
		p.lineTo(to)
		return
	}
	phi := rot * math.Pi / 180
	sin, cos := math.Sincos(phi)
	d := cur.sub(to).mul(0.5)
	x1, y1 := cos*d.X+sin*d.Y, -sin*d.X+cos*d.Y
	if l := x1*x1/(rx*rx) + y1*y1/(ry*ry); l > 1 {
		rx, ry = rx*math.Sqrt(l), ry*math.Sqrt(l)
	}
	num := rx*rx*ry*ry - rx*rx*y1*y1 - ry*ry*x1*x1
	k := math.Sqrt(math.Max(0, num) / (rx*rx*y1*y1 + ry*ry*x1*x1))
	if large == sweep {
		k = -k
	}
	cx1, cy1 := k*rx*y1/ry, -k*ry*x1/rx
	c := Point{cos*cx1 - sin*cy1 + (cur.X+to.X)/2, sin*cx1 + cos*cy1 + (cur.Y+to.Y)/2}
	a0 := math.Atan2((y1-cy1)/ry, (x1-cx1)/rx)
	da := math.Atan2((-y1-cy1)/ry, (-x1-cx1)/rx) - a0
	if sweep && da < 0 {
		da += 2 * math.Pi
	} else if !sweep && da > 0 {
		da -= 2 * math.Pi
	}
	if math.IsNaN(da) || math.IsNaN(c.X+c.Y) {
		p.lineTo(to)
		return
	}
	p.arc(c, rx, ry, phi, a0, da)
	p.Points[len(p.Points)-1] = to
}

type scanner struct {
	s string
	i int
}

func (s *scanner) skip() {
	for s.i < len(s.s) && strings.IndexByte(" \t\r\n\f,", s.s[s.i]) >= 0 {
		s.i++
	}
}

func (s *scanner) done() bool {
	s.skip()
	return s.i >= len(s.s)
}

// number reads one number as path data writes them: a sign, digits with
// or without a point, an exponent, nothing between them and the next.
func (s *scanner) number() (float64, bool) {
	s.skip()
	i := s.i
	if i < len(s.s) && (s.s[i] == '+' || s.s[i] == '-') {
		i++
	}
	digits := func() bool {
		j := i
		for i < len(s.s) && s.s[i] >= '0' && s.s[i] <= '9' {
			i++
		}
		return i > j
	}
	whole := digits()
	frac := false
	if i < len(s.s) && s.s[i] == '.' {
		i++
		frac = digits()
	}
	if !whole && !frac {
		return 0, false
	}
	if i < len(s.s) && (s.s[i] == 'e' || s.s[i] == 'E') {
		j := i
		i++
		if i < len(s.s) && (s.s[i] == '+' || s.s[i] == '-') {
			i++
		}
		if !digits() {
			i = j
		}
	}
	v, err := strconv.ParseFloat(s.s[s.i:i], 64)
	s.i = i
	return v, err == nil && !math.IsInf(v, 0)
}

// flag reads an arc's flag: one 0 or 1, which may touch what follows.
func (s *scanner) flag() (bool, bool) {
	s.skip()
	if s.i < len(s.s) && (s.s[s.i] == '0' || s.s[s.i] == '1') {
		s.i++
		return s.s[s.i-1] == '1', true
	}
	return false, false
}

func (s *scanner) numbers(v []float64) bool {
	for i := range v {
		x, ok := s.number()
		if !ok {
			return false
		}
		v[i] = x
	}
	return true
}

// list reads every number of s, up to limit of them.
func (s *scanner) list(limit int) []float64 {
	var out []float64
	for len(out) < limit {
		v, ok := s.number()
		if !ok {
			break
		}
		out = append(out, v)
	}
	return out
}

// pathReader reads path data into a Path.
type pathReader struct {
	scanner
	path Path
	// cur is where the path is, start where its subpath began, ctl the
	// last control point, which S and T mirror.
	cur, start, ctl Point
	// open is false between a Z and the next command, which then starts a
	// subpath of its own where the closed one began.
	open bool
	rel  bool
	v    [6]float64
}

func (r *pathReader) at(x, y float64) Point {
	if r.rel {
		return Point{r.cur.X + x, r.cur.Y + y}
	}
	return Point{x, y}
}

// line reads the arguments of M, L, H or V.
func (r *pathReader) line(cmd byte) bool {
	switch cmd {
	case 'm', 'l':
		if !r.numbers(r.v[:2]) {
			return false
		}
		r.cur = r.at(r.v[0], r.v[1])
	case 'h':
		if !r.numbers(r.v[:1]) {
			return false
		}
		r.cur.X = r.at(r.v[0], 0).X
	case 'v':
		if !r.numbers(r.v[:1]) {
			return false
		}
		r.cur.Y = r.at(0, r.v[0]).Y
	}
	if cmd == 'm' {
		r.start, r.open = r.cur, true
		r.path.moveTo(r.cur)
	} else {
		r.path.lineTo(r.cur)
	}
	return true
}

// curve reads the arguments of C, S, Q or T; prev is the command before.
func (r *pathReader) curve(cmd, prev byte) bool {
	v := &r.v
	mirror := r.cur
	switch cmd {
	case 'c':
		if !r.numbers(v[:6]) {
			return false
		}
		c2, to := r.at(v[2], v[3]), r.at(v[4], v[5])
		r.path.cubicTo(r.at(v[0], v[1]), c2, to)
		r.ctl, r.cur = c2, to
	case 's':
		if !r.numbers(v[:4]) {
			return false
		}
		if prev == 'c' || prev == 's' {
			mirror = r.cur.mul(2).sub(r.ctl)
		}
		c2, to := r.at(v[0], v[1]), r.at(v[2], v[3])
		r.path.cubicTo(mirror, c2, to)
		r.ctl, r.cur = c2, to
	case 'q':
		if !r.numbers(v[:4]) {
			return false
		}
		c, to := r.at(v[0], v[1]), r.at(v[2], v[3])
		r.path.quadTo(c, to)
		r.ctl, r.cur = c, to
	case 't':
		if !r.numbers(v[:2]) {
			return false
		}
		if prev == 'q' || prev == 't' {
			mirror = r.cur.mul(2).sub(r.ctl)
		}
		to := r.at(v[0], v[1])
		r.path.quadTo(mirror, to)
		r.ctl, r.cur = mirror, to
	}
	return true
}

func (r *pathReader) arc() bool {
	v := &r.v
	if !r.numbers(v[:3]) {
		return false
	}
	large, ok1 := r.flag()
	sweep, ok2 := r.flag()
	if !ok1 || !ok2 || !r.numbers(v[3:5]) {
		return false
	}
	to := r.at(v[3], v[4])
	r.path.arcTo(r.cur, v[0], v[1], v[2], large, sweep, to)
	r.cur = to
	return true
}

// command reads the arguments of cmd, a lower-case letter, once.
func (r *pathReader) command(cmd, prev byte) bool {
	if cmd != 'm' && cmd != 'z' && !r.open {
		r.path.moveTo(r.start)
		r.open = true
	}
	switch cmd {
	case 'z':
		if r.open {
			r.path.close()
		}
		r.cur, r.open = r.start, false
		return true
	case 'm', 'l', 'h', 'v':
		return r.line(cmd)
	case 'a':
		return r.arc()
	}
	return r.curve(cmd, prev)
}

// parsePath reads path data: as much of it as is right, as SVG asks of a
// path with an error in it, and no more than limit verbs.
func parsePath(d string, limit int) Path {
	r := &pathReader{scanner: scanner{s: d}}
	var cmd, prev byte
	for len(r.path.Verbs) < limit && !r.done() {
		if c := r.s[r.i]; strings.IndexByte("MmLlHhVvCcSsQqTtAaZz", c) >= 0 {
			cmd = c
			r.i++
		} else if cmd == 0 || cmd == 'Z' || cmd == 'z' {
			break
		}
		lower := cmd | 0x20
		if len(r.path.Verbs) == 0 && lower != 'm' {
			break
		}
		r.rel = cmd >= 'a'
		if !r.command(lower, prev) {
			break
		}
		prev = lower
		// Further pairs after a move are lines.
		switch cmd {
		case 'M':
			cmd = 'L'
		case 'm':
			cmd = 'l'
		}
	}
	return r.path
}

func rectPath(x, y, w, h, rx, ry float64) Path {
	var p Path
	if !(w > 0 && h > 0) {
		return p
	}
	rx, ry = math.Min(rx, w/2), math.Min(ry, h/2)
	if !(rx > 0 && ry > 0) {
		p.moveTo(Point{x, y})
		p.lineTo(Point{x + w, y})
		p.lineTo(Point{x + w, y + h})
		p.lineTo(Point{x, y + h})
		p.close()
		return p
	}
	q := math.Pi / 2
	p.moveTo(Point{x + rx, y})
	p.lineTo(Point{x + w - rx, y})
	p.arc(Point{x + w - rx, y + ry}, rx, ry, 0, -q, q)
	p.lineTo(Point{x + w, y + h - ry})
	p.arc(Point{x + w - rx, y + h - ry}, rx, ry, 0, 0, q)
	p.lineTo(Point{x + rx, y + h})
	p.arc(Point{x + rx, y + h - ry}, rx, ry, 0, q, q)
	p.lineTo(Point{x, y + ry})
	p.arc(Point{x + rx, y + ry}, rx, ry, 0, 2*q, q)
	p.close()
	return p
}

func ellipsePath(c Point, rx, ry float64) Path {
	var p Path
	if !(rx > 0 && ry > 0) {
		return p
	}
	p.moveTo(Point{c.X + rx, c.Y})
	p.arc(c, rx, ry, 0, 0, 2*math.Pi)
	p.close()
	return p
}

func pointsPath(list string, closed bool, limit int) Path {
	var p Path
	s := &scanner{s: list}
	var v [2]float64
	for len(p.Verbs) < limit && s.numbers(v[:]) {
		if len(p.Verbs) == 0 {
			p.moveTo(Point{v[0], v[1]})
		} else {
			p.lineTo(Point{v[0], v[1]})
		}
	}
	if closed && len(p.Verbs) > 0 {
		p.close()
	}
	return p
}

// Bounds is the smallest box holding the path's outline, curves measured
// where they turn, not at their control points; ok is false for a path
// with no point.
func (p Path) Bounds() (Box, bool) {
	lo, hi := Point{math.Inf(1), math.Inf(1)}, Point{math.Inf(-1), math.Inf(-1)}
	add := func(q Point) {
		lo = Point{math.Min(lo.X, q.X), math.Min(lo.Y, q.Y)}
		hi = Point{math.Max(hi.X, q.X), math.Max(hi.Y, q.Y)}
	}
	var cur Point
	i := 0
	for _, v := range p.Verbs {
		switch v {
		case MoveTo, LineTo:
			cur = p.Points[i]
			add(cur)
			i++
		case QuadTo:
			c, to := p.Points[i], p.Points[i+1]
			c1, c2 := lerp(cur, c, 2.0/3), lerp(to, c, 2.0/3)
			cubicExtrema(cur, c1, c2, to, add)
			cur = to
			add(cur)
			i += 2
		case CubicTo:
			cubicExtrema(cur, p.Points[i], p.Points[i+1], p.Points[i+2], add)
			cur = p.Points[i+2]
			add(cur)
			i += 3
		case Close:
		}
	}
	if lo.X > hi.X || !lo.finite() || !hi.finite() {
		return Box{}, false
	}
	return Box{lo.X, lo.Y, hi.X - lo.X, hi.Y - lo.Y}, true
}

func cubicAt(p0, p1, p2, p3 Point, t float64) Point {
	u := 1 - t
	a, b, c, d := u*u*u, 3*u*u*t, 3*u*t*t, t*t*t
	return Point{a*p0.X + b*p1.X + c*p2.X + d*p3.X, a*p0.Y + b*p1.Y + c*p2.Y + d*p3.Y}
}

// cubicExtrema gives add the points where the curve turns back along x
// or y.
func cubicExtrema(p0, p1, p2, p3 Point, add func(Point)) {
	axis := func(a, b, c, d float64) {
		// The derivative is 3(qa·t² + qb·t + qc).
		qa, qb, qc := -a+3*b-3*c+d, 2*(a-2*b+c), b-a
		root := func(t float64) {
			if t > 0 && t < 1 {
				add(cubicAt(p0, p1, p2, p3, t))
			}
		}
		if math.Abs(qa) < 1e-12*(math.Abs(qb)+math.Abs(qc)) || qa == 0 {
			if qb != 0 {
				root(-qc / qb)
			}
			return
		}
		disc := qb*qb - 4*qa*qc
		if disc < 0 {
			return
		}
		sq := math.Sqrt(disc)
		root((-qb + sq) / (2 * qa))
		root((-qb - sq) / (2 * qa))
	}
	axis(p0.X, p1.X, p2.X, p3.X)
	axis(p0.Y, p1.Y, p2.Y, p3.Y)
}
