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
