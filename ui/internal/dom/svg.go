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

// Picture is the address of the picture an element shows: an img's
// src, or the one Choose gave it, and for an svg element its markup as
// a data URL, drawn in the colour of the element's text. ok is false
// for an element that names none.
func Picture(n *Node) (string, bool) {
	if n.tag != Svg {
		if n.pipe.choosing {
			return n.pipe.chosen, n.pipe.chosen != ""
		}
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

// Choose has the img n show url in place of its src, "" for no
// picture, a picture of density device pixels a point: n is laid out
// again when that changes.
func Choose(n *Node, url string, density float32) {
	p := &n.pipe
	if !p.choosing || p.chosen != url || p.density != density {
		p.chosen, p.density, p.choosing = url, density, true
		n.mark(DirtyLayout)
	}
}

// Unchoose has n show its src again.
func Unchoose(n *Node) {
	if n.pipe.choosing {
		n.pipe.chosen, n.pipe.density, n.pipe.choosing = "", 0, false
		n.mark(DirtyLayout)
	}
}

// Chosen is the picture Choose last gave n, false while it shows its
// src.
func Chosen(n *Node) (string, bool) { return n.pipe.chosen, n.pipe.choosing }

// AutoSizes reports whether the sizes of the img n start with auto,
// which has it sized as Chrome's sheet sizes it, with
// contain-intrinsic-size: 300px 150px.
func AutoSizes(n *Node) bool {
	s, _ := n.Attr("sizes")
	return len(s) >= 4 && strings.EqualFold(s[:4], "auto") && (len(s) == 4 || s[4] == ',')
}

// Density is the device pixels a point of the picture n shows: what
// Choose gave it, else 1.
func Density(n *Node) float32 {
	if n.pipe.choosing && n.pipe.density > 0 {
		return n.pipe.density
	}
	return 1
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
