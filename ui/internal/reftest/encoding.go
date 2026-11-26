package reftest

import (
	"bufio"
	"bytes"
	"mime"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/net/html/charset"
)

// boms are the byte order marks a browser decodes by, whatever else a
// file says.
//
//nolint:gochecknoglobals // the table
var boms = []struct {
	mark     string
	encoding string
}{
	{"\xef\xbb\xbf", "utf-8"},
	{"\xfe\xff", "utf-16be"},
	{"\xff\xfe", "utf-16le"},
}

// The declarations of a page's encoding in its first bytes: an XML
// declaration's, and a meta element's, as charset or in a content type.
var (
	xmlEncoding  = regexp.MustCompile(`^<\?xml[^>]*?\sencoding\s*=\s*["']([^"']+)["']`)
	metaEncoding = regexp.MustCompile(`(?i)<meta[^>]*?charset\s*=\s*["']?([^"'\s/>;]+)`)
)

// servedType is the Content-Type the .headers file beside the file at
// name gives it, as WPT's server sends it, "" for none.
func servedType(name string) string {
	f, err := os.Open(name + ".headers") //nolint:gosec // a file of the checkout
	if err != nil {
		return ""
	}
	defer func() { _ = f.Close() }()
	s := bufio.NewScanner(f)
	for s.Scan() {
		if key, value, ok := strings.Cut(s.Text(), ":"); ok && strings.EqualFold(strings.TrimSpace(key), "Content-Type") {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

// served is the charset the file at name is served with, "" for none.
func served(name string) string {
	if _, params, err := mime.ParseMediaType(servedType(name)); err == nil {
		return params["charset"]
	}
	return ""
}

// declaresEncoding reports whether an HTML page says its encoding by a
// byte order mark or a meta element in its first 1,024 bytes.
func declaresEncoding(data []byte) bool {
	for _, b := range boms {
		if bytes.HasPrefix(data, []byte(b.mark)) {
			return true
		}
	}
	return metaEncoding.Match(data[:min(len(data), 1024)])
}

// servedAsCSS reports whether the file at name is served as a sheet, by
// its .headers or else its extension, as WPT's server types it: a page
// in standards mode drops a sheet of another type.
func servedAsCSS(name string) bool {
	kind := servedType(name)
	if kind == "" {
		kind = mime.TypeByExtension(filepath.Ext(name))
	}
	t, _, err := mime.ParseMediaType(kind)
	return err == nil && t == "text/css"
}

// servedSheet is a sheet's bytes as ui reads them: decoded in the
// charset the sheet at file is served with, when it has one and no byte
// order mark of its own, and given as UTF-8 behind one.
func servedSheet(data []byte, file string) []byte {
	name := encodingName(served(file))
	if name == "" {
		return data
	}
	for _, b := range boms {
		if bytes.HasPrefix(data, []byte(b.mark)) {
			return data
		}
	}
	text, _ := decodeIn(data, name)
	return append([]byte(boms[0].mark), text...)
}

// decodePage is the text of a page's bytes and the name of the encoding
// they were read in, as a browser decides it: its byte order mark, the
// charset it was served with, its XML declaration or, in HTML, its meta
// element in the first 1,024 bytes, else UTF-8 (WPT's pages are written
// in it; the runner serves an HTML page without a declaration as UTF-8
// to Chrome too).
func decodePage(data []byte, served string, isXML bool) (string, string) {
	for _, b := range boms {
		if bytes.HasPrefix(data, []byte(b.mark)) {
			return decodeIn(data[len(b.mark):], b.encoding)
		}
	}
	if name := encodingName(served); name != "" {
		return decodeIn(data, name)
	}
	head := data[:min(len(data), 1024)]
	declared := metaEncoding
	if isXML {
		declared = xmlEncoding
	}
	if m := declared.FindSubmatch(head); m != nil {
		if name := encodingName(string(m[1])); name != "" {
			if strings.HasPrefix(name, "utf-16") {
				name = "utf-8"
			}
			return decodeIn(data, name)
		}
	}
	return decodeIn(data, "utf-8")
}

// encodingName is the WHATWG name of the encoding label names, "" for
// none.
func encodingName(label string) string {
	if label == "" {
		return ""
	}
	_, name := charset.Lookup(label)
	return name
}

func decodeIn(data []byte, name string) (string, string) {
	e, _ := charset.Lookup(name)
	text, err := e.NewDecoder().Bytes(data)
	if err != nil {
		return string(data), name
	}
	return string(text), name
}
