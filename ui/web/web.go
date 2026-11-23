// Package web gets a Document's pictures, fonts and imported sheets
// over HTTP: a fetcher for ui.FetchImages, so a url() background or an
// img's src can name a picture on the web, one for ui.FetchFonts, so an
// @font-face can name a font there, and one for ui.FetchSheets, so an
// @import can name a sheet there. Importing it lets ui read JPEG, GIF, WebP, ICO,
// CUR and BMP pictures beside PNG (an animated GIF or WebP plays, an ICO
// shows its largest picture, as Chrome's do); an SVG is asked for too and handed
// over as it is, for a Document to draw once ui/svg is imported.
package web

import (
	"context"
	"fmt"
	"image"
	_ "image/gif" // the pictures of the web, beside ui's PNG
	_ "image/jpeg"
	"io"
	"mime"
	"net/http"
	"net/url"
	"time"

	_ "golang.org/x/image/webp" // and WebP

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/animated"
	"github.com/lantervale/ember/ui/internal/ico"
)

//nolint:gochecknoinits // registers the formats on import, as the image packages above do
func init() {
	image.RegisterFormat("ico", "\x00\x00\x01\x00", ico.Decode, ico.DecodeConfig)
	image.RegisterFormat("cur", "\x00\x00\x02\x00", ico.Decode, ico.DecodeConfig)
	animated.Register("GIF8", decodeGIF)
	animated.Register(animated.WebPMagic, decodeWebP)
	image.RegisterFormat("bmp", "BM????\x00\x00\x00\x00", ico.DecodeBMP, ico.DecodeBMPConfig)
}

const (
	// MaxBytes is the largest picture Images reads; a larger one is an
	// error.
	MaxBytes = 8 << 20
	// Timeout is how long one picture, or one font, may take.
	Timeout = 15 * time.Second
)

// Images is a fetcher that gets pictures with an HTTP GET: an http or
// https address as it is, anything else as a reference from base, as a
// browser resolves an img's src from its page's address ("door.png",
// "../art/door.png", "/art/door.png"). A url() in a stylesheet is
// resolved from base too, unless the sheet says where it came from
// (ui.From): it then arrives here in full. An answer that isn't
// a success, or a picture over MaxBytes, is an error, and so is every
// fetch when base is no http or https address.
func Images(base string) ui.ImageFetcher {
	return newFetcher(base).get
}

// fetcher gets the pictures named from base; err is why it gets none.
type fetcher struct {
	base   *url.URL
	err    error
	client *http.Client
}

// MaxFontBytes is the largest font file Fonts reads; a larger one is
// an error.
const MaxFontBytes = 16 << 20

// Fonts is a fetcher for ui.FetchFonts that gets the fonts of a page's
// @font-face rules with an HTTP GET, as Images gets its pictures: an
// http or https address as it is, anything else as a reference from
// base. The src of a sheet that says where it came from (ui.From)
// arrives here in full. An answer that isn't a success, or a font over
// MaxFontBytes, is an error, and so is every fetch when base is no
// http or https address.
func Fonts(base string) ui.FontFetcher {
	f := newFetcher(base)
	return func(ref, _ string) ([]byte, error) {
		return f.fetch(ref, "", "font/woff2, font/woff, font/ttf, font/otf, */*;q=0.1", MaxFontBytes, "font")
	}
}

// MaxSheetBytes is the largest sheet Sheets reads; a larger one is an
// error.
const MaxSheetBytes = 8 << 20

// Sheets is a fetcher for ui.FetchSheets that gets the sheets of a
// page's @import rules with an HTTP GET, as Images gets its pictures:
// an http or https address as it is, anything else as a reference from
// base, the importing sheet's address as the referrer. The import of a
// sheet that says where it came from (ui.From) arrives here in full. As
// Chrome, it takes a sheet served as text/css or as no type: another
// type is an error, as are an answer that isn't a success, a sheet over
// MaxSheetBytes, and every fetch when base is no http or https address.
func Sheets(base string) ui.SheetFetcher {
	f := newFetcher(base)
	return func(ref, sheet string) ([]byte, error) {
		return f.fetch(ref, sheet, "text/css,*/*;q=0.1", MaxSheetBytes, "sheet")
	}
}

func newFetcher(base string) *fetcher {
	f := &fetcher{client: &http.Client{Timeout: Timeout}}
	f.base, f.err = url.Parse(base)
	if f.err == nil && !overHTTP(f.base) {
		f.err = fmt.Errorf("web: %q is no http or https address", base)
	}
	return f
}

func overHTTP(at *url.URL) bool { return at.Scheme == "http" || at.Scheme == "https" }

// pictures names the pictures a Document reads once this package is
// imported, and SVG, read once ui/svg is.
const pictures = "image/png, image/jpeg, image/gif, image/webp, image/svg+xml, " +
	"image/x-icon, image/vnd.microsoft.icon, image/bmp"

func (f *fetcher) get(ref string) ([]byte, error) {
	return f.fetch(ref, "", pictures, MaxBytes, "picture")
}

// fetch gets what ref names from base, asking for accept with referrer
// as the Referer when it says one: the body of a successful answer, no
// larger than limit, and for a sheet one served as text/css or as no
// type.
func (f *fetcher) fetch(ref, referrer, accept string, limit int64, what string) ([]byte, error) {
	if f.err != nil {
		return nil, f.err
	}
	at, err := f.base.Parse(ref)
	if err != nil {
		return nil, err
	}
	if !overHTTP(at) {
		return nil, fmt.Errorf("web: %q is no http or https address", ref)
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, at.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", accept)
	if referrer != "" {
		req.Header.Set("Referer", referrer)
	}
	resp, err := f.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("web: %s", resp.Status)
	}
	if what == "sheet" {
		if kind, _, _ := mime.ParseMediaType(resp.Header.Get("Content-Type")); kind != "text/css" && kind != "" {
			return nil, fmt.Errorf("web: the sheet is %s, not text/css", kind)
		}
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("web: the %s is over %d bytes", what, limit)
	}
	return data, nil
}
