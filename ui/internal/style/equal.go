package style

import (
	"maps"
	"reflect"
	"slices"
)

// Equal reports whether s and o are the same style, field by field, as
// reflect.DeepEqual answers: a nil slice or map is not an empty one, a
// pointer is equal to itself or to one whose value is equal, and a NaN
// is equal to nothing but through a pointer, slice or map shared by
// both.
func (s *Computed) Equal(o *Computed) bool {
	if s == o {
		return true
	}
	if s == nil || o == nil {
		return false
	}
	return s.Box.equal(&o.Box) &&
		s.Flex == o.Flex &&
		s.Grid.equal(&o.Grid) &&
		s.Text.equal(&o.Text) &&
		s.Look.equal(&o.Look) &&
		s.ColorInherited == o.ColorInherited &&
		s.CurrentColor == o.CurrentColor &&
		sameMap(s.Vars, o.Vars)
}

//nolint:cyclop,gocyclo // one comparison a field
func (b *Box) equal(o *Box) bool {
	return b.Display == o.Display &&
		b.ListItem == o.ListItem &&
		b.FlowRoot == o.FlowRoot &&
		b.Table == o.Table &&
		b.BoxSizing == o.BoxSizing &&
		b.Position == o.Position &&
		b.Float == o.Float &&
		b.Clear == o.Clear &&
		b.VerticalAlign == o.VerticalAlign &&
		b.VerticalShift == o.VerticalShift &&
		b.Inset == o.Inset &&
		b.Size == o.Size &&
		b.MinSize == o.MinSize &&
		b.MaxSize == o.MaxSize &&
		b.AspectRatio == o.AspectRatio &&
		b.Margin == o.Margin &&
		b.Padding == o.Padding &&
		b.Border == o.Border &&
		b.Overflow == o.Overflow &&
		b.ScrollbarWidth == o.ScrollbarWidth &&
		b.ScrollbarNone == o.ScrollbarNone &&
		b.ScrollbarOverlay == o.ScrollbarOverlay &&
		b.Viewport.Equal(o.Viewport) &&
		b.OverflowPropagated == o.OverflowPropagated &&
		b.ZIndex == o.ZIndex &&
		b.ZAuto == o.ZAuto &&
		b.ColumnCount == o.ColumnCount &&
		b.ColumnWidth == o.ColumnWidth &&
		b.ColumnFillAuto == o.ColumnFillAuto &&
		b.BreakInsideAvoid == o.BreakInsideAvoid &&
		b.BreakBefore == o.BreakBefore &&
		b.BreakAfter == o.BreakAfter
}

func (g *Grid) equal(o *Grid) bool {
	return sameSliceFunc(g.Columns, o.Columns, (*TrackRepeat).equal) &&
		sameSliceFunc(g.Rows, o.Rows, (*TrackRepeat).equal) &&
		sameSlice(g.Areas, o.Areas) &&
		sameSlice(g.AutoColumns, o.AutoColumns) &&
		sameSlice(g.AutoRows, o.AutoRows) &&
		g.Flow == o.Flow &&
		g.Column == o.Column &&
		g.Row == o.Row
}

func (r *TrackRepeat) equal(o *TrackRepeat) bool {
	return r.Count == o.Count && sameSlice(r.Tracks, o.Tracks)
}

// Equal reports whether f and o are the same font, as Computed.Equal
// compares them.
func (f Font) Equal(o Font) bool {
	return sameSlice(f.Family, o.Family) &&
		f.Size == o.Size &&
		f.Weight == o.Weight &&
		f.Italic == o.Italic
}

//nolint:cyclop,gocyclo // one comparison a field
func (t *Text) equal(o *Text) bool {
	return t.Font.Equal(o.Font) &&
		t.LineHeight == o.LineHeight &&
		t.LetterSpacing == o.LetterSpacing &&
		t.WordSpacing == o.WordSpacing &&
		t.Color == o.Color &&
		t.TextAlign == o.TextAlign &&
		t.WhiteSpace == o.WhiteSpace &&
		t.TextIndent == o.TextIndent &&
		t.PlaceholderColor == o.PlaceholderColor &&
		t.HasPlaceholder == o.HasPlaceholder &&
		t.Backdrop == o.Backdrop &&
		t.SelectionColor == o.SelectionColor &&
		t.HasSelection == o.HasSelection &&
		t.SelectionText == o.SelectionText &&
		t.HasSelectionText == o.HasSelectionText &&
		sameSlice(t.TextShadows, o.TextShadows) &&
		t.TextDecoration == o.TextDecoration &&
		sameSliceFunc(t.Decorations, o.Decorations, (*Decoration).equal) &&
		t.AccentColor == o.AccentColor &&
		t.CaretColor == o.CaretColor &&
		t.AccentAuto == o.AccentAuto &&
		t.CaretAuto == o.CaretAuto &&
		t.Content.equal(&o.Content) &&
		sameSlice(t.CounterReset, o.CounterReset) &&
		sameSlice(t.CounterIncrement, o.CounterIncrement) &&
		sameSlice(t.CounterSet, o.CounterSet) &&
		sameSlice(t.Quotes, o.Quotes) &&
		t.QuotesNone == o.QuotesNone &&
		t.ListStyle == o.ListStyle &&
		t.BorderSpacing == o.BorderSpacing &&
		t.CaptionSide == o.CaptionSide &&
		t.TableLayout == o.TableLayout &&
		t.TextTransform == o.TextTransform &&
		t.PixelText == o.PixelText &&
		t.ExactText == o.ExactText &&
		t.Ellipsis == o.Ellipsis &&
		t.EllipsisStart == o.EllipsisStart &&
		t.Orphans == o.Orphans &&
		t.Widows == o.Widows &&
		t.BreakWord == o.BreakWord &&
		t.BreakAnywhere == o.BreakAnywhere
}

func (d *Decoration) equal(o *Decoration) bool {
	return d.TextDecoration == o.TextDecoration && d.Font.Equal(o.Font)
}

func (c *Content) equal(o *Content) bool {
	return c.Kind == o.Kind && sameSlice(c.Items, o.Items)
}

//nolint:cyclop,gocyclo // one comparison a field
func (l *Look) equal(o *Look) bool {
	return l.Background == o.Background &&
		l.BackgroundImage.equal(&o.BackgroundImage) &&
		l.BackgroundSize == o.BackgroundSize &&
		l.BackgroundNoRepeat == o.BackgroundNoRepeat &&
		l.BackgroundRepeat == o.BackgroundRepeat &&
		l.BackgroundPosition == o.BackgroundPosition &&
		l.MaskImage.equal(&o.MaskImage) &&
		l.MaskSize == o.MaskSize &&
		l.MaskNoRepeat == o.MaskNoRepeat &&
		l.MaskRepeat == o.MaskRepeat &&
		l.MaskPosition == o.MaskPosition &&
		l.ImageRendering == o.ImageRendering &&
		l.ObjectFit == o.ObjectFit &&
		l.ObjectPosition == o.ObjectPosition &&
		l.BorderColor == o.BorderColor &&
		l.BorderStyle == o.BorderStyle &&
		l.BorderImage.equal(&o.BorderImage) &&
		l.BorderImageRepeat == o.BorderImageRepeat &&
		l.BorderImageWidth == o.BorderImageWidth &&
		l.Radius == o.Radius &&
		sameSlice(l.Shadows, o.Shadows) &&
		l.OutlineWidth == o.OutlineWidth &&
		l.OutlineStyle == o.OutlineStyle &&
		l.OutlineColor == o.OutlineColor &&
		l.OutlineOffset == o.OutlineOffset &&
		l.ColumnRuleWidth == o.ColumnRuleWidth &&
		l.ColumnRuleStyle == o.ColumnRuleStyle &&
		l.ColumnRuleColor == o.ColumnRuleColor &&
		l.Opacity == o.Opacity &&
		l.Visible == o.Visible &&
		l.PointerNone == o.PointerNone &&
		l.AppDrag == o.AppDrag &&
		l.Cursor == o.Cursor &&
		l.UserSelect == o.UserSelect &&
		sameSlice(l.Transform, o.Transform) &&
		l.TransformOrigin == o.TransformOrigin &&
		sameSlice(l.Transitions, o.Transitions)
}

func (i *Image) equal(o *Image) bool {
	return i.Source == o.Source && i.Slice == o.Slice && i.Gradient.equal(o.Gradient)
}

func (g *Gradient) equal(o *Gradient) bool {
	if g == o {
		return true
	}
	if g == nil || o == nil {
		return false
	}
	return g.Kind == o.Kind &&
		g.Angle == o.Angle &&
		g.Corner == o.Corner &&
		g.At == o.At &&
		g.Circle == o.Circle &&
		g.Size == o.Size &&
		sameSlice(g.Stops, o.Stops)
}

func sameSlice[T comparable](a, b []T) bool {
	if len(a) != len(b) || (a == nil) != (b == nil) {
		return false
	}
	return len(a) == 0 || &a[0] == &b[0] || slices.Equal(a, b)
}

func sameSliceFunc[T any](a, b []T, equal func(a, b *T) bool) bool {
	if len(a) != len(b) || (a == nil) != (b == nil) {
		return false
	}
	if len(a) == 0 || &a[0] == &b[0] {
		return true
	}
	for i := range a {
		if !equal(&a[i], &b[i]) {
			return false
		}
	}
	return true
}

// sameMap is true at once for one map held by both, as an inherited Vars
// is.
func sameMap(a, b map[string]string) bool {
	if len(a) != len(b) || (a == nil) != (b == nil) {
		return false
	}
	return len(a) == 0 || reflect.ValueOf(a).Pointer() == reflect.ValueOf(b).Pointer() || maps.Equal(a, b)
}
