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
// before and after the thumb, and the thumb.
type BarParts struct {
	Bar, Track, Start, End, Thumb *Computed
}

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
		b.Thumb.Equal(o.Thumb)
}
