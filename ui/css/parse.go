package css

import (
	"fmt"
	"io/fs"
	"net/url"
	"path"
	"slices"
	"strings"
	"sync"
)

// Component is a component value: a token, or a function or a block with
// what it holds. A block has the Kind of its opening token (LParen,
// LBracket or LBrace).
type Component struct {
	Token
	Args []Component
}

func (c Component) String() string {
	switch c.Kind {
	case Function:
		return c.Text + "(" + join(c.Args) + ")"
	case LParen:
		return "(" + join(c.Args) + ")"
	case LBracket:
		return "[" + join(c.Args) + "]"
	case LBrace:
		return "{" + join(c.Args) + "}"
	case EOF, Ident, AtKeyword, Hash, String, BadString, URL, BadURL, Delim, Number, Percentage, Dimension, Whitespace,
		CDO, CDC, Colon, Semicolon, Comma, RBracket, RParen, RBrace:
		// nothing special for these tokens
	}
	return c.Token.String()
}

func join(cs []Component) string {
	var b strings.Builder
	for _, c := range cs {
		b.WriteString(c.String())
	}
	return b.String()
}

// Declaration is one property and its value. Name is lowercase, except a
// custom property's, which keeps its case. Value has no surrounding
// whitespace and no !important.
type Declaration struct {
	Name      string
	Value     []Component
	Important bool
	Pos       Pos
	// longhands is a shorthand's expansion, and vals a longhand's value
	// without whitespace, when it has no var().
	longhands map[string][]Component
	vals      []Component
	// base is where the sheet it came from is, for relative urls; nil
	// for a style attribute's.
	base *origin
}

// origin is where a sheet is: its directory in its file system, or the
// address it was fetched from (From).
type origin struct {
	dir  string
	from *url.URL
}

// resolve is the url src of a sheet at o as the Document's images are
// named: as written when it has a scheme (ember:, data:, https:), from
// the sheet's address when it has one, in its directory otherwise.
func (o *origin) resolve(src string) string {
	switch {
	case src == "" || o == nil || strings.Contains(src, ":"):
		return src
	case o.from != nil:
		if at, err := o.from.Parse(src); err == nil {
			return at.String()
		}
		return src
	case o.dir == "":
		return src
	}
	return path.Join(o.dir, src)
}

func (d Declaration) String() string {
	s := d.Name + ": " + join(d.Value)
	if d.Important {
		s += " !important"
	}
	return s
}

// Rule is a style rule: its selectors and its declarations.
type Rule struct {
	Selectors []Selector
	Decls     []Declaration
	Pos       Pos
	// media are the @media rules it stands in, the outermost first, as
	// places in its sheet's list: it applies while they all hold.
	media []int
	// layer is the @layer it stands in, as a place from 1 in its sheet's
	// list; 0 in none.
	layer int
}

// Sheet is a parsed stylesheet. Name is the file it came from.
type Sheet struct {
	Name  string
	Rules []Rule
	faces []FontFace
	once  sync.Once
	index *index
	// media are its @media rules' queries, in the order they open.
	media []*mediaList
	// layers are the full names of its cascade layers (theme.dark), in
	// the order they are first named; one without a name begins with
	// anonymous.
	layers []string
	// browser is whether it was read as a browser reads it (Browser), src
	// its text then, and from the address it came from (From).
	browser bool
	src     string
	from    string
	origin  *origin
}

// anonymous begins the name a layer without one goes by.
const anonymous = "\x00"

// Browser reports whether the sheet was read as a browser reads it.
func (s *Sheet) Browser() bool { return s.browser }

// From is the address the sheet came from, "" when it was given none.
func (s *Sheet) From() string { return s.from }

// URL is the picture a url(ref) of the sheet names, as the Document's
// images are named: ref itself when it has a scheme, from the sheet's
// address when it has one (From), beside the sheet's file otherwise.
func (s *Sheet) URL(ref string) string { return s.origin.resolve(ref) }

// String writes the sheet's rules back as CSS, each in the @media rules
// it stands in, without what was dropped or its @font-face rules. A
// sheet read as a browser's is its own text, as it was written: what ui
// dropped of it is a browser's to read.
func (s *Sheet) String() string {
	if s.browser {
		return s.src
	}
	var b strings.Builder
	for _, r := range s.Rules {
		for _, m := range r.media {
			b.WriteString("@media " + s.media[m].text + " { ")
		}
		for i, sel := range r.Selectors {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(sel.String())
		}
		b.WriteString(" {")
		for _, d := range r.Decls {
			b.WriteString(" ")
			b.WriteString(d.String())
			b.WriteString(";")
		}
		b.WriteString(" }" + strings.Repeat(" }", len(r.media)) + "\n")
	}
	return b.String()
}

// Warning is something a sheet says that ui drops: a syntax error, an
// unknown property, a value or selector outside the subset.
type Warning struct { //nolint:errname // a dropped declaration, reported, never returned as a failure
	File string
	Pos  Pos
	Msg  string
}

func (w Warning) Error() string {
	file := w.File
	if file == "" {
		file = "style"
	}
	return fmt.Sprintf("%s:%d:%d: %s", file, w.Pos.Line, w.Pos.Col, w.Msg)
}

// ParseOption says how a sheet or a style attribute is read.
type ParseOption func(*parser)

// Browser reads a sheet, or a style attribute, written for browsers: its
// @supports rules apply where ui supports what they ask, its @layer
// rules are cascade layers, and each vendor's prefixed names are read as
// that vendor's table has them (vendors.go): a plain name where Chrome
// reads one, nothing where Chrome drops them. Without it each of these
// is a warning.
func Browser() ParseOption { return func(p *parser) { p.browser = true } }

// From gives a sheet the address it was fetched from: a url() in it is
// then resolved from that address, as a browser does, where a sheet
// without one names files beside its own.
func From(address string) ParseOption { return func(p *parser) { p.from = address } }

// Load reads and parses the stylesheet name in fsys. The error is only for
// reading; what the sheet gets wrong comes back as warnings.
func Load(fsys fs.FS, name string, opts ...ParseOption) (*Sheet, []Warning, error) {
	data, err := fs.ReadFile(fsys, name)
	if err != nil {
		return nil, nil, fmt.Errorf("css: %w", err)
	}
	s, warns := Parse(name, string(data), opts...)
	return s, warns, nil
}

// Parse reads a stylesheet. name only labels warnings. What it can't use
// is dropped with a warning, as browsers drop it, and the rest is kept.
func Parse(name, src string, opts ...ParseOption) (*Sheet, []Warning) {
	p := parser{file: name, toks: Tokenize(src)}
	for _, o := range opts {
		o(&p)
	}
	p.sheet = &Sheet{Name: name, browser: p.browser}
	if p.browser {
		p.sheet.src = src
	}
	if name != "" {
		p.origin = &origin{dir: path.Dir(name)}
	}
	if p.from != "" {
		if at, err := url.Parse(p.from); err == nil {
			p.origin, p.sheet.from = &origin{from: at}, p.from
		} else {
			p.warnf(Pos{1, 1}, "the sheet's address %q is no address: %v", p.from, err)
		}
	}
	p.sheet.origin = p.origin
	p.rules(scope{})
	return p.sheet, p.warns
}

// ParseInline reads a style attribute's declarations.
func ParseInline(src string, opts ...ParseOption) ([]Declaration, []Warning) {
	p := parser{toks: Tokenize(src)}
	for _, o := range opts {
		o(&p)
	}
	return p.inline()
}

func (p *parser) inline() ([]Declaration, []Warning) {
	var block []Component
	for p.peek().Kind != EOF {
		block = append(block, p.component())
	}
	return p.declarations(block), p.warns
}

type parser struct {
	file  string
	toks  []Token
	i     int
	warns []Warning
	// browser reads what is written for browsers (Browser), and from is
	// the sheet's address (From).
	browser bool
	from    string
	sheet   *Sheet
	origin  *origin
	// depth counts the @media, @supports and @layer blocks being read.
	depth int
}

// scope is the @media rules and the @layer a rule stands in.
type scope struct {
	media []int
	layer int
}

// rules reads rules up to the end of the sheet, or of the block they
// stand in.
func (p *parser) rules(in scope) {
	for {
		t := p.peek()
		switch t.Kind {
		case EOF:
			return
		case Whitespace, CDO, CDC:
			p.next()
		case AtKeyword:
			p.next()
			p.atRule(t, in)
		case RBrace:
			if p.depth > 0 {
				p.next()
				return
			}
			p.styleRule(in)
		case Ident, Function, Hash, String, BadString, URL, BadURL, Delim, Number, Percentage, Dimension, Colon, Semicolon,
			Comma, LBracket, RBracket, LParen, RParen, LBrace:
			p.styleRule(in)
		}
	}
}

func (p *parser) styleRule(in scope) {
	if r, ok := p.rule(); ok {
		r.media, r.layer = in.media, in.layer
		p.sheet.Rules = append(p.sheet.Rules, r)
	}
}

func (p *parser) atRule(t Token, in scope) {
	name := strings.ToLower(t.Text)
	if v, rest := vendorOf(name); v != nil && p.browser {
		as, other := v.atRule(rest)
		if as == "" && other == dropped {
			p.skipAtRule()
			return
		}
		if as != "" {
			name = as
		}
	}
	switch {
	case name == "font-face":
		if f, ok := p.fontFace(t.Pos); ok {
			p.sheet.faces = append(p.sheet.faces, f)
		}
	case name == "media":
		p.media(t, in)
	case name == "supports" && p.browser:
		p.supports(t, in)
	case name == "layer" && p.browser:
		p.layer(t, in)
	default:
		p.warnf(t.Pos, "@%s is not supported", t.Text)
		p.skipAtRule()
	}
}

// prelude reads an at-rule up to its block, which it enters, or to its
// end without one.
func (p *parser) prelude() ([]Component, bool) {
	var cs []Component
	for {
		switch k := p.peek().Kind; {
		case k == EOF || k == RBrace && p.depth > 0:
			return cs, false
		case k == Semicolon:
			p.next()
			return cs, false
		case k == LBrace:
			p.next()
			return cs, true
		}
		cs = append(cs, p.component())
	}
}

// block reads the rules of the block just entered.
func (p *parser) block(in scope) {
	p.depth++
	p.rules(in)
	p.depth--
}

// skipBlock leaves the block just entered, unread.
func (p *parser) skipBlock() {
	for k := p.peek().Kind; k != EOF; k = p.peek().Kind {
		if k == RBrace {
			p.next()
			return
		}
		p.component()
	}
}

func (p *parser) media(t Token, in scope) {
	prelude, block := p.prelude()
	if !block {
		p.warnf(t.Pos, "@media has no { block }")
		return
	}
	queries := mediaParser{browser: p.browser, warn: p.warnf}
	p.sheet.media = append(p.sheet.media, queries.list(prelude))
	in.media = append(slices.Clip(in.media), len(p.sheet.media)-1)
	p.block(in)
}

// supports reads an @supports rule's block where ui supports what it
// asks, and leaves it out otherwise, as a browser does.
func (p *parser) supports(t Token, in scope) {
	prelude, block := p.prelude()
	switch {
	case !block:
		p.warnf(t.Pos, "@supports has no { block }")
	case supported(prelude):
		p.block(in)
	default:
		p.skipBlock()
	}
}

// supported answers an @supports condition: a (property: value) holds
// when a sheet read as a browser's keeps that declaration, a
// selector(…) when it keeps that selector, and any other function
// doesn't.
func supported(cond []Component) bool {
	cs := items(cond)
	if len(cs) == 0 {
		return false
	}
	if isWord(cs[0], "not") {
		return len(cs) == 2 && !supportedIn(cs[1])
	}
	result, joined := supportedIn(cs[0]), ""
	for i := 1; i < len(cs); i += 2 {
		if i+1 >= len(cs) || !isWord(cs[i], "and", "or") || joined != "" && !isWord(cs[i], joined) {
			return false
		}
		joined = strings.ToLower(cs[i].Text)
		if next := supportedIn(cs[i+1]); joined == "and" {
			result = result && next
		} else {
			result = result || next
		}
	}
	return result
}

func supportedIn(c Component) bool {
	asked := parser{browser: true}
	switch {
	case c.Kind == Function && strings.EqualFold(c.Text, "selector"):
		sels, ok := vendorSelectors(slices.Clone(c.Args))
		if !ok {
			return false
		}
		_, err := parseSelectors(sels)
		return err == nil
	case c.Kind != LParen:
		return false
	}
	inner := trim(c.Args)
	if tail := items(inner); len(tail) > 1 && tail[0].Kind == Ident && tail[1].Kind == Colon {
		_, ok := asked.declaration(slices.Clone(inner))
		return ok
	}
	return supported(inner)
}

// layer reads @layer a, b; which only orders layers, and @layer a { … },
// whose rules stand in a, a layer of its own when it has no name.
func (p *parser) layer(t Token, in scope) {
	prelude, block := p.prelude()
	var names []string
	for _, part := range splitOn(prelude, func(c Component) bool { return c.Kind == Comma }) {
		name, ok := layerName(trim(part))
		if !ok && (!block || len(trim(prelude)) > 0) {
			p.warnf(t.Pos, "@layer: %q is no layer name", strings.TrimSpace(join(part)))
			if block {
				p.skipBlock()
			}
			return
		}
		names = append(names, name)
	}
	switch {
	case !block:
		for _, name := range names {
			p.declare(in.layer, name)
		}
	case len(names) > 1:
		p.warnf(t.Pos, "@layer: a block takes one name")
		p.skipBlock()
	default:
		if names[0] == "" {
			names[0] = fmt.Sprint(anonymous, len(p.sheet.layers))
		}
		in.layer = p.declare(in.layer, names[0])
		p.block(in)
	}
}

// layerName reads a.b.c.
func layerName(cs []Component) (string, bool) {
	var name strings.Builder
	for i, c := range cs {
		switch {
		case i%2 == 0 && c.Kind == Ident:
			name.WriteString(c.Text)
		case i%2 == 1 && c.Kind == Delim && c.Text == "." && i+1 < len(cs):
			name.WriteByte('.')
		default:
			return "", false
		}
	}
	return name.String(), name.Len() > 0
}

// declare names the layer name inside the layer parent (0 for none),
// and the layers it stands in, each at its place from 1 in the sheet's
// list.
func (p *parser) declare(parent int, name string) int {
	full := ""
	if parent > 0 {
		full = p.sheet.layers[parent-1] + "."
	}
	at := 0
	for _, part := range strings.Split(name, ".") {
		full += part
		at = slices.Index(p.sheet.layers, full) + 1
		if at == 0 {
			p.sheet.layers = append(p.sheet.layers, full)
			at = len(p.sheet.layers)
		}
		full += "."
	}
	return at
}

func (p *parser) warnf(pos Pos, format string, args ...any) {
	p.warns = append(p.warns, Warning{File: p.file, Pos: pos, Msg: fmt.Sprintf(format, args...)})
}

func (p *parser) peek() Token { return p.toks[p.i] }

func (p *parser) next() Token {
	t := p.toks[p.i]
	if t.Kind != EOF {
		p.i++
	}
	return t
}

func (p *parser) skipAtRule() {
	for {
		switch p.peek().Kind {
		case EOF:
			return
		case Semicolon:
			p.next()
			return
		case LBrace:
			p.component()
			return
		case Ident, Function, AtKeyword, Hash, String, BadString, URL, BadURL, Delim, Number, Percentage, Dimension,
			Whitespace, CDO, CDC, Colon, Comma, LBracket, RBracket, LParen, RParen, RBrace:
			// nothing special for these tokens
		}
		p.component()
	}
}

func (p *parser) rule() (Rule, bool) {
	pos := p.peek().Pos
	var prelude []Component
	for {
		switch p.peek().Kind {
		case EOF:
			p.warnf(pos, "rule has no { block }")
			return Rule{}, false
		case RBrace:
			if p.depth > 0 {
				p.warnf(pos, "rule has no { block }")
				return Rule{}, false
			}
		case LBrace:
			block := p.component()
			if p.browser {
				var kept bool
				if prelude, kept = vendorSelectors(prelude); !kept {
					return Rule{}, false
				}
			}
			sels, err := parseSelectors(prelude)
			if err != nil {
				p.warnf(err.Pos, "%s; rule dropped", err.Msg)
				return Rule{}, false
			}
			decls := p.declarations(block.Args)
			if pseudoCount(sels) == len(sels) {
				// A rule of pseudo-elements alone says which of its
				// declarations none of them takes; in a rule shared with
				// elements those are left out of the pseudo-element's cascade
				// without a word, as in Chrome.
				kept := decls[:0]
				for _, d := range decls {
					if !slices.ContainsFunc(sels, func(s Selector) bool { return takesDeclared(s.PseudoElement, d.Name) }) {
						p.warnf(d.Pos, "::%s does not take %s", sels[0].PseudoElement, d.Name)
						continue
					}
					kept = append(kept, d)
				}
				decls = kept
			}
			return Rule{Selectors: sels, Decls: decls, Pos: pos}, true
		case Ident, Function, AtKeyword, Hash, String, BadString, URL, BadURL, Delim, Number, Percentage, Dimension,
			Whitespace, CDO, CDC, Colon, Semicolon, Comma, LBracket, RBracket, LParen, RParen:
			// nothing special for these tokens
		}
		prelude = append(prelude, p.component())
	}
}

func (p *parser) component() Component {
	t := p.next()
	var closer Kind
	switch t.Kind {
	case Function, LParen:
		closer = RParen
	case LBracket:
		closer = RBracket
	case LBrace:
		closer = RBrace
	case EOF, Ident, AtKeyword, Hash, String, BadString, URL, BadURL, Delim, Number, Percentage, Dimension, Whitespace,
		CDO, CDC, Colon, Semicolon, Comma, RBracket, RParen, RBrace:
		return Component{Token: t}
	}
	c := Component{Token: t}
	for {
		switch p.peek().Kind {
		case closer:
			p.next()
			return c
		case EOF:
			return c
		case Ident, Function, AtKeyword, Hash, String, BadString, URL, BadURL, Delim, Number, Percentage, Dimension,
			Whitespace, CDO, CDC, Colon, Semicolon, Comma, LBracket, RBracket, LParen, RParen, LBrace, RBrace:
			// nothing special for these tokens
		}
		c.Args = append(c.Args, p.component())
	}
}

func (p *parser) declarations(block []Component) []Declaration {
	var out []Declaration
	for i := 0; i < len(block); {
		c := block[i]
		switch c.Kind {
		case Whitespace, Semicolon:
			i++
			continue
		case EOF, Ident, Function, AtKeyword, Hash, String, BadString, URL, BadURL, Delim, Number, Percentage, Dimension,
			CDO, CDC, Colon, Comma, LBracket, RBracket, LParen, RParen, LBrace, RBrace:
			// nothing special for these tokens
		}
		end := i
		for end < len(block) && block[end].Kind != Semicolon {
			end++
		}
		part := block[i:end]
		i = end
		switch c.Kind {
		case Ident:
			if d, ok := p.declaration(part); ok {
				out = append(out, d)
			}
		case AtKeyword:
			p.warnf(c.Pos, "@%s is not supported", c.Text)
		case EOF, Function, Hash, String, BadString, URL, BadURL, Delim, Number, Percentage, Dimension, Whitespace, CDO, CDC,
			Colon, Semicolon, Comma, LBracket, RBracket, LParen, RParen, LBrace, RBrace:
			p.warnf(c.Pos, "expected a property name, found %q", c.String())
		}
	}
	return out
}

func (p *parser) declaration(part []Component) (Declaration, bool) {
	name := part[0]
	rest := trim(part[1:])
	if len(rest) == 0 || rest[0].Kind != Colon {
		p.warnf(name.Pos, "expected ':' after %q", name.Text)
		return Declaration{}, false
	}
	d := Declaration{Name: name.Text, Value: trim(rest[1:]), Pos: name.Pos, base: p.origin}
	if n := len(d.Value); n >= 2 && d.Value[n-1].Kind == Ident && strings.EqualFold(d.Value[n-1].Text, "important") {
		bang := n - 2
		for bang >= 0 && d.Value[bang].Kind == Whitespace {
			bang--
		}
		if bang >= 0 && d.Value[bang].Kind == Delim && d.Value[bang].Text == "!" {
			d.Important = true
			d.Value = trim(d.Value[:bang])
		}
	}
	if !strings.HasPrefix(d.Name, "--") {
		d.Name = strings.ToLower(d.Name)
	}
	if len(d.Value) == 0 && !strings.HasPrefix(d.Name, "--") {
		p.warnf(name.Pos, "%s has no value", d.Name)
		return Declaration{}, false
	}
	custom := strings.HasPrefix(d.Name, "--")
	written, foreign := d.Name, false
	if v, plain := vendorOf(d.Name); v != nil && p.browser {
		as, other := v.property(plain)
		if as == "" && other == dropped {
			return Declaration{}, false
		}
		if as != "" {
			d.Name = as
		}
	}
	if d.Name == "page-break-inside" {
		// the old name, its auto and avoid break-inside's too
		d.Name = propBreakInside
	}
	if p.browser && !custom {
		foreign = vendorValues(d.Value)
	}
	if msg := checkProperty(d.Name); msg != "" {
		if d.Name != written {
			msg = written + ": " + msg
		}
		p.warnf(name.Pos, "%s", msg)
		return Declaration{}, false
	}
	if !custom && !hasVar(d.Value) {
		if err := validate(d.Name, d.Value); err != nil {
			if !foreign {
				p.warnf(d.Value[0].Pos, "%s: %v", d.Name, err)
			}
			return Declaration{}, false
		}
		d.resolve()
	}
	return d, true
}

// resolve keeps a checked value as the cascade reads it: a shorthand's
// by longhand.
func (d *Declaration) resolve() {
	if _, ok := shorthands[d.Name]; ok {
		d.longhands, _ = expand(d.Name, items(d.Value))
		return
	}
	d.vals = items(d.Value)
}

// pseudoCount counts the pseudo-element selectors in sels.
func pseudoCount(sels []Selector) int {
	n := 0
	for _, s := range sels {
		if s.PseudoElement != "" {
			n++
		}
	}
	return n
}

func trim(cs []Component) []Component {
	for len(cs) > 0 && cs[0].Kind == Whitespace {
		cs = cs[1:]
	}
	for len(cs) > 0 && cs[len(cs)-1].Kind == Whitespace {
		cs = cs[:len(cs)-1]
	}
	return cs
}
