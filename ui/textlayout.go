package ui

import (
	"slices"
	"strings"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/control"
	"github.com/lantervale/ember/ui/internal/bidi"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

// line is a one-line text field's text laid out: where each position is
// along it. It measures the text as drawn and answers in the value's
// positions, through cases when text-transform changed the case.
type line struct {
	xs []float32
	// src, font, spacing and case are what xs was measured from, so a
	// frame where none changed keeps the line.
	src     []rune
	font    style.Font
	spacing spacing
	tt      style.TextTransform
	cases   *caseMap
}

// spacing is the room a style adds after each character and after each
// word separator.
type spacing struct{ letter, word float32 }

func spacingOf(st *style.Computed) spacing { return spacing{st.LetterSpacing, st.WordGap()} }

// after is the room s adds after r.
func (s spacing) after(r rune) float32 {
	if s.word != 0 && style.WordSeparator(r) {
		return s.letter + s.word
	}
	return s.letter
}

func newLine(text []rune, st *style.Computed, fonts box.Fonts) *line {
	l := &line{xs: make([]float32, len(text)+1), font: st.Font, spacing: spacingOf(st)}
	var room float32
	for i, r := range text {
		room += l.spacing.after(r)
		l.xs[i+1] = fonts.Advance(st.Font, string(text[:i+1])) + room
	}
	if st.Dir == style.RTL || slices.ContainsFunc(text, func(r rune) bool { return r >= firstRTL }) {
		l.xs = visualEdges(text, l.xs, st.Dir == style.RTL)
	}
	return l
}

// firstRTL is where the characters of a right-to-left script begin: text below it is
// left to right throughout.
const firstRTL = 0x0590

// visualEdges is where the caret stands for each place in text, laid out in a paragraph
// of direction rtl or not, from the line's left, xs being the edges of its characters
// in the text's order: before a character on the side its text comes from (the right
// of one shown right to left), at the end after the last one.
func visualEdges(text []rune, xs []float32, rtl bool) []float32 {
	var base bidi.Level
	if rtl {
		base = 1
	}
	p := bidi.Levels(text, base)
	levels := make([]bidi.Level, len(text))
	for i := range text {
		levels[i] = p.Level(i)
	}
	left := make([]float32, len(text))
	var x float32
	for _, k := range bidi.Reorder(levels) {
		left[k] = x
		x += xs[k+1] - xs[k]
	}
	out := make([]float32, len(text)+1)
	for i := range text {
		out[i] = left[i]
		if levels[i]%2 == 1 {
			out[i] += xs[i+1] - xs[i]
		}
	}
	switch n := len(text); {
	case n == 0:
	case levels[n-1]%2 == 1:
		out[n] = left[n-1]
	default:
		out[n] = left[n-1] + xs[n] - xs[n-1]
	}
	return out
}

func (l *line) Lines() int          { return 1 }
func (l *line) Line(int) int        { return 0 }
func (l *line) Span(int) (int, int) { return 0, l.cases.valueAt(len(l.xs) - 1) }
func (l *line) X(i int) float32     { return l.xs[min(max(l.cases.drawnAt(i), 0), len(l.xs)-1)] }
func (l *line) Page() int           { return 1 }
func (l *line) At(_ int, x float32) int {
	best := 0
	for i, at := range l.xs {
		if abs(at-x) < abs(l.xs[best]-x) {
			best = i
		}
	}
	return l.cases.valueAt(best)
}

func abs(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}

// layText gives each text field's machine the layout of what it shows,
// for its caret, its selection and the clicks placing them.
func (d *Document) layText() {
	for n, m := range d.machines {
		t := textOf(m)
		if t == nil {
			continue
		}
		s := dom.Data(n).Style
		if s == nil {
			continue
		}
		src := t.Editor.Runes()
		if t.Multiline() {
			d.layArea(n, t, s)
			continue
		}
		if l, ok := t.Layout.(*line); ok && slices.Equal(l.src, src) && sameFont(l.font, s.Font) &&
			l.spacing == spacingOf(s) && l.tt == s.TextTransform {
			continue
		}
		shown, cases := d.shownText(n, t)
		if !t.Empty() {
			n.SetShadow(shown, false)
		}
		l := newLine([]rune(shown), s, d.fonts)
		l.src, l.tt, l.cases = slices.Clone(src), s.TextTransform, cases
		t.Layout = l
	}
}

// layArea cuts a textarea's text along the lines layout gave it, once
// its text, font, spacing, width or case changed; lines laid out from
// other text than it draws wait for the next layout.
func (d *Document) layArea(n *Node, t *control.Text, st *style.Computed) {
	f := st.Font
	if d.tree == nil {
		return
	}
	b := d.tree.Of[n]
	if b == nil {
		return
	}
	src := t.Editor.Runes()
	width := b.ContentRect().W
	tt := st.TextTransform
	if a, ok := t.Layout.(*area); ok && a.width == width && slices.Equal(a.src, src) && sameFont(a.font, f) &&
		a.spacing == spacingOf(st) && a.tt == tt {
		return
	}
	var run *box.Box
	for _, c := range b.Children {
		if c.Kind == box.TextRun {
			run = c
			break
		}
	}
	m := d.fonts.Metrics(f)
	rows := int(b.ContentRect().H / max(m.Ascent+m.Descent+m.LineGap, 1))
	if run != nil && len(run.Lines) > 0 {
		rows = int(b.ContentRect().H / max(run.Lines[0].Rect.H, 1))
	}
	shown, cases := d.shownText(n, t)
	if sh := dom.Data(n).Shadow; !t.Empty() && sh != nil && sh.Text() != shown {
		n.SetShadow(shown, false)
		return
	}
	a := newArea([]rune(shown), run, st, d.fonts, rows)
	a.src, a.width, a.tt, a.cases = slices.Clone(src), width, tt, cases
	t.Layout = a
}

// shownText is a text field's text as it draws: bullets for a password, else
// its value as its text-transform cases it, with the map between them.
func (d *Document) shownText(n *Node, t *control.Text) (string, *caseMap) {
	st := dom.Data(n).Style
	if t.Password() || st == nil {
		return t.Shown(), nil
	}
	return fieldCase(t.Shown(), st.TextTransform)
}

func sameFont(a, b style.Font) bool {
	return a.Size == b.Size && a.Weight == b.Weight && a.Italic == b.Italic && slices.Equal(a.Family, b.Family)
}

// positionAt is the position in a text field nearest at, in the document's
// space: on a textarea's line under it. A point past the field's edge
// reaches the lines beyond.
func (d *Document) positionAt(n *Node, t *control.Text, at geom.Vec2) (int, bool) {
	b := d.tree.Of[n]
	if b == nil || t.Layout == nil {
		return 0, false
	}
	c := b.ContentRect()
	x := at.X - n.Box().X - c.X + b.Scroll.X
	line := 0
	if a, ok := t.Layout.(*area); ok {
		var run geom.Vec2
		for _, k := range b.Children {
			if k.Kind == box.TextRun {
				run = k.Pos
				break
			}
		}
		y := at.Y - n.Box().Y + b.Scroll.Y - run.Y
		for line < a.Lines()-1 && y >= a.Row(line).Y+a.Row(line).H {
			line++
		}
		x = at.X - n.Box().X + b.Scroll.X - run.X - a.Row(line).X
	}
	return t.Layout.At(line, x), true
}

// textClicks is the run of presses on a text field: a second one close in
// time and place selects the word there, a third the line, and a drag
// from them goes on by words or lines. from and to are what the last
// press selected.
type textClicks struct {
	on       *Node
	at       float32
	where    geom.Vec2
	count    int
	from, to int
}

// clickSlop is how far apart, in points, two presses still count as one
// run of clicks.
const clickSlop = 4

// pressText places a text field's caret where a press at falls, extend
// keeping the selection's other end; the second press of a run selects
// the word there instead and the third the line, as Chrome does.
func (d *Document) pressText(n *Node, t *control.Text, at geom.Vec2, extend bool) {
	i, ok := d.positionAt(n, t, at)
	if !ok {
		return
	}
	c := &d.textClicks
	if c.on == n && d.clock-c.at <= doubleClick && at.Sub(c.where).Len() <= clickSlop {
		c.count++
	} else {
		c.count = 1
	}
	c.on, c.at, c.where = n, d.clock, at
	if c.count == 1 || extend {
		t.Editor.Place(i, extend)
		return
	}
	c.from, c.to = clickSpan(t, i, c.count)
	t.Editor.Select(c.from, c.to)
}

// dragText takes a text field's selection to at while the press that
// began it is held: by positions, or by words or lines after a double or
// triple click, what that click selected staying selected. A drag past
// the field's edge reaches the lines beyond, which followCaret then
// scrolls in.
func (d *Document) dragText(n *Node, t *control.Text, at geom.Vec2) {
	i, ok := d.positionAt(n, t, at)
	if !ok {
		return
	}
	c := &d.textClicks
	if c.on != n || c.count < 2 {
		t.Editor.Place(i, true)
		return
	}
	from, to := clickSpan(t, i, c.count)
	if from < c.from {
		t.Editor.Select(c.to, from)
		return
	}
	t.Editor.Select(c.from, max(to, c.to))
}

// clickSpan is what the count-th click of a run at position i selects:
// the word, space or mark there, then the line with its break, a one-line
// field's whole text. A password has no words to find.
func clickSpan(t *control.Text, i, count int) (int, int) {
	switch {
	case t.Multiline() && count > 2:
		return t.Editor.Paragraph(i)
	case count > 2 || t.Password():
		return 0, len(t.Editor.Runes())
	}
	return t.Editor.Word(i)
}

// area is a textarea's text as layout cut it into lines: each line's
// first and last position, where each position is along it, and the
// line's rectangle in its text run's space. A line without text, which
// layout gives no fragment, stands a line's height under the one before.
type area struct {
	starts, ends []int
	xs           [][]float32
	rows         []geom.Rect
	page         int
	// src, font, spacing, width and case are what it was cut from; its
	// positions are the drawn text's, cases mapping the value's to them.
	src     []rune
	font    style.Font
	spacing spacing
	width   float32
	tt      style.TextTransform
	cases   *caseMap
}

func (a *area) Lines() int     { return len(a.starts) }
func (a *area) Line(i int) int { return a.line(a.cases.drawnAt(i)) }
func (a *area) line(j int) int {
	k, _ := slices.BinarySearch(a.starts, j+1)
	return max(k-1, 0)
}
func (a *area) Span(line int) (int, int) {
	return a.cases.valueAt(a.starts[line]), a.cases.valueAt(a.ends[line])
}
func (a *area) X(i int) float32 {
	j := a.cases.drawnAt(i)
	k := a.line(j)
	xs := a.xs[k]
	return xs[min(max(j-a.starts[k], 0), len(xs)-1)]
}
func (a *area) Page() int { return a.page }
func (a *area) At(line int, x float32) int {
	line = min(max(line, 0), len(a.starts)-1)
	best := 0
	for i, at := range a.xs[line] {
		if abs(at-x) < abs(a.xs[line][best]-x) {
			best = i
		}
	}
	return a.cases.valueAt(a.starts[line] + best)
}

// Row is line's rectangle in its text run's space.
func (a *area) Row(line int) geom.Rect { return a.rows[min(max(line, 0), len(a.rows)-1)] }

// newArea cuts text along run's lines, the fragments layout made of it:
// a hard break starts a line, a soft one drops the spaces it broke at.
func newArea(text []rune, run *box.Box, st *style.Computed, fonts box.Fonts, rowsShown int) *area {
	f := st.Font
	a := &area{font: f, spacing: spacingOf(st)}
	m := fonts.Metrics(f)
	lh := m.Ascent + m.Descent + m.LineGap
	var frags []box.Fragment
	if run != nil {
		frags = run.Lines
		if len(frags) > 0 {
			lh = frags[0].Rect.H
		}
	}
	next := float32(0)
	add := func(start int, shown []rune, r geom.Rect) {
		xs := make([]float32, len(shown)+1)
		var room float32
		for j, r := range shown {
			room += a.spacing.after(r)
			xs[j+1] = fonts.Advance(f, string(shown[:j+1])) + room
		}
		a.starts, a.ends = append(a.starts, start), append(a.ends, start+len(shown))
		a.xs, a.rows = append(a.xs, xs), append(a.rows, r)
		next = r.Y + r.H
	}
	fi, p := 0, 0
	for {
		end := p
		for end < len(text) && text[end] != '\n' {
			end++
		}
		q, first := p, true
		for q < end && fi < len(frags) {
			fr := frags[fi]
			shown := []rune(strings.TrimSuffix(fr.Text, emptyLine))
			for !first && q < end && !hasPrefix(text[q:end], shown) && text[q] == ' ' {
				q++
			}
			if !hasPrefix(text[q:end], shown) {
				break
			}
			add(q, shown, fr.Rect)
			q, first = q+len(shown), false
			fi++
		}
		if first {
			if fi < len(frags) && strings.TrimSuffix(frags[fi].Text, emptyLine) == "" && frags[fi].Rect.Y <= next+0.5 {
				add(p, nil, frags[fi].Rect)
				fi++
			} else {
				add(p, nil, geom.Rect{Y: next, H: lh})
			}
		} else if q < end {
			last := len(a.starts) - 1
			a.ends[last] = end
			for _, r := range text[q:end] {
				a.xs[last] = append(a.xs[last],
					a.xs[last][len(a.xs[last])-1]+fonts.Advance(f, string(r))+a.spacing.after(r))
			}
		}
		if end >= len(text) {
			break
		}
		p = end + 1
	}
	a.page = max(rowsShown, 1)
	return a
}

// emptyLine is what layout holds a textarea's line after its last break
// with: no text of its own.
const emptyLine = "\u00a0"

func hasPrefix(s, prefix []rune) bool {
	return len(s) >= len(prefix) && slices.Equal(s[:len(prefix)], prefix)
}
