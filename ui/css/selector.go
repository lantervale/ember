package css

import (
	"fmt"
	"math/bits"
	"slices"
	"strconv"
	"strings"

	"github.com/lantervale/ember/ui/internal/dom"
	"golang.org/x/text/unicode/bidi"
)

//nolint:gochecknoglobals,goconst // lookup table
var stateNames = map[string]dom.State{
	"hover": dom.Hover, "focus": dom.Focus, "focus-visible": dom.FocusVisible, "active": dom.Active,
	"disabled": dom.Disabled, "checked": dom.Checked, "open": dom.Open, "modal": dom.Modal,
	"focus-within": dom.FocusWithin, "target": dom.Target, "empty": dom.Empty,
	"placeholder-shown": dom.PlaceholderShown, "default": dom.Default, "valid": dom.Valid, "invalid": dom.Invalid,
	"user-valid": dom.UserValid, "user-invalid": dom.UserInvalid, "in-range": dom.InRange, "out-of-range": dom.OutOfRange,
	"popover-open": dom.PopoverOpen,
}

// Specificity is a selector's (ids, classes, types) weight, compared in
// that order.
type Specificity [3]int

// Less reports whether s weighs less than o.
func (s Specificity) Less(o Specificity) bool {
	for i := range s {
		if s[i] != o[i] {
			return s[i] < o[i]
		}
	}
	return false
}

// Combinator joins two compound selectors.
type Combinator uint8

const (
	Descendant Combinator = iota
	Child
	// Next is a + b, b the element right after a, and Later a ~ b, b any
	// element after a under the same parent.
	Next
	Later
)

func (c Combinator) String() string {
	return [...]string{" ", " > ", " + ", " ~ "}[c]
}

func (c Combinator) sibling() bool { return c == Next || c == Later }

type pseudoKind uint8

const (
	pseudoState pseudoKind = iota
	pseudoFirstChild
	pseudoLastChild
	pseudoRoot
	pseudoNot
	pseudoNthChild
	// pseudoLink is :link and :any-link, an element with an href, and
	// pseudoVisited :visited, which nothing is: ui keeps no history, nor
	// fills fields itself (:autofill).
	pseudoLink
	pseudoVisited
	// pseudoIs is :is(), and pseudoWhere :where(), which weighs nothing.
	pseudoIs
	pseudoWhere
	// pseudoEnabled is a control that is not disabled.
	pseudoEnabled
	pseudoOnlyChild
	pseudoNthLastChild
	pseudoFirstOfType
	pseudoLastOfType
	pseudoOnlyOfType
	pseudoNthOfType
	pseudoNthLastOfType
	// pseudoHas is :has(), whose selectors start at pseudoScope, the
	// element :has() is tested on.
	pseudoHas
	pseudoScope
	pseudoLang
	pseudoReadOnly
	pseudoReadWrite
	// pseudoIndeterminate is a control in that state, or a progress with
	// no value; pseudoRequired and pseudoOptional a control by its
	// required attribute; pseudoDir an element by its direction.
	pseudoIndeterminate
	pseudoRequired
	pseudoOptional
	pseudoDir
)

type pseudo struct {
	kind  pseudoKind
	state dom.State
	// sels are what :not(), :is() and :where() were given, and :has()'s
	// relative selectors, each led by a compound of pseudoScope.
	sels []Selector
	// a and b are :nth-child(An+B)'s, and the other :nth-'s.
	a, b int
	// lang is the range :lang() was given, dir the direction :dir()
	// was, in lower case.
	lang, dir string
}

// Attr is an attribute selector: [Name], or [Name="Value"] when HasValue.
// Op is how the value is matched: '=' the whole, '~' a word of it, '|' it
// or its start before a hyphen, '^' its start, '$' its end, '*' a part.
// Fold matches letters of either case.
type Attr struct {
	Name, Value string
	HasValue    bool
	Op          byte
	Fold        bool
	// caseless is set for the attributes whose values HTML compares
	// whatever the case on its elements.
	caseless bool
}

// matches reports whether the value v matches a, letters of either case
// when fold.
func (a Attr) matches(v string, fold bool) bool {
	if !a.HasValue {
		return true
	}
	want := a.Value
	if fold {
		v, want = strings.ToLower(v), strings.ToLower(want)
	}
	switch a.Op {
	case '~':
		return want != "" && !strings.ContainsAny(want, " \t\n\f\r") && slices.Contains(strings.Fields(v), want)
	case '|':
		return v == want || strings.HasPrefix(v, want+"-")
	case '^':
		return want != "" && strings.HasPrefix(v, want)
	case '$':
		return want != "" && strings.HasSuffix(v, want)
	case '*':
		return want != "" && strings.Contains(v, want)
	}
	return v == want
}

// Compound is a compound selector: a type (TextTag for any) with ids,
// classes, attributes and pseudo-classes, all of which must match.
type Compound struct {
	Tag     dom.Tag
	IDs     []string
	Classes []string
	Attrs   []Attr
	pseudos []pseudo
	text    string
}

// Selector is a complex selector: compounds joined by combinators, read
// left to right; Combinators[i] joins Compounds[i] and Compounds[i+1].
type Selector struct {
	Compounds   []Compound
	Combinators []Combinator
	Specificity Specificity
	// PseudoElement is the selector's pseudo-element, without its colons
	// ("before" for ::before and :before, "part" for ::part(x)); Matches
	// then tests the element it belongs to.
	PseudoElement string
	// pseudoArgs is what a functional pseudo-element was given, as written.
	pseudoArgs string
	// inactive is a ::selection's :window-inactive: it matches only while
	// the Document's window is not the active one.
	inactive bool
	// part is the states a scrollbar part's pseudo-classes ask for.
	part partState
	// native is the user agent's :-ember-native: the rule draws a
	// control's own look, and holds only while its appearance isn't none.
	native bool
}

func (s Selector) String() string {
	var b strings.Builder
	for i, c := range s.Compounds {
		if i > 0 {
			b.WriteString(s.Combinators[i-1].String())
		}
		b.WriteString(c.text)
	}
	if s.PseudoElement != "" {
		b.WriteString("::" + s.PseudoElement)
		if s.pseudoArgs != "" {
			b.WriteString("(" + s.pseudoArgs + ")")
		}
	}
	if s.inactive {
		b.WriteString(":window-inactive")
	}
	b.WriteString(s.part.String())
	if s.native {
		b.WriteString(":" + nativeClass)
	}
	return b.String()
}

// Matches reports whether the element e matches s.
func (s Selector) Matches(e *dom.Node) bool {
	return s.matchFrom(len(s.Compounds)-1, e, nil)
}

// matchFrom reports whether e matches s's compounds up to i, a relative
// selector's first compound being scope: what it matches lies under
// scope's parent, so ancestors are looked for no higher.
func (s Selector) matchFrom(i int, e, scope *dom.Node) bool {
	if !s.Compounds[i].matches(e, scope) {
		return false
	}
	if i == 0 {
		return true
	}
	switch s.Combinators[i-1] {
	case Child:
		p := e.Parent()
		return p != nil && s.matchFrom(i-1, p, scope)
	case Next, Later:
		before := elementsBefore(e)
		for j := len(before) - 1; j >= 0; j-- {
			if before[j].IsText() {
				continue
			}
			if s.matchFrom(i-1, before[j], scope) {
				return true
			}
			if s.Combinators[i-1] == Next {
				return false
			}
		}
		return false
	case Descendant:
	}
	var top *dom.Node
	if scope != nil {
		top = scope.Parent()
	}
	for p := e.Parent(); p != nil && p != top; p = p.Parent() {
		if s.matchFrom(i-1, p, scope) {
			return true
		}
	}
	return false
}

// has reports whether one of rels, relative selectors, matches an element
// below or after a.
func has(rels []Selector, a *dom.Node) bool {
	for _, r := range rels {
		last := len(r.Compounds) - 1
		match := func(e *dom.Node) bool { return r.matchFrom(last, e, a) }
		if !r.Combinators[0].sibling() {
			if last == 1 && r.Combinators[0] == Child {
				if slices.ContainsFunc(a.Children(), func(c *dom.Node) bool { return !c.IsText() && match(c) }) {
					return true
				}
			} else if below(a, match) {
				return true
			}
			continue
		}
		p := a.Parent()
		if p == nil {
			continue
		}
		// The element after a that the relative selector's sibling
		// combinators lead to is the k-th, or one further on past a ~.
		k, later := 0, false
		for _, c := range r.Combinators {
			if !c.sibling() {
				break
			}
			k++
			later = later || c == Later
		}
		deep := k < len(r.Combinators)
		n := 0
		for _, c := range p.Children()[a.Index()+1:] {
			if c.IsText() {
				continue
			}
			if n++; n < k {
				continue
			}
			if match(c) || deep && below(c, match) {
				return true
			}
			if !later {
				break
			}
		}
	}
	return false
}

// below reports whether fn holds for an element under n.
func below(n *dom.Node, fn func(*dom.Node) bool) bool {
	for _, c := range n.Children() {
		if !c.IsText() && (fn(c) || below(c, fn)) {
			return true
		}
	}
	return false
}

// elementsBefore are the children of e's parent before e.
func elementsBefore(e *dom.Node) []*dom.Node {
	p := e.Parent()
	if p == nil {
		return nil
	}
	kids := p.Children()
	return kids[:max(slices.Index(kids, e), 0)]
}

func (c *Compound) matches(e, scope *dom.Node) bool {
	if !c.names(e) {
		return false
	}
	for i := range c.pseudos {
		if p := &c.pseudos[i]; p.kind == pseudoScope && e != scope || p.kind != pseudoScope && !p.matches(e) {
			return false
		}
	}
	return true
}

// names reports whether e has c's type, ids, classes and attributes: all
// of c but its pseudo-classes, so what no state of e changes.
func (c *Compound) names(e *dom.Node) bool {
	if c.Tag != dom.TextTag && c.Tag != e.Tag() {
		return false
	}
	for _, id := range c.IDs {
		if e.ID() != id {
			return false
		}
	}
	for _, cl := range c.Classes {
		if !e.HasClass(cl) {
			return false
		}
	}
	for _, a := range c.Attrs {
		v, ok := e.Attr(a.Name)
		if !ok || !a.matches(v, a.Fold || a.caseless && e.Tag() != dom.Svg) {
			return false
		}
	}
	return true
}

//nolint:gocyclo,cyclop,funlen // one case per pseudo-class
func (p *pseudo) matches(e *dom.Node) bool {
	switch p.kind {
	case pseudoState:
		return e.State().Has(p.state)
	case pseudoFirstChild:
		return edgeChild(e, 1)
	case pseudoLastChild:
		return edgeChild(e, -1)
	case pseudoRoot:
		return e.Parent() == nil
	case pseudoLink:
		_, ok := e.Attr("href")
		return ok
	case pseudoVisited:
		return false
	case pseudoNthChild:
		return nth(p.a, p.b, childIndex(e))
	case pseudoNot:
		return !matchesAny(p.sels, e)
	case pseudoIs, pseudoWhere:
		return matchesAny(p.sels, e)
	case pseudoEnabled:
		return slices.Contains(controls, e.Tag()) && !e.State().Has(dom.Disabled)
	case pseudoOnlyChild:
		return edgeChild(e, 1) && edgeChild(e, -1)
	case pseudoNthLastChild:
		return nth(p.a, p.b, placeAmong(e, -1, false))
	case pseudoFirstOfType:
		return placeAmong(e, 1, true) == 1
	case pseudoLastOfType:
		return placeAmong(e, -1, true) == 1
	case pseudoOnlyOfType:
		return placeAmong(e, 1, true) == 1 && placeAmong(e, -1, true) == 1
	case pseudoNthOfType:
		return nth(p.a, p.b, placeAmong(e, 1, true))
	case pseudoNthLastOfType:
		return nth(p.a, p.b, placeAmong(e, -1, true))
	case pseudoHas:
		return has(p.sels, e)
	case pseudoLang:
		return langMatches(language(e), p.lang)
	case pseudoReadOnly:
		return !editable(e)
	case pseudoReadWrite:
		return editable(e)
	case pseudoIndeterminate:
		return e.State().Has(dom.Indeterminate) || e.Tag() == dom.Progress && !hasAttr(e, "value")
	case pseudoRequired:
		return required(e)
	case pseudoOptional:
		return slices.Contains(fields, e.Tag()) && !required(e)
	case pseudoDir:
		return directionOf(e) == p.dir
	case pseudoScope:
	}
	return true
}

func hasAttr(e *dom.Node, name string) bool { _, ok := e.Attr(name); return ok }

// fields are the elements :optional is about.
//
//nolint:gochecknoglobals // lookup table
var fields = []dom.Tag{dom.Button, dom.Input, dom.Select, dom.Textarea}

// required reports whether e is :required: a select, a textarea, or an
// input of a type a value is asked of, with the required attribute.
func required(e *dom.Node) bool {
	if !hasAttr(e, "required") {
		return false
	}
	if e.Tag() == dom.Input {
		kind, _ := e.Attr("type")
		return !slices.Contains(unrequired, strings.ToLower(kind))
	}
	return e.Tag() == dom.Select || e.Tag() == dom.Textarea
}

// unrequired are the types of input required does not apply to.
//
//nolint:gochecknoglobals // lookup table
var unrequired = strings.Fields("hidden range color submit image reset button")

const (
	dirLTR = "ltr"
	dirRTL = "rtl"
)

// directionOf is e's direction, ltr or rtl: its dir attribute's or that
// of the nearest element above it with one, auto (and a bdi without one)
// read from its text.
func directionOf(e *dom.Node) string {
	for p := e; p != nil; p = p.Parent() {
		v, _ := p.Attr("dir")
		if v = strings.ToLower(v); v == dirLTR || v == dirRTL {
			return v
		}
		if dir := autoDirection(p); dir != "" {
			return dir
		}
	}
	return dirLTR
}

// autoDirection is the direction of e read from its text when it is of
// dir="auto" or a bdi without a valid dir, "" for another element.
func autoDirection(e *dom.Node) string {
	v, _ := e.Attr("dir")
	if v = strings.ToLower(v); v != kwAuto && (e.Tag() != dom.Bdi || v == dirLTR || v == dirRTL) {
		return ""
	}
	if dir := textDirection(e); dir != "" {
		return dir
	}
	return dirLTR
}

// textDirection is the direction of the first strong letter in e's text,
// leaving out the elements that set their own (a bdi, a field, a valid
// dir), a field's that of its value; "" without one.
func textDirection(e *dom.Node) string {
	if e.Tag() == dom.Input || e.Tag() == dom.Textarea {
		if data := dom.Data(e); data.Shadow != nil && !data.Placeholder {
			return strongDirection(data.Shadow.Text())
		}
		return ""
	}
	for _, c := range e.Children() {
		if c.IsText() {
			if dir := strongDirection(c.Text()); dir != "" {
				return dir
			}
			continue
		}
		v, _ := c.Attr("dir")
		if v = strings.ToLower(v); c.Tag() == dom.Bdi || c.Tag() == dom.Textarea || c.Tag() == dom.Input ||
			v == dirLTR || v == dirRTL || v == kwAuto {
			continue
		}
		if dir := textDirection(c); dir != "" {
			return dir
		}
	}
	return ""
}

// strongDirection is the direction of the first letter of s that has
// one, "" for none.
func strongDirection(s string) string {
	for _, r := range s {
		p, _ := bidi.LookupRune(r)
		if c := p.Class(); c == bidi.L {
			return dirLTR
		} else if c == bidi.R || c == bidi.AL {
			return dirRTL
		}
	}
	return ""
}

// language is e's language: the lang attribute of e or of the nearest
// element above it that has one, else the Document's.
func language(e *dom.Node) string {
	for p := e; ; p = p.Parent() {
		if v, ok := p.Attr("lang"); ok {
			return v
		}
		if p.Parent() == nil {
			return dom.Data(p).Language
		}
	}
}

// langMatches reports whether the language tag is in the range :lang()
// was given, as Chrome has it: the range or it followed by a hyphen,
// whatever the case, both well formed, and a wildcard matching nothing.
func langMatches(tag, rng string) bool {
	if !wellFormed(tag) || !wellFormed(rng) || len(tag) < len(rng) || !strings.EqualFold(tag[:len(rng)], rng) {
		return false
	}
	return len(tag) == len(rng) || tag[len(rng)] == '-'
}

// wellFormed reports whether s is subtags of ASCII letters and digits
// joined by hyphens, the first of letters alone.
func wellFormed(s string) bool {
	for i, sub := range strings.Split(s, "-") {
		if sub == "" {
			return false
		}
		for _, r := range sub {
			letter := r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z'
			if !letter && (i == 0 || r < '0' || r > '9') {
				return false
			}
		}
	}
	return true
}

// editable reports whether the user can edit e, what :read-write
// matches: a text field neither read-only nor disabled, or an element a
// contenteditable on it or above it makes editable.
func editable(e *dom.Node) bool {
	if e.Tag() == dom.Input {
		kind, _ := e.Attr("type")
		return !slices.Contains(uneditable, strings.ToLower(kind)) && usable(e)
	}
	if e.Tag() == dom.Textarea {
		return usable(e)
	}
	for p := e; p != nil; p = p.Parent() {
		if v, ok := p.Attr("contenteditable"); ok {
			switch strings.ToLower(v) {
			case "", "true", "plaintext-only":
				return true
			case "false":
				return false
			}
		}
	}
	return false
}

// usable reports whether a field is neither read-only nor disabled.
func usable(e *dom.Node) bool {
	_, ro := e.Attr("readonly")
	_, off := e.Attr("disabled")
	return !ro && !off && !e.State().Has(dom.Disabled)
}

// uneditable are the types of input no text is typed in.
//
//nolint:gochecknoglobals,goconst // lookup table
var uneditable = []string{"checkbox", "radio", "range", "color", "file", "hidden", "submit", "reset", "button", "image"}

// controls are the elements :enabled is about.
//
//nolint:gochecknoglobals // lookup table
var controls = []dom.Tag{dom.Button, dom.Input, dom.Select, dom.Textarea, dom.Option, dom.Fieldset}

func matchesAny(sels []Selector, e *dom.Node) bool {
	for _, s := range sels {
		if s.Matches(e) {
			return true
		}
	}
	return false
}

// placeAmong is e's place among its parent's element children, of its
// type only when sameType, counted from the first (dir 1) or the last
// (dir -1), from 1; 0 without a parent.
func placeAmong(e *dom.Node, dir int, sameType bool) int {
	p := e.Parent()
	if p == nil {
		return 0
	}
	kids := p.Children()
	i, end := 0, len(kids)
	if dir < 0 {
		i, end = len(kids)-1, -1
	}
	n := 0
	for ; i != end; i += dir {
		c := kids[i]
		if c.IsText() || sameType && c.Tag() != e.Tag() {
			continue
		}
		n++
		if c == e {
			return n
		}
	}
	return 0
}

// parseNth reads :nth-child()'s argument, odd, even or An+B, as a and b.
func parseNth(src string) (int, int, bool) {
	s := strings.ToLower(strings.Join(strings.Fields(src), ""))
	switch s {
	case "odd":
		return 2, 1, true
	case "even":
		return 2, 0, true
	}
	i := strings.IndexByte(s, 'n')
	if i < 0 {
		n, err := strconv.Atoi(s)
		return 0, n, err == nil
	}
	var a, b int
	switch an := s[:i]; an {
	case "", "+":
		a = 1
	case "-":
		a = -1
	default:
		v, err := strconv.Atoi(an)
		if err != nil {
			return 0, 0, false
		}
		a = v
	}
	if rest := s[i+1:]; rest != "" {
		if rest[0] != '+' && rest[0] != '-' {
			return 0, 0, false
		}
		v, err := strconv.Atoi(rest)
		if err != nil {
			return 0, 0, false
		}
		b = v
	}
	return a, b, true
}

// nth reports whether the element child at place i, from 1, is a·n+b for
// some n ≥ 0.
func nth(a, b, i int) bool {
	if i < 1 {
		return false
	}
	if a == 0 {
		return i == b
	}
	return (i-b)%a == 0 && (i-b)/a >= 0
}

// childIndex is e's place among its parent's element children, from 1; 0
// without a parent.
func childIndex(e *dom.Node) int {
	p := e.Parent()
	if p == nil {
		return 0
	}
	i := 0
	for _, c := range p.Children() {
		if c.IsText() {
			continue
		}
		i++
		if c == e {
			return i
		}
	}
	return 0
}

// edgeChild reports whether e is its parent's first (dir 1) or last (dir
// -1) element child; text nodes don't count.
func edgeChild(e *dom.Node, dir int) bool {
	p := e.Parent()
	if p == nil {
		return false
	}
	kids := p.Children()
	i, end := 0, len(kids)
	if dir < 0 {
		i, end = len(kids)-1, -1
	}
	for ; i != end; i += dir {
		if !kids[i].IsText() {
			return kids[i] == e
		}
	}
	return false
}

// ParseSelectors reads a selector list such as "nav > .item:hover, #main".
func ParseSelectors(src string) ([]Selector, error) {
	p := parser{toks: Tokenize(src)}
	var cs []Component
	for p.peek().Kind != EOF {
		cs = append(cs, p.component())
	}
	sels, err := parseSelectors(cs)
	if err != nil {
		return nil, err
	}
	return sels, nil
}

// SelectorError is a selector outside the subset or badly written.
type SelectorError struct {
	Pos Pos
	Msg string
	// unmatched is a selector naming an element ui has none of: read by a
	// browser, it matches nothing here.
	unmatched bool
}

func (e *SelectorError) Error() string {
	return fmt.Sprintf("%d:%d: %s", e.Pos.Line, e.Pos.Col, e.Msg)
}

func selErr(c Component, format string, args ...any) *SelectorError {
	return &SelectorError{Pos: c.Pos, Msg: fmt.Sprintf(format, args...)}
}

func parseSelectors(cs []Component) ([]Selector, *SelectorError) {
	var out []Selector
	start := 0
	for i := 0; i <= len(cs); i++ {
		if i < len(cs) && cs[i].Kind != Comma {
			continue
		}
		part := trim(cs[start:i])
		if len(part) == 0 {
			at := Component{Token: Token{Pos: Pos{1, 1}}}
			switch {
			case i < len(cs):
				at = cs[i]
			case start > 0:
				at = cs[start-1]
			}
			return nil, selErr(at, "empty selector")
		}
		s, err := parseComplex(part)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
		start = i + 1
	}
	return out, nil
}

// matched are the selectors of a list a browser's sheet keeps: those
// naming an element ui has none of match nothing and go without a word,
// as the vendors' parts do.
func matched(cs []Component) []Component {
	var out []Component
	for _, part := range splitOn(cs, func(c Component) bool { return c.Kind == Comma }) {
		if sel := trim(part); len(sel) > 0 {
			if _, err := parseComplex(sel); err != nil && err.unmatched {
				continue
			}
		}
		if out != nil {
			out = append(out, Component{Token: Token{Kind: Comma, Text: ","}})
		}
		out = append(slices.Grow(out, len(part)), part...)
	}
	return out
}

func parseComplex(cs []Component) (Selector, *SelectorError) {
	var s Selector
	cs, s.native = nativeOf(cs)
	cs, s.part = partStatesOf(cs)
	cs, s.inactive = windowInactive(cs)
	name, n, err := pseudoElementAt(cs)
	if err != nil {
		return Selector{}, err
	}
	if n > 0 {
		s.PseudoElement = name
		if last := cs[len(cs)-1]; last.Kind == Function {
			s.pseudoArgs = join(last.Args)
		}
		cs = cs[:len(cs)-n]
		if len(cs) == 0 || cs[len(cs)-1].Kind == Whitespace || cs[len(cs)-1].Kind == Delim && cs[len(cs)-1].Text != "*" {
			cs = append(slices.Clip(cs), Component{Token: Token{Kind: Delim, Text: "*", Pos: cs0Pos(cs)}})
		}
	}
	i := 0
	for {
		c, read, bad := parseCompound(cs[i:])
		if bad != nil {
			return Selector{}, bad
		}
		s.Compounds = append(s.Compounds, c)
		i += read
		sawSpace := false
		for i < len(cs) && cs[i].Kind == Whitespace {
			sawSpace = true
			i++
		}
		if i == len(cs) {
			break
		}
		comb := Descendant
		if t := cs[i]; t.Kind == Delim {
			switch t.Text {
			case ">":
				comb = Child
				i++
			case "+":
				comb = Next
				i++
			case "~":
				comb = Later
				i++
			}
		}
		if comb == Descendant && !sawSpace {
			return Selector{}, selErr(cs[i], "unexpected %q in selector", cs[i].String())
		}
		for i < len(cs) && cs[i].Kind == Whitespace {
			i++
		}
		if i == len(cs) {
			return Selector{}, selErr(cs[i-1], "selector ends with a combinator")
		}
		s.Combinators = append(s.Combinators, comb)
	}
	s.Specificity = s.specificity()
	return s, nil
}

// specificity is the selector's: its compounds', a pseudo-element's, and
// a ::selection's :window-inactive.
func (s *Selector) specificity() Specificity {
	var sp Specificity
	for _, c := range s.Compounds {
		sp = sp.add(c.specificity())
	}
	if s.PseudoElement != "" {
		sp[2]++
	}
	if s.inactive {
		sp[1]++
	}
	sp[1] += bits.OnesCount16(uint16(s.part))
	if s.native {
		sp[1]++
	}
	if s.PseudoElement == beforeMarker || s.PseudoElement == afterMarker {
		sp[2]++
	}
	return sp
}

func cs0Pos(cs []Component) Pos {
	if len(cs) > 0 {
		return cs[len(cs)-1].Pos
	}
	return Pos{1, 1}
}

func (s Specificity) add(o Specificity) Specificity {
	return Specificity{s[0] + o[0], s[1] + o[1], s[2] + o[2]}
}

func (c *Compound) specificity() Specificity {
	sp := Specificity{len(c.IDs), len(c.Classes) + len(c.Attrs), 0}
	if c.Tag != dom.TextTag {
		sp[2]++
	}
	for _, p := range c.pseudos {
		switch p.kind {
		case pseudoWhere, pseudoScope:
		case pseudoNot, pseudoIs, pseudoHas:
			var most Specificity
			for _, s := range p.sels {
				if most.Less(s.Specificity) {
					most = s.Specificity
				}
			}
			sp = sp.add(most)
		case pseudoState, pseudoFirstChild, pseudoLastChild, pseudoRoot, pseudoNthChild, pseudoLink, pseudoVisited,
			pseudoEnabled, pseudoOnlyChild, pseudoNthLastChild, pseudoFirstOfType, pseudoLastOfType, pseudoOnlyOfType,
			pseudoNthOfType, pseudoNthLastOfType, pseudoLang, pseudoReadOnly, pseudoReadWrite, pseudoIndeterminate,
			pseudoRequired, pseudoOptional, pseudoDir:
			sp[1]++
		}
	}
	return sp
}

func parseCompound(cs []Component) (Compound, int, *SelectorError) {
	var c Compound
	i := 0
	if i < len(cs) {
		switch t := cs[i]; {
		case t.Kind == Ident:
			tag, ok := dom.TagNamed(strings.ToLower(t.Text))
			if !ok {
				err := selErr(t, "ui has no <%s> element", strings.ToLower(t.Text))
				err.unmatched = true
				return c, 0, err
			}
			c.Tag = tag
			i++
		case t.Kind == Delim && t.Text == "*":
			i++
		case t.Kind == Delim && t.Text == "|":
			return c, 0, selErr(t, "namespaces are not supported")
		}
	}
	for i < len(cs) {
		t := cs[i]
		switch {
		case t.Kind == Hash && t.HasID:
			c.IDs = append(c.IDs, t.Text)
			i++
		case t.Kind == Delim && t.Text == ".":
			if i+1 >= len(cs) || cs[i+1].Kind != Ident {
				return c, 0, selErr(t, "expected a class name after '.'")
			}
			c.Classes = append(c.Classes, cs[i+1].Text)
			i += 2
		case t.Kind == Colon:
			p, n, err := parsePseudo(cs[i:])
			if err != nil {
				return c, 0, err
			}
			c.pseudos = append(c.pseudos, p)
			i += n
		case t.Kind == LBracket:
			a, err := parseAttr(t)
			if err != nil {
				return c, 0, err
			}
			c.Attrs = append(c.Attrs, a)
			i++
		case t.Kind == Ident || t.Kind == Delim && t.Text == "*":
			return c, 0, selErr(t, "a type selector must come first in a compound")
		default:
			if i == 0 {
				return c, 0, selErr(t, "unexpected %q in selector", t.String())
			}
			c.text = join(cs[:i])
			return c, i, nil
		}
	}
	c.text = join(cs[:i])
	return c, i, nil
}

func parsePseudo(cs []Component) (pseudo, int, *SelectorError) {
	if len(cs) < 2 {
		return pseudo{}, 0, selErr(cs[0], "expected a pseudo-class after ':'")
	}
	t := cs[1]
	switch t.Kind {
	case Colon:
		if len(cs) > 2 && (cs[2].Kind == Ident || cs[2].Kind == Function) {
			name := strings.ToLower(cs[2].Text)
			if _, ok := pseudoElements[name]; ok {
				return pseudo{}, 0, selErr(t, "::%s must end its selector: nothing may follow a pseudo-element", name)
			}
			return pseudo{}, 0, selErr(t, "unknown pseudo-element ::%s", name)
		}
		return pseudo{}, 0, selErr(t, "expected a pseudo-element after '::'")
	case Ident:
		return pseudoClass(t)
	case Function:
		return pseudoFunction(t)
	case EOF, AtKeyword, Hash, String, BadString, URL, BadURL, Delim, Number, Percentage, Dimension, Whitespace, CDO, CDC,
		Semicolon, Comma, LBracket, RBracket, LParen, RParen, LBrace, RBrace:
		// nothing special for these tokens
	}
	return pseudo{}, 0, selErr(t, "expected a pseudo-class after ':'")
}

// pseudoClass reads the pseudo-class a name after ':' is.
func pseudoClass(t Component) (pseudo, int, *SelectorError) {
	name := strings.ToLower(t.Text)
	if pseudoElements[name].legacy {
		return pseudo{}, 0, selErr(t, ":%s must end its selector: nothing may follow a pseudo-element", name)
	}
	if st, ok := stateNames[name]; ok {
		return pseudo{kind: pseudoState, state: st}, 2, nil
	}
	switch name {
	case "first-child":
		return pseudo{kind: pseudoFirstChild}, 2, nil
	case "last-child":
		return pseudo{kind: pseudoLastChild}, 2, nil
	case "root":
		return pseudo{kind: pseudoRoot}, 2, nil
	case "link", classAnyLink:
		return pseudo{kind: pseudoLink}, 2, nil
	case "visited":
		return pseudo{kind: pseudoVisited}, 2, nil
	case "enabled":
		return pseudo{kind: pseudoEnabled}, 2, nil
	case "only-child":
		return pseudo{kind: pseudoOnlyChild}, 2, nil
	case "first-of-type":
		return pseudo{kind: pseudoFirstOfType}, 2, nil
	case "last-of-type":
		return pseudo{kind: pseudoLastOfType}, 2, nil
	case "only-of-type":
		return pseudo{kind: pseudoOnlyOfType}, 2, nil
	case "read-only":
		return pseudo{kind: pseudoReadOnly}, 2, nil
	case "read-write":
		return pseudo{kind: pseudoReadWrite}, 2, nil
	case "indeterminate":
		return pseudo{kind: pseudoIndeterminate}, 2, nil
	case "required":
		return pseudo{kind: pseudoRequired}, 2, nil
	case "optional":
		return pseudo{kind: pseudoOptional}, 2, nil
	case "autofill", "-webkit-autofill", "window-inactive":
		// matching nothing: no field is filled in, and an element is never
		// inactive, only its ::selection
		return pseudo{kind: pseudoVisited}, 2, nil
	}
	return pseudo{}, 0, selErr(t, "the :%s pseudo-class is not supported", name)
}

// pseudoFunction reads the pseudo-class a function after ':' is.
func pseudoFunction(t Component) (pseudo, int, *SelectorError) {
	name := strings.ToLower(t.Text)
	if kind, counts := nthKinds[name]; counts {
		a, b, ok := parseNth(join(t.Args))
		if !ok {
			return pseudo{}, 0, selErr(t, ":%s() takes odd, even or An+B", name)
		}
		return pseudo{kind: kind, a: a, b: b}, 2, nil
	}
	switch name {
	case "is":
		return pseudo{kind: pseudoIs, sels: forgiving(t.Args)}, 2, nil
	case "where":
		return pseudo{kind: pseudoWhere, sels: forgiving(t.Args)}, 2, nil
	case "has":
		rels, err := relatives(t)
		return pseudo{kind: pseudoHas, sels: rels}, 2, err
	case "lang":
		if v := trim(t.Args); len(v) == 1 && v[0].Kind == Ident {
			return pseudo{kind: pseudoLang, lang: v[0].Text}, 2, nil
		}
		return pseudo{}, 0, selErr(t, ":lang() takes one language range, as an identifier")
	case "dir":
		if v := trim(t.Args); len(v) == 1 && v[0].Kind == Ident {
			return pseudo{kind: pseudoDir, dir: strings.ToLower(v[0].Text)}, 2, nil
		}
		return pseudo{}, 0, selErr(t, ":dir() takes ltr or rtl")
	case "not":
	default:
		return pseudo{}, 0, selErr(t, "the :%s() pseudo-class is not supported", name)
	}
	if len(trim(t.Args)) == 0 {
		return pseudo{}, 0, selErr(t, ":not() is empty")
	}
	sels, err := parseSelectors(t.Args)
	if err != nil {
		err.unmatched = false
		return pseudo{}, 0, err
	}
	return pseudo{kind: pseudoNot, sels: sels}, 2, nil
}

// relatives reads the relative selectors :has() was given, each from the
// element tested: a selector naming an element ui has none of is left
// out, as it matches nothing, and anything else wrong refuses the rule,
// a :has() inside another included, as Chrome refuses it.
func relatives(t Component) ([]Selector, *SelectorError) {
	if len(trim(t.Args)) == 0 {
		return nil, selErr(t, ":has() is empty")
	}
	var out []Selector
	for _, part := range splitOn(t.Args, func(c Component) bool { return c.Kind == Comma }) {
		part = trim(part)
		if len(part) == 0 {
			return nil, selErr(t, "an empty selector in :has()")
		}
		lead := Descendant
		if c := part[0]; c.Kind == Delim {
			switch c.Text {
			case ">":
				lead = Child
			case "+":
				lead = Next
			case "~":
				lead = Later
			}
			if lead != Descendant {
				if part = trim(part[1:]); len(part) == 0 {
					return nil, selErr(c, "nothing after %q in :has()", c.Text)
				}
			}
		}
		s, err := parseComplex(part)
		switch {
		case err != nil && err.unmatched:
			continue
		case err != nil:
			return nil, err
		case s.PseudoElement != "":
			return nil, selErr(t, "::%s in :has()", s.PseudoElement)
		}
		if err = dropHas(s, t); err != nil {
			return nil, err
		}
		s.Compounds = append([]Compound{{pseudos: []pseudo{{kind: pseudoScope}}}}, s.Compounds...)
		s.Combinators = append([]Combinator{lead}, s.Combinators...)
		out = append(out, s)
	}
	return out, nil
}

// dropHas leaves the selectors holding a :has() out of the :is() and
// :where() of s, a selector :has() was given, and refuses one that stands
// elsewhere in it.
func dropHas(s Selector, at Component) *SelectorError {
	for i := range s.Compounds {
		c := &s.Compounds[i]
		for j := range c.pseudos {
			p := &c.pseudos[j]
			switch p.kind {
			case pseudoHas:
				return selErr(at, ":has() inside :has()")
			case pseudoIs, pseudoWhere:
				p.sels = slices.DeleteFunc(p.sels, func(n Selector) bool { return dropHas(n, at) != nil })
			case pseudoNot:
				for _, n := range p.sels {
					if err := dropHas(n, at); err != nil {
						return err
					}
				}
			case pseudoState, pseudoFirstChild, pseudoLastChild, pseudoRoot, pseudoNthChild, pseudoLink, pseudoVisited,
				pseudoEnabled, pseudoOnlyChild, pseudoNthLastChild, pseudoFirstOfType, pseudoLastOfType, pseudoOnlyOfType,
				pseudoNthOfType, pseudoNthLastOfType, pseudoScope, pseudoLang, pseudoReadOnly, pseudoReadWrite,
				pseudoIndeterminate, pseudoRequired, pseudoOptional, pseudoDir:
			}
		}
	}
	return nil
}

// caselessAttrs are the attributes whose values a selector compares
// whatever the case on an HTML element: Chrome's list, HTML 4's.
//
//nolint:gochecknoglobals // lookup table
var caselessAttrs = func() map[string]bool {
	set := map[string]bool{}
	for _, name := range strings.Fields(`accept accept-charset align alink axis bgcolor charset checked clear codetype
		color compact declare defer dir direction disabled enctype face frame hreflang http-equiv lang language link
		media method multiple nohref noresize noshade nowrap readonly rel rev rules scope scrolling selected shape
		target text type valign valuetype vlink`) {
		set[name] = true
	}
	return set
}()

//nolint:gochecknoglobals // lookup table
var nthKinds = map[string]pseudoKind{
	"nth-child": pseudoNthChild, "nth-last-child": pseudoNthLastChild,
	"nth-of-type": pseudoNthOfType, "nth-last-of-type": pseudoNthLastOfType,
}

// forgiving reads the selectors :is() and :where() were given, leaving
// out the ones ui can't read, as CSS has it: with none left it matches
// nothing.
func forgiving(args []Component) []Selector {
	var out []Selector
	for _, part := range splitOn(args, func(c Component) bool { return c.Kind == Comma }) {
		if part = trim(part); len(part) == 0 {
			continue
		}
		if s, err := parseComplex(part); err == nil && s.PseudoElement == "" {
			out = append(out, s)
		}
	}
	return out
}

// parseAttr reads [name], [name="value"] and the other matchers (~=, |=,
// ^=, $=, *=), with the i flag; Chrome reads no s flag.
func parseAttr(b Component) (Attr, *SelectorError) {
	v := items(b.Args)
	if len(v) == 0 || v[0].Kind != Ident {
		return Attr{}, selErr(b, "expected an attribute name in [ ]")
	}
	a := Attr{Name: strings.ToLower(v[0].Text), Op: '='}
	a.caseless = caselessAttrs[a.Name]
	if len(v) == 1 {
		return a, nil
	}
	i := 1
	if v[i].Kind == Delim && len(v[i].Text) == 1 && strings.ContainsAny(v[i].Text, "^$*|~") {
		a.Op = v[i].Text[0]
		i++
	}
	if i >= len(v) || v[i].Kind != Delim || v[i].Text != "=" {
		at := v[min(i, len(v)-1)]
		return Attr{}, selErr(at, "unexpected %q in [ ]", at.String())
	}
	i++
	if i >= len(v) || v[i].Kind != Ident && v[i].Kind != String {
		return Attr{}, selErr(b, "expected [%s=value]", a.Name)
	}
	a.Value, a.HasValue = v[i].Text, true
	switch rest := v[i+1:]; {
	case len(rest) == 0:
	case len(rest) == 1 && isWord(rest[0], "i"):
		a.Fold = true
	default:
		return Attr{}, selErr(rest[0], "unexpected %q in [ ]", rest[0].String())
	}
	return a, nil
}

// stateRead is a state a selector reads on an element other than its
// subject, :not(), :is() and :where() included: a change of it there may
// change what the selector matches.
type stateRead struct {
	states dom.State
	// on is the compound that reads it, after those holding it in a
	// pseudo-class of theirs on the same element, and subject the
	// selector's last.
	on      []*Compound
	subject *Compound
	// beside is set when the first combinator from on toward the subject
	// is a sibling's: the selector then matches among the elements after
	// the one in the state and under them, else under that one.
	beside bool
	rule   int
}

// names reports whether e has what the compounds reading the state name.
func (r *stateRead) names(e *dom.Node) bool {
	return !slices.ContainsFunc(r.on, func(c *Compound) bool { return !c.names(e) })
}

// stateReads calls add for each compound of s, and of the selectors its
// pseudo-classes hold, that reads states on an element other than s's
// subject, with on and beside as stateRead has them.
func (s Selector) stateReads(add func(on []*Compound, states dom.State, beside bool)) {
	s.readsFrom(nil, false, false, add)
}

// readsFrom is stateReads for a selector the compounds outer hold, whose
// subject stands above the one asked about (apart), or before it (apart
// and beside).
func (s Selector) readsFrom(outer []*Compound, apart, beside bool,
	add func(on []*Compound, states dom.State, beside bool)) {
	for i := range s.Compounds {
		c := &s.Compounds[i]
		on, away, before := append(slices.Clip(outer), c), apart, beside
		if i < len(s.Combinators) {
			on, away, before = on[len(outer):], true, s.Combinators[i].sibling()
		}
		var states dom.State
		for j := range c.pseudos {
			p := &c.pseudos[j]
			states |= p.reads()
			for _, n := range p.nested() {
				n.readsFrom(on, away, before, add)
			}
		}
		if away && states != 0 {
			add(on, states, before)
		}
	}
}

// reads is the state p itself tests.
func (p *pseudo) reads() dom.State {
	switch p.kind {
	case pseudoState:
		return p.state
	case pseudoEnabled, pseudoReadOnly, pseudoReadWrite:
		return dom.Disabled
	case pseudoIndeterminate:
		return dom.Indeterminate
	case pseudoFirstChild, pseudoLastChild, pseudoRoot, pseudoNot, pseudoNthChild, pseudoLink, pseudoVisited, pseudoIs,
		pseudoWhere, pseudoOnlyChild, pseudoNthLastChild, pseudoFirstOfType, pseudoLastOfType, pseudoOnlyOfType,
		pseudoNthOfType, pseudoNthLastOfType, pseudoHas, pseudoScope, pseudoLang, pseudoRequired, pseudoOptional,
		pseudoDir:
	}
	return 0
}

// nested are the selectors p holds that read the element p is on, or
// elements round it: those of :not(), :is() and :where(), not :has()'s,
// whose elements are below or after it.
func (p *pseudo) nested() []Selector {
	if p.kind == pseudoHas {
		return nil
	}
	return p.sels
}

// beside reports whether the compound at i stands before a sibling
// combinator: what it reads on an element says what the elements after
// that one match.
func (s Selector) beside(i int) bool {
	return i < len(s.Combinators) && s.Combinators[i].sibling()
}

// besideKeys are the classes, ids and attributes s reads on an element
// before a sibling combinator, keyed as dom.Pipeline.Changed.
func (s Selector) besideKeys(add func(string)) {
	for i, c := range s.Compounds {
		if s.beside(i) {
			c.keys(add)
		}
		for _, p := range c.pseudos {
			for _, n := range p.nested() {
				n.besideKeys(add)
				if s.beside(i) {
					for _, nc := range n.Compounds {
						nc.keys(add)
					}
				}
			}
		}
	}
}

// siblings reports whether s has a sibling combinator.
func (s Selector) siblings() bool {
	for i, c := range s.Compounds {
		if s.beside(i) {
			return true
		}
		for _, p := range c.pseudos {
			if slices.ContainsFunc(p.nested(), Selector.siblings) {
				return true
			}
		}
	}
	return false
}

// aboveKeys adds the classes, ids and attributes s reads in a compound
// above its subject, keyed as dom.Pipeline.Changed: a change of those
// on an element can restyle its descendants.
func (s Selector) aboveKeys(add func(string)) {
	for i, c := range s.Compounds {
		above := i < len(s.Compounds)-1
		if above {
			c.keys(add)
		}
		for _, p := range c.pseudos {
			for _, n := range p.nested() {
				if above {
					for _, nc := range n.Compounds {
						nc.keys(add)
					}
				} else {
					n.aboveKeys(add)
				}
			}
		}
	}
}

func (c Compound) keys(add func(string)) {
	for _, id := range c.IDs {
		add("#" + id)
	}
	for _, cl := range c.Classes {
		add("." + cl)
	}
	for _, a := range c.Attrs {
		add("[" + a.Name)
	}
	for _, p := range c.pseudos {
		for _, k := range p.attrs() {
			add(k)
		}
	}
}

// attrs are the attributes p reads, keyed as dom.Pipeline.Changed.
func (p *pseudo) attrs() []string {
	switch p.kind {
	case pseudoLang:
		return []string{"[lang"}
	case pseudoReadOnly, pseudoReadWrite:
		return []string{"[readonly", "[disabled", "[type", "[contenteditable"}
	case pseudoIndeterminate:
		return []string{"[value"}
	case pseudoRequired, pseudoOptional:
		return []string{"[required", "[type"}
	case pseudoDir:
		return []string{"[dir"}
	case pseudoState, pseudoFirstChild, pseudoLastChild, pseudoRoot, pseudoNot, pseudoNthChild, pseudoLink,
		pseudoVisited, pseudoIs, pseudoWhere, pseudoEnabled, pseudoOnlyChild, pseudoNthLastChild, pseudoFirstOfType,
		pseudoLastOfType, pseudoOnlyOfType, pseudoNthOfType, pseudoNthLastOfType, pseudoHas, pseudoScope:
	}
	return nil
}

// walk calls fn for each compound of s and of the selectors its
// pseudo-classes hold, :has()'s included.
func (s Selector) walk(fn func(c *Compound)) {
	for i := range s.Compounds {
		c := &s.Compounds[i]
		fn(c)
		for _, p := range c.pseudos {
			for _, n := range p.sels {
				n.walk(fn)
			}
		}
	}
}

// awayReads calls key and state for what s, a selector a pseudo-class
// holds, reads on an element other than its subject (its compounds before
// the last, and what they hold), and for what the selectors its subject's
// pseudo-classes hold read so: in a :has(), those may be elements above
// or beside the one the :has() is on.
func (s Selector) awayReads(key func(string), state func(dom.State)) {
	last := len(s.Compounds) - 1
	for i := range s.Compounds {
		c := &s.Compounds[i]
		if i < last {
			Selector{Compounds: s.Compounds[i : i+1]}.walk(func(c *Compound) {
				c.keys(key)
				for j := range c.pseudos {
					state(c.pseudos[j].reads())
				}
			})
			continue
		}
		for j := range c.pseudos {
			for _, n := range c.pseudos[j].nested() {
				n.awayReads(key, state)
			}
		}
	}
}

// hasReads calls add for each :has() of s, and of the selectors its
// pseudo-classes hold, with the compounds the element it is tested on
// must name (as stateRead's on), and whether that element stands above
// s's subject (away) or before it (away and beside).
func (s Selector) hasReads(add func(on []*Compound, rels []Selector, away, beside bool)) {
	s.hasFrom(nil, false, false, add)
}

// hasFrom is hasReads for a selector the compounds outer hold, as
// readsFrom.
func (s Selector) hasFrom(outer []*Compound, apart, beside bool,
	add func(on []*Compound, rels []Selector, away, beside bool)) {
	for i := range s.Compounds {
		c := &s.Compounds[i]
		on, away, before := append(slices.Clip(outer), c), apart, beside
		if i < len(s.Combinators) {
			on, away, before = on[len(outer):], true, s.Combinators[i].sibling()
		}
		for j := range c.pseudos {
			p := &c.pseudos[j]
			if p.kind == pseudoHas {
				add(on, p.sels, away, before)
			}
			for _, n := range p.nested() {
				n.hasFrom(on, away, before, add)
			}
		}
	}
}
