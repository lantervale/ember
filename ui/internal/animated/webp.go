package animated

import (
	"encoding/binary"
	"errors"
	"image"
	"image/draw"
	"math"
	"time"
)

// WebPMagic is how an extended WebP file starts, which an animation is.
const WebPMagic = "RIFF????WEBPVP8X"

const (
	riffHeader  = 12
	chunkHeader = 8
	vp8xLength  = 10
	animLength  = 6
	anmfLength  = 16

	animationFlag = 0x02
	alphaFlag     = 0x10
	noBlend       = 0x02
	disposeFlag   = 0x01
)

var (
	errWebP       = errors.New("animated: not a WebP file")
	errChunk      = errors.New("animated: a WebP chunk is outside the file")
	errFrame      = errors.New("animated: a WebP frame is outside its canvas")
	errFrameData  = errors.New("animated: a WebP frame holds no picture")
	errFrameSize  = errors.New("animated: a WebP frame's picture is not the size it says")
	errCanvasSize = errors.New("animated: the WebP canvas is over the most pixels read")
)

// chunk is one chunk of a RIFF file: its kind and what it holds.
type chunk struct {
	kind string
	data []byte
}

// chunks are the chunks data holds, each checked to lie in it.
func chunks(data []byte) ([]chunk, error) {
	var out []chunk
	for len(data) > 0 {
		if len(data) < chunkHeader {
			return nil, errChunk
		}
		n := uint64(binary.LittleEndian.Uint32(data[4:]))
		if n > uint64(len(data)-chunkHeader) { //nolint:gosec // len(data) is at least chunkHeader
			return nil, errChunk
		}
		out = append(out, chunk{string(data[:4]), data[chunkHeader : chunkHeader+n]})
		data = data[chunkHeader+n:]
		if n%2 == 1 && len(data) > 0 {
			data = data[1:]
		}
	}
	return out, nil
}

// DecodeWebP reads an animated WebP file, decoding each frame's picture,
// a WebP file of its own, with frame: its frames drawn on a clear canvas
// as Chrome draws them, blended over what is there unless a frame says
// not to, cleared where one disposes of itself to the background,
// whatever background the file names. A loop count of n plays the
// animation n times. A WebP that isn't animated is nil.
func DecodeWebP(data []byte, frame func([]byte) (image.Image, error)) (*Picture, error) {
	if len(data) < riffHeader || string(data[:4]) != "RIFF" || string(data[8:12]) != "WEBP" {
		return nil, errWebP
	}
	size := uint64(binary.LittleEndian.Uint32(data[4:]))
	body := data[riffHeader:]
	if size >= 4 && size-4 < uint64(len(body)) {
		body = body[:size-4]
	}
	all, err := chunks(body)
	if err != nil {
		return nil, err
	}
	if len(all) == 0 || all[0].kind != "VP8X" || len(all[0].data) < vp8xLength {
		return nil, errWebP
	}
	head := all[0].data
	if head[0]&animationFlag == 0 {
		return nil, nil //nolint:nilnil // a still picture
	}
	w, h := int(uint24(head[4:]))+1, int(uint24(head[7:]))+1
	if w*h > MaxPixels {
		return nil, errCanvasSize
	}
	plays := 0
	for _, c := range all {
		if c.kind == "ANIM" && len(c.data) >= animLength {
			plays = int(binary.LittleEndian.Uint16(c.data[4:]))
			break
		}
	}
	canvas := NewCanvas(w, h, plays)
	var last image.Rectangle
	dispose := false
	for _, c := range all {
		if c.kind != "ANMF" {
			continue
		}
		if len(c.data) < anmfLength {
			return nil, errChunk
		}
		f := c.data
		at := image.Rect(0, 0, int(uint24(f[6:]))+1, int(uint24(f[9:]))+1).
			Add(image.Pt(2*int(uint24(f[0:])), 2*int(uint24(f[3:]))))
		if !at.In(canvas.Rect) {
			return nil, errFrame
		}
		cleared, wipe := last, dispose
		step := func(img *image.NRGBA) (func(), error) {
			pic, frameErr := framePicture(f[anmfLength:], at.Dx(), at.Dy(), frame)
			if frameErr != nil {
				return nil, frameErr
			}
			if wipe {
				Clear(img, cleared)
			}
			if f[15]&noBlend != 0 {
				draw.Draw(img, at, pic, pic.Bounds().Min, draw.Src)
			} else {
				blend(img, at, pic)
			}
			return Keep, nil
		}
		if err = canvas.Add(time.Duration(uint24(f[12:]))*time.Millisecond, step); err != nil {
			return nil, err
		}
		last, dispose = at, f[15]&disposeFlag != 0
	}
	return canvas.Picture(), nil
}

func uint24(b []byte) uint32 { return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 }

// framePicture decodes what a frame holds, its ALPH chunk and its VP8
// chunk or its VP8L chunk, as a WebP file of w×h pixels of its own.
func framePicture(data []byte, w, h int, frame func([]byte) (image.Image, error)) (image.Image, error) {
	parts, err := chunks(data)
	if err != nil {
		return nil, err
	}
	var alph, pix *chunk
	for i, c := range parts {
		switch c.kind {
		case "ALPH":
			if alph == nil {
				alph = &parts[i]
			}
		case "VP8 ", "VP8L":
			if pix == nil {
				pix = &parts[i]
			}
		}
	}
	if pix == nil {
		return nil, errFrameData
	}
	var file []byte
	if alph != nil && pix.kind == "VP8 " {
		head := make([]byte, vp8xLength)
		head[0] = alphaFlag
		put24(head[4:], uint32(w-1)) //nolint:gosec // at most MaxPixels
		put24(head[7:], uint32(h-1)) //nolint:gosec // at most MaxPixels
		file = appendChunk(file, "VP8X", head)
		file = appendChunk(file, "ALPH", alph.data)
	}
	file = appendChunk(file, pix.kind, pix.data)
	riff := append([]byte("RIFF"), 0, 0, 0, 0)
	riff = append(riff, "WEBP"...)
	binary.LittleEndian.PutUint32(riff[4:], uint32(4+len(file))) //nolint:gosec // within the file read
	img, err := frame(append(riff, file...))
	if err != nil {
		return nil, err
	}
	if img.Bounds().Dx() != w || img.Bounds().Dy() != h {
		return nil, errFrameSize
	}
	return img, nil
}

func put24(b []byte, v uint32) { b[0], b[1], b[2] = byte(v), byte(v>>8), byte(v>>16) } //nolint:gosec // 24 bits

func appendChunk(out []byte, kind string, data []byte) []byte {
	out = append(out, kind...)
	out = binary.LittleEndian.AppendUint32(out, uint32(len(data))) //nolint:gosec // within the file read
	out = append(out, data...)
	if len(data)%2 == 1 {
		out = append(out, 0)
	}
	return out
}

// blend draws pic over r of dst, as WebP blends a frame: in colours not
// premultiplied by their alpha.
func blend(dst *image.NRGBA, r image.Rectangle, pic image.Image) {
	src := image.NewNRGBA(pic.Bounds())
	draw.Draw(src, src.Rect, pic, src.Rect.Min, draw.Src)
	for y := range r.Dy() {
		for x := range r.Dx() {
			s := src.Pix[src.PixOffset(src.Rect.Min.X+x, src.Rect.Min.Y+y):][:4]
			d := dst.Pix[dst.PixOffset(r.Min.X+x, r.Min.Y+y):][:4]
			sa, da := float64(s[3])/math.MaxUint8, float64(d[3])/math.MaxUint8
			under := da * (1 - sa)
			a := sa + under
			if a == 0 {
				d[0], d[1], d[2], d[3] = 0, 0, 0, 0
				continue
			}
			for i := range 3 {
				d[i] = uint8((float64(s[i])*sa+float64(d[i])*under)/a + 0.5)
			}
			d[3] = uint8(a*math.MaxUint8 + 0.5)
		}
	}
}
