package paint

import (
	"encoding/binary"
	"math"
	"sync"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/outline"
	"github.com/lantervale/ember/ui/internal/style"
)

// clipShape is the outline a clip-path cuts an element and all it holds
// to (decision 0119), in the space its border box is placed in.
type clipShape struct {
	path    outline.Path
	evenOdd bool
	// bounds holds the outline, empty when it holds nothing.
	bounds geom.Rect
	// boxed is set for a rectangle, inset()'s or a box's, whose right and
	// bottom edges are out as a box's are.
	boxed bool
	// parts, when set, are outlines the mask covers where each of them
	// does, in place of path: a border's band cut to its side.
	parts []maskPart
}

// holds reports whether q is inside the outline, as Chrome hits it: on a
// rectangle's left and top edges, not its right and bottom ones, and on
// any other outline.
func (s *clipShape) holds(q geom.Vec2) bool {
	b := s.bounds
	const e = 1e-3
	in := q.X >= b.X-e && q.X <= b.X+b.W+e && q.Y >= b.Y-e && q.Y <= b.Y+b.H+e
	if s.boxed {
		in = hits(b, q)
	}
	return in && s.path.Contains(q, s.evenOdd)
}

// shaped is a Surface that takes a clip-path's outline itself rather
// than its mask: a hit test's, which tests points against it, and the
// cull check's, which only notes it.
type shaped interface {
	pushShape(s *clipShape)
}

// clipShapeOf is the outline st's clip-path cuts b to, its border box's
// top-left at at; ok is false for none.
func clipShapeOf(b *box.Box, at geom.Vec2, st *style.Computed) (clipShape, bool) {
	c := &st.ClipPath
	if b.Kind != box.Element || c.None() {
		return clipShape{}, false
	}
	ref, round := referenceBox(b, at, st, c.Box)
	s := clipShape{evenOdd: c.EvenOdd, boxed: c.Shape == style.ShapeNone || c.Shape == style.ShapeInset}
	w, h := ref.W, ref.H
	switch c.Shape {
	case style.ShapeNone:
		var r [4]geom.Vec2
		for i, k := range round {
			r[i] = geom.V2(k, k)
		}
		s.path = outline.RoundedRect(ref, r)
	case style.ShapeInset:
		s.path = insetPath(c, ref)
	case style.ShapeCircle, style.ShapeEllipse:
		centre := geom.V2(ref.X+c.At[0].At(w), ref.Y+c.At[1].At(h))
		rx, ry := radiiAt(c, ref, centre)
		s.path = outline.Ellipse(centre, rx, ry)
	case style.ShapePolygon:
		pts := make([]geom.Vec2, len(c.Points))
		for i, p := range c.Points {
			pts[i] = geom.V2(ref.X+p[0].At(w), ref.Y+p[1].At(h))
		}
		s.path = outline.Polygon(pts)
	case style.ShapePath:
		p, _, _ := outline.Parse(c.Path)
		s.path = p.Translate(geom.V2(ref.X, ref.Y))
	}
	s.bounds, _ = s.path.Bounds()
	return s, true
}

// radiiAt are a circle()'s or an ellipse()'s radii in ref, centred on c.
func radiiAt(c *style.ClipPath, ref geom.Rect, centre geom.Vec2) (float32, float32) {
	near := [2]float32{min(abs(centre.X-ref.X), abs(ref.X+ref.W-centre.X)),
		min(abs(centre.Y-ref.Y), abs(ref.Y+ref.H-centre.Y))}
	far := [2]float32{max(abs(centre.X-ref.X), abs(ref.X+ref.W-centre.X)),
		max(abs(centre.Y-ref.Y), abs(ref.Y+ref.H-centre.Y))}
	if c.Shape == style.ShapeCircle {
		switch r := c.Radius[0]; r.Side {
		case style.RadiusClosest:
			k := min(near[0], near[1])
			return k, k
		case style.RadiusFarthest:
			k := max(far[0], far[1])
			return k, k
		case style.RadiusLength:
			k := r.Length.At(float32(math.Hypot(float64(ref.W), float64(ref.H)) / math.Sqrt2))
			return k, k
		}
	}
	var rs [2]float32
	for i := range rs {
		switch r := c.Radius[i]; r.Side {
		case style.RadiusClosest:
			rs[i] = near[i]
		case style.RadiusFarthest:
			rs[i] = far[i]
		case style.RadiusLength:
			rs[i] = r.Length.At([2]float32{ref.W, ref.H}[i])
		}
	}
	return rs[0], rs[1]
}

// referenceBox is the box a clip-path's shape is placed in, and that
// box's corners' radii, its border's grown or shrunk to it.
func referenceBox(b *box.Box, at geom.Vec2, st *style.Computed, which style.ShapeBox) (geom.Rect, radii) {
	r := geom.Rect{X: at.X, Y: at.Y, W: b.Size.X, H: b.Size.Y}
	round := radiiOf(st, b.Size)
	in := func(e box.Edges) {
		r = geom.Rect{X: r.X + e.Left, Y: r.Y + e.Top, W: max(r.W-e.Horizontal(), 0), H: max(r.H-e.Vertical(), 0)}
		round = round.inside(e)
	}
	switch which {
	case style.ShapePaddingBox:
		in(b.Border)
	case style.ShapeContentBox, style.ShapeFillBox:
		in(b.Border)
		in(b.Padding)
	case style.ShapeMarginBox:
		m := b.Margin
		r = geom.Rect{X: r.X - m.Left, Y: r.Y - m.Top, W: r.W + m.Horizontal(), H: r.H + m.Vertical()}
		grow := [4]float32{max(m.Left, m.Top), max(m.Right, m.Top), max(m.Right, m.Bottom), max(m.Left, m.Bottom)}
		for i := range round {
			if round[i] > 0 {
				round[i] += grow[i]
			}
		}
	case style.ShapeBoxUnset, style.ShapeBorderBox, style.ShapeStrokeBox, style.ShapeViewBox:
	}
	return r, round
}

// insetPath is inset()'s rectangle in ref, its sides shortened together
// where they meet, as CSS has them, and its corners rounded.
func insetPath(c *style.ClipPath, ref geom.Rect) outline.Path {
	t, rt, bt, l := c.Inset.Top.At(ref.H), c.Inset.Right.At(ref.W), c.Inset.Bottom.At(ref.H), c.Inset.Left.At(ref.W)
	if l+rt > ref.W && l+rt > 0 {
		k := ref.W / (l + rt)
		l, rt = l*k, rt*k
	}
	if t+bt > ref.H && t+bt > 0 {
		k := ref.H / (t + bt)
		t, bt = t*k, bt*k
	}
	r := geom.Rect{X: ref.X + l, Y: ref.Y + t, W: ref.W - l - rt, H: ref.H - t - bt}
	corner := func(s style.Size[style.Length]) geom.Vec2 { return geom.V2(s.Width.At(ref.W), s.Height.At(ref.H)) }
	rr := c.Round
	return outline.RoundedRect(r,
		[4]geom.Vec2{corner(rr.TopLeft), corner(rr.TopRight), corner(rr.BottomRight), corner(rr.BottomLeft)})
}

// maxMaskSide is the longest side, in pixels, a clip-path's mask is
// drawn at; a larger shape's mask is drawn coarser and stretched.
const maxMaskSide = 2048

// masks keeps the masks clip-paths were drawn with, by outline and size,
// for the frames that draw them again.
//
//nolint:gochecknoglobals // a cache every paint shares, under its mutex
var masks maskCache

// maskCache holds up to keepMasks masks, the least recently used going
// first.
type maskCache struct {
	mu   sync.Mutex
	of   map[string]*cachedMask
	tick uint64
	key  []byte
}

type cachedMask struct {
	tex  *twod.Texture
	used uint64
}

const keepMasks = 256

// mask is s's mask at k pixels a point: white, opaque inside the outline,
// covering s.bounds.
func (m *maskCache) mask(s *clipShape, k float32) *twod.Texture {
	w, h := max(int(math.Ceil(float64(s.bounds.W*k))), 1), max(int(math.Ceil(float64(s.bounds.H*k))), 1)
	if side := max(w, h); side > maxMaskSide {
		w, h = max(w*maxMaskSide/side, 1), max(h*maxMaskSide/side, 1)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	key := m.keyOf(s, w, h)
	m.tick++
	if c, ok := m.of[string(key)]; ok {
		c.used = m.tick
		return c.tex
	}
	var tex *twod.Texture
	if len(s.parts) > 0 {
		tex = twod.NewTexture(partsMask(s.parts, s.bounds, w, h))
	} else {
		tex = twod.NewTexture(s.path.Mask(s.bounds, w, h, s.evenOdd))
	}
	if m.of == nil {
		m.of = map[string]*cachedMask{}
	}
	if len(m.of) >= keepMasks {
		m.forgetOld()
	}
	m.of[string(key)] = &cachedMask{tex: tex, used: m.tick}
	return tex
}

// forgetOld lets the older half of the masks go.
func (m *maskCache) forgetOld() {
	var cut uint64
	for _, c := range m.of {
		cut += c.used
	}
	cut /= uint64(len(m.of))
	for k, c := range m.of {
		if c.used <= cut {
			delete(m.of, k)
		}
	}
}

// keyOf is what tells s's mask of w×h pixels from any other, in m.key.
func (m *maskCache) keyOf(s *clipShape, w, h int) []byte {
	b := m.key[:0]
	b = binary.LittleEndian.AppendUint32(b, uint32(w)) //nolint:gosec // a mask's side, at most maxMaskSide
	b = binary.LittleEndian.AppendUint32(b, uint32(h)) //nolint:gosec // as w
	if s.evenOdd {
		b = append(b, 1)
	}
	f := func(v float32) { b = binary.LittleEndian.AppendUint32(b, math.Float32bits(v)) }
	path := func(p outline.Path) {
		for _, c := range p.Contours {
			b = binary.LittleEndian.AppendUint32(b, uint32(len(c))) //nolint:gosec // a contour's length
			for _, q := range c {
				f(q.X - s.bounds.X)
				f(q.Y - s.bounds.Y)
			}
		}
	}
	path(s.path)
	for _, m := range s.parts {
		b = append(b, 2)
		if m.evenOdd {
			b = append(b, 1)
		}
		path(m.path)
	}
	f(s.bounds.W)
	f(s.bounds.H)
	m.key = b
	return b
}

// maskScale is how many device pixels a point of what is drawn now
// takes: the paint's Scale, by the surface's transform.
// slant is how far past an edge, in the space drawn in, a point on the
// screen can lie when the transform turns or skews it: one over its
// smallest stretch, or less. It is 0 drawn square.
func (p *painter) slant() float32 {
	t := p.s.Transform()
	det := abs(t.X.X*t.Y.Y - t.X.Y*t.Y.X)
	if t.X.Y == 0 && t.Y.X == 0 || det == 0 {
		return 0
	}
	return float32(math.Sqrt(float64(t.X.Dot(t.X)+t.Y.Dot(t.Y)))) / det
}

func (p *painter) maskScale() float32 {
	k := p.scale
	if k <= 0 {
		k = 1
	}
	s := p.s.Transform().Scale()
	return k * max(abs(s.X), abs(s.Y))
}
