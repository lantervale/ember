// Package bidi resolves the embedding levels of a paragraph by the Unicode
// Bidirectional Algorithm (UAX #9), its rules P2 to I2, and orders a line by
// rule L2. The classes of the characters are golang.org/x/text's, whose
// own package leaves the levels out of its API.
package bidi

import (
	"slices"

	"golang.org/x/text/unicode/bidi"
)

// Level is an embedding level: even left to right, odd right to left.
type Level = int8

// Auto asks Levels to find the paragraph's level from its first strong
// character, as P2 and P3 do.
const Auto Level = -1

// maxDepth is the deepest level an embedding or an isolate may open.
const maxDepth = 125

// Paragraph is one paragraph's characters, classes and resolved levels.
type Paragraph struct {
	text []rune
	// initial are the characters' classes, types those the rules resolve.
	initial, types []bidi.Class
	levels         []Level
	base           Level
	// pdi is the matching PDI of each isolate initiator, -1 for none;
	// opener the initiator each matched PDI closes.
	pdi, opener []int
}

// Class is r's bidirectional class.
func Class(r rune) bidi.Class {
	p, _ := bidi.LookupRune(r)
	return p.Class()
}

// Levels resolves the levels of text, one paragraph, at base: 0 or 1, or
// Auto.
func Levels(text []rune, base Level) *Paragraph {
	p := &Paragraph{text: text, initial: make([]bidi.Class, len(text)), levels: make([]Level, len(text))}
	for i, r := range text {
		p.initial[i] = Class(r)
	}
	p.types = slices.Clone(p.initial)
	p.matchIsolates()
	if base == Auto {
		base = p.firstStrong(0, len(text), 0)
	}
	p.base = base
	p.explicit()
	for _, seq := range p.sequences() {
		seq.weak()
		seq.brackets()
		seq.neutral()
		seq.implicit()
	}
	p.removedLevels()
	p.separators()
	return p
}

// separators gives the paragraph's level to its segment and paragraph
// separators and to the white space before them (L1's first clauses: the
// one at a line's end is the caller's).
func (p *Paragraph) separators() {
	for i, c := range p.initial {
		if c != bidi.B && c != bidi.S {
			continue
		}
		p.levels[i] = p.base
		for j := i - 1; j >= 0 && blank(p.initial[j]); j-- {
			p.levels[j] = p.base
		}
	}
}

// blank reports whether L1 reads c as white space: a space, an isolate
// control or what X9 removes.
func blank(c bidi.Class) bool {
	//nolint:exhaustive // the classes this rule reads; the others pass
	switch c {
	case bidi.WS, bidi.LRI, bidi.RLI, bidi.FSI, bidi.PDI, bidi.BN, bidi.RLE, bidi.LRE, bidi.RLO, bidi.LRO,
		bidi.PDF:
		return true
	default:
		return false
	}
}

// Base is the paragraph's level.
func (p *Paragraph) Base() Level { return p.base }

// Level is the level of character i, before the line rules.
func (p *Paragraph) Level(i int) Level { return p.levels[i] }

// Initial is the class character i has in the text.
func (p *Paragraph) Initial(i int) bidi.Class { return p.initial[i] }

// matchIsolates pairs each isolate initiator with its PDI (BD9).
func (p *Paragraph) matchIsolates() {
	p.pdi, p.opener = make([]int, len(p.text)), make([]int, len(p.text))
	var open []int
	for i, c := range p.initial {
		p.pdi[i], p.opener[i] = -1, -1
		//nolint:exhaustive // the classes this rule reads; the others pass
		switch c {
		case bidi.LRI, bidi.RLI, bidi.FSI:
			open = append(open, i)
		case bidi.PDI:
			if n := len(open); n > 0 {
				p.pdi[open[n-1]], p.opener[i] = i, open[n-1]
				open = open[:n-1]
			}
		default:
		}
	}
}

// firstStrong is the level the first strong character of [from, to) asks
// for, past isolates, def without one (P2, P3).
func (p *Paragraph) firstStrong(from, to int, def Level) Level {
	for i := from; i < to; i++ {
		//nolint:exhaustive // the classes this rule reads; the others pass
		switch p.initial[i] {
		case bidi.L:
			return 0
		case bidi.R, bidi.AL:
			return 1
		case bidi.LRI, bidi.RLI, bidi.FSI:
			if p.pdi[i] < 0 {
				return def
			}
			i = p.pdi[i]
		default:
		}
	}
	return def
}

// entry is a level of the directional status stack.
type entry struct {
	level    Level
	override bidi.Class
	isolate  bool
}

// status is the directional status stack and its counters (X1).
type status struct {
	stack                              []entry
	overIsolates, overEmbeds, isolates int
}

func (st *status) top() entry { return st.stack[len(st.stack)-1] }

// next is the least level above the top's of the parity rtl asks.
func (st *status) next(rtl bool) Level {
	l := st.top().level + 1
	if rtl != (l%2 == 1) {
		l++
	}
	return l
}

// embed opens an embedding or an override (X2 to X5).
func (st *status) embed(c bidi.Class) {
	l := st.next(c == bidi.RLE || c == bidi.RLO)
	if l > maxDepth || st.overIsolates > 0 || st.overEmbeds > 0 {
		if st.overIsolates == 0 {
			st.overEmbeds++
		}
		return
	}
	e := entry{level: l, override: bidi.ON}
	//nolint:exhaustive // the overrides alone
	switch c {
	case bidi.RLO:
		e.override = bidi.R
	case bidi.LRO:
		e.override = bidi.L
	}
	st.stack = append(st.stack, e)
}

// isolate opens an isolate, right to left when rtl (X5a to X5c).
func (st *status) isolate(rtl bool) {
	if l := st.next(rtl); l <= maxDepth && st.overIsolates == 0 && st.overEmbeds == 0 {
		st.isolates++
		st.stack = append(st.stack, entry{level: l, override: bidi.ON, isolate: true})
		return
	}
	st.overIsolates++
}

// closeIsolate closes the isolate a PDI ends (X6a).
func (st *status) closeIsolate() {
	switch {
	case st.overIsolates > 0:
		st.overIsolates--
	case st.isolates > 0:
		st.overEmbeds = 0
		for !st.top().isolate {
			st.stack = st.stack[:len(st.stack)-1]
		}
		st.stack = st.stack[:len(st.stack)-1]
		st.isolates--
	}
}

// pop closes the embedding a PDF ends (X7).
func (st *status) pop() {
	switch {
	case st.overIsolates > 0:
	case st.overEmbeds > 0:
		st.overEmbeds--
	case !st.top().isolate && len(st.stack) >= 2:
		st.stack = st.stack[:len(st.stack)-1]
	}
}

// explicit sets every character's explicit level and applies the
// overrides (X1 to X8).
func (p *Paragraph) explicit() {
	st := status{stack: []entry{{level: p.base, override: bidi.ON}}}
	take := func(i int) {
		top := st.top()
		p.levels[i] = top.level
		if top.override != bidi.ON {
			p.types[i] = top.override
		}
	}
	for i, c := range p.initial {
		//nolint:exhaustive // the classes this rule reads; the others pass
		switch c {
		case bidi.RLE, bidi.LRE, bidi.RLO, bidi.LRO:
			p.levels[i] = st.top().level
			st.embed(c)
		case bidi.RLI, bidi.LRI, bidi.FSI:
			take(i)
			st.isolate(p.isolatesRTL(i))
		case bidi.PDI:
			st.closeIsolate()
			take(i)
		case bidi.PDF:
			p.levels[i] = st.top().level
			st.pop()
		case bidi.B:
			p.levels[i] = p.base
		case bidi.BN:
			p.levels[i] = st.top().level
		default:
			take(i)
		}
	}
}

// isolatesRTL reports whether the isolate initiator i opens a right-to-left
// isolate: an FSI as its text's first strong character says.
func (p *Paragraph) isolatesRTL(i int) bool {
	if p.initial[i] != bidi.FSI {
		return p.initial[i] == bidi.RLI
	}
	end := p.pdi[i]
	if end < 0 {
		end = len(p.text)
	}
	return p.firstStrong(i+1, end, 0) == 1
}

// removed reports whether X9 takes character i out of the rules.
func (p *Paragraph) removed(i int) bool {
	//nolint:exhaustive // the classes this rule reads; the others pass
	switch p.initial[i] {
	case bidi.RLE, bidi.LRE, bidi.RLO, bidi.LRO, bidi.PDF, bidi.BN:
		return true
	default:
		return false
	}
}

// removedLevels gives the characters X9 took out the level of the one
// before them, the paragraph's at its start.
func (p *Paragraph) removedLevels() {
	for i := range p.levels {
		if !p.removed(i) {
			continue
		}
		p.levels[i] = p.base
		if i > 0 {
			p.levels[i] = p.levels[i-1]
		}
	}
}

// sequence is an isolating run sequence (BD13): the characters the weak,
// neutral and implicit rules read as one, and the classes before and
// after it.
type sequence struct {
	p        *Paragraph
	at       []int
	level    Level
	sos, eos bidi.Class
}

func direction(l Level) bidi.Class {
	if l%2 == 1 {
		return bidi.R
	}
	return bidi.L
}

// sequences cuts the paragraph into its isolating run sequences (X10).
func (p *Paragraph) sequences() []*sequence {
	var runs [][]int
	var cur []int
	for i := range p.text {
		if p.removed(i) {
			continue
		}
		if len(cur) > 0 && p.levels[cur[len(cur)-1]] != p.levels[i] {
			runs = append(runs, cur)
			cur = nil
		}
		cur = append(cur, i)
	}
	if len(cur) > 0 {
		runs = append(runs, cur)
	}
	startOf := map[int]int{}
	for k, r := range runs {
		startOf[r[0]] = k
	}
	var out []*sequence
	for _, r := range runs {
		if p.initial[r[0]] == bidi.PDI && p.opener[r[0]] >= 0 {
			continue
		}
		at := slices.Clone(r)
		for {
			last := at[len(at)-1]
			c := p.initial[last]
			if c != bidi.LRI && c != bidi.RLI && c != bidi.FSI || p.pdi[last] < 0 {
				break
			}
			k, ok := startOf[p.pdi[last]]
			if !ok {
				break
			}
			at = append(at, runs[k]...)
		}
		out = append(out, p.sequence(at))
	}
	return out
}

func (p *Paragraph) sequence(at []int) *sequence {
	level := p.levels[at[0]]
	before := p.base
	for i := at[0] - 1; i >= 0; i-- {
		if !p.removed(i) {
			before = p.levels[i]
			break
		}
	}
	after := p.base
	last := at[len(at)-1]
	if c := p.initial[last]; c != bidi.LRI && c != bidi.RLI && c != bidi.FSI {
		for i := last + 1; i < len(p.text); i++ {
			if !p.removed(i) {
				after = p.levels[i]
				break
			}
		}
	}
	return &sequence{p: p, at: at, level: level, sos: direction(max(level, before)),
		eos: direction(max(level, after))}
}

func (s *sequence) t(k int) bidi.Class { return s.p.types[s.at[k]] }

func (s *sequence) set(k int, c bidi.Class) { s.p.types[s.at[k]] = c }

func isolating(c bidi.Class) bool {
	return c == bidi.LRI || c == bidi.RLI || c == bidi.FSI || c == bidi.PDI
}

// weak resolves the weak types (W1 to W7).
func (s *sequence) weak() {
	n := len(s.at)
	prev := s.sos
	for k := range n {
		switch c := s.t(k); {
		case c == bidi.NSM && isolating(prev):
			s.set(k, bidi.ON)
		case c == bidi.NSM:
			s.set(k, prev)
		}
		prev = s.t(k)
	}
	s.numbersAfter(bidi.AL, bidi.AN, bidi.L, bidi.R, bidi.AL)
	for k := range n {
		if s.t(k) == bidi.AL {
			s.set(k, bidi.R)
		}
	}
	s.separators()
	for k := range n {
		if c := s.t(k); c == bidi.ES || c == bidi.ET || c == bidi.CS {
			s.set(k, bidi.ON)
		}
	}
	s.numbersAfter(bidi.L, bidi.L, bidi.L, bidi.R)
}

// numbersAfter gives the European numbers whose last strong type before
// them, among strong, is after the type to (W2, W7).
func (s *sequence) numbersAfter(after, to bidi.Class, strong ...bidi.Class) {
	last := s.sos
	for k := range s.at {
		switch c := s.t(k); {
		case c == bidi.EN && last == after:
			s.set(k, to)
		case slices.Contains(strong, c):
			last = c
		}
	}
}

// separators joins numbers across a single separator and takes the
// terminators beside European numbers into them (W4, W5).
func (s *sequence) separators() {
	n := len(s.at)
	for k := 1; k+1 < n; k++ {
		before, c, after := s.t(k-1), s.t(k), s.t(k+1)
		switch {
		case (c == bidi.ES || c == bidi.CS) && before == bidi.EN && after == bidi.EN:
			s.set(k, bidi.EN)
		case c == bidi.CS && before == bidi.AN && after == bidi.AN:
			s.set(k, bidi.AN)
		}
	}
	for k := 0; k < n; k++ {
		if s.t(k) != bidi.ET {
			continue
		}
		end := k
		for end < n && s.t(end) == bidi.ET {
			end++
		}
		if k > 0 && s.t(k-1) == bidi.EN || end < n && s.t(end) == bidi.EN {
			for j := k; j < end; j++ {
				s.set(j, bidi.EN)
			}
		}
		k = end - 1
	}
}

// neutralish reports whether c is a neutral or an isolate formatting
// character, as N1 and N2 read them.
func neutralish(c bidi.Class) bool {
	//nolint:exhaustive // the classes this rule reads; the others pass
	switch c {
	case bidi.B, bidi.S, bidi.WS, bidi.ON, bidi.BN, bidi.LRI, bidi.RLI, bidi.FSI, bidi.PDI:
		return true
	default:
		return false
	}
}

// strongOf is c as N1 reads it: numbers as R.
func strongOf(c bidi.Class) bidi.Class {
	if c == bidi.EN || c == bidi.AN {
		return bidi.R
	}
	return c
}

// neutral resolves the neutrals (N1, N2).
func (s *sequence) neutral() {
	n := len(s.at)
	e := direction(s.level)
	for k := 0; k < n; k++ {
		if !neutralish(s.t(k)) {
			continue
		}
		end := k
		for end < n && neutralish(s.t(end)) {
			end++
		}
		before, after := s.sos, s.eos
		if k > 0 {
			before = strongOf(s.t(k - 1))
		}
		if end < n {
			after = strongOf(s.t(end))
		}
		to := e
		if before == after {
			to = before
		}
		for j := k; j < end; j++ {
			s.set(j, to)
		}
		k = end - 1
	}
}

// implicit sets the levels from the resolved types (I1, I2).
func (s *sequence) implicit() {
	for k, i := range s.at {
		l := s.p.levels[i]
		switch c := s.t(k); {
		case l%2 == 0 && c == bidi.R:
			l++
		case l%2 == 0 && (c == bidi.AN || c == bidi.EN):
			l += 2
		case l%2 == 1 && (c == bidi.L || c == bidi.EN || c == bidi.AN):
			l++
		}
		s.p.levels[i] = l
	}
}

// Reorder is the visual order of a line of levels (L2): the index, in the
// line, of the item shown at each place from the left.
func Reorder(levels []Level) []int {
	out := make([]int, len(levels))
	for i := range out {
		out[i] = i
	}
	high, lowOdd := Level(0), Level(maxDepth+2)
	for _, l := range levels {
		high = max(high, l)
		if l%2 == 1 {
			lowOdd = min(lowOdd, l)
		}
	}
	for l := high; l >= lowOdd; l-- {
		for i := 0; i < len(levels); i++ {
			if levels[out[i]] < l {
				continue
			}
			end := i
			for end < len(levels) && levels[out[end]] >= l {
				end++
			}
			slices.Reverse(out[i:end])
			i = end
		}
	}
	return out
}
