package layout

import (
	"slices"
	"unicode/utf8"

	xbidi "golang.org/x/text/unicode/bidi"

	"github.com/lantervale/ember/ui/internal/bidi"
	"github.com/lantervale/ember/ui/internal/style"
)

// The bidi controls an inline element's unicode-bidi stands for (CSS Writing Modes 3, 2.4.2).
const (
	lre, rle, pdf = rune(0x202a), rune(0x202b), rune(0x202c)
	lro, rlo      = rune(0x202d), rune(0x202e)
	lri, rli, fsi = rune(0x2066), rune(0x2067), rune(0x2068)
	pdi           = rune(0x2069)
	objectChar    = rune(0xfffc)
	// firstRTL is where the characters of a right-to-left script or a bidi control
	// begin: text below it is left to right throughout.
	firstRTL = rune(0x0590)
)

// baseLevel is the paragraph level of a block container of style st: Auto under
// unicode-bidi: plaintext.
func baseLevel(st *style.Computed) bidi.Level {
	switch {
	case st.Bidi == style.BidiPlaintext:
		return bidi.Auto
	case st.Dir == style.RTL:
		return 1
	default:
		return 0
	}
}

// mixed reports whether the content may need reordering: its owner or one of its
// inline elements is right to left or orders its content apart, or its text has a
// right-to-left character or a bidi control.
func (c *inlineContent) mixed() bool {
	if c.owner.st.Dir == style.RTL || c.owner.st.Bidi == style.BidiPlaintext {
		return true
	}
	for _, s := range c.spans {
		if s.st.Dir == style.RTL || s.st.Bidi.Overrides() {
			return true
		}
	}
	for _, r := range c.runs {
		for _, ch := range r.text {
			if ch >= firstRTL {
				return true
			}
		}
	}
	return false
}

// opening is what an inline element of style st opens in the bidi text, its
// unicode-bidi as controls.
func opening(st *style.Computed) []rune {
	rtl := st.Dir == style.RTL
	pick := func(l, r rune) rune {
		if rtl {
			return r
		}
		return l
	}
	switch st.Bidi {
	case style.BidiEmbed:
		return []rune{pick(lre, rle)}
	case style.BidiIsolate:
		return []rune{pick(lri, rli)}
	case style.BidiOverride:
		return []rune{pick(lro, rlo)}
	case style.BidiIsolateOverride:
		return []rune{pick(lri, rli), pick(lro, rlo)}
	case style.BidiPlaintext:
		return []rune{fsi}
	case style.BidiNormal:
	}
	return nil
}

// closing is what an inline element of style st closes in the bidi text.
func closing(st *style.Computed) []rune {
	switch st.Bidi {
	case style.BidiEmbed, style.BidiOverride:
		return []rune{pdf}
	case style.BidiIsolate, style.BidiPlaintext:
		return []rune{pdi}
	case style.BidiIsolateOverride:
		return []rune{pdf, pdi}
	case style.BidiNormal:
	}
	return nil
}

// levels resolves the content's bidi levels, the runs read as one text, atoms as
// U+FFFC and each inline element's unicode-bidi as controls round its runs, a paragraph
// to each forced break: per run, the level of each character of its text as cut reads
// it, the one of an atom. nil when the content needs no reordering.
func (c *inlineContent) levels() [][]bidi.Level {
	c.base = 0
	if !c.mixed() {
		return nil
	}
	var text []rune
	starts := make([]int, len(c.runs))
	open := func(i int) {
		for k := len(c.spans) - 1; k >= 0; k-- {
			if s := c.spans[k]; s.first == i {
				text = append(text, opening(s.st)...)
				if s.end == i {
					text = append(text, closing(s.st)...)
				}
			}
		}
	}
	for i, r := range c.runs {
		open(i)
		starts[i] = len(text)
		if r.atomic != nil {
			text = append(text, objectChar)
		} else {
			text = append(text, []rune(collapsedBlanks(r.text, r.st.WhiteSpace))...)
		}
		for _, s := range c.spans {
			if s.end == i+1 && s.first <= i {
				text = append(text, closing(s.st)...)
			}
		}
	}
	open(len(c.runs))
	levels := make([]bidi.Level, len(text))
	base := baseLevel(c.owner.st)
	for from := 0; from < len(text); {
		to := from
		for to < len(text) && text[to] != '\n' {
			to++
		}
		to = min(to+1, len(text))
		p := bidi.Levels(text[from:to], base)
		if from == 0 {
			c.base = p.Base()
		}
		for i := from; i < to; i++ {
			levels[i] = p.Level(i - from)
		}
		from = to
	}
	out := make([][]bidi.Level, len(c.runs))
	for i, r := range c.runs {
		n := 1
		if r.atomic == nil {
			n = utf8.RuneCountInString(collapsedBlanks(r.text, r.st.WhiteSpace))
		}
		out[i] = levels[starts[i] : starts[i]+n]
	}
	return out
}

// levelEdges gives the pieces that hold no text of their own a level: a span's start
// edge that of the first piece inside it, its end edge that of the last, so neither cuts
// the text round them in two; the others that of the piece before them.
func (c *inlineContent) levelEdges() {
	inside := func(i int, step int) (bidi.Level, bool) {
		for k := i + step; k >= 0 && k < len(c.pieces); k += step {
			switch c.pieces[k].kind {
			case word, gap, atomic, newline:
				return c.pieces[k].level, true
			case edge:
				if c.pieces[k].start == (step < 0) {
					return 0, false
				}
			case breakHere, emergency, marker, anchor, tab:
			}
		}
		return 0, false
	}
	prev := c.base
	for i := range c.pieces {
		p := &c.pieces[i]
		switch p.kind {
		case word, gap, atomic, newline, tab:
		case marker:
			p.level = c.base
		case edge:
			p.level = prev
			step := 1
			if !p.start {
				step = -1
			}
			if l, ok := inside(i, step); ok {
				p.level = l
			}
		case breakHere, emergency, anchor:
			p.level = prev
		}
		prev = p.level
	}
}

// visual is a line's pieces as shown in the order they stand from the left, and the
// place of each in pieces: its spaces at its end at the paragraph's level (L1), then
// reordered by level (L2). Without reordering, pieces themselves.
func (c *inlineContent) visual(pieces []piece) ([]piece, []int) {
	if !c.reorders {
		return pieces, nil
	}
	levels := make([]bidi.Level, len(pieces))
	for i, p := range pieces {
		levels[i] = p.level
	}
	for i := len(pieces) - 1; i >= 0 && trailing(pieces[i].kind); i-- {
		if k := pieces[i].kind; k != edge && k != anchor {
			levels[i] = c.base
		}
	}
	order := bidi.Reorder(levels)
	out := make([]piece, len(pieces))
	for i, k := range order {
		out[i] = pieces[k]
	}
	return out, order
}

// ordered is line l's pieces as shown, in visual order, their spans' edges placed, and
// the place of each in the line (visual's).
func (c *inlineContent) ordered(l line) ([]piece, []int) {
	shown, order := c.visual(c.shown(l))
	c.placeEdges(shown, order, l.from)
	return shown, order
}

// trailing reports whether a piece of kind k at a line's end stands at the paragraph's
// level: a space, or what holds no text.
func trailing(k pieceKind) bool {
	switch k {
	case gap, breakHere, emergency, anchor, edge:
		return true
	case word, atomic, newline, marker, tab:
	}
	return false
}

// logical is the place in its line of the piece shown at i, by visual's order.
func logical(order []int, i int) int {
	if order == nil {
		return i
	}
	return order[i]
}

// placeEdges moves the edges of the spans on a line, shown in visual order with order
// mapping each to its place in the line starting at piece from, to the sides of their
// span's content its direction gives them: the start edge on the right of it under rtl,
// on its left otherwise, the end edge on the other side, as Chrome draws an inline box
// whose content the line reorders. Each edge's start then marks the one on the left.
func (c *inlineContent) placeEdges(shown []piece, order []int, from int) {
	if !c.reorders {
		return
	}
	move := func(at, to int) {
		p, o := shown[at], order[at]
		if to > at {
			copy(shown[at:to], shown[at+1:to+1])
			copy(order[at:to], order[at+1:to+1])
		} else {
			copy(shown[to+1:at+1], shown[to:at])
			copy(order[to+1:at+1], order[to:at])
		}
		shown[to], order[to] = p, o
	}
	for _, sp := range c.spans {
		rtl := sp.st.Dir == style.RTL
		for _, edgeAt := range []int{sp.open, sp.close} {
			at, lo, hi := -1, -1, -1
			for v := range shown {
				switch i := from + order[v]; {
				case i == edgeAt:
					at = v
				case i > sp.open && i < sp.close:
					if lo < 0 {
						lo = v
					}
					hi = v
				}
			}
			if at < 0 {
				continue
			}
			left := (edgeAt == sp.open) != rtl
			shown[at].start = left
			switch {
			case lo < 0:
			case left && at > lo:
				move(at, lo)
			case !left && at < hi:
				move(at, hi)
			}
		}
	}
}

// visualText is s, one paragraph of level base, as it is drawn from the left: reordered
// by its levels, the characters at odd ones mirrored.
func visualText(s string, base bidi.Level) string {
	text := []rune(s)
	p := bidi.Levels(text, base)
	levels := make([]bidi.Level, len(text))
	for i := range text {
		levels[i] = p.Level(i)
	}
	for i := len(text) - 1; i >= 0 && bidi.Class(text[i]) == xbidi.WS; i-- {
		levels[i] = p.Base()
	}
	out := make([]rune, len(text))
	for i, k := range bidi.Reorder(levels) {
		out[i] = text[k]
		if levels[k]%2 == 1 {
			if m := []rune(xbidi.ReverseString(string(text[k]))); len(m) == 1 {
				out[i] = m[0]
			}
		}
	}
	return string(out)
}

// startSides are the margin, border and padding a span of style st has at its start,
// then at its end: its left then its right, the other way round under rtl.
func startSides(sp *inlineSpan) (float32, float32) {
	l := sp.margin.l + sp.border.l + sp.padding.l
	r := sp.margin.r + sp.border.r + sp.padding.r
	if sp.st.Dir == style.RTL {
		return r, l
	}
	return l, r
}

// textOf is the text of the pieces of one run a line shows together, from the left:
// those of a run shown right to left are in the reverse of the text's order.
func textOf(pieces []piece, rtl bool) []piece {
	if !rtl {
		return pieces
	}
	out := slices.Clone(pieces)
	slices.Reverse(out)
	return out
}
