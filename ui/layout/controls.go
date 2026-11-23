package layout

import (
	"math"
	"strconv"
	"strings"

	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

type controlKind uint8

const (
	noControl controlKind = iota
	inputControl
	selectControl
	textareaControl
)

// maxCharFactor is Chrome's max-char width over the average, as it measures the Go faces;
// an input's width is ceil(size × average) + (max - average).
const maxCharFactor = 1.875

// emptyLine holds the line after a textarea's last line break, which Chrome counts as
// one: its caret stands there, and the textarea scrolls to it.
const emptyLine = "\u00a0"

// textareaCaret is the point Chrome adds beside a scrolling textarea's scrollbar gutter.
const textareaCaret = 1

func controlOf(e *dom.Node) controlKind {
	if dom.Data(e).Shadow == nil {
		return noControl
	}
	switch e.Tag() {
	case dom.Input:
		return inputControl
	case dom.Select:
		if dom.ListBox(e) {
			return noControl
		}
		return selectControl
	case dom.Textarea:
		return textareaControl
	// no control
	case dom.TextTag, dom.HTML, dom.Body, dom.Div, dom.Section, dom.Header, dom.Footer, dom.Nav, dom.Main, dom.Aside,
		dom.Article, dom.P, dom.H1, dom.H2, dom.H3, dom.H4, dom.Span, dom.Strong, dom.Em, dom.Code, dom.Label, dom.Button,
		dom.Option, dom.Details, dom.Summary, dom.Dialog, dom.Table, dom.Thead, dom.Tbody, dom.Tr, dom.Th, dom.Td,
		dom.Progress, dom.Meter, dom.Img, dom.Svg, dom.Hr, dom.Canvas, dom.Kbd, dom.Samp, dom.Pre, dom.Caption, dom.Tfoot,
		dom.Ul, dom.Ol, dom.Li, dom.Dl, dom.Dt, dom.Dd, dom.H5, dom.H6, dom.Blockquote, dom.Figure, dom.Figcaption,
		dom.Form, dom.Address, dom.Hgroup, dom.Menu, dom.Search, dom.A, dom.B, dom.I, dom.U, dom.S, dom.Small,
		dom.Abbr, dom.Cite, dom.Mark, dom.Time, dom.Var, dom.Del, dom.Ins, dom.Dfn, dom.Bdi, dom.Bdo, dom.DataElement,
		dom.Br,
		dom.Center, dom.Big, dom.Font, dom.Sub, dom.Sup, dom.Q, dom.Fieldset, dom.Legend, dom.Iframe,
		dom.Col, dom.Colgroup, dom.PictureElement, dom.Source, dom.Optgroup:
	}
	return noControl
}

// buildControl gives an input, select or textarea its shadow text as its only inline
// content: on one unwrapped line, except a textarea's, and never a select's options.
func (t *tree) buildControl(n *node, kind controlKind) {
	shadow := dom.Data(n.box.Node).Shadow
	st := *styleOf(shadow)
	if kind != textareaControl {
		st.WhiteSpace = style.Pre
	}
	text := Translated(shadow, t.translate)
	if kind == textareaControl && strings.HasSuffix(text, "\n") {
		text += emptyLine
	}
	b := &box.Box{Kind: box.TextRun, Node: shadow, Style: styleOf(shadow)}
	c := &inlineContent{owner: n, runs: []*textRun{{box: b, parent: n.box, st: &st, span: -1, text: text}},
		unit: unitsPerPoint(t.fonts), height: none}
	c.cut(t.fonts)
	n.inline = c
	n.control = kind
}

// caretWidth is the point a field's caret takes after its text.
const caretWidth = 1

// inputText is how wide an input's text runs as shown (a password's bullets), its caret
// after it: what it scrolls along, as Chrome's inner editor does, though its text never
// widens it. A placeholder scrolls nothing.
func (t *tree) inputText(n *node) float32 {
	d := dom.Data(n.box.Node)
	if d.Placeholder {
		return 0
	}
	return t.fonts.Advance(styleOf(d.Shadow).Font, Translated(d.Shadow, t.translate)) + caretWidth
}

// centresContent reports a button, input, select or cell, which place their content in
// the height of their content box; a checkbox's or a radio's generated content starts at
// its top, as Chrome's.
func centresContent(n *node) bool {
	if n.cell {
		return true
	}
	if n.box.Node == nil {
		return false
	}
	e := n.box.Node
	if e.Tag() != dom.Input {
		return e.Tag() == dom.Button || e.Tag() == dom.Select && !dom.ListBox(e)
	}
	t, _ := e.Attr("type")
	t = strings.ToLower(strings.TrimSpace(t))
	return t != "checkbox" && t != "radio"
}

// contentAlign is where n's content stands in its height: where its align-content says,
// else in the middle of a button, input or select, and where its vertical-align says in
// a cell (top, middle or bottom; baseline and the rest as top).
func contentAlign(n *node) style.Align {
	if n.st.AlignContent != style.AlignNormal || !centresContent(n) {
		return n.st.AlignContent
	}
	if !n.cell {
		return style.AlignCenter
	}
	switch n.st.VerticalAlign {
	case style.AlignMiddle:
		return style.AlignCenter
	case style.AlignBottom:
		return style.AlignEnd
	case style.AlignBaselineText, style.AlignSub, style.AlignSuper, style.AlignTextTop, style.AlignTextBottom,
		style.AlignTop, style.AlignLength, style.AlignBaselineMiddle:
	}
	return style.AlignStart
}

// onBaseline reports a cell whose first line stands on its row's baseline: one with a
// vertical-align that is not top, middle or bottom, and no align-content.
func onBaseline(n *node) bool {
	return n.cell && n.st.AlignContent == style.AlignNormal && n.st.VerticalAlign != style.AlignTop &&
		contentAlign(n) == style.AlignStart
}

// lower sets how far a cell's content stands below its content box's top, its layouts
// with another forgotten.
func (n *node) lower(by float32) {
	if n.lowered != by {
		n.lowered = by
		n.cache.clear()
	}
}

// textTop is where the owner's first line box starts: the top of its content box, or the
// lines centred in it for a button, input or select, or where a cell's vertical-align
// puts them, unless its align-content says where.
func (t *tree) textTop(n *node, ls []line, top, contentHeight float32) float32 {
	trim, end := t.trims(n.inline, ls)
	share := contentShare(n)
	if share == 0 {
		return top + n.lowered - trim
	}
	h := -trim - end
	for _, l := range ls {
		lh, _ := t.lineBox(n.inline, l)
		h += l.push + lh
	}
	return top + max(contentHeight-h, 0)*share - trim
}

// contentShare is the share of the free height a box that centres its content leaves
// above it: half, or what its align-content says, or a cell's vertical-align.
func contentShare(n *node) float32 {
	if !centresContent(n) {
		return 0
	}
	switch contentAlign(n).Position() {
	case style.AlignStart, style.AlignFlexStart, style.AlignBaseline:
		return 0
	case style.AlignEnd, style.AlignFlexEnd, style.AlignLastBaseline:
		return 1
	case style.AlignNormal, style.AlignAuto, style.AlignStretch, style.AlignCenter, style.AlignAnchorCenter,
		style.AlignSpaceBetween,
		style.AlignSpaceAround, style.AlignSpaceEvenly:
	}
	return 0.5
}

// isButtonInput reports an input that is a button, as wide as what it says.
func isButtonInput(e *dom.Node) bool {
	typ, _ := e.Attr("type")
	return typ == "button" || typ == "submit" || typ == "reset"
}

func intAttr(e *dom.Node, name string, fallback int) int {
	if v, ok := e.Attr(name); ok {
		if i, err := strconv.Atoi(v); err == nil && i > 0 {
			return i
		}
	}
	return fallback
}

// measureControl is a control's content size as Chrome sizes it: its text never widens
// it. An input is size (20) average characters, a textarea cols (20) by rows (2) with
// room for its scrollbar, unless its gutter already holds it, a select its widest option,
// its placeholder counted while it shows it, as Chrome counts the hidden option that holds it.
func (t *tree) measureControl(n *node, known size) (size, float32, float32) {
	st := n.st
	e := n.box.Node
	lh, base := t.lineHeight(n.inline, st)
	avg := t.fonts.Advance(st.Font, "x")
	rows := 1
	var w float32
	switch n.control {
	case inputControl:
		if isButtonInput(e) {
			w = t.fonts.Advance(st.Font, n.inline.runs[0].text)
			break
		}
		w = float32(math.Ceil(float64(avg*float32(intAttr(e, "size", 20))))) + (maxCharFactor-1)*avg
	case textareaControl:
		w = avg * float32(intAttr(e, "cols", 20))
		if st.Overflow.Height != style.Hidden && st.Overflow.Height != style.Clip {
			w += textareaCaret
			if n.ownGutter().x == 0 {
				w += st.ScrollbarWidth
			}
		}
		rows = intAttr(e, "rows", 2)
	case selectControl:
		for _, o := range dom.Options(e) {
			w = max(w, t.fonts.Advance(st.Font, GroupIndent(o)+o.Text()))
		}
		if dom.Data(e).Placeholder && len(n.inline.runs) > 0 {
			w = max(w, t.fonts.Advance(st.Font, n.inline.runs[0].text))
		}
		w = float32(math.Ceil(float64(w)))
	case noControl:
	}
	s := size{or(known.w, w), or(known.h, float32(rows)*lh)}
	return s, base, base + float32(rows-1)*lh
}

// GroupIndent is what Chrome sets before the text of an option in an
// optgroup to measure a select's width and to show it in the list.
func GroupIndent(o *dom.Node) string {
	if p := o.Parent(); p != nil && p.Tag() == dom.Optgroup {
		return "    "
	}
	return ""
}

// textareaText is a textarea's text wrapped in its used width, as its scrollable content
// (never less than its content box, which Chrome's inner editor fills), and its gutter: once the text overflows the
// block axis under overflow auto, Chrome
// reserves the scrollbar's room and wraps the text again beside it.
func (t *tree) textareaText(n *node, s size, pb rect, gutter point) (size, point) {
	wrap := func(g point) size {
		box := size{fmax(s.w-pb.horizontal()-g.x, 0), fmax(s.h-pb.vertical()-g.y, 0)}
		m, _, _, _ := t.measure(n.inline, size{none, none}, spaces{w: def(box.w), h: maxContentSpace}, size{box.w, none}, 0)
		return m.mMax(box)
	}
	m := wrap(gutter)
	if n.st.Overflow.Height == style.OverflowAuto && gutter.x == 0 && m.h > s.h-pb.vertical()-gutter.y {
		gutter.x = n.st.ScrollbarWidth
		m = wrap(gutter)
	}
	return m, gutter
}
