package shape

import (
	"sort"
	"unicode"
	"unicode/utf8"
)

// joining is a character's joining type, Syriac's Alaph and Dalath
// Rish groups apart, as HarfBuzz's Arabic shaper reads them; joinC
// joins as joinD does.
type joining uint8

const (
	joinU joining = iota
	joinL
	joinR
	joinD
	joinAlaph
	joinDalathRish
	joinT
	joinC = joinD
)

type joiningRange struct {
	lo, hi rune
	kind   joining
}

// joiningOf is r's joining type: ArabicShaping.txt's, else transparent
// for a mark or a format character, else none.
func joiningOf(r rune) joining {
	i := sort.Search(len(joinings), func(i int) bool { return joinings[i].hi >= r })
	if i < len(joinings) && joinings[i].lo <= r {
		return joinings[i].kind
	}
	if unicode.In(r, unicode.Mn, unicode.Me, unicode.Cf) {
		return joinT
	}
	return joinU
}

// JoinsAcross reports whether a letter between before and after may join
// with them: the character nearest it on either side, marks passed over,
// joins as Arabic letters do.
func JoinsAcross(before, after string) bool {
	for i := len(before); i > 0; {
		r, n := utf8.DecodeLastRuneInString(before[:i])
		i -= n
		if t := joiningOf(r); t != joinT {
			if t != joinU {
				return true
			}
			break
		}
	}
	for _, r := range after {
		if t := joiningOf(r); t != joinT {
			return t != joinU
		}
	}
	return false
}

// form is the joined form a letter takes, the feature that draws it.
type form uint8

const (
	isol form = iota
	fina
	fin2
	fin3
	medi
	med2
	initial
	none
)

// formTags are the features of the forms, in the order HarfBuzz applies
// them.
//
//nolint:gochecknoglobals // a fixed list
var formTags = [...]Tag{
	MakeTag("isol"), MakeTag("fina"), MakeTag("fin2"), MakeTag("fin3"), MakeTag("medi"), MakeTag("med2"),
	MakeTag("init"),
}

// joinStep is what a letter of a joining type does after a state: the
// form the letter before it takes, its own, and the next state.
type joinStep struct {
	prev, cur form
	next      uint8
}

// joinStates is HarfBuzz's Arabic joining machine, a row a state, a
// column a joining type (joinU to joinDalathRish).
//
//nolint:gochecknoglobals // a fixed table
var joinStates = [7][6]joinStep{
	// After a letter that does not join.
	{{none, none, 0}, {none, isol, 2}, {none, isol, 1}, {none, isol, 2}, {none, isol, 1}, {none, isol, 6}},
	// After one that joins on its right only, or an Alaph alone.
	{{none, none, 0}, {none, isol, 2}, {none, isol, 1}, {none, isol, 2}, {none, fin2, 5}, {none, isol, 6}},
	// After a dual- or left-joining letter alone, ready to join.
	{{none, none, 0}, {none, isol, 2}, {initial, fina, 1}, {initial, fina, 3}, {initial, fina, 4}, {initial, fina, 6}},
	// After a dual-joining one in its final form, ready to join.
	{{none, none, 0}, {none, isol, 2}, {medi, fina, 1}, {medi, fina, 3}, {medi, fina, 4}, {medi, fina, 6}},
	// After a final Alaph.
	{{none, none, 0}, {none, isol, 2}, {med2, isol, 1}, {med2, isol, 2}, {med2, fin2, 5}, {med2, isol, 6}},
	// After an Alaph in its second or third final form.
	{{none, none, 0}, {none, isol, 2}, {isol, isol, 1}, {isol, isol, 2}, {isol, fin2, 5}, {isol, isol, 6}},
	// After a Dalath or a Rish.
	{{none, none, 0}, {none, isol, 2}, {none, isol, 1}, {none, isol, 2}, {none, fin3, 5}, {none, isol, 6}},
}

// join gives each glyph of the run the mask of the form its character
// takes among those beside it, transparent ones passed over, the text
// round the run included; Mongolian's free variation selectors take
// their letter's.
func (b *buffer) join(masks *[len(formTags)]uint32) {
	forms := make([]form, len(b.glyphs))
	prev, state := -1, uint8(0)
	for _, r := range b.before {
		if t := joiningOf(r); t != joinT {
			state = joinStates[state][t].next
			break
		}
	}
	for i := range b.glyphs {
		forms[i] = none
		t := joiningOf(b.glyphs[i].Char)
		if t == joinT {
			continue
		}
		step := joinStates[state][t]
		if step.prev != none && prev >= 0 {
			forms[prev] = step.prev
		}
		forms[i] = step.cur
		prev, state = i, step.next
	}
	for _, r := range b.after {
		if t := joiningOf(r); t != joinT {
			if step := joinStates[state][t]; step.prev != none && prev >= 0 {
				forms[prev] = step.prev
			}
			break
		}
	}
	if b.plan.script == mong {
		for i := 1; i < len(forms); i++ {
			if c := b.glyphs[i].Char; c >= 0x180B && c <= 0x180D || c == 0x180F {
				forms[i] = forms[i-1]
			}
		}
	}
	for i, f := range forms {
		if f != none {
			b.glyphs[i].mask |= masks[f]
		}
	}
}

// arabicLigature is a ligature HarfBuzz's Arabic fallback shaping forms
// of the characters parts, the last 0 for a ligature of two.
type arabicLigature struct {
	parts [3]rune
	lig   rune
}
