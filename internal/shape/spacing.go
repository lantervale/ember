package shape

import (
	"slices"
	"unicode/utf8"

	"github.com/rivo/uniseg"
)

// cursiveScripts are the scripts CSS Text names cursive, whose letters
// letter-spacing does not part, as Chrome lists them.
//
//nolint:gochecknoglobals // a fixed list
var cursiveScripts = tags("arab", "syrc", "mong", "nko ", "mand", "phag", "rohg")

// Cursive reports whether script is cursive: letter-spacing follows none
// of its runs' clusters but their spaces.
func Cursive(script Tag) bool { return slices.Contains(cursiveScripts, script) }

// SpacedIn reports whether letter-spacing follows a cluster starting with
// r in a run of script: every cluster but those of a cursive run other
// than a space or a no-break space, and none starting with a default
// ignorable character (a bidi control, a zero-width space…), as Chrome
// spaces them.
func SpacedIn(script Tag, r rune) bool {
	return !DefaultIgnorable(r) && (r == ' ' || r == ' ' || !Cursive(script))
}

// ScriptAfter is the script the characters of no script at the start of
// s take after before: the last of before's characters' of a script, else
// the first of s's (Script).
func ScriptAfter(before, s string) Tag {
	for i := len(before); i > 0; {
		r, n := utf8.DecodeLastRuneInString(before[:i])
		i -= n
		if sc, ok := ScriptOf(r); ok {
			return sc
		}
	}
	return Script(s)
}

// EachSpaced calls fn with the end, in bytes, of each of s's grapheme
// clusters that letter-spacing follows (SpacedIn) after the text before:
// each cluster is of the script of its first character of one, else of
// the cluster before it, those at the start of the one ScriptAfter says.
func EachSpaced(before, s string, fn func(end int)) {
	script, state, at := ScriptAfter(before, s), -1, 0
	for len(s) > 0 {
		var c string
		c, s, _, state = uniseg.FirstGraphemeClusterInString(s, state)
		at += len(c)
		for _, r := range c {
			if sc, ok := ScriptOf(r); ok {
				script = sc
				break
			}
		}
		if r, _ := utf8.DecodeRuneInString(c); SpacedIn(script, r) {
			fn(at)
		}
	}
}

// LetterSpaced is how many of s's grapheme clusters letter-spacing
// follows after the text before (EachSpaced).
func LetterSpaced(before, s string) int {
	n := 0
	EachSpaced(before, s, func(int) { n++ })
	return n
}
