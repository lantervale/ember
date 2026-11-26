package css

import (
	"bytes"
	"strings"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/htmlindex"
)

// Decode is the text of a sheet's bytes and the name of the encoding it
// was read in, as CSS Syntax decodes a sheet: its byte order mark first,
// then its @charset, then environment, the encoding of what refers to it
// (its document's, or the sheet that imports it), then UTF-8. Names and
// labels are the Encoding Standard's. The charset a sheet was served
// with comes before its @charset: a fetcher that knows one decodes the
// sheet and gives it back as UTF-8 behind a byte order mark.
func Decode(data []byte, environment string) (string, string) {
	for _, b := range []struct{ mark, name string }{
		{"\xef\xbb\xbf", encodingUTF8}, {"\xfe\xff", "utf-16be"}, {"\xff\xfe", "utf-16le"},
	} {
		if bytes.HasPrefix(data, []byte(b.mark)) {
			return decodeIn(data[len(b.mark):], b.name)
		}
	}
	if name := charsetRule(data); name != "" {
		return decodeIn(data, name)
	}
	if e, err := htmlindex.Get(environment); err == nil {
		if name, nameErr := htmlindex.Name(e); nameErr == nil {
			return decodeIn(data, name)
		}
	}
	return string(data), encodingUTF8
}

// charsetRule is the encoding the @charset rule data starts with names,
// byte for byte as CSS Syntax reads it in the first 1,024 bytes, UTF-8
// for UTF-16, "" for none.
func charsetRule(data []byte) string {
	const lead = `@charset "`
	head := data[:min(len(data), 1024)]
	if !bytes.HasPrefix(head, []byte(lead)) {
		return ""
	}
	label, _, ok := strings.Cut(string(head[len(lead):]), `";`)
	if !ok || strings.Contains(label, `"`) {
		return ""
	}
	e, err := htmlindex.Get(label)
	if err != nil {
		return ""
	}
	name, err := htmlindex.Name(e)
	if err != nil {
		return ""
	}
	if name == "utf-16be" || name == "utf-16le" {
		return encodingUTF8
	}
	return name
}

// encodingUTF8 is UTF-8's name.
const encodingUTF8 = "utf-8"

func decodeIn(data []byte, name string) (string, string) {
	if name == encodingUTF8 {
		return string(data), name
	}
	e, err := htmlindex.Get(name)
	if err != nil {
		e = encoding.Nop
	}
	text, err := e.NewDecoder().Bytes(data)
	if err != nil {
		return string(data), name
	}
	return string(text), name
}
