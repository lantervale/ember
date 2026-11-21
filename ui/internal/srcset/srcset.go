// Package srcset reads the srcset of an img or a source as HTML does,
// and chooses its picture for a screen as Chrome does.
package srcset

import (
	"cmp"
	"slices"
	"strconv"
	"strings"
)

// Candidate is one picture a srcset offers: its address and the width
// in pixels or the density it says it has, 0 for what it doesn't say.
type Candidate struct {
	URL     string
	Width   int
	Density float64
}

const whitespace = "\t\n\f\r "

// Parse reads a srcset as HTML's parser does: an address runs to the
// next whitespace, so it may hold commas, and a candidate whose
// descriptors are no valid set is dropped.
func Parse(text string) []Candidate {
	var set []Candidate
	each(text, func(at, end int, descriptors []string) {
		if c, ok := candidate(text[at:end], descriptors); ok {
			set = append(set, c)
		}
	})
	return set
}

// Replace is text with the address of each of its entries, valid or
// not, replaced by what f gives for it, the rest as written.
func Replace(text string, f func(url string) string) string {
	var b strings.Builder
	last := 0
	each(text, func(at, end int, _ []string) {
		b.WriteString(text[last:at])
		b.WriteString(f(text[at:end]))
		last = end
	})
	b.WriteString(text[last:])
	return b.String()
}

// each calls fn with each entry of text: where its address starts and
// ends, and its descriptors.
func each(text string, fn func(at, end int, descriptors []string)) {
	at := 0
	for {
		at += len(text[at:]) - len(strings.TrimLeft(text[at:], whitespace+","))
		if at == len(text) {
			return
		}
		end := at + strings.IndexAny(text[at:], whitespace)
		if end < at {
			end = len(text)
		}
		url := text[at:end]
		if bare := strings.TrimRight(url, ","); bare != url {
			fn(at, at+len(bare), nil)
			at = end
			continue
		}
		descriptors, read := tokens(text[end:])
		fn(at, end, descriptors)
		at = end + read
	}
}

// tokens reads the descriptors after an address up to the comma that
// ends them, one in parentheses holding its commas, and how much of
// text they took.
func tokens(text string) ([]string, int) {
	var descriptors []string
	var current strings.Builder
	flush := func() {
		if current.Len() > 0 {
			descriptors = append(descriptors, current.String())
			current.Reset()
		}
	}
	inParens := false
	for i, c := range text {
		switch {
		case inParens:
			current.WriteRune(c)
			inParens = c != ')'
		case strings.ContainsRune(whitespace, c):
			flush()
		case c == ',':
			flush()
			return descriptors, i + 1
		default:
			current.WriteRune(c)
			inParens = c == '('
		}
	}
	flush()
	return descriptors, len(text)
}

// candidate is url with its descriptors: a width, a density, or a width
// and a height, each once.
func candidate(url string, descriptors []string) (Candidate, bool) {
	c := Candidate{URL: url}
	hasDensity, hasHeight := false, false
	for _, d := range descriptors {
		value := d[:len(d)-1]
		switch d[len(d)-1] {
		case 'w':
			n, ok := integer(value)
			if !ok || n == 0 || c.Width > 0 || hasDensity {
				return c, false
			}
			c.Width = n
		case 'x':
			v, ok := float(value)
			if !ok || v < 0 || c.Width > 0 || hasDensity || hasHeight {
				return c, false
			}
			c.Density, hasDensity = v, true
		case 'h':
			n, ok := integer(value)
			if !ok || n == 0 || hasHeight || hasDensity {
				return c, false
			}
			hasHeight = true
		default:
			return c, false
		}
	}
	return c, !hasHeight || c.Width > 0
}

// integer reads a valid non-negative integer of HTML: digits alone.
func integer(s string) (int, bool) {
	if s == "" || strings.Trim(s, "0123456789") != "" {
		return 0, false
	}
	n, err := strconv.Atoi(s)
	return n, err == nil
}

// float reads a valid floating-point number of HTML: a sign only for
// less than zero, digits on one side of the point at least, and an
// exponent.
func float(s string) (float64, bool) {
	mantissa, exponent, hasExponent := strings.Cut(strings.ToLower(strings.TrimPrefix(s, "-")), "e")
	whole, fraction, hasPoint := strings.Cut(mantissa, ".")
	if len(exponent) > 0 && (exponent[0] == '+' || exponent[0] == '-') {
		exponent = exponent[1:]
	}
	switch {
	case whole != "" && !digits(whole), !hasPoint && whole == "", hasPoint && !digits(fraction):
		return 0, false
	case hasExponent && !digits(exponent):
		return 0, false
	}
	v, err := strconv.ParseFloat(s, 64)
	return v, err == nil
}

func digits(s string) bool { return s != "" && strings.Trim(s, "0123456789") == "" }

// Choose is the picture of set an img shows on a screen of scale device
// pixels a point, its slot size points wide, and the density it has
// there, as Chrome chooses: each candidate's density, a width over the
// slot's, src one of 1 while no candidate says a width; the least
// density not under scale wins, else the greatest; then the densest
// candidate held wins over it, a picture there is no need to fetch.
// "" is none.
func Choose(set []Candidate, src string, size, scale float64, held func(url string) bool) (string, float64) {
	type option struct {
		url     string
		density float64
	}
	options := make([]option, 0, len(set)+1)
	widths := false
	for _, c := range set {
		o := option{c.URL, c.Density}
		switch {
		case c.Width > 0:
			o.density, widths = float64(c.Width)/size, true
		case o.density == 0:
			o.density = 1
		}
		options = append(options, o)
	}
	if src != "" && !widths {
		options = append(options, option{src, 1})
	}
	if len(options) == 0 {
		return "", 0
	}
	slices.SortStableFunc(options, func(a, b option) int { return cmp.Compare(a.density, b.density) })
	options = slices.CompactFunc(options, func(a, b option) bool { return a.density == b.density })
	i := slices.IndexFunc(options, func(o option) bool { return o.density >= scale })
	if i < 0 {
		i = len(options) - 1
	}
	for j := len(options) - 1; j > i; j-- {
		if held(options[j].url) {
			return options[j].url, options[j].density
		}
	}
	return options[i].url, options[i].density
}
