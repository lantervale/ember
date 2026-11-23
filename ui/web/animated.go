package web

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/draw"
	"image/gif"
	"time"

	"golang.org/x/image/webp"

	"github.com/lantervale/ember/ui/internal/animated"
)

// decodeGIF reads an animated GIF as Chrome plays it: its frames drawn
// on a clear canvas, each disposed of as it says before the next (left,
// cleared, whatever background the file names, or put back to what was
// under it), played once without a loop count, for ever with one of 0,
// and n+1 times with one of n. A first frame reaching past the picture
// makes it as large as that frame reaches, and a later one is cut at
// its edge, as Chrome does. A GIF of one frame is nil, unless that frame
// reaches past the picture, which the image package refuses: then it is
// the one frame.
func decodeGIF(data []byte) (*animated.Picture, error) {
	r := gifReach(data)
	past := r.past
	if past {
		data = bytes.Clone(data)
		binary.LittleEndian.PutUint16(data[6:], uint16(min(r.reach.X, 0xffff))) //nolint:gosec // at most 16 bits
		binary.LittleEndian.PutUint16(data[8:], uint16(min(r.reach.Y, 0xffff))) //nolint:gosec // at most 16 bits
	}
	g, err := gif.DecodeAll(bytes.NewReader(data))
	if err != nil || len(g.Image) < 2 && !past {
		return nil, err
	}
	if past {
		g.Config.Width, g.Config.Height = max(r.size.X, r.first.X), max(r.size.Y, r.first.Y)
	}
	plays := 1
	if g.LoopCount >= 0 {
		plays = g.LoopCount + 1
		if g.LoopCount == 0 {
			plays = 0
		}
	}
	canvas := animated.NewCanvas(g.Config.Width, g.Config.Height, plays)
	for i, frame := range g.Image {
		var disposal byte
		if i < len(g.Disposal) {
			disposal = g.Disposal[i]
		}
		step := func(img *image.NRGBA) (func(), error) {
			var saved []byte
			if disposal == gif.DisposalPrevious {
				saved = bytes.Clone(img.Pix)
			}
			at := frame.Bounds().Intersect(img.Rect)
			draw.Draw(img, at, frame, at.Min, draw.Over)
			switch disposal {
			case gif.DisposalBackground:
				return func() { animated.Clear(img, at) }, nil
			case gif.DisposalPrevious:
				return func() { copy(img.Pix, saved) }, nil
			}
			return animated.Keep, nil
		}
		if err = canvas.Add(time.Duration(g.Delay[i])*10*time.Millisecond, step); err != nil {
			return nil, err
		}
	}
	if past && len(g.Image) == 1 {
		return canvas.Still(), nil
	}
	return canvas.Picture(), nil
}

// reach is the size a GIF says its picture is, how far right and down
// its first frame and all its frames reach, and whether one reaches past
// the picture.
type reach struct {
	size, first, reach image.Point
	past               bool
}

// gifReach is data's reach: not past for a file that ends or breaks
// before its trailer, left to the image package to read.
func gifReach(data []byte) reach {
	const header = 13
	var r reach
	if len(data) < header {
		return r
	}
	r.size = image.Pt(int(binary.LittleEndian.Uint16(data[6:])), int(binary.LittleEndian.Uint16(data[8:])))
	at := header + colorTable(data[10])
	blocks := func() bool {
		for at < len(data) {
			n := int(data[at])
			at += 1 + n
			if n == 0 {
				return at <= len(data)
			}
		}
		return false
	}
	for at < len(data) {
		switch data[at] {
		case 0x21:
			at += 2
			if !blocks() {
				return r
			}
		case 0x2c:
			if at+10 > len(data) {
				return r
			}
			d := data[at+1:]
			x, y := int(binary.LittleEndian.Uint16(d)), int(binary.LittleEndian.Uint16(d[2:]))
			w, h := int(binary.LittleEndian.Uint16(d[4:])), int(binary.LittleEndian.Uint16(d[6:]))
			r.reach = image.Pt(max(r.reach.X, x+w), max(r.reach.Y, y+h))
			if r.first == (image.Point{}) {
				r.first = r.reach
			}
			at += 10 + colorTable(d[8]) + 1
			if !blocks() {
				return r
			}
		case 0x3b:
			r.past = r.reach.X > r.size.X || r.reach.Y > r.size.Y
			return r
		default:
			return r
		}
	}
	return r
}

// colorTable is the length of the colour table a GIF's flags byte says
// follows.
func colorTable(flags byte) int {
	if flags&0x80 == 0 {
		return 0
	}
	return 3 << (flags&7 + 1)
}

func decodeWebP(data []byte) (*animated.Picture, error) {
	return animated.DecodeWebP(data, func(b []byte) (image.Image, error) { return webp.Decode(bytes.NewReader(b)) })
}
