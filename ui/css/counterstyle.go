package css

import (
	_ "embed"
	"fmt"
	"math"
	"slices"
	"strings"
	"sync"

	"github.com/lantervale/ember/ui/internal/style"
)

// predefinedStyles are the counter styles CSS Counter Styles predefines,
// as its own @counter-style rules, read as the user agent's.
//
//go:embed counterstyles.css
var predefinedStyles string

// predefinedSheet is the name of the sheet predefinedStyles is read as.
const predefinedSheet = "ember:counter-styles"

// CounterStyle is an @counter-style rule: its name and its descriptors,
// each checked, by name.
type CounterStyle struct {
	Name  string
	Pos   Pos
	Descs map[string][]Component
	// media are the @media rules it stands in, as a Rule's.
	media []int
}

// fixedStyles are the counter styles no @counter-style may redefine.
//
//nolint:gochecknoglobals // a fixed list
var fixedStyles = []string{kwDecimal, kwDisc, kwSquare, kwCircle, "disclosure-open", "disclosure-closed"}

// counterStyleDescriptors check each descriptor of an @counter-style.
//
//nolint:gochecknoglobals // lookup table
var counterStyleDescriptors = map[string]func([]Component) error{
	"system":           checkSystem,
	"symbols":          checkSymbols,
	"additive-symbols": checkAdditiveSymbols,
	"negative": func(v []Component) error {
		if len(v) < 1 || len(v) > 2 || !allSymbols(v) {
			return fmt.Errorf("expected one or two symbols")
		}
		return nil
	},
	"prefix":   checkOneSymbol,
	"suffix":   checkOneSymbol,
	"range":    checkRange,
	"pad":      checkPad,
	"fallback": checkStyleName,
	"speak-as": func(v []Component) error {
		if len(v) != 1 || v[0].Kind != Ident {
			return fmt.Errorf("expected auto, bullets, numbers, words, spell-out or a counter style's name")
		}
		return nil
	},
}

// counterStyle reads an @counter-style rule: a name, then a block of
// descriptors. A descriptor that doesn't check is dropped, as Chrome
// drops it.
func (p *parser) counterStyle(t Token, in scope) {
	prelude, block := p.prelude()
	if !block {
		p.warnf(t.Pos, "@counter-style has no { block }")
		return
	}
	name := items(prelude)
	if len(name) != 1 || checkStyleName(name) != nil {
		p.warnf(t.Pos, "@counter-style: %q is no counter style name; the rule is dropped", strings.TrimSpace(join(prelude)))
		p.skipBlock()
		return
	}
	cs := CounterStyle{Name: styleName(name[0].Text), Pos: t.Pos, Descs: map[string][]Component{}, media: in.media}
	if slices.Contains(fixedStyles, cs.Name) && p.sheet.Name != predefinedSheet {
		p.warnf(t.Pos, "@counter-style %s: %s can't be redefined; the rule is dropped", cs.Name, cs.Name)
		p.skipBlock()
		return
	}
	var body []Component
	for k := p.peek().Kind; k != EOF && k != RBrace; k = p.peek().Kind {
		body = append(body, p.component())
	}
	p.next()
	for _, part := range splitOn(body, func(c Component) bool { return c.Kind == Semicolon }) {
		part = trim(part)
		if len(part) == 0 {
			continue
		}
		v := trim(part[1:])
		if part[0].Kind != Ident || len(v) < 2 || v[0].Kind != Colon {
			p.warnf(part[0].Pos, "expected a descriptor in @counter-style")
			continue
		}
		desc := strings.ToLower(part[0].Text)
		check, ok := counterStyleDescriptors[desc]
		if !ok {
			p.warnf(part[0].Pos, "@counter-style: %s is no descriptor", part[0].Text)
			continue
		}
		v = items(trim(v[1:]))
		if err := check(v); err != nil {
			p.warnf(part[0].Pos, "@counter-style %s: %s: %v", cs.Name, desc, err)
			continue
		}
		cs.Descs[desc] = v
	}
	p.sheet.CounterStyles = append(p.sheet.CounterStyles, cs)
}

// String is the rule as CSS.
func (cs *CounterStyle) String() string {
	var b strings.Builder
	b.WriteString("@counter-style " + cs.Name + " {")
	names := make([]string, 0, len(cs.Descs))
	for name := range cs.Descs {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		parts := make([]string, len(cs.Descs[name]))
		for i, c := range cs.Descs[name] {
			parts[i] = c.String()
		}
		b.WriteString(" " + name + ": " + strings.ReplaceAll(strings.Join(parts, " "), " ,", ",") + ";")
	}
	b.WriteString(" }")
	return b.String()
}

// styleName is a counter style's name as written: those CSS predefines
// are matched whatever their case.
func styleName(s string) string {
	if l := strings.ToLower(s); predefinedNames()[l] || l == kwNone {
		return l
	}
	return s
}

func checkStyleName(v []Component) error {
	if len(v) != 1 || v[0].Kind != Ident || isWord(v[0], kwNone, "initial", "inherit", "unset", "default", "revert",
		"revert-layer") {
		return fmt.Errorf("expected a counter style's name")
	}
	return nil
}

func checkSystem(v []Component) error {
	switch {
	case len(v) == 1 && isWord(v[0], "cyclic", "numeric", "alphabetic", "symbolic", "additive", kwFixed):
		return nil
	case len(v) == 2 && isWord(v[0], kwFixed) && v[1].Kind == Number && v[1].Int:
		return nil
	case len(v) == 2 && isWord(v[0], "extends") && checkStyleName(v[1:]) == nil:
		return nil
	}
	return fmt.Errorf("expected cyclic, numeric, alphabetic, symbolic, additive, fixed or extends a name")
}

func isSymbol(c Component) bool { return c.Kind == String || c.Kind == Ident }

func allSymbols(v []Component) bool {
	return !slices.ContainsFunc(v, func(c Component) bool { return !isSymbol(c) })
}

func checkSymbols(v []Component) error {
	if len(v) == 0 || !allSymbols(v) {
		return fmt.Errorf("expected strings or names; ui takes no picture for a symbol")
	}
	return nil
}

func checkOneSymbol(v []Component) error {
	if len(v) != 1 || !isSymbol(v[0]) {
		return fmt.Errorf("expected a string or a name")
	}
	return nil
}

func checkAdditiveSymbols(v []Component) error {
	last := math.MaxInt
	for _, part := range splitOn(v, func(c Component) bool { return c.Kind == Comma }) {
		w, _, ok := weighted(part)
		if !ok {
			return fmt.Errorf("expected a whole number at least 0 and a symbol, found %q", join(part))
		}
		if w >= last {
			return fmt.Errorf("the weights must go down")
		}
		last = w
	}
	return nil
}

// weighted reads a whole number at least 0 and a symbol, in either order,
// as additive-symbols and pad take them.
func weighted(part []Component) (int, string, bool) {
	if len(part) != 2 {
		return 0, "", false
	}
	n, s := part[0], part[1]
	if isSymbol(n) {
		n, s = s, n
	}
	if n.Kind != Number || !n.Int || n.Num < 0 || !isSymbol(s) {
		return 0, "", false
	}
	return int(n.Num), s.Text, true
}

func checkPad(v []Component) error {
	if _, _, ok := weighted(v); !ok {
		return fmt.Errorf("expected a whole number at least 0 and a symbol")
	}
	return nil
}

func checkRange(v []Component) error {
	if len(v) == 1 && isWord(v[0], kwAuto) {
		return nil
	}
	for _, part := range splitOn(v, func(c Component) bool { return c.Kind == Comma }) {
		r, ok := bounds(part)
		if !ok {
			return fmt.Errorf("expected auto, or pairs of whole numbers or infinite, found %q", join(part))
		}
		if r[0] > r[1] {
			return fmt.Errorf("a range's lower bound is above its upper one")
		}
	}
	return nil
}

func bounds(part []Component) ([2]int, bool) {
	if len(part) != 2 {
		return [2]int{}, false
	}
	var r [2]int
	for i, c := range part {
		switch {
		case isWord(c, "infinite"):
			r[i] = style.Infinite
			if i == 0 {
				r[i] = -style.Infinite
			}
		case c.Kind == Number && c.Int:
			r[i] = int(c.Num)
		default:
			return r, false
		}
	}
	return r, true
}

// counterStyles are the counter styles one set of sheets defines over
// those CSS predefines, each resolved once asked for.
type counterStyles struct {
	rules    map[string]*CounterStyle
	resolved map[string]*style.CounterStyle
	// up is the set the names this one doesn't define go to: the
	// predefined styles' for an author's set, nil for those.
	up *counterStyles
}

//nolint:gochecknoglobals // the predefined styles, read once
var predefinedOnce = sync.OnceValue(func() *counterStyles {
	s, _ := Parse(predefinedSheet, predefinedStyles)
	return newCounterStyles(s.CounterStyles, nil)
})

func predefined() *counterStyles { return predefinedOnce() }

//nolint:gochecknoglobals // the predefined styles' names, read once
var predefinedNames = sync.OnceValue(func() map[string]bool {
	names := map[string]bool{}
	for line := range strings.SplitSeq(predefinedStyles, "\n") {
		if name, ok := strings.CutPrefix(line, "@counter-style "); ok {
			names[strings.Fields(name)[0]] = true
		}
	}
	return names
})

func newCounterStyles(rules []CounterStyle, up *counterStyles) *counterStyles {
	cs := &counterStyles{rules: map[string]*CounterStyle{}, resolved: map[string]*style.CounterStyle{}, up: up}
	for i := range rules {
		cs.rules[rules[i].Name] = &rules[i]
	}
	return cs
}

// counterStylesOn says, rule by rule, which @counter-style rules of
// sheets stand in @media rules that hold for m; nil when none stands in
// any.
func counterStylesOn(sheets []*Sheet, m Media) []bool {
	var on []bool
	for _, s := range sheets {
		for _, cs := range s.CounterStyles {
			if len(cs.media) > 0 {
				on = append(on, !slices.ContainsFunc(cs.media, func(i int) bool { return !s.media[i].matches(m) }))
			}
		}
	}
	return on
}

// authorStyles are the counter styles of sheets, the @counter-style rules
// whose @media rules hold for m, the last of a name winning.
func authorStyles(sheets []*Sheet, m Media) *counterStyles {
	var rules []CounterStyle
	for _, s := range sheets {
		for _, cs := range s.CounterStyles {
			if !slices.ContainsFunc(cs.media, func(i int) bool { return !s.media[i].matches(m) }) {
				rules = append(rules, cs)
			}
		}
	}
	if len(rules) == 0 {
		return predefined()
	}
	return newCounterStyles(rules, predefined())
}

// get is the counter style name stands for: decimal when no rule
// defines it, or the rule doesn't define one.
func (cs *counterStyles) get(name string) *style.CounterStyle {
	if s := cs.lookup(name, nil); s != nil {
		return s
	}
	return predefined().lookup(kwDecimal, nil)
}

// lookup resolves name, nil when nothing defines it; extending are the
// names being resolved that extend it, whose loop makes each extend
// decimal.
func (cs *counterStyles) lookup(name string, extending []string) *style.CounterStyle {
	if s, ok := cs.resolved[name]; ok {
		return s
	}
	r := cs.rules[name]
	if r == nil {
		if cs.up != nil {
			return cs.up.lookup(name, extending)
		}
		return nil
	}
	if slices.Contains(extending, name) {
		return nil
	}
	s := cs.resolve(r, append(extending, name))
	cs.resolved[name] = s
	return s
}

// resolve works r out: its descriptors over those of the style it
// extends, or over CSS's initial ones. A rule whose system lacks the
// symbols it needs defines none.
func (cs *counterStyles) resolve(r *CounterStyle, extending []string) *style.CounterStyle {
	out := &style.CounterStyle{Negative: [2]string{"-"}, Suffix: ". ", Fallback: kwDecimal}
	system := r.Descs["system"]
	if len(system) == 2 && isWord(system[0], "extends") {
		if r.Descs["symbols"] != nil || r.Descs["additive-symbols"] != nil {
			return nil
		}
		base := cs.lookup(styleName(system[1].Text), extending)
		if base == nil {
			base = predefined().lookup(kwDecimal, nil)
		}
		*out = *base
	} else {
		out.System, out.First = systemOf(system)
		for _, c := range r.Descs["symbols"] {
			out.Symbols = append(out.Symbols, c.Text)
		}
		for _, part := range splitOn(r.Descs["additive-symbols"], func(c Component) bool { return c.Kind == Comma }) {
			w, s, _ := weighted(part)
			out.Additive = append(out.Additive, style.AdditiveSymbol{Weight: w, Symbol: s})
		}
		if !enoughSymbols(out) {
			return nil
		}
	}
	out.Name = r.Name
	cs.descriptors(r, out)
	return out
}

func (cs *counterStyles) descriptors(r *CounterStyle, out *style.CounterStyle) {
	if v := r.Descs["negative"]; v != nil {
		out.Negative = [2]string{v[0].Text}
		if len(v) == 2 {
			out.Negative[1] = v[1].Text
		}
	}
	if v := r.Descs["prefix"]; v != nil {
		out.Prefix = v[0].Text
	}
	if v := r.Descs["suffix"]; v != nil {
		out.Suffix = v[0].Text
	}
	if v := r.Descs["range"]; v != nil {
		out.Ranges = nil
		if !isWord(v[0], kwAuto) {
			for _, part := range splitOn(v, func(c Component) bool { return c.Kind == Comma }) {
				b, _ := bounds(part)
				out.Ranges = append(out.Ranges, b)
			}
		}
	}
	if v := r.Descs["pad"]; v != nil {
		out.Pad, out.PadSymbol, _ = weighted(v)
	}
	if v := r.Descs["fallback"]; v != nil {
		out.Fallback = styleName(v[0].Text)
	}
}

func systemOf(v []Component) (style.CounterSystem, int) {
	if len(v) == 0 {
		return style.CounterSymbolic, 0
	}
	switch strings.ToLower(v[0].Text) {
	case "cyclic":
		return style.CounterCyclic, 0
	case "numeric":
		return style.CounterNumeric, 0
	case "alphabetic":
		return style.CounterAlphabetic, 0
	case "additive":
		return style.CounterAdditive, 0
	case kwFixed:
		if len(v) == 2 {
			return style.CounterFixed, int(v[1].Num)
		}
		return style.CounterFixed, 1
	}
	return style.CounterSymbolic, 0
}

func enoughSymbols(s *style.CounterStyle) bool {
	switch s.System {
	case style.CounterAlphabetic, style.CounterNumeric:
		return len(s.Symbols) >= 2
	case style.CounterAdditive:
		return len(s.Additive) > 0
	case style.CounterCyclic, style.CounterFixed, style.CounterSymbolic:
	}
	return len(s.Symbols) > 0
}

// represent writes n in the style name, or in its fallbacks where it
// can't, decimal last.
func (cs *counterStyles) represent(name string, n int) string {
	s := cs.get(name)
	for seen := []string{}; ; {
		if text, ok := s.Represent(n); ok {
			return text
		}
		seen = append(seen, s.Name)
		if slices.Contains(seen, s.Fallback) {
			s = predefined().lookup(kwDecimal, nil)
			continue
		}
		s = cs.get(s.Fallback)
	}
}

// marker is a list item's marker string for n in the style name: its
// representation between the style's prefix and suffix.
func (cs *counterStyles) marker(name string, n int) string {
	s := cs.get(name)
	return s.Prefix + cs.represent(name, n) + s.Suffix
}

// counterStylesWithin is keyframesWithin for @counter-style rules.
func counterStylesWithin(rules []CounterStyle, outer []int, base int) []CounterStyle {
	out := make([]CounterStyle, len(rules))
	for i, r := range rules {
		r.media = append(slices.Clone(outer), r.media...)
		for j := len(outer); j < len(r.media); j++ {
			r.media[j] += base
		}
		out[i] = r
	}
	return out
}
