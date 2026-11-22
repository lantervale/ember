package outline

import (
	"math"
	"strconv"
	"strings"

	"github.com/lantervale/ember/geom"
)

// Parse reads SVG path data d, as CSS's path() holds it: the outline it
// fills, and d again with every command absolute and its numbers spaced,
// as Chrome computes it. ok is false unless all of d is right and it
// draws something.
func Parse(d string) (Path, string, bool) {
	r := reader{s: d}
	if !r.read() || r.b.Len() == 0 {
		return Path{}, "", false
	}
	r.end()
	return r.path, r.b.String(), true
}

// reader reads path data, the outline into path and the absolute data
// into b.
type reader struct {
	s    string
	i    int
	b    strings.Builder
	path Path
	// open is the subpath being drawn; cur where it is, start where it
	// began, ctl the last control point, which S and T mirror.
	open            []geom.Vec2
	cur, start, ctl geom.Vec2
	last            byte
}

func (r *reader) skip() {
	for r.i < len(r.s) && strings.IndexByte(" \t\r\n\f,", r.s[r.i]) >= 0 {
		r.i++
	}
}

func (r *reader) done() bool {
	r.skip()
	return r.i >= len(r.s)
}

// number reads one number as path data writes them.
func (r *reader) number() (float64, bool) {
	r.skip()
	j := r.i
	if j < len(r.s) && (r.s[j] == '+' || r.s[j] == '-') {
		j++
	}
	digits := func() bool {
		k := j
		for j < len(r.s) && r.s[j] >= '0' && r.s[j] <= '9' {
			j++
		}
		return j > k
	}
	whole := digits()
	frac := false
	if j < len(r.s) && r.s[j] == '.' {
		j++
		frac = digits()
	}
	if !whole && !frac {
		return 0, false
	}
	if j < len(r.s) && (r.s[j] == 'e' || r.s[j] == 'E') {
		k := j
		j++
		if j < len(r.s) && (r.s[j] == '+' || r.s[j] == '-') {
			j++
		}
		if !digits() {
			j = k
		}
	}
	v, err := strconv.ParseFloat(r.s[r.i:j], 64)
	r.i = j
	return v, err == nil && !math.IsInf(v, 0)
}

// flag reads an arc's flag: one 0 or 1, which may touch what follows.
func (r *reader) flag() (float64, bool) {
	r.skip()
	if r.i < len(r.s) && (r.s[r.i] == '0' || r.s[r.i] == '1') {
		r.i++
		return float64(r.s[r.i-1] - '0'), true
	}
	return 0, false
}

func (r *reader) numbers(n int) ([]float64, bool) {
	out := make([]float64, n)
	for i := range out {
		v, ok := r.number()
		if !ok {
			return nil, false
		}
		out[i] = v
	}
	return out, true
}

// write adds a command and its numbers to the absolute data.
func (r *reader) write(cmd byte, v ...float64) {
	if r.b.Len() > 0 {
		r.b.WriteByte(' ')
	}
	r.b.WriteByte(cmd)
	for _, x := range v {
		r.b.WriteByte(' ')
		r.b.WriteString(strconv.FormatFloat(float64(float32(x)), 'f', -1, 32))
	}
}

// end closes the subpath being drawn into the outline.
func (r *reader) end() {
	if len(r.open) > 2 {
		r.path.Contours = append(r.path.Contours, r.open)
	}
	r.open = nil
}

func (r *reader) read() bool {
	var cmd byte
	for !r.done() {
		if c := r.s[r.i]; strings.IndexByte("MmLlHhVvCcSsQqTtAaZz", c) >= 0 {
			cmd = c
			r.i++
		} else if cmd == 0 || cmd == 'Z' || cmd == 'z' {
			return false
		}
		if r.b.Len() == 0 && cmd|0x20 != 'm' {
			return false
		}
		if !r.command(cmd) {
			return false
		}
		switch cmd {
		case 'M':
			cmd = 'L'
		case 'm':
			cmd = 'l'
		}
	}
	return true
}

func (r *reader) command(cmd byte) bool {
	up := cmd &^ 0x20
	rel := cmd != up
	at := func(x, y float64) geom.Vec2 {
		p := geom.V2(float32(x), float32(y))
		if rel {
			return p.Add(r.cur)
		}
		return p
	}
	if r.open == nil && up != 'M' {
		r.open = []geom.Vec2{r.cur}
	}
	smooth := r.last
	r.last = up
	switch up {
	case 'Z':
		r.write('Z')
		r.end()
		r.cur = r.start
		return true
	case 'H', 'V':
		return r.straight(up, rel)
	case 'A':
		return r.arc(at)
	}
	n := map[byte]int{'M': 2, 'L': 2, 'T': 2, 'C': 6, 'S': 4, 'Q': 4}[up]
	v, ok := r.numbers(n)
	if !ok {
		return false
	}
	pts := make([]geom.Vec2, 0, n/2)
	flat := make([]float64, 0, n)
	for i := 0; i < n; i += 2 {
		p := at(v[i], v[i+1])
		pts, flat = append(pts, p), append(flat, float64(p.X), float64(p.Y))
	}
	r.write(up, flat...)
	mirrored := func(kinds string) geom.Vec2 {
		if strings.IndexByte(kinds, smooth) >= 0 {
			return r.cur.Scale(2).Sub(r.ctl)
		}
		return r.cur
	}
	switch up {
	case 'M':
		r.end()
		r.open, r.start, r.cur = []geom.Vec2{pts[0]}, pts[0], pts[0]
	case 'L':
		r.open, r.cur = append(r.open, pts[0]), pts[0]
	case 'T':
		r.quad(mirrored("QT"), pts[0])
	case 'C':
		r.cubic(pts[0], pts[1], pts[2])
	case 'S':
		r.cubic(mirrored("CS"), pts[0], pts[1])
	case 'Q':
		r.quad(pts[0], pts[1])
	}
	return true
}

// straight reads the argument of H or V.
func (r *reader) straight(up byte, rel bool) bool {
	v, ok := r.number()
	if !ok {
		return false
	}
	p, axis := r.cur, &r.cur.X
	if up == 'V' {
		axis = &r.cur.Y
	}
	k := float32(v)
	if rel {
		k += *axis
	}
	if up == 'H' {
		p.X = k
	} else {
		p.Y = k
	}
	r.write(up, float64(k))
	r.open, r.cur = append(r.open, p), p
	return true
}

func (r *reader) cubic(c1, c2, p geom.Vec2) {
	r.open = cubicPoints(r.open, r.cur, c1, c2, p)
	r.ctl, r.cur = c2, p
}

func (r *reader) quad(c, p geom.Vec2) {
	r.open = cubicPoints(r.open, r.cur, r.cur.Add(c.Sub(r.cur).Scale(2.0/3)), p.Add(c.Sub(p).Scale(2.0/3)), p)
	r.ctl, r.cur = c, p
}

// arc reads an A command, drawn as SVG 1.1's implementation notes F.6.5
// and F.6.6 have it.
func (r *reader) arc(at func(x, y float64) geom.Vec2) bool {
	var v [7]float64
	for i := range v {
		var ok bool
		if i == 3 || i == 4 {
			v[i], ok = r.flag()
		} else {
			v[i], ok = r.number()
		}
		if !ok {
			return false
		}
	}
	to := at(v[5], v[6])
	r.write('A', v[0], v[1], v[2], v[3], v[4], float64(to.X), float64(to.Y))
	from := r.cur
	r.cur = to
	rx, ry := math.Abs(v[0]), math.Abs(v[1])
	if from == to {
		return true
	}
	if rx == 0 || ry == 0 {
		r.open = append(r.open, to)
		return true
	}
	phi := v[2] * math.Pi / 180
	sin, cos := math.Sincos(phi)
	dx, dy := float64(from.X-to.X)/2, float64(from.Y-to.Y)/2
	x1, y1 := cos*dx+sin*dy, -sin*dx+cos*dy
	if l := x1*x1/(rx*rx) + y1*y1/(ry*ry); l > 1 {
		rx, ry = rx*math.Sqrt(l), ry*math.Sqrt(l)
	}
	num := rx*rx*ry*ry - rx*rx*y1*y1 - ry*ry*x1*x1
	k := math.Sqrt(max(num, 0) / (rx*rx*y1*y1 + ry*ry*x1*x1))
	if (v[3] == 1) == (v[4] == 1) {
		k = -k
	}
	cx1, cy1 := k*rx*y1/ry, -k*ry*x1/rx
	cx := cos*cx1 - sin*cy1 + float64(from.X+to.X)/2
	cy := sin*cx1 + cos*cy1 + float64(from.Y+to.Y)/2
	angle := func(ux, uy, vx, vy float64) float64 { return math.Atan2(ux*vy-uy*vx, ux*vx+uy*vy) }
	a0 := angle(1, 0, (x1-cx1)/rx, (y1-cy1)/ry)
	sweep := angle((x1-cx1)/rx, (y1-cy1)/ry, (-x1-cx1)/rx, (-y1-cy1)/ry)
	if v[4] == 0 && sweep > 0 {
		sweep -= 2 * math.Pi
	} else if v[4] == 1 && sweep < 0 {
		sweep += 2 * math.Pi
	}
	pts := arcPoints(nil, geom.V2(float32(cx), float32(cy)), float32(rx), float32(ry), phi, a0, sweep)
	r.open = append(r.open, pts[1:len(pts)-1]...)
	r.open = append(r.open, to)
	return true
}
