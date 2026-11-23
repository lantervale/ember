// Package animated holds the pictures that move: an animated GIF or WebP
// read into whole frames, how long each shows and how often the whole
// plays, and which frame shows a time after it began, as Chrome plays
// it. The readers of the formats register here, so that ui reads
// animations only of the formats a program imported (ui/web).
package animated

import (
	"bytes"
	"image"
	"image/draw"
	"sync"
	"time"
)

// MaxPixels is the most pixels of all the frames of a picture held
// together: those past it are drawn as they show, from the frame before.
const MaxPixels = 1 << 25

// Picture is an animation: its frames as they show, one after the
// other, how long each does as the file says, and how many times the
// whole plays, 0 for ever. Frames are those held, the first within
// MaxPixels; Frame draws the others as they come.
type Picture struct {
	Frames []*image.NRGBA
	Delays []time.Duration
	Plays  int
	later  *later
}

// later are the frames of a Picture past those it holds: drawn one
// after the other on live, from the canvas as the last held frame left
// it, each disposed of before the next.
type later struct {
	from    []byte
	steps   []Step
	live    *image.NRGBA
	next    int
	dispose func()
}

const (
	// shortest is the longest delay Chrome takes for none: a frame the
	// file gives it or less shows for slowest.
	shortest = 10 * time.Millisecond
	slowest  = 100 * time.Millisecond
)

// Len is how many frames p shows.
func (p *Picture) Len() int { return len(p.Delays) }

// Moves reports whether p is an animation, not a Still.
func (p *Picture) Moves() bool { return p.Len() > 1 }

// Duration is how long frame i shows, as Chrome shows it: 100 ms for
// one the file gives 10 ms or less, as the ads that asked for frames as
// quick as they come once had browsers agree.
func (p *Picture) Duration(i int) time.Duration {
	if d := p.Delays[i]; d > shortest {
		return d
	}
	return slowest
}

// Whole is how long one play of p lasts.
func (p *Picture) Whole() time.Duration {
	var whole time.Duration
	for i := range p.Delays {
		whole += p.Duration(i)
	}
	return whole
}

// Within is the frame that shows t into a play, t below Whole.
func (p *Picture) Within(t time.Duration) int {
	for i := range p.Delays {
		if t < p.Duration(i) {
			return i
		}
		t -= p.Duration(i)
	}
	return p.Len() - 1
}

// Frame is frame i: one held, else drawn from the frames before it,
// on a canvas Frame draws every later frame on, so it holds until the
// next call. A frame that can't be drawn leaves the one before.
func (p *Picture) Frame(i int) *image.NRGBA {
	if i < len(p.Frames) {
		return p.Frames[i]
	}
	l := p.later
	if l.live == nil || i < l.next {
		if l.live == nil {
			l.live = image.NewNRGBA(p.Frames[0].Rect)
		}
		copy(l.live.Pix, l.from)
		l.next, l.dispose = len(p.Frames), Keep
	}
	for ; l.next <= i; l.next++ {
		l.dispose()
		var err error
		if l.dispose, err = l.steps[l.next-len(p.Frames)](l.live); err != nil {
			l.dispose = Keep
		}
	}
	return l.live
}

// Step draws a frame on the canvas, and returns what disposes of it
// once shown.
type Step func(canvas *image.NRGBA) (func(), error)

// Keep is the disposal of a frame that leaves the canvas as it is.
func Keep() {}

// Decoder reads an animation: nil and no error for a picture of one
// frame the image package reads, a Still for one it doesn't.
type Decoder func(data []byte) (*Picture, error)

type format struct {
	magic  string
	decode Decoder
}

//nolint:gochecknoglobals // the registry of formats, as the image package keeps its own
var (
	mu      sync.Mutex
	formats []format
)

// Register has Decode read the files that start with magic, where a ?
// stands for any byte, with decode.
func Register(magic string, decode Decoder) {
	mu.Lock()
	defer mu.Unlock()
	formats = append(formats, format{magic, decode})
}

// Decode reads data as an animation, by the first format registered
// that knows it: nil and no error when none does or the picture holds a
// single frame.
func Decode(data []byte) (*Picture, error) {
	mu.Lock()
	known := formats
	mu.Unlock()
	for _, f := range known {
		if matches(f.magic, data) {
			return f.decode(data)
		}
	}
	return nil, nil //nolint:nilnil // no animation, and nothing wrong
}

func matches(magic string, data []byte) bool {
	if len(data) < len(magic) {
		return false
	}
	for i := range len(magic) {
		if magic[i] != '?' && magic[i] != data[i] {
			return false
		}
	}
	return true
}

// Canvas is where an animation's frames are drawn, one over the other:
// a frame shows the canvas once it is drawn.
type Canvas struct {
	*image.NRGBA
	pic *Picture
}

// NewCanvas is a clear canvas of w×h pixels for an animation that plays
// plays times, 0 for ever.
func NewCanvas(w, h, plays int) *Canvas {
	return &Canvas{NRGBA: image.NewNRGBA(image.Rect(0, 0, w, h)), pic: &Picture{Plays: plays}}
}

// Add has step draw the next frame, shown for delay: at once while the
// frames held stay within MaxPixels, kept to draw as it shows past them.
// An error drawing a frame at once is Add's.
func (c *Canvas) Add(delay time.Duration, step Step) error {
	p := c.pic
	if p.later == nil && (len(p.Frames)+1)*len(c.Pix)/4 > MaxPixels && len(p.Frames) > 0 {
		p.later = &later{from: bytes.Clone(c.Pix)}
	}
	if p.later != nil {
		p.later.steps, p.Delays = append(p.later.steps, step), append(p.Delays, delay)
		return nil
	}
	dispose, err := step(c.NRGBA)
	if err != nil {
		return err
	}
	frame := image.NewNRGBA(c.Rect)
	copy(frame.Pix, c.Pix)
	p.Frames, p.Delays = append(p.Frames, frame), append(p.Delays, delay)
	dispose()
	return nil
}

// Clear clears r of img, as a frame disposed to the background is in
// Chrome, whatever background the file names.
func Clear(img *image.NRGBA, r image.Rectangle) {
	draw.Draw(img, r, image.Transparent, image.Point{}, draw.Src)
}

// Still is the one frame drawn so far, as a picture that doesn't move:
// for a file the image package can't read, nil when there is none.
func (c *Canvas) Still() *Picture {
	if c.pic.Len() == 0 {
		return nil
	}
	return &Picture{Frames: c.pic.Frames[:1], Delays: c.pic.Delays[:1], Plays: 1}
}

// Picture is the animation drawn so far, nil when it holds one frame
// or none.
func (c *Canvas) Picture() *Picture {
	if !c.pic.Moves() {
		return nil
	}
	return c.pic
}
