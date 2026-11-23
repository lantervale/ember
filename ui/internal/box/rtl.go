package box

import (
	"strings"
	"unicode/utf8"

	"github.com/rivo/uniseg"

	"github.com/lantervale/ember/ui/internal/bidi"
)

// shownRTL is s drawn right to left, from the left: its graphemes in the
// reverse order, each character of a mirrored glyph drawn as it (UAX #9, L4).
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
		r, n := utf8.DecodeRuneInString(shown)
		if m, ok := bidi.Mirror(r); ok {
			b.WriteRune(m)
			shown = shown[n:]
		}
		b.WriteString(shown)
	}
	return b.String()
}
