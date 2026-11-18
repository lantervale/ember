package layout

import (
	"strconv"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/dom"
)

// defaultWidth and defaultHeight are the size CSS gives a replaced
// element whose picture asks for none.
const defaultWidth, defaultHeight = 300, 150

// brokenIcon is the side of the icon an img that didn't load shows, as Chrome's
// (paint.BrokenIcon).
const brokenIcon = 16

// buildImage makes an img a replaced box, sized by its picture. One whose picture
// Images doesn't find is sized as Chrome sizes a broken image: its icon and its alt
// text on a line, the icon alone without alt text, nothing with an empty one.
func (t *tree) buildImage(n *node, e *dom.Node) {
	n.replaced = true
	if t.images == nil {
		return
	}
	src, ok := dom.Picture(e)
	if !ok {
		return
	}
	if shapes, has := t.images.(box.IntrinsicImages); has {
		if in, found := shapes.ImageIntrinsic(src); found {
			n.shape, n.sized = in, true
			if !n.fills() {
				s := in.Natural(geom.V2(defaultWidth, defaultHeight))
				n.natural = size{s.X, s.Y}
			}
			return
		}
	}
	if s, found := t.images.ImageSize(src); found {
		n.natural = size{s.X, s.Y}
		return
	}
	alt, hasAlt := e.Attr("alt")
	switch {
	case !hasAlt:
		n.natural = size{brokenIcon, brokenIcon}
	case alt != "":
		m := t.fonts.Metrics(n.st.Font)
		n.natural = size{brokenIcon + t.fonts.Advance(n.st.Font, alt), max(brokenIcon, m.Ascent+m.Descent+m.LineGap)}
	}
}

// buildCanvas makes a canvas a replaced box of its width and height
// attributes, 300 by 150 where one isn't a number of 0 or more, as in HTML.
func buildCanvas(n *node, e *dom.Node) {
	n.replaced = true
	n.natural = size{attrSize(e, "width", 300), attrSize(e, "height", 150)}
}

func attrSize(e *dom.Node, name string, or float32) float32 {
	s, _ := e.Attr(name)
	if v, err := strconv.ParseFloat(s, 32); err == nil && v >= 0 {
		return float32(v)
	}
	return or
}

// naturalRatio is a replaced box's width over height, none without a picture.
func (n *node) naturalRatio() float32 {
	if n.sized {
		if n.shape.Ratio > 0 {
			return n.shape.Ratio
		}
		return none
	}
	if !n.replaced || n.natural.w <= 0 || n.natural.h <= 0 {
		return none
	}
	return n.natural.w / n.natural.h
}

// fills reports an img whose picture has proportions and asks for no
// size: it is as wide as the room it is given, as Chrome lays it out,
// and adds nothing to its block's own width.
func (n *node) fills() bool {
	return n.sized && n.shape.Ratio > 0 && n.shape.Width <= 0 && n.shape.Height <= 0
}

// naturalIn is the natural size of a replaced node in the room avail: a picture that
// fills takes it.
func (n *node) naturalIn(avail space, pb float32) size {
	if n.fills() {
		return n.filled(avail, pb)
	}
	return n.natural
}

// filled is the natural size of an img that fills, in avail less its
// padding and border.
func (n *node) filled(avail space, pb float32) size {
	if avail.kind != definite {
		return size{}
	}
	w := max(avail.v-pb, 0)
	return size{w, w / n.shape.Ratio}
}

// replacedSize is a replaced box's size with neither side set: its natural size, scaled
// through its ratio to meet the min and max sizes.
func replacedSize(natural size, ratio float32, minSize, maxSize size) size {
	w, h := natural.w, natural.h
	if some(ratio) {
		if some(maxSize.w) && w > maxSize.w {
			w, h = maxSize.w, maxSize.w/ratio
		}
		if some(maxSize.h) && h > maxSize.h {
			w, h = maxSize.h*ratio, maxSize.h
		}
		if some(minSize.w) && w < minSize.w {
			w, h = minSize.w, minSize.w/ratio
		}
		if some(minSize.h) && h < minSize.h {
			w, h = minSize.h*ratio, minSize.h
		}
	}
	return size{w, h}
}

// measureReplaced is a replaced box's content size: its natural size, or the side it is
// given and the other through its ratio.
func measureReplaced(n *node, known size) size {
	s := n.natural
	if r := n.naturalRatio(); some(r) {
		switch {
		case some(known.w) && !some(known.h):
			s = size{known.w, known.w / r}
		case some(known.h) && !some(known.w):
			s = size{known.h * r, known.h}
		}
	}
	return known.or(s)
}
