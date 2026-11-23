package css

import (
	"hash/fnv"
	"strconv"
	"strings"

	"github.com/lantervale/ember/ui/internal/dom"
)

// substitutes reports whether c is a function the cascade replaces on
// each element: var(), env(), attr(), if() and a custom function; or one
// of the math functions that read the element, sibling-index(),
// sibling-count() and random().
func substitutes(c Component) bool {
	if c.Kind != Function {
		return false
	}
	if strings.HasPrefix(c.Text, "--") {
		return true
	}
	switch strings.ToLower(c.Text) {
	case "var", "env", "attr", "if", fnSiblingIndex, fnSiblingCount, fnRandom:
		return true
	}
	return false
}

const (
	fnSiblingIndex = "sibling-index"
	fnSiblingCount = "sibling-count"
	fnRandom       = "random"
)

// ifValue is what if() c stands for: the value of its first branch
// whose condition holds, nothing when none does.
func (x *ctx) ifValue(c Component) ([]Component, bool) {
	for _, branch := range splitOn(c.Args, func(c Component) bool { return c.Kind == Semicolon }) {
		branch = trim(branch)
		if len(branch) == 0 {
			continue
		}
		colon := -1
		for i, b := range branch {
			if b.Kind == Colon {
				colon = i
				break
			}
		}
		if colon < 0 {
			return nil, false
		}
		holds, ok := x.condition(items(branch[:colon]))
		if !ok {
			return nil, false
		}
		if holds {
			return x.substitute(trim(branch[colon+1:]))
		}
	}
	return nil, true
}

// condition answers an if() condition: else, or style(), media() and
// supports() queries joined by and, or or not, false for one it can't
// read.
func (x *ctx) condition(cs []Component) (bool, bool) {
	if len(cs) == 1 && isWord(cs[0], "else") {
		return true, true
	}
	if len(cs) == 0 {
		return false, false
	}
	if isWord(cs[0], "not") {
		if len(cs) != 2 {
			return false, false
		}
		v, ok := x.test(cs[1])
		return !v, ok
	}
	result, ok := x.test(cs[0])
	joined := ""
	for i := 1; ok && i < len(cs); i += 2 {
		if i+1 >= len(cs) || !isWord(cs[i], "and", "or") || joined != "" && !isWord(cs[i], joined) {
			return false, false
		}
		joined = strings.ToLower(cs[i].Text)
		var next bool
		next, ok = x.test(cs[i+1])
		if joined == "and" {
			result = result && next
		} else {
			result = result || next
		}
	}
	return result, ok
}

// test answers one query of an if() condition.
func (x *ctx) test(c Component) (bool, bool) {
	switch {
	case c.Kind == LParen:
		return x.condition(items(c.Args))
	case c.Kind != Function:
		return false, false
	}
	switch strings.ToLower(c.Text) {
	case fnStyle:
		return x.styleQuery(c.Args)
	case "media":
		p := mediaParser{warn: func(Pos, string, ...any) {}}
		x.env.mediaRead = true
		q := c.Args
		if feature(items(q)) {
			q = []Component{{Token: Token{Kind: LParen}, Args: q}}
		}
		return p.list(q).matches(x.env.media), true
	case atSupports:
		args := trim(c.Args)
		if len(args) == 1 && args[0].Kind == Function {
			return supportedIn(args[0]), true
		}
		return supportedIn(Component{Token: Token{Kind: LParen}, Args: c.Args}), true
	}
	return false, false
}

// feature reports whether cs is a media feature without its
// parentheses, as media() takes one: a name and a colon or a comparison.
func feature(cs []Component) bool {
	return len(cs) > 1 && (cs[1].Kind == Colon || cs[1].Kind == Delim && strings.ContainsAny(cs[1].Text, "<>=")) ||
		len(cs) > 2 && cs[0].Kind != Ident
}

// styleQuery answers style(--name: value) on the element: whether its
// custom property computes to what value does; style(--name) whether it
// has a value other than its initial one.
func (x *ctx) styleQuery(args []Component) (bool, bool) {
	args = trim(args)
	if len(args) == 0 || args[0].Kind != Ident || !strings.HasPrefix(args[0].Text, "--") {
		return false, false
	}
	name := args[0].Text
	own, has := x.lookup(name)
	r := x.env.props.get(name)
	rest := trim(args[1:])
	if len(rest) == 0 {
		if r != nil {
			return has && join(own) != r.initial, true
		}
		return has, true
	}
	if rest[0].Kind != Colon {
		return false, false
	}
	want, ok := x.substitute(trim(rest[1:]))
	if !ok {
		return false, true
	}
	if r != nil {
		w, typed := x.typed(r.syntax, want)
		return typed && has && w == join(own), true
	}
	return has && normalTokens(own) == normalTokens(want), true
}

// normalTokens writes v with its whitespace runs as one space.
func normalTokens(v []Component) string {
	return strings.Join(strings.Fields(join(trim(v))), " ")
}

// attrValue is what attr() c stands for on the element: its attribute
// as a string, or parsed as the type or unit it names, else its
// fallback; false when it has neither.
func (x *ctx) attrValue(c Component) ([]Component, bool) {
	args := trim(c.Args)
	if len(args) == 0 || args[0].Kind != Ident {
		return nil, false
	}
	name := args[0].Text
	rest := trim(args[1:])
	var kind Component
	if len(rest) > 0 && rest[0].Kind != Comma {
		kind, rest = rest[0], trim(rest[1:])
	}
	var fallback []Component
	hasFallback := false
	if len(rest) > 0 {
		if rest[0].Kind != Comma {
			return nil, false
		}
		fallback, hasFallback = trim(rest[1:]), true
	}
	if v, ok := x.attrTyped(name, kind); ok {
		return v, true
	}
	if hasFallback {
		return x.substitute(fallback)
	}
	if kind.Kind == EOF || isWord(kind, "raw-string") {
		return []Component{{Token: Token{Kind: String}}}, true
	}
	return nil, false
}

// attrTyped is the element's attribute name as kind reads it: a string
// without one, else parsed as type(<syntax>) or a number in a unit.
func (x *ctx) attrTyped(name string, kind Component) ([]Component, bool) {
	if x.node == nil {
		return nil, false
	}
	text, ok := x.node.Attr(name)
	if !ok {
		return nil, false
	}
	if kind.Kind == EOF || isWord(kind, "raw-string") {
		return []Component{{Token: Token{Kind: String, Text: text}}}, true
	}
	v := trim(componentsOf(text))
	if kind.Kind == Function && strings.EqualFold(kind.Text, "type") {
		syn, err := parseSyntax(join(kind.Args))
		if err != nil || len(v) == 0 || hasVar(v) {
			return nil, false
		}
		if _, typed := x.typed(syn, v); !typed {
			return nil, false
		}
		return v, true
	}
	var unit string
	switch {
	case kind.Kind == Ident:
		unit = kind.Text
	case kind.Kind == Delim && kind.Text == "%":
		unit = "%"
	default:
		return nil, false
	}
	if len(v) != 1 || v[0].Kind != Number {
		return nil, false
	}
	n := v[0]
	if unit == "%" {
		n.Kind = Percentage
		return []Component{n}, true
	}
	if _, err := dimOf(Component{Token: Token{Kind: Dimension, Num: 1, Unit: unit}}); err != nil {
		return nil, false
	}
	n.Kind, n.Unit = Dimension, unit
	return []Component{n}, true
}

// elementMath replaces in v the math functions that read the element:
// sibling-index() and sibling-count() by numbers, random() by one keyed
// to the element, the property and its place in the value.
func (x *ctx) elementMath(v []Component, prop string) []Component {
	if !readsElement(v) {
		return v
	}
	out := make([]Component, len(v))
	copy(out, v)
	at := 0
	x.rewriteMath(out, prop, &at)
	return out
}

func readsElement(v []Component) bool {
	for _, c := range v {
		if c.Kind == Function && (isFn(c, fnSiblingIndex) || isFn(c, fnSiblingCount) || isFn(c, fnRandom)) {
			return true
		}
		if readsElement(c.Args) {
			return true
		}
	}
	return false
}

func isFn(c Component, name string) bool { return strings.EqualFold(c.Text, name) }

func (x *ctx) rewriteMath(v []Component, prop string, at *int) {
	for i := range v {
		c := &v[i]
		if c.Kind != Function {
			continue
		}
		switch {
		case isFn(*c, fnSiblingIndex) && len(trim(c.Args)) == 0:
			at, _ := siblings(x.node)
			*c = Component{Token: Token{Kind: Number, Num: float64(at), Int: true}}
			continue
		case isFn(*c, fnSiblingCount) && len(trim(c.Args)) == 0:
			_, n := siblings(x.node)
			*c = Component{Token: Token{Kind: Number, Num: float64(n), Int: true}}
			continue
		case isFn(*c, fnRandom):
			*at++
			c.Args = x.randomKey(c.Args, prop, *at)
		}
		if len(c.Args) > 0 {
			args := make([]Component, len(c.Args))
			copy(args, c.Args)
			c.Args = args
			x.rewriteMath(c.Args, prop, at)
		}
	}
}

// siblings are the element's place among its parent's elements, from 1,
// and how many they are; a pseudo-element's are its element's.
func siblings(n *dom.Node) (int, int) {
	for n != nil && n.Pseudo() != dom.NoPseudo {
		n = n.Parent()
	}
	if n == nil || n.Parent() == nil {
		return 1, 1
	}
	index, count := 0, 0
	for _, k := range n.Parent().Children() {
		if k.IsText() || k.Pseudo() != dom.NoPseudo {
			continue
		}
		count++
		if k == n {
			index = count
		}
	}
	return index, count
}

// randomKey gives random()'s arguments a fixed number in place of its
// caching key: from its <dashed-ident> or the element-shared keyword,
// else from the element, the property and the function's place in the
// value, so the one random() keeps its value on the element.
func (x *ctx) randomKey(args []Component, prop string, at int) []Component {
	parts := splitOn(args, func(c Component) bool { return c.Kind == Comma })
	if len(parts) == 0 {
		return args
	}
	head := items(parts[0])
	if len(head) > 0 && isWord(head[0], "fixed") {
		return args
	}
	key, shared, keyed := "", false, true
	for _, c := range head {
		switch {
		case c.Kind == Ident && strings.HasPrefix(c.Text, "--"):
			key = c.Text
		case isWord(c, "element-shared"):
			shared = true
		case isWord(c, kwAuto):
		default:
			keyed = false
		}
	}
	if keyed {
		parts = parts[1:]
	} else {
		key, shared = "", false
	}
	if key == "" {
		key = prop + "#" + strconv.Itoa(at)
	}
	if !shared {
		key += "@" + nodeKey(x.node)
	}
	h := fnv.New64a()
	_, _ = h.Write([]byte(key))
	f := float64(h.Sum64()>>11) / (1 << 53)
	out := []Component{word("fixed"), {Token: Token{Kind: Whitespace, Text: " "}},
		{Token: Token{Kind: Number, Num: f}}}
	for _, p := range parts {
		out = append(out, Component{Token: Token{Kind: Comma, Text: ","}})
		out = append(out, p...)
	}
	return out
}

// nodeKey names n for random()'s keys: its path from the root.
func nodeKey(n *dom.Node) string {
	var b strings.Builder
	for ; n != nil; n = n.Parent() {
		i, _ := siblings(n)
		b.WriteString(strconv.Itoa(i))
		b.WriteByte('.')
	}
	return b.String()
}
