package layout

import (
	"strconv"
	"strings"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

// defaultWidth and defaultHeight are the size CSS gives a replaced
// element whose picture asks for none.
const defaultWidth, defaultHeight = 300, 150

// brokenIcon is the side of the icon an img that didn't load shows, as Chrome's
// (paint.BrokenIcon).
const brokenIcon = 16

// noRatio is the Ratio of the shape of an img with no picture and no proportions from
// its attributes: its icon and alt text give it a size, never proportions, as Chrome's.
const noRatio = -1

// pictured reports an element laid out as a picture: an img, an svg, or
// one its content property replaces by a picture.
func pictured(e *dom.Node) bool {
	_, replaced := dom.ContentPicture(e)
	return e.Tag() == dom.Img || e.Tag() == dom.Svg || replaced
}

// buildImage makes an img a replaced box, sized by its picture. One whose picture
// Images doesn't find is sized as Chrome sizes a broken image: its icon and its alt
// text on a line, the icon alone without alt text, nothing with an empty one.
func (t *tree) buildImage(n *node, e *dom.Node) {
	n.replaced = true
	n.natural, n.shape, n.sized = pictureOf(e, n.st, t.images, t.fonts)
}

// pictureOf is what the picture of the img e gives its box: its natural size at the
// density it was chosen for, and for a picture that doesn't ask for a size of pixels
// what it asks for, sized being whether e has one. An img whose sizes start with auto is
// 300 by 150 without proportions whatever its picture, as Chrome's sheet has it
// (contain: size; contain-intrinsic-size: 300px 150px).
func pictureOf(e *dom.Node, st *style.Computed, images box.Images, fonts box.Fonts) (size, box.Intrinsic, bool) {
	if images == nil {
		return size{}, box.Intrinsic{}, false
	}
	if e.Tag() == dom.Img && dom.AutoSizes(e) {
		return size{defaultWidth, defaultHeight}, box.Intrinsic{Width: defaultWidth, Height: defaultHeight}, true
	}
	src, ok := dom.Picture(e)
	if !ok {
		return size{}, box.Intrinsic{Ratio: attributeRatio(e)}, false
	}
	density := dom.Density(e)
	if shapes, has := images.(box.IntrinsicImages); has {
		if in, found := shapes.ImageIntrinsic(src); found {
			in.Width, in.Height = in.Width/density, in.Height/density
			if fillsRoom(in) {
				return size{}, in, true
			}
			s := in.Natural(geom.V2(defaultWidth, defaultHeight))
			return size{s.X, s.Y}, in, true
		}
	}
	if s, found := images.ImageSize(src); found {
		return size{s.X / density, s.Y / density}, box.Intrinsic{}, false
	}
	shape := box.Intrinsic{Ratio: attributeRatio(e)}
	if shape.Ratio == 0 {
		shape.Ratio = noRatio
	}
	alt, hasAlt := e.Attr("alt")
	switch {
	case !hasAlt:
		return size{brokenIcon, brokenIcon}, shape, false
	case alt != "":
		m := fonts.Metrics(st.Font)
		return size{brokenIcon + fonts.Advance(st.Font, alt), max(brokenIcon, m.Ascent+m.Descent+m.LineGap)},
			shape, false
	}
	return size{}, shape, false
}

// attributeRatio is the proportions the width and height attributes of an img give it
// while it has no picture, loading or broken, as Chrome's sheet gives them (aspect-ratio:
// auto w / h): 0 for an element of another kind, or but two numbers above 0.
func attributeRatio(e *dom.Node) float32 {
	if e.Tag() != dom.Img {
		return 0
	}
	w, h := attrNumber(e, "width"), attrNumber(e, "height")
	if w <= 0 || h <= 0 {
		return 0
	}
	return w / h
}

// attrNumber is the number an attribute starts with, as HTML reads a dimension, -1 for
// none or a percentage.
func attrNumber(e *dom.Node, name string) float32 {
	v, _ := e.Attr(name)
	v = strings.TrimLeft(v, " \t\r\n\f")
	end := 0
	for end < len(v) && (v[end] >= '0' && v[end] <= '9' || v[end] == '.') {
		end++
	}
	f, err := strconv.ParseFloat(v[:end], 32)
	if err != nil || strings.HasPrefix(v[end:], "%") {
		return -1
	}
	return float32(f)
}

// Landed tells the kept layout that the pictures landed reports arrived since it ran.
// An img of one whose layout read its picture's size or proportions is flagged to be
// laid out again, as a changed element is; any other keeps its box, its node taking the
// picture's size for the layouts to come.
func (r *Retained) Landed(landed func(url string) bool, images box.Images, fonts box.Fonts) {
	for e, n := range r.nodes {
		if !n.replaced || !pictured(e) {
			continue
		}
		src, ok := dom.Picture(e)
		if !ok || !landed(src) {
			continue
		}
		natural, shape, sized := pictureOf(e, n.st, images, fonts)
		switch {
		case natural == n.natural && shape == n.shape && sized == n.sized:
		case n.intrinsic || n.fills() || sized && fillsRoom(shape):
			relay(e)
		default:
			n.natural, n.shape, n.sized = natural, shape, sized
		}
	}
}

// relay flags e to be laid out again: with the generated box it is in and the element
// that box is of, as css flags a change of generated content.
func relay(e *dom.Node) {
	dom.Data(e).Dirty |= dom.DirtyLayout
	p := e.Parent()
	for ; p != nil && e.Generated(); e, p = p, p.Parent() {
		dom.Data(p).Dirty |= dom.DirtyLayout
	}
	for ; p != nil; p = p.Parent() {
		dom.Data(p).Dirty |= dom.DirtyDescendant
	}
}

// buildCanvas makes a canvas a replaced box of its width and height
// attributes, 300 by 150 where one isn't a number of 0 or more, as in HTML.
func buildCanvas(n *node, e *dom.Node) {
	n.replaced = true
	n.natural = size{attrSize(e, "width", 300), attrSize(e, "height", 150)}
}

// buildFrame makes an iframe a replaced box of 300 by 150 whose sides
// don't hold each other: its width and height attributes are hints css
// reads, as Chrome's.
func buildFrame(n *node) {
	n.replaced, n.sized = true, true
	n.natural = size{defaultWidth, defaultHeight}
	n.shape = box.Intrinsic{Width: defaultWidth, Height: defaultHeight}
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
	switch {
	case n.replaced && n.shape.Ratio > 0:
		// an img's attributes, while it has no picture
		return n.shape.Ratio
	case n.shape.Ratio == noRatio:
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
func (n *node) fills() bool { return n.sized && fillsRoom(n.shape) }

// fillsRoom reports a picture that has proportions and asks for no size.
func fillsRoom(in box.Intrinsic) bool { return in.Ratio > 0 && in.Width <= 0 && in.Height <= 0 }

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
// given and the other through its ratio. A picture that fills is as wide as room.
func measureReplaced(n *node, known size, room space) size {
	s := n.natural
	if n.fills() {
		s = n.filled(room, 0)
	}
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
