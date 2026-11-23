package svg

import (
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
	once sync.Once
	sans [4]*sfnt.Font
	mono [4]*sfnt.Font
}

func face(st *style) *sfnt.Font {
	faces.once.Do(func() {
		for i, ttf := range [...][]byte{goregular.TTF, gobold.TTF, goitalic.TTF, gobolditalic.TTF} {
			faces.sans[i], _ = sfnt.Parse(ttf)
		}
		for i, ttf := range [...][]byte{gomono.TTF, gomonobold.TTF, gomonoitalic.TTF, gomonobolditalic.TTF} {
			faces.mono[i], _ = sfnt.Parse(ttf)
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
}

// textLayout gathers the characters of a text element.
type textLayout struct {
	b     *builder
	chars []textChar
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
		case k.name == "tspan", k.name == "a":
			props := l.b.props(k)
			if strings.TrimSpace(props["display"]) == none {
				continue
			}
			inner := *st
			inner.apply(props, l.b)
			l.collect(k, &inner)
		case k.name == "textPath":
			l.b.unsupported("textPath")
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
		l.chars = append(l.chars, textChar{r: r, st: st})
	}
}

// text appends the shapes of a text element: Go's glyphs, placed along
// its lines, a shape for each run of characters of one style.
func (b *builder) text(n *node, st style, m Matrix, out *[]Item) error {
	l := &textLayout{b: b}
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
		if c.hasX {
			anchor(i)
			chunk = i
			pen.X = c.x
			prev = nil
		}
		if c.hasY {
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
		pen.X += c.advance
		prev = c
	}
	anchor(len(chars))
}

// glyphs appends the outlines of chars, a shape for each run of one
// style, painted with the text's bounding box for its units.
func (b *builder) glyphs(chars []textChar, buf *sfnt.Buffer, m Matrix, out *[]Item) error {
	var paths []Path
	var styles []*style
	for i := range chars {
		c := &chars[i]
		if c.font == nil || c.scale == 0 || c.st.hidden {
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
