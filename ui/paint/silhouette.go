package paint

import (
	"image"
	"image/draw"

	"github.com/lantervale/ember/twod"
)

// shapes keeps each texture's silhouette while the texture lives.
//
//nolint:gochecknoglobals // a cache every paint shares, under its mutex
var shapes derived[shape]

type shape struct {
	tex *twod.Texture
	// of and version are the picture the silhouette was made from.
	of      image.Image
	version uint64
}

// silhouette is tex in white with its own alpha, so a tint paints its
// shape in one colour: what a mask-image keeps of its picture. It is made
// once, and again when tex has a new picture. A render texture has no
// picture to read and stays as it is.
func silhouette(tex *twod.Texture) *twod.Texture {
	if tex.Image == nil {
		return tex
	}
	shapes.mu.Lock()
	defer shapes.mu.Unlock()
	if s, ok := shapes.get(tex); ok && s.of == tex.Image && s.version == tex.Version() {
		return s.tex
	}
	out := image.NewNRGBA(tex.Image.Bounds())
	draw.Draw(out, out.Bounds(), tex.Image, tex.Image.Bounds().Min, draw.Src)
	for i := 0; i < len(out.Pix); i += 4 {
		out.Pix[i], out.Pix[i+1], out.Pix[i+2] = 0xff, 0xff, 0xff
	}
	made := twod.NewTexture(out)
	made.Repeat, made.Filter = tex.Repeat, tex.Filter
	shapes.put(tex, shape{tex: made, of: tex.Image, version: tex.Version()})
	return made
}
