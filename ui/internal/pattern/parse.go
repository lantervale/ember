// Package pattern reads a regular expression as JavaScript does with the
// v flag, the way HTML reads an input's pattern attribute, and matches a
// whole value against it: lookarounds, backreferences, named groups,
// flag modifiers, Unicode properties and the v flag's class set
// operations and strings, by backtracking over code points.
package pattern

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"unicode"
)

// ErrUnsupported is a pattern JavaScript reads that this package does
// not: a property of strings (\p{RGI_Emoji}), or a property or a script
// Go's unicode tables do not have.
var ErrUnsupported = errors.New("pattern: not supported")

// Regexp is a compiled pattern.
type Regexp struct {
	root   node
	groups int
}

// flags are the modifiers in force where a node is parsed.
type flags struct{ fold, multiline, dotAll bool }

type parser struct {
	src    []rune
	pos    int
	groups int
	names  map[string][]int
	// refs and named are the backreferences, by number and by name,
	// checked once every group is known.
	refs  []*backref
	named []*backref
}

// Compile reads src as JavaScript's RegExp(src, "v") does, or returns
// the SyntaxError it would throw, or ErrUnsupported.
func Compile(src string) (*Regexp, error) {
	p := &parser{src: []rune(src), names: map[string][]int{}}
	n, err := p.disjunction(flags{})
	if err != nil {
		return nil, err
	}
	if p.pos < len(p.src) {
		return nil, p.errorf("unmatched %q", p.src[p.pos])
	}
	for _, r := range p.refs {
		if r.number > p.groups {
			return nil, fmt.Errorf("pattern: no group %d", r.number)
		}
		r.groups = []int{r.number}
	}
	for _, r := range p.named {
		g, ok := p.names[r.name]
		if !ok {
			return nil, fmt.Errorf("pattern: no group named %q", r.name)
		}
		r.groups = g
	}
	return &Regexp{root: n, groups: p.groups}, nil
}

func (p *parser) errorf(format string, args ...any) error {
	return fmt.Errorf("pattern: at %d: "+format, append([]any{p.pos}, args...)...)
}

func (p *parser) more() bool { return p.pos < len(p.src) }

func (p *parser) peek() rune {
	if p.pos < len(p.src) {
		return p.src[p.pos]
	}
	return -1
}

// peekNext is the character after the next.
func (p *parser) peekNext() rune {
	if p.pos+1 < len(p.src) {
		return p.src[p.pos+1]
	}
	return -1
}

func (p *parser) eat(s string) bool {
	rs := []rune(s)
	if p.pos+len(rs) > len(p.src) || !slices.Equal(p.src[p.pos:p.pos+len(rs)], rs) {
		return false
	}
	p.pos += len(rs)
	return true
}

func (p *parser) disjunction(f flags) (node, error) {
	var alts []node
	for {
		a, err := p.alternative(f)
		if err != nil {
			return nil, err
		}
		alts = append(alts, a)
		if !p.eat("|") {
			break
		}
	}
	if len(alts) == 1 {
		return alts[0], nil
	}
	return &alt{alts: alts}, nil
}

func (p *parser) alternative(f flags) (node, error) {
	var terms []node
	for p.more() && p.peek() != '|' && p.peek() != ')' {
		t, err := p.term(f)
		if err != nil {
			return nil, err
		}
		terms = append(terms, t)
	}
	return &seq{terms: terms}, nil
}

func (p *parser) term(f flags) (node, error) {
	before := p.groups
	atom, quantifiable, err := p.atom(f)
	if err != nil {
		return nil, err
	}
	r, err := p.quantifier()
	if err != nil || r == nil {
		return atom, err
	}
	if !quantifiable {
		return nil, p.errorf("nothing to repeat")
	}
	r.body, r.from, r.to = atom, before, p.groups
	return r, nil
}

// quantifier reads a quantifier, if one follows: its bounds (max -1 for
// none) and whether it is greedy.
func (p *parser) quantifier() (*repeat, error) {
	r := &repeat{}
	switch p.peek() {
	case '*':
		p.pos++
		r.max = -1
	case '+':
		p.pos++
		r.min, r.max = 1, -1
	case '?':
		p.pos++
		r.max = 1
	case '{':
		p.pos++
		lo, ok := p.decimal()
		if !ok {
			return nil, p.errorf("lone {")
		}
		r.min, r.max = lo, lo
		if p.eat(",") {
			r.max = -1
			if hi, given := p.decimal(); given {
				r.max = hi
			}
		}
		if !p.eat("}") {
			return nil, p.errorf("unterminated {")
		}
		if r.max >= 0 && r.max < r.min {
			return nil, p.errorf("numbers out of order in {}")
		}
	default:
		return nil, nil //nolint:nilnil // no quantifier is no error
	}
	r.greedy = !p.eat("?")
	return r, nil
}

// decimal reads digits, capped far beyond any value's length.
func (p *parser) decimal() (int, bool) {
	start := p.pos
	v := 0
	for p.more() && p.peek() >= '0' && p.peek() <= '9' {
		v = min(v*10+int(p.peek()-'0'), 1<<30)
		p.pos++
	}
	return v, p.pos > start
}

// atom reads an atom or an assertion, and reports whether a quantifier
// may follow it.
func (p *parser) atom(f flags) (node, bool, error) {
	c := p.peek()
	switch c {
	case '^':
		p.pos++
		return &assert{kind: lineStart, multiline: f.multiline}, false, nil
	case '$':
		p.pos++
		return &assert{kind: lineEnd, multiline: f.multiline}, false, nil
	case '.':
		p.pos++
		return &dot{all: f.dotAll}, true, nil
	case '(':
		return p.group(f)
	case '[':
		p.pos++
		s, err := p.class(f)
		if err != nil {
			return nil, false, err
		}
		return s.node(), true, nil
	case '\\':
		return p.atomEscape(f)
	case '*', '+', '?', '{', '}', ']', ')', '|':
		return nil, false, p.errorf("unexpected %q", c)
	}
	p.pos++
	return &char{r: c, fold: f.fold}, true, nil
}

func (p *parser) group(f flags) (node, bool, error) {
	p.pos++
	switch {
	case p.eat("?="), p.eat("?!"):
		neg := p.src[p.pos-1] == '!'
		body, err := p.closeGroup(f)
		return &look{body: body, neg: neg}, false, err
	case p.eat("?<="), p.eat("?<!"):
		neg := p.src[p.pos-1] == '!'
		body, err := p.closeGroup(f)
		return &look{body: body, neg: neg, behind: true}, false, err
	case p.eat("?<"):
		name, err := p.groupName()
		if err != nil {
			return nil, false, err
		}
		p.groups++
		index := p.groups
		if len(p.names[name]) > 0 {
			return nil, false, p.errorf("a second group named %q", name)
		}
		p.names[name] = append(p.names[name], index)
		body, err := p.closeGroup(f)
		return &group{body: body, index: index}, true, err
	case p.eat("?"):
		g, err := p.modifiers(f)
		if err != nil {
			return nil, false, err
		}
		body, err := p.closeGroup(g)
		return body, true, err
	}
	p.groups++
	index := p.groups
	body, err := p.closeGroup(f)
	return &group{body: body, index: index}, true, err
}

func (p *parser) closeGroup(f flags) (node, error) {
	body, err := p.disjunction(f)
	if err != nil {
		return nil, err
	}
	if !p.eat(")") {
		return nil, p.errorf("unterminated group")
	}
	return body, nil
}

// modifiers reads what follows "(?": ":" alone, or flags to add, "-",
// flags to remove, then ":".
func (p *parser) modifiers(f flags) (flags, error) {
	if p.eat(":") {
		return f, nil
	}
	seen := map[rune]bool{}
	read := func(on bool) int {
		n := 0
		for p.more() && (p.peek() == 'i' || p.peek() == 'm' || p.peek() == 's') {
			r := p.peek()
			if seen[r] {
				return -1
			}
			seen[r] = true
			switch r {
			case 'i':
				f.fold = on
			case 'm':
				f.multiline = on
			case 's':
				f.dotAll = on
			}
			p.pos++
			n++
		}
		return n
	}
	add := read(true)
	remove := 0
	dash := p.eat("-")
	if dash {
		remove = read(false)
	}
	if add < 0 || remove < 0 || dash && add+remove == 0 || !dash && add == 0 || !p.eat(":") {
		return f, p.errorf("bad group")
	}
	return f, nil
}

func (p *parser) groupName() (string, error) {
	var name []rune
	for {
		if !p.more() {
			return "", p.errorf("unterminated group name")
		}
		r := p.peek()
		if r == '>' {
			p.pos++
			break
		}
		if r == '\\' {
			p.pos++
			if !p.eat("u") {
				return "", p.errorf("bad group name")
			}
			var err error
			if r, err = p.unicodeEscape(); err != nil {
				return "", err
			}
		} else {
			p.pos++
		}
		if !nameRune(r, len(name) == 0) {
			return "", p.errorf("bad group name")
		}
		name = append(name, r)
	}
	if len(name) == 0 {
		return "", p.errorf("empty group name")
	}
	return string(name), nil
}

const zwnj, zwj = 0x200c, 0x200d

// nameRune reports whether r may stand in a group name, first or not.
func nameRune(r rune, first bool) bool {
	if r == '$' || r == '_' || unicode.IsLetter(r) || unicode.Is(unicode.Nl, r) {
		return true
	}
	return !first && (unicode.IsDigit(r) || unicode.In(r, unicode.Mn, unicode.Mc, unicode.Pc) || r == zwnj || r == zwj)
}

func (p *parser) atomEscape(f flags) (node, bool, error) {
	p.pos++
	c := p.peek()
	switch {
	case c == 'b' || c == 'B':
		p.pos++
		return &assert{kind: wordBoundary, neg: c == 'B', word: leaf(isWord, f.fold).in}, false, nil
	case c >= '1' && c <= '9':
		v, _ := p.decimal()
		r := &backref{number: v, fold: f.fold}
		p.refs = append(p.refs, r)
		return r, true, nil
	case c == 'k':
		p.pos++
		if !p.eat("<") {
			return nil, false, p.errorf("bad \\k")
		}
		name, err := p.groupName()
		if err != nil {
			return nil, false, err
		}
		r := &backref{name: name, fold: f.fold}
		p.named = append(p.named, r)
		return r, true, nil
	}
	if s, ok, err := p.classEscape(f); ok || err != nil {
		if err != nil {
			return nil, false, err
		}
		return s.node(), true, nil
	}
	r, err := p.charEscape(false)
	if err != nil {
		return nil, false, err
	}
	return &char{r: r, fold: f.fold}, true, nil
}

// classEscape reads, after a backslash, \d, \D, \s, \S, \w, \W, \p{…}
// or \P{…}, if one is there.
func (p *parser) classEscape(f flags) (*set, bool, error) {
	c := p.peek()
	var s *set
	switch c {
	case 'd', 'D':
		s = leaf(isDigit, f.fold)
	case 's', 'S':
		s = leaf(isSpace, f.fold)
	case 'w', 'W':
		s = leaf(isWord, f.fold)
	case 'p', 'P':
		p.pos++
		if !p.eat("{") {
			return nil, true, p.errorf("bad \\%c", c)
		}
		end := slices.Index(p.src[p.pos:], '}')
		if end < 0 {
			return nil, true, p.errorf("unterminated \\%c{", c)
		}
		body := string(p.src[p.pos : p.pos+end])
		p.pos += end + 1
		in, err := property(body)
		if err != nil {
			return nil, true, err
		}
		s = leaf(in, f.fold)
		if c == 'P' {
			s = s.complement()
		}
		return s, true, nil
	default:
		return nil, false, nil
	}
	p.pos++
	if unicode.IsUpper(c) {
		s = s.complement()
	}
	return s, true, nil
}

// charEscape reads, after a backslash, an escaped character; inClass
// takes \b as a backspace and the v flag's class punctuators.
func (p *parser) charEscape(inClass bool) (rune, error) {
	if !p.more() {
		return 0, p.errorf("\\ at the end")
	}
	c := p.peek()
	p.pos++
	switch c {
	case 'f':
		return '\f', nil
	case 'n':
		return '\n', nil
	case 'r':
		return '\r', nil
	case 't':
		return '\t', nil
	case 'v':
		return '\v', nil
	case 'c':
		if l := p.peek(); l >= 'a' && l <= 'z' || l >= 'A' && l <= 'Z' {
			p.pos++
			return l % 32, nil
		}
		return 0, p.errorf("bad \\c")
	case '0':
		if d := p.peek(); d >= '0' && d <= '9' {
			return 0, p.errorf("bad \\0")
		}
		return 0, nil
	case 'x':
		if v, ok := p.hex(2); ok {
			return v, nil
		}
		return 0, p.errorf("bad \\x")
	case 'u':
		return p.unicodeEscape()
	case 'b':
		if inClass {
			return '\b', nil
		}
	case '^', '$', '\\', '.', '*', '+', '?', '(', ')', '[', ']', '{', '}', '|', '/':
		return c, nil
	}
	if inClass && slices.Contains([]rune("&-!#%,:;<=>@`~"), c) {
		return c, nil
	}
	return 0, p.errorf("bad escape \\%c", c)
}

// unicodeEscape reads what follows \u: four hex digits, a pair of them
// for a surrogate pair, or {hex}.
func (p *parser) unicodeEscape() (rune, error) {
	if p.eat("{") {
		end := slices.Index(p.src[p.pos:], '}')
		if end <= 0 {
			return 0, p.errorf("bad \\u{}")
		}
		v, err := strconv.ParseUint(string(p.src[p.pos:p.pos+end]), 16, 32)
		if err != nil || v > unicode.MaxRune {
			return 0, p.errorf("bad \\u{}")
		}
		p.pos += end + 1
		return rune(v), nil
	}
	hi, ok := p.hex(4)
	if !ok {
		return 0, p.errorf("bad \\u")
	}
	if hi >= 0xd800 && hi <= 0xdbff && p.peek() == '\\' && p.peekNext() == 'u' {
		save := p.pos
		p.pos += 2
		if lo, pair := p.hex(4); pair && lo >= 0xdc00 && lo <= 0xdfff {
			return (hi-0xd800)<<10 + (lo - 0xdc00) + 0x10000, nil
		}
		p.pos = save
	}
	return hi, nil
}

func (p *parser) hex(n int) (rune, bool) {
	if p.pos+n > len(p.src) {
		return 0, false
	}
	v, err := strconv.ParseInt(string(p.src[p.pos:p.pos+n]), 16, 32)
	if err != nil || v < 0 {
		return 0, false
	}
	p.pos += n
	return rune(v), true
}

// class reads a class after its "[", up to and with its "]".
func (p *parser) class(f flags) (*set, error) {
	negate := p.eat("^")
	s, err := p.classContents(f)
	if err != nil {
		return nil, err
	}
	if !p.eat("]") {
		return nil, p.errorf("unterminated class")
	}
	if negate {
		if len(s.strs) > 0 {
			return nil, p.errorf("a negated class holding strings")
		}
		s = s.complement()
	}
	return s, nil
}

func (p *parser) classContents(f flags) (*set, error) {
	if p.peek() == ']' {
		return &set{in: func(rune) bool { return false }, fold: f.fold}, nil
	}
	first, ranged, err := p.rangeOrOperand(f)
	if err != nil {
		return nil, err
	}
	if op := p.peek(); !ranged && (op == '&' || op == '-') && p.peekNext() == op {
		return p.operations(f, first, op)
	}
	acc := first
	for p.more() && p.peek() != ']' {
		next, _, bad := p.rangeOrOperand(f)
		if bad != nil {
			return nil, bad
		}
		acc = acc.union(next)
	}
	return acc, nil
}

// operations reads the rest of a class of intersections (op '&') or of
// subtractions ('-') from acc, its first operand.
func (p *parser) operations(f flags, acc *set, op rune) (*set, error) {
	for p.peek() == op && p.peekNext() == op {
		p.pos += 2
		if op == '&' && p.peek() == '&' {
			return nil, p.errorf("&&&")
		}
		next, err := p.operand(f)
		if err != nil {
			return nil, err
		}
		if op == '&' {
			acc = acc.intersect(next)
		} else {
			acc = acc.subtract(next)
		}
	}
	if p.peek() != ']' {
		return nil, p.errorf("mixed class operators")
	}
	return acc, nil
}

// rangeOrOperand reads a class operand, or a range of two characters,
// and reports which.
func (p *parser) rangeOrOperand(f flags) (*set, bool, error) {
	if p.peek() == '[' || p.peek() == '\\' && (p.peekNext() == 'q' || isClassEscape(p.peekNext())) {
		s, err := p.operand(f)
		return s, false, err
	}
	lo, err := p.classChar()
	if err != nil {
		return nil, false, err
	}
	if p.peek() == '-' && p.peekNext() != '-' {
		p.pos++
		hi, bad := p.classChar()
		if bad != nil {
			return nil, false, bad
		}
		if hi < lo {
			return nil, false, p.errorf("range out of order")
		}
		return leaf(func(r rune) bool { return r >= lo && r <= hi }, f.fold), true, nil
	}
	return single(lo, f.fold), false, nil
}

func isClassEscape(c rune) bool { return c >= 0 && slices.Contains([]rune("dDsSwWpP"), c) }

// operand reads a nested class, \q{…}, a class escape or a character.
func (p *parser) operand(f flags) (*set, error) {
	switch {
	case p.eat("["):
		return p.class(f)
	case p.peek() == '\\' && p.peekNext() == 'q':
		p.pos += 2
		if !p.eat("{") {
			return nil, p.errorf("bad \\q")
		}
		return p.strings(f)
	case p.peek() == '\\' && isClassEscape(p.peekNext()):
		p.pos++
		s, _, err := p.classEscape(f)
		return s, err
	}
	r, err := p.classChar()
	if err != nil {
		return nil, err
	}
	return single(r, f.fold), nil
}

// strings reads the strings of \q{…} after its "{".
func (p *parser) strings(f flags) (*set, error) {
	s := &set{in: func(rune) bool { return false }, fold: f.fold}
	var cur []rune
	for {
		if !p.more() {
			return nil, p.errorf("unterminated \\q{")
		}
		if end := p.peek(); end == '|' || end == '}' {
			p.pos++
			if len(cur) == 1 {
				s = s.union(single(cur[0], f.fold))
			} else {
				s = s.union(&set{in: func(rune) bool { return false }, strs: []string{string(cur)}, fold: f.fold})
			}
			if end == '}' {
				return s, nil
			}
			cur = nil
			continue
		}
		r, err := p.classChar()
		if err != nil {
			return nil, err
		}
		cur = append(cur, r)
	}
}

// classChar reads a character of a class as the v flag has it: no
// doubled punctuator, no class syntax character unescaped.
func (p *parser) classChar() (rune, error) {
	if !p.more() {
		return 0, p.errorf("unterminated class")
	}
	c := p.peek()
	if c == p.peekNext() && slices.Contains([]rune("&!#$%*+,.:;<=>?@^`~"), c) {
		return 0, p.errorf("doubled %q in a class", c)
	}
	switch c {
	case '\\':
		p.pos++
		return p.charEscape(true)
	case '(', ')', '[', ']', '{', '}', '/', '-', '|':
		return 0, p.errorf("unescaped %q in a class", c)
	}
	p.pos++
	return c, nil
}
