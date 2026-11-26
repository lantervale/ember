package ui

import (
	"bytes"
	"fmt"
	"image"
	_ "image/png" // ui.Images and url() backgrounds decode PNG
	"io/fs"
	"math"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/ui/css"
	"github.com/lantervale/ember/ui/internal/animated"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/dom"
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
// then the Document paints again, and lays out again what the picture's
// size moves. A fetch that fails, or
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
	// anim is the picture when it moves.
	anim *animated.Picture
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
	// queue are the urls asked for that wait for one of the
	// fetchesAtOnce fetches, running, the most urgent first once a frame
	// sorted them; fetch is the fetcher they go to.
	queue   []string
	running int
	fetch   ImageFetcher
	// holding is set during a layout, which asks for its pictures all
	// before fetchQueued sorts them.
	holding bool
	// onWay are the urls asked for that have not landed, read and
	// written on the frame.
	onWay map[string]bool
	// drawing is how many vector pictures are being drawn off the frame
	// and not yet landed, read and written on the frame.
	drawing int
}

// ImagesLoading is how many of the Document's pictures are on their way:
// asked for and not yet landed, fetched or drawn at a size off the frame.
func (d *Document) ImagesLoading() int { return len(d.fetching.onWay) + d.fetching.drawing }

// fetchImage asks the fetcher for url off the frame: during a layout,
// once it sorted it among the others asked for (fetchQueued).
func (d *Document) fetchImage(url string) {
	f := &d.fetching
	if f.slots == nil {
		f.slots = make(chan struct{}, fetchesAtOnce)
	}
	if f.onWay == nil {
		f.onWay = map[string]bool{}
	}
	f.onWay[url] = true
	f.Add(1)
	f.mu.Lock()
	defer f.mu.Unlock()
	f.fetch, f.queue = d.fetch, append(f.queue, url)
	for !f.holding && f.running < fetchesAtOnce && len(f.queue) > 0 {
		f.startNext()
	}
}

// fetchQueued sorts the pictures waiting to be fetched by the priority
// their imgs' fetchpriority gives them, high first and low last, as
// Chrome does, then has them fetched while fewer than fetchesAtOnce are.
func (d *Document) fetchQueued() {
	f := &d.fetching
	f.mu.Lock()
	defer f.mu.Unlock()
	f.holding = false
	if len(f.queue) == 0 {
		return
	}
	priority := map[string]int{}
	for _, n := range d.prioritized {
		url, ok := dom.Picture(n)
		if !ok {
			continue
		}
		v, _ := n.Attr("fetchpriority")
		switch strings.ToLower(v) {
		case "high":
			priority[url] = max(priority[url], 1)
		case "low":
			if _, set := priority[url]; !set {
				priority[url] = -1
			}
		}
	}
	slices.SortStableFunc(f.queue, func(a, b string) int { return priority[b] - priority[a] })
	for f.running < fetchesAtOnce && len(f.queue) > 0 {
		f.startNext()
	}
}

// startNext fetches the first picture of the queue off the frame, then
// the next once it is back; f.mu is held.
func (f *fetches) startNext() {
	url, fetch := f.queue[0], f.fetch
	f.queue = f.queue[1:]
	f.running++
	go func() {
		defer f.Done()
		f.slots <- struct{}{}
		got := fetched(url, fetch)
		<-f.slots
		f.mu.Lock()
		defer f.mu.Unlock()
		f.landed = append(f.landed, got)
		f.running--
		if len(f.queue) > 0 {
			f.startNext()
		}
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
		if got.anim, got.err = animated.Decode(data); got.anim == nil && got.err == nil {
			got.img, _, got.err = image.Decode(bytes.NewReader(data))
		}
		return got
	}
	if got.vec, got.err = format.parse(data); got.err == nil {
		got.warning = format.undrawn(url, got.vec)
	}
	return got
}

// landImages takes the pictures fetched since the last layout: each
// becomes its url's texture and is painted, and only an img whose box its
// picture sizes is laid out again, with what its size moves.
func (d *Document) landImages() {
	f := &d.fetching
	f.mu.Lock()
	landed, drawn := f.landed, f.drawn
	f.landed, f.drawn = nil, nil
	f.mu.Unlock()
	var shown map[string]bool
	for _, got := range landed {
		delete(f.onWay, got.url)
		switch {
		case got.err != nil:
			d.warn(fmt.Errorf("ui: image %s: %w", got.url, got.err))
			continue
		case got.vec != nil:
			d.keepVector(got.url, got.vec, got.warning, true)
		case got.anim != nil && got.anim.Moves():
			d.images[got.url] = d.play(got.url, got.anim)
		case got.anim != nil:
			d.images[got.url] = twod.NewTexture(got.anim.Frames[0], twod.Encoded())
		default:
			d.images[got.url] = twod.NewTexture(got.img, twod.Encoded())
		}
		if shown == nil {
			shown = map[string]bool{}
		}
		shown[got.url] = true
	}
	if shown != nil {
		d.replay.repaint = true
		d.kept.Landed(func(url string) bool { return shown[url] }, d, d.fonts)
	}
	for _, got := range drawn {
		f.drawing--
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
	anim, err := animated.Decode(b)
	switch {
	case anim != nil && anim.Moves():
		return d.play(url, anim)
	case anim != nil:
		return twod.NewTexture(anim.Frames[0], twod.Encoded())
	}
	if err != nil {
		d.warn(fmt.Errorf("ui: image %s: %w", url, err))
		return nil
	}
	img, _, err := image.Decode(bytes.NewReader(b))
	if err != nil {
		d.warn(err)
		return nil
	}
	return twod.NewTexture(img, twod.Encoded())
}

// playing is an animated picture on show: one texture every element
// showing its url draws, whose image is the frame of the moment, as
// Chrome's pictures of one address play as one.
type playing struct {
	pic   *animated.Picture
	tex   *twod.Texture
	frame int
	// start is the clock when the play under way began, -1 until the
	// picture is first drawn, and played the plays done before it; done is
	// set once it has played its times. shown is whether the last paint
	// drew it.
	start  float32
	played int
	shown  bool
	done   bool
}

// play is the texture of url's animation, which the Document's clock
// moves once it is drawn.
func (d *Document) play(url string, pic *animated.Picture) *twod.Texture {
	p := &playing{pic: pic, tex: twod.NewTexture(pic.Frames[0], twod.Encoded()), start: -1}
	if d.playing == nil {
		d.playing = map[string]*playing{}
	}
	d.playing[url] = p
	return p.tex
}

// paintImage is image for paint, which notes the animations it draws:
// one begins when first drawn.
func (d *Document) paintImage(url string) *twod.Texture {
	if p, ok := d.playing[url]; ok {
		if d.drawnPictures == nil {
			d.drawnPictures = map[string]bool{}
		}
		d.drawnPictures[url] = true
		if p.start < 0 {
			p.start = max(d.clock, 0)
		}
	}
	return d.image(url)
}

// animate shows each animation's frame at the clock, and repaints where
// one changed. An animation moves only while the last paint drew it,
// as Chrome's only moves in view: drawn again within the play it left,
// it shows the frame of the moment; past it, the next play begins.
func (d *Document) animate() {
	for url, p := range d.playing {
		back := d.drawnPictures[url] && !p.shown
		p.shown = d.drawnPictures[url]
		if p.done || !p.shown || p.start < 0 {
			continue
		}
		whole := p.pic.Whole().Seconds()
		t := float64(d.clock - p.start)
		if t >= whole {
			if back {
				p.played++
				p.start, t = d.clock, 0
			} else {
				n := math.Floor(t / whole)
				p.played += int(n)
				p.start += float32(n * whole)
				t -= n * whole
			}
		}
		i := p.pic.Within(time.Duration(t * float64(time.Second)))
		if plays := p.pic.Plays; plays > 0 && p.played >= plays {
			p.done, i = true, p.pic.Len()-1
		}
		if i != p.frame {
			p.frame, p.tex.Image = i, p.pic.Frame(i)
			p.tex.Touch()
			d.replay.repaint = true
		}
	}
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
