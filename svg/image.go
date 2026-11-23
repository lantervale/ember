package svg

import (
	"bytes"
	"encoding/base64"
	"image"
	imagedraw "image/draw"
	_ "image/gif"  // images of data: URLs
	_ "image/jpeg" // images of data: URLs
	_ "image/png"  // images of data: URLs
	"strings"

	_ "golang.org/x/image/webp" // images of data: URLs
)

const (
	// maxImagePixels is the most pixels an image element's picture may
	// have.
	maxImagePixels = 1 << 24
	// maxNesting is how deep SVG pictures may be images of one another.
	maxNesting = 4
)

// dataURL is the bytes a data: URL holds; ok is false for any other URL.
func dataURL(s string) ([]byte, bool) {
	s = strings.TrimSpace(s)
	if len(s) < 5 || !strings.EqualFold(s[:5], "data:") {
		return nil, false
	}
	meta, payload, found := strings.Cut(s[5:], ",")
	if !found {
		return nil, false
	}
	if strings.HasSuffix(strings.ToLower(strings.TrimSpace(meta)), ";base64") {
		payload = strings.Map(func(r rune) rune {
			if r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == '\f' {
				return -1
			}
			return r
		}, unescape(payload))
		data, err := base64.StdEncoding.DecodeString(payload)
		if err != nil {
			data, err = base64.RawStdEncoding.DecodeString(strings.TrimRight(payload, "="))
		}
		return data, err == nil
	}
	return []byte(unescape(payload)), true
}

// unescape decodes a URL's %XX escapes, leaving any other % as it is.
func unescape(s string) string {
	if !strings.Contains(s, "%") {
		return s
	}
	hex := func(c byte) (byte, bool) {
		switch {
		case c >= '0' && c <= '9':
			return c - '0', true
		case c >= 'a' && c <= 'f':
			return c - 'a' + 10, true
		case c >= 'A' && c <= 'F':
			return c - 'A' + 10, true
		}
		return 0, false
	}
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == '%' && i+2 < len(s) {
			hi, ok1 := hex(s[i+1])
			lo, ok2 := hex(s[i+2])
			if ok1 && ok2 {
				out = append(out, hi<<4|lo)
				i += 2
				continue
			}
		}
		out = append(out, s[i])
	}
	return string(out)
}

// isSVG reports bytes that start, past a prolog, with markup rather than
// a raster format's signature.
func isSVG(data []byte) bool {
	data = bytes.TrimLeft(bytes.TrimPrefix(data, []byte("\xef\xbb\xbf")), " \t\r\n")
	return bytes.HasPrefix(data, []byte("<"))
}

// raster decodes a PNG, JPEG, GIF or WebP picture, alpha premultiplied;
// nil for anything else, or one over maxImagePixels.
func raster(data []byte) *image.RGBA {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width*cfg.Height > maxImagePixels {
		return nil
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil
	}
	b := img.Bounds()
	out := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	imagedraw.Draw(out, out.Rect, img, b.Min, imagedraw.Src)
	return out
}

// decode reads the picture an image element holds: pixels or an SVG
// picture, and the box of it that is fitted into the element; an empty
// box for none, as a broken image draws nothing in a browser.
func (b *builder) decode(data []byte) (*image.RGBA, *Picture, Box) {
	if !isSVG(data) {
		pix := raster(data)
		if pix == nil {
			return nil, nil, Box{}
		}
		return pix, nil, Box{0, 0, float64(pix.Rect.Dx()), float64(pix.Rect.Dy())}
	}
	if b.nesting >= maxNesting {
		return nil, nil, Box{}
	}
	pic, err := newPicture(data, b.nesting+1)
	if err != nil {
		return nil, nil, Box{}
	}
	for _, what := range pic.unsupported {
		b.unsupported(what)
	}
	if !pic.ViewBox.Empty() {
		return nil, pic, pic.ViewBox
	}
	w, h := pic.Size()
	return nil, pic, Box{0, 0, w, h}
}

// image appends what an image element draws: a picture of a data: URL,
// fitted into its rectangle as its preserveAspectRatio asks. Nothing is
// fetched: any other URL is reported and draws nothing.
func (b *builder) image(props map[string]string, st *style, m Matrix, out *[]Item) error {
	if st.hidden {
		return nil
	}
	href := strings.TrimSpace(props["href"])
	if href == "" {
		return nil
	}
	data, ok := dataURL(href)
	if !ok {
		b.unsupported("external reference")
		return nil
	}
	pix, pic, natural := b.decode(data)
	if natural.Empty() {
		return nil
	}
	size := func(k string, whole, natural float64) float64 {
		v := strings.TrimSpace(props[k])
		if s, valid := length(v, lengthContext{whole: whole, em: st.fontSize}); valid && v != valAuto {
			return s
		}
		return natural
	}
	x := size("x", b.vw, 0)
	y := size("y", b.vh, 0)
	w, h := size("width", b.vw, -1), size("height", b.vh, -1)
	switch {
	case w < 0 && h < 0:
		w, h = natural.W, natural.H
	case w < 0:
		w = h * natural.W / natural.H
	case h < 0:
		h = w * natural.H / natural.W
	}
	if !(w > 0) || !(h > 0) || natural.Empty() {
		return nil
	}
	aspect := parseAspect(props["preserveAspectRatio"])
	place := m.Mul(translate(x, y)).Mul(fit(natural, aspect, w, h))
	if pic != nil {
		items := relocate(pic.Items, place)
		if len(items) == 0 {
			return nil
		}
		for _, s := range shapesOf(items, nil) {
			if err := b.count(len(s.Path.Verbs)); err != nil {
				return err
			}
		}
		*out = append(*out, Item{Group: &Group{Opacity: 1, Items: items, Clip: rectClip(Box{x, y, w, h}, m)}})
		return nil
	}
	item := Item{Image: &Image{Pix: pix, Transform: place, Opacity: 1, Pixelated: st.pixelated}}
	if aspect.Slice && !aspect.Stretch {
		item = Item{Group: &Group{Opacity: 1, Items: []Item{item}, Clip: rectClip(Box{x, y, w, h}, m)}}
	}
	*out = append(*out, item)
	return nil
}
