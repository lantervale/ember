package css

import (
	"slices"
	"strings"
)

// treatment is what a sheet read as a browser's (Browser) gets for a
// vendor's prefixed name its table doesn't list.
type treatment uint8

const (
	// warned leaves the name as written: ui refuses it with the warning it
	// gives any name outside the subset, since Chrome does something with
	// it that ui doesn't.
	warned treatment = iota
	// dropped takes it out without a word, as Chrome does: the
	// declaration, the rule or the at-rule it stands in.
	dropped
	// unmatched is a selector Chrome keeps for a part ui has none of: that
	// selector alone goes, without a word, and its rule keeps the others.
	unmatched
)

func (t treatment) String() string {
	return [...]string{warned: "warned", dropped: "dropped", unmatched: "unmatched"}[t]
}

// vendor is what a sheet read as a browser's gets for one vendor's
// prefixed names. Every name is written without the prefix. Chrome is the
// reference: a name is another name of a plain one where Chrome reads it
// so, goes where Chrome drops it, and keeps its warning where Chrome
// reads it and ui can't.
type vendor struct {
	prefix string

	// same are the properties read as the plain property of the same name,
	// and renamed those read as a plain one of another name. The
	// declaration keeps its place in the cascade.
	same    []string
	renamed map[string]string
	// own are the vendor's properties Chrome reads that are no other name
	// of a plain one, each with why. They stay as written.
	own map[string]string
	// legacy are those of own read as a plain one all the same, each with
	// why Chrome keeps it apart: what Chrome reads apart is what ui
	// reads, or refuses.
	legacy map[string]string
	// words are the words a property's value reads as its plain one's,
	// by property.
	words map[string]map[string]string
	// properties is what any other property gets.
	properties treatment

	// values are the words and functions of a value read as a plain one;
	// a property the tables above alias is read as its plain name there
	// too (transition: -webkit-transform 1s). otherValues is what a
	// declaration ui refuses gets when its value holds another.
	values      map[string]string
	otherValues treatment

	// elements are the pseudo-elements read as a plain one, otherElements
	// what any other gets.
	elements      map[string]string
	otherElements treatment
	// classes are the pseudo-classes read as a plain one, never those
	// Chrome reads that nothing in ui is ever in (unmatched), ownClasses
	// those Chrome reads that ui can't, which stay as written, and
	// otherClasses what any other gets.
	classes      map[string]string
	never        []string
	ownClasses   []string
	otherClasses treatment

	// atRules are the at-rules read as a plain one, otherAtRules what any
	// other gets.
	atRules      map[string]string
	otherAtRules treatment

	// features are the media features read as a plain one, a bare number
	// as dppx; ownFeatures those Chrome answers that ui can't, which keep
	// their warning. Any other is false without a word, as in Chrome.
	features    map[string]string
	ownFeatures []string
}

// Why a -webkit- property is in webkit.own.
const (
	ownChrome  = "Chrome's own, with no plain property"
	ownOldFlex = "the old flexbox of display: -webkit-box, with no plain property"
	ownAxis    = "one axis of the plain property, which has no such longhand"
	ownMask    = "Chrome's own masks, kept apart from mask and its longhands"
	ownBreak   = "takes always, which the break-* property it sets refuses"
	ownUI      = "a property of ui under this very name"
)

// webkit is -webkit-, the prefix Chrome itself reads. Its property tables
// are every name of headless Chrome 157's style object that begins with
// -webkit-: same and renamed those that store their value under a plain
// property, own the others, and four Chrome stores under a plain
// property but reads other values for
// (TestChromeReadsWebkitPropertiesAsTheTable).
//
//nolint:gochecknoglobals,goconst // a vendor's table
var webkit = vendor{
	prefix: "-webkit-",

	same: strings.Fields(`align-content align-items align-self animation animation-delay animation-direction
		animation-duration animation-fill-mode animation-iteration-count animation-name animation-play-state
		animation-timing-function app-region appearance backface-visibility background-clip background-origin
		background-size border-bottom-left-radius border-bottom-right-radius border-radius border-top-left-radius
		border-top-right-radius box-shadow box-sizing clip-path column-count column-gap column-rule
		column-rule-color column-rule-style column-rule-width column-span column-width columns filter flex
		flex-basis flex-direction flex-flow flex-grow flex-shrink flex-wrap font-feature-settings
		hyphenate-character justify-content mask-clip mask-composite mask-image mask-origin mask-repeat mask-size
		opacity order
		perspective perspective-origin print-color-adjust shape-image-threshold shape-margin shape-outside
		text-emphasis text-emphasis-color text-emphasis-position text-emphasis-style text-size-adjust transform
		transform-origin transform-style transition transition-delay transition-duration transition-property
		transition-timing-function user-select`),
	renamed: map[string]string{
		"border-after": "border-block-end", "border-after-color": "border-block-end-color",
		"border-after-style": "border-block-end-style", "border-after-width": "border-block-end-width",
		"border-before": "border-block-start", "border-before-color": "border-block-start-color",
		"border-before-style": "border-block-start-style", "border-before-width": "border-block-start-width",
		"border-end": "border-inline-end", "border-end-color": "border-inline-end-color",
		"border-end-style": "border-inline-end-style", "border-end-width": "border-inline-end-width",
		"border-start": "border-inline-start", "border-start-color": "border-inline-start-color",
		"border-start-style": "border-inline-start-style", "border-start-width": "border-inline-start-width",
		"logical-height": "block-size", "logical-width": "inline-size",
		"max-logical-height": "max-block-size", "max-logical-width": "max-inline-size",
		"min-logical-height": "min-block-size", "min-logical-width": "min-inline-size",
		"margin-after": "margin-block-end", "margin-before": "margin-block-start",
		"margin-end": "margin-inline-end", "margin-start": "margin-inline-start",
		"padding-after": "padding-block-end", "padding-before": "padding-block-start",
		"padding-end": "padding-inline-end", "padding-start": "padding-inline-start",
	},
	own: map[string]string{
		"column-break-after": ownBreak, "column-break-before": ownBreak, "column-break-inside": ownBreak,
		"font-smoothing": ownUI,
		"border-image":   "fills the middle without the fill word, where border-image leaves it empty",
		"box-align":      ownOldFlex, "box-direction": ownOldFlex, "box-flex": ownOldFlex,
		"box-ordinal-group": ownOldFlex, "box-orient": ownUI, "box-pack": ownOldFlex,
		"mask-box-image": ownMask, "mask-box-image-outset": ownMask,
		"mask-box-image-repeat": ownMask, "mask-box-image-slice": ownMask, "mask-box-image-source": ownMask,
		"mask-box-image-width": ownMask, "mask-position-x": ownMask, "mask-position-y": ownMask,
		"perspective-origin-x": ownAxis, "perspective-origin-y": ownAxis, "transform-origin-x": ownAxis,
		"transform-origin-y": ownAxis, "transform-origin-z": ownAxis,
		"border-horizontal-spacing": ownChrome, "border-vertical-spacing": ownChrome,
		"box-decoration-break": ownChrome, "box-reflect": ownChrome, "line-break": ownChrome,
		"line-clamp": ownUI, "locale": ownChrome, "rtl-ordering": ownChrome, "ruby-position": ownChrome,
		"tap-highlight-color": ownChrome, "text-combine": ownChrome, "text-decorations-in-effect": ownChrome,
		"text-fill-color": ownChrome, "text-orientation": ownChrome, "text-security": ownChrome,
		"text-stroke": ownChrome, "text-stroke-color": ownChrome, "text-stroke-width": ownChrome,
		"user-drag": ownChrome, "user-modify": ownChrome, "writing-mode": ownChrome,
	},
	legacy: map[string]string{
		"mask":          "a shorthand of Chrome's own longhands, which take three-value positions and refuse no-clip",
		"mask-position": "a shorthand of -webkit-mask-position-x and -y, which take three-value positions",
	},
	// Chrome reads -webkit-mask-composite's compositing words, the four
	// that are mask-composite's as theirs; the others stay refused.
	words: map[string]map[string]string{
		"mask-composite": {"source-over": "add", "source-in": "intersect", "source-out": "subtract", "xor": "exclude"},
	},
	// Safari's and old Chrome's (-webkit-touch-callout,
	// -webkit-overflow-scrolling…): today's Chrome knows none of them.
	properties: dropped,

	// Chrome lays these out as the plain word. Its other -webkit- values
	// are its own (-webkit-zoom-in, -webkit-gradient()…), and ui can't
	// tell those from the ones Chrome refuses; text-align's
	// (-webkit-center…) and display's (-webkit-box) are in its grammar,
	// the legacy gradients rewritten as plain ones.
	values: map[string]string{
		"calc": "calc", "min-content": "min-content", "max-content": "max-content",
		"fit-content": "fit-content", "fill-available": "stretch", "flex": "flex", "inline-flex": "inline-flex",
		"grab":     "grab",
		"grabbing": "grabbing",
	},
	otherValues: warned,

	elements: map[string]string{"input-placeholder": "placeholder"},
	// Chrome takes any ::-webkit- name as a pseudo-element, most of them
	// parts of its own controls and bars, which ui draws its own way.
	otherElements: unmatched,
	classes:       map[string]string{"any-link": "any-link"},
	never:         []string{"autofill", "drag", "full-page-media", "full-screen", "full-screen-ancestor"},
	// :-webkit-any() is :is() with another weight, so it is not read as it.
	ownClasses:   []string{"any"},
	otherClasses: dropped,

	// @-webkit-keyframes is @keyframes, as Chrome reads it.
	atRules:      map[string]string{atKeyframes: atKeyframes},
	otherAtRules: dropped,

	features: map[string]string{
		"device-pixel-ratio": "resolution", "min-device-pixel-ratio": "min-resolution",
		"max-device-pixel-ratio": "max-resolution",
	},
	ownFeatures: []string{"transform-3d"},
}

// The other vendors' names are nothing to Chrome, which drops each: the
// declaration of a property or of a value it doesn't know, the whole rule
// of a selector, the block of an at-rule, and a media query is false.
//
//nolint:gochecknoglobals // the vendors' tables
var (
	// moz is -moz-, Firefox's.
	moz = vendor{prefix: "-moz-", properties: dropped, otherValues: dropped, otherElements: dropped,
		otherClasses: dropped, otherAtRules: dropped}
	// ms is -ms-, Internet Explorer's and the old Edge's.
	ms = vendor{prefix: "-ms-", properties: dropped, otherValues: dropped, otherElements: dropped,
		otherClasses: dropped, otherAtRules: dropped}
	// opera is -o-, Opera's before it was built on Chrome.
	opera = vendor{prefix: "-o-", properties: dropped, otherValues: dropped, otherElements: dropped,
		otherClasses: dropped, otherAtRules: dropped}
	// khtml is -khtml-, Konqueror's, which the first Safari read too.
	khtml = vendor{prefix: "-khtml-", properties: dropped, otherValues: dropped, otherElements: dropped,
		otherClasses: dropped, otherAtRules: dropped}

	vendors = []*vendor{&webkit, &moz, &ms, &opera, &khtml}
)

// vendorOf is the vendor whose prefix name, in lower case, begins with,
// and the name without it; nil for a name with none of them.
func vendorOf(name string) (*vendor, string) {
	if len(name) < 3 || name[0] != '-' || name[1] == '-' {
		return nil, name
	}
	for _, v := range vendors {
		if rest, ok := strings.CutPrefix(name, v.prefix); ok {
			return v, rest
		}
	}
	return nil, name
}

// property is the plain property the vendor reads rest as, or what the
// declaration gets when there is none.
func (v *vendor) property(rest string) (string, treatment) {
	if as, ok := v.renamed[rest]; ok {
		return as, warned
	}
	if slices.Contains(v.same, rest) {
		return rest, warned
	}
	if _, ok := v.legacy[rest]; ok {
		return rest, warned
	}
	if _, ok := v.own[rest]; ok {
		return "", warned
	}
	return "", v.properties
}

// plainWords reads the words of value that rest, a property of the
// vendor's, reads as its plain one's, in place.
func (v *vendor) plainWords(rest string, value []Component) {
	m := v.words[rest]
	for i := range value {
		if c := &value[i]; c.Kind == Ident {
			if as, ok := m[strings.ToLower(c.Text)]; ok {
				c.Text = as
			}
		}
	}
}

// vendorValues reads the vendor words and functions of a value as their
// plain ones, in place, and reports whether one is left that Chrome
// drops its declaration for.
func vendorValues(value []Component) bool {
	foreign := false
	for i := range value {
		c := &value[i]
		if len(c.Args) > 0 && vendorValues(c.Args) {
			foreign = true
		}
		if c.Kind != Ident && c.Kind != Function {
			continue
		}
		v, rest := vendorOf(strings.ToLower(c.Text))
		if v == nil || v == &webkit && legacyGradient(c, rest) {
			continue
		}
		as, ok := v.values[rest]
		if !ok && c.Kind == Ident {
			as, _ = v.property(rest)
			ok = as != ""
		}
		switch {
		case ok:
			c.Text = as
		case v.otherValues == dropped:
			foreign = true
		case v.otherValues == warned, v.otherValues == unmatched:
			// left for the property's own check
		}
	}
	return foreign
}

// vendorSelectors reads the vendor pseudo-elements and pseudo-classes of
// a selector list as their plain ones and takes out the selectors that
// match nothing in ui. It reports false when Chrome drops the rule, or
// no selector is left.
func vendorSelectors(prelude []Component) ([]Component, bool) {
	var out []Component
	kept, start := 0, 0
	for i := 0; i <= len(prelude); i++ {
		if i < len(prelude) && prelude[i].Kind != Comma {
			continue
		}
		sel := prelude[start:i]
		switch vendorSelector(sel) {
		case dropped:
			return nil, false
		case unmatched:
		case warned:
			if kept > 0 {
				out = append(out, prelude[start-1])
			}
			out = append(out, sel...)
			kept++
		}
		start = i + 1
	}
	return out, kept > 0
}

func vendorSelector(sel []Component) treatment {
	result := warned
	for i := range sel {
		// One inside :not() and the like is the selector's own check's,
		// unless Chrome drops the rule for it.
		if sel[i].Kind == Function && len(sel[i].Args) > 0 && vendorSelector(sel[i].Args) == dropped {
			return dropped
		}
		if sel[i].Kind != Colon || i+1 >= len(sel) {
			continue
		}
		element := sel[i+1].Kind == Colon
		at := i + 1
		if element {
			at++
		}
		if at >= len(sel) || (sel[at].Kind != Ident && sel[at].Kind != Function) || (i > 0 && sel[i-1].Kind == Colon) {
			continue
		}
		v, rest := vendorOf(strings.ToLower(sel[at].Text))
		if v == nil {
			continue
		}
		aliases, other := v.classes, v.otherClasses
		if element {
			aliases, other = v.elements, v.otherElements
		}
		switch as, ok := aliases[rest]; {
		case ok:
			sel[at].Text = as
		case !element && slices.Contains(v.never, rest):
			result = unmatched
		case !element && slices.Contains(v.ownClasses, rest):
			// left for the selector's own check
		case other == dropped:
			return dropped
		case other == unmatched:
			result = unmatched
		case other == warned:
			// left for the selector's own check
		}
	}
	return result
}

// atRule is the plain at-rule the vendor reads rest as, or what the
// at-rule gets when there is none.
func (v *vendor) atRule(rest string) (string, treatment) {
	if as, ok := v.atRules[rest]; ok {
		return as, warned
	}
	return "", v.otherAtRules
}

// feature is the plain media feature the vendor reads rest as; without
// one, known reports whether Chrome answers it, so its warning stays.
func (v *vendor) feature(rest string) (string, bool) {
	if as, ok := v.features[rest]; ok {
		return as, true
	}
	return "", slices.Contains(v.ownFeatures, rest)
}
