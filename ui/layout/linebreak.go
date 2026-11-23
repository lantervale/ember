package layout

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/lantervale/ember/ui/internal/style"
)

// A character a line-break strictness breaks around as around an
// ideograph is replaced by one before UAX #14 reads the text: uniseg's
// rules are strict's, which ICU's other rules relax by classing these
// characters ID. Both ideographs take as many bytes as what they replace.
const (
	iterationMarks = "々〻ゝゞヽヾ"
	ideograph3     = "漢"
	ideograph4     = "\U00020000"
)

// asIdeograph reports whether s breaks around r as around an ideograph:
// small kana and prolonged sound marks (UAX #14's CJ) but in strict,
// iteration marks in loose, the wave dash and the double hyphen in the
// CJK rules, and centred punctuation in loose CJK.
func asIdeograph(r rune, s style.Strictness) bool {
	switch {
	case s == style.StrictnessStrict:
		return false
	case conditionalJapanese(r):
		return true
	case s == style.StrictnessNormal:
		return false
	case r == '〜' || r == '゠':
		return s == style.StrictnessNormalCJK || s == style.StrictnessLooseCJK
	case strings.ContainsRune(iterationMarks, r):
		return s == style.StrictnessLoose || s == style.StrictnessLooseCJK
	case strings.ContainsRune("・：；‼⁇⁈⁉！？", r):
		return s == style.StrictnessLooseCJK
	}
	return false
}

// conditionalJapanese reports whether r is of UAX #14's class CJ.
func conditionalJapanese(r rune) bool {
	switch {
	case r >= 0x3041 && r <= 0x30FC:
		return strings.ContainsRune("ぁぃぅぇぉっゃゅょゎゕゖァィゥェォッャュョヮヵヶー", r)
	case r >= 0x31F0 && r <= 0x31FF, r >= 0xFF67 && r <= 0xFF70:
		return true
	case r >= 0x1B132:
		return r == 0x1B132 || r >= 0x1B150 && r <= 0x1B152 || r == 0x1B155 || r >= 0x1B164 && r <= 0x1B167
	}
	return false
}

// startsNoLine reports whether s keeps a line from starting with letter
// r, which word-break: break-all leaves it: a small kana or prolonged
// sound mark in strict, an iteration mark but in loose.
func startsNoLine(r rune, s style.Strictness) bool {
	return (conditionalJapanese(r) || strings.ContainsRune(iterationMarks, r)) && !asIdeograph(r, s)
}

// writeStrict writes text to b with the characters s breaks around as
// ideographs replaced by one, its first character by first's, and
// reports whether it replaced any.
func writeStrict(b *strings.Builder, text string, s, first style.Strictness) bool {
	if s == style.StrictnessStrict && first == style.StrictnessStrict || !mayBreakApart(text) {
		b.WriteString(text)
		return false
	}
	replaced := false
	for at, r := range text {
		by := s
		if at == 0 {
			by = first
		}
		switch {
		case !asIdeograph(r, by):
			b.WriteRune(r)
		case utf8.RuneLen(r) == 4:
			b.WriteString(ideograph4)
			replaced = true
		default:
			b.WriteString(ideograph3)
			replaced = true
		}
	}
	return replaced
}

// strictnesses are the strictness each run's text breaks by, and the one
// its first character does: that of the text before it, as Chrome breaks
// between two elements by the rules of the text before.
func (c *inlineContent) strictnesses() ([]style.Strictness, []style.Strictness) {
	own, first := make([]style.Strictness, len(c.runs)), make([]style.Strictness, len(c.runs))
	var last *textRun
	for i, r := range c.runs {
		if r.atomic != nil {
			last = nil
			continue
		}
		own[i], first[i] = r.st.Strictness, r.st.Strictness
		if last != nil {
			first[i] = last.st.Strictness
		}
		if r.text != "" {
			last = r
		}
	}
	return own, first
}

// looseBreaks calls fn with each byte of text a line may break at by
// the rules of loose that are no change of class, strictness(pos) the
// strictness at a byte: between two inseparable characters, and in
// loose CJK before a CJK hyphen after an ideograph, before a postfix
// after an ideograph or a digit, before a full-width prefix after a
// letter and after it before a letter, a digit, an ideograph or an
// opening bracket. It calls keep with each byte a line may not break at
// though UAX #14 breaks there: after a CJK hyphen a line may break
// before, which stays with a letter after it as a word's first hyphen.
// classed is text as writeStrict wrote it.
func looseBreaks(text, classed string, strictness func(pos int) style.Strictness, fn, keep func(pos int)) {
	if !mayBreakApart(text) {
		return
	}
	var a rune
	hyphenFirst := false
	for pos, b := range text {
		if pos == 0 {
			a = b
			continue
		}
		s := strictness(pos)
		if s != style.StrictnessLoose && s != style.StrictnessLooseCJK {
			a, hyphenFirst = b, false
			continue
		}
		before, _ := utf8.DecodeLastRuneInString(classed[:pos])
		after, _ := utf8.DecodeRuneInString(classed[pos:])
		breaks := inseparable(a) && inseparable(b)
		if s == style.StrictnessLooseCJK {
			switch {
			case b == '‐' || b == '–':
				breaks = ideographic(before)
			case strings.ContainsRune("％℃‰′″￠", b):
				breaks = ideographic(before) || unicode.IsDigit(a)
			case b == '°':
				breaks = ideographic(before)
			case fullPrefix(b):
				breaks = unicode.IsLetter(a) && !ideographic(before)
			}
			if fullPrefix(a) {
				breaks = breaks || unicode.IsLetter(after) || unicode.IsDigit(b) || unicode.Is(unicode.Ps, b)
			}
			if hyphenFirst && unicode.IsLetter(b) && !ideographic(after) {
				keep(pos)
			}
		}
		if breaks {
			fn(pos)
		}
		hyphenFirst = breaks && (b == '‐' || b == '–')
		a = b
	}
}

// mayBreakApart reports whether text has a character from U+2000 on,
// where every character the strictnesses tell apart is.
func mayBreakApart(text string) bool {
	for i := 0; i < len(text); i++ {
		if text[i] >= 0xE2 {
			return true
		}
	}
	return false
}

// anywhereEnds calls add with the start of each text run after one of
// line-break: anywhere: a line may break where such text ends, as in
// Chrome, which breaks between two elements by the rules of the text
// before.
func (c *inlineContent) anywhereEnds(add func(pos int), starts []int) {
	var last *textRun
	for i, r := range c.runs {
		switch {
		case r.atomic != nil:
			last = nil
			continue
		case r.text == "":
			continue
		}
		if last != nil && last.st.LineBreak == style.LineBreakAnywhere {
			add(starts[i])
		}
		last = r
	}
}

// inseparable reports whether r is of UAX #14's class IN.
func inseparable(r rune) bool {
	return r == '․' || r == '‥' || r == '…' || r == '⋯' || r == '︙'
}

// fullPrefix reports whether r is a prefix loose CJK breaks after.
func fullPrefix(r rune) bool { return r == '＄' || r == '￥' || r == '€' }

// ideographic reports whether r is an ideograph or a kana.
func ideographic(r rune) bool {
	return unicode.In(r, unicode.Han, unicode.Hiragana, unicode.Katakana)
}
