package ui

import (
	"cmp"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/internal/shape"
	"github.com/lantervale/ember/ui/control"
	"github.com/lantervale/ember/ui/internal/bidi"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
	"github.com/lantervale/ember/ui/layout"
)

// line is a one-line text field's text laid out: where each position is
// along it, and when it shows text right to left, the room each character
// is shown in. It measures the text as drawn and answers in the value's
// positions, through cases when text-transform changed the case.
type line struct {
	xs    []float32
	cells [][2]float32
	// src, font, spacing and case are what xs was measured from, so a
	// frame where none changed keeps the line.
	src     []rune
	font    style.Font
	spacing spacing
	tt      style.TextTransform
	cases   *caseMap
}

// spacing is the room a style adds after each character and after each
// word separator, and its text-spacing-trim and text-autospace.
type spacing struct {
	letter, word float32
	trim         style.SpacingTrim
	auto         bool
}

func spacingOf(st *style.Computed) spacing {
	return spacing{st.LetterGap(), st.WordGap(), st.SpacingTrim, st.Autospace}
}

// after is the room s adds after r, the last character of a grapheme
// cluster letter-spacing follows or not (spacedAfter).
func (s spacing) after(r rune, spaced bool) float32 {
	var room float32
	if spaced {
		room = s.letter
	}
	if s.word != 0 && style.WordSeparator(r) {
		room += s.word
	}
	return room
}

// spacedAfter says, for each character of text after before, whether it
// ends a grapheme cluster letter-spacing follows (shape.EachSpaced).
func spacedAfter(before, text string) []bool {
	out := make([]bool, utf8.RuneCountInString(text))
	n, at := 0, 0
	shape.EachSpaced(before, text, func(end int) {
		for ; at < end; n++ {
			_, size := utf8.DecodeRuneInString(text[at:])
			at += size
		}
		out[n-1] = true
	})
	return out
}

func newLine(text []rune, st *style.Computed, fonts box.Fonts) *line {
	l := &line{font: st.Font, spacing: spacingOf(st)}
	l.xs, l.cells = measureLine(text, st, fonts, layout.LineSpacing(fonts, st, string(text)))
	return l
}

// measureLine is where each position of text stands along its line and,
// for a line shown right to left or in both directions, the room each
// character is shown in, as visualLine gives them. han is the room
// text-spacing-trim and text-autospace give its characters, by byte.
func measureLine(text []rune, st *style.Computed, fonts box.Fonts, han []box.Spacing) ([]float32, [][2]float32) {
	xs := make([]float32, len(text)+1)
	sp := spacingOf(st)
	spaced := spacedAfter("", string(text))
	var room float32
	at := 0
	for i, r := range text {
		room += sp.after(r, spaced[i])
		for len(han) > 0 && han[0].At <= at {
			if han[0].At == at {
				room += han[0].Space
			}
			han = han[1:]
		}
		at += utf8.RuneLen(r)
		xs[i+1] = fonts.Advance(st.Font, string(text[:i+1])) + room
	}
	if st.Dir == style.RTL || slices.ContainsFunc(text, func(r rune) bool { return r >= firstRTL }) {
		return visualLine(text, xs, st.Dir == style.RTL)
	}
	return xs, nil
}

// firstRTL is where the characters of a right-to-left script begin: text below it is
// left to right throughout.
const firstRTL = 0x0590

// visualLine lays text out in a paragraph of direction rtl or not, from
// the line's left, xs being the edges of its characters in the text's
// order. It returns where the caret stands for each place, before a
// character on the side its text comes from (the right of one shown right
// to left), at the end after the last one, and the room each character is
// shown in.
func visualLine(text []rune, xs []float32, rtl bool) ([]float32, [][2]float32) {
	var base bidi.Level
	if rtl {
		base = 1
	}
	p := bidi.Levels(text, base)
	levels := make([]bidi.Level, len(text))
	for i := range text {
		levels[i] = p.Level(i)
	}
	cells := make([][2]float32, len(text))
	var x float32
	for _, k := range bidi.Reorder(levels) {
		w := xs[k+1] - xs[k]
		cells[k] = [2]float32{x, x + w}
		x += w
	}
	edges := make([]float32, len(text)+1)
	for i := range text {
		edges[i] = cells[i][0]
		if levels[i]%2 == 1 {
			edges[i] = cells[i][1]
		}
	}
	switch n := len(text); {
	case n == 0:
	case levels[n-1]%2 == 1:
		edges[n] = cells[n-1][0]
	default:
		edges[n] = cells[n-1][1]
	}
	return edges, cells
}

// ranges are the stretches of a line its characters from a up to z are
// shown in, from the left, one for each run of them side by side: a
// selection over text of both directions lies in several, as Chrome draws
// it. Without cells the line is left to right throughout.
func ranges(xs []float32, cells [][2]float32, a, z int) [][2]float32 {
	if z <= a {
		return nil
	}
	if cells == nil {
		return [][2]float32{{xs[a], xs[z]}}
	}
	part := slices.Clone(cells[a:z])
	slices.SortFunc(part, func(p, q [2]float32) int { return cmp.Compare(p[0], q[0]) })
	out := part[:1]
	for _, c := range part[1:] {
		if last := &out[len(out)-1]; c[0] <= last[1]+0.01 {
			last[1] = max(last[1], c[1])
			continue
		}
		out = append(out, c)
	}
	return out
}

func (l *line) Lines() int          { return 1 }
func (l *line) Line(int) int        { return 0 }
func (l *line) Span(int) (int, int) { return 0, l.cases.valueAt(len(l.xs) - 1) }
func (l *line) X(i int) float32     { return l.xs[l.drawn(i)] }
func (l *line) Page() int           { return 1 }
func (l *line) drawn(i int) int     { return min(max(l.cases.drawnAt(i), 0), len(l.xs)-1) }

// Ranges are the stretches of the line the selection from up to to is
// shown in, from its left.
func (l *line) Ranges(_, from, to int) [][2]float32 {
	return ranges(l.xs, l.cells, l.drawn(from), l.drawn(to))
}

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
// first and last position, where each position is along it, the room each
// character is shown in on a line shown right to left, and the line's
// rectangle in its text run's space. A line without text, which layout
// gives no fragment, stands a line's height under the one before.
type area struct {
	starts, ends []int
	xs           [][]float32
	cells        [][][2]float32
	rows         []geom.Rect
	page         int
	// broken is whether each line ends at a line break, which a selection
	// going on past it shows as a space's room at the line's end, on the
	// left in an rtl textarea.
	broken []bool
	space  float32
	rtl    bool
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

// Ranges are the stretches of line the selection from up to to is shown
// in, from the left of its row: with a space's room at the line's end,
// on its left right to left, where the selection goes on past its break.
func (a *area) Ranges(line, from, to int) [][2]float32 {
	line = min(max(line, 0), len(a.starts)-1)
	s, n := a.starts[line], len(a.xs[line])-1
	in := func(i int) int { return min(max(a.cases.drawnAt(i)-s, 0), n) }
	out := ranges(a.xs[line], a.cells[line], in(from), in(to))
	if !a.broken[line] || a.cases.drawnAt(to) <= a.ends[line] {
		return out
	}
	if a.rtl {
		left := slices.Min(a.xs[line])
		if len(out) > 0 && out[0][0] <= left+0.01 {
			out[0][0] = left - a.space
			return out
		}
		return slices.Insert(out, 0, [2]float32{left - a.space, left})
	}
	right := slices.Max(a.xs[line])
	if k := len(out) - 1; k >= 0 && out[k][1] >= right-0.01 {
		out[k][1] = right + a.space
		return out
	}
	return append(out, [2]float32{right, right + a.space})
}

// Row is line's rectangle in its text run's space.
func (a *area) Row(line int) geom.Rect { return a.rows[min(max(line, 0), len(a.rows)-1)] }

// row is the fragments layout put side by side on one line, bidi's runs
// of it in the order they show: the characters from start up to end of
// the text they hold, and its rectangle round them all.
type row struct {
	frags      []box.Fragment
	start, end int
	rect       geom.Rect
}

// rowsOf gathers frags into the rows they stand on, and reports whether
// each holds the characters of text it says it starts at: none do when
// layout cut other text than text.
func rowsOf(frags []box.Fragment, text []rune) ([]row, bool) {
	var out []row
	for _, f := range frags {
		shown := []rune(strings.TrimSuffix(f.Text, emptyLine))
		start, end := int(f.Start), int(f.Start)+len(shown)
		if start < 0 || end > len(text) || !slices.Equal(text[start:end], shown) {
			return nil, false
		}
		f.Text = string(shown)
		if n := len(out); n > 0 && abs(out[n-1].rect.Y-f.Rect.Y) < 0.5 {
			r := &out[n-1]
			r.frags = append(r.frags, f)
			r.start, r.end = min(r.start, start), max(r.end, end)
			right := max(r.rect.X+r.rect.W, f.Rect.X+f.Rect.W)
			r.rect.X = min(r.rect.X, f.Rect.X)
			r.rect.W = right - r.rect.X
			continue
		}
		out = append(out, row{frags: []box.Fragment{f}, start: start, end: end, rect: f.Rect})
	}
	return out, true
}

// newArea cuts text along run's lines, the fragments layout made of it:
// a hard break starts a line, a soft one drops the spaces it broke at.
func newArea(text []rune, run *box.Box, st *style.Computed, fonts box.Fonts, rowsShown int) *area {
	a := &area{font: st.Font, spacing: spacingOf(st), space: fonts.Advance(st.Font, " "), rtl: st.Dir == style.RTL}
	m := fonts.Metrics(st.Font)
	lh := m.Ascent + m.Descent + m.LineGap
	var rows []row
	if run != nil {
		rows, _ = rowsOf(run.Lines, text)
		if len(rows) > 0 {
			lh = rows[0].rect.H
		}
	}
	next := float32(0)
	add := func(start, end int, r row) {
		xs, cells := measureLine(text[start:end], st, fonts, rowSpacing(text, r, start))
		if cells == nil && slices.Contains(text[start:end], '\t') {
			tabbed(xs, start, r)
		}
		a.starts, a.ends = append(a.starts, start), append(a.ends, end)
		a.xs, a.cells, a.rows = append(a.xs, xs), append(a.cells, cells), append(a.rows, r.rect)
		a.broken = append(a.broken, end < len(text) && text[end] == '\n')
		next = r.rect.Y + r.rect.H
	}
	ri, p := 0, 0
	for {
		end := p
		for end < len(text) && text[end] != '\n' {
			end++
		}
		first := true
		for ri < len(rows) && rows[ri].start >= p && (rows[ri].start < end || rows[ri].start == p && first) {
			add(rows[ri].start, rows[ri].end, rows[ri])
			first = false
			ri++
		}
		switch last := len(a.starts) - 1; {
		case first:
			add(p, p, row{rect: geom.Rect{Y: next, H: lh}})
		case a.ends[last] < end:
			// spaces hung past the line's end are on it
			r := rows[ri-1]
			a.starts, a.ends, a.xs, a.cells, a.rows = a.starts[:last], a.ends[:last], a.xs[:last], a.cells[:last], a.rows[:last]
			a.broken = a.broken[:last]
			r.end = end
			add(r.start, end, r)
		}
		if end >= len(text) {
			break
		}
		p = end + 1
	}
	a.page = max(rowsShown, 1)
	return a
}

// rowSpacing is the room text-spacing-trim and text-autospace give the
// characters of row r of text, which starts at character start, by byte
// from there.
func rowSpacing(text []rune, r row, start int) []box.Spacing {
	var out []box.Spacing
	for _, f := range r.frags {
		if len(f.Spacing) == 0 || int(f.Start) < start {
			continue
		}
		before := len(string(text[start:f.Start]))
		for _, s := range f.Spacing {
			s.At += before
			out = append(out, s)
		}
	}
	slices.SortFunc(out, func(a, b box.Spacing) int { return cmp.Compare(a.At, b.At) })
	return out
}

// tabbed moves the places xs measured along a row from start to where
// layout put its fragments, a tab's room being the gap between two.
func tabbed(xs []float32, start int, r row) {
	for _, f := range r.frags {
		off := int(f.Start) - start
		delta := f.Rect.X - r.rect.X - xs[off]
		for j := off; j < len(xs); j++ {
			xs[j] += delta
		}
	}
}

// emptyLine is what layout holds a textarea's line after its last break
// with: no text of its own.
const emptyLine = "\u00a0"
