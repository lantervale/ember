package css

import (
	"fmt"
	"io/fs"
	"net/url"
	"path"
	"slices"
	"strconv"
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
	// flow is set on a logical declaration, named as in horizontal
	// left-to-right lines: an element of another writing-mode or of
	// direction rtl gives its value to the physical longhand it stands for.
	flow bool
	// logical is the name a logical declaration was written with.
	logical string
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
	name := d.Name
	if d.logical != "" {
		name = d.logical
	}
	s := name + ": " + join(d.Value)
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
	// containers are the @container rules it stands in, the outermost
	// first: it applies to an element while they all hold for it.
	containers []*containerRule
	// layer is the @layer it stands in, as a place from 1 in its sheet's
	// list; 0 in none.
	layer int
	// in is the sheet an @import brought it from, nil for its sheet's
	// own.
	in *Sheet
	// starting is set in an @starting-style rule: it applies only to the
	// style a transition starts from where an element had none.
	starting bool
	// scoped is the innermost @scope rule it stands in: it applies to an
	// element in the scope of one of that rule's roots.
	scoped *scopeRule
	// nest is, from 1, the place in its sheet's nests of the outermost
	// rule holding others it was read from, 0 for none.
	nest int
}

// nest is a style rule with rules nested in it, or an @scope rule, as
// written: the first rule read from it stands in its media, containers,
// @starting-style and @scope rules, and as many more of each as it
// opened itself before that rule.
type nest struct {
	text              string
	media, containers int
	starting          bool
	scopes            int
}

// Keyframes is an @keyframes rule: what the animations naming it run
// through, a keyframe for each offset its selectors give, in the order
// written.
type Keyframes struct {
	Name   string
	Frames []Keyframe
	Pos    Pos
	// media are the @media rules it stands in, as a Rule's, and written
	// its name as written.
	media   []int
	written string
}

// Keyframe is the declarations of one offset of an @keyframes rule, from
// 0 to 1, with the animation-timing-function of the interval it begins
// among them.
type Keyframe struct {
	At    float32
	Decls []Declaration
}

// Sheet is a parsed stylesheet. Name is the file it came from.
type Sheet struct {
	Name          string
	Rules         []Rule
	Keyframes     []Keyframes
	CounterStyles []CounterStyle
	PositionTries []PositionTry
	faces         []FontFace
	palettes      []FontPaletteValues
	once          sync.Once
	index         *index
	// media are its @media rules' queries, in the order they open.
	media []*mediaList
	// layers are the full names of its cascade layers (theme.dark), in
	// the order they are first named; one without a name begins with
	// anonymous.
	layers []string
	// nests are its style rules with rules nested in them, and its
	// @scope rules, which it writes back as they were written.
	nests []nest
	// browser is whether it was read as a browser reads it (Browser), src
	// its text then, and from the address it came from (From).
	browser bool
	src     string
	from    string
	origin  *origin
	// text is the text of a sheet ParseRules read, joined the sheets a
	// sheet Join made stands for, and within the media query list Join
	// put them all in.
	text   string
	joined []*Sheet
	within string
	// imports are its @import rules; a sheet Imported made has none, and
	// stands for base with the sheets spliced in.
	imports []Import
	base    *Sheet
	spliced []splice
	// properties are its @property rules, and functions its @function
	// rules, in order.
	properties []*registration
	functions  []*customFunction
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
	if s.browser && s.base != nil {
		return s.importedText()
	}
	if s.browser && s.joined != nil {
		var b strings.Builder
		for _, part := range s.joined {
			b.WriteString(part.String())
			b.WriteString("\n")
		}
		if s.within != "" {
			return "@media " + s.within + " {\n" + b.String() + "}\n"
		}
		return b.String()
	}
	if s.browser {
		if s.text != "" {
			return s.text
		}
		return s.src
	}
	var b strings.Builder
	b.WriteString(paletteRules(s.palettes))
	written := make([]bool, len(s.nests))
	for _, r := range s.Rules {
		if r.nest > 0 {
			if written[r.nest-1] {
				continue
			}
			written[r.nest-1] = true
		}
		s.writeRule(&b, r)
	}
	for _, k := range s.Keyframes {
		for _, m := range k.media {
			b.WriteString("@media " + s.media[m].text + " { ")
		}
		b.WriteString("@keyframes " + k.written + " {")
		for _, f := range k.Frames {
			b.WriteString(" " + strconv.FormatFloat(float64(f.At)*100, 'f', -1, 32) + "% {")
			for _, d := range f.Decls {
				b.WriteString(" " + d.String() + ";")
			}
			b.WriteString(" }")
		}
		b.WriteString(" }" + strings.Repeat(" }", len(k.media)) + "\n")
	}
	for _, pt := range s.PositionTries {
		b.WriteString("@position-try " + pt.Name + " {")
		for _, d := range pt.Decls {
			b.WriteString(" " + d.String() + ";")
		}
		b.WriteString(" }\n")
	}
	for _, cs := range s.CounterStyles {
		for _, m := range cs.media {
			b.WriteString("@media " + s.media[m].text + " { ")
		}
		b.WriteString(cs.String() + strings.Repeat(" }", len(cs.media)) + "\n")
	}
	for _, r := range s.properties {
		b.WriteString(r.written + "\n")
	}
	for _, f := range s.functions {
		b.WriteString(f.written + "\n")
	}
	return b.String()
}

// writeRule writes r in the rules it stands in; the first rule read from
// a nest stands for the whole nest, as written.
func (s *Sheet) writeRule(b *strings.Builder, r Rule) {
	var n nest
	if r.nest > 0 {
		n = s.nests[r.nest-1]
		r.media, r.containers = r.media[:len(r.media)-n.media], r.containers[:len(r.containers)-n.containers]
		r.starting = r.starting && !n.starting
		for range n.scopes {
			r.scoped = r.scoped.parent
		}
	}
	for _, m := range r.media {
		b.WriteString("@media " + s.media[m].text + " { ")
	}
	closing := len(r.media) + len(r.containers)
	if r.starting {
		b.WriteString("@starting-style { ")
		closing++
	}
	for _, c := range r.containers {
		b.WriteString("@container " + c.text + " { ")
	}
	for _, sr := range r.scoped.chain() {
		b.WriteString("@scope " + sr.text + " { ")
		closing++
	}
	if r.nest > 0 {
		b.WriteString(n.text + strings.Repeat(" }", closing) + "\n")
		return
	}
	for i, sel := range r.Selectors {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(sel.String())
	}
	b.WriteString(" {")
	for _, d := range r.Decls {
		b.WriteString(" " + d.String() + ";")
	}
	b.WriteString(" }" + strings.Repeat(" }", closing) + "\n")
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
	text, _ := Decode(data, "")
	s, warns := Parse(name, text, opts...)
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
	// depth counts the @media, @supports and @layer blocks being read,
	// and begun is whether a rule an @import must come before was read.
	depth int
	begun bool
	// holding counts the rules holding others being read, one inside the
	// other: style rules, and @scope rules out of any.
	holding int
}

// scope is the @media, @container and @scope rules and the @layer a
// rule stands in, and the style rule it is nested in.
type scope struct {
	media      []int
	containers []*containerRule
	layer      int
	starting   bool
	scoped     *scopeRule
	nest       *nesting
}

// rules reads rules up to the end of the sheet, or of the block they
// stand in.
func (p *parser) rules(in scope) {
	if in.nest != nil {
		p.nestedRules(in, nil)
		return
	}
	for p.item(in) {
	}
}

// item reads the next rule, or the white space before it, and reports
// whether there may be more: false at the end of the sheet, or of the
// block the rules stand in.
func (p *parser) item(in scope) bool {
	t := p.peek()
	switch t.Kind {
	case EOF:
		return false
	case Whitespace, CDO, CDC:
		p.next()
	case AtKeyword:
		p.next()
		p.atRule(t, in)
	case RBrace:
		if p.depth > 0 {
			p.next()
			return false
		}
		p.styleRule(in)
	case Ident, Function, Hash, String, BadString, URL, BadURL, Delim, Number, Percentage, Dimension, Colon, Semicolon,
		Comma, LBracket, RBracket, LParen, RParen, LBrace:
		p.styleRule(in)
	}
	return true
}

// styleRule reads a style rule and the rules nested in it. The
// outermost one, when it nests any, keeps its text for String.
func (p *parser) styleRule(in scope) {
	start, pos := p.i, p.peek().Pos
	prelude, ok := p.selectorPrelude()
	if !ok {
		return
	}
	sels, written, ok := p.selectors(prelude, in.nest)
	if !ok {
		p.skipBlock()
		return
	}
	p.begun = true
	from, outer := len(p.sheet.Rules), p.holding == 0
	inner := in
	inner.nest = &nesting{sels: sels, written: written}
	p.depth++
	p.holding++
	nested := p.nestedRules(inner, &Rule{Selectors: sels, Pos: pos})
	p.holding--
	p.depth--
	if outer && nested {
		p.written(start, from, in)
	}
}

// written keeps the text of the tokens from start on as the nest of the
// rules from from on, read in in.
func (p *parser) written(start, from int, in scope) {
	if from == len(p.sheet.Rules) {
		return
	}
	var text strings.Builder
	for _, t := range p.toks[start:p.i] {
		text.WriteString(t.String())
	}
	first := p.sheet.Rules[from]
	n := nest{text: strings.TrimSpace(text.String()), media: len(first.media) - len(in.media),
		containers: len(first.containers) - len(in.containers), starting: first.starting && !in.starting,
		scopes: len(first.scoped.chain()) - len(in.scoped.chain())}
	p.sheet.nests = append(p.sheet.nests, n)
	for i := from; i < len(p.sheet.Rules); i++ {
		p.sheet.Rules[i].nest = len(p.sheet.nests)
	}
}

// selectorPrelude reads a style rule's selectors up to its block, which
// it enters.
func (p *parser) selectorPrelude() ([]Component, bool) {
	pos := p.peek().Pos
	var prelude []Component
	for {
		switch k := p.peek().Kind; {
		case k == EOF || k == RBrace && p.depth > 0:
			p.warnf(pos, "rule has no { block }")
			return nil, false
		case k == LBrace:
			p.next()
			return prelude, true
		}
		prelude = append(prelude, p.component())
	}
}

// selectors reads a style rule's selector list, a nested rule's relative
// to the rule it stands in; written is the list as read, which an & of
// the rules nested in it stands for.
func (p *parser) selectors(prelude []Component, n *nesting) ([]Selector, []Component, bool) {
	prelude = n.resolve(prelude)
	if p.browser {
		var kept bool
		if prelude, kept = vendorSelectors(prelude); !kept {
			return nil, nil, false
		}
		if prelude = matched(prelude); prelude == nil {
			return nil, nil, false
		}
	}
	sels, err := parseSelectors(prelude)
	if err != nil {
		p.warnf(err.Pos, "%s; rule dropped", err.Msg)
		return nil, nil, false
	}
	return sels, trim(prelude), true
}

// nestedRules reads the block of a style rule, or of a group rule in
// one, up to its end. lead, the style rule, takes the declarations
// before anything nested; a run of them after a nested rule is a rule of
// its own in its place, of the selectors of the rule it stands in, as
// Chrome's CSSNestedDeclarations. It reports whether anything was
// nested.
func (p *parser) nestedRules(in scope, lead *Rule) bool {
	var decls []Declaration
	nested := false
	flush := func() {
		switch {
		case lead != nil:
			lead.Decls = p.taken(lead.Selectors, decls)
			p.add(*lead, in)
			lead = nil
		case len(decls) > 0:
			p.add(Rule{Selectors: in.nest.sels, Decls: p.taken(in.nest.sels, decls), Pos: decls[0].Pos}, in)
		}
		decls = nil
	}
	for {
		switch t := p.peek(); t.Kind {
		case EOF:
			flush()
			return nested
		case RBrace:
			p.next()
			flush()
			return nested
		case Whitespace, Semicolon, CDO, CDC:
			p.next()
		case AtKeyword:
			p.next()
			name, ok := p.atName(t)
			if !ok || !p.nestable(name) {
				if ok {
					p.warnf(t.Pos, "@%s is not supported inside a style rule", t.Text)
				}
				p.skipAtRule()
				continue
			}
			flush()
			nested = true
			p.atRule(t, in)
		case Ident, Function, Hash, String, BadString, URL, BadURL, Delim, Number, Percentage, Dimension, Colon, Comma,
			LBracket, RBracket, LParen, RParen, LBrace:
			if d, ok := p.nestedDeclaration(); ok {
				decls = append(decls, d...)
				continue
			}
			flush()
			nested = true
			p.styleRule(in)
		}
	}
}

// add puts r in the sheet, in what in stands for.
func (p *parser) add(r Rule, in scope) {
	r.media, r.containers, r.layer, r.starting, r.scoped = in.media, in.containers, in.layer, in.starting, in.scoped
	p.sheet.Rules = append(p.sheet.Rules, r)
}

// nestedDeclaration reads a declaration where a nested rule may stand
// instead. What holds a { block } is a rule, which it leaves unread and
// reports false for, unless it reads as a declaration whose value is
// that block alone, or a custom property's.
func (p *parser) nestedDeclaration() ([]Declaration, bool) {
	start := p.i
	var part []Component
	block := false
	for k := p.peek().Kind; k != EOF && k != Semicolon && k != RBrace; k = p.peek().Kind {
		c := p.component()
		block = block || c.Kind == LBrace
		part = append(part, c)
	}
	if block {
		rest := trim(part[1:])
		value := []Component(nil)
		if part[0].Kind == Ident && len(rest) > 0 && rest[0].Kind == Colon {
			value = trim(rest[1:])
		}
		custom := part[0].Kind == Ident && strings.HasPrefix(part[0].Text, "--")
		if value == nil || !custom && (len(value) != 1 || value[0].Kind != LBrace) {
			p.i = start
			return nil, false
		}
	}
	if p.peek().Kind == Semicolon {
		p.next()
	}
	return p.declarations(part), true
}

// taken are the declarations of decls a rule of sels takes: a rule of
// pseudo-elements alone says which of its declarations none of them
// takes; in a rule shared with elements those are left out of the
// pseudo-element's cascade without a word, as in Chrome.
func (p *parser) taken(sels []Selector, decls []Declaration) []Declaration {
	if len(sels) == 0 || pseudoCount(sels) != len(sels) {
		return decls
	}
	kept := decls[:0]
	for _, d := range decls {
		if !slices.ContainsFunc(sels, func(s Selector) bool { return takesDeclared(s.PseudoElement, d.Name) }) {
			p.warnf(d.Pos, "::%s does not take %s", sels[0].PseudoElement, d.Name)
			continue
		}
		kept = append(kept, d)
	}
	return kept
}

// atName is the name of an at-rule as ui reads it, a vendor's as the
// plain one it stands for; false for one a browser's sheet drops.
func (p *parser) atName(t Token) (string, bool) {
	name := strings.ToLower(t.Text)
	if v, rest := vendorOf(name); v != nil && p.browser {
		as, other := v.atRule(rest)
		if as == "" && other == dropped {
			return "", false
		}
		if as != "" {
			name = as
		}
	}
	return name, true
}

// nestable reports whether the at-rule name may stand in a style rule:
// the conditional and grouping rules, which hold declarations and rules.
func (p *parser) nestable(name string) bool {
	switch name {
	case atMedia, kwContainer, "starting-style", kwScope:
		return true
	case atSupports, atLayer:
		return p.browser
	}
	return false
}

func (p *parser) atRule(t Token, in scope) {
	name, read := p.atName(t)
	if !read {
		p.skipAtRule()
		return
	}
	if name != "import" && name != "charset" && name != atLayer {
		p.begun = true
	}
	switch {
	case name == "import":
		p.importRule(t)
	case name == "font-face":
		if f, ok := p.fontFace(t.Pos); ok {
			p.sheet.faces = append(p.sheet.faces, f)
		}
	case name == "font-palette-values":
		p.fontPaletteValues(t)
	case name == atMedia:
		p.media(t, in)
	case name == kwContainer:
		p.container(t, in)
	case name == atKeyframes:
		p.keyframes(t, in)
	case name == "starting-style":
		p.startingStyle(t, in)
	case name == kwScope:
		p.scope(t, in)
	case name == "counter-style":
		p.counterStyle(t, in)
	case name == propPositionTry:
		p.positionTry(t)
	case name == "property":
		p.propertyRule(t)
	case name == "function":
		p.functionRule(t)
	case name == atSupports && p.browser:
		p.supports(t, in)
	case name == atLayer && p.browser:
		p.layer(t, in)
	case name == "charset":
		p.skipAtRule()
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

// container reads an @container rule, dropped whole when its prelude is
// malformed, as Chrome drops it.
func (p *parser) container(t Token, in scope) {
	prelude, block := p.prelude()
	if !block {
		p.warnf(t.Pos, "@container has no { block }")
		return
	}
	queries := mediaParser{container: true, warn: p.warnf}
	r, ok := queries.containerRule(prelude)
	if !ok {
		p.warnf(t.Pos, "@container: %q is no container condition; the rule is dropped",
			strings.TrimSpace(join(prelude)))
		p.skipBlock()
		return
	}
	in.containers = append(slices.Clip(in.containers), r)
	p.block(in)
}

// startingStyle reads an @starting-style rule: its rules style where a
// transition starts from on an element that had no style.
func (p *parser) startingStyle(t Token, in scope) {
	if _, block := p.prelude(); !block {
		p.warnf(t.Pos, "@starting-style has no { block }")
		return
	}
	in.starting = true
	p.block(in)
}

// keyframes reads an @keyframes rule: its name, then keyframes of a
// selector list and a block of declarations each.
func (p *parser) keyframes(t Token, in scope) {
	prelude, block := p.prelude()
	if !block {
		p.warnf(t.Pos, "@keyframes has no { block }")
		return
	}
	name := items(prelude)
	if len(name) != 1 || keyframesName(name[0]) != nil {
		p.warnf(t.Pos, "@keyframes: %q is no keyframes name", strings.TrimSpace(join(prelude)))
		p.skipBlock()
		return
	}
	k := Keyframes{Name: name[0].Text, Pos: t.Pos, media: in.media, written: name[0].String()}
	var sel []Component
	for kind := p.peek().Kind; kind != EOF; kind = p.peek().Kind {
		switch kind {
		case RBrace:
			p.next()
			p.sheet.Keyframes = append(p.sheet.Keyframes, k)
			return
		case LBrace:
			k.Frames = append(k.Frames, p.keyframe(sel, p.component().Args)...)
			sel = nil
			continue
		case EOF, Ident, Function, AtKeyword, Hash, String, BadString, URL, BadURL, Delim, Number, Percentage, Dimension,
			Whitespace, CDO, CDC, Colon, Semicolon, Comma, LBracket, RBracket, LParen, RParen:
			// part of the next keyframe's selectors
		}
		sel = append(sel, p.component())
	}
	p.sheet.Keyframes = append(p.sheet.Keyframes, k)
}

// keyframe reads one keyframe: a keyframe for each offset sel gives,
// none when one of them is no offset. Its declarations leave out what
// a keyframe can't set: !important, which Chrome ignores there, and the
// properties that don't animate.
func (p *parser) keyframe(sel, block []Component) []Keyframe {
	var at []float32
	for _, part := range splitOn(sel, func(c Component) bool { return c.Kind == Comma }) {
		cs := items(part)
		switch {
		case len(cs) == 1 && isWord(cs[0], "from"):
			at = append(at, 0)
		case len(cs) == 1 && isWord(cs[0], "to"):
			at = append(at, 1)
		case len(cs) == 1 && cs[0].Kind == Percentage && cs[0].Num >= 0 && cs[0].Num <= 100:
			at = append(at, float32(cs[0].Num/100))
		default:
			pos := Pos{}
			if len(sel) > 0 {
				pos = sel[0].Pos
			}
			p.warnf(pos, "@keyframes: %q is no keyframe selector; keyframe dropped", strings.TrimSpace(join(part)))
			return nil
		}
	}
	var decls []Declaration
	for _, d := range p.declarations(block) {
		switch {
		case d.Important:
			p.warnf(d.Pos, "!important in a keyframe is ignored, as in Chrome: %s dropped", d.Name)
		case !animates(d.Name):
			p.warnf(d.Pos, "%s does not animate in a keyframe", d.Name)
		default:
			decls = append(decls, d)
		}
	}
	out := make([]Keyframe, len(at))
	for i, a := range at {
		out[i] = Keyframe{At: a, Decls: decls}
	}
	return out
}

// animates reports whether a keyframe may set the property name: one of
// its longhands takes part in animations.
func animates(name string) bool {
	if name == propAnimationTiming || name == propAnimationComposition {
		return true
	}
	if strings.HasPrefix(name, "--") {
		return true
	}
	return slices.ContainsFunc(longhandsOf(name), func(l string) bool {
		p, ok := props[l]
		return !ok || p.anim.kind != still
	})
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
		var outside *nesting
		sels, ok := vendorSelectors(outside.resolve(slices.Clone(c.Args)))
		if !ok {
			return false
		}
		_, err := parseSelectors(sels)
		return err == nil
	case c.Kind == Function && strings.EqualFold(c.Text, "font-tech"):
		tech := items(trim(c.Args))
		return len(tech) == 1 && tech[0].Kind == Ident && colourTech(tech[0].Text)
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
	case !block && in.nest != nil:
		p.warnf(t.Pos, "@layer without a block is dropped inside a style rule, as in Chrome")
	case !block:
		for _, name := range names {
			p.declare(in.layer, name)
		}
	case len(names) > 1:
		p.warnf(t.Pos, "@layer: a block takes one name")
		p.skipBlock()
	default:
		p.begun = true
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
			if c.Kind == AtKeyword && block[end-1].Kind == LBrace {
				// a nested at-rule ends with its block
				break
			}
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
	if !p.vendorProperty(&d, name.Pos) {
		return Declaration{}, false
	}
	if as, ok := logical[d.Name]; ok {
		d.logical = d.Name
		d.Name, d.flow = as, true
	} else if flowShorthands[d.Name] {
		d.flow = true
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

// vendorProperty reads d, a declaration of a vendor's property in a
// browser's sheet, as the plain one the vendor reads it as, and reports
// false when the vendor drops it.
func (p *parser) vendorProperty(d *Declaration, at Pos) bool {
	if as, ok := oldNames[d.Name]; ok && p.browser {
		d.Name = as
		return true
	}
	v, plain := vendorOf(d.Name)
	if v == nil || !p.browser {
		return true
	}
	written := d.Name
	as, other := v.property(plain)
	if as == "" {
		return other != dropped
	}
	d.Name = as
	v.plainWords(plain, d.Value)
	if v != &webkit {
		return true
	}
	var err error
	if d.Value, err = webkitMask(plain, d.Value); err != nil {
		p.warnf(at, "%s: %v", written, err)
		return false
	}
	return true
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
