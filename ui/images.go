package ui

import (
	"bytes"
	"fmt"
	"image"
	_ "image/png" // ui.Images and url() backgrounds decode PNG
	"io/fs"
	"sync"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/ui/css"
	"github.com/lantervale/ember/ui/internal/box"
)

// imageBytes is the file url names: one of css.DefaultImages, else a path
// in the stylesheets' file systems, the last added first.
func (d *Document) imageBytes(url string) ([]byte, bool) {
	if b, ok := css.DefaultImages()[url]; ok {
		return b, true
	}
	for i := len(d.files) - 1; i >= 0; i-- {
		if b, err := fs.ReadFile(d.files[i], url); err == nil {
			return b, true
		}
	}
	return nil, false
}

// ImageFetcher gets the picture a url() or an img's src names, as it is
// written there, when no file system of the Document holds it: over the
// network, from a cache, wherever the game keeps what it didn't ship. It
// is called off the frame, a few pictures at a time, so it may take as
// long as it needs and must be safe to call from several goroutines.
type ImageFetcher func(url string) ([]byte, error)

// FetchImages has fetch get the pictures the Document's file systems
// don't hold. A picture is asked for once, when it is first laid out or
// painted; its element shows as one without a picture until it lands,
// then the Document lays out and paints again. A fetch that fails, or
// bytes that are no picture, are a warning. ui/web has a fetcher over
// HTTP.
func FetchImages(fetch ImageFetcher) DocumentOption {
	return func(d *Document) { d.fetch = fetch }
}

// fetchesAtOnce is how many pictures a Document fetches at a time.
const fetchesAtOnce = 6

// fetchedImage is a picture back from the fetcher, decoded off the
// frame, or why there is none.
type fetchedImage struct {
	url string
	img image.Image
	// vec is the picture when it is one of shapes, warning what of it
	// isn't drawn.
	vec     VectorImage
	warning error
	err     error
}

// fetches are a Document's pictures on their way: landed holds those
// back since the last layout, which takes them.
type fetches struct {
	sync.WaitGroup
	mu     sync.Mutex
	landed []fetchedImage
	// drawn are the vector pictures drawn off the frame since the last
	// layout.
	drawn []drawnVector
	slots chan struct{}
}

// fetchImage asks the fetcher for url off the frame.
func (d *Document) fetchImage(url string) {
	f := &d.fetching
	if f.slots == nil {
		f.slots = make(chan struct{}, fetchesAtOnce)
	}
	fetch := d.fetch
	f.Add(1)
	go func() {
		defer f.Done()
		f.slots <- struct{}{}
		defer func() { <-f.slots }()
		got := fetched(url, fetch)
		f.mu.Lock()
		defer f.mu.Unlock()
		f.landed = append(f.landed, got)
	}()
}

// fetched is the picture fetch gets for url, read as a registered format
// of shapes when one recognises it.
func fetched(url string, fetch ImageFetcher) fetchedImage {
	got := fetchedImage{url: url}
	data, err := fetch(url)
	if err != nil {
		got.err = err
		return got
	}
	format, shapes := vectorFormatOf(url, data)
	if !shapes {
		got.img, _, got.err = image.Decode(bytes.NewReader(data))
		return got
	}
	if got.vec, got.err = format.parse(data); got.err == nil {
		got.warning = format.undrawn(url, got.vec)
	}
	return got
}

// landImages takes the pictures fetched since the last layout: each
// becomes its url's texture, and the Document is laid out and painted
// again with them.
func (d *Document) landImages() {
	f := &d.fetching
	f.mu.Lock()
	landed, drawn := f.landed, f.drawn
	f.landed, f.drawn = nil, nil
	f.mu.Unlock()
	for _, got := range landed {
		switch {
		case got.err != nil:
			d.warn(fmt.Errorf("ui: image %s: %w", got.url, got.err))
			continue
		case got.vec != nil:
			d.keepVector(got.url, got.vec, got.warning, true)
		default:
			d.images[got.url] = twod.NewTexture(got.img, twod.Encoded())
		}
		d.relayout = true
	}
	for _, got := range drawn {
		got.v.land(got.at, got.img, got.err)
		d.replay.repaint = true
	}
}

// image is the texture url names, decoded once; nil when there is none,
// with a warning, while the fetcher is getting it, and for a picture of
// shapes, which vector has.
func (d *Document) image(url string) *twod.Texture {
	if t, ok := d.images[url]; ok {
		return t
	}
	var tex *twod.Texture
	b, ok := dataURL(url)
	if !ok {
		b, ok = d.imageBytes(url)
	}
	switch {
	case ok:
		tex = d.decoded(url, b)
	case d.fetch != nil:
		d.fetchImage(url)
	default:
		d.warn(&missingImageError{url})
	}
	if d.images == nil {
		d.images = map[string]*twod.Texture{}
	}
	d.images[url] = tex
	return tex
}

// decoded is the texture of the picture file b that url names: nil with
// a warning when it can't be read, and for a picture of shapes, kept as
// url's vector.
func (d *Document) decoded(url string, b []byte) *twod.Texture {
	if format, shapes := vectorFormatOf(url, b); shapes {
		pic, err := format.parse(b)
		if err != nil {
			d.warn(fmt.Errorf("ui: image %s: %w", url, err))
			return nil
		}
		d.keepVector(url, pic, format.undrawn(url, pic), false)
		return nil
	}
	img, _, err := image.Decode(bytes.NewReader(b))
	if err != nil {
		d.warn(err)
		return nil
	}
	return twod.NewTexture(img, twod.Encoded())
}

type missingImageError struct{ url string }

func (e *missingImageError) Error() string { return "ui: no image " + e.url }

// ImageSize is the natural size of the image url names, one point per
// pixel, for layout's replaced elements (box.Images).
func (d *Document) ImageSize(url string) (geom.Vec2, bool) {
	t := d.image(url)
	if v, ok := d.vectors[url]; ok {
		return v.Intrinsic().Natural(geom.V2(300, 150)), true
	}
	if t == nil {
		return geom.Vec2{}, false
	}
	return t.Size(), true
}

var _ box.Images = (*Document)(nil)
