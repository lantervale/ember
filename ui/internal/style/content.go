package style

import (
	"strconv"
	"strings"
)

// ContentKind says what a content property asks for.
type ContentKind uint8

const (
	// ContentNormal is normal: no box for ::before and ::after, the
	// marker's own text for ::marker.
	ContentNormal ContentKind = iota
	// ContentNone is none: no box.
	ContentNone
	// ContentItems is a list of strings, counters, quotes and pictures.
	ContentItems
)

// Content is a computed content property.
type Content struct {
	Kind  ContentKind
	Items []ContentItem
}

// ContentItemKind is one piece of generated content.
type ContentItemKind uint8

const (
	// ContentString is a string, attr() included: Text.
	ContentString ContentItemKind = iota
	// ContentImage is a url(): Text is its source.
	ContentImage
	// ContentCounter is counter(Name, Style); ContentCounters is
	// counters(Name, Text, Style), Text between the values.
	ContentCounter
	ContentCounters
	ContentOpenQuote
	ContentCloseQuote
	ContentNoOpenQuote
	ContentNoCloseQuote
)

// ContentItem is one piece of a content property.
type ContentItem struct {
	Kind  ContentItemKind
	Text  string
	Name  string
	Style ListStyle
}

// Dynamic reports content whose text depends on where its element stands
// in the document: counters and quotes.
func (c Content) Dynamic() bool {
	for _, it := range c.Items {
		if it.Kind != ContentString && it.Kind != ContentImage {
			return true
		}
	}
	return false
}

// Counter is one counter of a counter-reset, counter-increment or
// counter-set: its name and the value given or left out (0 for a reset
// or a set, 1 for an increment). Reversed is counter-reset's
// reversed(name), which counts down to 1 when no value is given
// (HasValue).
type Counter struct {
	Name     string
	Value    int
	Reversed bool
	HasValue bool
}

// Format is n written as l counts, the way counter() and a list item's
// number show it: a bullet for disc, circle and square whatever n is,
// nothing for none, and decimal where l can't write n.
func (l ListStyle) Format(n int) string {
	switch l {
	case ListDisc:
		return "•"
	case ListCircle:
		return "◦"
	case ListSquare:
		return "▪"
	case ListNone, ListDisclosureClosed, ListDisclosureOpen:
		return ""
	case ListDecimalLeadingZero:
		switch {
		case n >= 0 && n < 10:
			return "0" + strconv.Itoa(n)
		case n < 0 && n > -10:
			return "-0" + strconv.Itoa(-n)
		}
	case ListLowerAlpha, ListUpperAlpha:
		if n >= 1 {
			return alphabetic(n, l == ListUpperAlpha)
		}
	case ListLowerRoman, ListUpperRoman:
		if n >= 1 && n <= 3999 {
			r := roman(n)
			if l == ListLowerRoman {
				r = strings.ToLower(r)
			}
			return r
		}
	case ListDecimal:
	}
	return strconv.Itoa(n)
}

// Numeric reports a list style that writes its item's number.
func (l ListStyle) Numeric() bool {
	switch l {
	case ListDecimal, ListDecimalLeadingZero, ListLowerAlpha, ListUpperAlpha, ListLowerRoman, ListUpperRoman:
		return true
	case ListDisc, ListNone, ListDisclosureClosed, ListDisclosureOpen, ListCircle, ListSquare:
	}
	return false
}

func alphabetic(n int, upper bool) string {
	first := byte('a')
	if upper {
		first = 'A'
	}
	var out []byte
	for n > 0 {
		n--
		out = append([]byte{first + byte(n%26)}, out...)
		n /= 26
	}
	return string(out)
}

func roman(n int) string {
	values := [...]int{1000, 900, 500, 400, 100, 90, 50, 40, 10, 9, 5, 4, 1}
	marks := [...]string{"M", "CM", "D", "CD", "C", "XC", "L", "XL", "X", "IX", "V", "IV", "I"}
	var b strings.Builder
	for i, v := range values {
		for n >= v {
			b.WriteString(marks[i])
			n -= v
		}
	}
	return b.String()
}
