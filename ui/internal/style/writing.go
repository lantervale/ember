package style

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
// upright: every one under upright, none under sideways, and under mixed those of the
// scripts written in vertical lines (Han, kana, Hangul, Bopomofo, Yi) with the
// fullwidth forms and CJK symbols, UTR #50's upright ones there.
func Upright(r rune, o TextOrientation) bool {
	switch o {
	case OrientUpright:
		return true
	case OrientSideways:
		return false
	case OrientMixed:
	}
	switch {
	case r >= 0x1100 && r <= 0x11ff, r >= 0x2e80 && r <= 0x2fdf, r >= 0x3000 && r <= 0x30ff,
		r >= 0x3100 && r <= 0x31ff, r >= 0x3200 && r <= 0x33ff, r >= 0x3400 && r <= 0x4dbf,
		r >= 0x4e00 && r <= 0x9fff, r >= 0xa000 && r <= 0xa4cf, r >= 0xa960 && r <= 0xa97f,
		r >= 0xac00 && r <= 0xd7ff, r >= 0xf900 && r <= 0xfaff, r >= 0xfe10 && r <= 0xfe1f,
		r >= 0xfe30 && r <= 0xfe4f, r >= 0xff01 && r <= 0xff60, r >= 0xffe0 && r <= 0xffe6,
		r >= 0x20000 && r <= 0x3ffff:
		return true
	}
	return false
}

// Stands reports text set in vertical lines whose characters may stand upright.
func (s *Computed) Stands() bool {
	return (s.Mode == VerticalRL || s.Mode == VerticalLR) && s.Orientation != OrientSideways
}
