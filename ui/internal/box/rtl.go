package box

import (
	"strings"

	"github.com/rivo/uniseg"
	"golang.org/x/text/unicode/bidi"
)

// shownRTL is s drawn right to left, from the left: its graphemes in the
// reverse order, each bracket mirrored (UAX #9, L4).
func shownRTL(s string) string {
	var graphemes []string
	rest, state := s, -1
	var g string
	for len(rest) > 0 {
		g, rest, _, state = uniseg.FirstGraphemeClusterInString(rest, state)
		graphemes = append(graphemes, g)
	}
	var b strings.Builder
	b.Grow(len(s))
	for i := len(graphemes) - 1; i >= 0; i-- {
		shown := graphemes[i]
		if p, n := bidi.LookupString(shown); n == len(shown) && p.IsBracket() {
			shown = bidi.ReverseString(shown)
		}
		b.WriteString(shown)
	}
	return b.String()
}
