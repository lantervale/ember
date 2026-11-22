package paint

import (
	"encoding/binary"
	"fmt"
	"hash/fnv"
	"math"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/render"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/cull"
)

// checkPaint paints t twice on nothing, walking every box and skipping
// those out of view, and reports the first drawing the two don't share
// (cull.Check): each must draw the same things, in the same order, under
// the same clips and transform.
func checkPaint(s Surface, t *box.Tree, opts []Option) {
	whole, culled := &tally{xf: s.Transform()}, &tally{xf: s.Transform()}
	paintOn(whole, t, opts, true)
	paintOn(culled, t, opts, false)
	for i := range max(len(whole.drawn), len(culled.drawn)) {
		a, b := "nothing", "nothing"
		if i < len(whole.drawn) {
			a = whole.drawn[i].String()
		}
		if i < len(culled.drawn) {
			b = culled.drawn[i].String()
		}
		if i >= len(whole.drawn) || i >= len(culled.drawn) || whole.drawn[i] != culled.drawn[i] {
			cull.Differ(fmt.Sprintf("the drawing: number %d of %d is %s, %s walking every box (%d)",
				i, len(culled.drawn), b, a, len(whole.drawn)))
			return
		}
	}
}

// named is b for a report.
func named(b *box.Box) string {
	if b == nil {
		return "none"
	}
	tag := "an anonymous box"
	if b.Node != nil {
		tag = b.Node.Tag().String()
	}
	return fmt.Sprintf("%s at %v", tag, b.Rect())
}

// drawing is one thing drawn: what, where, and the sum of all it was
// drawn with, its clips and transform included.
type drawing struct {
	kind string
	at   geom.Rect
	sum  uint64
}

func (d drawing) String() string { return fmt.Sprintf("%s %v (%x)", d.kind, d.at, d.sum) }

// tally is a Surface that draws nothing and keeps what it was asked to
// draw, for checkPaint.
type tally struct {
	xf    geom.Transform2D
	clips []uint64
	drawn []drawing
}

func (t *tally) add(kind string, at geom.Rect, values ...float32) {
	h := fnv.New64a()
	var buf [8]byte
	word := func(v uint64) {
		binary.LittleEndian.PutUint64(buf[:], v)
		_, _ = h.Write(buf[:])
	}
	float := func(v float32) { word(uint64(math.Float32bits(v))) }
	_, _ = h.Write([]byte(kind))
	xf := t.xf
	for _, v := range []float32{at.X, at.Y, at.W, at.H, xf.X.X, xf.X.Y, xf.Y.X, xf.Y.Y, xf.Origin.X, xf.Origin.Y} {
		float(v)
	}
	for _, v := range values {
		float(v)
	}
	for _, c := range t.clips {
		word(c)
	}
	t.drawn = append(t.drawn, drawing{kind: kind, at: at, sum: h.Sum64()})
}

func rgba(c color.Color) []float32 { return []float32{c.R, c.G, c.B, c.A} }

func flat(points []geom.Vec2, more ...float32) []float32 {
	out := make([]float32, 0, 2*len(points)+len(more))
	for _, p := range points {
		out = append(out, p.X, p.Y)
	}
	return append(out, more...)
}

func bounds(points []geom.Vec2) geom.Rect {
	if len(points) == 0 {
		return geom.Rect{}
	}
	r := geom.Rect{X: points[0].X, Y: points[0].Y}
	for _, p := range points[1:] {
		r = union(r, geom.Rect{X: p.X, Y: p.Y})
	}
	return r
}

// pictured is kind with whether it has a texture: a picture drawn from
// shapes may be another texture each time it is asked for.
func pictured(kind string, tex *twod.Texture) string {
	if tex == nil {
		return kind + " of nothing"
	}
	return kind
}

func (t *tally) push(kind string, r geom.Rect, values ...float32) {
	mark := &tally{xf: t.xf}
	mark.add(kind, r, values...)
	t.clips = append(t.clips, mark.drawn[0].sum)
}

func (t *tally) PushClip(r geom.Rect)                          { t.push("clip", r) }
func (t *tally) PushClipRounded(r geom.Rect, radii [4]float32) { t.push("clip", r, radii[:]...) }
func (t *tally) PushClipMask(r geom.Rect, mask *twod.Texture)  { t.push(pictured("mask", mask), r) }

func (t *tally) pushShape(c *clipShape) {
	var values []float32
	for _, contour := range c.path.Contours {
		values = append(flat(contour), values...)
	}
	if c.evenOdd {
		values = append(values, 1)
	}
	t.push("shape", c.bounds, values...)
}
func (t *tally) PopClip()                         { t.clips = t.clips[:len(t.clips)-1] }
func (t *tally) Transform() geom.Transform2D      { return t.xf }
func (t *tally) SetTransform(xf geom.Transform2D) { t.xf = xf }
func (t *tally) Rect(r geom.Rect, c color.Color)  { t.add("rect", r, rgba(c)...) }
func (t *tally) RoundRect(r geom.Rect, k float32, c color.Color) {
	t.add("round", r, append(rgba(c), k)...)
}
func (t *tally) RoundRectCorners(r geom.Rect, radii [4]float32, c color.Color) {
	t.add("corners", r, append(rgba(c), radii[:]...)...)
}

func (t *tally) BorderCorners(r geom.Rect, radii [4]float32, thickness float32, c color.Color) {
	t.add("border corners", r, append(append(rgba(c), radii[:]...), thickness)...)
}

func (t *tally) Border(r geom.Rect, radius, thickness float32, c color.Color) {
	t.add("border", r, append(rgba(c), radius, thickness)...)
}

func (t *tally) Glow(r geom.Rect, radius, spread float32, c color.Color) {
	t.add("glow", r, append(rgba(c), radius, spread)...)
}

func (t *tally) Line(a, b geom.Vec2, width float32, c color.Color) {
	t.add("line", geom.Rect{X: a.X, Y: a.Y, W: b.X - a.X, H: b.Y - a.Y}, append(rgba(c), width)...)
}

func (t *tally) Circle(center geom.Vec2, radius float32, c color.Color) {
	t.add("circle", geom.Rect{X: center.X, Y: center.Y}, append(rgba(c), radius)...)
}

func (t *tally) Triangles(points []geom.Vec2, indices []int, c color.Color) {
	t.add("triangles", bounds(points), flat(points, append(rgba(c), float32(len(indices)))...)...)
}

func (t *tally) TrianglesColored(points []geom.Vec2, colors []color.Color, indices []int) {
	values := flat(points, float32(len(indices)))
	for _, c := range colors {
		values = append(values, rgba(c)...)
	}
	t.add("coloured triangles", bounds(points), values...)
}

func (t *tally) Text(at geom.Vec2, text string, _ ...twod.TextOption) geom.Vec2 {
	t.add("text "+text, geom.Rect{X: at.X, Y: at.Y})
	return geom.Vec2{}
}

func (t *tally) Image(r geom.Rect, tex *twod.Texture, uv geom.Rect, tint color.Color) {
	t.add(pictured("image", tex), r, append(rgba(tint), uv.X, uv.Y, uv.W, uv.H)...)
}

func (t *tally) NinePatch(r geom.Rect, tex *twod.Texture, left, top, right, bottom float32, tint color.Color) {
	t.add(pictured("nine-patch", tex), r, append(rgba(tint), left, top, right, bottom)...)
}

func (t *tally) BeginPicture() { t.add("picture", geom.Rect{}) }

func (t *tally) EndPicture(margin float32) *twod.Picture {
	t.add("picture's end", geom.Rect{}, margin)
	return &twod.Picture{}
}

func (t *tally) DrawPicture(_ *twod.Picture, effects ...*render.Effect) {
	values := make([]float32, 0, len(effects))
	for _, e := range effects {
		values = append(values, float32(len(e.Shader())))
	}
	t.add("picture shown", geom.Rect{}, values...)
}

func (t *tally) SumPicture(_ *twod.Picture, effects ...*render.Effect) {
	values := make([]float32, 0, len(effects))
	for _, e := range effects {
		values = append(values, float32(len(e.Shader())))
	}
	t.add("picture summed", geom.Rect{}, values...)
}

func (t *tally) PushClipPicture(_ *twod.Picture, opts ...twod.MaskOption) {
	t.push("picture's clip", geom.Rect{}, float32(len(opts)))
}

func (t *tally) Effect(shader string) *render.Effect { return render.NewEffect(shader) }
