package svg

import (
	"bytes"
	"encoding/xml"
	"errors"
	"io"
	"strings"
	utf16std "unicode/utf16"
)

const (
	svgNS     = "http://www.w3.org/2000/svg"
	elemSVG   = "svg"
	elemStyle = "style"
	none      = "none"
)

// node is an element of the file, as read, or a run of the characters
// of a text element.
type node struct {
	name    string
	attr    map[string]string
	kids    []*node
	parent  *node
	classes []string
	// text is a style element's sheet, or a run's characters.
	text string
	// chars marks a run of characters, which has no name.
	chars bool
}

// entities are the <!ENTITY name "value"> declarations of a DOCTYPE,
// which Illustrator writes its namespaces with. They are expanded here,
// once and not inside one another, against a budget: the XML decoder is
// given none, so a file cannot make it grow what it reads.
type entities struct {
	values map[string]string
	budget int
}

func (e *entities) declare(doctype string) {
	for len(e.values) < maxEntities {
		i := strings.Index(doctype, "<!ENTITY")
		if i < 0 {
			return
		}
		doctype = doctype[i+len("<!ENTITY"):]
		open := strings.IndexAny(doctype, `"'`)
		if open < 0 {
			return
		}
		end := strings.IndexByte(doctype[open+1:], doctype[open])
		if end < 0 {
			return
		}
		name := strings.TrimSpace(doctype[:open])
		if name != "" && !strings.ContainsAny(name, " \t\r\n%") {
			if e.values == nil {
				e.values = map[string]string{}
			}
			if _, dup := e.values[name]; !dup {
				e.values[name] = doctype[open+1 : open+1+end]
			}
		}
		doctype = doctype[open+end+2:]
	}
}

func (e *entities) expand(s string) (string, error) {
	if len(e.values) == 0 || !strings.Contains(s, "&") {
		return s, nil
	}
	var out strings.Builder
	for {
		i := strings.IndexByte(s, '&')
		if i < 0 {
			break
		}
		end := strings.IndexByte(s[i:], ';')
		if end < 0 {
			break
		}
		out.WriteString(s[:i])
		if v, ok := e.values[s[i+1:i+end]]; ok {
			if e.budget -= len(v); e.budget < 0 {
				return "", ErrTooComplex
			}
			out.WriteString(v)
		} else {
			out.WriteString(s[i : i+end+1])
		}
		s = s[i+end+1:]
	}
	out.WriteString(s)
	return out.String(), nil
}

// latin1 reads bytes as ISO 8859-1, for the files that declare an
// encoding other than UTF-8: every byte is a character and ASCII stays,
// so the markup reads whatever the encoding was.
type latin1 struct {
	r   io.Reader
	buf [512]byte
}

func (l *latin1) Read(p []byte) (int, error) {
	n, err := l.r.Read(l.buf[:min(len(l.buf), len(p)/2)])
	out := 0
	for _, c := range l.buf[:n] {
		if c < 0x80 {
			p[out] = c
			out++
		} else {
			p[out], p[out+1] = 0xc0|c>>6, 0x80|c&0x3f
			out += 2
		}
	}
	if n == 0 && err == nil && len(p) < 2 {
		return 0, io.ErrShortBuffer
	}
	return out, err
}

type document struct {
	root   *node
	ids    map[string]*node
	sheets []string
}

// reader builds a document from the tokens of its file.
type reader struct {
	doc      *document
	ent      entities
	stack    []*node
	elements int
}

func (r *reader) start(t xml.StartElement) error {
	if r.elements++; r.elements > MaxElements || len(r.stack) >= MaxDepth {
		return ErrTooComplex
	}
	n := &node{attr: make(map[string]string, len(t.Attr))}
	space, err := r.ent.expand(t.Name.Space)
	if err != nil {
		return err
	}
	if space == "" || space == svgNS {
		n.name = t.Name.Local
	}
	for _, a := range t.Attr {
		if a.Name.Space != "" && a.Name.Local != "href" || a.Name.Space == "xmlns" {
			continue
		}
		if _, set := n.attr[a.Name.Local]; set && a.Name.Space != "" {
			continue // href wins over xlink:href
		}
		if n.attr[a.Name.Local], err = r.ent.expand(a.Value); err != nil {
			return err
		}
	}
	if id := n.attr["id"]; id != "" {
		if _, dup := r.doc.ids[id]; !dup {
			r.doc.ids[id] = n
		}
	}
	n.classes = strings.Fields(n.attr["class"])
	if len(r.stack) > 0 {
		n.parent = r.stack[len(r.stack)-1]
		n.parent.kids = append(n.parent.kids, n)
	} else if r.doc.root == nil {
		r.doc.root = n
	}
	r.stack = append(r.stack, n)
	return nil
}

func (r *reader) end() error {
	if len(r.stack) == 0 {
		return nil
	}
	n := r.stack[len(r.stack)-1]
	r.stack = r.stack[:len(r.stack)-1]
	if n.name != elemStyle || n.text == "" {
		return nil
	}
	sheet, err := r.ent.expand(n.text)
	r.doc.sheets = append(r.doc.sheets, sheet)
	n.text = ""
	return err
}

func (r *reader) token(tok xml.Token) error {
	switch t := tok.(type) {
	case xml.Directive:
		r.ent.declare(string(t))
	case xml.StartElement:
		return r.start(t)
	case xml.EndElement:
		return r.end()
	case xml.CharData:
		if len(r.stack) == 0 {
			return nil
		}
		top := r.stack[len(r.stack)-1]
		if top.name == elemStyle {
			top.text += string(t)
			return nil
		}
		for _, n := range r.stack {
			if n.name == elemText {
				return r.chars(top, string(t))
			}
		}
	}
	return nil
}

// chars appends a run of characters to an element of a text element.
func (r *reader) chars(parent *node, s string) error {
	if n := len(parent.kids); n > 0 && parent.kids[n-1].chars {
		parent.kids[n-1].text += s
		return nil
	}
	if r.elements++; r.elements > MaxElements {
		return ErrTooComplex
	}
	parent.kids = append(parent.kids, &node{text: s, chars: true, parent: parent})
	return nil
}

// utf16 is data as UTF-8 when it is UTF-16, which its byte order mark
// or its first character, a '<', tells: big- or little-endian.
func utf16(data []byte) ([]byte, bool) {
	var big bool
	switch {
	case bytes.HasPrefix(data, []byte{0xfe, 0xff}):
		big, data = true, data[2:]
	case bytes.HasPrefix(data, []byte{0xff, 0xfe}):
		data = data[2:]
	case bytes.HasPrefix(data, []byte{0, '<'}):
		big = true
	case bytes.HasPrefix(data, []byte{'<', 0}):
	default:
		return data, false
	}
	units := make([]uint16, len(data)/2)
	for i := range units {
		if big {
			units[i] = uint16(data[2*i])<<8 | uint16(data[2*i+1])
		} else {
			units[i] = uint16(data[2*i+1])<<8 | uint16(data[2*i])
		}
	}
	return []byte(string(utf16std.Decode(units))), true
}

func readDocument(data []byte) (*document, error) {
	if len(data) > MaxBytes {
		return nil, ErrTooLarge
	}
	data, wide := utf16(data)
	// The decoder is given no entity and follows no reference: it reads
	// these bytes and nothing else.
	d := xml.NewDecoder(bytes.NewReader(data)) //nolint:gosec // untrusted on purpose: bounded by the limits of svg.go
	d.Strict = false
	d.CharsetReader = func(charset string, in io.Reader) (io.Reader, error) {
		switch strings.ToLower(charset) {
		case "utf-8", "utf8", "us-ascii", "ascii":
			return in, nil
		case "utf-16", "utf-16le", "utf-16be":
			if wide {
				return in, nil
			}
		}
		return &latin1{r: in}, nil
	}
	r := &reader{doc: &document{ids: map[string]*node{}}, ent: entities{budget: maxEntityBytes}}
	for {
		tok, err := d.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			if r.doc.root != nil && len(r.stack) == 0 {
				// Rubbish after the root is closed takes nothing from the picture.
				break
			}
			return nil, &FormatError{err}
		}
		if err = r.token(tok); err != nil {
			return nil, err
		}
	}
	if r.doc.root == nil || r.doc.root.name != elemSVG {
		return nil, &FormatError{errors.New("no svg root element")}
	}
	return r.doc, nil
}
