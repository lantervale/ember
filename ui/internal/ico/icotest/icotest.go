// Package icotest writes ICO and BMP files, for the tests and the test
// pages of what reads them: no picture file is kept in the tree.
package icotest

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"slices"
)

const (
	fileHeader   = 14
	infoHeader   = 40
	infoHeaderV2 = 52
	dirHeader    = 6
	dirEntry     = 16
)

// Bitmap is a bitmap to write: At gives the value of each pixel as its
// rows hold it (a palette index, or the pixel's bits), Hole the pixels
// an icon's AND mask cuts. Header is the header's length, 40 when zero;
// Masks are written in a header of 52 bytes or more, after it otherwise.
// Runs, when set, are written in place of the rows: a run-length encoded
// bitmap's (Comp 1 or 2), which At then plays no part in.
type Bitmap struct {
	W, H, BPP int
	Header    int
	Comp      uint32
	Masks     []uint32
	Palette   []color.NRGBA
	Used      uint32
	TopDown   bool
	At        func(x, y int) uint32
	Hole      func(x, y int) bool
	Runs      []byte
}

// DIB is the bitmap's bytes without a file header, and where its rows
// start: its palette whole unless Used counts its colours. An icon's
// has its height doubled and its AND mask after its rows.
func (b Bitmap) DIB(icon bool) ([]byte, int) {
	size := b.Header
	if size == 0 {
		size = infoHeader
	}
	head := make([]byte, size)
	h := b.H
	if icon {
		h *= 2
	}
	if b.TopDown {
		h = -h
	}
	binary.LittleEndian.PutUint32(head, uint32(size))
	binary.LittleEndian.PutUint32(head[4:], uint32(b.W))    //nolint:gosec // as the file holds it
	binary.LittleEndian.PutUint32(head[8:], uint32(h))      //nolint:gosec // as the file holds it
	binary.LittleEndian.PutUint16(head[12:], 1)             // planes
	binary.LittleEndian.PutUint16(head[14:], uint16(b.BPP)) //nolint:gosec // as the file holds it
	binary.LittleEndian.PutUint32(head[16:], b.Comp)
	binary.LittleEndian.PutUint32(head[32:], b.Used)
	var after []byte
	for i, m := range b.Masks {
		if size >= infoHeaderV2 {
			binary.LittleEndian.PutUint32(head[infoHeader+4*i:], m)
		} else {
			after = binary.LittleEndian.AppendUint32(after, m)
		}
	}
	for _, c := range b.Palette {
		after = append(after, c.B, c.G, c.R, 0)
	}
	if b.BPP <= 8 && b.Used == 0 {
		after = append(after, make([]byte, 4*(1<<b.BPP-len(b.Palette)))...)
	}
	pix := size + len(after)
	rows := b.Runs
	if rows == nil {
		rows = b.rows(b.BPP, b.At)
	} else {
		binary.LittleEndian.PutUint32(head[20:], uint32(len(rows))) //nolint:gosec // a test file's length
	}
	if !icon {
		return slices.Concat(head, after, rows), pix
	}
	mask := b.rows(1, func(x, y int) uint32 {
		if b.Hole != nil && b.Hole(x, y) {
			return 1
		}
		return 0
	})
	return slices.Concat(head, after, rows, mask), pix
}

func (b Bitmap) rows(bpp int, at func(x, y int) uint32) []byte {
	n := (b.W*bpp + 31) / 32 * 4
	out := make([]byte, n*b.H)
	for i := range b.H {
		y := b.H - 1 - i
		if b.TopDown {
			y = i
		}
		row := out[n*i:][:n]
		for x := range b.W {
			v := at(x, y)
			switch bpp {
			case 1, 4, 8:
				row[x*bpp/8] |= byte(v << (8 - bpp - x*bpp%8)) //nolint:gosec // the pixel's bits
			case 16:
				binary.LittleEndian.PutUint16(row[2*x:], uint16(v)) //nolint:gosec // the pixel's bits
			case 24:
				row[3*x], row[3*x+1], row[3*x+2] = byte(v), byte(v>>8), byte(v>>16) //nolint:gosec // the pixel's bits
			default:
				binary.LittleEndian.PutUint32(row[4*x:], v)
			}
		}
	}
	return out
}

// File is the bitmap as a BMP file.
func (b Bitmap) File() []byte {
	data, pix := b.DIB(false)
	out := make([]byte, fileHeader, fileHeader+len(data))
	out[0], out[1] = 'B', 'M'
	binary.LittleEndian.PutUint32(out[2:], uint32(fileHeader+len(data))) //nolint:gosec // a test file's length
	binary.LittleEndian.PutUint32(out[10:], uint32(fileHeader+pix))      //nolint:gosec // a test file's length
	return append(out, data...)
}

// Entry is one picture of an ICO file to write: what the directory says
// of it, and its bytes.
type Entry struct {
	W, H, BPP int
	Data      []byte
}

// Entry is the bitmap as a picture of an ICO file.
func (b Bitmap) Entry() Entry {
	data, _ := b.DIB(true)
	return Entry{b.W, b.H, b.BPP, data}
}

// ICO is an ICO file of entries, in their order.
func ICO(entries ...Entry) []byte { return file(1, entries) }

// CUR is a cursor file of entries, in their order: an ICO file of kind
// 2, each entry's hotspot at 1,2 where an icon's says its planes and its
// bits a pixel.
func CUR(entries ...Entry) []byte {
	data := file(2, entries)
	for i := range entries {
		at := data[dirHeader+dirEntry*i:]
		binary.LittleEndian.PutUint16(at[4:], 1)
		binary.LittleEndian.PutUint16(at[6:], 2)
	}
	return data
}

func file(kind uint16, entries []Entry) []byte {
	dir := make([]byte, dirHeader+dirEntry*len(entries))
	binary.LittleEndian.PutUint16(dir[2:], kind)
	binary.LittleEndian.PutUint16(dir[4:], uint16(len(entries))) //nolint:gosec // a few
	var pics []byte
	for i, e := range entries {
		at := dir[dirHeader+dirEntry*i:]
		at[0], at[1] = byte(e.W), byte(e.H) //nolint:gosec // 256 is written 0
		binary.LittleEndian.PutUint16(at[4:], 1)
		binary.LittleEndian.PutUint16(at[6:], uint16(e.BPP))               //nolint:gosec // as the file holds it
		binary.LittleEndian.PutUint32(at[8:], uint32(len(e.Data)))         //nolint:gosec // a test file's length
		binary.LittleEndian.PutUint32(at[12:], uint32(len(dir)+len(pics))) //nolint:gosec // a test file's length
		pics = append(pics, e.Data...)
	}
	return slices.Concat(dir, pics)
}

// PNG is a PNG file of w by h pixels of c.
func PNG(w, h int, c color.NRGBA) []byte {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = c.R, c.G, c.B, c.A
	}
	var b bytes.Buffer
	_ = png.Encode(&b, img)
	return b.Bytes()
}
