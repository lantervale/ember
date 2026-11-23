// Package svg reads SVG pictures and draws them into images: parse a file
// once into a Picture, a plain model of paths and paints, then Rasterise
// it at any size. It is pure Go and reads nothing but the bytes it is
// given: no file, no network, whatever the picture refers to.
//
// It draws the static part of SVG that icons and logos are made of:
// paths and the basic shapes, groups, use, symbols, nested svg elements,
// transforms, solid colours, linear and radial gradients, patterns,
// strokes with caps, joins and dashes, markers, fill rules, opacity, clip
// paths (of a clipPath or a basic shape), masks (of a mask or a CSS
// gradient), filters, blend modes, strokes that do not scale, images
// held in data: addresses, text in Go's
// faces (along paths too), style sheets of simple selectors, the view box
// and preserveAspectRatio, in UTF-8 or UTF-16. What a file uses beyond
// that (foreignObject, animation…) is not drawn, and
// Picture.Unsupported names it, so a caller can warn or fall back instead
// of showing a wrong picture.
package svg

import (
	"errors"
	"fmt"
	"image/color"
	"io/fs"
	"sort"
)

// The limits a picture is read and drawn within. A file past one of them
// is an error, never a slow or huge answer.
const (
	// MaxBytes is the largest file read.
	MaxBytes = 16 << 20
	// MaxElements is the most elements a file may have.
	MaxElements = 100_000
	// MaxDepth is how deep its elements may nest.
	MaxDepth = 256
	// MaxShapes is the most shapes a picture may draw, each use of an
	// element counted.
	MaxShapes = 100_000
	// MaxSegments is the most path steps a picture may draw, each use
	// counted.
	MaxSegments = 2_000_000
	// MaxUseDepth is how deep use elements may refer to one another.
	MaxUseDepth = 32
	// MaxRules is the most style rules read; the rest are dropped.
	MaxRules = 4096
	// MaxSide is the longest side Rasterise draws, in pixels.
	MaxSide = 16384
	// MaxPixels is the most pixels Rasterise draws.
	MaxPixels = 1 << 25
)

const (
	maxEntities    = 256
	maxEntityBytes = 1 << 20
	maxTransforms  = 64
	maxStops       = 1024
	maxDashes      = 64
	// maxLayers is how deep groups with an opacity get a layer of their
	// own; deeper ones fade each of their shapes instead.
	maxLayers = 4
	// maxFlatPoints is the most points one path is flattened into.
	maxFlatPoints = 1 << 21
	// defaultWidth and defaultHeight are the size CSS gives a picture
	// that asks for none.
	defaultWidth, defaultHeight = 300, 150
)

var (
	// ErrTooLarge is returned for a file over MaxBytes and a raster over
	// MaxSide or MaxPixels.
	ErrTooLarge = errors.New("svg: too large")
	// ErrTooComplex is returned for a file over the other limits.
	ErrTooComplex = errors.New("svg: too complex")
)

// FormatError is returned for bytes that are no SVG picture.
type FormatError struct{ Err error }

func (e *FormatError) Error() string { return "svg: " + e.Err.Error() }
func (e *FormatError) Unwrap() error { return e.Err }

// Align is where a view box that does not fill its viewport sits along
// one axis.
type Align uint8

const (
	// AlignMid centres it, the default.
	AlignMid Align = iota
	// AlignMin puts it at the left or the top.
	AlignMin
	// AlignMax puts it at the right or the bottom.
	AlignMax
)

// Aspect is how a view box fits a viewport of another shape: the
// preserveAspectRatio attribute.
type Aspect struct {
	// Stretch fills the viewport, the two axes scaled apart (none).
	Stretch bool
	X, Y    Align
	// Slice covers the viewport and lets the view box overflow, where
	// meet, the default, shows it whole.
	Slice bool
}

// Picture is a parsed SVG picture: what it draws, as a model another
// renderer could draw too, and how it asks to be sized.
type Picture struct {
	// Items are what the picture draws, the first one underneath.
	Items []Item
	// ViewBox is the part of user space the picture shows; empty when the
	// file has none, and user space then starts at the picture's corner,
	// a unit a pixel of its intrinsic size.
	ViewBox Box
	Aspect  Aspect

	width, height       float64
	hasWidth, hasHeight bool
	unsupported         []string
	currentColor        color.NRGBA
	// transform is the root's, about the viewport's centre in CSS pixels.
	transform Matrix
	nesting   int
	filters   *elementFilters
}

// Option sets how a picture is read.
type Option func(*Picture)

// CurrentColor is the colour currentColor stands for in the file, black
// unless given: the text colour of what shows an icon drawn to take it.
func CurrentColor(c color.NRGBA) Option {
	return func(p *Picture) { p.currentColor = c }
}

// CurrentColor is the colour currentColor stood for when p was read.
func (p *Picture) CurrentColor() color.NRGBA { return p.currentColor }

// NewPicture parses the SVG file data.
func NewPicture(data []byte, opts ...Option) (*Picture, error) {
	return newPicture(data, 0, opts...)
}

func newPicture(data []byte, nesting int, opts ...Option) (*Picture, error) {
	p := &Picture{currentColor: color.NRGBA{A: 255}, nesting: nesting}
	for _, opt := range opts {
		opt(p)
	}
	doc, err := readDocument(data)
	if err != nil {
		return nil, err
	}
	if err = build(doc, p); err != nil {
		return nil, err
	}
	return p, nil
}

// LoadPicture parses the SVG file name in fsys, such as an embed.FS.
func LoadPicture(fsys fs.FS, name string, opts ...Option) (*Picture, error) {
	data, err := fs.ReadFile(fsys, name)
	if err != nil {
		return nil, fmt.Errorf("svg: %w", err)
	}
	p, err := NewPicture(data, opts...)
	if err != nil {
		return nil, fmt.Errorf("svg: %s: %w", name, err)
	}
	return p, nil
}

// Width is the width the picture asks for in CSS pixels: its width
// attribute, else its height by its view box's ratio. ok is false when
// it asks for none, as a file with only a view box or a width in
// percent, which takes the room it is given.
func (p *Picture) Width() (float64, bool) {
	if p.hasWidth {
		return p.width, true
	}
	if ratio, has := p.Ratio(); has && p.hasHeight {
		return p.height * ratio, true
	}
	return 0, false
}

// Height is the height the picture asks for, as Width.
func (p *Picture) Height() (float64, bool) {
	if p.hasHeight {
		return p.height, true
	}
	if ratio, has := p.Ratio(); has && p.hasWidth {
		return p.width / ratio, true
	}
	return 0, false
}

// Ratio is the picture's width over its height: its view box's, else its
// width and height attributes'. ok is false when it has neither, and a
// picture with a Ratio but no Width keeps it at whatever size it is
// given.
func (p *Picture) Ratio() (float64, bool) {
	if !p.ViewBox.Empty() {
		return p.ViewBox.W / p.ViewBox.H, true
	}
	if p.hasWidth && p.hasHeight && p.width > 0 && p.height > 0 {
		return p.width / p.height, true
	}
	return 0, false
}

// Size is the size the picture has where nothing else sizes it, as an
// image element's naturalWidth and naturalHeight: its Width and Height,
// and 300 and 150 for the ones it does not ask for.
func (p *Picture) Size() (float64, float64) {
	w, h := float64(defaultWidth), float64(defaultHeight)
	if v, ok := p.Width(); ok {
		w = v
	}
	if v, ok := p.Height(); ok {
		h = v
	}
	return w, h
}

// Unsupported names what the file uses that Rasterise does not draw,
// sorted: elements ("text", "image"), properties ("clip-path", "mask",
// "filter"), paints ("pattern") and the like. Empty for a file drawn in
// full.
func (p *Picture) Unsupported() []string { return p.unsupported }

func sortedKeys(set map[string]bool) []string {
	if len(set) == 0 {
		return nil
	}
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// ViewTransform takes the picture's user space to a viewport of w×h
// whose corner is the origin: the view box fitted as Aspect asks, then
// moved by the root's transform about the viewport's centre. A picture
// with no view box is stretched from its width and height, and keeps a
// unit a pixel along an axis it gives no size for.
func (p *Picture) ViewTransform(w, h float64) Matrix { return p.viewTransform(w, h, 1) }

// viewTransform is ViewTransform where an axis that keeps its unit, and
// the root's transform, take unit pixels for a CSS pixel.
func (p *Picture) viewTransform(w, h, unit float64) Matrix {
	m := p.fitted(w, h, unit)
	if t := p.transform; t != (Matrix{}) && t != Identity() {
		m = translate(w/2, h/2).Mul(scale(unit, unit)).Mul(t).Mul(scale(1/unit, 1/unit)).
			Mul(translate(-w/2, -h/2)).Mul(m)
	}
	return m
}

func (p *Picture) fitted(w, h, unit float64) Matrix {
	box := p.ViewBox
	if box.Empty() {
		sx, sy := unit, unit
		if p.hasWidth && p.width > 0 {
			sx = w / p.width
		}
		if p.hasHeight && p.height > 0 {
			sy = h / p.height
		}
		return scale(sx, sy)
	}
	return fit(box, p.Aspect, w, h)
}

// fit takes box to a viewport of w×h whose corner is the origin, as a
// asks.
func fit(box Box, a Aspect, w, h float64) Matrix {
	sx, sy := w/box.W, h/box.H
	if !a.Stretch {
		if a.Slice {
			sx = max(sx, sy)
		} else {
			sx = min(sx, sy)
		}
		sy = sx
	}
	share := func(a Align) float64 {
		switch a {
		case AlignMin:
			return 0
		case AlignMax:
			return 1
		case AlignMid:
		}
		return 0.5
	}
	return translate((w-box.W*sx)*share(a.X), (h-box.H*sy)*share(a.Y)).
		Mul(scale(sx, sy)).Mul(translate(-box.X, -box.Y))
}
