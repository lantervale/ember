package ui

import (
	"errors"
	"image"
	"strings"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/css"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/srcset"
	"github.com/lantervale/ember/ui/internal/style"
)

// choices are the imgs of a Document that choose their picture
// (decision 0115): by a srcset, or in a picture.
type choices struct {
	imgs []*Node
	// stale is set when the tree was walked since they last chose,
	// media is what they chose for, and waiting whether one keeps its
	// picture until the one chosen in its place lands.
	stale   bool
	media   css.Media
	waiting bool
	// dppx is the density each img last chose for: Chrome's keeps it
	// until something has it choose again.
	dppx map[*Node]float32
}

// choosesPicture reports whether the img n chooses its picture: one
// that loads lazily shows none until it nears the viewport.
func choosesPicture(n *Node) bool {
	if _, ok := n.Attr("srcset"); ok || lazy(n) {
		return true
	}
	p := n.Parent()
	return p != nil && p.Tag() == dom.PictureElement
}

// choosePictures has the imgs that choose their picture choose it for
// m, when it or the tree changed since they last did, or one waits for
// its picture, or the media moved (viewportMoves, featuresMove): each at the
// density it last chose for, unless the media moved so as to have it
// choose again. Once laid out,
// only those whose sizes say auto choose, by the width they were laid
// out at; until then they show none. An img that loads lazily shows
// none until it nears the viewport (nearing).
// It reports whether one chose by another source than before, which
// restyles it, or neared the viewport.
func (d *Document) choosePictures(m css.Media, laid bool) bool {
	c := &d.choices
	restyle, resized, other := false, false, false
	if !laid {
		resized, other = viewportMoves(c.media, m), featuresMove(c.media, m)
		if !c.stale && !resized && !other && !c.waiting {
			c.media = m
			return false
		}
		c.stale, c.media, c.waiting = false, m, false
	} else {
		restyle = d.nearing(geom.V2(m.Width, m.Height))
	}
	kept := c.dppx
	c.dppx = make(map[*Node]float32, len(c.imgs))
	for _, n := range c.imgs {
		at := m
		if dppx, ok := kept[n]; ok && !resized && (!other || !mediaSources(n)) {
			at.Resolution = dppx
		}
		c.dppx[n] = at.Resolution
		if lazy(n) && !dom.Data(n).Near {
			if !laid {
				dom.Choose(n, "", 0, nil)
			}
			continue
		}
		auto := float32(-1)
		if dom.AutoSizes(n) && lazy(n) {
			b, _ := dom.Data(n).Placed.(*box.Box)
			_, chose := dom.Chosen(n)
			switch {
			case b != nil:
				auto = b.ContentRect().W
			case !laid && !chose:
				restyle = dom.Choose(n, "", 0, nil) || restyle
				continue
			}
		} else if laid {
			continue
		}
		restyle = d.choosePicture(n, at, auto) || restyle
	}
	return restyle
}

// Chrome's margins round the viewport, and round each scroll container
// an img is in, within which an img that loads lazily is near enough to
// load: across and along its lines, in points, as a desktop Chrome on a
// fast network has them (measured on Chrome for Testing 156).
const nearAcross, nearAlong = 625, 1250

// nearing has each img that loads lazily and is laid out within the
// margins round the viewport, of size, and round every scroll container
// it is in, choose its picture, as Chrome's do; reporting whether one
// did.
func (d *Document) nearing(size geom.Vec2) bool {
	near := false
	for _, n := range d.choices.imgs {
		if !lazy(n) || dom.Data(n).Near {
			continue
		}
		b, _ := dom.Data(n).Placed.(*box.Box)
		if b == nil || !reaches(b, size) {
			continue
		}
		dom.Data(n).Near = true
		dom.Choose(n, "", 0, nil)
		dom.Data(n).Dirty |= dom.DirtyLayout
		d.choices.stale, near = true, true
	}
	return near
}

// reaches reports whether b lies within the margins round each scroll
// container it is in and round the viewport of size: its edges inside
// theirs, the far ones touching them, so that a box of no size counts.
func reaches(b *box.Box, size geom.Vec2) bool {
	r := b.Rect()
	inside := func(c geom.Rect) bool {
		c = geom.Rect{X: c.X - nearAcross, Y: c.Y - nearAlong, W: c.W + 2*nearAcross, H: c.H + 2*nearAlong}
		x, y := max(r.X, c.X), max(r.Y, c.Y)
		right, bottom := min(r.X+r.W, c.X+c.W), min(r.Y+r.H, c.Y+c.H)
		if x >= c.X+c.W || y >= c.Y+c.H || right < x || bottom < y {
			return false
		}
		r = geom.Rect{X: x, Y: y, W: right - x, H: bottom - y}
		return true
	}
	for a := b.Parent; a != nil; a = a.Parent {
		if a.Node == nil || a.Style == nil || !scrollContainer(a.Style.Overflow) || a.Parent == nil {
			continue
		}
		p := a.Rect()
		p = geom.Rect{X: p.X + a.Border.Left, Y: p.Y + a.Border.Top,
			W: p.W - a.Border.Horizontal() - a.Scrollbar.X, H: p.H - a.Border.Vertical() - a.Scrollbar.Y}
		if !inside(p) {
			return false
		}
	}
	return inside(geom.Rect{W: size.X, H: size.Y})
}

// scrollContainer reports whether an overflow makes a scroll container.
func scrollContainer(o style.Size[style.Overflow]) bool {
	sc := func(v style.Overflow) bool { return v != style.Visible && v != style.Clip }
	return sc(o.Width) || sc(o.Height)
}

// viewportMoves reports whether the viewport resized from was to now,
// which has every img choose again, as Chrome's do.
func viewportMoves(was, now css.Media) bool {
	return was.Width != now.Width || was.Height != now.Height
}

// featuresMove reports whether a media feature other than the viewport's
// size and the density changed from was to now, which has those in a
// picture with a source of media choose again, as Chrome's do. Chrome
// for Testing 156 keeps the pictures it chose when the density alone
// changes.
func featuresMove(was, now css.Media) bool {
	was.Width, was.Height, was.Resolution, was.Inactive = now.Width, now.Height, now.Resolution, now.Inactive
	return was != now
}

// mediaSources reports whether the img n is in a picture with a source
// of media.
func mediaSources(n *Node) bool {
	p := n.Parent()
	if p == nil || p.Tag() != dom.PictureElement {
		return false
	}
	for _, s := range p.Children() {
		if _, ok := s.Attr("media"); ok && s.Tag() == dom.Source {
			return true
		}
	}
	return false
}

// lazy reports whether the img n loads its picture lazily, which lets
// its sizes say auto.
func lazy(n *Node) bool {
	v, _ := n.Attr("loading")
	return strings.EqualFold(v, "lazy")
}

// choosePicture has the img n show its picture for m, auto the width
// it was laid out at, when its sizes may say auto: once landed when it
// shows another, which it keeps meanwhile. It reports whether n chose
// by another source than before.
func (d *Document) choosePicture(n *Node, m css.Media, auto float32) bool {
	url, density, source := d.choice(n, m, auto)
	shown, chose := dom.Chosen(n)
	if chose && url != shown && d.hasPicture(shown) && d.landing(url) {
		d.choices.waiting = true
		return false
	}
	return dom.Choose(n, url, float32(density), source)
}

// choice is the picture the img n shows in m, its density and the
// source it is chosen by: the first source before it in its picture
// whose srcset offers one, whose media holds and whose type ui reads,
// else none, the img's own srcset with its src, else its src. A denser
// candidate d has cached wins.
func (d *Document) choice(n *Node, m css.Media, auto float32) (string, float64, *Node) {
	dppx := 1.0
	if m.Resolution > 0 {
		dppx = float64(m.Resolution)
	}
	if p := n.Parent(); p != nil && p.Tag() == dom.PictureElement {
		for _, s := range p.Children() {
			if s == n {
				break
			}
			if s.Tag() != dom.Source {
				continue
			}
			text, _ := s.Attr("srcset")
			set := srcset.Parse(text)
			media, hasMedia := s.Attr("media")
			typ, hasType := s.Attr("type")
			if len(set) == 0 || hasMedia && !css.ParseMediaQueries(media).Hold(m) || hasType && !readsType(typ) {
				continue
			}
			url, density := srcset.Choose(set, "", slotWidth(s, m, auto), dppx, d.cached)
			return url, density, s
		}
	}
	src, _ := n.Attr("src")
	text, ok := n.Attr("srcset")
	if !ok {
		return src, 1, nil
	}
	url, density := srcset.Choose(srcset.Parse(text), src, slotWidth(n, m, auto), dppx, d.cached)
	return url, density, nil
}

// slotWidth is the width the sizes of n, an img or a source, give its
// picture in m.
func slotWidth(n *Node, m css.Media, auto float32) float64 {
	sizes, _ := n.Attr("sizes")
	return float64(css.ParseSizes(sizes).Width(m, auto))
}

// cached reports whether d needs not fetch url's picture, as Chrome
// answers for its memory cache: a data URL, or a picture d has.
func (d *Document) cached(url string) bool {
	return len(url) >= 5 && strings.EqualFold(url[:5], "data:") || d.hasPicture(url)
}

// hasPicture reports whether url's picture is one d has.
func (d *Document) hasPicture(url string) bool {
	_, vector := d.vectors[url]
	return d.images[url] != nil || vector
}

// landing reports whether url's picture is on its way, asking for it.
func (d *Document) landing(url string) bool {
	if url == "" {
		return false
	}
	d.image(url)
	return d.fetching.onWay[url]
}

// The first bytes of a file of each format of pixels a source may name.
const (
	pngStart  = "\x89PNG\r\n\x1a\n"
	jpegStart = "\xff\xd8"
	icoStart  = "\x00\x00\x01\x00"
	bmpStart  = "BM\x00\x00\x00\x00\x00\x00\x00\x00"
)

// pixelFormats are the media types of pictures of pixels a source may
// name, by the first bytes of a file of theirs.
//
//nolint:gochecknoglobals // lookup table
var pixelFormats = map[string]string{
	"image/png": pngStart, "image/apng": pngStart, "image/x-png": pngStart,
	"image/jpeg": jpegStart, "image/jpg": jpegStart, "image/pjpeg": jpegStart,
	"image/gif":    "GIF8",
	"image/webp":   "RIFF\x00\x00\x00\x00WEBPVP8",
	"image/x-icon": icoStart, "image/vnd.microsoft.icon": icoStart,
	"image/bmp": bmpStart, "image/x-ms-bmp": bmpStart,
}

// readsType reports whether ui reads pictures of the media type t, its
// parameters and case aside, as a source's type asks: a registered
// format of shapes, or one of pixels the image package was given a
// decoder of; "" names any.
func readsType(t string) bool {
	t, _, _ = strings.Cut(t, ";")
	t = strings.ToLower(strings.TrimSpace(t))
	if t == "" || vectorType(t) {
		return true
	}
	magic, ok := pixelFormats[t]
	if !ok {
		return false
	}
	_, _, err := image.DecodeConfig(strings.NewReader(magic))
	return !errors.Is(err, image.ErrFormat)
}
