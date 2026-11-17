// Package web gets a Document's pictures over HTTP: a fetcher for
// ui.FetchImages, so a url() background or an img's src can name a
// picture on the web. Importing it lets ui read JPEG and GIF pictures
// beside PNG; an SVG is asked for too and handed over as it is, for a
// Document to draw once ui/svg is imported.
package web

import (
	"context"
	"fmt"
	_ "image/gif" // the pictures of the web, beside ui's PNG
	_ "image/jpeg"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/lantervale/ember/ui"
)

const (
	// MaxBytes is the largest picture Images reads; a larger one is an
	// error.
	MaxBytes = 8 << 20
	// Timeout is how long one picture may take.
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
	from, err := url.Parse(base)
	if err == nil && from.Scheme != "http" && from.Scheme != "https" {
		err = fmt.Errorf("web: %q is no http or https address", base)
	}
	client := &http.Client{Timeout: Timeout}
	return func(ref string) ([]byte, error) {
		if err != nil {
			return nil, err
		}
		at, err := from.Parse(ref)
		if err != nil {
			return nil, err
		}
		if at.Scheme != "http" && at.Scheme != "https" {
			return nil, fmt.Errorf("web: %q is no http or https address", ref)
		}
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, at.String(), nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Accept", "image/png, image/jpeg, image/gif, image/svg+xml")
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
			return nil, fmt.Errorf("web: %s", resp.Status)
		}
		data, err := io.ReadAll(io.LimitReader(resp.Body, MaxBytes+1))
		if err != nil {
			return nil, err
		}
		if len(data) > MaxBytes {
			return nil, fmt.Errorf("web: the picture is over %d bytes", MaxBytes)
		}
		return data, nil
	}
}
