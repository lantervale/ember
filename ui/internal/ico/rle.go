package ico

import (
	"image"
	"math"
)

// The escapes of a run-length encoded bitmap: a run of no pixels whose
// second byte says what to do.
const (
	endOfLine   = 0
	endOfBitmap = 1
	delta       = 2
)

// decodeRuns writes the pixels the runs of a run-length encoded bitmap
// give, rows from the bottom, as Chrome reads them: a pixel no run
// reaches is opaque black, a run of one colour past the end of its row
// is cut there, and a move or a run of pixels written out that would
// leave the picture ends it, as the runs ending do.
func (d *dib) decodeRuns(runs []byte, img *image.NRGBA) {
	for i := 3; i < len(img.Pix); i += 4 {
		img.Pix[i] = math.MaxUint8
	}
	x, y := 0, 0
	put := func(i byte) {
		c := d.palette[i]
		at := img.PixOffset(x, d.h-1-y)
		img.Pix[at], img.Pix[at+1], img.Pix[at+2] = c[0], c[1], c[2]
		x++
	}
	for len(runs) >= 2 && y < d.h {
		n, code := int(runs[0]), runs[1]
		runs = runs[2:]
		switch {
		case n > 0:
			for k := range min(n, d.w-x) {
				put(d.nibble(code, k))
			}
		case code == endOfLine:
			x, y = 0, y+1
		case code == endOfBitmap:
			return
		case code == delta:
			if len(runs) < 2 || x+int(runs[0]) > d.w || y+int(runs[1]) > d.h {
				return
			}
			x, y, runs = x+int(runs[0]), y+int(runs[1]), runs[2:]
		default:
			n = int(code)
			bytes := (n*d.bpp + 7) / 8
			if x+n > d.w || len(runs) < bytes {
				return
			}
			for k := range n {
				put(d.nibble(runs[k*d.bpp/8], k))
			}
			runs = runs[min(len(runs), (bytes+1)/2*2):]
		}
	}
}

// nibble is the k-th index of a run's byte b: b itself in a bitmap of 8
// bits, else its high nibble for even k and its low one for odd k.
func (d *dib) nibble(b byte, k int) byte {
	switch {
	case d.bpp == 8:
		return b
	case k%2 == 0:
		return b >> 4
	}
	return b & 0xf
}
