package ico

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"image/png"
	"io"
)

const (
	// dirHeader is the length of what an ICO file starts with: a zero,
	// its kind (1, an icon, 2 a cursor) and how many pictures it holds.
	dirHeader = 6
	// dirEntry is the length of what the file says of each picture.
	dirEntry = 16

	pngSignature = "\x89PNG\r\n\x1a\n"

	icon   = 1
	cursor = 2
)

var (
	errNoICO   = errors.New("ico: not an ICO file")
	errEmpty   = errors.New("ico: the file holds no picture")
	errEntry   = errors.New("ico: a picture is outside the file")
	errPNGSize = errors.New("ico: the PNG picture has no size")
)

// Decode reads an ICO file, or a CUR file, a cursor's, which is one of
// another kind: the largest of its pictures, the one of the most bits a
// pixel among those of that size, as Chrome shows it whatever the size
// it is shown at. A picture is a PNG file, or a bitmap as a BMP file's,
// of 1, 4, 8, 16, 24 or 32 bits a pixel, run-length encoded or not; the
// bitmap's AND mask cuts it where it has no alpha, and one of 32 bits
// whose alpha is zero everywhere is cut by the mask too.
func Decode(r io.Reader) (image.Image, error) {
	data, err := readAll(r)
	if err != nil {
		return nil, err
	}
	pic, err := best(data)
	if err != nil {
		return nil, err
	}
	if isPNG(pic) {
		if _, err = pngConfig(pic); err != nil {
			return nil, err
		}
		return png.Decode(bytes.NewReader(pic))
	}
	d, err := readDIB(pic, -1)
	if err != nil {
		return nil, err
	}
	return d.decode(pic), nil
}

// DecodeConfig is the size of the picture Decode reads of an ICO file.
func DecodeConfig(r io.Reader) (image.Config, error) {
	data, err := readAll(r)
	if err != nil {
		return image.Config{}, err
	}
	pic, err := best(data)
	if err != nil {
		return image.Config{}, err
	}
	if isPNG(pic) {
		return pngConfig(pic)
	}
	d, err := readDIB(pic, -1)
	if err != nil {
		return image.Config{}, err
	}
	return d.config(), nil
}

func isPNG(pic []byte) bool { return bytes.HasPrefix(pic, []byte(pngSignature)) }

// pngConfig is the size a PNG picture says it has, an error when it is
// over what a reader builds.
func pngConfig(pic []byte) (image.Config, error) {
	cfg, err := png.DecodeConfig(bytes.NewReader(pic))
	switch {
	case err != nil:
		return cfg, err
	case cfg.Width <= 0 || cfg.Height <= 0:
		return cfg, errPNGSize
	case cfg.Width > MaxSide || cfg.Height > MaxSide || cfg.Width*cfg.Height > MaxPixels:
		return cfg, errTooLarge
	}
	return cfg, nil
}

// best is the bytes of the picture of an ICO file to show: the largest
// by what the directory says, then the one of the most bits a pixel.
// Every entry must lie in the file, past the directory.
func best(data []byte) ([]byte, error) {
	if len(data) < dirHeader {
		return nil, errShort
	}
	kind := binary.LittleEndian.Uint16(data[2:])
	if binary.LittleEndian.Uint16(data) != 0 || kind != icon && kind != cursor {
		return nil, errNoICO
	}
	n := int(binary.LittleEndian.Uint16(data[4:]))
	if n == 0 {
		return nil, errEmpty
	}
	pictures := dirHeader + dirEntry*n
	if len(data) < pictures {
		return nil, errShort
	}
	var pic []byte
	area, depth := 0, 0
	for i := range n {
		e := data[dirHeader+dirEntry*i:][:dirEntry]
		size, at := uint64(binary.LittleEndian.Uint32(e[8:])), uint64(binary.LittleEndian.Uint32(e[12:]))
		if at < uint64(pictures) || at+size > uint64(len(data)) {
			return nil, errEntry
		}
		entry := data[at : at+size]
		a, bpp := side(e[0])*side(e[1]), depthOf(e, entry, kind == cursor)
		if pic == nil || a > area || a == area && bpp > depth {
			pic, area, depth = entry, a, bpp
		}
	}
	return pic, nil
}

// side is a width or a height of the directory, where 0 stands for 256.
func side(b byte) int {
	if b == 0 {
		return 256
	}
	return int(b)
}

// depthOf is the bits a pixel of an entry: what the directory says, or,
// where it says nothing, what the picture does (a PNG counting as 32).
// A cursor's directory says its hotspot there, so the picture says it.
func depthOf(e, pic []byte, cursor bool) int {
	if bpp := int(binary.LittleEndian.Uint16(e[6:])); bpp != 0 && !cursor {
		return bpp
	}
	if isPNG(pic) {
		return 32
	}
	if len(pic) < infoHeader {
		return 0
	}
	return int(binary.LittleEndian.Uint16(pic[14:]))
}
