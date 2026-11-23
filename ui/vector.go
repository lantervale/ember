package ui

import (
	"fmt"
	"image"
	"math"
	"net/http"
	"slices"
	"strings"
	"sync"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/fx"
	"github.com/lantervale/ember/ui/paint"
)

// VectorImage is a picture drawn from shapes, with no pixels of its
// own, as an SVG file is: a Document draws it at each size it is shown
// at. It is read once and drawn from several goroutines.
type VectorImage interface {
	// Width and Height are the size the picture asks for in CSS pixels,
	// and Ratio its width over its height; ok is false for one it
	// doesn't ask for, which it takes from where it is shown.
	Width() (w float64, ok bool)
	Height() (h float64, ok bool)
	Ratio() (r float64, ok bool)
	// Rasterise draws the picture w×h pixels large, alpha premultiplied,
	// for a screen of scale pixels a CSS pixel, which is what a length
	// the file gives in pixels takes. Past what the format draws it
	// returns a smaller image of the same shape, which is stretched.
	Rasterise(w, h int, scale float64) (*image.RGBA, error)
	// Unsupported names what the file uses that Rasterise doesn't draw;
	// a Document warns of it.
	Unsupported() []string
}

type vectorFormat struct {
	mime  string
	is    func(name string, data []byte) bool
	parse func(data []byte) (VectorImage, error)
}

var (
	vectorFormatsMu sync.RWMutex   //nolint:gochecknoglobals // guards vectorFormats
	vectorFormats   []vectorFormat //nolint:gochecknoglobals // the registry formats add themselves to when imported
)

// RegisterVectorFormat registers a format of pictures drawn from shapes,
// of the media type mime: is reports whether a file is one, by the name
// it was asked for by or by how its bytes start, and parse reads it. An
// img's src and a url() naming such a file then show it, drawn at the
// size it takes on the screen. Formats register themselves when
// imported:
//
//	import _ "github.com/lantervale/ember/ui/svg"
func RegisterVectorFormat(
	mime string, is func(name string, data []byte) bool, parse func(data []byte) (VectorImage, error),
) {
	vectorFormatsMu.Lock()
	defer vectorFormatsMu.Unlock()
	vectorFormats = append(vectorFormats, vectorFormat{mime, is, parse})
}

// vectorType reports whether a registered format of shapes is of the
// media type mime.
func vectorType(mime string) bool {
	vectorFormatsMu.RLock()
	defer vectorFormatsMu.RUnlock()
	return slices.ContainsFunc(vectorFormats, func(f vectorFormat) bool { return f.mime == mime })
}

// vectorFormatOf is the registered format the file name with these bytes
// is of.
func vectorFormatOf(name string, data []byte) (vectorFormat, bool) {
	vectorFormatsMu.RLock()
	defer vectorFormatsMu.RUnlock()
	for _, f := range vectorFormats {
		if f.is(name, data) {
			return f, true
		}
	}
	return vectorFormat{}, false
}

// mediaType is the media type of a picture's bytes, for export's data
// URLs: a registered vector format's, else what the bytes look like.
func mediaType(name string, data []byte) string {
	if f, ok := vectorFormatOf(name, data); ok {
		return f.mime
	}
	return http.DetectContentType(data)
}

const (
	// vectorRasters and vectorPixels are how many sizes of one vector
	// picture a Document keeps drawn, and how many pixels they hold
	// together. Past either, a size no longer shown makes room; when
	// every one is showing, the nearest is stretched instead.
	vectorRasters = 32
	vectorPixels  = 1 << 24
	// vectorSide is the longest side one is drawn at, what every GPU
	// takes as a texture; a larger box gets that, stretched.
	vectorSide = 8192
)

// drawnAt is a size a vector picture is drawn at: pixels, and how many
// of them a CSS pixel takes.
type drawnAt struct {
	w, h  int
	scale float32
}

// raster is a vector picture drawn at one size, and the paint it was
// last shown in.
type raster struct {
	drawnAt
	tex  *twod.Texture
	used uint64
}

// vector is a Document's picture of shapes and the sizes it is drawn at.
type vector struct {
	d   *Document
	url string
	pic VectorImage
	// async draws off the frame, as a picture that was fetched was read.
	async   bool
	rasters []*raster
	drawing map[drawnAt]bool
	failed  bool
	// asked are the sizes the last two paints asked for, each with the
	// first and the last paint of its run, and shown and before the last
	// two paints that asked for any. A size is drawn once a second paint
	// running asks for it, or at once when the picture was not showing:
	// one whose size changes every frame is stretched from what it has
	// until it settles.
	asked         map[drawnAt]asks
	shown, before uint64
}

// asks are the first and the last of the paints running that asked a
// vector picture for one size.
type asks struct{ first, last uint64 }

func (v *vector) Intrinsic() box.Intrinsic {
	var in box.Intrinsic
	if w, ok := v.pic.Width(); ok {
		in.Width = float32(w)
	}
	if h, ok := v.pic.Height(); ok {
		in.Height = float32(h)
	}
	if r, ok := v.pic.Ratio(); ok {
		in.Ratio = float32(r)
	}
	return in
}

// vectorFit is w×h pixels at scale, or the largest size of that shape
// within vectorSide and vectorPixels, its scale as much smaller.
func vectorFit(w, h int, scale float32) drawnAt {
	w, h = max(w, 1), max(h, 1)
	if scale <= 0 {
		scale = 1
	}
	k := min(1, vectorSide/float64(w), vectorSide/float64(h), math.Sqrt(vectorPixels/(float64(w)*float64(h))))
	if k < 1 {
		w, h, scale = max(int(float64(w)*k), 1), max(int(float64(h)*k), 1), scale*float32(k)
	}
	return drawnAt{w, h, scale}
}

// Texture is the picture drawn w×h pixels large for a screen of scale
// pixels a point (paint.Vector): the one
// kept for that size, else drawn now, or off the frame for a fetched
// picture. While it is being drawn, while its size is changing, and when
// vectorRasters sizes are showing at once, it is the nearest size kept,
// nil when there is none; the Document paints again once the size has
// settled or the drawing has landed.
func (v *vector) Texture(w, h int, scale float32) *twod.Texture {
	if v.failed {
		return nil
	}
	size, now := vectorFit(w, h, scale), v.d.painted
	if v.shown != now {
		v.before, v.shown = v.shown, now
	}
	for s, a := range v.asked {
		if a.last+1 < now {
			delete(v.asked, s)
		}
	}
	a, again := v.asked[size]
	if !again {
		a.first = now
	}
	a.last = now
	if v.asked == nil {
		v.asked = map[drawnAt]asks{}
	}
	v.asked[size] = a
	for _, r := range v.rasters {
		if r.drawnAt == size {
			r.used = now
			return r.tex
		}
	}
	showing := v.before > 0 && v.before+1 == now
	settled := a.first < now || !showing || len(v.rasters) == 0
	switch {
	case !settled:
		v.d.unsettled = true
	case v.drawing[size] || !v.room(size):
	case v.async:
		v.drawOffTheFrame(size)
	default:
		img, err := v.rasterise(size)
		if r := v.land(size, img, err); r != nil {
			return r.tex
		}
	}
	return v.nearest(size)
}

func (v *vector) rasterise(at drawnAt) (*image.RGBA, error) {
	img, err := v.pic.Rasterise(at.w, at.h, float64(at.scale))
	if err == nil && (img == nil || img.Rect.Empty()) {
		err = fmt.Errorf("nothing drawn at %d×%d", at.w, at.h)
	}
	return img, err
}

// drawOffTheFrame draws the picture at a size in one of the fetches'
// slots; landImages takes it.
func (v *vector) drawOffTheFrame(at drawnAt) {
	if v.drawing == nil {
		v.drawing = map[drawnAt]bool{}
	}
	v.drawing[at] = true
	f := &v.d.fetching
	f.Add(1)
	go func() {
		defer f.Done()
		f.slots <- struct{}{}
		defer func() { <-f.slots }()
		img, err := v.rasterise(at)
		f.mu.Lock()
		defer f.mu.Unlock()
		f.drawn = append(f.drawn, drawnVector{v, at, img, err})
	}()
}

// drawnVector is a vector picture drawn off the frame at one size.
type drawnVector struct {
	v   *vector
	at  drawnAt
	img *image.RGBA
	err error
}

// land keeps img as the picture at a size, nil when there is no room
// left for it; an error fails the picture for good, with a warning.
func (v *vector) land(at drawnAt, img *image.RGBA, err error) *raster {
	delete(v.drawing, at)
	if err != nil {
		v.failed, v.rasters = true, nil
		v.d.warn(fmt.Errorf("ui: image %s: %w", v.url, err))
		return nil
	}
	if !v.room(at) {
		return nil
	}
	r := &raster{drawnAt: at, tex: twod.NewTexture(img, twod.Encoded()), used: v.d.painted}
	v.rasters = append(v.rasters, r)
	return r
}

// room makes room for a raster of a size among those kept, dropping the
// ones the current paint has not shown, the longest unseen first; false
// when those showing leave none.
func (v *vector) room(at drawnAt) bool {
	for {
		pixels := at.w * at.h
		oldest := -1
		for i, r := range v.rasters {
			pixels += r.w * r.h
			if r.used < v.d.painted && (oldest < 0 || r.used < v.rasters[oldest].used) {
				oldest = i
			}
		}
		if len(v.rasters) < vectorRasters && (pixels <= vectorPixels || len(v.rasters) == 0) {
			return true
		}
		if oldest < 0 {
			return false
		}
		v.rasters = append(v.rasters[:oldest], v.rasters[oldest+1:]...)
	}
}

// nearest is the raster kept whose size is closest to at, to stretch
// over the same box.
func (v *vector) nearest(at drawnAt) *twod.Texture {
	var best *raster
	far := math.Inf(1)
	for _, r := range v.rasters {
		d := math.Abs(math.Log(float64(r.w)/float64(at.w))) + math.Abs(math.Log(float64(r.h)/float64(at.h)))
		if d < far {
			best, far = r, d
		}
	}
	if best == nil {
		return nil
	}
	best.used = v.d.painted
	return best.tex
}

// undrawn is the warning naming what pic, the file url names, uses that
// the format doesn't draw; nil when it draws all of it.
func (f vectorFormat) undrawn(url string, pic VectorImage) error {
	not := pic.Unsupported()
	if len(not) == 0 {
		return nil
	}
	return fmt.Errorf("ui: image %s: %s is not drawn: %s", url, f.mime, strings.Join(not, ", "))
}

// keepVector makes pic the picture url names.
func (d *Document) keepVector(url string, pic VectorImage, warning error, async bool) {
	if warning != nil {
		d.warn(warning)
	}
	if d.vectors == nil {
		d.vectors = map[string]*vector{}
	}
	d.vectors[url] = &vector{d: d, url: url, pic: pic, async: async}
}

// vector is the picture of shapes url names, nil for any other
// (paint.Vectors).
func (d *Document) vector(url string) paint.Vector {
	d.image(url)
	if v, ok := d.vectors[url]; ok {
		return v
	}
	return nil
}

// ImageIntrinsic is what the picture of shapes url names asks to be
// sized by, for layout's replaced elements (box.IntrinsicImages).
func (d *Document) ImageIntrinsic(url string) (box.Intrinsic, bool) {
	d.image(url)
	if v, ok := d.vectors[url]; ok {
		return v.Intrinsic(), true
	}
	return box.Intrinsic{}, false
}

var _ box.IntrinsicImages = (*Document)(nil)

// filterGraphs is a picture of shapes whose filter elements an element
// outside it can name: ui/svg's.
type filterGraphs interface {
	FilterGraph(id string, w, h float32) (*fx.Graph, bool)
}

// svgFilter is the filter element id names in the first svg element of d
// holding one, read for an element of size points; nil for none.
func (d *Document) svgFilter(id string, size geom.Vec2) *fx.Graph {
	var g *fx.Graph
	d.html.Walk(func(n *dom.Node) bool {
		if g != nil || n.IsText() {
			return false
		}
		if n.Tag() != dom.Svg {
			return true
		}
		url, ok := dom.Picture(n)
		if !ok {
			return false
		}
		d.image(url)
		if v, has := d.vectors[url]; has {
			if f, can := v.pic.(filterGraphs); can {
				g, _ = f.FilterGraph(id, size.X, size.Y)
			}
		}
		return false
	})
	return g
}
