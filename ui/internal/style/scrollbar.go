package style

// ScrollbarParts are a scroll container's ::-webkit-scrollbar parts, each
// styled for the bar it is on, drawn as Chrome draws a styled scrollbar
// (decision 0158). A part no rule styles is nil and draws nothing.
type ScrollbarParts struct {
	V, H BarParts
	// Corner is ::-webkit-scrollbar-corner, where the two bars meet.
	Corner *Computed
}

// BarParts are one bar's: the bar itself, its track, the track's pieces
// before and after the thumb, the thumb, and its four
// ::-webkit-scrollbar-buttons, as Chrome places them: at the start one
// going back then one going forward, at the end one going back then one
// going forward (decision 0198).
type BarParts struct {
	Bar, Track, Start, End, Thumb *Computed
	Buttons                       [4]*Computed
}

// BarPart is a part of a scroll container's bars, as Chrome tells the
// part under the pointer and the one pressed, BarHorizontal set on the
// horizontal bar's: what the parts' :hover and :active read.
type BarPart uint8

const (
	NoBarPart BarPart = iota
	// BarBackground is the bar off its track and buttons.
	BarBackground
	// The buttons, in the order of BarParts.Buttons.
	BarBackStart
	BarForwardStart
	BarBackEnd
	BarForwardEnd
	BarBackTrack
	BarThumb
	BarForwardTrack
	BarHorizontal BarPart = 1 << 7
)

// Part is p on either bar.
func (p BarPart) Part() BarPart { return p &^ BarHorizontal }

// Equal reports whether s and o style the same parts the same way.
func (s *ScrollbarParts) Equal(o *ScrollbarParts) bool {
	if s == o {
		return true
	}
	if s == nil || o == nil {
		return false
	}
	return s.V.equal(&o.V) && s.H.equal(&o.H) && s.Corner.Equal(o.Corner)
}

func (b *BarParts) equal(o *BarParts) bool {
	return b.Bar.Equal(o.Bar) && b.Track.Equal(o.Track) && b.Start.Equal(o.Start) && b.End.Equal(o.End) &&
		b.Thumb.Equal(o.Thumb) && b.Buttons[0].Equal(o.Buttons[0]) && b.Buttons[1].Equal(o.Buttons[1]) &&
		b.Buttons[2].Equal(o.Buttons[2]) && b.Buttons[3].Equal(o.Buttons[3])
}

// StyledBars are a scroller's bars as an element's rules style them,
// and their thicknesses: the viewport's, which Chrome styles from body's
// rules, else the root's, kept on that element (Computed.ViewportBars).
type StyledBars struct {
	Parts         *ScrollbarParts
	Width, Height float32
}

// On is a copy of base, the viewport's style, with b's bars.
func (b *StyledBars) On(base *Computed) *Computed {
	c := *base
	c.Scrollbar, c.ScrollbarWidth, c.ScrollbarHeight = b.Parts, b.Width, b.Height
	c.ScrollbarNone = b.Parts == nil
	return &c
}

// Equal reports whether b and o are the same bars.
func (b *StyledBars) Equal(o *StyledBars) bool {
	if b == nil || o == nil {
		return b == o
	}
	return b.Width == o.Width && b.Height == o.Height && b.Parts.Equal(o.Parts)
}
