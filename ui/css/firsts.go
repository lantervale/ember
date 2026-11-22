package css

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/rivo/uniseg"

	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

// firsts keeps, through the tree under root, the styles of the first
// lines and the boxes of the first letters (decision 0127): what each
// block's ::first-line makes of its text and of the inline elements on
// its first line, which layout gives that line, and the node of its
// ::first-letter with the letter its first text starts with.
func (r *restyler) firsts(root *dom.Node) {
	data := dom.Data(root)
	if !r.firstLine && !r.firstLetter && !data.Firsts || data.Style == nil {
		return
	}
	data.Firsts = r.firstLine || r.firstLetter
	over, transit, warns := r.over, r.transit, len(r.warns)
	r.over, r.transit = nil, nil
	defer func() { r.over, r.transit, r.warns = over, transit, r.warns[:warns] }()
	r.firstsOf(root, nil, data.Style.Font.Size)
}

// firstsOf keeps n's and its subtree's, line being the style of the first
// line n's first line is, as the first block of a block whose first line
// it holds, nil for none.
func (r *restyler) firstsOf(n *dom.Node, line *style.Computed, rem float32) {
	data := dom.Data(n)
	st := data.Style
	if st == nil || st.Display == style.DisplayNone {
		r.dropFirsts(n)
		return
	}
	container := blockContainer(st) && generates(n.Tag())
	if !container {
		line = nil
	}
	if container && r.firstLine && r.may(n, "first-line") {
		parent := st
		if line != nil {
			parent = line
		}
		if fl := r.firstStyle(n, parent, rem, "first-line"); fl != nil {
			line = fl
		}
	}
	r.setLine(n, n, line)
	r.backOf(n, data, line)
	open := line != nil
	r.lineContent(n, n, line, rem, &open)
	if container {
		r.letterOf(n, data, rem)
	}
}

// lineContent keeps the first line's styles of what n holds, line being
// n's style there (nil when n holds no first line), for block, whose
// first line it is. open is set while nothing of the line came before: a
// block then starts it.
func (r *restyler) lineContent(n, block *dom.Node, line *style.Computed, rem float32, open *bool) {
	for _, k := range n.Flow() {
		if k.IsText() {
			if strings.TrimLeft(k.Text(), " \t\n\r\f") != "" {
				*open = false
			}
			continue
		}
		st := dom.Data(k).Style
		switch {
		case st == nil || st.Display == style.DisplayNone:
			r.dropFirsts(k)
		case st.Float != style.FloatNone || st.Position == style.Absolute || st.Position == style.FixedPosition:
			r.firstsOf(k, nil, rem)
		case k.Tag() != dom.Br && !generates(k.Tag()) || st.Display == style.InlineBlock ||
			st.Display == style.InlineFlex || st.Display == style.InlineGrid:
			*open = false
			r.setLine(k, block, nil)
			r.firstsOf(k, nil, rem)
		case st.Display == style.Inline || st.Display == style.Contents:
			var in *style.Computed
			if line != nil {
				in = r.underLine(n, k, line, rem)
			}
			r.setLine(k, block, in)
			if k.Tag() == dom.Br {
				*open = false
			}
			r.lineContent(k, block, in, rem, open)
		default:
			var first *style.Computed
			if *open {
				first = line
			}
			*open = false
			r.firstsOf(k, first, rem)
		}
	}
}

// underLine is the style of k, an inline element or a ::before or ::after
// of n, on a first line: its own, inheriting from line, n's style there.
func (r *restyler) underLine(n, k *dom.Node, line *style.Computed, rem float32) *style.Computed {
	if p := k.Pseudo(); p != dom.NoPseudo {
		if s := r.generated(n, line, rem, p.String()); s != nil {
			return s
		}
		return nil
	}
	return r.compute(k, line, rem)
}

// setLine gives k the style line on block's first line, the one it had
// kept when equal, and lays block out again when it changed.
func (r *restyler) setLine(k, block *dom.Node, line *style.Computed) {
	data := dom.Data(k)
	if line == nil && data.FirstLine == nil || line != nil && data.FirstLine != nil && data.FirstLine.Equal(line) {
		return
	}
	data.FirstLine = line
	relayout(block)
}

// backOf keeps the node of the box behind n's first line while line has a
// background: the line's style less what Chrome draws of none of it there,
// its opacity and its shadows.
func (r *restyler) backOf(n *dom.Node, data *dom.Pipeline, line *style.Computed) {
	if line == nil || line.Background.A <= 0 && len(line.BackgroundImages) == 0 {
		if data.LineBack != nil {
			data.LineBack = nil
			relayout(n)
		}
		return
	}
	st := *line
	st.Opacity, st.Shadows = 1, nil
	g := data.LineBack
	if g == nil {
		g = dom.NewGenerated(n, dom.FirstLine)
		data.LineBack = g
		relayout(n)
	}
	if gd := dom.Data(g); gd.Style == nil || !gd.Style.Equal(&st) {
		gd.Style, gd.Shown = &st, &st
		relayout(n)
	}
}

// dropFirsts forgets what n's subtree kept of first lines and letters.
func (r *restyler) dropFirsts(n *dom.Node) {
	n.Walk(func(k *dom.Node) bool {
		if k.IsText() {
			return false
		}
		data := dom.Data(k)
		if data.FirstLine != nil || data.LineBack != nil {
			data.FirstLine, data.LineBack = nil, nil
			relayout(k)
		}
		if data.FirstLetter != nil {
			r.dropLetter(k, data)
		}
		for _, g := range []*dom.Node{data.Before, data.After} {
			if g != nil {
				dom.Data(g).FirstLine = nil
			}
		}
		return true
	})
}

// relayout lays n out again, and so its ancestors.
func relayout(n *dom.Node) {
	dom.Data(n).Dirty |= dom.DirtyLayout
	for p := n.Parent(); p != nil; p = p.Parent() {
		dom.Data(p).Dirty |= dom.DirtyDescendant
	}
}

// firstStyle is the style of n's pseudo-element name, inheriting from
// parent, nil when no rule matches it.
func (r *restyler) firstStyle(n *dom.Node, parent *style.Computed, rem float32, name string) *style.Computed {
	matched := r.pm.matchedFor(n, r.sheets, nil, name)
	if len(matched) == 0 {
		return nil
	}
	casc := r.pm.cascadeFor(matched, name)
	c := initial
	x := &ctx{c: &c, parent: parent, em: parent.Font.Size, rem: rem, env: r.env, overlay: r.overlay, node: n,
		self: true}
	x.cascaded(casc, false)
	return &c
}

// letterOf keeps n's ::first-letter: its style, inheriting from the
// element its letter is in, and the letter.
func (r *restyler) letterOf(n *dom.Node, data *dom.Pipeline, rem float32) {
	var text *dom.Node
	end := 0
	if r.firstLetter && r.may(n, "first-letter") {
		text, end, _ = letterIn(n)
	}
	if text != nil && dom.Data(text).Letter != nil && dom.Data(text).Letter != data.FirstLetter {
		// a block inside takes it with its own
		text = nil
	}
	var st *style.Computed
	if text != nil {
		in := text.Parent()
		parent := dom.Data(in).FirstLine
		if parent == nil {
			parent = dom.Data(in).Style
		}
		st = r.firstStyle(n, parent, rem, "first-letter")
	}
	if st == nil {
		if data.FirstLetter != nil {
			r.dropLetter(n, data)
		}
		return
	}
	g := data.FirstLetter
	if g == nil {
		g = dom.NewGenerated(n, dom.FirstLetter)
		data.FirstLetter = g
		relayout(n)
	}
	gd := dom.Data(g)
	if gd.Style == nil || !gd.Style.Equal(st) {
		gd.Style, gd.Shown = st, st
		relayout(n)
	}
	letter := strings.TrimLeft(text.Text()[:end], " \t\n\r\f")
	if kids := g.Children(); len(kids) != 1 || kids[0].Text() != letter {
		g.SetGenerated([]*dom.Node{dom.NewText(letter)})
		relayout(n)
	}
	for _, k := range g.Children() {
		dom.Data(k).Style, dom.Data(k).Shown = gd.Style, gd.Style
	}
	if old := gd.Letter; old != text || dom.Data(text).LetterEnd != end {
		if old != nil && old != text {
			dom.Data(old).Letter, dom.Data(old).LetterEnd = nil, 0
		}
		gd.Letter = text
		dom.Data(text).Letter, dom.Data(text).LetterEnd = g, end
		relayout(n)
		if p := text.Parent(); p != nil {
			relayout(p)
		}
	}
}

// dropLetter takes n's ::first-letter away.
func (r *restyler) dropLetter(n *dom.Node, data *dom.Pipeline) {
	g := data.FirstLetter
	if t := dom.Data(g).Letter; t != nil && dom.Data(t).Letter == g {
		dom.Data(t).Letter, dom.Data(t).LetterEnd = nil, 0
		if p := t.Parent(); p != nil {
			relayout(p)
		}
	}
	data.FirstLetter = nil
	relayout(n)
}

// letterIn finds the text node n's first letter is in, and where the
// letter ends in it, as Chrome looks for it: in the first text of its
// flow, through inline elements, ::before and the first block inside,
// past floats and positioned elements; a picture, a control, an inline
// block or a br before any text has none taken. stop is set once the
// search is over, found or not.
func letterIn(n *dom.Node) (*dom.Node, int, bool) {
	for _, k := range n.Flow() {
		if k.IsText() {
			if strings.TrimLeft(k.Text(), " \t\n\r\f") == "" {
				continue
			}
			if end := letterEnd(k.Text()); end > 0 {
				return k, end, true
			}
			return nil, 0, true
		}
		st := dom.Data(k).Style
		switch {
		case st == nil || st.Display == style.DisplayNone || st.Float != style.FloatNone ||
			st.Position == style.Absolute || st.Position == style.FixedPosition:
			continue
		case !generates(k.Tag()) || st.Display == style.InlineBlock || st.Display == style.InlineFlex ||
			st.Display == style.InlineGrid:
			return nil, 0, true
		case st.Display == style.Inline || st.Display == style.Contents || blockContainer(st):
			if t, end, stop := letterIn(k); stop {
				return t, end, true
			}
		default:
			return nil, 0, true
		}
	}
	return nil, 0, false
}

// letterEnd is where the first letter of s ends, in bytes, past the
// spaces before it, the punctuation round it: 0 when s has none before a
// space or its end.
func letterEnd(s string) int {
	i := 0
	skip := func(f func(rune) bool) {
		for i < len(s) {
			c, w := utf8.DecodeRuneInString(s[i:])
			if !f(c) {
				return
			}
			i += w
		}
	}
	skip(letterSpace)
	skip(letterPunct)
	if c, _ := utf8.DecodeRuneInString(s[i:]); i == len(s) || letterSpace(c) {
		return 0
	}
	g, _, _, _ := uniseg.FirstGraphemeClusterInString(s[i:], -1)
	i += len(g)
	skip(letterPunct)
	return i
}

func letterSpace(c rune) bool { return c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\f' }

// letterPunct is the punctuation a first letter takes round it.
func letterPunct(c rune) bool {
	return unicode.In(c, unicode.Ps, unicode.Pe, unicode.Pi, unicode.Pf, unicode.Po)
}

// blockContainer reports whether st lays out a block container, which
// has a first line: a block, a list item, an inline block, a cell or a
// caption; not a flex or grid container, nor a table.
func blockContainer(st *style.Computed) bool {
	switch st.Display {
	case style.Block, style.InlineBlock:
		return !st.Table && (st.TablePart == style.NoTablePart || st.TablePart == style.TableCell ||
			st.TablePart == style.TableCaption)
	case style.Inline, style.FlexDisplay, style.InlineFlex, style.GridDisplay, style.InlineGrid, style.DisplayNone,
		style.Contents:
	}
	return false
}
