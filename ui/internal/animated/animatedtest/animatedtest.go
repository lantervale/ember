// Package animatedtest writes animated WebP files, for the tests and
// the test pages of what reads them: no picture file is kept in the
// tree. Its frames are rectangles of one colour, which a lossless WebP
// holds in a few bytes.
package animatedtest

import (
	"encoding/binary"
	"image"
	"image/color"
)

// Frame is one frame of an animated WebP: a rectangle of one colour, at
// an even place, shown for Delay milliseconds, drawn over what is there
// unless NoBlend, and cleared once shown when Dispose.
type Frame struct {
	At      image.Rectangle
	Color   color.NRGBA
	Delay   int
	NoBlend bool
	Dispose bool
}

// WebP is an animated WebP file of a w×h canvas whose frames play loops
// times, 0 for ever.
func WebP(w, h, loops int, frames ...Frame) []byte {
	head := make([]byte, 10)
	head[0] = 0x02 | 0x10
	put24(head[4:], w-1)
	put24(head[7:], h-1)
	body := chunk(nil, "VP8X", head)
	anim := make([]byte, 6)
	binary.LittleEndian.PutUint16(anim[4:], uint16(loops)) //nolint:gosec // a test's few
	body = chunk(body, "ANIM", anim)
	for _, f := range frames {
		payload := make([]byte, 16)
		put24(payload, f.At.Min.X/2)
		put24(payload[3:], f.At.Min.Y/2)
		put24(payload[6:], f.At.Dx()-1)
		put24(payload[9:], f.At.Dy()-1)
		put24(payload[12:], f.Delay)
		if f.NoBlend {
			payload[15] |= 0x02
		}
		if f.Dispose {
			payload[15] |= 0x01
		}
		payload = chunk(payload, "VP8L", VP8L(f.At.Dx(), f.At.Dy(), f.Color))
		body = chunk(body, "ANMF", payload)
	}
	out := append([]byte("RIFF"), 0, 0, 0, 0)
	binary.LittleEndian.PutUint32(out[4:], uint32(4+len(body))) //nolint:gosec // a test file's length
	return append(append(out, "WEBP"...), body...)
}

// Still is a WebP file of one picture, w×h pixels of c.
func Still(w, h int, c color.NRGBA) []byte {
	body := chunk(nil, "VP8L", VP8L(w, h, c))
	out := append([]byte("RIFF"), 0, 0, 0, 0)
	binary.LittleEndian.PutUint32(out[4:], uint32(4+len(body))) //nolint:gosec // a test file's length
	return append(append(out, "WEBP"...), body...)
}

// VP8L is a lossless bitstream of w×h pixels of c: no transform, and a
// prefix code of a single symbol for each channel, so its pixels take
// no bits.
func VP8L(w, h int, c color.NRGBA) []byte {
	var b bits
	b.put(0x2f, 8)
	b.put(uint32(w-1), 14) //nolint:gosec // a test's size
	b.put(uint32(h-1), 14) //nolint:gosec // a test's size
	b.put(1, 1)
	b.put(0, 3)
	b.put(0, 1) // no transform
	b.put(0, 1) // no colour cache
	b.put(0, 1) // no meta prefix codes
	for _, v := range []uint8{c.G, c.R, c.B, c.A, 0} {
		b.put(1, 1) // a simple code
		b.put(0, 1) // of one symbol
		b.put(1, 1) // of 8 bits
		b.put(uint32(v), 8)
	}
	return b.out
}

type bits struct {
	out []byte
	n   int
}

// put writes v's n low bits, the lowest first.
func (b *bits) put(v uint32, n int) {
	for i := range n {
		if b.n%8 == 0 {
			b.out = append(b.out, 0)
		}
		b.out[len(b.out)-1] |= byte(v>>i&1) << (b.n % 8)
		b.n++
	}
}

func put24(b []byte, v int) { b[0], b[1], b[2] = byte(v), byte(v>>8), byte(v>>16) } //nolint:gosec // 24 bits

func chunk(out []byte, kind string, data []byte) []byte {
	out = append(out, kind...)
	out = binary.LittleEndian.AppendUint32(out, uint32(len(data))) //nolint:gosec // a test file's length
	out = append(out, data...)
	if len(data)%2 == 1 {
		out = append(out, 0)
	}
	return out
}
