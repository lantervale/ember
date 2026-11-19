package css

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/lantervale/ember/ui/internal/dom"
)

//nolint:gochecknoglobals,goconst // lookup table
var stateNames = map[string]dom.State{
	"hover": dom.Hover, "focus": dom.Focus, "focus-visible": dom.FocusVisible, "active": dom.Active,
	"disabled": dom.Disabled, "checked": dom.Checked, "open": dom.Open, "modal": dom.Modal,
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
	// pseudoVisited :visited, which nothing is: ui keeps no history.
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
)

type pseudo struct {
	kind  pseudoKind
	state dom.State
	// sels are what :not(), :is() and :where() were given.
	sels []Selector
	// a and b are :nth-child(An+B)'s, and the other :nth-'s.
	a, b int
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
}

func (a Attr) matches(v string) bool {
	if !a.HasValue {
		return true
	}
	want := a.Value
	if a.Fold {
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
	return b.String()
}

// Matches reports whether the element e matches s.
func (s Selector) Matches(e *dom.Node) bool {
	return s.matchFrom(len(s.Compounds)-1, e)
}

func (s Selector) matchFrom(i int, e *dom.Node) bool {
	if !s.Compounds[i].matches(e) {
		return false
	}
	if i == 0 {
		return true
	}
	switch s.Combinators[i-1] {
	case Child:
		p := e.Parent()
		return p != nil && s.matchFrom(i-1, p)
	case Next, Later:
		before := elementsBefore(e)
		for j := len(before) - 1; j >= 0; j-- {
			if before[j].IsText() {
				continue
			}
			if s.matchFrom(i-1, before[j]) {
				return true
			}
			if s.Combinators[i-1] == Next {
				return false
			}
		}
		return false
	case Descendant:
	}
	for p := e.Parent(); p != nil; p = p.Parent() {
		if s.matchFrom(i-1, p) {
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

func (c *Compound) matches(e *dom.Node) bool {
	if !c.names(e) {
		return false
	}
	for i := range c.pseudos {
		if !c.pseudos[i].matches(e) {
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
		if !ok || !a.matches(v) {
			return false
		}
	}
	return true
}

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
	}
	return true
}

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
	for _, c := range s.Compounds {
		s.Specificity = s.Specificity.add(c.specificity())
	}
	if s.PseudoElement != "" {
		s.Specificity[2]++
	}
	return s, nil
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
		case pseudoWhere:
		case pseudoNot, pseudoIs:
			var most Specificity
			for _, s := range p.sels {
				if most.Less(s.Specificity) {
					most = s.Specificity
				}
			}
			sp = sp.add(most)
		case pseudoState, pseudoFirstChild, pseudoLastChild, pseudoRoot, pseudoNthChild, pseudoLink, pseudoVisited,
			pseudoEnabled, pseudoOnlyChild, pseudoNthLastChild, pseudoFirstOfType, pseudoLastOfType, pseudoOnlyOfType,
			pseudoNthOfType, pseudoNthLastOfType:
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
			for _, n := range p.sels {
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
	case pseudoEnabled:
		return dom.Disabled
	case pseudoFirstChild, pseudoLastChild, pseudoRoot, pseudoNot, pseudoNthChild, pseudoLink, pseudoVisited, pseudoIs,
		pseudoWhere, pseudoOnlyChild, pseudoNthLastChild, pseudoFirstOfType, pseudoLastOfType, pseudoOnlyOfType,
		pseudoNthOfType, pseudoNthLastOfType:
	}
	return 0
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
			for _, n := range p.sels {
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
			if slices.ContainsFunc(p.sels, Selector.siblings) {
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
			for _, n := range p.sels {
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
}
