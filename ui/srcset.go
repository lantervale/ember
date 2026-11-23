package ui

import (
	"errors"
	"image"
	"strings"

	"github.com/lantervale/ember/ui/css"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/srcset"
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
}

// choosesPicture reports whether the img n chooses its picture.
func choosesPicture(n *Node) bool {
	if _, ok := n.Attr("srcset"); ok {
		return true
	}
	p := n.Parent()
	return p != nil && p.Tag() == dom.PictureElement
}

// choosePictures has the imgs that choose their picture choose it for
// m, when it or the tree changed since they last did, or one waits for
// its picture. Once laid out, only those whose sizes say auto choose,
// by the width they were laid out at; until then they show none. It
// reports whether one chose by another source than before, which
// restyles it.
func (d *Document) choosePictures(m css.Media, laid bool) bool {
	c := &d.choices
	if !laid {
		if !c.stale && m == c.media && !c.waiting {
			return false
		}
		c.stale, c.media, c.waiting = false, m, false
	}
	restyle := false
	for _, n := range c.imgs {
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
		restyle = d.choosePicture(n, m, auto) || restyle
	}
	return restyle
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
