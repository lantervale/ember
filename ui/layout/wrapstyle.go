package layout

import (
	"math"
	"unicode/utf8"

	"github.com/lantervale/ember/ui/internal/style"
)

// text-wrap-style: balance and pretty, as Chrome 157 breaks them (decision
// 0177): its greedy lines first, then, where they qualify, the break points
// of the least score over the lines of a paragraph, Minikin's optimal line
// breaker as Blink's ScoreLineBreaker ports it; a balance it can't score is
// bisected instead, ParagraphLineBreaker's.

const (
	// maxBalanced and maxPretty are the most lines a paragraph balances and
	// the lines pretty scores at its end.
	maxBalanced = 6
	maxPretty   = 4
	// overfull is the score of a line too wide for its room.
	overfull = float32(1e12)
	// orphanPenalty is added to the last break before a paragraph's end.
	orphanPenalty = float32(10000)
	// lastLineMultiplier weighs the penalty of a break ending the line
	// before a paragraph's last.
	lastLineMultiplier = float32(4)
	// layoutEpsilon is the layout unit a line may exceed its room by.
	layoutEpsilon = float32(1) / 64
)

// candidate is a place a paragraph's line may end: at is the piece the
// next line begins at, noBreak where that line starts and ifBreak where
// this one would end, along one line holding the whole paragraph.
type candidate struct {
	at               int
	noBreak, ifBreak float32
	penalty          float32
	hyphenated       bool
}

// styled is lines, broken greedily at width, broken again as the owner's
// text-wrap-style asks.
func (c *inlineContent) styled(width float32, lines []line) []line {
	st := c.owner.st
	if width >= unbounded || len(lines) < 2 || c.hasFirst() || c.owner.control != noControl {
		return lines
	}
	if n, _ := st.Clamp(); n != 0 {
		return lines
	}
	switch st.WrapStyle {
	case style.WrapBalance:
		return c.balance(width, lines)
	case style.WrapPretty:
		return c.pretty(width, lines)
	case style.WrapAuto, style.WrapStable:
	}
	return lines
}

// paragraphEnd is the index past the last line of the paragraph starting
// at line k: the one a forced break ends, or the block's last.
func paragraphEnd(lines []line, k int) int {
	for ; k < len(lines); k++ {
		if lines[k].forced {
			return k + 1
		}
	}
	return len(lines)
}

// overflows reports whether a line of lines is wider than its room: Chrome
// scores and bisects no paragraph its words overflow.
func overflows(lines []line) bool {
	for _, l := range lines {
		if l.width > l.avail+layoutEpsilon {
			return true
		}
	}
	return false
}

// balance balances the block's first paragraph by score when it has no
// more than maxBalanced lines, else, without a forced break, bisects the
// width its lines are broken at.
func (c *inlineContent) balance(width float32, lines []line) []line {
	end := paragraphEnd(lines, 0)
	if end <= maxBalanced && !overflows(lines[:end]) {
		if out, ok := c.optimize(width, lines, 0, end, true); ok {
			return out
		}
	}
	return c.bisect(width, lines)
}

// pretty scores the last maxPretty lines of each paragraph whose last line
// is short: under a third of its room, one word.
func (c *inlineContent) pretty(width float32, lines []line) []line {
	for k := 0; k < len(lines); {
		end := paragraphEnd(lines, k)
		from := max(k, end-maxPretty)
		k = end
		if end-from < 2 || overflows(lines[from:end]) {
			continue
		}
		last := lines[end-1]
		short := last.width < last.avail/3
		if short {
			cands, _ := c.candidates(lines, from, end)
			short = !inside(cands, last)
		}
		hyphenated := end-from >= 3 && lines[end-2].hyphen >= 0 && lines[end-3].hyphen >= 0
		if !short && !hyphenated {
			continue
		}
		if out, ok := c.optimize(width, lines, from, end, false); ok {
			lines = out
		}
	}
	return lines
}

// inside reports whether a line may break inside l.
func inside(cands []candidate, l line) bool {
	for _, cd := range cands[1 : len(cands)-1] {
		if cd.at > l.from && cd.at < l.to {
			return true
		}
	}
	return false
}

// windowEnd is the piece the line after lines[end-1] begins at, all of
// them when it is the last.
func windowEnd(lines []line, end int, total int) int {
	if end < len(lines) {
		return lines[end].from
	}
	return total
}

// candidates walks lines [from, end) as one line and lists where it may
// break, between a sentinel at its start and one at its end.
func (c *inlineContent) candidates(lines []line, from, end int) ([]candidate, bool) {
	stop := windowEnd(lines, end, len(c.pieces))
	out := []candidate{{at: lines[from].from}}
	b := breaker{c: c, stop: stop}
	b.seen = func(i int) {
		at := b.at
		if b.pending >= 0 {
			for k := b.pending; k < i; k++ {
				if kind := c.pieces[k].kind; kind == gap || kind == tab {
					at.step(c.pieces[k], 0)
				}
			}
		}
		hy := c.hyphenAt(i)
		out = append(out, candidate{at: i, noBreak: at.at(), ifBreak: b.kept.end() + hy, hyphenated: hy > 0 ||
			c.hyphenEnd(i) >= 0})
	}
	b.open(lines[from].from, 0, unbounded, 0)
	i, ended := b.one(lines[from].from, unbounded)
	w := b.kept.end()
	if ended {
		w = b.out[len(b.out)-1].width
	}
	if ended && i != stop && !lines[end-1].forced {
		return nil, false
	}
	return append(out, candidate{at: stop, noBreak: w, ifBreak: w}), true
}

// optimize breaks lines [from, end) again at the break points of the least
// score, false where Chrome would keep the greedy ones: too few places to
// break, or another number of lines.
func (c *inlineContent) optimize(width float32, lines []line, from, end int, balanced bool) ([]line, bool) {
	cands, ok := c.candidates(lines, from, end)
	if !ok || len(cands) < 5 {
		return nil, false
	}
	st := c.owner.st
	justified := st.TextAlign == style.Justify
	hyphen := width * st.Font.Size * 2
	if justified {
		hyphen = width * st.Font.Size / 2
	}
	for k := range cands {
		if cands[k].hyphenated {
			cands[k].penalty = hyphen
		}
	}
	for k := len(cands) - 2; k >= 0; k-- {
		cands[k].penalty += orphanPenalty
		if !cands[k].hyphenated {
			break
		}
	}
	perLine := hyphen * 2
	if justified {
		perLine = 0
	}
	indent := float32(0)
	if from == 0 {
		indent = c.indent(width)
	}
	fit := func(k int) float32 {
		w := width
		if k == 0 {
			w -= indent
		}
		return max(w, 0) + layoutEpsilon
	}
	points := scoreBreaks(cands, fit, balanced, justified, perLine)
	if len(points) != end-from {
		return nil, false
	}
	stops := make([]int, len(points))
	for k, p := range points {
		stops[k] = cands[p].at
	}
	redone, ok := c.rebreak(width, lines, from, end, stops)
	if !ok {
		return nil, false
	}
	out := append(append(append([]line{}, lines[:from]...), redone...), lines[end:]...)
	return out, true
}

// rebreak breaks lines [from, end) again, each ending at its stop.
func (c *inlineContent) rebreak(width float32, lines []line, from, end int, stops []int) ([]line, bool) {
	b := breaker{c: c}
	i := lines[from].from
	if from == 0 {
		indent := c.indent(width)
		b.open(i, c.startInset(indent), width-indent, 0)
		b.indent(indent)
	} else {
		b.open(i, 0, width, 0)
	}
	for k, stop := range stops {
		b.stop = stop
		var ended bool
		i, ended = b.one(i, unbounded)
		if !ended {
			if k != len(stops)-1 {
				return nil, false
			}
			b.cur.width = b.kept.end()
			b.out = append(b.out, b.cur)
			break
		}
		if i != stop && !(k == len(stops)-1 && b.out[len(b.out)-1].forced) {
			return nil, false
		}
		b.open(i, 0, width, 0)
	}
	if len(b.out) != end-from {
		return nil, false
	}
	for k := range b.out {
		b.out[k].first = lines[from+k].first
	}
	return b.out, true
}

// scoreBreaks is the candidates ending each line in the break of least
// score, Blink's ScoreLineBreaker::ComputeScores and ComputeBreakPoints:
// each line scores its slack squared (none for pretty's last), a break its
// penalty and a line linePenalty. fit is the room of the line of an index.
func scoreBreaks(cands []candidate, fit func(int) float32, balanced, justified bool, linePenalty float32) []int {
	type score struct {
		score      float32
		prev, line int
	}
	scores := make([]score, 1, len(cands))
	active := 0
	for end := 1; end < len(cands); end++ {
		ec := cands[end]
		last := end == len(cands)-1
		best, bestPrev := float32(math.MaxFloat32), 0
		lastLine := scores[active].line
		room := fit(lastLine)
		startEdge := ec.ifBreak - room
		hope := float32(0)
		for start := active; start < end; start++ {
			ss := scores[start]
			if ss.line != lastLine {
				lastLine = ss.line
				if r := fit(lastLine); r != room {
					room, startEdge, hope = r, ec.ifBreak-r, 0
				}
			}
			if ss.score+hope >= best {
				continue
			}
			sc := cands[start]
			delta := sc.noBreak - startEdge
			var width, extra float32
			switch {
			case (last || !justified) && delta < 0:
				width = overfull
			case last && !balanced:
				extra = lastLineMultiplier * sc.penalty
			case delta < 0:
				width = overfull
			default:
				width = delta * delta
			}
			if delta < 0 {
				active = start + 1
			} else {
				hope = width
			}
			if s := ss.score + width + extra; s <= best {
				best, bestPrev = s, start
			}
		}
		scores = append(scores, score{best + ec.penalty + linePenalty, bestPrev, scores[bestPrev].line + 1})
	}
	var points []int
	for i := len(scores) - 1; i > 0; i = scores[i].prev {
		points = append(points, i)
	}
	for l, r := 0, len(points)-1; l < r; l, r = l+1, r-1 {
		points[l], points[r] = points[r], points[l]
	}
	return points
}

// bisect is lines broken at the narrowest width, to a pixel, that breaks
// them into no more lines than width does, their room width's: Blink's
// ParagraphLineBreaker for a block of one paragraph of up to maxBalanced
// lines.
func (c *inlineContent) bisect(width float32, lines []line) []line {
	if len(lines) > maxBalanced || paragraphEnd(lines, 0) != len(lines) || overflows(lines) {
		return lines
	}
	if c.space > 0 {
		chars := int(width / c.space)
		if chars <= 0 || (c.textLength()+chars-1)/chars > maxBalanced*2 {
			return lines
		}
	}
	var sum float32
	for _, l := range lines {
		sum += snapUp(l.width, c.unit)
	}
	unit := c.unit
	lower := float32(math.Round(float64(sum/float32(len(lines))*0.8*unit))) / unit
	upper := width
	for lower+1 < upper {
		middle := float32(math.Floor(float64((upper+lower)*unit/2))) / unit
		b := breaker{c: c}
		if ls := b.breakAll(middle); len(ls) <= len(lines) && !overflows(ls) {
			upper = middle
		} else {
			lower = middle
		}
	}
	if upper >= width {
		return lines
	}
	b := breaker{c: c}
	out := b.breakAll(upper)
	for k := range out {
		out[k].avail += width - upper
	}
	return out
}

// textLength is how many characters the content's text holds, an atomic
// inline one.
func (c *inlineContent) textLength() int {
	n := 0
	for _, r := range c.runs {
		if r.atomic != nil {
			n++
			continue
		}
		n += utf8.RuneCountInString(collapsedBlanks(r.text, r.st.WhiteSpace))
	}
	return n
}
