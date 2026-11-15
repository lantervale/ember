package paint

import (
	"sync"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/render"
	"github.com/lantervale/ember/twod"
)

// sharp keeps each texture's pixelated copy, up to maxSharp, then starts
// over.
//
//nolint:gochecknoglobals // a cache every paint shares, under its mutex
var (
	sharp   = map[*twod.Texture]*twod.Texture{}
	sharpMu sync.Mutex
)

const maxSharp = 256

// pixelated is tex sampled by its nearest pixels: a copy sharing its
// picture, made once, given tex's new picture and touched whenever tex
// has been since, so the GPU uploads it again. A render texture has no picture to share and
// stays as it is.
func pixelated(tex *twod.Texture) *twod.Texture {
	if tex.Image == nil {
		return tex
	}
	sharpMu.Lock()
	defer sharpMu.Unlock()
	c, ok := sharp[tex]
	if !ok {
		if len(sharp) >= maxSharp {
			clear(sharp)
		}
		opts := []twod.TextureOption{twod.Pixelated()}
		if tex.Encoded() {
			opts = append(opts, twod.Encoded())
		}
		c = twod.NewTexture(tex.Image, opts...)
		c.Repeat = tex.Repeat
		sharp[tex] = c
	}
	if c.Image != tex.Image {
		c.Image = tex.Image
		c.Touch()
	}
	for c.Version() < tex.Version() {
		c.Touch()
	}
	return c
}

// tieBias is how far up and left of a pixel's centre a pixelated picture
// is sampled, in texels: far less than a texel, so only a pixel whose
// centre falls on the line between two texels changes texel.
const tieBias = 1.0 / 1024

// texels is the whole of tex for a draw: when tex is sampled by its
// nearest pixels, nudged up and left by tieBias, so a pixel whose centre
// falls on the line between two texels (a texel a whole and a half
// pixels wide, as a 16-texel picture over 72) takes the upper left one,
// as Chrome's does; the GPU alone gives it the lower right one.
func texels(tex *twod.Texture) geom.Rect {
	size := tex.Size()
	if tex.Filter != render.FilterNearest || size.X <= 0 || size.Y <= 0 {
		return whole
	}
	return geom.Rect{X: -tieBias / size.X, Y: -tieBias / size.Y, W: 1, H: 1}
}
