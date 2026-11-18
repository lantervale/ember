package ui

import (
	"encoding/base64"
	"net/url"
	"strings"
)

// dataURL is what a data: address holds: its bytes after the comma, from
// base64 when its media type ends with ;base64, percent-decoded
// otherwise. ok is false for any other address and for one that doesn't
// read.
func dataURL(address string) ([]byte, bool) {
	const scheme = "data:"
	if len(address) < len(scheme) || !strings.EqualFold(address[:len(scheme)], scheme) {
		return nil, false
	}
	kind, body, found := strings.Cut(address[len(scheme):], ",")
	if !found {
		return nil, false
	}
	if i := strings.LastIndexByte(kind, ';'); i >= 0 && strings.EqualFold(strings.TrimSpace(kind[i+1:]), "base64") {
		b, err := base64.RawStdEncoding.DecodeString(strings.TrimRight(strings.TrimSpace(body), "="))
		return b, err == nil
	}
	text, err := url.PathUnescape(body)
	return []byte(text), err == nil
}
