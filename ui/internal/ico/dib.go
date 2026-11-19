// Package ico reads the pictures Windows left the web: an ICO file, as
// a site's favicon is, and a BMP file. Both hold a device-independent
// bitmap, which an ICO may swap for a PNG. The bytes come from the
// network, so every size, offset and count is checked against what is
// there, nothing larger than MaxSide and MaxPixels is built, and a file
// that says anything else is an error, never a panic.
package ico

import (
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/color"
	"io"
	"math"
	"math/bits"
)

const (
	// MaxSide is the longest side of a picture a reader builds, in
	// pixels, and MaxPixels the most pixels: the limits of ember's svg
	// reader.
	MaxSide = 16384
	// MaxPixels is the most pixels of a picture a reader builds.
	MaxPixels = 1 << 25

	// maxFile is the longest file read: the largest picture at four
	// bytes a pixel, and room for its headers.
	maxFile = 4*MaxPixels + 1<<20
)

const (
	infoHeader   = 40
	infoHeaderV2 = 52
	infoHeaderV3 = 56
	infoHeaderV4 = 108
	infoHeaderV5 = 124

	compRGB            = 0
	compRLE8           = 1
	compRLE4           = 2
	compBitfields      = 3
	compAlphaBitfields = 6
)

var (
	errShort       = errors.New("ico: the file ends early")
	errFileSize    = fmt.Errorf("ico: the file is over %d bytes", maxFile)
	errTooLarge    = fmt.Errorf("ico: the picture is over %d pixels a side or %d pixels", MaxSide, MaxPixels)
	errNoSize      = errors.New("ico: the picture has no size")
	errHeader      = errors.New("ico: the bitmap header is none this reader knows")
	errDepth       = errors.New("ico: the bitmap's bits per pixel are none this reader knows")
	errRLE         = errors.New("ico: run-length encoded bitmaps are not read")
	errCompression = errors.New("ico: the bitmap's compression is none this reader knows")
	errPalette     = errors.New("ico: the bitmap's palette is larger than its bits per pixel allow")
	errOffset      = errors.New("ico: the bitmap's pixels start outside the file")
)

// channel is one colour of a pixel of 16 or 32 bits: its bits, and how
// far right they go.
type channel struct {
	mask  uint32
	shift int
}

func channelOf(mask uint32) channel {
	return channel{mask: mask, shift: bits.TrailingZeros32(mask)}
}

// of is the channel's value in px, as 8 bits.
func (c channel) of(px uint32) uint8 {
	if c.mask == 0 {
		return 0
	}
	most := uint64(c.mask >> c.shift)
	return uint8((uint64(px&c.mask>>c.shift)*math.MaxUint8 + most/2) / most) //nolint:gosec // at most 255
}

// dib is a bitmap's header, read: what its rows hold and where they are.
type dib struct {
	w, h    int
	bpp     int
	topDown bool
	// icon is a bitmap of an ICO file: its AND mask follows its rows.
	icon     bool
	channels [4]channel
	palette  [256][3]uint8
	// pix is where the rows start in the bitmap's bytes.
	pix int
}

// stride is the bytes of a row of w pixels of bpp bits, a multiple of
// four.
func stride(w, bpp int) int { return (w*bpp + 31) / 32 * 4 }

// readDIB reads the header of the bitmap data starts with, its masks
// and its palette. In a BMP file pix is where the file says the rows
// start; an ICO's bitmap gives -1, its rows following its palette and
// its height counting the AND mask's rows too.
func readDIB(data []byte, pix int) (*dib, error) {
	if len(data) < infoHeader {
		return nil, errShort
	}
	size := binary.LittleEndian.Uint32(data)
	switch size {
	case infoHeader, infoHeaderV2, infoHeaderV3, infoHeaderV4, infoHeaderV5:
	default:
		return nil, errHeader
	}
	if len(data) < int(size) {
		return nil, errShort
	}
	d := &dib{icon: pix < 0, bpp: int(binary.LittleEndian.Uint16(data[14:]))}
	if err := d.setSize(int32(binary.LittleEndian.Uint32(data[4:])), int32(binary.LittleEndian.Uint32(data[8:]))); err != nil { //nolint:gosec,lll // signed fields
		return nil, err
	}
	switch d.bpp {
	case 1, 4, 8, 16, 24, 32:
	default:
		return nil, errDepth
	}
	after, err := d.setChannels(data, int(size))
	if err != nil {
		return nil, err
	}
	colours, err := d.setPalette(data, after)
	if err != nil {
		return nil, err
	}
	d.pix = pix
	if d.icon {
		d.pix = colours
	}
	if d.pix < after || d.pix > len(data) {
		return nil, errOffset
	}
	need := stride(d.w, d.bpp) * d.h
	if d.icon && d.bpp != 32 {
		need += stride(d.w, 1) * d.h
	}
	if len(data)-d.pix < need {
		return nil, errShort
	}
	return d, nil
}

// setSize takes the header's width and height: a height below zero is
// rows from the top, and an icon's is twice its picture's.
func (d *dib) setSize(w, h int32) error {
	if h < 0 {
		if h == math.MinInt32 || d.icon {
			return errHeader
		}
		d.topDown, h = true, -h
	}
	if d.icon {
		if h%2 != 0 {
			return errHeader
		}
		h /= 2
	}
	if w <= 0 || h <= 0 {
		return errNoSize
	}
	if w > MaxSide || h > MaxSide || int(w)*int(h) > MaxPixels {
		return errTooLarge
	}
	d.w, d.h = int(w), int(h)
	return nil
}

// setChannels reads the compression and the bit masks of a header of
// size bytes, and returns where what follows them starts.
func (d *dib) setChannels(data []byte, size int) (int, error) {
	comp := binary.LittleEndian.Uint32(data[16:])
	switch comp {
	case compRGB:
	case compBitfields, compAlphaBitfields:
		if d.bpp != 16 && d.bpp != 32 {
			return 0, errHeader
		}
	case compRLE8, compRLE4:
		return 0, errRLE
	default:
		return 0, errCompression
	}
	// Without masks, 32 bits have alpha in an icon alone, as in Chrome,
	// unless a V4 header or later names the alpha bits.
	masks := [4]uint32{0xff0000, 0xff00, 0xff, 0}
	if d.bpp == 16 {
		masks = [4]uint32{0x7c00, 0x3e0, 0x1f, 0}
	}
	after := size
	switch {
	case comp == compRGB:
		switch {
		case d.bpp != 32:
		case d.icon:
			masks[3] = 0xff000000
		case size >= infoHeaderV4:
			masks[3] = binary.LittleEndian.Uint32(data[52:])
		}
	case size >= infoHeaderV2:
		masks[3] = 0
		for i := range (size - infoHeader) / 4 {
			if i < len(masks) {
				masks[i] = binary.LittleEndian.Uint32(data[infoHeader+4*i:])
			}
		}
	default:
		n := 3
		if comp == compAlphaBitfields {
			n = 4
		}
		if len(data) < size+4*n {
			return 0, errShort
		}
		masks[3] = 0
		for i := range n {
			masks[i] = binary.LittleEndian.Uint32(data[size+4*i:])
		}
		after += 4 * n
	}
	for i, m := range masks {
		d.channels[i] = channelOf(m)
	}
	return after, nil
}

// setPalette reads the colours of a bitmap of 8 bits or fewer, which
// start at after, and returns where they end. The file must hold them
// all, though a BMP file's rows may start before they end, as Chrome
// reads one.
func (d *dib) setPalette(data []byte, after int) (int, error) {
	if d.bpp > 8 {
		return after, nil
	}
	n := 1 << d.bpp
	if used := binary.LittleEndian.Uint32(data[32:]); used != 0 {
		if used > uint32(n) { //nolint:gosec // at most 256
			return 0, errPalette
		}
		n = int(used)
	}
	if len(data)-after < 4*n {
		return 0, errShort
	}
	for i := range n {
		c := data[after+4*i:]
		d.palette[i] = [3]uint8{c[2], c[1], c[0]}
	}
	return after + 4*n, nil
}

func (d *dib) config() image.Config {
	return image.Config{ColorModel: color.NRGBAModel, Width: d.w, Height: d.h}
}

// decode builds the picture of the bitmap data holds, as Chrome shows
// it: a BMP file's alpha is what its masks say, and an icon's alpha
// bits that are all zero say nothing, the picture then cut by its AND
// mask, as an icon without alpha bits is.
func (d *dib) decode(data []byte) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, d.w, d.h))
	rows := stride(d.w, d.bpp)
	alpha := false
	for y := range d.h {
		row := data[d.pix+rows*y:][:rows]
		out := img.Pix[d.line(y)*img.Stride:][:4*d.w]
		switch d.bpp {
		case 1, 4, 8:
			d.indexed(row, out)
		case 24:
			for x := range d.w {
				out[4*x], out[4*x+1], out[4*x+2], out[4*x+3] = row[3*x+2], row[3*x+1], row[3*x], math.MaxUint8
			}
		default:
			alpha = d.masked(row, out) || alpha
		}
	}
	if alpha || !d.icon && d.channels[3].mask != 0 {
		return img
	}
	for i := 3; i < len(img.Pix); i += 4 {
		img.Pix[i] = math.MaxUint8
	}
	if d.icon {
		d.cut(data, img)
	}
	return img
}

// line is the picture's line the y-th row of the file is.
func (d *dib) line(y int) int {
	if d.topDown {
		return y
	}
	return d.h - 1 - y
}

// indexed writes a row of palette indices, the leftmost pixel in the
// high bits of its byte.
func (d *dib) indexed(row, out []byte) {
	for x := range d.w {
		bit := x * d.bpp
		c := d.palette[row[bit/8]>>(8-d.bpp-bit%8)&(1<<d.bpp-1)]
		out[4*x], out[4*x+1], out[4*x+2], out[4*x+3] = c[0], c[1], c[2], math.MaxUint8
	}
}

// masked writes a row of pixels of 16 or 32 bits through the bitmap's
// channels, and reports whether any has alpha.
func (d *dib) masked(row, out []byte) bool {
	alpha := false
	for x := range d.w {
		var px uint32
		if d.bpp == 16 {
			px = uint32(binary.LittleEndian.Uint16(row[2*x:]))
		} else {
			px = binary.LittleEndian.Uint32(row[4*x:])
		}
		for i, c := range d.channels {
			out[4*x+i] = c.of(px)
		}
		alpha = alpha || out[4*x+3] != 0
	}
	return alpha
}

// cut clears the pixels of img the icon's AND mask sets: a bit a pixel
// after the bitmap's rows, in rows as theirs. An icon of 32 bits may
// end without one, and is then left whole.
func (d *dib) cut(data []byte, img *image.NRGBA) {
	rows := stride(d.w, 1)
	mask := data[d.pix+stride(d.w, d.bpp)*d.h:]
	if len(mask) < rows*d.h {
		return
	}
	for y := range d.h {
		row := mask[rows*y:][:rows]
		out := img.Pix[d.line(y)*img.Stride:]
		for x := range d.w {
			if row[x/8]>>(7-x%8)&1 == 1 {
				out[4*x], out[4*x+1], out[4*x+2], out[4*x+3] = 0, 0, 0, 0
			}
		}
	}
}

// readAll is the file r holds, an error when it is over maxFile.
func readAll(r io.Reader) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, maxFile+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxFile {
		return nil, errFileSize
	}
	return data, nil
}
