package style

import (
	"unicode/utf8"

	"github.com/rivo/uniseg"

	"github.com/lantervale/ember/ui/internal/orient"
)

// WritingMode is writing-mode: whether lines run across the box, the blocks stacking down
// it, or down it, the blocks stacking from its right or its left.
type WritingMode uint8

const (
	HorizontalTB WritingMode = iota
	VerticalRL
	VerticalLR
	SidewaysRL
	SidewaysLR
)

// Vertical reports a mode whose lines run down or up the box.
func (m WritingMode) Vertical() bool { return m != HorizontalTB }

// BlockFromRight reports a vertical mode whose blocks stack from the box's right.
func (m WritingMode) BlockFromRight() bool { return m == VerticalRL || m == SidewaysRL }

// Upward reports a mode whose lines run up the box, their glyphs turned anticlockwise.
func (m WritingMode) Upward() bool { return m == SidewaysLR }

// TextOrientation is text-orientation: how the glyphs of vertical lines stand.
type TextOrientation uint8

const (
	// OrientMixed stands the ideographs and the scripts written upright, and turns the
	// others a quarter clockwise.
	OrientMixed TextOrientation = iota
	OrientUpright
	OrientSideways
)

// Upright reports whether a character of vertical lines in text-orientation o stands
// upright: every one under upright, none under sideways, and under mixed those UAX #50
// doesn't turn (the CJK scripts, emoji, ©, §, the CJK brackets in their vertical forms).
func Upright(r rune, o TextOrientation) bool {
	switch o {
	case OrientUpright:
		return true
	case OrientSideways:
		return false
	case OrientMixed:
	}
	return orient.Of(r).Stands()
}

// Oriented calls fn with the parts of s, text of vertical lines in text-orientation o, in
// order: each run of grapheme clusters turned with the line, and each cluster standing
// upright alone, the orientation of its first character.
func Oriented(s string, o TextOrientation, fn func(part string, upright bool)) {
	from, at := 0, 0
	rest, state := s, -1
	var c string
	for len(rest) > 0 {
		c, rest, _, state = uniseg.FirstGraphemeClusterInString(rest, state)
		r, _ := utf8.DecodeRuneInString(c)
		if Upright(r, o) {
			if from < at {
				fn(s[from:at], false)
			}
			fn(c, true)
			from = at + len(c)
		}
		at += len(c)
	}
	if from < len(s) {
		fn(s[from:], false)
	}
}

// Stands reports text set in vertical lines whose characters may stand upright.
func (s *Computed) Stands() bool {
	return (s.Mode == VerticalRL || s.Mode == VerticalLR) && s.Orientation != OrientSideways
}
