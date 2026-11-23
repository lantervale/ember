// Package export writes a Document as an HTML page with its stylesheets
// and fonts, which Chrome lays out as ui does: the reference of the
// layout tests, and pages for the docs.
package export

import (
	"encoding/base64"
	"fmt"
	"html"
	"io"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode/utf16"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/css"
	"github.com/lantervale/ember/ui/internal/assets"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/fonts"
	"github.com/lantervale/ember/ui/internal/srcset"
	"github.com/lantervale/ember/ui/internal/style"
	"github.com/lantervale/ember/ui/layout"
)

// Option adds to the exported page.
type Option func(*page)

type page struct {
	head     []string
	snapshot func(n *ui.Node) ([]byte, bool)
}

// Head adds raw markup to the page's head, such as a test's script.
func Head(markup string) Option { return func(p *page) { p.head = append(p.head, markup) } }

// Snapshots gives each canvas the PNG of what ember drew in its content
// box, at the device scale, for Chrome to draw in it.
func Snapshots(of func(n *ui.Node) ([]byte, bool)) Option {
	return func(p *page) { p.snapshot = of }
}

// generatedBoxes sizes Chrome's ::before and ::after boxes by their
// border box as ui's are, which the user-agent sheet's * rule leaves out.
const generatedBoxes = "::before, ::after { box-sizing: border-box; }\n"

// Write writes d as a self-contained HTML page: the user-agent sheet,
// the fonts as data URLs, the document's sheets, then its tree.
func Write(w io.Writer, d *ui.Document, opts ...Option) error {
	var p page
	for _, o := range opts {
		o(&p)
	}
	assets.Styled(d)
	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html")
	attrs(&b, d, d.HTML())
	if d.ScrollingViewport() {
		b.WriteString(" " + ScrollingViewport)
	}
	b.WriteString(">\n<head>\n<meta charset=\"utf-8\">\n" + contentLanguage(d) + "<style>\n")
	for _, f := range fonts.Of(d).Faces() {
		style := "normal"
		if f.Italic {
			style = "italic"
		}
		fmt.Fprintf(&b, "@font-face { font-family: %q; %s font-style: %s; "+
			"src: url(data:font/ttf;base64,%s); }\n",
			f.Family, f.Descriptors(), style, base64.StdEncoding.EncodeToString(f.TTF))
	}
	set := fonts.Of(d)
	for _, f := range set.Faces() {
		if r := set.FallbackRanges(f); r != "" {
			style := "normal"
			if f.Italic {
				style = "italic"
			}
			fmt.Fprintf(&b, "@font-face { font-family: %q; font-weight: %d; font-style: %s; "+
				"src: url(data:font/ttf;base64,%s); unicode-range: %s; }\n",
				f.Family, f.Weight, style, base64.StdEncoding.EncodeToString(set.Fallback().TTF), r)
		}
	}
	switch {
	case d.OverlayScrollbars():
		b.WriteString(overlayScrollbars)
	case d.ScrollingViewport():
		b.WriteString(classicScrollbars)
	default:
		b.WriteString(classicScrollbars + stillViewport)
	}
	b.WriteString(ownLook)
	agent := strings.ReplaceAll(css.Default, ":"+nativeClass, ownLookOn) + listTypes
	if d.BrowserSheets() {
		agent += css.BrowserDefault
	} else {
		agent += generatedBoxes
	}
	b.WriteString("@layer ember-user-agent {\n" + states(inline(d, agent, nil)) + "}\n")
	b.WriteString(truncateStart)
	for _, s := range d.Sheets() {
		b.WriteString(states(inline(d, s.String(), s.URL)))
	}
	b.WriteString("</style>\n")
	b.WriteString(showModals)
	b.WriteString(scrollTo)
	b.WriteString(selectText)
	b.WriteString(drawSnapshots)
	for _, h := range p.head {
		b.WriteString(h)
		b.WriteByte('\n')
	}
	b.WriteString("</head>\n")
	node(&b, &p, d, d.Body())
	b.WriteString("\n</html>\n")
	_, err := io.WriteString(w, b.String())
	return err
}

// tr is d's translator as layout takes it, nil for none.
func tr(d *ui.Document) func(string) string {
	if t := d.Translator(); t != nil {
		return t.Tr
	}
	return nil
}

// dataURL is the picture url names in d's files as a data URL, url
// itself for one they don't hold.
func dataURL(d *ui.Document, url string) string {
	if data, ok := assets.Of(d, url); ok {
		return "data:" + assets.MediaType(url, data) + ";base64," + base64.StdEncoding.EncodeToString(data)
	}
	return url
}

func node(b *strings.Builder, p *page, d *ui.Document, n *dom.Node) {
	if n.IsText() {
		b.WriteString(html.EscapeString(layout.Translated(n, tr(d))))
		return
	}
	b.WriteByte('<')
	b.WriteString(n.Tag().String())
	attrs(b, d, n)
	if n.Tag() == dom.Canvas && p.snapshot != nil {
		if data, ok := p.snapshot(n); ok {
			fmt.Fprintf(b, ` data-ember-snapshot="data:image/png;base64,%s"`, base64.StdEncoding.EncodeToString(data))
		}
	}
	if n.Tag() == dom.Iframe {
		frame(b, d, n)
	}
	b.WriteByte('>')
	if n.Tag().Void() {
		return
	}
	if n.Tag() == dom.Svg {
		b.WriteString(n.Markup() + "</svg>")
		return
	}
	if ph, ok := n.Attr("placeholder"); ok && n.Tag() == dom.Select && d.Selected(n) < 0 {
		fmt.Fprintf(b, `<option value="" disabled selected hidden data-ember-implied="">%s</option>`,
			html.EscapeString(layout.TranslatedAt(n, ph, tr(d))))
	}
	cut := slices.Contains(n.Classes(), "truncate-start")
	if cut {
		b.WriteString(`<bdi dir="ltr" data-ember-implied="">`)
	}
	for _, c := range n.Children() {
		node(b, p, d, c)
	}
	if cut {
		b.WriteString("</bdi>")
	}
	b.WriteString("</")
	b.WriteString(n.Tag().String())
	b.WriteByte('>')
}

// frame writes the Document the iframe n shows as its srcdoc, laid out
// at n's content box when n has one.
func frame(b *strings.Builder, d *ui.Document, n *dom.Node) {
	child := d.Frame(n)
	if child == nil {
		return
	}
	if r, ok := d.BoxOf(n); ok {
		child.Layout(geom.V2(r.W, r.H).Scale(1 / d.Zoom()))
	}
	var page strings.Builder
	if err := Write(&page, child); err != nil {
		return
	}
	fmt.Fprintf(b, ` srcdoc="%s"`, html.EscapeString(page.String()))
}

// contentLanguage is the meta element giving the page the Document's
// language, which Chrome's :lang() reads as a ui Document's.
func contentLanguage(d *ui.Document) string {
	if lang := d.Language(); lang != "" {
		return `<meta http-equiv="content-language" content="` + html.EscapeString(lang) + "\">\n"
	}
	return ""
}

// attrs writes n's attributes, a control's as it is now: an input's
// value and checked, a select's chosen option. A select's placeholder,
// a ui extension, becomes the hidden first option Chrome shows instead.
func attrs(b *strings.Builder, d *ui.Document, n *dom.Node) {
	write := func(name, value string) {
		fmt.Fprintf(b, " %s=\"%s\"", name, html.EscapeString(value))
	}
	for _, name := range []string{"id", "class"} {
		if v, ok := n.Attr(name); ok {
			write(name, v)
		}
	}
	for _, p := range pseudo {
		if n.State().Has(p.state) {
			write("data-ember-"+p.name, "")
		}
	}
	if s := dom.Data(n).Scroll; s.X != 0 || s.Y != 0 {
		write("data-ember-scroll", fmt.Sprintf("%g %g", s.X, s.Y))
	}
	if sel := d.Selection(); !sel.Empty() {
		if sel.Anchor.Node.Parent() == n {
			write("data-ember-anchor", place(d, sel.Anchor))
		}
		if sel.Focus.Node.Parent() == n {
			write("data-ember-focus", place(d, sel.Focus))
		}
	}
	if from, to := d.SelectionRange(n); from != to && (d.Focused(n) || hasAttr(n, "autofocus")) {
		write("data-ember-field-selection", fmt.Sprintf("%d %d", from, to))
	}
	if style := liveStyle(d, n); style != "" {
		write("style", inline(d, style, nil))
	}
	typ, _ := n.Attr("type")
	var layer uint64
	modal := false
	if n.Tag() == dom.Dialog {
		layer, modal = assets.Modal(d, n)
	}
	if modal {
		write("data-ember-modal", strconv.FormatUint(layer, 10))
	}
	if st := dom.Data(n).Style; st != nil && st.Scrollbar != nil {
		write("data-ember-bars", "")
	}
	if bare(n) {
		write("data-ember-bare", "")
	}
	live := map[string]bool{
		"open": modal, "style": true, "checked": n.Tag() == dom.Input, "value": n.Tag() == dom.Input && typ != checkboxType,
		"placeholder": n.Tag() == dom.Select, "selected": n.Tag() == dom.Option,
	}
	for _, a := range n.Attrs() {
		if live[a.Name] {
			continue
		}
		if n.Tag() == dom.Img && a.Name == "src" {
			write("src", dataURL(d, a.Value))
			continue
		}
		if (n.Tag() == dom.Img || n.Tag() == dom.Source) && a.Name == "srcset" {
			write("srcset", srcset.Replace(a.Value, func(url string) string { return dataURL(d, url) }))
			continue
		}
		if a.Name == "placeholder" {
			write(a.Name, layout.TranslatedAt(n, a.Value, tr(d)))
			continue
		}
		write(a.Name, a.Value)
	}
	if name, value, ok := controlState(d, n, typ); ok {
		write(name, value)
	}
}

// liveStyle is n's style attribute with what its control shows now: a
// switch's thumb, a bar's fill, a placeholder's colour.
func liveStyle(d *ui.Document, n *dom.Node) string {
	style, _ := n.Attr("style")
	if thumb := switchThumb(d, n); thumb != "" {
		style = strings.TrimPrefix(strings.TrimSpace(style+"; "+thumb), "; ")
	}
	if accent := accentLook(d, n); accent != "" {
		style = strings.TrimPrefix(strings.TrimSpace(style+"; "+accent), "; ")
	}
	if bar := barLook(d, n); bar != "" {
		style = strings.TrimPrefix(strings.TrimSpace(style+"; "+bar), "; ")
	}
	if c, ok := placeholderColor(d, n); ok {
		style = strings.TrimSpace(style + "; color: " + c)
		style = strings.TrimPrefix(style, "; ")
	}
	return style
}

// controlState is the attribute that says a control's state now: a
// checkbox or radio checked, an input's value, the chosen option
// selected.
func controlState(d *ui.Document, n *dom.Node, typ string) (string, string, bool) {
	switch {
	case n.Tag() == dom.Input && (typ == checkboxType || typ == "radio"):
		return "checked", "", d.Checked(n)
	case n.Tag() == dom.Input:
		v := d.Value(n)
		return "value", v, v != ""
	case n.Tag() == dom.Option:
		p := n.Parent()
		return "selected", "", p != nil && p.Tag() == dom.Select && optionIndex(p, n) == d.Selected(p)
	}
	return "", "", false
}

func optionIndex(sel, opt *dom.Node) int {
	i := 0
	for _, c := range sel.Children() {
		if c == opt {
			return i
		}
		if c.Tag() == dom.Option {
			i++
		}
	}
	return -1
}

var urls = regexp.MustCompile(`url\(\s*(['"]?)([^'")]+)['"]?\s*\)`)

// inline replaces the urls of css, each named as resolve has it (a
// sheet's URL; nil for the user-agent sheet and a style attribute, whose
// urls are as written), by data URLs of the document's images, so the
// page carries them. A picture the document doesn't hold keeps its url,
// written in full when its sheet has an address: the page is no longer
// beside it.
func inline(d *ui.Document, src string, resolve func(string) string) string {
	return urls.ReplaceAllStringFunc(src, func(m string) string {
		url := urls.FindStringSubmatch(m)[2]
		name := url
		if resolve != nil {
			name = resolve(url)
		}
		b, ok := assets.Of(d, name)
		if !ok && name != url && strings.Contains(name, "://") {
			return "url(" + name + ")"
		}
		if !ok {
			return m
		}
		return fmt.Sprintf("url(data:%s;base64,%s)", assets.MediaType(name, b), base64.StdEncoding.EncodeToString(b))
	})
}

// switchThumb is the declarations that draw a switch's thumb in Chrome,
// which draws none for the switch attribute: a circle inset 2 points in
// the track, in the element's colour, at its start or, checked, its end,
// as paint draws it.
func switchThumb(d *ui.Document, n *dom.Node) string {
	if n.Tag() != dom.Input {
		return ""
	}
	if _, ok := n.Attr("switch"); !ok {
		return ""
	}
	st := dom.Data(n).Style
	if st == nil {
		return ""
	}
	h, ok := st.Size.Height.Resolve(0)
	if !ok || st.Size.Height.Percent != 0 {
		return ""
	}
	pad := func(l style.Length) float32 { v, _ := l.Resolve(0); return v }
	if st.BoxSizing == style.BorderBox {
		h -= st.Border.Top + st.Border.Bottom + pad(st.Padding.Top) + pad(st.Padding.Bottom)
	}
	r := max(h/2-2, 1)
	svg := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%g" height="%g">`+
		`<circle cx="%g" cy="%g" r="%g" fill="%s" fill-opacity="%g"/></svg>`,
		h, h, h/2, h/2, r, st.Color.Hex(), st.Color.A)
	side := "left"
	if d.Checked(n) {
		side = "right"
	}
	return fmt.Sprintf("background-image: url(data:image/svg+xml;base64,%s); background-repeat: no-repeat; "+
		"background-origin: content-box; background-position: %s center",
		base64.StdEncoding.EncodeToString([]byte(svg)), side)
}

// nativeClass is the user agent's pseudo-class of a control's own look,
// which holds while its appearance isn't none: on the page, while it has
// no data-ember-bare.
const (
	nativeClass = "-ember-native"
	ownLookOn   = ":not([data-ember-bare])"
)

// ownLook keeps Chrome from drawing its own controls, which the user
// agent's rules of their look draw in ui's: a page's appearance is read
// in ui's cascade, and a control of appearance: none is marked
// data-ember-bare. Its style element can be taken out to read Chrome's
// own appearance.
const ownLook = "</style>\n<style id=\"ember-own-look\">\n" +
	"button, input, select, textarea, progress { appearance: none !important; }\n" + bareBars + "</style>\n<style>\n"

// bare reports whether n is a control of appearance: none, which draws
// from its CSS alone.
func bare(n *dom.Node) bool {
	st := dom.Data(n).Style
	t := n.Tag()
	control := t == dom.Input || t == dom.Button || t == dom.Select || t == dom.Textarea || t == dom.Progress ||
		t == dom.Meter
	return control && st != nil && !st.Appearance.Native()
}

// accentLook draws a checked checkbox or radio of ui's own look whose
// accent-color is set as paint does: the checkbox filled with it, the
// radio's ring in it.
func accentLook(d *ui.Document, n *dom.Node) string {
	st := dom.Data(n).Style
	typ, _ := n.Attr("type")
	if st == nil || st.AccentAuto || !st.Appearance.Native() || n.Tag() != dom.Input || !d.Checked(n) {
		return ""
	}
	a := st.AccentColor.Hex()
	switch strings.ToLower(typ) {
	case checkboxType:
		return "background-color: " + a + " !important; border-color: " + a + " !important"
	case "radio":
		return "border-color: " + a + " !important"
	}
	return ""
}

// checkboxType is a checkbox's type attribute.
const checkboxType = "checkbox"

// bareBars takes Chrome's own look off ranges and progress bars, which
// barLook draws as paint does. Its selectors weigh nothing (:where), so
// an author's rules on the same elements win as they do in ember.
const bareBars = `:where(input[type="range"]:not([data-ember-bare])) { appearance: none; background: none; }
:where(input[type="range"]:not([data-ember-bare]))::-webkit-slider-runnable-track { background: none; }
:where(input[type="range"]:not([data-ember-bare]))::-webkit-slider-thumb { appearance: none; width: 0; height: 0; }
:where(progress:not([data-ember-bare])) { appearance: none; background: none; }
:where(progress:not([data-ember-bare]))::-webkit-progress-bar,
:where(progress:not([data-ember-bare]))::-webkit-progress-value { background: none; }
`

// barLook is the declarations that draw a range or a progress bar in
// Chrome as ember's convention has paint draw them: its background-color
// the track, its accent-color the fill to the value, and a range's thumb,
// 14 across on the value, in its color, the accent while hovered. A
// range's is an SVG border image outset each side by the thumb's
// overhang, in place of any border; a progress bar's an SVG background
// in its content box, inside its border. Its size is the laid-out Rect,
// so a document exported before any layout shows Chrome's bare bars.
func barLook(d *ui.Document, n *dom.Node) string {
	st := dom.Data(n).Style
	if st == nil || bare(n) {
		return ""
	}
	typ, _ := n.Attr("type")
	isRange := n.Tag() == dom.Input && strings.EqualFold(typ, "range")
	if !isRange && n.Tag() != dom.Progress {
		return ""
	}
	r := n.Box()
	pad := func(l style.Length) float32 { v, _ := l.Resolve(r.W); return v }
	x0 := st.Border.Left + pad(st.Padding.Left)
	y0 := st.Border.Top + pad(st.Padding.Top)
	w := r.W - x0 - st.Border.Right - pad(st.Padding.Right)
	h := r.H - y0 - st.Border.Bottom - pad(st.Padding.Bottom)
	if w <= 0 || h <= 0 {
		return ""
	}
	fill := func(c color.Color) string {
		return fmt.Sprintf(`fill="%s" fill-opacity="%g"`, c.Hex(), c.A)
	}
	track := st.Background
	if track.A <= 0 {
		track = st.Color
		track.A *= 0.25
	}
	accent := st.AccentColor
	if st.AccentAuto {
		accent = st.Color
	}
	var svg strings.Builder
	var out float32
	if isRange {
		f := rangeFraction(d, n)
		th, thumb := min(6, h), min(14, h)
		out = thumb / 2
		ink := st.Color
		if n.State().Has(dom.Hover) {
			ink = accent
		}
		fmt.Fprintf(&svg, `<svg xmlns="http://www.w3.org/2000/svg" width="%g" height="%g">`, w+2*out, h)
		fmt.Fprintf(&svg, `<rect x="%g" y="%g" width="%g" height="%g" rx="%g" %s/>`, out, (h-th)/2, w, th, th/2, fill(track))
		if f > 0 {
			fmt.Fprintf(&svg, `<rect x="%g" y="%g" width="%g" height="%g" rx="%g" %s/>`,
				out, (h-th)/2, w*f, th, th/2, fill(accent))
		}
		fmt.Fprintf(&svg, `<circle cx="%g" cy="%g" r="%g" %s/>`, out+w*f, h/2, out, fill(ink))
	} else {
		fmt.Fprintf(&svg, `<svg xmlns="http://www.w3.org/2000/svg" width="%g" height="%g">`, w, h)
		fmt.Fprintf(&svg, `<rect x="0" y="0" width="%g" height="%g" %s/>`, w, h, fill(track))
		if f, ok := progressFraction(n); ok && f > 0 {
			fmt.Fprintf(&svg, `<rect x="0" y="0" width="%g" height="%g" %s/>`, w*f, h, fill(accent))
		}
	}
	svg.WriteString("</svg>")
	url := "url(data:image/svg+xml;base64," + base64.StdEncoding.EncodeToString([]byte(svg.String())) + ")"
	if !isRange {
		return "background: " + url + " no-repeat content-box"
	}
	// A range keeps its background-color, as the cascade computed it,
	// but clipped to its text, of which it has none, so only the
	// border-image draws.
	return fmt.Sprintf("-webkit-background-clip: text; background-clip: text; border-image-source: %s; "+
		"border-image-slice: 0 fill; border-image-width: 0; border-image-outset: 0 %gpx", url, out)
}

// rangeFraction is where a range's value lies between its min and max,
// 0 to 100 by default, as the Document has it now.
func rangeFraction(d *ui.Document, n *dom.Node) float32 {
	lo, hi := attrFloat(n, "min", 0), attrFloat(n, "max", 100)
	v, err := strconv.ParseFloat(d.Value(n), 64)
	if err != nil || hi <= lo {
		return 0
	}
	return float32(min(max((v-lo)/(hi-lo), 0), 1))
}

// progressFraction is a progress bar's value over its max (1 by default),
// not ok when it has no value: indeterminate.
func progressFraction(n *dom.Node) (float32, bool) {
	s, ok := n.Attr("value")
	if !ok {
		return 0, false
	}
	v, err := strconv.ParseFloat(s, 64)
	hi := attrFloat(n, "max", 1)
	if err != nil || hi <= 0 {
		return 0, false
	}
	return float32(min(max(v/hi, 0), 1)), true
}

func attrFloat(n *dom.Node, name string, def float64) float64 {
	if s, ok := n.Attr(name); ok {
		if v, err := strconv.ParseFloat(s, 64); err == nil {
			return v
		}
	}
	return def
}

// placeholderColor is the ::placeholder colour of a select showing its
// placeholder, which Chrome doesn't give a select.
func placeholderColor(d *ui.Document, n *dom.Node) (string, bool) {
	if n.Tag() != dom.Select || d.Selected(n) >= 0 {
		return "", false
	}
	if _, ok := n.Attr("placeholder"); !ok {
		return "", false
	}
	st := dom.Data(n).Style
	if st == nil || !st.HasPlaceholder {
		return "", false
	}
	c := st.PlaceholderColor
	if c.A >= 1 {
		return c.Hex(), true
	}
	s := c.SRGB()
	return fmt.Sprintf("rgba(%d, %d, %d, %.3g)", int(s[0]*255+0.5), int(s[1]*255+0.5), int(s[2]*255+0.5), c.A), true
}

// pseudo are the pseudo-classes Chrome can't be put in by a page, which
// export writes as data-ember-<name> attributes and selectors also match.
var pseudo = []struct { //nolint:gochecknoglobals // a read-only table
	name  string
	state dom.State
}{{"hover", dom.Hover}, {"active", dom.Active}, {"focus-visible", dom.FocusVisible}, {"focus-within", dom.FocusWithin},
	{"focus", dom.Focus}, {"target", dom.Target}}

var pseudoClasses = regexp.MustCompile(`:(hover|active|focus-visible|focus-within|focus|target)\b`)

// states makes css's :hover, :active, :focus, :focus-visible,
// :focus-within and :target also match the elements export marks as in
// that state, so Chrome draws a pointer's or the focus's look, or a
// fragment's, in a still page.
func states(css string) string {
	return pseudoClasses.ReplaceAllString(css, ":is(:$1, [data-ember-$1])")
}

// showModals opens with showModal the dialogs the Document has open modal,
// in the order of their layers, which stacks them in the top layer as
// paint does, and gives them their ::backdrop.
const (
	showModals = `<script>addEventListener("DOMContentLoaded", () => ` +
		`[...document.querySelectorAll("dialog[data-ember-modal]")]` +
		`.sort((a, b) => a.dataset.emberModal - b.dataset.emberModal).forEach(d => d.showModal()));</script>` + "\n"
)

// place is where p stands for Chrome: the index of its text node among
// its parent's child nodes, text nodes side by side being one there, and
// its offset in that node in UTF-16 units.
func place(d *ui.Document, p ui.Position) string {
	index, offset, text := 0, 0, false
	for _, c := range p.Node.Parent().Children() {
		if !c.IsText() {
			index, offset, text = index+1, 0, false
			continue
		}
		if !text {
			index, text = index+1, true
		}
		s := []rune(layout.Translated(c, tr(d)))
		if c == p.Node {
			offset += len(utf16.Encode(s[:min(p.Offset, len(s))]))
			break
		}
		offset += len(utf16.Encode(s))
	}
	return fmt.Sprintf("%d %d", index-1, offset)
}

func hasAttr(n *dom.Node, name string) bool {
	_, ok := n.Attr(name)
	return ok
}

// selectText selects in Chrome the text the Document has selected, and
// the characters of its focused field, or of one to be focused, once the page is parsed and again
// once it has loaded.
const selectText = `<script>const emberSelect = () => { ` +
	`const end = name => { const e = document.querySelector("[data-ember-" + name + "]"); if (!e) return null; ` +
	`const [i, o] = e.dataset["ember" + name[0].toUpperCase() + name.slice(1)].split(" ").map(Number); ` +
	`return [e.childNodes[i], o]; };
const a = end("anchor"), f = end("focus"); if (a && f) getSelection().setBaseAndExtent(a[0], a[1], f[0], f[1]);
const field = document.querySelector("[data-ember-field-selection]"); if (field) { field.focus(); ` +
	`field.setSelectionRange(...field.dataset.emberFieldSelection.split(" ").map(Number)); } };
addEventListener("DOMContentLoaded", emberSelect); ` +
	`addEventListener("load", () => document.fonts.ready.then(emberSelect));</script>` + "\n"

// scrollTo scrolls each scroller to the offset the Document has it at,
// once the page is parsed and again once it has loaded, its fonts with it.
const scrollTo = `<script>const emberScroll = () => document.querySelectorAll("[data-ember-scroll]").forEach(e => { ` +
	`const [x, y] = e.dataset.emberScroll.split(" ").map(Number); e.scrollLeft = x; e.scrollTop = y; });
addEventListener("DOMContentLoaded", emberScroll); ` +
	`addEventListener("load", () => document.fonts.ready.then(emberScroll));</script>` + "\n"

// drawSnapshots draws each canvas's snapshot into it at the device scale:
// its used size held as its CSS size, so the bitmap set to its content box
// in device pixels leaves the layout alone.
const drawSnapshots = `<script>addEventListener("load", () => ` +
	`document.querySelectorAll("canvas[data-ember-snapshot]").forEach(c => {
const s = getComputedStyle(c);
c.style.width = s.width; c.style.height = s.height;
const w = c.clientWidth - parseFloat(s.paddingLeft) - parseFloat(s.paddingRight), ` +
	`h = c.clientHeight - parseFloat(s.paddingTop) - parseFloat(s.paddingBottom);
c.width = Math.round(w * devicePixelRatio); c.height = Math.round(h * devicePixelRatio);
const img = new Image(); img.onload = () => c.getContext("2d").drawImage(img, 0, 0, c.width, c.height); ` +
	`img.src = c.dataset.emberSnapshot;
}));</script>` + "\n"

// listTypes has the type attribute of a list or a list item number it in
// Chrome as HTML maps it, over the user-agent sheet's list-style-type,
// which the page holds as the author's.
const listTypes = "ol[type], ul[type], li[type] { list-style-type: revert-layer; }\n"

// truncateStart has Chrome's ellipsis cut the start of a truncate-start
// element, as layout does: right to left outside, its content isolated
// left to right inside (a bdi marked data-ember-implied, which the tree
// doesn't have) so a path's slashes stay where they are. It comes before
// the author's sheets, which may override it.
const truncateStart = ".truncate-start { direction: rtl; text-align: left; }\n"

// classicScrollbars has Chrome reserve the 15-point gutter css gives
// scrolling boxes (style.Box.ScrollbarWidth), not overlay scrollbars.
const classicScrollbars = ownBars + "::-webkit-scrollbar { width: 15px; height: 15px; }\n" + thumbs

// stillViewport has the page itself show no bar, as the Document of a
// window does unless its viewport scrolls (ui.ScrollingViewport).
const stillViewport = "html::-webkit-scrollbar { display: none; }\n"

// ScrollingViewport is the attribute html carries in the page of a
// Document whose viewport scrolls: Chrome scrolls it as ui does, to the
// offset data-ember-scroll gives html, so whatever opens the page must
// leave the viewport its size and its bar.
const ScrollingViewport = "data-ember-scrolling-viewport"

// overlayScrollbars has Chrome keep no room for the bars of a Document
// with ui.OverlayScrollbars, whatever the machine's own are; Chrome then
// draws none, so a picture of the page lacks the thumbs paint lays over
// the content.
const overlayScrollbars = "::-webkit-scrollbar { width: 0; height: 0; }\n"

// thumbs draws Chrome's scrollbar thumbs as paint draws ember's: 6 points
// wide, 2 from the far edge and 4 from the track's ends, rounded by 3, in
// white at a quarter; the transparent borders place it in the gutter.
const thumbs = ownBars + "::-webkit-scrollbar-thumb { background-color: rgba(255, 255, 255, 0.25); " +
	"background-clip: padding-box; border: solid transparent; }\n" +
	ownBars + "::-webkit-scrollbar-thumb:vertical { border-width: 4px 2px 4px 7px; " +
	"border-radius: 10px 5px 5px 10px / 7px; }\n" +
	ownBars + "::-webkit-scrollbar-thumb:horizontal { border-width: 7px 4px 2px 4px; " +
	"border-radius: 7px / 10px 10px 5px 5px; }\n" +
	ownBars + "::-webkit-scrollbar-corner { background: transparent; }\n"

// ownBars keeps ui's look off the bars an element's sheets style
// (data-ember-bars), which Chrome draws from those sheets alone, as paint does.
const ownBars = ":not([data-ember-bars])"
