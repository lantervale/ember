package woff

import (
	"encoding/binary"
	"math"
)

// First is the first font of the collection data as a file of its own,
// as Chrome draws a collection an @font-face names, whatever fragment
// its address ends in: its tables copied after a directory of their own.
func First(data []byte) ([]byte, error) {
	be := binary.BigEndian
	if len(data) < 16 || be.Uint32(data) != sigTTCF {
		return nil, errShort
	}
	if be.Uint32(data[8:]) == 0 {
		return nil, errCollection
	}
	off := uint64(be.Uint32(data[12:]))
	if off+12 > uint64(len(data)) {
		return nil, errShort
	}
	version := be.Uint32(data[off:])
	n := uint64(be.Uint16(data[off+4:]))
	if off+12+16*n > uint64(len(data)) {
		return nil, errShort
	}
	head := 12 + 16*n
	size := head
	for i := range n {
		rec := data[off+12+16*i:]
		at, length := uint64(be.Uint32(rec[8:])), uint64(be.Uint32(rec[12:]))
		if at+length > uint64(len(data)) {
			return nil, errShort
		}
		size += pad4(length)
	}
	if size > MaxSize || size > math.MaxInt {
		return nil, errTooLarge
	}
	out := make([]byte, size)
	copy(out, data[off:off+12])
	be.PutUint32(out, version)
	at := head
	for i := range n {
		rec := data[off+12+16*i:]
		src, length := uint64(be.Uint32(rec[8:])), uint64(be.Uint32(rec[12:]))
		dst := out[12+16*i:]
		copy(dst, rec[:16])
		be.PutUint32(dst[8:], uint32(at)) //nolint:gosec // under MaxSize
		copy(out[at:], data[src:src+length])
		at += pad4(length)
	}
	return out, nil
}
