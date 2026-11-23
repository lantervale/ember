package css

import (
	"strings"
)

// customFunction is an @function rule: its parameters, the type its
// result computes to (any value when universal), and its declarations,
// result's and its local custom properties', each under the @media
// rules it stands in.
type customFunction struct {
	name    string
	params  []funcParam
	returns syntax
	decls   []funcDecl
	written string
}

type funcParam struct {
	name   string
	syntax syntax
	def    []Component
	hasDef bool
}

type funcDecl struct {
	name  string
	value []Component
	media []*mediaList
}

// maxCallDepth bounds custom functions calling custom functions.
const maxCallDepth = 32

// functionRule reads an @function rule into the sheet; one whose
// prelude or a parameter's type doesn't read is dropped, as in Chrome.
func (p *parser) functionRule(t Token) {
	prelude, block := p.prelude()
	if !block {
		p.warnf(t.Pos, "@function has no { block }")
		return
	}
	var body []Component
	for k := p.peek().Kind; k != EOF && k != RBrace; k = p.peek().Kind {
		body = append(body, p.component())
	}
	p.next()
	f, ok := functionPrelude(prelude)
	if !ok {
		p.warnf(t.Pos, "@function: %q is no function prelude; the rule is dropped", strings.TrimSpace(join(prelude)))
		return
	}
	f.written = "@function " + strings.TrimSpace(join(prelude)) + " {" + join(body) + "}"
	p.functionBody(&f, body, nil)
	p.sheet.functions = append(p.sheet.functions, &f)
}

// functionPrelude reads --name(parameters) returns <type>.
func functionPrelude(prelude []Component) (customFunction, bool) {
	cs := trim(prelude)
	if len(cs) == 0 || cs[0].Kind != Function || !strings.HasPrefix(cs[0].Text, "--") {
		return customFunction{}, false
	}
	f := customFunction{name: cs[0].Text, returns: syntax{universal: true}}
	if args := trim(cs[0].Args); len(args) > 0 {
		for _, part := range splitOn(args, func(c Component) bool { return c.Kind == Comma }) {
			prm, ok := functionParam(trim(part))
			if !ok {
				return customFunction{}, false
			}
			f.params = append(f.params, prm)
		}
	}
	rest := trim(cs[1:])
	if len(rest) > 0 {
		if !isWord(rest[0], "returns") {
			return customFunction{}, false
		}
		syn, ok := typeOf(trim(rest[1:]))
		if !ok {
			return customFunction{}, false
		}
		f.returns = syn
	}
	return f, true
}

// functionParam reads --name <type>? (: default)?.
func functionParam(cs []Component) (funcParam, bool) {
	if len(cs) == 0 || cs[0].Kind != Ident || !strings.HasPrefix(cs[0].Text, "--") {
		return funcParam{}, false
	}
	prm := funcParam{name: cs[0].Text, syntax: syntax{universal: true}}
	rest := trim(cs[1:])
	colon := len(rest)
	for i, c := range rest {
		if c.Kind == Colon {
			colon = i
			break
		}
	}
	if typ := trim(rest[:colon]); len(typ) > 0 {
		syn, ok := typeOf(typ)
		if !ok {
			return funcParam{}, false
		}
		prm.syntax = syn
	}
	if colon < len(rest) {
		prm.def, prm.hasDef = trim(rest[colon+1:]), true
	}
	return prm, true
}

// typeOf reads a function's type: <type>, *, or type(<syntax>).
func typeOf(cs []Component) (syntax, bool) {
	text := join(cs)
	if len(cs) == 1 && cs[0].Kind == Function && strings.EqualFold(cs[0].Text, "type") {
		text = join(cs[0].Args)
	}
	syn, err := parseSyntax(text)
	return syn, err == nil
}

// functionBody reads the declarations of a function's body, those of
// its @media rules under them, and of its @supports rules that hold.
func (p *parser) functionBody(f *customFunction, body []Component, media []*mediaList) {
	for i := 0; i < len(body); {
		c := body[i]
		if c.Kind == Whitespace || c.Kind == Semicolon {
			i++
			continue
		}
		if c.Kind == AtKeyword {
			end := i + 1
			for end < len(body) && body[end].Kind != LBrace && body[end].Kind != Semicolon {
				end++
			}
			prelude := body[i+1 : min(end, len(body))]
			if end < len(body) && body[end].Kind == LBrace {
				switch strings.ToLower(c.Text) {
				case atMedia:
					q := mediaParser{warn: p.warnf}
					p.functionBody(f, body[end].Args, append(media[:len(media):len(media)], q.list(prelude)))
				case atSupports:
					if supported(prelude) {
						p.functionBody(f, body[end].Args, media)
					}
				default:
					p.warnf(c.Pos, "@%s is not supported in @function", c.Text)
				}
			}
			i = end + 1
			continue
		}
		end := i
		for end < len(body) && body[end].Kind != Semicolon {
			end++
		}
		part := trim(body[i:end])
		i = end
		rest := trim(part[1:])
		name := part[0].Text
		if part[0].Kind != Ident || len(rest) == 0 || rest[0].Kind != Colon ||
			name != "result" && !strings.HasPrefix(name, "--") {
			p.warnf(part[0].Pos, "@function %s takes result and custom properties, found %q", f.name, join(part))
			continue
		}
		f.decls = append(f.decls, funcDecl{name: name, value: trim(rest[1:]), media: media})
	}
}

// frame is a custom function's call: its parameters' values, nil for
// the guaranteed-invalid value, and its local custom properties.
type frame struct {
	fn        *customFunction
	vals      map[string][]Component
	locals    map[string][]Component
	resolving map[string]bool
}

// call is what the custom function call c stands for: its result for
// the arguments given, in the element's context; false when no function
// of its name is defined, it calls itself, or its result is invalid.
func (x *ctx) call(c Component) ([]Component, bool) {
	f := x.env.props.function(c.Text)
	if f == nil || len(x.frames) >= maxCallDepth {
		return nil, false
	}
	for _, fr := range x.frames {
		if fr.fn == f {
			return nil, false
		}
	}
	var args [][]Component
	if a := trim(c.Args); len(a) > 0 {
		args = splitOn(a, func(c Component) bool { return c.Kind == Comma })
	}
	if len(args) > len(f.params) {
		return nil, false
	}
	fr := &frame{fn: f, vals: map[string][]Component{}, locals: map[string][]Component{}, resolving: map[string]bool{}}
	for i, prm := range f.params {
		var arg []Component
		if i < len(args) {
			arg = trim(args[i])
		}
		fr.vals[prm.name] = x.argument(prm, arg, i < len(args))
	}
	var result []Component
	hasResult := false
	for i := range f.decls {
		d := &f.decls[i]
		if !x.holds(d.media) {
			continue
		}
		if d.name == "result" {
			result, hasResult = d.value, true
		} else {
			fr.locals[d.name] = d.value
		}
	}
	if !hasResult {
		return nil, false
	}
	x.frames = append(x.frames, fr)
	v, ok := x.substitute(result)
	x.frames = x.frames[:len(x.frames)-1]
	if !ok {
		return nil, false
	}
	if !f.returns.universal {
		x.readFont(v)
		text, typed := x.typed(f.returns, v)
		if !typed {
			return nil, false
		}
		return parsedVar(text), true
	}
	return v, true
}

// holds reports whether every query list of media holds.
func (x *ctx) holds(media []*mediaList) bool {
	for _, m := range media {
		x.env.mediaRead = true
		if !m.matches(x.env.media) {
			return false
		}
	}
	return true
}

// argument is the value of the parameter prm for arg, given or not: its
// default where not, computed as its type says, nil for the
// guaranteed-invalid value.
func (x *ctx) argument(prm funcParam, arg []Component, given bool) []Component {
	if len(arg) == 1 && arg[0].Kind == LBrace {
		arg = trim(arg[0].Args)
	}
	if !given {
		if !prm.hasDef {
			return nil
		}
		arg = prm.def
	}
	v, ok := x.substitute(arg)
	if !ok {
		return nil
	}
	if prm.syntax.universal {
		return v
	}
	x.readFont(v)
	text, typed := x.typed(prm.syntax, v)
	if !typed {
		return nil
	}
	return parsedVar(text)
}

// frameVar is the value of the custom property name in the custom
// functions being called, the innermost first: a local, else a
// parameter. found is false when none of them has it.
func (x *ctx) frameVar(name string) ([]Component, bool, bool) {
	for i := len(x.frames) - 1; i >= 0; i-- {
		fr := x.frames[i]
		if raw, local := fr.locals[name]; local {
			if fr.resolving[name] {
				return nil, false, true
			}
			fr.resolving[name] = true
			frames := x.frames
			x.frames = x.frames[:i+1]
			v, ok := x.substitute(raw)
			x.frames = frames
			fr.resolving[name] = false
			return v, ok, true
		}
		if v, param := fr.vals[name]; param {
			return v, v != nil, true
		}
	}
	return nil, false, false
}

// function is the custom function name, nil when none is defined.
func (g *registry) function(name string) *customFunction {
	if g == nil {
		return nil
	}
	return g.functions[name]
}
