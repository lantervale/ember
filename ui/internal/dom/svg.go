package dom

import (
	"html"
	"math"
	"net/url"
	"strconv"
	"strings"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/ui/internal/style"
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
// a data URL, drawn in the colour of the element's text; the picture its
// content property names in place of all of these (ContentPicture). ok
// is false for an element that names none.
func Picture(n *Node) (string, bool) {
	if src, ok := ContentPicture(n); ok {
		return src, true
	}
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

// ContentPicture is the picture an element's content property replaces
// it by, as Chrome replaces it: a content of one url() alone on an
// element that isn't a form control, a frame or a canvas. ok is false
// for any other element, and for a pseudo-element's box.
func ContentPicture(n *Node) (string, bool) {
	st := n.pipe.Style
	if st == nil || st.Content.Kind != style.ContentItems || len(st.Content.Items) != 1 ||
		st.Content.Items[0].Kind != style.ContentImage || n.gen != NoPseudo {
		return "", false
	}
	switch n.tag { //nolint:exhaustive // the elements ui draws whole
	case Input, Textarea, Select, Progress, Meter, Canvas, Iframe, Svg, TextTag, HTML:
		return "", false
	}
	return st.Content.Items[0].Text, true
}

// Choose has the img n show url in place of its src, "" for no
// picture, a picture of density device pixels a point, chosen by the
// source element source, or nil: n is laid out again when that changes,
// and styled again when the source does, whose width and height may
// hint its size (DimensionSource). It reports whether the source did.
func Choose(n *Node, url string, density float32, source *Node) bool {
	p := &n.pipe
	if !p.choosing || p.chosen != url || p.density != density {
		p.chosen, p.density, p.choosing = url, density, true
		n.mark(DirtyLayout)
	}
	if p.source == source {
		return false
	}
	p.source = source
	n.mark(DirtyStyle | DirtyLayout)
	return true
}

// Unchoose has n show its src again.
func Unchoose(n *Node) {
	if n.pipe.choosing {
		n.pipe.chosen, n.pipe.density, n.pipe.choosing, n.pipe.source = "", 0, false, nil
		n.mark(DirtyStyle | DirtyLayout)
	}
}

// ChosenSource is the source element the img n chose its picture by,
// nil when none.
func ChosenSource(n *Node) *Node { return n.pipe.source }

// DimensionSource is the element whose width and height attributes size
// the img n, as Chrome's: the source element it chose its picture by
// when that has either, else n.
func DimensionSource(n *Node) *Node {
	if s := n.pipe.source; s != nil {
		if _, ok := s.Attr("width"); ok {
			return s
		}
		if _, ok := s.Attr("height"); ok {
			return s
		}
	}
	return n
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
