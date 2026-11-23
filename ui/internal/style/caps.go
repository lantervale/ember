package style

import (
	"math"
	"unicode"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// smallCapsScale is the size of a synthesised small capital to the
// font's, Chrome's kSmallCapsFontSizeMultiplier.
const smallCapsScale = 0.7

// CapsRun is a part of a text drawn in one font where small capitals are
// synthesised.
type CapsRun struct {
	Text string
	Font Font
}

// SmallCaps splits text into the runs f's font-variant-caps draws it in,
// as Chrome synthesises small capitals for a face that has none: what it
// makes small capitals uppercased at 70% of the size rounded to a whole
// one, the rest as they are, each run's font without Caps. It is nil
// where f synthesises none.
func (f Font) SmallCaps(text string) []CapsRun {
	if !f.Caps.synthesised() {
		return nil
	}
	plain := f
	plain.Caps = NormalCaps
	small := plain
	// Chrome makes the scaled face at a whole size (SimpleFontData::CreateScaledFontData).
	small.Size = float32(math.Round(float64(f.Size * smallCapsScale)))
	byFeatures := f.Caps == UnicaseFeatures
	if byFeatures {
		small = plain
		small.Features = "smcp,c2sc"
		if f.Features != "" {
			small.Features += "," + f.Features
		}
	}
	var out []CapsRun
	start, smallRun := 0, false
	flush := func(end int) {
		if end == start {
			return
		}
		switch {
		case smallRun && byFeatures:
			out = append(out, CapsRun{Text: text[start:end], Font: small})
		case smallRun:
			out = append(out, CapsRun{Text: upper.String(text[start:end]), Font: small})
		default:
			out = append(out, CapsRun{Text: text[start:end], Font: plain})
		}
		start = end
	}
	for i, r := range text {
		if s := f.Caps.smallens(r); s != smallRun {
			flush(i)
			smallRun = s
		}
	}
	flush(len(text))
	return out
}

func (c FontCaps) synthesised() bool { return c != NormalCaps && c != TitlingCaps }

// upper uppercases a small capitals' run as Chrome does, ß as SS.
var upper = cases.Upper(language.Und) //nolint:gochecknoglobals // a Caser built once

// smallens reports whether c draws r as a small capital: a lowercase
// letter, every character for the all- values, all but lowercase letters
// for unicase, as Chrome draws them.
func (c FontCaps) smallens(r rune) bool {
	switch c {
	case SmallCaps, PetiteCaps:
		return unicode.IsLower(r)
	case AllSmallCaps, AllPetiteCaps:
		return true
	case Unicase, UnicaseFeatures:
		return !unicode.IsLower(r)
	case NormalCaps, TitlingCaps:
	}
	return false
}
