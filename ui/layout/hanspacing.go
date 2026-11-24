package layout

import (
	"strings"
	"unicode"
	"unicode/utf8"

	eaw "golang.org/x/text/width"

	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/style"
)

// hanClass is how text-spacing-trim sees a character, as Chrome's
// HanKerning types it.
type hanClass uint8

const (
	hanOther hanClass = iota
	hanOpen
	hanClose
	hanMiddle
	hanDot
	hanOpenNarrow
	hanCloseNarrow
)

// hanClassOf is r's class, g its glyph: full-width brackets open and
// close, the ideographic comma and full stop and the full-width comma,
// stop, colon and semicolon are dots where their ink stands on the left
// (Japanese, Simplified Chinese) and middle dots where it is centred, and
// curly quotes are full-width brackets in a face that draws them an em
// wide.
func hanClassOf(r rune, g box.HanGlyph) hanClass {
	switch r {
	case 0x3001, 0x3002, 0xFF0C, 0xFF0E, 0xFF1A, 0xFF1B:
		switch g.Ink {
		case box.InkLeft:
			return hanDot
		case box.InkCentred:
			return hanMiddle
		case box.InkElsewhere:
		}
		return hanOther
	case 0x00B7, 0x30FB, 0x3000:
		return hanMiddle
	case 0x2018, 0x201C:
		if g.Full {
			return hanOpen
		}
		return hanOpenNarrow
	case 0x2019, 0x201D:
		if g.Full {
			return hanClose
		}
		return hanCloseNarrow
	}
	switch {
	case unicode.Is(unicode.Ps, r):
		if fullWidth(r) {
			return hanOpen
		}
		return hanOpenNarrow
	case unicode.Is(unicode.Pe, r):
		if fullWidth(r) {
			return hanClose
		}
		return hanCloseNarrow
	}
	return hanOther
}

// glyphClassed reports whether r's class depends on its glyph.
func glyphClassed(r rune) bool {
	switch r {
	case 0x3001, 0x3002, 0xFF0C, 0xFF0E, 0xFF1A, 0xFF1B, 0x2018, 0x2019, 0x201C, 0x201D:
		return true
	}
	return false
}

func fullWidth(r rune) bool {
	k := eaw.LookupRune(r).Kind()
	return k == eaw.EastAsianWide || k == eaw.EastAsianFullwidth
}

// trimsCurrent reports whether a character of class cur after one of
// class last is set half wide: an opening bracket after punctuation.
func trimsCurrent(cur, last hanClass) bool {
	return cur == hanOpen &&
		(last == hanOpen || last == hanMiddle || last == hanClose || last == hanDot || last == hanOpenNarrow)
}

// trimsLast reports whether a character of class last before one of
// class cur is set half wide: a closing bracket or a dot before
// punctuation that closes.
func trimsLast(cur, last hanClass) bool {
	return (last == hanClose || last == hanDot) &&
		(cur == hanClose || cur == hanMiddle || cur == hanDot || cur == hanCloseNarrow)
}

// spaceClass is how text-autospace sees a character.
type spaceClass uint8

const (
	spaceOther spaceClass = iota
	ideograph
	alphanumeric
)

// spaceClassOf is r's class, as Chrome's TextAutoSpace types it: Han
// (with the ideographic closing mark), and kana, bopomofo and the rest
// from U+3041 to U+31FF but punctuation, are ideographs; letters, marks
// and digits neither full-width nor half-width are alphanumeric.
func spaceClassOf(r rune) spaceClass {
	switch {
	case r >= 0x3041 && r <= 0x31FF && !unicode.IsPunct(r), r == 0x3006, unicode.Is(unicode.Han, r):
		return ideograph
	case unicode.In(r, unicode.L, unicode.M, unicode.Nd):
		if k := eaw.LookupRune(r).Kind(); k != eaw.EastAsianFullwidth && k != eaw.EastAsianHalfwidth {
			return alphanumeric
		}
	}
	return spaceOther
}

// hanTrim is how far text-spacing-trim moves a character's glyph and
// what it takes off its advance.
type hanTrim struct{ shift, space float32 }

// hanChar is a character of a word that text-spacing-trim or
// text-autospace spaces: the bytes [at, end) of its text (its marks
// with it), its trim, and the room text-autospace adds after it.
type hanChar struct {
	at, end int
	hanTrim
	auto float32
}

// hanSpacing is what text-spacing-trim and text-autospace do to a word:
// the characters they space within a line, in its width. wrapped and
// opening are its first character's trim at the start of a line a wrap
// begins and of one that begins the block or follows a forced break;
// closing is its last's where a line it would overflow ends after it.
type hanSpacing struct {
	chars                     []hanChar
	wrapped, opening, closing hanTrim
	// opens is set when the word starts with a full-width opening bracket,
	// whose trim a line's start sets.
	opens bool
}

// char is the entry of the character at byte at, ending at end, added
// in order.
func (h *hanSpacing) char(at, end int) *hanChar {
	if n := len(h.chars); n > 0 && h.chars[n-1].at == at {
		return &h.chars[n-1]
	}
	h.chars = append(h.chars, hanChar{at: at, end: end})
	return &h.chars[len(h.chars)-1]
}

// room is what h adds to its word's width within a line.
func (h *hanSpacing) room() float32 {
	var w float32
	for _, c := range h.chars {
		w += c.space + c.auto
	}
	return w
}

// lineStart is how a line's start sets its first word.
type lineStart uint8

const (
	midLine lineStart = iota
	// wrappedStart is a line a wrap begins.
	wrappedStart
	// openingStart is the block's first line or one after a forced break.
	openingStart
)

// closes reports whether a line ending after h's word of n bytes may
// trim its last character to fit, and how much wider that sets the word
// (less than 0).
func (h *hanSpacing) closes(n int) (float32, bool) {
	if h == nil || h.closing == (hanTrim{}) || len(h.chars) == 0 {
		return 0, false
	}
	c := h.chars[len(h.chars)-1]
	if c.end != n || c.space <= h.closing.space {
		return 0, false
	}
	return h.closing.space - c.space, true
}

// placed is h as a line places its word of n bytes, the line starting
// with it as start says and ending after it when end, its last
// character trimmed to fit when closed, with how much wider that makes
// the word.
func (h *hanSpacing) placed(n int, start lineStart, end, closed bool) (*hanSpacing, float32) {
	if h == nil || start == midLine && !end {
		return h, 0
	}
	out := &hanSpacing{chars: append([]hanChar(nil), h.chars...)}
	d := h.startRoom(start)
	if start != midLine && h.opens {
		out.chars[0].hanTrim = h.startTrim(start)
	}
	if end && len(out.chars) > 0 && out.chars[len(out.chars)-1].end == n {
		c := &out.chars[len(out.chars)-1]
		if trim, ok := h.closes(n); ok && closed {
			d += trim
			c.hanTrim = h.closing
		}
		d -= c.auto
		c.auto = 0
	}
	return out, d
}

// startRoom is how much wider a line starting as start sets h's word.
func (h *hanSpacing) startRoom(start lineStart) float32 {
	if h == nil || start == midLine || !h.opens {
		return 0
	}
	return h.startTrim(start).space - h.chars[0].space
}

// startTrim is the trim of h's opening bracket at a line's start.
func (h *hanSpacing) startTrim(start lineStart) hanTrim {
	if start == openingStart {
		return h.opening
	}
	return h.wrapped
}

// endRoom is how much wider a line ending after h's word of n bytes
// sets it, where the line's break drops the room text-autospace adds
// after its last character: a break inside the text of its own element,
// which Chrome shapes again, where it keeps the room at an element's
// end.
func (h *hanSpacing) endRoom(n int, drops bool) float32 {
	if h == nil || !drops || len(h.chars) == 0 || h.chars[len(h.chars)-1].end != n {
		return 0
	}
	return -h.chars[len(h.chars)-1].auto
}

// leadWord is the word piece k a line starts with, -1 for none, and how
// much wider the line's start sets it.
type leadWord struct {
	k int
	d float32
}

// of is piece k of ps as wide as l sets it.
func (l leadWord) of(ps []piece, k int) piece {
	p := ps[k]
	if k == l.k {
		p.width += l.d
	}
	return p
}

// lead is the word the unit [i, end) starts the open line's text with,
// as wide as the line's start sets it.
func (b *breaker) lead(i, end int) leadWord {
	if !b.blank || b.pending >= 0 {
		return leadWord{k: -1}
	}
	k := b.c.firstWord(i, end)
	if k < 0 || b.c.pieces[k].han == nil {
		return leadWord{k: -1}
	}
	return leadWord{k, b.c.pieces[k].han.startRoom(b.cur.start)}
}

// firstWord is the first word of the pieces [from, to) when nothing but
// edges, anchors, breaks and spaces a line drops stands before it, -1
// otherwise.
func (c *inlineContent) firstWord(from, to int) int {
	for k := from; k < to; k++ {
		switch {
		case c.pieces[k].kind == word:
			return k
		case !c.passes(c.pieces[k]):
			return -1
		}
	}
	return -1
}

// lastWord is the last word of the pieces [from, to) when nothing but
// edges, anchors, breaks and spaces a line drops stands after it, -1
// otherwise.
func (c *inlineContent) lastWord(from, to int) int {
	for k := to - 1; k >= from; k-- {
		switch {
		case c.pieces[k].kind == word:
			return k
		case !c.passes(c.pieces[k]):
			return -1
		}
	}
	return -1
}

// endFit is how much wider a line from piece from, at wide when it ends
// before piece end, is set by its end: its last word without the room
// text-autospace adds after it, and, where that leaves it wider than
// width, its closing bracket trimmed as Chrome measures it, before more
// text of its own or at the end of the content.
func (c *inlineContent) endFit(from, end int, at, width float32) float32 {
	k := c.lastWord(from, end)
	if k < 0 || c.pieces[k].han == nil {
		return 0
	}
	h, n := c.pieces[k].han, len(c.pieces[k].text)
	d := h.endRoom(n, c.runGoesOn(k, end))
	if at+d > width && c.ownUnit(k) && (c.runGoesOn(k, end) || c.endsContent(end)) {
		if trim, ok := h.closes(n); ok {
			d += trim
		}
	}
	return d
}

// lineEnd is the width of line l as its end sets it, and whether its
// closing bracket is trimmed: where it overflows its room otherwise, a
// wrap ending it before more text of the bracket's own or the end of
// the content, as Chrome 158 trims it.
func (c *inlineContent) lineEnd(l line, wrapped bool) (float32, bool) {
	k := c.lastWord(l.from, l.to)
	if k < 0 || c.pieces[k].han == nil {
		return l.width, false
	}
	h, n := c.pieces[k].han, len(c.pieces[k].text)
	w := l.width + h.endRoom(n, c.runGoesOn(k, l.to))
	if w > l.avail+layoutEpsilon && c.ownUnit(k) && (wrapped && c.runGoesOn(k, l.to) || c.endsContent(l.to)) {
		if trim, ok := h.closes(n); ok {
			return w + trim, true
		}
	}
	return w, false
}

// ownUnit reports whether the words a line may not break between that
// end with word k are all of its run.
func (c *inlineContent) ownUnit(k int) bool {
	for j := k - 1; j >= 0; j-- {
		switch p := c.pieces[j]; p.kind {
		case word:
			if p.run != c.pieces[k].run {
				return false
			}
		case edge, atomic, marker:
			return false
		case gap, breakHere, newline, emergency, anchor, tab, softHyphen:
			return true
		}
	}
	return true
}

// runGoesOn reports whether the next text after piece end, spaces and
// breaks aside, is a word of word k's run.
func (c *inlineContent) runGoesOn(k, end int) bool {
	if k < 0 {
		return false
	}
	for j := end; j < len(c.pieces); j++ {
		switch p := c.pieces[j]; p.kind {
		case gap, breakHere:
			continue
		case word:
			return p.run == c.pieces[k].run
		case newline, atomic, edge, emergency, marker, anchor, tab, softHyphen:
		}
		return false
	}
	return false
}

// endsContent reports whether nothing but spaces, breaks, edges and
// anchors stands from piece end on.
func (c *inlineContent) endsContent(end int) bool {
	for j := end; j < len(c.pieces); j++ {
		if k := c.pieces[j].kind; k != gap && k != breakHere && k != edge && k != anchor {
			return false
		}
	}
	return true
}

// spaced is the Fragment spacing of h's characters, its word's text
// standing at byte from of the fragment's, shown maps a byte of the
// word to the fragment's.
func (h *hanSpacing) spaced(out []box.Spacing, from int, shown func(int) int) []box.Spacing {
	if h == nil {
		return out
	}
	for _, c := range h.chars {
		if c.shift != 0 || c.space != 0 || c.auto != 0 {
			out = append(out, box.Spacing{At: from + shown(c.at), Shift: c.shift, Space: c.space + c.auto})
		}
	}
	return out
}

// fragmentSpacing is the Fragment spacing of the words of a line's
// pieces of one run, shown left to right, nil for none.
func fragmentSpacing(pieces []piece) []box.Spacing {
	var out []box.Spacing
	at := 0
	for _, p := range pieces {
		if p.kind != word && p.kind != gap {
			continue
		}
		if p.han != nil {
			out = p.han.spaced(out, at, func(b int) int { return len(unshown(p.text[:b])) })
		}
		at += len(unshown(p.text))
	}
	return out
}

// mayHan reports whether text has a character text-spacing-trim or
// text-autospace may space: ideographs and the punctuation round them,
// curly quotes and middle dots.
func mayHan(text string) bool {
	for i := 0; i < len(text); i++ {
		switch b := text[i]; {
		case b == 0xC2 && i+1 < len(text) && text[i+1] == 0xB7:
			return true
		case b >= 0xE2:
			r, _ := utf8.DecodeRuneInString(text[i:])
			if r >= 0x2018 && r <= 0x201D || r >= 0x2E80 {
				return true
			}
		}
	}
	return false
}

// hanSpaced is a character the spacing pass has read: its class for
// each property, its glyph, the word piece and the entry it is in, and
// its run's style.
type hanSpaced struct {
	trim  hanClass
	space spaceClass
	g     box.HanGlyph
	piece int
	at    int
	end   int
	st    *style.Computed
}

// spaceHan has text-spacing-trim and text-autospace space the words of
// the content, across its elements as Chrome spaces a paragraph's text:
// full-width punctuation after punctuation set half wide by its face's
// halt, and an eighth of the font size between an ideograph and a letter
// or a digit, after the first; each word's width takes the room. Where
// a line starts or ends changes some, which lineStart and lineEnd say.
func (c *inlineContent) spaceHan(fonts box.Fonts) {
	hf, ok := fonts.(box.HanFonts)
	if !ok || !c.hanText() {
		return
	}
	var last hanSpaced
	have := false
	for k := range c.pieces {
		p := &c.pieces[k]
		switch p.kind {
		case edge, anchor:
			continue
		case breakHere, emergency:
			if p.text == "" {
				continue
			}
			have = false
			continue
		case word:
		case gap, newline, atomic, marker, tab, softHyphen:
			have = false
			continue
		}
		st := c.runs[p.run].st
		if st.Mode.Vertical() {
			have = false
			continue
		}
		for at, r := range p.text {
			if have && last.piece == k && (unicode.In(r, unicode.Mn, unicode.Me) || isSelector(r)) {
				last.end = at + utf8.RuneLen(r)
				c.extend(last)
				continue
			}
			cur := readHan(hf, st, r, k, at)
			if have {
				c.pair(last, cur)
			}
			if at == 0 {
				c.opening(cur)
			}
			last, have = cur, true
		}
		if have && last.piece == k {
			c.closing(last)
		}
	}
	for k := range c.pieces {
		if h := c.pieces[k].han; h != nil {
			c.pieces[k].width += h.room()
		}
	}
}

// readHan is character r of style st, at byte at of word piece k, as
// the spacing pass reads it.
func readHan(hf box.HanFonts, st *style.Computed, r rune, k, at int) hanSpaced {
	cur := hanSpaced{space: spaceClassOf(r), piece: k, at: at, end: at + utf8.RuneLen(r), st: st}
	if r == softHyphenChar || unicode.Is(unicode.Cf, r) {
		cur.space = spaceOther
	}
	if glyphClassed(r) || unicode.In(r, unicode.Ps, unicode.Pe) {
		cur.g = hf.HanGlyph(st.Font, r)
	}
	cur.trim = hanClassOf(r, cur.g)
	return cur
}

// opening notes the trim a line's start gives ch, a word's first
// character, when it is a full-width opening bracket its face trims.
func (c *inlineContent) opening(ch hanSpaced) {
	if ch.trim != hanOpen || !ch.g.Halt || ch.g.Narrow == 0 {
		return
	}
	t := hanTrim{ch.g.Shift, ch.g.Narrow}
	h := c.han(ch.piece)
	h.char(0, ch.end)
	h.opens = true
	switch ch.st.SpacingTrim {
	case style.TrimStartSpacing:
		h.wrapped, h.opening = t, t
	case style.SpaceFirst:
		h.wrapped = t
	case style.SpacingTrimNormal, style.SpaceAll:
	}
}

// closing notes the trim a line's end may give ch, a word's last
// character, when it is a full-width closing bracket its face trims.
func (c *inlineContent) closing(ch hanSpaced) {
	if ch.trim != hanClose || !ch.g.Halt || ch.g.Narrow == 0 || ch.st.SpacingTrim == style.SpaceAll {
		return
	}
	h := c.han(ch.piece)
	h.char(ch.at, ch.end)
	h.closing = hanTrim{ch.g.Shift, ch.g.Narrow}
}

// hanText reports whether some run's text may be spaced.
func (c *inlineContent) hanText() bool {
	for _, r := range c.runs {
		if r.atomic == nil && mayHan(r.text) {
			return true
		}
	}
	return false
}

// han is piece k's spacing, made when it has none.
func (c *inlineContent) han(k int) *hanSpacing {
	p := &c.pieces[k]
	if p.han == nil {
		p.han = &hanSpacing{}
	}
	return p.han
}

// extend has the entry of character ch, when it has one, take the marks
// after it.
func (c *inlineContent) extend(ch hanSpaced) {
	h := c.pieces[ch.piece].han
	if h == nil {
		return
	}
	if n := len(h.chars); n > 0 && h.chars[n-1].at == ch.at {
		h.chars[n-1].end = ch.end
	}
}

// pair spaces characters last and cur, side by side: the room
// text-autospace adds after last, and the trim of whichever of the two
// text-spacing-trim sets half wide.
func (c *inlineContent) pair(last, cur hanSpaced) {
	if last.st.Autospace && cur.st.Autospace && last.space != spaceOther && cur.space != spaceOther &&
		last.space != cur.space {
		c.han(last.piece).char(last.at, last.end).auto = last.st.Font.Size / 8
	}
	switch {
	case trimsCurrent(cur.trim, last.trim):
		if cur.st.SpacingTrim != style.SpaceAll && cur.g.Halt {
			c.han(cur.piece).char(cur.at, cur.end).hanTrim = hanTrim{cur.g.Shift, cur.g.Narrow}
		}
	case trimsLast(cur.trim, last.trim):
		if last.st.SpacingTrim != style.SpaceAll && last.g.Halt {
			c.han(last.piece).char(last.at, last.end).hanTrim = hanTrim{last.g.Shift, last.g.Narrow}
		}
	}
}

// quoteBreaks calls fn with each byte of text a line may break at by
// Unicode 15.1's rule LB19a, which uniseg's older rules lack and Chrome
// follows: before an opening curly quote between wide characters, the
// one before it no opening bracket, and after a closing one between
// wide characters, the one after it an ideograph or an opening bracket.
func quoteBreaks(text string, fn func(pos int)) {
	if !mayHan(text) {
		return
	}
	type char struct {
		r  rune
		at int
	}
	var cs []char
	for at, r := range text {
		cs = append(cs, char{r, at})
	}
	for i := 1; i < len(cs); i++ {
		a, b := cs[i-1].r, cs[i].r
		switch {
		case curlyQuote(b) && unicode.Is(unicode.Pi, b):
			if eastAsian(a) && !unicode.Is(unicode.Ps, a) && i+1 < len(cs) && eastAsian(cs[i+1].r) {
				fn(cs[i].at)
			}
		case curlyQuote(a) && unicode.Is(unicode.Pf, a):
			if i >= 2 && eastAsian(cs[i-2].r) && opensLine(b) {
				fn(cs[i].at)
			}
		}
	}
}

func curlyQuote(r rune) bool {
	return r >= 0x2018 && r <= 0x201F || r == 0xAB || r == 0xBB || r == 0x2039 || r == 0x203A
}

// eastAsian reports a character of East Asian width full, wide or half.
func eastAsian(r rune) bool {
	k := eaw.LookupRune(r).Kind()
	return k == eaw.EastAsianFullwidth || k == eaw.EastAsianWide || k == eaw.EastAsianHalfwidth
}

// opensLine reports whether a line may start with r after a closing
// quote: an ideograph but the small kana and iteration and length marks
// no line starts with, or a full-width opening bracket.
func opensLine(r rune) bool {
	switch {
	case unicode.Is(unicode.Ps, r):
		return fullWidth(r)
	case strings.ContainsRune("ぁぃぅぇぉっゃゅょゎゕゖァィゥェォッャュョヮヵヶーゝゞヽヾ々〻", r), r >= 0x31F0 && r <= 0x31FF:
		return false
	}
	return spaceClassOf(r) == ideograph
}

// isSelector reports a variation selector, which its character takes.
func isSelector(r rune) bool { return r >= 0xFE00 && r <= 0xFE0F || r >= 0xE0100 && r <= 0xE01EF }
