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

// MaxPixels is the most pixels of all the frames of a picture together:
// the frames past it are dropped, the animation playing those before.
const MaxPixels = 1 << 25

// Picture is an animation: its frames as they show, one after the
// other, how long each does as the file says, and how many times the
// whole plays, 0 for ever.
type Picture struct {
	Frames []*image.NRGBA
	Delays []time.Duration
	Plays  int
}

const (
	// shortest is the longest delay Chrome takes for none: a frame the
	// file gives it or less shows for slowest.
	shortest = 10 * time.Millisecond
	slowest  = 100 * time.Millisecond
)

// Duration is how long frame i shows, as Chrome shows it: 100 ms for
// one the file gives 10 ms or less, as the ads that asked for frames as
// quick as they come once had browsers agree.
func (p *Picture) Duration(i int) time.Duration {
	if d := p.Delays[i]; d > shortest {
		return d
	}
	return slowest
}

// At is the frame that shows t after the animation began, and whether
// another follows it: once the animation has played its times, its last
// frame stays.
func (p *Picture) At(t time.Duration) (int, bool) {
	var whole time.Duration
	for i := range p.Frames {
		whole += p.Duration(i)
	}
	last := len(p.Frames) - 1
	if p.Plays > 0 && t >= whole*time.Duration(p.Plays) {
		return last, false
	}
	t %= whole
	for i := range p.Frames {
		if t < p.Duration(i) {
			return i, true
		}
		t -= p.Duration(i)
	}
	return last, true
}

// add appends a frame, a copy of canvas shown for delay, reporting
// whether it was within MaxPixels.
func (p *Picture) add(canvas *image.NRGBA, delay time.Duration) bool {
	if (len(p.Frames)+1)*len(canvas.Pix)/4 > MaxPixels {
		return false
	}
	frame := image.NewNRGBA(canvas.Rect)
	copy(frame.Pix, canvas.Pix)
	p.Frames, p.Delays = append(p.Frames, frame), append(p.Delays, delay)
	return true
}

// Decoder reads an animation: nil and no error for a picture of one
// frame, which the image package reads.
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
	pic  *Picture
	full bool
}

// NewCanvas is a clear canvas of w×h pixels for an animation that plays
// plays times, 0 for ever.
func NewCanvas(w, h, plays int) *Canvas {
	return &Canvas{NRGBA: image.NewNRGBA(image.Rect(0, 0, w, h)), pic: &Picture{Plays: plays}}
}

// Show keeps the canvas as it is as the next frame, shown for delay,
// and reports whether the picture has room for more.
func (c *Canvas) Show(delay time.Duration) bool {
	if !c.full && !c.pic.add(c.NRGBA, delay) {
		c.full = true
	}
	return !c.full
}

// Clear clears r of the canvas, as a frame disposed to the background
// is in Chrome, whatever background the file names.
func (c *Canvas) Clear(r image.Rectangle) {
	draw.Draw(c.NRGBA, r, image.Transparent, image.Point{}, draw.Src)
}

// Save is a copy of the canvas, to Restore once a frame disposed to
// what was before it has shown.
func (c *Canvas) Save() []byte { return bytes.Clone(c.Pix) }

// Restore puts back what Save kept.
func (c *Canvas) Restore(saved []byte) { copy(c.Pix, saved) }

// Picture is the animation drawn so far, nil when it holds one frame
// or none.
func (c *Canvas) Picture() *Picture {
	if len(c.pic.Frames) < 2 {
		return nil
	}
	return c.pic
}
