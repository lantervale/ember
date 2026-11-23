package pattern

import "fmt"

// budget is how many steps a match takes at most: a pattern that
// backtracks past it on a value is taken to match it.
const budget = 1 << 20

// unset is a capture's bound before its group matched.
const unset = -1

func errorf(format string, args ...any) error { return fmt.Errorf("pattern: "+format, args...) }

// Match reports whether the pattern matches the whole of s, as HTML
// tests a value against ^(?:pattern)$. A match that has not been decided
// after a million steps of backtracking is taken to match.
func (re *Regexp) Match(s string) bool {
	m := &matcher{in: []rune(s), caps: make([]int, 2*(re.groups+1))}
	for i := range m.caps {
		m.caps[i] = unset
	}
	matched := re.root.match(m, 0, func(j int) bool { return j == len(m.in) })
	return matched || m.steps > budget
}

type matcher struct {
	in   []rune
	caps []int
	// back is set inside a lookbehind, which matches from right to left.
	back  bool
	steps int
}

// node is a part of a pattern: it matches from i, then asks k whether
// the rest matches from where it stopped.
type node interface {
	match(m *matcher, i int, k func(int) bool) bool
}

// next is the character a node consumes from i and where it leaves.
func (m *matcher) next(i int) (rune, int, bool) {
	if m.back {
		if i == 0 {
			return 0, i, false
		}
		return m.in[i-1], i - 1, true
	}
	if i == len(m.in) {
		return 0, i, false
	}
	return m.in[i], i + 1, true
}

func (m *matcher) spent() bool {
	m.steps++
	return m.steps > budget
}

type char struct {
	r    rune
	fold bool
}

func (c *char) match(m *matcher, i int, k func(int) bool) bool {
	r, j, ok := m.next(i)
	return ok && !m.spent() && same(r, c.r, c.fold) && k(j)
}

type dot struct{ all bool }

func (d *dot) match(m *matcher, i int, k func(int) bool) bool {
	r, j, ok := m.next(i)
	return ok && !m.spent() && (d.all || !isLineEnd(r)) && k(j)
}

type class struct {
	in    func(rune) bool
	strs  [][]rune
	empty bool
	fold  bool
}

func (c *class) match(m *matcher, i int, k func(int) bool) bool {
	if m.spent() {
		return false
	}
	for _, s := range c.strs {
		if j, ok := m.consume(i, s, c.fold); ok && k(j) {
			return true
		}
	}
	if r, j, ok := m.next(i); ok && c.in(r) && k(j) {
		return true
	}
	return c.empty && k(i)
}

// consume matches the runes s from i, in the matcher's direction.
func (m *matcher) consume(i int, s []rune, fold bool) (int, bool) {
	from, to := i, i+len(s)
	if m.back {
		from, to = i-len(s), i
	}
	if from < 0 || to > len(m.in) || !sameRunes(m.in[from:to], s, fold) {
		return i, false
	}
	if m.back {
		return from, true
	}
	return to, true
}

type seq struct{ terms []node }

func (s *seq) match(m *matcher, i int, k func(int) bool) bool {
	return s.from(m, 0, i, k)
}

func (s *seq) from(m *matcher, t, i int, k func(int) bool) bool {
	if t == len(s.terms) {
		return k(i)
	}
	n := s.terms[t]
	if m.back {
		n = s.terms[len(s.terms)-1-t]
	}
	return n.match(m, i, func(j int) bool { return s.from(m, t+1, j, k) })
}

type alt struct{ alts []node }

func (a *alt) match(m *matcher, i int, k func(int) bool) bool {
	for _, n := range a.alts {
		if m.spent() {
			return false
		}
		if n.match(m, i, k) {
			return true
		}
	}
	return false
}

type group struct {
	body  node
	index int
}

func (g *group) match(m *matcher, i int, k func(int) bool) bool {
	return g.body.match(m, i, func(j int) bool {
		at := 2 * g.index
		was0, was1 := m.caps[at], m.caps[at+1]
		m.caps[at], m.caps[at+1] = min(i, j), max(i, j)
		if k(j) {
			return true
		}
		m.caps[at], m.caps[at+1] = was0, was1
		return false
	})
}

type look struct {
	body        node
	neg, behind bool
}

func (l *look) match(m *matcher, i int, k func(int) bool) bool {
	saved := append([]int(nil), m.caps...)
	back := m.back
	m.back = l.behind
	found := l.body.match(m, i, func(int) bool { return true })
	m.back = back
	if found != l.neg && k(i) {
		return true
	}
	copy(m.caps, saved)
	return false
}

type assertKind uint8

const (
	lineStart assertKind = iota
	lineEnd
	wordBoundary
)

type assert struct {
	kind           assertKind
	multiline, neg bool
	word           func(rune) bool
}

func (a *assert) match(m *matcher, i int, k func(int) bool) bool {
	var ok bool
	switch a.kind {
	case lineStart:
		ok = i == 0 || a.multiline && isLineEnd(m.in[i-1])
	case lineEnd:
		ok = i == len(m.in) || a.multiline && isLineEnd(m.in[i])
	case wordBoundary:
		before := i > 0 && a.word(m.in[i-1])
		after := i < len(m.in) && a.word(m.in[i])
		ok = (before != after) != a.neg
	}
	return ok && k(i)
}

type backref struct {
	number int
	name   string
	groups []int
	fold   bool
}

func (b *backref) match(m *matcher, i int, k func(int) bool) bool {
	for _, g := range b.groups {
		if s, e := m.caps[2*g], m.caps[2*g+1]; s != unset {
			j, ok := m.consume(i, m.in[s:e], b.fold)
			return ok && k(j)
		}
	}
	return k(i)
}

// repeat is a quantified atom: from and to are the groups it holds,
// which each repetition clears.
type repeat struct {
	body     node
	min, max int
	greedy   bool
	from, to int
}

func (r *repeat) match(m *matcher, i int, k func(int) bool) bool {
	return r.times(m, r.min, r.max, i, k)
}

func (r *repeat) times(m *matcher, lo, hi, i int, k func(int) bool) bool {
	if m.spent() {
		return false
	}
	if hi == 0 {
		return k(i)
	}
	again := func(j int) bool {
		if lo == 0 && j == i {
			return false
		}
		nextHi := hi
		if hi > 0 {
			nextHi--
		}
		return r.times(m, max(lo-1, 0), nextHi, j, k)
	}
	try := func() bool {
		saved := append([]int(nil), m.caps[2*(r.from+1):2*(r.to+1)]...)
		for c := 2 * (r.from + 1); c < 2*(r.to+1); c++ {
			m.caps[c] = unset
		}
		if r.body.match(m, i, again) {
			return true
		}
		copy(m.caps[2*(r.from+1):], saved)
		return false
	}
	if lo > 0 {
		return try()
	}
	if !r.greedy {
		return k(i) || try()
	}
	return try() || k(i)
}
