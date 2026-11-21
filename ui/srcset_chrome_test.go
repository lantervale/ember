//go:build chrome

package ui_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
	"github.com/lantervale/ember/ui/internal/fonts"
)

// candidates are the pictures an img offers, each its own, so that no
// picture another img loaded is in Chrome's cache when it chooses:
// files both Chrome and the Document's fetcher read by their file URL.
// names are the pictures by their address.
type candidates struct {
	t     *testing.T
	dir   string
	names map[string]string
}

func newCandidates(t *testing.T) *candidates {
	return &candidates{t: t, dir: t.TempDir(), names: map[string]string{}}
}

// png is the address of a picture of the img, w pixels wide and half as
// tall.
func (c *candidates) png(img string, w int) string {
	pic := image.NewNRGBA(image.Rect(0, 0, w, w/2))
	ink := []uint8{uint8(len(c.names) * 37), uint8(w), 90, 255}
	for i := range pic.Pix {
		pic.Pix[i] = ink[i%4]
	}
	var b bytes.Buffer
	require.NoError(c.t, png.Encode(&b, pic))
	return c.file(fmt.Sprintf("%s-%d.png", img, w), b.Bytes())
}

func (c *candidates) file(name string, data []byte) string {
	path := filepath.Join(c.dir, name)
	require.NoError(c.t, os.WriteFile(path, data, 0o600))
	url := "file://" + filepath.ToSlash(path)
	c.names[url] = name
	return url
}

// inline is a picture of the img as a data URL, which Chrome takes as
// held.
func (c *candidates) inline(img string, w int) string {
	url := c.png(img, w)
	data, err := os.ReadFile(strings.TrimPrefix(url, "file://"))
	require.NoError(c.t, err)
	inline := "data:image/png;base64," + base64.StdEncoding.EncodeToString(data)
	c.names[inline] = c.names[url]
	return inline
}

func (c *candidates) fetch(url string) ([]byte, error) {
	return os.ReadFile(strings.TrimPrefix(url, "file://"))
}

func TestAnImgChoosesThePictureChromeChoosesAndShowsItAtItsDensity(t *testing.T) {
	for _, scale := range []float32{1, 2} {
		t.Run(fmt.Sprintf("%gx", scale), func(t *testing.T) {
			c := newCandidates(t)
			img := func(id string, attrs ...string) *ui.Node {
				args := []ui.Arg{ui.ID(id)}
				for i := 0; i < len(attrs); i += 2 {
					args = append(args, ui.Attr(attrs[i], attrs[i+1]))
				}
				return ui.Img(args...)
			}
			set := func(id string, entries ...string) string {
				var s []string
				for _, e := range entries {
					w, descriptors, _ := strings.Cut(e, " ")
					width, err := strconv.Atoi(w)
					require.NoError(t, err)
					s = append(s, strings.TrimSpace(c.png(id, width)+" "+descriptors))
				}
				return strings.Join(s, ", ")
			}
			d := ui.NewDocument(ui.FetchImages(c.fetch))
			d.Body().Append(
				img("x", "srcset", set("x", "100 1x", "200 2x", "400 4x")),
				img("x3", "srcset", set("x3", "300 3x")),
				img("x15", "srcset", set("x15", "102 1.5x")),
				img("w", "srcset", set("w", "100 100w", "200 200w", "400 400w"), "sizes", "100px"),
				img("w2", "srcset", set("w2", "100 100w", "300 300w"), "sizes", "150px"),
				img("wmedia", "srcset", set("wmedia", "100 100w", "300 300w"),
					"sizes", "(min-width: 2000px) 10px, calc(10vw - 1px)"),
				img("srcx", "src", c.png("srcx", 100), "srcset", set("srcx", "200 2x")),
				img("src1", "src", c.png("src1", 100), "srcset", set("src1", "200 1x")),
				img("srcw", "src", c.png("srcw", 300), "srcset", set("srcw", "100 50w"), "sizes", "100px"),
				img("dup", "srcset", set("dup", "100 2x", "200 2x")),
				img("nodesc", "srcset", set("nodesc", "200", "100 2x")),
				img("bad", "srcset", set("bad", "100 1x 2x", "200 2x", "300 50h")),
				img("pct", "srcset", set("pct", "100 100w", "200 200w"), "sizes", "50%, (orientation: portrait) 9px, 50px"),
				img("apart", "srcset", set("apart", "100 1x", "300 3x")),
				img("between", "srcset", set("between", "100 1.2x", "300 2.5x")),
				img("over", "srcset", set("over", "150 1.5x", "400 4x")),
				img("held", "srcset", c.inline("held", 100)+" 1x, "+c.inline("held", 200)+" 2x, "+c.inline("held", 300)+" 3x"),
				ui.Picture(
					ui.Source(ui.Attr("srcset", c.png("pic", 400)), ui.Attr("media", "(max-width: 10px)")),
					ui.Source(ui.Attr("srcset", c.png("pic", 300)), ui.Attr("type", "image/x-foo")),
					ui.Source(ui.Attr("srcset", "")),
					ui.Source(ui.Attr("srcset", set("pic", "200 2x")+", "+c.png("pic", 102)+" 1x"),
						ui.Attr("type", "IMAGE/PNG; q=1"), ui.Attr("media", "screen and (min-width: 100px)")),
					img("pic", "src", c.png("pic", 100)),
					ui.Source(ui.Attr("srcset", c.png("pic", 304))),
				),
				ui.Picture(
					ui.Source(ui.Attr("srcset", set("sized", "200 200w")), ui.Attr("sizes", "50px")),
					img("sized", "src", c.png("sized", 100), "sizes", "100px"),
				),
				img("auto", "loading", "lazy", "srcset", set("auto", "100 100w", "200 200w", "400 400w"),
					"sizes", "auto, 400px", "style", "width: 150px"),
			)
			size := geom.V2(800, 600)
			fonts.Of(d).SetScale(scale)
			ui.LandImages(d, size)

			chrome.MatchScaled(t, d, size, scale)
			out, err := chrome.EvalScaled(d, size, scale, `const names = `+mustJSON(t, c.names)+`;
				return Object.fromEntries([...document.querySelectorAll("img[id]")].map(e =>
					[e.id, names[e.currentSrc] ?? e.currentSrc]));`)
			require.NoError(t, err)
			var want map[string]string
			require.NoError(t, json.Unmarshal(out, &want))
			delete(want, "auto")
			require.Len(t, want, 19)
			d.HTML().Walk(func(n *ui.Node) bool {
				if w, ok := want[n.ID()]; ok {
					got, _ := ui.PictureOf(n)
					assert.Equal(t, w, c.names[got], "#%s", n.ID())
				}
				return true
			})
		})
	}
}

func mustJSON(t *testing.T, v any) string {
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return string(b)
}
