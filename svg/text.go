package svg

import (
	"math"
	"sort"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/gobolditalic"
	"golang.org/x/image/font/gofont/goitalic"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/gomonobold"
	"golang.org/x/image/font/gofont/gomonobolditalic"
	"golang.org/x/image/font/gofont/gomonoitalic"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

type textAnchor uint8

const elemTextPath = "textPath"

const (
	anchorStart textAnchor = iota
	anchorMiddle
	anchorEnd
)

// maxChars is the most characters one text element lays out.
const maxChars = 1 << 16

// faces are Go's fonts, the engine's: sans-serif, and monospace for a
// family that asks for it; regular, bold, italic and both.
//
//nolint:gochecknoglobals // parsed once, on the first text drawn
var faces struct {
	once  sync.Once
	sans  [4]*sfnt.Font
	mono  [4]*sfnt.Font
	lines map[*sfnt.Font]fontLines
}

// fontLines are where a face draws its underline and its line through,
// and how thick, in its units: post's and OS/2's.
type fontLines struct {
	under, underThickness, through, throughThickness float64
}

// readLines reads a TrueType file's fontLines from its tables.
func readLines(ttf []byte) fontLines {
	l := fontLines{under: -150, underThickness: 100, through: 500, throughThickness: 100}
	be16 := func(b []byte, at int) float64 { return float64(int16(uint16(b[at])<<8 | uint16(b[at+1]))) } //nolint:gosec // a signed field
	if len(ttf) < 12 {
		return l
	}
	n := int(ttf[4])<<8 | int(ttf[5])
	for i := range n {
		rec := 12 + 16*i
		if rec+16 > len(ttf) {
			break
		}
		at := int(ttf[rec+8])<<24 | int(ttf[rec+9])<<16 | int(ttf[rec+10])<<8 | int(ttf[rec+11])
		switch string(ttf[rec : rec+4]) {
		case "post":
			if at+12 <= len(ttf) {
				l.under, l.underThickness = be16(ttf, at+8), be16(ttf, at+10)
			}
		case "OS/2":
			if at+30 <= len(ttf) {
				l.throughThickness, l.through = be16(ttf, at+26), be16(ttf, at+28)
			}
		}
	}
	return l
}

func face(st *style) *sfnt.Font {
	faces.once.Do(func() {
		faces.lines = map[*sfnt.Font]fontLines{}
		for i, ttf := range [...][]byte{goregular.TTF, gobold.TTF, goitalic.TTF, gobolditalic.TTF} {
			faces.sans[i], _ = sfnt.Parse(ttf)
			faces.lines[faces.sans[i]] = readLines(ttf)
		}
		for i, ttf := range [...][]byte{gomono.TTF, gomonobold.TTF, gomonoitalic.TTF, gomonobolditalic.TTF} {
			faces.mono[i], _ = sfnt.Parse(ttf)
			faces.lines[faces.mono[i]] = readLines(ttf)
		}
	})
	i := 0
	if st.bold {
		i++
	}
	if st.italic {
		i += 2
	}
	for _, mono := range [...]string{"monospace", "mono", "courier", "consolas", "menlo", "monaco"} {
		if strings.Contains(st.fontFamily, mono) {
			return faces.mono[i]
		}
	}
	return faces.sans[i]
}

// textChar is a character of a text element, where it is placed and
// with the style of the element it is in.
type textChar struct {
	r              rune
	st             *style
	x, y, dx, dy   float64
	hasX, hasY     bool
	at             Point
	glyph          sfnt.GlyphIndex
	font           *sfnt.Font
	advance, scale float64
	// decorations are the lines drawn along it, of the elements it is in.
	decorations []*decoration
	// path is the textPath it is set along; placed, it is drawn through
	// along, or not at all past the path's ends when hidden.
	path   *textPath
	along  Matrix
	hidden bool
}

// decoration is the lines an element's text-decoration draws, in the
// element's style.
type decoration struct {
	st                         *style
	under, over, through, done bool
}

// textLayout gathers the characters of a text element.
type textLayout struct {
	b           *builder
	chars       []textChar
	decorations []*decoration
	path        *textPath
}

// collect appends the characters of n's content, laid out by st, then
// gives them the positions n's x, y, dx and dy attributes list, where an
// element inside gave them none.
func (l *textLayout) collect(n *node, st *style) {
	from := len(l.chars)
	for _, k := range n.kids {
		switch {
		case k.chars:
			l.text(k.text, st)
		case k.name == "tspan", k.name == "a", k.name == elemTextPath && l.path == nil:
			props := l.b.props(k)
			if strings.TrimSpace(props["display"]) == none {
				continue
			}
			inner := *st
			inner.apply(props, l.b)
			saved, savedPath := l.decorations, l.path
			if k.name == elemTextPath {
				if l.path = l.b.textPath(props, &inner); l.path == nil {
					continue
				}
			}
			l.decorate(props, &inner)
			l.collect(k, &inner)
			l.decorations, l.path = saved, savedPath
		}
	}
	props := l.b.props(n)
	chars := l.chars[from:]
	lists := func(k string, whole float64, set func(c *textChar, v float64)) {
		fields := strings.FieldsFunc(props[k], func(r rune) bool { return r == ',' || unicode.IsSpace(r) })
		for i, f := range fields[:min(len(fields), len(chars))] {
			if v, ok := length(f, lengthContext{whole: whole, em: st.fontSize}); ok {
				set(&chars[i], v)
			}
		}
	}
	lists("x", l.b.vw, func(c *textChar, v float64) {
		if !c.hasX {
			c.x, c.hasX = v, true
		}
	})
	lists("y", l.b.vh, func(c *textChar, v float64) {
		if !c.hasY {
			c.y, c.hasY = v, true
		}
	})
	lists("dx", l.b.vw, func(c *textChar, v float64) { c.dx += v })
	lists("dy", l.b.vh, func(c *textChar, v float64) { c.dy += v })
}

// text appends s, its white space collapsed as CSS's normal does.
func (l *textLayout) text(s string, st *style) {
	for _, r := range s {
		if len(l.chars) >= maxChars {
			return
		}
		switch r {
		case '\n', '\r', '\t':
			r = ' '
		}
		if r == ' ' && (len(l.chars) == 0 || l.chars[len(l.chars)-1].r == ' ') {
			continue
		}
		if r == utf8.RuneError {
			continue
		}
		l.chars = append(l.chars, textChar{r: r, st: st, decorations: l.decorations, path: l.path})
	}
}

// decorate adds the lines props's text-decoration asks for to those the
// characters inside are drawn with.
func (l *textLayout) decorate(props map[string]string, st *style) {
	v, ok := props["text-decoration"]
	if !ok {
		v, ok = props["text-decoration-line"]
	}
	if !ok {
		return
	}
	d := &decoration{st: st}
	for _, w := range strings.Fields(v) {
		switch w {
		case "underline":
			d.under = true
		case "overline":
			d.over = true
		case "line-through":
			d.through = true
		}
	}
	if d.under || d.over || d.through {
		l.decorations = append(l.decorations[:len(l.decorations):len(l.decorations)], d)
	}
}

// text appends the shapes of a text element: Go's glyphs, placed along
// its lines, a shape for each run of characters of one style.
func (b *builder) text(n *node, st style, m Matrix, out *[]Item) error {
	l := &textLayout{b: b}
	l.decorate(b.props(n), &st)
	l.collect(n, &st)
	chars := l.chars
	if k := len(chars); k > 0 && chars[k-1].r == ' ' {
		chars = chars[:k-1]
	}
	if len(chars) == 0 {
		return nil
	}
	var buf sfnt.Buffer
	place(chars, &buf)
	return b.glyphs(chars, &buf, m, out)
}

// place gives chars their glyphs and their places along their lines:
// one after another from where their positions start a chunk, each chunk
// moved as its text-anchor asks.
func place(chars []textChar, buf *sfnt.Buffer) {
	pen := Point{}
	chunk := 0
	anchor := func(end int) {
		if chunk >= end {
			return
		}
		width := pen.X - chars[chunk].at.X
		shift := 0.0
		switch chars[chunk].st.anchor {
		case anchorMiddle:
			shift = -width / 2
		case anchorEnd:
			shift = -width
		case anchorStart:
		}
		for i := chunk; i < end; i++ {
			chars[i].at.X += shift
		}
	}
	var prev *textChar
	for i := range chars {
		c := &chars[i]
		if i > 0 && c.path != chars[i-1].path {
			anchor(i)
			chunk = i
			pen = Point{}
			if c.path == nil {
				pen = chars[i-1].path.end
			}
			prev = nil
		}
		if c.hasX && c.path == nil {
			anchor(i)
			chunk = i
			pen.X = c.x
			prev = nil
		}
		if c.hasY && c.path == nil {
			pen.Y = c.y
		}
		pen = pen.add(Point{c.dx, c.dy})
		c.font = face(c.st)
		if c.font == nil || !(c.st.fontSize > 0) {
			c.at = pen
			continue
		}
		upem := fixed.Int26_6(c.font.UnitsPerEm()) << 6
		c.scale = c.st.fontSize / float64(c.font.UnitsPerEm()) / 64
		c.glyph, _ = c.font.GlyphIndex(buf, c.r)
		if prev != nil && prev.font == c.font && prev.scale == c.scale {
			if k, err := c.font.Kern(buf, prev.glyph, c.glyph, upem, font.HintingNone); err == nil {
				pen.X += float64(k) * c.scale
			}
		}
		adv, _ := c.font.GlyphAdvance(buf, c.glyph, upem, font.HintingNone)
		c.advance = float64(adv) * c.scale
		c.at = pen
		pen.X += c.advance + c.st.letterSpacing
		if c.r == ' ' {
			pen.X += c.st.wordSpacing
		}
		prev = c
	}
	anchor(len(chars))
	for i := range chars {
		if c := &chars[i]; c.path != nil {
			c.along, c.hidden = c.path.place(c.at, c.advance)
		}
	}
}

// glyphs appends the outlines of chars, a shape for each run of one
// style, painted with the text's bounding box for its units.
func (b *builder) glyphs(chars []textChar, buf *sfnt.Buffer, m Matrix, out *[]Item) error {
	var paths []Path
	var styles []*style
	for i := range chars {
		c := &chars[i]
		if c.font == nil || c.scale == 0 || c.st.hidden || c.hidden {
			continue
		}
		if len(styles) == 0 || styles[len(styles)-1] != c.st {
			paths, styles = append(paths, Path{}), append(styles, c.st)
		}
		p := &paths[len(paths)-1]
		upem := fixed.Int26_6(c.font.UnitsPerEm()) << 6
		segs, err := c.font.LoadGlyph(buf, c.glyph, upem, nil)
		if err != nil {
			continue
		}
		pt := func(a fixed.Point26_6) Point {
			if c.path != nil {
				return c.along.Apply(Point{float64(a.X) * c.scale, float64(a.Y) * c.scale})
			}
			return Point{c.at.X + float64(a.X)*c.scale, c.at.Y + float64(a.Y)*c.scale}
		}
		open := false
		for _, s := range segs {
			switch s.Op {
			case sfnt.SegmentOpMoveTo:
				if open {
					p.close()
				}
				p.moveTo(pt(s.Args[0]))
				open = true
			case sfnt.SegmentOpLineTo:
				p.lineTo(pt(s.Args[0]))
			case sfnt.SegmentOpQuadTo:
				p.Verbs = append(p.Verbs, QuadTo)
				p.Points = append(p.Points, pt(s.Args[0]), pt(s.Args[1]))
			case sfnt.SegmentOpCubeTo:
				p.Verbs = append(p.Verbs, CubicTo)
				p.Points = append(p.Points, pt(s.Args[0]), pt(s.Args[1]), pt(s.Args[2]))
			}
		}
		if open {
			p.close()
		}
	}
	var all Path
	for _, p := range paths {
		all.Verbs = append(all.Verbs, p.Verbs...)
		all.Points = append(all.Points, p.Points...)
	}
	bounds, _ := all.Bounds()
	under, through := decorationLines(chars, buf)
	for i := range under {
		paths, styles = append([]Path{under[i].path}, paths...), append([]*style{under[i].st}, styles...)
	}
	for _, l := range through {
		paths, styles = append(paths, l.path), append(styles, l.st)
	}
	for i, p := range paths {
		if len(p.Verbs) == 0 {
			continue
		}
		if err := b.count(len(p.Verbs)); err != nil {
			return err
		}
		if b.clipping {
			*out = append(*out, Item{Shape: b.clipShape(p, styles[i], m)})
		} else if s := b.paintShape(p, styles[i], m, bounds, false); s != nil {
			*out = append(*out, Item{Shape: s})
		}
	}
	return nil
}

// decorationLine is a line of a text-decoration: its outline and the
// style it is painted in.
type decorationLine struct {
	path Path
	st   *style
}

// decorationLines are the lines chars's decorations draw: under- and
// overlines, painted under the text, and lines through it, over it. Each
// runs along the characters it decorates on one line, at the place and
// thickness its element's face gives for its font size.
func decorationLines(chars []textChar, buf *sfnt.Buffer) (under, through []decorationLine) {
	for i := range chars {
		for _, d := range chars[i].decorations {
			if d.done {
				continue
			}
			d.done = true
			f := face(d.st)
			if f == nil || !(d.st.fontSize > 0) {
				continue
			}
			lines := faces.lines[f]
			k := d.st.fontSize / float64(f.UnitsPerEm())
			ascent := 0.0
			if m, err := f.Metrics(buf, fixed.Int26_6(f.UnitsPerEm())<<6, font.HintingNone); err == nil {
				ascent = float64(m.Ascent) / 64
			}
			for _, run := range decoratedRuns(chars, d) {
				line := func(top, thickness float64) Path {
					return rectPath(run.x0, run.base+top, run.x1-run.x0, math.Max(thickness, 0), 0, 0)
				}
				if d.under {
					under = append(under, decorationLine{line(-lines.under*k, lines.underThickness*k), run.st})
				}
				if d.over {
					under = append(under, decorationLine{line(-ascent*k, lines.underThickness*k), run.st})
				}
				if d.through {
					through = append(through, decorationLine{line(-lines.through*k, lines.throughThickness*k), run.st})
				}
			}
		}
	}
	return under, through
}

// decoratedRun is a stretch of characters of one style a decoration
// runs along: where it starts and ends, and its baseline.
type decoratedRun struct {
	x0, x1, base float64
	st           *style
}

// decoratedRuns are the stretches of chars d decorates, one per line of
// them and per style, painted in it as Chrome paints them.
func decoratedRuns(chars []textChar, d *decoration) []decoratedRun {
	var runs []decoratedRun
	open := false
	for i := range chars {
		c := &chars[i]
		has := false
		if c.path == nil && c.font != nil && !c.st.hidden {
			for _, cd := range c.decorations {
				has = has || cd == d
			}
		}
		if !has {
			open = false
			continue
		}
		end := c.at.X + c.advance
		if last := len(runs) - 1; open && c.st == runs[last].st && c.at.Y == runs[last].base && c.at.X >= runs[last].x0 {
			runs[last].x1 = math.Max(runs[last].x1, end)
			continue
		}
		runs = append(runs, decoratedRun{c.at.X, end, c.at.Y, c.st})
		open = true
	}
	return runs
}

// textPath is the outline a textPath sets its characters along: its
// points, each with how far along the outline it is.
type textPath struct {
	pts   []Point
	dist  []float64
	start float64
	end   Point
}

// textPath reads the path a textPath element names, through the
// transform of the path element, and where its text starts along it.
func (b *builder) textPath(props map[string]string, st *style) *textPath {
	var path Path
	if d, ok := props["path"]; ok {
		path = parsePath(d, MaxSegments-b.segments+1)
	} else {
		ref := b.target(props["href"])
		if ref == nil {
			return nil
		}
		switch ref.name {
		case elemPath, elemRect, elemCircle, elemEllipse, elemLine, elemPolyline, elemPolygon:
		default:
			return nil
		}
		refProps := b.props(ref)
		path = b.path(ref, refProps, st.fontSize)
		if t, ok := parseTransform(refProps["transform"]); ok && t.finite() {
			moved := Path{Verbs: path.Verbs, Points: make([]Point, len(path.Points))}
			for i, p := range path.Points {
				moved.Points[i] = t.Apply(p)
			}
			path = moved
		}
	}
	if !path.sound() || len(path.Verbs) == 0 {
		return nil
	}
	tp := &textPath{}
	for _, sub := range flatten(path, 8, 0) {
		for _, p := range sub.pts {
			if len(tp.pts) > 0 {
				tp.dist = append(tp.dist, tp.dist[len(tp.dist)-1]+p.sub(tp.pts[len(tp.pts)-1]).len())
			} else {
				tp.dist = append(tp.dist, 0)
			}
			tp.pts = append(tp.pts, p)
		}
	}
	if len(tp.pts) < 2 {
		return nil
	}
	tp.end = tp.pts[len(tp.pts)-1]
	if v, ok := props["startOffset"]; ok {
		tp.start, _ = length(v, lengthContext{whole: tp.dist[len(tp.dist)-1], em: st.fontSize})
	}
	return tp
}

// place is the transform taking a character's glyph, placed at at along
// the line of text, to the path: its middle on the path, turned with it,
// moved off it by at's y. hidden is set for a middle past the path's
// ends.
func (tp *textPath) place(at Point, advance float64) (Matrix, bool) {
	mid := tp.start + at.X + advance/2
	total := tp.dist[len(tp.dist)-1]
	if mid < 0 || mid > total {
		return Identity(), true
	}
	i := sort.SearchFloat64s(tp.dist, mid)
	i = min(max(i, 1), len(tp.pts)-1)
	a, b := tp.pts[i-1], tp.pts[i]
	seg := tp.dist[i] - tp.dist[i-1]
	t := 0.0
	if seg > 0 {
		t = (mid - tp.dist[i-1]) / seg
	}
	p := lerp(a, b, t)
	dir := b.sub(a)
	angle := math.Atan2(dir.Y, dir.X) * 180 / math.Pi
	return translate(p.X, p.Y).Mul(rotation(angle)).Mul(translate(-advance/2, at.Y)), false
}
