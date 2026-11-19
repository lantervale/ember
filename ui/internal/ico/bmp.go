package ico

import (
	"encoding/binary"
	"errors"
	"image"
	"io"
)

// fileHeader is the length of what a BMP file starts with, before its
// bitmap: "BM", the file's length, and where the rows start.
const fileHeader = 14

var errNoBMP = errors.New("ico: not a BMP file")

// DecodeBMP reads a BMP file: uncompressed rows of 1, 4, 8, 16, 24 or
// 32 bits a pixel, from the bottom or from the top, the 16 and 32 bit
// ones through bit masks when the file has them. It has alpha only
// where its header names the alpha bits. A run-length encoded file is
// an error.
func DecodeBMP(r io.Reader) (image.Image, error) {
	data, err := readAll(r)
	if err != nil {
		return nil, err
	}
	d, err := bmp(data)
	if err != nil {
		return nil, err
	}
	return d.decode(data[fileHeader:]), nil
}

// DecodeBMPConfig is the size of the picture a BMP file holds.
func DecodeBMPConfig(r io.Reader) (image.Config, error) {
	data, err := readAll(r)
	if err != nil {
		return image.Config{}, err
	}
	d, err := bmp(data)
	if err != nil {
		return image.Config{}, err
	}
	return d.config(), nil
}

func bmp(data []byte) (*dib, error) {
	if len(data) < fileHeader {
		return nil, errShort
	}
	if data[0] != 'B' || data[1] != 'M' {
		return nil, errNoBMP
	}
	pix := binary.LittleEndian.Uint32(data[10:])
	if pix < fileHeader || uint64(pix) > uint64(len(data)) {
		return nil, errOffset
	}
	return readDIB(data[fileHeader:], int(pix)-fileHeader)
}
