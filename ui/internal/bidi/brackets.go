package bidi

import (
	"slices"

	"golang.org/x/text/unicode/bidi"
)

// maxBrackets is how many opening brackets BD16 keeps open at most.
const maxBrackets = 63

// pairOf is the opening bracket bracket r pairs by, canonically.
func pairOf(r rune, opening bool) rune {
	if !opening {
		if m := []rune(bidi.ReverseString(string(r))); len(m) == 1 {
			r = m[0]
		}
	}
	switch r {
	case '〈':
		return '〈'
	case '〉':
		return '〉'
	default:
		return r
	}
}

// pairs are the bracket pairs of the sequence (BD16), by the place of
// their opening bracket.
func (s *sequence) pairs() [][2]int {
	type open struct {
		pair rune
		at   int
	}
	var stack []open
	var out [][2]int
	for k, i := range s.at {
		if s.t(k) != bidi.ON {
			continue
		}
		p, _ := bidi.LookupRune(s.p.text[i])
		if !p.IsBracket() {
			continue
		}
		if p.IsOpeningBracket() {
			if len(stack) == maxBrackets {
				break
			}
			stack = append(stack, open{pairOf(s.p.text[i], true), k})
			continue
		}
		want := pairOf(s.p.text[i], false)
		for j := len(stack) - 1; j >= 0; j-- {
			if stack[j].pair == want {
				out = append(out, [2]int{stack[j].at, k})
				stack = stack[:j]
				break
			}
		}
	}
	slices.SortFunc(out, func(a, b [2]int) int { return a[0] - b[0] })
	return out
}

// brackets resolves the paired brackets (N0).
func (s *sequence) brackets() {
	e := direction(s.level)
	for _, pr := range s.pairs() {
		inside := bidi.ON
		for k := pr[0] + 1; k < pr[1]; k++ {
			//nolint:exhaustive // the classes this rule reads; the others pass
			switch c := strongOf(s.t(k)); c {
			case bidi.L, bidi.R:
				if c == e {
					inside = e
				} else if inside == bidi.ON {
					inside = c
				}
			default:
			}
			if inside == e {
				break
			}
		}
		switch {
		case inside == bidi.ON:
			continue
		case inside != e:
			before := s.sos
			for k := pr[0] - 1; k >= 0; k-- {
				if c := strongOf(s.t(k)); c == bidi.L || c == bidi.R {
					before = c
					break
				}
			}
			if before == e {
				inside = e
			}
		}
		for _, k := range pr {
			s.set(k, inside)
			for j := k + 1; j < len(s.at) && s.p.initial[s.at[j]] == bidi.NSM; j++ {
				s.set(j, inside)
			}
		}
	}
}

// Mirror is r's Bidi_Mirroring_Glyph, the character drawn for it at an odd level (L4),
// and false when it has none.
func Mirror(r rune) (rune, bool) {
	i, ok := slices.BinarySearchFunc(mirrors[:], r, func(m [2]rune, r rune) int { return int(m[0] - r) })
	if !ok {
		return r, false
	}
	return mirrors[i][1], true
}
