package css

import (
	"slices"
	"strings"
)

// pseudoStatus is what ui does with a pseudo-element Chrome knows.
type pseudoStatus uint8

const (
	// The zero status styles something ui has.
	_ pseudoStatus = iota
	// pseudoUnmatched is read and matches nothing: Chrome styles a part
	// with it that no ui element has.
	pseudoUnmatched
	// pseudoRefused drops its rule with a warning: ui has the thing, or
	// an author would look for it, and can't style it this way.
	pseudoRefused
)

// pseudoForm says how a pseudo-element is written.
type pseudoForm uint8

const (
	formName pseudoForm = iota
	formFunction
	formEither
)

// pseudoElement is one pseudo-element of Chrome's.
type pseudoElement struct {
	status pseudoStatus
	form   pseudoForm
	// legacy is set for the four CSS 2 wrote with one colon too.
	legacy bool
	// takes are the properties it takes, longhands and shorthands, nil for
	// every one; any other declaration is left out of its cascade, as in
	// Chrome.
	takes []string
	// why is the warning of a refused one.
	why string
}

//nolint:gochecknoglobals,goconst // the properties each pseudo-element takes
var (
	fontProps = []string{"font", "font-family", "font-size", "font-style", "font-weight", "font-stretch",
		propFontVariations, "line-height"}
	markerProps = append([]string{"color", "content", "cursor", "letter-spacing", "overflow-wrap", "quotes",
		"text-shadow", "text-transform", shortWhiteSpace, propCollapse, propTextWrapMode, "transition", "transition-property",
		"transition-duration",
		"transition-timing-function", "transition-delay", "-webkit-font-smoothing", "word-spacing"}, fontProps...)
	// Longhands only: a shorthand is taken where one of its longhands is.
	firstLineProps = append([]string{"color", "opacity", "visibility", "vertical-align", "box-shadow",
		"-webkit-font-smoothing", "text-shadow", "text-transform", "letter-spacing", "word-spacing",
		"text-decoration-line", "text-decoration-style", "text-decoration-color", "text-decoration-thickness",
		"background-color", "background-image", "background-repeat", "background-size", "background-position",
		"background-attachment", "background-origin", "background-clip"}, fontProps...)
	firstLetterProps = append([]string{"float", propInitialLetter, "margin-top", "margin-right", "margin-bottom",
		"margin-left",
		"padding-top", "padding-right", "padding-bottom", "padding-left", "border-top-width", "border-right-width",
		"border-bottom-width", "border-left-width", "border-top-style", "border-right-style", "border-bottom-style",
		"border-left-style", "border-top-color", "border-right-color", "border-bottom-color", "border-left-color",
		"border-top-left-radius", "border-top-right-radius", "border-bottom-right-radius", "border-bottom-left-radius",
		"border-image-source", "border-image-slice", "border-image-width", "border-image-outset",
		"border-image-repeat"}, firstLineProps...)
	placeholderProps = []string{"color", "opacity"}
	selectionProps   = []string{"color", "background-color", "background", "text-shadow", shortDecoration,
		"text-decoration-line", "text-decoration-style", "text-decoration-color", "text-decoration-thickness",
		"-webkit-text-fill-color", shortStroke, "-webkit-text-stroke-width", "-webkit-text-stroke-color"}
	backdropProps = []string{"background-color", "background", "opacity"}
)

// pseudoElements are the pseudo-elements Chrome 157 reads
// (TestChromeReadsThePseudoElementsOfTheTable), each with what ui does
// with it.
//
//nolint:gochecknoglobals,goconst // lookup table
var pseudoElements = map[string]pseudoElement{
	"before":       {legacy: true},
	"after":        {legacy: true},
	"marker":       {takes: markerProps},
	"first-line":   {legacy: true, takes: firstLineProps},
	"first-letter": {legacy: true, takes: firstLetterProps},
	"placeholder":  {takes: placeholderProps},
	"selection":    {takes: selectionProps},
	"backdrop":     {takes: backdropProps},
	"file-selector-button": {status: pseudoRefused,
		why: "::file-selector-button is not supported: ui has no file input"},
	"details-content": {status: pseudoRefused,
		why: "::details-content is not supported: a details' content has no box of its own in ui"},

	// A scroll container's bars, as Chrome draws them once its
	// ::-webkit-scrollbar is styled (decision 0158).
	webkitScrollbar:            {},
	scrollbarTrack:             {},
	scrollbarTrackPiece:        {},
	scrollbarThumb:             {},
	scrollbarCorner:            {},
	"-webkit-scrollbar-button": {status: pseudoUnmatched},
	"-webkit-resizer":          {status: pseudoUnmatched},

	// The parts of a range, a progress and a meter, as Chrome lays them
	// out once the control's appearance is none (decision 0160).
	sliderContainer:   {},
	sliderTrack:       {},
	sliderThumb:       {},
	progressBar:       {},
	progressValue:     {},
	meterBar:          {},
	meterOptimum:      {},
	meterSuboptimum:   {},
	meterEvenLessGood: {},
	progressInner:     {},
	meterInner:        {},

	"part":                           {status: pseudoUnmatched, form: formFunction},
	"slotted":                        {status: pseudoUnmatched, form: formFunction},
	"highlight":                      {status: pseudoUnmatched, form: formFunction},
	"spelling-error":                 {status: pseudoUnmatched},
	"grammar-error":                  {status: pseudoUnmatched},
	"target-text":                    {status: pseudoUnmatched},
	"search-text":                    {status: pseudoUnmatched},
	"cue":                            {status: pseudoUnmatched, form: formEither},
	"view-transition":                {status: pseudoUnmatched},
	"view-transition-group":          {status: pseudoUnmatched, form: formFunction},
	"view-transition-image-pair":     {status: pseudoUnmatched, form: formFunction},
	"view-transition-old":            {status: pseudoUnmatched, form: formFunction},
	"view-transition-new":            {status: pseudoUnmatched, form: formFunction},
	"view-transition-group-children": {status: pseudoUnmatched, form: formFunction},
	"scroll-marker":                  {status: pseudoUnmatched},
	"scroll-marker-group":            {status: pseudoUnmatched},
	"scroll-button":                  {status: pseudoUnmatched, form: formFunction},
	"checkmark":                      {status: pseudoUnmatched},
	"picker":                         {status: pseudoUnmatched, form: formFunction},
	"picker-icon":                    {status: pseudoUnmatched},
	"column":                         {status: pseudoUnmatched},
	"permission-icon":                {status: pseudoUnmatched},
}

// takesProperty reports whether the pseudo-element name takes the
// property, a longhand or a shorthand.
func takesProperty(name, property string) bool {
	pe := pseudoElements[tableName(name)]
	if pe.takes == nil || strings.HasPrefix(property, "--") {
		return true
	}
	for _, t := range pe.takes {
		if t == property {
			return true
		}
	}
	return false
}

// tableName is the name pseudoElements has name under: marker for a
// generated box's ::marker.
func tableName(name string) string {
	if name == beforeMarker || name == afterMarker {
		return kwMarker
	}
	return name
}

// The parts of a scroll container's bars.
const (
	webkitScrollbar     = "-webkit-scrollbar"
	scrollbarTrack      = "-webkit-scrollbar-track"
	scrollbarTrackPiece = "-webkit-scrollbar-track-piece"
	scrollbarThumb      = "-webkit-scrollbar-thumb"
	scrollbarCorner     = "-webkit-scrollbar-corner"
)

// The parts of the controls drawn from their CSS.
const (
	progressInner     = "-webkit-progress-inner-element"
	meterInner        = "-webkit-meter-inner-element"
	sliderContainer   = "-webkit-slider-container"
	sliderTrack       = "-webkit-slider-runnable-track"
	sliderThumb       = "-webkit-slider-thumb"
	progressBar       = "-webkit-progress-bar"
	progressValue     = "-webkit-progress-value"
	meterBar          = "-webkit-meter-bar"
	meterOptimum      = "-webkit-meter-optimum-value"
	meterSuboptimum   = "-webkit-meter-suboptimum-value"
	meterEvenLessGood = "-webkit-meter-even-less-good-value"
)

// partState is the states of a scrollbar's part that its pseudo-classes
// ask for (::-webkit-scrollbar-thumb:vertical): a selector holds where
// the part is in each.
type partState uint16

const (
	partVertical partState = 1 << iota
	partHorizontal
	partStart
	partEnd
	partCornerPresent
	partEnabled
	partNoButton
	// A hovered and a pressed part: a control's, from under the pointer;
	// no part of ui's bars is ever in them.
	partHover
	partActive
	// The states no part of ui's bars is ever in: buttons, an inactive
	// window's bars.
	partDecrement
	partIncrement
	partSingleButton
	partDoubleButton
	partDisabled
	partWindowInactive
)

//nolint:gochecknoglobals,goconst // lookup table
var partStates = map[string]partState{
	"vertical": partVertical, "horizontal": partHorizontal, "start": partStart, "end": partEnd,
	"corner-present": partCornerPresent, "enabled": partEnabled, "no-button": partNoButton, "hover": partHover,
	"active": partActive, "decrement": partDecrement, "increment": partIncrement, "single-button": partSingleButton,
	"double-button": partDoubleButton, "disabled": partDisabled, "window-inactive": partWindowInactive,
}

// String is the pseudo-classes asking for s, in the order of partState.
func (s partState) String() string {
	var b strings.Builder
	for st := partVertical; st <= partWindowInactive; st <<= 1 {
		if s&st == 0 {
			continue
		}
		for name, v := range partStates {
			if v == st {
				b.WriteString(":" + name)
			}
		}
	}
	return b.String()
}

// ownPart reports whether name is a vendor's pseudo-element ui styles
// under that name, as written: a part of Chrome's scrollbars.
func ownPart(name string) bool {
	pe, ok := pseudoElements[name]
	return ok && pe.status == 0 && strings.HasPrefix(name, "-")
}

// scrollbarPart reports whether name is a part of a scrollbar, which
// takes the pseudo-classes of partStates.
func scrollbarPart(name string) bool {
	return strings.HasPrefix(name, webkitScrollbar) || name == "-webkit-resizer"
}

// controlPart reports whether name is a part of a range, a progress or a
// meter, which takes :hover and :active.
func controlPart(name string) bool {
	switch name {
	case progressInner, meterInner, sliderContainer, sliderTrack, sliderThumb, progressBar, progressValue, meterBar,
		meterOptimum, meterSuboptimum, meterEvenLessGood:
		return true
	}
	return false
}

// partStatesOf is cs without the pseudo-classes that follow a scrollbar's
// or a control's part, and the states they ask for; cs as it is when it
// ends with none.
func partStatesOf(cs []Component) ([]Component, partState) {
	var want partState
	n := len(cs)
	for n >= 3 && cs[n-2].Kind == Colon && cs[n-3].Kind != Colon && cs[n-1].Kind == Ident {
		st, ok := partStates[strings.ToLower(cs[n-1].Text)]
		if !ok {
			break
		}
		want |= st
		n -= 2
	}
	if want == 0 || n < 3 || cs[n-2].Kind != Colon || cs[n-3].Kind != Colon || cs[n-1].Kind != Ident {
		return cs, 0
	}
	if name := strings.ToLower(cs[n-1].Text); !scrollbarPart(name) &&
		(!controlPart(name) || want&^(partHover|partActive) != 0) {
		return cs, 0
	}
	return cs[:n], want
}

// nativeClass is the user agent's pseudo-class of the rules that draw a
// control's own look, which hold while its appearance isn't none: Export
// writes it as an attribute only the controls of appearance: none lack.
const nativeClass = "-ember-native"

// nativeOf is cs without its :-ember-native, and whether it had one.
func nativeOf(cs []Component) ([]Component, bool) {
	for i := 1; i < len(cs); i++ {
		if cs[i].Kind == Ident && cs[i-1].Kind == Colon && (i < 2 || cs[i-2].Kind != Colon) &&
			strings.EqualFold(cs[i].Text, nativeClass) {
			return append(slices.Clip(cs[:i-1]), cs[i+1:]...), true
		}
	}
	return cs, false
}

// beforeMarker and afterMarker name the ::marker of a ::before or an
// ::after that is a list item, as a selector's PseudoElement.
const (
	beforeMarker = "before::marker"
	afterMarker  = "after::marker"
)

// takesDeclared is takesProperty for a declaration's name, a shorthand
// being taken when one of its longhands is.
func takesDeclared(name, property string) bool {
	if takesProperty(name, property) {
		return true
	}
	return slices.ContainsFunc(shorthands[property], func(l string) bool { return takesProperty(name, l) })
}

// windowInactive is cs without the :window-inactive that follows its
// ::selection, and whether it had one: the one pseudo-class Chrome takes
// after a pseudo-element that ui has.
func windowInactive(cs []Component) ([]Component, bool) {
	n := len(cs)
	if n < 4 || cs[n-2].Kind != Colon || cs[n-3].Kind == Colon || !isWord(cs[n-1], "window-inactive") {
		return cs, false
	}
	if name, k, err := pseudoElementAt(cs[:n-2]); err != nil || k == 0 || name != kwSelection {
		return cs, false
	}
	return cs[:n-2], true
}

// pseudoElementAt reads the pseudo-element a complex selector ends with:
// its name and how many components it takes, 0 when it ends with none.
func pseudoElementAt(cs []Component) (string, int, *SelectorError) {
	n := len(cs)
	if n < 2 || cs[n-2].Kind != Colon || cs[n-1].Kind != Ident && cs[n-1].Kind != Function {
		return "", 0, nil
	}
	last := cs[n-1]
	name := strings.ToLower(last.Text)
	pe, known := pseudoElements[name]
	two := n >= 3 && cs[n-3].Kind == Colon
	if !two && (!known || !pe.legacy || last.Kind != Ident) {
		return "", 0, nil
	}
	colons := 2
	if two {
		colons = 3
	}
	switch {
	case !known && strings.HasPrefix(name, "-"):
		return "", 0, selErr(last, "the vendor-prefixed pseudo-element ::%s is not supported", name)
	case !known:
		return "", 0, selErr(last, "unknown pseudo-element ::%s", name)
	case pe.status == pseudoRefused:
		return "", 0, selErr(last, "%s", pe.why)
	case last.Kind == Function && pe.form == formName:
		return "", 0, selErr(last, "::%s takes no ( )", name)
	case last.Kind == Ident && pe.form == formFunction:
		return "", 0, selErr(last, "::%s takes an argument in ( )", name)
	case last.Kind == Function && len(trim(last.Args)) == 0:
		return "", 0, selErr(last, "::%s() is empty", name)
	}
	if name == kwMarker && colons == 3 && n >= 6 && cs[n-4].Kind == Ident && cs[n-5].Kind == Colon &&
		cs[n-6].Kind == Colon && isWord(cs[n-4], "before", "after") {
		return strings.ToLower(cs[n-4].Text) + "::marker", 6, nil
	}
	return name, colons, nil
}
