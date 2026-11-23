// Package orient is the orientation of each character in vertical lines,
// Unicode's Vertical_Orientation (UAX #50), which text-orientation: mixed
// follows.
package orient

//go:generate go run ./gen -src https://www.unicode.org/Public/17.0.0/ucd/VerticalOrientation.txt -out orient_gen.go

// Orientation is how a character is drawn in vertical lines.
type Orientation uint8

const (
	// Rotated is turned a quarter with the line, as Latin letters.
	Rotated Orientation = iota
	// Upright stands, as ideographs, ©, § and most emoji.
	Upright
	// TransformedUpright stands, in a glyph of its own for vertical lines
	// where the font has one, as the ideographic comma, moved to the
	// line's top right.
	TransformedUpright
	// TransformedRotated stands in a glyph of its own for vertical lines,
	// turned, where the font has one, as the CJK brackets.
	TransformedRotated
)

// Stands reports whether o is drawn upright under mixed: every orientation
// but Rotated, the transformed ones in their vertical glyphs.
func (o Orientation) Stands() bool { return o != Rotated }

type span struct {
	lo, hi rune
	o      Orientation
}

// Of is r's orientation.
func Of(r rune) Orientation {
	lo, hi := 0, len(ranges)
	for lo < hi {
		m := int(uint(lo+hi) >> 1)
		switch s := ranges[m]; {
		case r < s.lo:
			hi = m
		case r > s.hi:
			lo = m + 1
		default:
			return s.o
		}
	}
	return Rotated
}
