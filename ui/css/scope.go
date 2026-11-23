package css

import (
	"strings"

	"github.com/lantervale/ember/ui/internal/dom"
)

// scopeRule is an @scope rule: its rules apply to the elements under a
// root its start matches, the root included, down to those its end
// matches, which it leaves out.
type scopeRule struct {
	// start is nil for an @scope of no start, rooted at the root element,
	// as a sheet of no element's is in Chrome, or at the roots of the
	// @scope rule it stands in, parent.
	start, end []Selector
	parent     *scopeRule
	// text is its prelude as written.
	text string
}

// scope reads an @scope rule: a start, an end after to, both in ( ), and
// a block whose rules apply in the scope. Its start is relative to the
// rule it stands in, and its end to its root, which its block's & is,
// weighing nothing.
func (p *parser) scope(t Token, in scope) {
	start := p.i - 1
	prelude, block := p.prelude()
	if !block {
		p.warnf(t.Pos, "@scope has no { block }")
		return
	}
	s := &scopeRule{parent: in.scoped, text: strings.TrimSpace(join(prelude))}
	cs := items(prelude)
	ok := true
	if len(cs) > 0 && cs[0].Kind == LParen {
		s.start, ok = p.scopeSelectors(cs[0], in.nest)
		cs = cs[1:]
	}
	if ok && len(cs) == 2 && isWord(cs[0], "to") && cs[1].Kind == LParen {
		s.end, ok = p.scopeSelectors(cs[1], inScope())
		cs = cs[2:]
	}
	switch {
	case !ok:
		p.skipBlock()
		return
	case len(cs) > 0:
		p.warnf(t.Pos, "@scope: %q is no scope; the rule is dropped", s.text)
		p.skipBlock()
		return
	}
	outer := in
	in.scoped, in.nest = s, inScope()
	if p.holding > 0 {
		p.block(in)
		return
	}
	from := len(p.sheet.Rules)
	p.holding++
	p.block(in)
	p.holding--
	p.written(start, from, outer)
}

// scopeSelectors reads the selector list of an @scope's start or end,
// relative to n; a browser's sheet drops a list naming only elements ui
// has none of without a word.
func (p *parser) scopeSelectors(c Component, n *nesting) ([]Selector, bool) {
	list := n.resolve(trim(c.Args))
	if p.browser {
		if list = matched(list); list == nil {
			return nil, false
		}
	}
	sels, err := parseSelectors(list)
	if err == nil {
		for _, s := range sels {
			if s.PseudoElement != "" {
				err = selErr(c, "::%s in @scope", s.PseudoElement)
			}
		}
	}
	if err != nil {
		p.warnf(err.Pos, "@scope: %s; the rule is dropped", err.Msg)
		return nil, false
	}
	return sels, true
}

// chain is s and the @scope rules it stands in, the outermost first.
func (s *scopeRule) chain() []*scopeRule {
	if s == nil {
		return nil
	}
	return append(s.parent.chain(), s)
}

// within calls fn with each root of s whose scope e is in, the nearest
// first, and how many generations e is under it, until fn returns true;
// it reports whether fn did.
func (s *scopeRule) within(e *dom.Node, fn func(root *dom.Node, near int) bool) bool {
	for _, r := range s.roots(e) {
		if !s.limited(e, r.node) && fn(r.node, r.near) {
			return true
		}
	}
	return false
}

// root is a root of an @scope rule, near generations above the element
// its roots were found from.
type root struct {
	node *dom.Node
	near int
}

// roots is the roots of s among e and its ancestors, the nearest first.
// Each rule it stands in finds its own roots once, so nested @scope
// rules cost the product of their depth and e's, never a power of it.
func (s *scopeRule) roots(e *dom.Node) []root {
	var outer []root
	if s.parent != nil {
		outer = s.parent.roots(e)
		if len(outer) == 0 {
			return nil
		}
	}
	var out []root
	near := 0
	for a := e; a != nil; a = a.Parent() {
		if s.parent == nil {
			if s.at(a, nil) {
				out = append(out, root{a, near})
			}
		} else {
			for _, r := range outer {
				if r.near >= near && !s.parent.limited(a, r.node) && s.at(a, r.node) {
					out = append(out, root{a, near})
					break
				}
			}
		}
		near++
	}
	return out
}

// at reports whether a is a root of s in the scope of root, its parent
// rule's root, nil for s outermost.
func (s *scopeRule) at(a, root *dom.Node) bool {
	if s.start == nil {
		return a == root || root == nil && a.Parent() == nil
	}
	for _, sel := range s.start {
		if sel.MatchesIn(a, root) {
			return true
		}
	}
	return false
}

// limited reports whether e, under root, is out of its scope: e or an
// element between them, root included, matches s's end.
func (s *scopeRule) limited(e, root *dom.Node) bool {
	if len(s.end) == 0 {
		return false
	}
	for a := e; a != nil; a = a.Parent() {
		for _, sel := range s.end {
			if sel.MatchesIn(a, root) {
				return true
			}
		}
		if a == root {
			return false
		}
	}
	return false
}

// matchesScoped reports whether e matches sel, a selector of a rule in
// s, and how near the root it matches through is: the nearest it can.
func (s *scopeRule) matchesScoped(sel Selector, e *dom.Node) (int, bool) {
	near := 0
	ok := s.within(e, func(root *dom.Node, n int) bool {
		near = n
		return sel.MatchesIn(e, root)
	})
	return near, ok
}
