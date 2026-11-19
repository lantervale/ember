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
	fontProps   = []string{"font", "font-family", "font-size", "font-style", "font-weight", "line-height"}
	markerProps = append([]string{"color", "content", "cursor", "letter-spacing", "overflow-wrap", "quotes",
		"text-shadow", "text-transform", "white-space", "transition", "transition-property", "transition-duration",
		"transition-timing-function", "transition-delay", "-webkit-font-smoothing", "word-spacing"}, fontProps...)
	placeholderProps = []string{"color", "opacity"}
	selectionProps   = []string{"color", "background-color", "background"}
	backdropProps    = []string{"background-color", "background", "opacity"}
)

// pseudoElements are the pseudo-elements Chrome 157 reads
// (TestChromeReadsThePseudoElementsOfTheTable), each with what ui does
// with it.
//
//nolint:gochecknoglobals,goconst // lookup table
var pseudoElements = map[string]pseudoElement{
	"before": {legacy: true},
	"after":  {legacy: true},
	"marker": {takes: markerProps},
	"first-line": {status: pseudoRefused, legacy: true,
		why: "::first-line is not supported: ui styles no line of a block apart"},
	"first-letter": {status: pseudoRefused, legacy: true,
		why: "::first-letter is not supported: ui styles no letter of a block apart"},
	"placeholder": {takes: placeholderProps},
	"selection":   {takes: selectionProps},
	"backdrop":    {takes: backdropProps},
	"file-selector-button": {status: pseudoRefused,
		why: "::file-selector-button is not supported: ui has no file input"},
	"details-content": {status: pseudoRefused,
		why: "::details-content is not supported: a details' content has no box of its own in ui"},

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
	pe := pseudoElements[name]
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

// takesDeclared is takesProperty for a declaration's name, a shorthand
// being taken when one of its longhands is.
func takesDeclared(name, property string) bool {
	if takesProperty(name, property) {
		return true
	}
	return slices.ContainsFunc(shorthands[property], func(l string) bool { return takesProperty(name, l) })
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
	return name, colons, nil
}
