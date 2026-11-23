package css

import "strings"

// nesting is the style rule, or the @scope rule, that the rules being
// read stand in.
type nesting struct {
	// sels are the rule's selectors, which its declarations after a
	// nested rule take, and written the list as read, which & stands for.
	sels    []Selector
	written []Component
	// scoped is set in an @scope rule's block: & is :where(:scope) there,
	// and a selector naming :scope is not relative.
	scoped bool
}

// inScope is the nesting of an @scope rule's block: its declarations
// apply to the scope's root, weighing nothing.
func inScope() *nesting {
	written := []Component{{Token: Token{Kind: Colon, Text: ":"}},
		{Token: Token{Kind: Function, Text: "where"}, Args: scopeClass(Pos{})}}
	sels, _ := parseSelectors(written)
	return &nesting{sels: sels, written: written, scoped: true}
}

// resolve is the selector list of a rule nested in n as CSS Nesting
// reads it: a selector without & is relative, after & and a descendant
// combinator or the combinator it begins with, and & stands for the
// rule it is nested in, as :is() would. Outside any rule (n nil), & is
// :scope.
func (n *nesting) resolve(prelude []Component) []Component {
	if n == nil && !holdsAmpersand(prelude) {
		return prelude
	}
	var out []Component
	for i, part := range splitOn(prelude, func(c Component) bool { return c.Kind == Comma }) {
		if i > 0 {
			out = append(out, Component{Token: Token{Kind: Comma, Text: ","}})
		}
		part = trim(part)
		if n != nil && len(part) > 0 && !holdsAmpersand(part) && (!n.scoped || !holdsScope(part)) {
			lead := Component{Token: Token{Kind: Delim, Text: "&", Pos: part[0].Pos}}
			part = append([]Component{lead, {Token: Token{Kind: Whitespace, Text: " "}}}, part...)
		}
		out = append(out, n.substitute(part)...)
	}
	return out
}

// substitute puts what & stands for in its place in cs, in functions'
// arguments too.
func (n *nesting) substitute(cs []Component) []Component {
	out := make([]Component, 0, len(cs))
	for i, c := range cs {
		switch {
		case c.Kind == Delim && c.Text == "&":
			out = append(out, n.ampersand(c.Pos, i == 0 || compoundStarts(cs[i-1]))...)
		case c.Kind == Function:
			c.Args = n.substitute(c.Args)
			out = append(out, c)
		default:
			out = append(out, c)
		}
	}
	return out
}

// ampersand is what & stands for at pos, first in its compound when
// first: the rule's own compound where :is() of it matches the same
// with the same weight, :is() of its list otherwise.
func (n *nesting) ampersand(pos Pos, first bool) []Component {
	switch {
	case n == nil:
		return scopeClass(pos)
	case n.scoped:
		return n.written
	case n.inline(first):
		return n.written
	}
	return []Component{{Token: Token{Kind: Colon, Text: ":", Pos: pos}},
		{Token: Token{Kind: Function, Text: "is", Pos: pos}, Args: n.written}}
}

// inline reports whether the rule's selector can stand for & as
// written: one compound, of no pseudo-element, and of no type unless &
// begins its compound.
func (n *nesting) inline(first bool) bool {
	if len(n.sels) != 1 {
		return false
	}
	s := n.sels[0]
	if len(s.Compounds) != 1 || s.PseudoElement != "" || s.inactive || s.part != 0 {
		return false
	}
	typed := len(n.written) > 0 && (n.written[0].Kind == Ident || n.written[0].Kind == Delim && n.written[0].Text == "*")
	return first || !typed
}

// scopeClass is :scope, written at pos.
func scopeClass(pos Pos) []Component {
	return []Component{{Token: Token{Kind: Colon, Text: ":", Pos: pos}},
		{Token: Token{Kind: Ident, Text: kwScope, Pos: pos}}}
}

// compoundStarts reports whether a compound begins after c: c is white
// space or a combinator.
func compoundStarts(c Component) bool {
	return c.Kind == Whitespace || c.Kind == Delim && (c.Text == ">" || c.Text == "+" || c.Text == "~")
}

// holdsAmpersand reports whether cs has an &, in a function's arguments
// too.
func holdsAmpersand(cs []Component) bool {
	for _, c := range cs {
		if c.Kind == Delim && c.Text == "&" || c.Kind == Function && holdsAmpersand(c.Args) {
			return true
		}
	}
	return false
}

// holdsScope reports whether cs names :scope, in a function's arguments
// too.
func holdsScope(cs []Component) bool {
	for i, c := range cs {
		if c.Kind == Colon && i+1 < len(cs) && cs[i+1].Kind == Ident && strings.EqualFold(cs[i+1].Text, kwScope) ||
			c.Kind == Function && holdsScope(c.Args) {
			return true
		}
	}
	return false
}
