package shape

import (
	"fmt"
	"regexp"
	"strings"
)

// machine finds syllables in a sequence of categories as HarfBuzz's
// Ragel scanners do: at each place the longest of its rules that
// matches, the first listed of those as long, makes a syllable of its
// kind.
type machine struct {
	rules []*regexp.Regexp
	kinds []uint8
}

// rule is a rule of a machine written as Ragel writes it: names of
// categories or of other definitions, '.' or a space between parts,
// '|', '?', '*', '+' and parentheses; "any" is any category.
type rule struct {
	expr string
	kind uint8
}

// newMachine compiles rules whose names are categories or defs, the
// definitions one may use those before it.
//
//nolint:goconst // Ragel's name for any category
func newMachine(categories map[string]uint8, defs [][2]string, rules []rule) *machine {
	known := map[string]string{"any": `[\x{E000}-\x{E0FF}]`}
	for name, c := range categories {
		known[name] = fmt.Sprintf(`\x{%X}`, 0xE000+int(c))
	}
	for _, d := range defs {
		known[d[0]] = "(?:" + translate(d[1], known) + ")"
	}
	m := &machine{}
	for _, r := range rules {
		re := regexp.MustCompile("^(?:" + translate(r.expr, known) + ")")
		re.Longest()
		m.rules = append(m.rules, re)
		m.kinds = append(m.kinds, r.kind)
	}
	return m
}

// translate writes a Ragel expression as a Go regular expression.
func translate(expr string, known map[string]string) string {
	var b strings.Builder
	for i := 0; i < len(expr); {
		c := expr[i]
		switch {
		case c == '.' || c == ' ' || c == '\t':
			i++
		case isWordByte(c):
			j := i
			for j < len(expr) && isWordByte(expr[j]) {
				j++
			}
			x, ok := known[expr[i:j]]
			if !ok {
				panic("shape: no category or definition " + expr[i:j])
			}
			b.WriteString(x)
			i = j
		default:
			b.WriteByte(c)
			i++
		}
	}
	return b.String()
}

func isWordByte(c byte) bool {
	return c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9'
}

// find calls fn with each syllable of cats, its start, its end and its
// kind.
func (m *machine) find(cats []uint8, fn func(start, end int, kind uint8)) {
	var b strings.Builder
	for _, c := range cats {
		b.WriteRune(0xE000 + rune(c))
	}
	s := b.String()
	const width = 3 // the bytes of a character of the private use area
	for at := 0; at < len(cats); {
		best, kind := 0, uint8(0)
		for k, re := range m.rules {
			if loc := re.FindStringIndex(s[at*width:]); loc != nil && loc[1]/width > best {
				best, kind = loc[1]/width, m.kinds[k]
			}
		}
		best = max(best, 1)
		fn(at, at+best, kind)
		at += best
	}
}

// syllables gives each glyph the syllable m finds it in, numbered from
// 1 to 15 over and over, its kind below.
func (b *buffer) syllables(m *machine, cats []uint8, index []int) {
	serial := uint8(1)
	m.find(cats, func(start, end int, kind uint8) {
		from, to := start, end
		if index != nil {
			from = index[start]
			to = len(b.glyphs)
			if end < len(index) {
				to = index[end]
			}
		}
		for i := from; i < to; i++ {
			b.glyphs[i].syllable = serial<<4 | kind
		}
		serial++
		if serial == 16 {
			serial = 1
		}
	})
}

// eachSyllable calls fn with the start and end of each syllable.
func (b *buffer) eachSyllable(fn func(start, end int)) {
	for start := 0; start < len(b.glyphs); {
		end := start + 1
		for end < len(b.glyphs) && b.glyphs[end].syllable == b.glyphs[start].syllable {
			end++
		}
		fn(start, end)
		start = end
	}
}

// insertDottedCircles puts a dotted circle, of category cat and
// position pos, where the base of each syllable of kind broken would
// be, after a repha (of category repha, -1 for none), when the font has
// one, as HarfBuzz completes broken clusters.
func (b *buffer) insertDottedCircles(broken, cat, pos uint8, repha int) {
	has := false
	for _, g := range b.glyphs {
		if g.syllable&0x0F == broken {
			has = true
			break
		}
	}
	if !has {
		return
	}
	id := b.f.nominal(dottedCircle)
	if id == 0 {
		return
	}
	out := make([]Glyph, 0, len(b.glyphs)+4)
	last := uint8(0)
	for i := 0; i < len(b.glyphs); {
		g := b.glyphs[i]
		if g.syllable == last || g.syllable&0x0F != broken {
			out = append(out, g)
			i++
			continue
		}
		last = g.syllable
		dc := Glyph{ID: id, Char: dottedCircle, Cluster: g.Cluster, mask: g.mask, syllable: g.syllable, cat: cat,
			pos: pos, cc: 0}
		for repha >= 0 && i < len(b.glyphs) && b.glyphs[i].syllable == last && int(b.glyphs[i].cat) == repha {
			out = append(out, b.glyphs[i])
			i++
		}
		out = append(out, dc)
	}
	b.glyphs = out
}
