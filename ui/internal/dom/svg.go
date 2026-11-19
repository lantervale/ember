package dom

import (
	"html"
	"math"
	"net/url"
	"strconv"
	"strings"

	"github.com/lantervale/ember/color"
)

// Markup is what an svg element holds: the SVG inside it, as text.
func (n *Node) Markup() string { return n.pipe.Markup }

// SetMarkup gives an svg element other SVG to hold, drawn from the next
// frame.
func (n *Node) SetMarkup(markup string) {
	if n.pipe.Markup != markup {
		n.pipe.Markup = markup
		n.mark(DirtyLayout)
	}
}

// Picture is the address of the picture an element shows: an img's src,
// and for an svg element its markup as a data URL, drawn in the colour
// of the element's text. ok is false for an element that names none.
func Picture(n *Node) (string, bool) {
	if n.tag != Svg {
		return n.Attr("src")
	}
	p := &n.pipe
	ink := color.Black
	if p.Style != nil {
		ink = p.Style.Color
	}
	if p.picture == "" || p.pictured != ink {
		p.picture, p.pictured = "data:image/svg+xml,"+url.PathEscape(n.svg(ink)), ink
	}
	return p.picture, true
}

// svg is the element as an SVG file: its attributes but the size, which
// is the element's own, and currentColor standing for ink.
func (n *Node) svg(ink color.Color) string {
	var b strings.Builder
	b.WriteString(`<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink"`)
	rgb := ink.SRGB()
	channel := func(v float32) string { return strconv.Itoa(int(math.Round(float64(v) * 255))) }
	b.WriteString(` color="rgba(` + channel(rgb[0]) + "," + channel(rgb[1]) + "," + channel(rgb[2]) + "," +
		strconv.FormatFloat(float64(ink.A), 'f', -1, 32) + `)"`)
	for _, a := range n.attrs {
		switch a.Name {
		case "width", "height", "xmlns", "xmlns:xlink", "color":
			continue
		}
		b.WriteString(" " + a.Name + `="` + html.EscapeString(a.Value) + `"`)
	}
	if n.style != "" {
		b.WriteString(` style="` + html.EscapeString(n.style) + `"`)
	}
	b.WriteString(">" + n.pipe.Markup + "</svg>")
	return b.String()
}
