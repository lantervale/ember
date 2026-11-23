package css

import (
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/lantervale/ember/ui/internal/style"
)

// PropertyDefinition registers a custom property, as an @property rule
// or CSS.registerProperty does: the values its Syntax takes compute as
// that type, an invalid one is unset, it inherits only when Inherits
// says so, and Initial is its value where nothing sets one. Initial may
// be empty only for the universal syntax "*".
type PropertyDefinition struct {
	Name     string
	Syntax   string
	Inherits bool
	Initial  string
}

// registration is a custom property's PropertyDefinition, read: its
// syntax, and its initial value as computed, unset for none.
type registration struct {
	name     string
	syntax   syntax
	inherits bool
	initial  string
	unset    bool
	// written is the @property rule as a sheet writes it back.
	written string
}

// registry is the custom properties registered, by name, and those of
// them that don't inherit.
type registry struct {
	props     map[string]*registration
	reset     []*registration
	initials  map[string]string
	functions map[string]*customFunction
}

// syntaxKind is a data type a syntax names, or a keyword.
type syntaxKind uint8

const (
	synKeyword syntaxKind = iota
	synLength
	synNumber
	synPercentage
	synLengthPercentage
	synColor
	synImage
	synURL
	synInteger
	synAngle
	synTime
	synResolution
	synTransformFunction
	synCustomIdent
	synTransformList
	synString
)

//nolint:gochecknoglobals // lookup table
var syntaxTypes = map[string]syntaxKind{
	"length": synLength, "number": synNumber, "percentage": synPercentage,
	"length-percentage": synLengthPercentage, fnColor: synColor, "image": synImage, "url": synURL,
	"integer": synInteger, "angle": synAngle, "time": synTime, nameResolution: synResolution,
	"transform-function": synTransformFunction, "custom-ident": synCustomIdent,
	"transform-list": synTransformList, "string": synString,
}

// syntaxPart is one alternative of a syntax: a type or a keyword, alone,
// space-separated (+) or comma-separated (#).
type syntaxPart struct {
	kind    syntaxKind
	keyword string
	mult    byte
}

// syntax is a registered property's syntax: universal, any value, or
// the first of its parts a value matches.
type syntax struct {
	universal bool
	parts     []syntaxPart
}

// parseSyntax reads a syntax string as CSS Properties and Values: * or
// alternatives split by |, each a <type> or a keyword, + or # after it.
func parseSyntax(src string) (syntax, error) {
	src = strings.TrimSpace(src)
	if src == "*" {
		return syntax{universal: true}, nil
	}
	if src == "" {
		return syntax{}, fmt.Errorf("the syntax is empty")
	}
	var s syntax
	for _, alt := range strings.Split(src, "|") {
		alt = strings.TrimSpace(alt)
		var p syntaxPart
		if n := len(alt); n > 0 && (alt[n-1] == '+' || alt[n-1] == '#') {
			p.mult, alt = alt[n-1], alt[:n-1]
		}
		switch {
		case strings.HasPrefix(alt, "<") && strings.HasSuffix(alt, ">"):
			kind, ok := syntaxTypes[alt[1:len(alt)-1]]
			if !ok {
				return syntax{}, fmt.Errorf("%s is no syntax type", alt)
			}
			if kind == synTransformList && p.mult != 0 {
				return syntax{}, fmt.Errorf("<transform-list> takes no multiplier")
			}
			p.kind = kind
		case keywordName(alt):
			p.keyword = alt
		default:
			return syntax{}, fmt.Errorf("%q is no syntax component", alt)
		}
		s.parts = append(s.parts, p)
	}
	return s, nil
}

// keywordName reports whether s is an identifier a syntax may name: no
// CSS-wide keyword, nor default.
func keywordName(s string) bool {
	toks := Tokenize(s)
	if n := len(toks); n > 0 && toks[n-1].Kind == EOF {
		toks = toks[:n-1]
	}
	return len(toks) == 1 && toks[0].Kind == Ident && !cssWide(toks[0].Text)
}

func cssWide(s string) bool {
	switch strings.ToLower(s) {
	case kwInitial, kwInherit, kwUnset, kwRevert, kwRevertLayer, "default":
		return true
	}
	return false
}

// String writes the syntax back as a syntax string.
func (s syntax) String() string {
	if s.universal {
		return "*"
	}
	parts := make([]string, len(s.parts))
	for i, p := range s.parts {
		w := p.keyword
		if w == "" {
			for name, k := range syntaxTypes {
				if k == p.kind {
					w = "<" + name + ">"
				}
			}
		}
		if p.mult != 0 {
			w += string(p.mult)
		}
		parts[i] = w
	}
	return strings.Join(parts, " | ")
}

// interpolable reports whether values of kind move between two.
func (k syntaxKind) interpolable() bool {
	switch k {
	case synLength, synNumber, synPercentage, synLengthPercentage, synColor, synInteger, synAngle, synTime,
		synResolution, synTransformFunction, synTransformList:
		return true
	case synKeyword, synImage, synURL, synCustomIdent, synString:
	}
	return false
}

// register reads def as a registration, its initial value computed
// as one that may depend on nothing.
func register(def PropertyDefinition) (*registration, error) {
	if !strings.HasPrefix(def.Name, "--") || len(def.Name) < 3 {
		return nil, fmt.Errorf("%q is no custom property name", def.Name)
	}
	syn, err := parseSyntax(def.Syntax)
	if err != nil {
		return nil, err
	}
	r := &registration{name: def.Name, syntax: syn, inherits: def.Inherits}
	init := trim(componentsOf(def.Initial))
	switch {
	case len(init) == 0 && !syn.universal:
		return nil, fmt.Errorf("a syntax other than * needs an initial value")
	case len(init) == 0:
		r.unset = true
	case syn.universal:
		r.initial = join(init)
	default:
		if hasVar(init) || dependent(init) {
			return nil, fmt.Errorf("the initial value %q depends on the element", strings.TrimSpace(def.Initial))
		}
		x := &ctx{c: &initialStyle0, parent: &initialStyle0, env: &measures{}, sized: true, lined: true}
		v, ok := x.typed(syn, init)
		if !ok {
			return nil, fmt.Errorf("the initial value %q is no %s", strings.TrimSpace(def.Initial), syn)
		}
		r.initial = v
	}
	r.written = "@property " + def.Name + " { syntax: " + strconv.Quote(def.Syntax) + "; inherits: " +
		strconv.FormatBool(def.Inherits) + ";"
	if len(init) > 0 {
		r.written += " initial-value: " + join(init) + ";"
	}
	r.written += " }"
	return r, nil
}

//nolint:gochecknoglobals // a blank style the initial values compute on
var initialStyle0 = initialStyle()

// dependent reports whether v has a unit the element measures: those of
// its font, of the viewport, or of a container.
func dependent(v []Component) bool {
	for _, c := range v {
		if c.Kind == Dimension {
			u := strings.ToLower(c.Unit)
			if _, ok := relativeUnits[u]; ok || u == "em" || u == unitRem {
				return true
			}
		}
		if dependent(c.Args) {
			return true
		}
	}
	return false
}

// propertyRule reads an @property rule into the sheet; one that lacks a
// descriptor it needs, or whose initial value its syntax refuses, is
// dropped, as in Chrome.
func (p *parser) propertyRule(t Token) {
	prelude, block := p.prelude()
	if !block {
		p.warnf(t.Pos, "@property has no { block }")
		return
	}
	name := items(prelude)
	if len(name) != 1 || name[0].Kind != Ident || !strings.HasPrefix(name[0].Text, "--") {
		p.warnf(t.Pos, "@property: %q is no custom property name; the rule is dropped", strings.TrimSpace(join(prelude)))
		p.skipBlock()
		return
	}
	var body []Component
	for k := p.peek().Kind; k != EOF && k != RBrace; k = p.peek().Kind {
		body = append(body, p.component())
	}
	p.next()
	def := PropertyDefinition{Name: name[0].Text}
	var syntaxSet, inheritsSet bool
	for _, part := range splitOn(body, func(c Component) bool { return c.Kind == Semicolon }) {
		part = trim(part)
		if len(part) == 0 {
			continue
		}
		v := trim(part[1:])
		if part[0].Kind != Ident || len(v) == 0 || v[0].Kind != Colon {
			p.warnf(part[0].Pos, "expected a descriptor in @property")
			continue
		}
		v = trim(v[1:])
		switch desc := strings.ToLower(part[0].Text); desc {
		case "syntax":
			if len(v) == 1 && v[0].Kind == String {
				def.Syntax, syntaxSet = v[0].Text, true
			}
		case "inherits":
			if len(v) == 1 && isWord(v[0], "true", "false") {
				def.Inherits, inheritsSet = strings.EqualFold(v[0].Text, "true"), true
			}
		case "initial-value":
			def.Initial = join(v)
		default:
			p.warnf(part[0].Pos, "@property: %s is no descriptor", part[0].Text)
		}
	}
	if !syntaxSet || !inheritsSet {
		p.warnf(t.Pos, "@property %s needs a syntax and inherits; the rule is dropped", def.Name)
		return
	}
	r, err := register(def)
	if err != nil {
		p.warnf(t.Pos, "@property %s: %v; the rule is dropped", def.Name, err)
		return
	}
	p.sheet.properties = append(p.sheet.properties, r)
}

// newRegistry is the custom properties defined registers and those the
// sheets' @property rules do, the last rule of a name winning, a
// definition over any rule.
func newRegistry(sheets []*Sheet, defined []*registration) *registry {
	var g *registry
	made := func() {
		if g == nil {
			g = &registry{props: map[string]*registration{}, functions: map[string]*customFunction{}}
		}
	}
	add := func(r *registration) {
		made()
		g.props[r.name] = r
	}
	for _, s := range sheets {
		for _, r := range s.properties {
			add(r)
		}
		for _, f := range s.functions {
			made()
			g.functions[f.name] = f
		}
	}
	for _, r := range defined {
		add(r)
	}
	if g == nil {
		return nil
	}
	g.initials = map[string]string{}
	for _, r := range g.props {
		if !r.unset {
			g.initials[r.name] = r.initial
		}
		if !r.inherits {
			g.reset = append(g.reset, r)
		}
	}
	slices.SortFunc(g.reset, func(a, b *registration) int { return strings.Compare(a.name, b.name) })
	return g
}

// get is the registration of name, nil when it has none.
func (g *registry) get(name string) *registration {
	if g == nil {
		return nil
	}
	return g.props[name]
}

// inherited is what an element's custom properties start from: its
// parent's, those that don't inherit at their initial values, or, for
// the root, every registered one's initial value.
func (g *registry) inherited(parent map[string]string, root bool) map[string]string {
	if g == nil {
		return parent
	}
	if root {
		return g.initials
	}
	out, copied := parent, false
	for _, r := range g.reset {
		v, has := parent[r.name]
		if has == !r.unset && v == r.initial {
			continue
		}
		if !copied {
			copied = true
			out = make(map[string]string, len(parent))
			for k, v := range parent {
				out[k] = v
			}
		}
		if r.unset {
			delete(out, r.name)
		} else {
			out[r.name] = r.initial
		}
	}
	return out
}

// fontRelative reports whether v has a unit of the element's font.
func fontRelative(v []Component) bool {
	for _, c := range v {
		if c.Kind == Dimension {
			u := strings.ToLower(c.Unit)
			if r, ok := relativeUnits[u]; u == "em" || ok && 1<<r&fontUnits != 0 && r < unitRex {
				return true
			}
		}
		if fontRelative(c.Args) {
			return true
		}
	}
	return false
}

// typed is v computed as the first part of s it matches, written as
// getComputedStyle writes it; false when it matches none.
func (x *ctx) typed(s syntax, v []Component) (string, bool) {
	v = trim(v)
	if s.universal {
		return join(v), true
	}
	if len(v) == 0 {
		return "", false
	}
	for _, p := range s.parts {
		if out, ok := x.typedPart(p, v); ok {
			return out, true
		}
	}
	return "", false
}

func (x *ctx) typedPart(p syntaxPart, v []Component) (string, bool) {
	var list [][]Component
	sep := " "
	switch {
	case p.kind == synTransformList || p.mult == '+':
		for _, c := range items(v) {
			if c.Kind == Comma {
				return "", false
			}
			list = append(list, []Component{c})
		}
	case p.mult == '#':
		sep = ", "
		list = splitOn(v, func(c Component) bool { return c.Kind == Comma })
	default:
		list = [][]Component{v}
	}
	kind := p.kind
	if kind == synTransformList {
		kind = synTransformFunction
	}
	out := make([]string, len(list))
	for i, part := range list {
		cs := items(part)
		if len(cs) != 1 {
			return "", false
		}
		w, ok := x.typedOne(p, kind, cs[0])
		if !ok {
			return "", false
		}
		out[i] = w
	}
	return strings.Join(out, sep), true
}

// typedOne is c computed as kind, written back.
func (x *ctx) typedOne(p syntaxPart, kind syntaxKind, c Component) (string, bool) {
	switch kind {
	case synKeyword:
		return c.Text, c.Kind == Ident && c.Text == p.keyword
	case synCustomIdent:
		return c.Text, c.Kind == Ident && !cssWide(c.Text)
	case synString, synURL, synImage:
		return written(kind, c)
	case synColor:
		if isWord(c, kwCurrentColor) {
			return kwCurrentColor, true
		}
		c = inScheme([]Component{c}, x.c.Dark)[0]
		col, err := parseColor(c)
		if err != nil {
			return "", false
		}
		return col.String(), true
	case synResolution:
		return x.resolution(c)
	case synTransformFunction:
		return x.transformFunction(c)
	case synLength, synNumber, synPercentage, synLengthPercentage, synInteger, synAngle, synTime, synTransformList:
	}
	if kind == synLength && c.Kind == Number && c.Num == 0 {
		return "0px", true
	}
	if c.Kind != Number && c.Kind != Percentage && c.Kind != Dimension && c.Kind != Function {
		return "", false
	}
	if kind == synInteger && c.Kind == Number && !c.Int {
		return "", false
	}
	d, err := dimOf(c)
	if err != nil {
		return "", false
	}
	return x.dim(kind, d)
}

// written is c as a <string>, a <url> or an <image>, which compute as
// they are written.
func written(kind syntaxKind, c Component) (string, bool) {
	isURL := c.Kind == URL || c.Kind == Function && strings.EqualFold(c.Text, "url")
	switch {
	case kind == synString:
		return strconv.Quote(c.Text), c.Kind == String
	case kind == synImage:
		return c.String(), isURL || c.Kind == Function && (strings.HasSuffix(strings.ToLower(c.Text), "gradient") ||
			strings.EqualFold(c.Text, "image-set"))
	case c.Kind == URL:
		return "url(" + strconv.Quote(c.Text) + ")", true
	case isURL:
		a := items(c.Args)
		if len(a) != 1 || a[0].Kind != String {
			return "", false
		}
		return "url(" + strconv.Quote(a[0].Text) + ")", true
	}
	return "", false
}

// dim is d computed as kind.
func (x *ctx) dim(kind syntaxKind, d Dim) (string, bool) {
	d = d.Absolute(x.units(&d))
	switch kind {
	case synLength, synLengthPercentage:
		if d.typ != typeLength && (kind != synLengthPercentage || d.typ != typePercent) {
			return "", false
		}
		if len(d.terms) > 0 {
			return "", false
		}
		if !d.pct {
			return cssNum(d.Px) + "px", true
		}
		if kind == synLength {
			return "", false
		}
		return writeLP(d.Px, d.Pct, d.Px != 0, true), true
	case synPercentage:
		v, ok := d.constantPct()
		return cssNum(v) + "%", ok && d.typ == typePercent
	case synNumber, synInteger:
		v, ok := d.constant()
		if !ok || d.typ != typeNumber {
			return "", false
		}
		if kind == synInteger {
			v = math.Floor(v + 0.5)
		}
		return cssNum(v), true
	case synAngle:
		v, ok := d.constant()
		return cssNum(v) + unitDeg, ok && d.typ == typeAngle
	case synTime:
		v, ok := d.constant()
		return cssNum(v) + "s", ok && d.typ == typeTime
	case synKeyword, synColor, synImage, synURL, synResolution, synTransformFunction, synCustomIdent, synTransformList,
		synString:
	}
	return "", false
}

// constantPct is d's percentage when nothing else is in it.
func (d Dim) constantPct() (float64, bool) {
	if len(d.terms) > 0 || d.Px != 0 || d.Num != 0 || d.Deg != 0 {
		return 0, false
	}
	return d.Pct, true
}

// writeLP writes a length-percentage as Chrome computes it: a
// length or a percentage alone, else a calc() of both, the percentage
// first.
func writeLP(px, pct float64, hasPx, hasPct bool) string {
	switch {
	case !hasPct:
		return cssNum(px) + "px"
	case !hasPx:
		return cssNum(pct) + "%"
	case px < 0:
		return "calc(" + cssNum(pct) + "% - " + cssNum(-px) + "px)"
	}
	return "calc(" + cssNum(pct) + "% + " + cssNum(px) + "px)"
}

//nolint:gochecknoglobals // lookup table
var dppxUnits = map[string]float64{"dppx": 1, "x": 1, "dpi": 1.0 / 96, "dpcm": 2.54 / 96}

func (x *ctx) resolution(c Component) (string, bool) {
	if c.Kind != Dimension {
		return "", false
	}
	f, ok := dppxUnits[strings.ToLower(c.Unit)]
	return cssNum(c.Num*f) + "dppx", ok
}

// transformFunction is a 2D transform function, its lengths in pixels
// and its angles in degrees.
func (x *ctx) transformFunction(c Component) (string, bool) {
	if c.Kind != Function || checkTransform([]Component{c}) != nil {
		return "", false
	}
	args := splitOn(items(c.Args), func(c Component) bool { return c.Kind == Comma })
	out := make([]string, len(args))
	for i, a := range args {
		d, err := dimOf(a[0])
		if err != nil {
			return "", false
		}
		d = d.Absolute(x.units(&d))
		switch d.typ {
		case typeAngle:
			v, _ := d.constant()
			out[i] = cssNum(v) + unitDeg
		case typeLength, typePercent:
			if len(d.terms) > 0 {
				return "", false
			}
			out[i] = writeLP(d.Px, d.Pct, d.typ == typeLength, d.pct)
		default:
			v, _ := d.constant()
			out[i] = cssNum(v)
		}
	}
	return c.Text + "(" + strings.Join(out, ", ") + ")", true
}

// cssNum writes v as getComputedStyle does: six significant digits.
func cssNum(v float64) string {
	if v == 0 || math.IsNaN(v) {
		return "0"
	}
	return strconv.FormatFloat(v, 'g', 6, 64)
}

// registered is the computed value of the registered property r for v,
// its var()s substituted: inherit, initial and unset as CSS-wide
// keywords, and a value its syntax refuses unset; false for the
// guaranteed-invalid value.
func (x *ctx) registered(r *registration, v []Component, ok bool) (string, bool) {
	kw := kwUnset
	if ok {
		kw = globalKeyword(v)
	}
	if kw == "" {
		if fontRelative(v) {
			x.fontRead = true
		}
		if out, typed := x.typed(r.syntax, v); typed {
			return out, true
		}
		kw = kwUnset
	}
	if kw == kwUnset && r.inherits || kw == kwInherit {
		if x.root {
			return r.initial, !r.unset
		}
		out, has := x.parent.Vars[r.name]
		return out, has
	}
	return r.initial, !r.unset
}

// typedValue is one item of a computed value read for interpolating.
type typedValue struct {
	kind     syntaxKind
	num, pct float64
	hasPx    bool
	hasPct   bool
	color    Color
	fn       string
	args     []typedValue
	text     string
}

// readTyped reads text, a value computed for s, as the part of s it is
// and its items; false when it can't move.
func readTyped(s syntax, text string) (int, []typedValue, bool) {
	if s.universal {
		return 0, nil, false
	}
	v := trim(componentsOf(text))
	x := &ctx{c: &initialStyle0, parent: &initialStyle0, env: &measures{}, sized: true, lined: true}
	for i, p := range s.parts {
		if _, ok := x.typedPart(p, v); !ok {
			continue
		}
		if !p.kind.interpolable() {
			return i, nil, false
		}
		var list [][]Component
		if p.mult == '#' {
			list = splitOn(v, func(c Component) bool { return c.Kind == Comma })
		} else {
			for _, c := range items(v) {
				list = append(list, []Component{c})
			}
		}
		out := make([]typedValue, len(list))
		for k, part := range list {
			tv, ok := readOne(p.kind, items(part)[0])
			if !ok {
				return i, nil, false
			}
			out[k] = tv
		}
		return i, out, true
	}
	return -1, nil, false
}

func readOne(kind syntaxKind, c Component) (typedValue, bool) {
	tv := typedValue{kind: kind}
	switch kind {
	case synColor:
		if isWord(c, kwCurrentColor) {
			return tv, false
		}
		col, err := parseColor(c)
		tv.color = col
		return tv, err == nil
	case synTransformFunction, synTransformList:
		tv.kind, tv.fn = synTransformFunction, c.Text
		for _, a := range splitOn(items(c.Args), func(c Component) bool { return c.Kind == Comma }) {
			at, ok := readOne(synLengthPercentage, a[0])
			if !ok {
				return tv, false
			}
			tv.args = append(tv.args, at)
		}
		return tv, true
	case synKeyword, synImage, synURL, synCustomIdent, synString:
		return tv, false
	case synLength, synNumber, synPercentage, synLengthPercentage, synInteger, synAngle, synTime, synResolution:
	}
	switch c.Kind {
	case Number:
		tv.num = c.Num
	case Percentage:
		tv.pct, tv.hasPct = c.Num, true
	case Dimension:
		tv.num, tv.hasPx, tv.text = c.Num, true, c.Unit
	case Function:
		d, err := dimOf(c)
		if err != nil {
			return tv, false
		}
		tv.num, tv.pct, tv.hasPx, tv.hasPct, tv.text = d.Px, d.Pct, d.Px != 0, true, "px"
	case EOF, Ident, AtKeyword, Hash, String, BadString, URL, BadURL, Delim, Whitespace, CDO, CDC, Colon, Semicolon,
		Comma, LBracket, RBracket, LParen, RParen, LBrace, RBrace:
		return tv, false
	}
	return tv, true
}

// current is v, a computed value of r, with currentcolor the colour of
// s, as an interpolation of a <color> reads it.
func (r *registration) current(v string, s *style.Computed) string {
	if r == nil || v != kwCurrentColor {
		return v
	}
	rgb := s.Color.SRGB()
	return Color{R: float64(rgb[0]), G: float64(rgb[1]), B: float64(rgb[2]), A: float64(s.Color.A)}.String()
}

// moves reports whether r interpolates between a and b, two of its
// computed values, where they would otherwise flip.
func (r *registration) moves(a, b string) bool {
	_, ok := r.lerp(a, b, 0.5)
	return ok
}

// interpolate is the value of r t of the way from a to b, two of its
// computed values: item by item where both are the same part of its
// syntax, of a type that moves, with as many items; flipping halfway
// otherwise.
func (r *registration) interpolate(a, b string, t float32) string {
	if v, ok := r.lerp(a, b, t); ok {
		return v
	}
	if t < 0.5 {
		return a
	}
	return b
}

// lerp is interpolate where a and b move, false where they flip.
func (r *registration) lerp(a, b string, t float32) (string, bool) {
	if r == nil {
		return "", false
	}
	pa, va, okA := readTyped(r.syntax, a)
	pb, vb, okB := readTyped(r.syntax, b)
	if !okA || !okB || pa != pb || len(va) != len(vb) {
		return "", false
	}
	out := make([]string, len(va))
	for i := range va {
		var ok bool
		if out[i], ok = lerpTyped(va[i], vb[i], float64(t)); !ok {
			return "", false
		}
	}
	sep := " "
	if r.syntax.parts[pa].mult == '#' {
		sep = ", "
	}
	return strings.Join(out, sep), true
}

func lerpTyped(a, b typedValue, t float64) (string, bool) {
	f := func(x, y float64) float64 { return x + (y-x)*t }
	switch a.kind {
	case synColor:
		pa, pb := [4]float64{a.color.R * a.color.A, a.color.G * a.color.A, a.color.B * a.color.A, a.color.A},
			[4]float64{b.color.R * b.color.A, b.color.G * b.color.A, b.color.B * b.color.A, b.color.A}
		alpha := clamp01(f(pa[3], pb[3]))
		c := Color{A: alpha}
		if alpha > 0 {
			c.R, c.G, c.B = clamp01(f(pa[0], pb[0])/alpha), clamp01(f(pa[1], pb[1])/alpha), clamp01(f(pa[2], pb[2])/alpha)
		}
		return c.String(), true
	case synTransformFunction:
		if !strings.EqualFold(a.fn, b.fn) || len(a.args) != len(b.args) {
			return "", false
		}
		out := make([]string, len(a.args))
		for i := range a.args {
			x, y := a.args[i], b.args[i]
			if x.text != y.text && x.hasPx && y.hasPx {
				return "", false
			}
			switch {
			case x.text == unitDeg || y.text == unitDeg:
				out[i] = cssNum(f(x.num, y.num)) + unitDeg
			case x.hasPx || y.hasPx || x.hasPct || y.hasPct:
				out[i] = writeLP(f(x.num, y.num), f(x.pct, y.pct), x.hasPx || y.hasPx, x.hasPct || y.hasPct)
			default:
				out[i] = cssNum(f(x.num, y.num))
			}
		}
		return a.fn + "(" + strings.Join(out, ", ") + ")", true
	case synInteger:
		return cssNum(math.Floor(f(a.num, b.num) + 0.5)), true
	case synNumber:
		return cssNum(f(a.num, b.num)), true
	case synPercentage:
		return cssNum(f(a.pct, b.pct)) + "%", true
	case synLengthPercentage:
		return writeLP(f(a.num, b.num), f(a.pct, b.pct), a.hasPx || b.hasPx, a.hasPct || b.hasPct), true
	case synLength, synAngle, synTime, synResolution:
		return cssNum(f(a.num, b.num)) + a.text, true
	case synKeyword, synImage, synURL, synCustomIdent, synTransformList, synString:
	}
	return "", false
}
