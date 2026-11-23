package style

import (
	"unicode/utf8"

	"github.com/lantervale/ember/color"
)

// Computed is one node's computed style. The zero value is CSS's initial
// value for every property except where noted; css starts from Initial().
type Computed struct {
	Box
	Flex
	Grid
	Text
	Look
	// ColorInherited is set when color comes from the parent (inherit,
	// unset, currentcolor or no declaration), so a parent's colour
	// transition carries on to it.
	ColorInherited bool
	// CurrentColor marks the colours that are currentColor, so they follow
	// color, its transitions included.
	CurrentColor CurrentColorFields
	// Vars is the custom properties (--name) as CSS text, set and inherited
	// by css.
	Vars map[string]string
}

// Initial is CSS's initial values, with ui's box-sizing: border-box.
func Initial() Computed {
	return Computed{
		Box: Box{
			Display:   Inline,
			BoxSizing: BorderBox,
			MaxSize:   Size[Length]{Length{Kind: None}, Length{Kind: None}},
			Margin:    All(Px(0)),
			Padding:   All(Px(0)),
			ZAuto:     true,
		},
		Flex: Flex{
			Shrink: 1, AlignItems: AlignNormal, AlignContent: AlignNormal, JustifyContent: AlignNormal,
			AlignSelf: AlignAuto, JustifyItems: AlignNormal, JustifySelf: AlignAuto,
		},
		Text: Text{
			Font:       Font{Family: []string{"sans-serif"}, Size: 16, Weight: 400, Stretch: 100},
			LineHeight: LineHeight{Normal: true},
			Color:      color.Color{A: 1},
			TextAlign:  Start,
			Orphans:    2, Widows: 2,
			TabSize: TabSize{Spaces: 8},
		},
		Look: Look{
			Opacity: 1, Visible: true, TransformOrigin: Size[Length]{Percent(50), Percent(50)},
			BorderImageWidth: Sides[ImageWidth]{
				Top: ImageWidth{WidthTimes, 1}, Right: ImageWidth{WidthTimes, 1},
				Bottom: ImageWidth{WidthTimes, 1}, Left: ImageWidth{WidthTimes, 1},
			},
		},
	}
}

// Display is the outer and inner display type, as one keyword.
type Display uint8

const (
	Inline Display = iota
	Block
	InlineBlock
	FlexDisplay
	InlineFlex
	GridDisplay
	InlineGrid
	DisplayNone
	Contents
)

// UserSelect is whether a selection takes an element's text.
type UserSelect uint8

const (
	SelectText UserSelect = iota
	SelectNone
	// SelectAll is taken whole or not at all.
	SelectAll
)

// AppRegion is app-region: what a press there does to the system's window.
type AppRegion uint8

const (
	// AppRegionNone is none set on the box or above it: the page's own.
	AppRegionNone AppRegion = iota
	// AppRegionDrag moves the window.
	AppRegionDrag
	// AppRegionNoDrag takes the box out of a drag region, as Chrome computes
	// none once set too.
	AppRegionNoDrag
)

type BoxSizing uint8

const (
	BorderBox BoxSizing = iota
	ContentBox
)

type Position uint8

const (
	Static Position = iota
	Relative
	Absolute
	FixedPosition
	Sticky
)

// Float is float: the side of its containing block a box is taken out of
// the flow to, the content after it wrapping round it.
type Float uint8

const (
	FloatNone Float = iota
	FloatLeft
	FloatRight
)

// Clear is clear: the sides whose earlier floats a box goes below.
type Clear uint8

const (
	ClearNone Clear = iota
	ClearLeft
	ClearRight
	ClearBoth
)

type Overflow uint8

const (
	Visible Overflow = iota
	Hidden
	Clip
	Scroll
	OverflowAuto
)

// Box holds what places and sizes the node's box.
type Box struct {
	Display Display
	// ListItem is display: list-item, laid out as Block, with a marker
	// where the element has one (a summary's).
	ListItem bool
	// FlowRoot is display: flow-root, laid out as Block, holding its floats
	// and standing beside those around it.
	FlowRoot bool
	// Table is display: table or inline-table: the element is laid out as
	// a table, its children its rows, row groups, columns and captions, or
	// wrapped in anonymous ones. TablePart is one of those displays, on an
	// element that is none once it floats or leaves the flow.
	Table     bool
	TablePart TablePart
	BoxSizing BoxSizing
	Position  Position
	// Float and Clear are float and clear; a float applies to a box in the
	// flow of a block container, not to a flex or grid item or a box
	// position takes out of the flow.
	Float Float
	Clear Clear
	// VerticalAlign is vertical-align: where an inline-level box stands on its line, or
	// a table cell's content in the cell. VerticalShift is how far up an AlignLength
	// raises it, its percentage of the box's own line height.
	VerticalAlign VerticalAlign
	VerticalShift Length
	// Inset is top, right, bottom, left; Auto where unset.
	Inset   Sides[Length]
	Size    Size[Length]
	MinSize Size[Length]
	MaxSize Size[Length]
	// AspectRatio is width over height, 0 for auto.
	AspectRatio float32
	Margin      Sides[Length]
	Padding     Sides[Length]
	// Border is the used border widths: 0 where border-style is none.
	Border   Sides[float32]
	Overflow Size[Overflow]
	// ScrollbarWidth is the gutter, in points, that overflow: scroll reserves
	// across the scrolling axis (scrollbar-width: none is 0).
	ScrollbarWidth float32
	// ScrollbarNone is scrollbar-width: none: no gutter and no bars, the
	// box still scrolling by the wheel.
	ScrollbarNone bool
	// ScrollbarOverlay is a Document's overlay scrollbars on a box that
	// shows bars: they take no gutter and draw over its content.
	ScrollbarOverlay bool
	// Viewport is the style of the viewport, in a Document whose viewport
	// scrolls: on the root always, and on body when the viewport takes
	// body's overflow. OverflowPropagated is set on the one of them whose
	// overflow the viewport took: its own Overflow is visible.
	Viewport           *Computed
	OverflowPropagated bool
	// ZIndex applies when ZAuto is false.
	ZIndex int
	ZAuto  bool
	// ColumnCount is column-count, 0 for auto, and ColumnWidth column-width,
	// auto or points: either makes the box a multi-column container, whose
	// column-gap is Gap.Width.
	ColumnCount int
	ColumnWidth Length
	// ColumnFillAuto is column-fill: auto: columns fill in turn, not
	// balanced.
	ColumnFillAuto bool
	// ColumnSpanAll is column-span: all: a block of a multi-column
	// container shown across all its columns, cutting them in rows.
	ColumnSpanAll bool
	// BreakInsideAvoid is break-inside: avoid; BreakBefore and BreakAfter
	// are break-before and break-after between columns.
	BreakInsideAvoid        bool
	BreakBefore, BreakAfter Break
	// ContainerType is container-type, and ContainerNames container-name,
	// nil for none: the names an @container rule picks the element by.
	ContainerType  ContainerType
	ContainerNames []string
}

// ContainerType is container-type: whether an element is a query
// container for its size, and on which axes. A size container's size
// doesn't depend on its content on those axes.
type ContainerType uint8

const (
	ContainerNormal ContainerType = iota
	ContainerInlineSize
	ContainerSize
)

// Multicol reports a multi-column container.
func (b *Box) Multicol() bool { return b.ColumnCount > 0 || b.ColumnWidth.Kind != Auto }

// Break is a break-before or break-after between columns.
type Break uint8

const (
	BreakAuto Break = iota
	// BreakAvoid keeps the boxes on either side in one column where it can.
	BreakAvoid
	// BreakColumn starts a new column.
	BreakColumn
)

type FlexDirection uint8

const (
	Row FlexDirection = iota
	RowReverse
	Column
	ColumnReverse
)

type FlexWrap uint8

const (
	NoWrap FlexWrap = iota
	Wrap
	WrapReverse
)

// Align is a value of align-*, justify-* and place-*.
type Align uint8

const (
	AlignNormal Align = iota
	AlignAuto
	AlignStretch
	AlignStart
	AlignEnd
	AlignFlexStart
	AlignFlexEnd
	AlignCenter
	AlignBaseline
	AlignSpaceBetween
	AlignSpaceAround
	AlignSpaceEvenly
)

// Flex holds the flex and box-alignment properties.
type Flex struct {
	Direction FlexDirection
	Wrap      FlexWrap
	Grow      float32
	Shrink    float32
	Basis     Length
	// Gap is column-gap (Width) and row-gap (Height).
	Gap            Size[Length]
	AlignItems     Align
	AlignSelf      Align
	AlignContent   Align
	JustifyContent Align
	JustifyItems   Align
	JustifySelf    Align
	// Order is the flex and grid order.
	Order int
}

// Track is one grid track size: minmax(Min, Max), or a single size in
// both.
type Track struct{ Min, Max Length }

// Repeat counts of a TrackList entry.
const (
	AutoFill = -1
	AutoFit  = -2
)

// TrackList entry: Tracks repeated Count times (1 for no repeat()), or
// AutoFill/AutoFit. Before names the line before the entry and After the
// line after it (the list's last entry's only); Names, when a repeat()
// names any, are the names of the lines round its Tracks, one more than
// them, its last and first meeting on one line between two repetitions.
type TrackRepeat struct {
	Count         int
	Tracks        []Track
	Names         [][]string
	Before, After []string
}

// GridLine is one end of a placement: auto, a line number (negative from
// the end), a span, the Line-th line (from the end when negative) or a
// span to the Span-th line of a Name, or a Name alone: the line Name-start
// or Name-end on that side, else the first line of that name.
type GridLine struct {
	Line int
	Span int
	Name string
}

// IsAuto reports a line that is neither a number, a span nor a name.
func (g GridLine) IsAuto() bool { return g.Line == 0 && g.Span == 0 && g.Name == "" }

// GridArea is a named area of grid-template-areas: the lines round it,
// from 1.
type GridArea struct {
	Name        string
	Row, Column [2]int
}

type GridFlow uint8

const (
	FlowRow GridFlow = iota
	FlowColumn
	FlowRowDense
	FlowColumnDense
)

type Grid struct {
	Columns []TrackRepeat
	Rows    []TrackRepeat
	// Areas are grid-template-areas' named areas.
	Areas       []GridArea
	AutoColumns []Track
	AutoRows    []Track
	Flow        GridFlow
	// Subgrid is grid-template-columns (Width) and grid-template-rows
	// (Height) being subgrid: that axis's entries are then the names of its
	// lines, each a bracket or a repeat() of them (Count, AutoFill), Names
	// its brackets and no Tracks.
	Subgrid Size[bool]
	// Column and Row are grid-column and grid-row, Start/End.
	Column [2]GridLine
	Row    [2]GridLine
}

// Font is what picks and sizes a face; the root package maps it to a
// twod.Face.
type Font struct {
	// Family is font-family's list, generic names last.
	Family []string
	Size   float32
	Weight float32
	Italic bool
	// Stretch is font-stretch as a percentage of the normal width, 0
	// standing for 100.
	Stretch float32
	// Synthesis is what font-synthesis forbids making up.
	Synthesis FontSynthesis
	// Caps is font-variant-caps, which measuring and drawing synthesise
	// (SmallCaps).
	Caps FontCaps
}

// FontSynthesis is the faces font-synthesis forbids making up from
// another of the family, none forbidden at zero.
type FontSynthesis uint8

const (
	// NoSyntheticBold leaves a bold the family lacks drawn in its
	// lighter face.
	NoSyntheticBold FontSynthesis = 1 << iota
	// NoSyntheticItalic leaves an italic the family lacks upright.
	NoSyntheticItalic
)

// LineHeight is normal, a factor of the font size, or points.
type LineHeight struct {
	Normal bool
	Factor float32
	Px     float32
}

// TablePart is the part of a table an element's display makes it.
type TablePart uint8

const (
	NoTablePart TablePart = iota
	TableRow
	TableCell
	TableRowGroup
	TableHeaderGroup
	TableFooterGroup
	TableColumn
	TableColumnGroup
	TableCaption
)

// CaptionSide is caption-side: a table's caption above it or below.
type CaptionSide uint8

const (
	CaptionTop CaptionSide = iota
	CaptionBottom
)

// TableLayout is table-layout: columns from every cell's content, or from
// the first row's and the table's width alone.
type TableLayout uint8

const (
	TableAuto TableLayout = iota
	TableFixed
)

// TextTransform is text-transform: how text shows cased, its source kept.
type TextTransform uint8

const (
	NoTransform TextTransform = iota
	Uppercase
	Lowercase
	Capitalize
)

// TextAlign is a text-align. The Block ones are Chrome's -webkit-left,
// -webkit-right and -webkit-center, which HTML's align attribute and
// center element stand for: lines as the plain word, and the blocks
// narrower than their container placed the same way.
type TextAlign uint8

const (
	Start TextAlign = iota
	End
	Left
	Right
	Center
	Justify
	BlockLeft
	BlockRight
	BlockCenter
)

// Direction is direction.
type Direction uint8

const (
	LTR Direction = iota
	RTL
)

// UnicodeBidi is unicode-bidi.
type UnicodeBidi uint8

const (
	BidiNormal UnicodeBidi = iota
	BidiEmbed
	BidiIsolate
	BidiOverride
	BidiIsolateOverride
	BidiPlaintext
)

// Isolates reports whether u isolates its content from the text round it.
func (u UnicodeBidi) Isolates() bool {
	return u == BidiIsolate || u == BidiIsolateOverride || u == BidiPlaintext
}

// Overrides reports whether u orders its content by its direction alone.
func (u UnicodeBidi) Overrides() bool { return u == BidiOverride || u == BidiIsolateOverride }

// Resolve is a, start and end on the sides d puts them, left or right;
// the others as they are.
func (a TextAlign) Resolve(d Direction) TextAlign {
	switch {
	case a == Start && d == RTL, a == End && d == LTR:
		return Right
	case a == Start, a == End:
		return Left
	}
	return a
}

type WhiteSpace uint8

const (
	Normal WhiteSpace = iota
	NoWrapSpace
	Pre
	PreWrap
	PreLine
)

// Spacing is letter-spacing or word-spacing: Px points after each
// character or word separator, and Percent of the font size, which a child
// inherits as a percentage of its own.
type Spacing struct{ Px, Percent float32 }

// WordSeparator reports whether word-spacing follows r: a space or a
// no-break space, the two Chrome widens.
func WordSeparator(r rune) bool { return r == ' ' || r == '\u00a0' }

// Fill is the colour glyphs are filled with: -webkit-text-fill-color's,
// else color.
func (t *Text) Fill() color.Color {
	if t.TextFilled {
		return t.TextFill
	}
	return t.Color
}

// WordGap is the room word-spacing adds after each word separator in t's
// font.
func (t *Text) WordGap() float32 { return t.WordSpacing.Px + t.WordSpacing.Percent*t.Font.Size/100 }

// LetterGap is the room letter-spacing adds after each character in t's
// font.
func (t *Text) LetterGap() float32 {
	return t.LetterSpacing.Px + t.LetterSpacing.Percent*t.Font.Size/100
}

// Spacing is the room letter-spacing and word-spacing add to s: the one
// after each character, the other after each word separator.
func (t *Text) Spacing(s string) float32 {
	word, letter := t.WordGap(), t.LetterGap()
	if word == 0 {
		if letter == 0 {
			return 0
		}
		return letter * float32(utf8.RuneCountInString(s))
	}
	n, seps := 0, 0
	for _, r := range s {
		n++
		if WordSeparator(r) {
			seps++
		}
	}
	return letter*float32(n) + word*float32(seps)
}

// FontCaps is font-variant-caps.
type FontCaps uint8

const (
	NormalCaps FontCaps = iota
	SmallCaps
	AllSmallCaps
	PetiteCaps
	AllPetiteCaps
	Unicase
	TitlingCaps
)

// FontVariant is font-variant's longhands but its capitals (Font.Caps),
// kept as written, their keywords in the order CSS lists them, "" for
// normal. They change nothing drawn: ui applies no OpenType feature of a
// face.
type FontVariant struct {
	Ligatures, Numeric, EastAsian, Alternates, Position string
}

type Text struct {
	Font          Font
	FontVariant   FontVariant
	LineHeight    LineHeight
	LetterSpacing Spacing
	WordSpacing   Spacing
	Color         color.Color
	TextAlign     TextAlign
	WhiteSpace    WhiteSpace
	// Dir is direction: the order of the inline flow, and the side
	// each start and end stands on. Bidi is unicode-bidi, how the element
	// takes part in the bidi algorithm of the text round it.
	Dir  Direction
	Bidi UnicodeBidi
	// TextIndent is text-indent: how far the first line of a block's text
	// starts in from its edge, out when negative; a percentage is of the
	// block's width.
	TextIndent Length
	// PlaceholderColor is the color of the element's ::placeholder, set
	// when HasPlaceholder is (a ::placeholder rule matched).
	PlaceholderColor color.Color
	HasPlaceholder   bool
	// Backdrop is a dialog's ::backdrop background-color, drawn under it
	// while it is modal; transparent draws none.
	Backdrop color.Color
	// SelectionColor is the ::selection background-color, set when
	// HasSelection is (a ::selection rule of the element or of an
	// ancestor set it), and SelectionText its color, set when
	// HasSelectionText is. With either set the other is CSS's own:
	// no background, the text's colour.
	SelectionColor   color.Color
	HasSelection     bool
	SelectionText    color.Color
	HasSelectionText bool
	// SelectionShadows is the ::selection's text-shadow, set when
	// HasSelectionShadows is: the selected text's shadows in place of its
	// own. SelectionDecoration is its text-decoration, set when
	// HasSelectionDecoration is, drawn on the selected text over the
	// text's own.
	SelectionShadows       []Shadow
	HasSelectionShadows    bool
	SelectionDecoration    TextDecoration
	HasSelectionDecoration bool
	// TextShadows is text-shadow: Spread and Inset are unused.
	TextShadows []Shadow
	// TextDecoration is the element's own text-decoration, and Decorations
	// those drawn on its text, the outermost box's first: its ancestors',
	// unless it is a float, out of flow or an atomic inline, then its own.
	TextDecoration TextDecoration
	Decorations    []Decoration
	// AccentColor and CaretColor are accent-color and caret-color,
	// resolved: their element's color when AccentAuto or CaretAuto (auto
	// or currentColor, kept as such when inherited).
	AccentColor, CaretColor color.Color
	AccentAuto, CaretAuto   bool
	// TextFill is -webkit-text-fill-color, the colour glyphs are filled
	// with when TextFilled; else they take color, as for currentColor.
	TextFill   color.Color
	TextFilled bool
	// ColorScheme is color-scheme, and Dark the scheme the element is
	// drawn in: its system colours and light-dark() resolved, and the
	// canvas behind a dark root.
	ColorScheme ColorScheme
	Dark        bool
	// Content is the content property: what a ::before, ::after or
	// ::marker box holds.
	Content Content
	// CounterReset, CounterIncrement and CounterSet are the counter
	// properties, in the order written.
	CounterReset, CounterIncrement, CounterSet []Counter
	// Quotes are the quotes property's marks, an opening and a closing one
	// per depth; none when QuotesNone, the language's own when empty.
	Quotes     []string
	QuotesNone bool
	// ListStyle is list-style-type: a bullet, a summary's disclosure
	// triangle, none, or the counter style or the string ListName names.
	ListStyle ListStyle
	ListName  string
	// ListImage is list-style-image: the address of the picture a list
	// item's marker shows in place of its list-style-type, "" for none.
	ListImage string
	// BorderSpacing is a table's border-spacing, across and down, in points.
	BorderSpacing [2]float32
	// BorderCollapse is border-collapse: collapse: a table's cells share
	// their borders, the widest winning each edge.
	BorderCollapse bool
	// EmptyCellsHide is empty-cells: hide: a cell with nothing in it draws
	// no border and no background, borders kept apart.
	EmptyCellsHide bool
	CaptionSide    CaptionSide
	TableLayout    TableLayout
	TextTransform  TextTransform
	// PixelText draws the text's glyphs unsmoothed, as
	// -webkit-font-smoothing: none does in Chrome: for pixel fonts.
	PixelText bool
	// ExactText draws them at their exact coverage, without stem
	// darkening, as -webkit-font-smoothing: antialiased does in Chrome.
	ExactText bool
	// Ellipsis is text-overflow: ellipsis.
	Ellipsis bool
	// EllipsisStart cuts the start of the text rather than its end, as a
	// path keeps its last folders: class truncate-start, an ember
	// extension.
	EllipsisStart bool
	// Orphans and Widows are the fewest lines of a block left at the end
	// of a column and taken to the start of the next.
	Orphans, Widows int
	// BreakWord is overflow-wrap: anywhere or break-word.
	BreakWord bool
	// BreakAnywhere is overflow-wrap: anywhere, whose breaks also count
	// in min-content widths.
	BreakAnywhere bool
	WordBreak     WordBreak
	LineBreak     LineBreak
	TabSize       TabSize
	// InkThrough is text-decoration-skip-ink: none: underlines and
	// overlines go through the glyphs' ink, which they otherwise skip.
	InkThrough bool
}

// WordBreak is word-break: where lines may break inside words.
type WordBreak uint8

const (
	WordBreakNormal WordBreak = iota
	// BreakAll breaks between any two letters, as between ideographs.
	BreakAll
	// KeepAll breaks no word of letters, ideographs neither.
	KeepAll
	// WordBreakBreakWord is break-word, the old name of overflow-wrap:
	// anywhere.
	WordBreakBreakWord
)

// LineBreak is line-break. Its values but anywhere differ only on the
// punctuation of Chinese and Japanese, where ui breaks as normal.
type LineBreak uint8

const (
	LineBreakAuto LineBreak = iota
	LineBreakLoose
	LineBreakNormal
	LineBreakStrict
	// LineBreakAnywhere breaks between any two characters.
	LineBreakAnywhere
)

// TabSize is tab-size: Spaces spaces of the text's font, or Px points
// when Length.
type TabSize struct {
	Spaces, Px float32
	Length     bool
}

// TabStop is the distance between t's tab stops, space the advance of a
// space in its font.
func (t *Text) TabStop(space float32) float32 {
	if t.TabSize.Length {
		return t.TabSize.Px
	}
	return t.TabSize.Spaces * space
}

// WrapsAnywhere reports whether a word too wide for its line breaks
// where it must: overflow-wrap or word-break: break-word.
func (t *Text) WrapsAnywhere() bool { return t.BreakWord || t.WordBreak == WordBreakBreakWord }

// MinAnywhere reports whether those breaks also count in min-content
// widths: overflow-wrap: anywhere, word-break: break-word.
func (t *Text) MinAnywhere() bool { return t.BreakAnywhere || t.WordBreak == WordBreakBreakWord }

type BorderStyle uint8

const (
	BorderNone BorderStyle = iota
	BorderSolid
	BorderDashed
	BorderDotted
	// BorderInset and BorderOutset draw a side in its colour or a darker
	// shade of it, as Chrome does: inset darker on top and left, outset
	// on bottom and right.
	BorderInset
	BorderOutset
	// BorderGroove and BorderRidge draw a side in two halves: groove's
	// outer one as inset and inner one as outset, ridge's the other way.
	BorderGroove
	BorderRidge
	// BorderDouble draws a side as two solid lines, each about a third of
	// its width, and solid where it is too thin to show the gap.
	BorderDouble
	// BorderAuto is outline-style: auto, the focus ring; a border never
	// takes it.
	BorderAuto
)

// Shadow is one box-shadow.
type Shadow struct {
	X, Y, Blur, Spread float32
	Color              color.Color
	Inset              bool
	// CurrentColor is set when Color is currentColor: the colour of the
	// element it is on, inherited as such.
	CurrentColor bool
}

// Image is a background-image or border-image source by name, with
// border-image's slice for nine-patches, or a background's
// linear-gradient().
type Image struct {
	Source string
	Slice  Sides[float32]
	// Fill is border-image-slice's fill: the middle drawn too.
	Fill     bool
	Gradient *Gradient
}

// Gradient is a linear-gradient(), a conic-gradient() or a
// radial-gradient(): its kind, its geometry and its colour stops.
type Gradient struct {
	Kind GradientKind
	// Angle is a linear gradient's line, in degrees, 0 up and 90 to the
	// right, while Corner is zero; a conic gradient's from angle.
	Angle float32
	// Corner is a linear gradient's "to" corner, -1 or 1 across and down,
	// whose angle follows the box's shape.
	Corner [2]int8
	// At is a conic or radial gradient's centre, 50% 50% by default.
	At BackgroundPosition
	// Circle and Size are a radial gradient's shape (an ellipse unless
	// Circle) and its extent.
	Circle bool
	Size   RadialSize
	// Stops are the colour stops; a conic gradient's positions are
	// percentages of a turn, its degrees made so.
	Stops []Stop
}

// GradientKind is which of the gradient functions a Gradient is.
type GradientKind uint8

const (
	LinearGradient GradientKind = iota
	ConicGradient
	RadialGradient
)

// RadialSize is a radial gradient's extent keyword.
type RadialSize uint8

const (
	FarthestCorner RadialSize = iota
	ClosestSide
	ClosestCorner
	FarthestSide
)

// Stop is a colour stop: its colour and, when Placed, where it stands on
// the gradient line.
type Stop struct {
	Color  color.Color
	At     Length
	Placed bool
}

// Transition is one transition: a property or "all", in seconds.
type Transition struct {
	Property        string
	Duration, Delay float32
	Easing          string
	// Discrete is transition-behavior: allow-discrete: the property
	// transitions between values that don't interpolate, flipping.
	Discrete bool
}

// Animation is one entry of the animation list: the @keyframes it runs,
// in seconds, Iterations +Inf for infinite.
type Animation struct {
	Name            string
	Duration, Delay float32
	Easing          string
	Iterations      float32
	Direction       AnimationDirection
	Fill            AnimationFill
	Paused          bool
	Composite       Composite
}

// Composite is animation-composition: how an animation's values combine
// with what is under them.
type Composite uint8

const (
	ComposeReplace Composite = iota
	ComposeAdd
	ComposeAccumulate
)

type AnimationDirection uint8

const (
	AnimateNormal AnimationDirection = iota
	AnimateReverse
	AnimateAlternate
	AnimateAlternateReverse
)

// AnimationFill is animation-fill-mode.
type AnimationFill uint8

const (
	FillNone AnimationFill = iota
	FillForwards
	FillBackwards
	FillBoth
)

type Look struct {
	// Background is background-color, painted under the last layer in
	// the box its clip names.
	Background color.Color
	// BackgroundImages are background-image's layers, the first on top,
	// and the other lists their longhands' as written: layer i takes item
	// i of each, a list shorter than the pictures starting again and a
	// longer one cut; nil is the initial value alone (BackgroundLayer).
	BackgroundImages      []Image
	BackgroundSizes       []BackgroundSize
	BackgroundRepeats     []BackgroundRepeat
	BackgroundPositions   []BackgroundPosition
	BackgroundAttachments []BackgroundAttachment
	// BackgroundOrigins are background-origin's boxes, the box a picture
	// is placed in (the padding box when InitialBox), and BackgroundClips
	// background-clip's, the box it shows in (the border box when
	// InitialBox).
	BackgroundOrigins, BackgroundClips []BackgroundBox
	// BackgroundBlends are background-blend-mode's: how each layer mixes
	// with the layers and the colour under it.
	BackgroundBlends []BlendMode
	// MaskImages are mask-image's layers, the first on top, and the other
	// lists their longhands', read as the background's are (MaskLayer):
	// the pictures whose alpha, or luminance, the element and all it holds
	// show through (decision 0133).
	MaskImages             []Image
	MaskSizes              []BackgroundSize
	MaskRepeats            []BackgroundRepeat
	MaskPositions          []BackgroundPosition
	MaskOrigins, MaskClips []BackgroundBox
	MaskModes              []MaskMode
	MaskComposites         []MaskComposite
	// ClipPath is clip-path: the shape the element and all it holds are
	// painted and hit in (decision 0119).
	ClipPath ClipPath
	// ObjectFit and ObjectPosition place a replaced element's picture
	// (an img's) in its content box.
	// ImageRendering is image-rendering: how pictures scale.
	ImageRendering ImageRendering
	ObjectFit      ObjectFit
	ObjectPosition BackgroundPosition
	BorderColor    Sides[color.Color]
	BorderStyle    Sides[BorderStyle]
	// BorderHidden marks the sides whose border-style is hidden: none, and
	// in a table of collapsed borders no border at all on that edge.
	BorderHidden Sides[bool]
	BorderImage  Image
	// BorderImageRepeat is border-image-repeat across and down.
	BorderImageRepeat [2]ImageRepeat
	// BorderImageWidth is border-image-width: how thick the border image's
	// edges are drawn, side by side.
	BorderImageWidth Sides[ImageWidth]
	// Radius is each corner's horizontal and vertical radius.
	Radius        Corners[Size[Length]]
	Shadows       []Shadow
	OutlineWidth  float32
	OutlineStyle  BorderStyle
	OutlineColor  color.Color
	OutlineOffset float32
	// ColumnRule* are column-rule: the line drawn between the columns of a
	// multi-column container.
	ColumnRuleWidth float32
	ColumnRuleStyle BorderStyle
	ColumnRuleColor color.Color
	Opacity         float32
	// Visible is visibility: visible, and Collapse visibility: collapse,
	// which takes a table's row or column out with its room.
	Visible  bool
	Collapse bool
	// PointerNone is pointer-events: none: hit testing passes through the
	// box to what is under it.
	PointerNone bool
	// AppRegion is app-region, inherited: drag, its own or the one of the
	// box it is in, has a press there move the system's window.
	AppRegion AppRegion
	// Cursor is the cursor keyword, "" for auto.
	Cursor string
	// UserSelect is user-select as used: auto is its parent's, text on the
	// root.
	UserSelect UserSelect
	// Transform is transform's functions in order, about
	// TransformOrigin (percentages of the border box).
	Transform       []TransformOp
	TransformOrigin Size[Length]
	// Filters is filter's functions in order, none when empty, and
	// BackdropFilters backdrop-filter's.
	Filters         []Filter
	BackdropFilters []Filter
	// BlendMode is mix-blend-mode: how the element and its content, as one
	// picture, mix with what is under them in the group they stack in, and
	// Isolate is isolation: isolate, the element such a group.
	BlendMode   BlendMode
	Isolate     bool
	Transitions []Transition
	Animations  []Animation
}

type TransformKind uint8

const (
	Translate TransformKind = iota
	ScaleOp
	Rotate
	Skew
	Matrix
)

// TransformOp is one transform function: translate(X, Y) with lengths of
// the border box, scale(A[0], A[1]), rotate(A[0] radians), skew(A[0],
// A[1] radians) or matrix(A…).
type TransformOp struct {
	Kind TransformKind
	X, Y Length
	A    [6]float32
}

// BlendMode is how colours mix with those under them, as the
// Compositing and Blending spec names its modes.
type BlendMode uint8

const (
	BlendNormal BlendMode = iota
	BlendMultiply
	BlendScreen
	BlendOverlay
	BlendDarken
	BlendLighten
	BlendColorDodge
	BlendColorBurn
	BlendHardLight
	BlendSoftLight
	BlendDifference
	BlendExclusion
	BlendHue
	BlendSaturation
	BlendColor
	BlendLuminosity
)

// Filtered reports whether the element filters itself or its backdrop,
// which holds its absolute and fixed descendants.
func (l *Look) Filtered() bool { return len(l.Filters) > 0 || len(l.BackdropFilters) > 0 }

// BackgroundBlended reports whether a background layer mixes otherwise
// than normally.
func (l *Look) BackgroundBlended() bool {
	for _, b := range l.BackgroundBlends {
		if b != BlendNormal {
			return true
		}
	}
	return false
}

type FilterKind uint8

const (
	Blur FilterKind = iota
	Brightness
	Contrast
	Grayscale
	HueRotate
	Invert
	OpacityFilter
	Saturate
	Sepia
	DropShadow
)

// Filter is one filter function: blur(Amount px), hue-rotate(Amount
// degrees), drop-shadow(Shadow, its Blur the radius and Spread unused),
// or the others' Amount, 1 for 100%.
type Filter struct {
	Kind   FilterKind
	Amount float32
	Shadow Shadow
}

// CurrentColorFields is a set of colours given as currentColor.
type CurrentColorFields uint8

const (
	BorderTopCurrent CurrentColorFields = 1 << iota
	BorderRightCurrent
	BorderBottomCurrent
	BorderLeftCurrent
	OutlineCurrent
	DecorationCurrent
	ColumnRuleCurrent
)

// FollowColor sets every colour marked currentColor to c, a decoration's
// to -webkit-text-fill-color's when one is set.
func (s *Computed) FollowColor(c color.Color) {
	for f, p := range map[CurrentColorFields]*color.Color{
		BorderTopCurrent: &s.BorderColor.Top, BorderRightCurrent: &s.BorderColor.Right,
		BorderBottomCurrent: &s.BorderColor.Bottom, BorderLeftCurrent: &s.BorderColor.Left,
		OutlineCurrent: &s.OutlineColor, DecorationCurrent: &s.TextDecoration.Color,
		ColumnRuleCurrent: &s.ColumnRuleColor,
	} {
		if s.CurrentColor&f != 0 {
			*p = c
		}
	}
	if s.CurrentColor&DecorationCurrent != 0 && s.TextFilled {
		// Chrome draws a decoration of currentColor in the glyphs' fill.
		s.TextDecoration.Color = s.TextFill
	}
}

// VerticalAlign is a vertical-align: where an inline-level box stands on its line
// (CSS 2 section 10.8), from its parent's baseline, or from the line's top or bottom.
type VerticalAlign uint8

const (
	AlignBaselineText VerticalAlign = iota
	// AlignSub and AlignSuper shift as Chrome does: a fifth of the parent's font size
	// and a pixel down, a third of it and a pixel up.
	AlignSub
	AlignSuper
	AlignTextTop
	AlignTextBottom
	AlignMiddle
	AlignTop
	AlignBottom
	// AlignLength raises the box by the style's VerticalShift.
	AlignLength
)

// ListStyle is a list-style-type: a list item's bullet or marker string,
// drawn outside its box, or a summary's disclosure triangle, first on its
// line. ListDisc is CSS's initial value.
type ListStyle uint8

const (
	ListDisc ListStyle = iota
	ListNone
	ListDisclosureClosed
	ListDisclosureOpen
	ListCircle
	ListSquare
	// ListCounter writes the list item's number in the counter style
	// ListName names, decimal among them.
	ListCounter
	// ListString is the string ListName.
	ListString
)

// BackgroundAttachment is background-attachment.
type BackgroundAttachment uint8

const (
	// AttachScroll places the image in its element's box, where it stays
	// as the element's content scrolls.
	AttachScroll BackgroundAttachment = iota
	// AttachFixed places it in the viewport, shown where the element is.
	AttachFixed
	// AttachLocal places it in the element's content, and it scrolls with
	// it.
	AttachLocal
)

// BackgroundBox is one of an element's boxes, for background-origin and
// background-clip: InitialBox is the property's own initial one.
type BackgroundBox uint8

const (
	InitialBox BackgroundBox = iota
	UnderBorder
	UnderPadding
	UnderContent
	// FillBox, StrokeBox and ViewBox are an SVG shape's boxes a mask may
	// name: an element's content box for the first, its border box for
	// the others.
	FillBox
	StrokeBox
	ViewBox
	// NoClip is mask-clip: no-clip, a mask shown wherever the element
	// draws.
	NoClip
	// UnderText is background-clip: text, a layer shown through the
	// glyphs of the element and of those it holds.
	UnderText
)

// MaskMode is mask-mode: what of a mask layer's picture the element
// shows through.
type MaskMode uint8

const (
	// MaskMatchSource is a picture's alpha, as no layer of ui is an SVG
	// mask element.
	MaskMatchSource MaskMode = iota
	MaskAlpha
	// MaskLuminance is its colours' luminance times their alpha.
	MaskLuminance
)

// MaskComposite is mask-composite: how a mask layer joins the layers
// under it.
type MaskComposite uint8

const (
	// CompositeAdd lays it over them.
	CompositeAdd MaskComposite = iota
	// CompositeSubtract keeps it where they are clear.
	CompositeSubtract
	// CompositeIntersect keeps it where they are opaque.
	CompositeIntersect
	// CompositeExclude keeps it and them where the other is clear.
	CompositeExclude
)

// BackgroundPosition is background-position: X from the left edge, or from
// the right one when FromRight, and Y from the top, or the bottom when
// FromBottom. A percentage places the image's point at the box's point
// (center is 50%), as CSS does.
type BackgroundPosition struct {
	X, Y                  Length
	FromRight, FromBottom bool
}

// BackgroundSize is background-size: cover or contain, else a width and a
// height, each a length, a percentage of the padding box, or auto: the
// image's own size, or what keeps its proportions beside the other.
type BackgroundSize struct {
	Scale         BackgroundScale
	Width, Height Length
}

// BackgroundScale is a background-size that scales the image whole.
type BackgroundScale uint8

const (
	// ScaleNone sizes the image by its Width and Height.
	ScaleNone BackgroundScale = iota
	// ScaleCover is the smallest size covering the padding box, the
	// image's proportions kept.
	ScaleCover
	// ScaleContain is the largest size fitting in it.
	ScaleContain
)

// ObjectFit is object-fit: how an img's picture fills its content box.
type ObjectFit uint8

const (
	// FitFill stretches the picture to the box.
	FitFill ObjectFit = iota
	// FitContain scales it to fit inside, keeping its aspect.
	FitContain
	// FitCover scales it to cover the box, keeping its aspect, cut to it.
	FitCover
	// FitNone keeps its size.
	FitNone
	// FitScaleDown is FitNone or FitContain, whichever is smaller.
	FitScaleDown
)

// BackgroundRepeat is background-repeat, or mask-repeat: how a picture
// fills its area across and down.
type BackgroundRepeat struct{ X, Y Tiling }

// NoRepeat is background-repeat: no-repeat.
//
//nolint:gochecknoglobals // a constant Go can't declare
var NoRepeat = BackgroundRepeat{X: TileNoRepeat, Y: TileNoRepeat}

// Tiling is how a background picture fills one axis of its area.
type Tiling uint8

const (
	// TileRepeat repeats it at its size from where it is placed.
	TileRepeat Tiling = iota
	// TileNoRepeat draws it once.
	TileNoRepeat
	// TileSpace repeats it as often as it fits whole, the first and the
	// last at the area's edges and the room left shared between them.
	TileSpace
	// TileRound repeats it a whole number of times, scaled to fill the
	// area.
	TileRound
)

// BackgroundLayer is one layer of a background: its picture, sized,
// tiled and placed in the box Origin names, shown in the one Clip names.
type BackgroundLayer struct {
	Image        Image
	Size         BackgroundSize
	Repeat       BackgroundRepeat
	Position     BackgroundPosition
	Attachment   BackgroundAttachment
	Origin, Clip BackgroundBox
	Blend        BlendMode
}

// BackgroundLayers is how many layers the background has: one a
// picture, and one for background-image: none.
func (l *Look) BackgroundLayers() int { return max(len(l.BackgroundImages), 1) }

// BackgroundLayer is the background's layer i, 0 on top.
func (l *Look) BackgroundLayer(i int) BackgroundLayer {
	return BackgroundLayer{
		Image:      layerItem(l.BackgroundImages, i, Image{}),
		Size:       layerItem(l.BackgroundSizes, i, BackgroundSize{}),
		Repeat:     layerItem(l.BackgroundRepeats, i, BackgroundRepeat{}),
		Position:   layerItem(l.BackgroundPositions, i, TopLeft),
		Attachment: layerItem(l.BackgroundAttachments, i, AttachScroll),
		Origin:     layerItem(l.BackgroundOrigins, i, InitialBox),
		Clip:       layerItem(l.BackgroundClips, i, InitialBox),
		Blend:      layerItem(l.BackgroundBlends, i, BlendNormal),
	}
}

// MaskLayer is one layer of a mask: its picture placed as a background
// layer's is, in the border box by default, what of it shows and how it
// joins the layers under it.
type MaskLayer struct {
	BackgroundLayer
	Mode      MaskMode
	Composite MaskComposite
}

// MaskLayers is how many layers the mask has, one for mask-image: none.
func (l *Look) MaskLayers() int { return max(len(l.MaskImages), 1) }

// MaskLayer is the mask's layer i, 0 on top.
func (l *Look) MaskLayer(i int) MaskLayer {
	return MaskLayer{
		BackgroundLayer: BackgroundLayer{
			Image:    layerItem(l.MaskImages, i, Image{}),
			Size:     layerItem(l.MaskSizes, i, BackgroundSize{}),
			Repeat:   layerItem(l.MaskRepeats, i, BackgroundRepeat{}),
			Position: layerItem(l.MaskPositions, i, TopLeft),
			Origin:   layerItem(l.MaskOrigins, i, InitialBox),
			Clip:     layerItem(l.MaskClips, i, InitialBox),
		},
		Mode:      layerItem(l.MaskModes, i, MaskMatchSource),
		Composite: layerItem(l.MaskComposites, i, CompositeAdd),
	}
}

// Masked reports whether the element is masked: a layer of its mask has
// a picture, there or not.
func (l *Look) Masked() bool {
	for i := range l.MaskImages {
		if img := &l.MaskImages[i]; img.Source != "" || img.Gradient != nil {
			return true
		}
	}
	return false
}

// HasBackgroundImage reports whether a layer has a picture.
func (l *Look) HasBackgroundImage() bool {
	for i := range l.BackgroundImages {
		if img := &l.BackgroundImages[i]; img.Source != "" || img.Gradient != nil {
			return true
		}
	}
	return false
}

func layerItem[T any](list []T, i int, initial T) T {
	if len(list) == 0 {
		return initial
	}
	return list[i%len(list)]
}

// TopLeft is background-position's initial value, 0% 0%.
//
//nolint:gochecknoglobals // a constant Go can't declare
var TopLeft = BackgroundPosition{X: Length{Kind: Fixed, IsPercent: true}, Y: Length{Kind: Fixed, IsPercent: true}}

// ImageRepeat is border-image-repeat along one axis: how a border image's
// edges and centre fill their length.
type ImageRepeat uint8

const (
	// RepeatStretch stretches the slice over the whole length.
	RepeatStretch ImageRepeat = iota
	// RepeatTile tiles it at its size, centred, cut at both ends.
	RepeatTile
	// RepeatRound tiles it a whole number of times, scaled to fit.
	RepeatRound
	// RepeatSpace tiles it a whole number of times at its size, the
	// leftover shared out as equal gaps.
	RepeatSpace
)

// ImageWidth is one side of border-image-width: V times the border's
// width, V points, V percent of the box's side, or the slice's own size.
type ImageWidth struct {
	Kind ImageWidthKind
	V    float32
}

// ImageWidthKind is what an ImageWidth's V measures.
type ImageWidthKind uint8

const (
	// WidthTimes is a number: V times the border's width.
	WidthTimes ImageWidthKind = iota
	// WidthLength is V points.
	WidthLength
	// WidthPercent is V percent of the box's width, or height for the top
	// and bottom.
	WidthPercent
	// WidthAuto is the slice's own size.
	WidthAuto
)

// ImageRendering is image-rendering.
type ImageRendering uint8

const (
	// RenderAuto smooths a scaled picture.
	RenderAuto ImageRendering = iota
	// RenderPixelated scales it by its nearest pixels.
	RenderPixelated
)

// ColorScheme is a color-scheme value: the schemes an element can be
// drawn in, none for normal, and only, which keeps a browser from
// darkening it by force.
type ColorScheme uint8

const (
	SchemeLight ColorScheme = 1 << iota
	SchemeDark
	SchemeOnly
)

// Dark reports whether an element of scheme s is drawn dark: in the one
// the Document prefers where s has both, else in the one s has, light
// for normal.
func (s ColorScheme) Dark(prefersDark bool) bool {
	if s&SchemeLight != 0 && s&SchemeDark != 0 {
		return prefersDark
	}
	return s&SchemeDark != 0
}
