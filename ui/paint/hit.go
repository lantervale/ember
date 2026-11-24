package paint

import (
	"fmt"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/cull"
)

// HitTest is the element box at p, in the root's space, that paints on
// top there: the last one painted whose border box holds p, within its
// clips and through its transforms. Hidden boxes and text runs never are,
// but transparent ones are, as in CSS.
// Controls, if given, finds modal dialogs, whose backdrop takes the page's
// pointer.
func HitTest(t *box.Tree, p geom.Vec2, opts ...Option) *box.Box {
	mode := cull.Now()
	hit, _ := hitTest(t, p, opts, mode == cull.Off)
	if mode == cull.Check {
		if whole, _ := hitTest(t, p, opts, true); whole != hit {
			cull.Differ(fmt.Sprintf("the box hit at %v: %s, %s walking every box", p, named(hit), named(whole)))
		}
	}
	return hit
}

// hitTest is HitTest walking every box when whole, else only those whose
// ink holds p, and how many boxes under the root the walk reached.
func hitTest(t *box.Tree, p geom.Vec2, opts []Option, whole bool) (*box.Box, int) {
	pt := scan(t, opts, p, true, whole)
	if pt == nil {
		return nil, 0
	}
	defer scanners.Put(pt)
	hit := pt.found
	for _, pp := range pt.popups {
		if pt.place(pp).rect.Contains(p) {
			hit = pp.b
		}
	}
	return hit, pt.reached
}

// probed is told, on a hit test's walk, of each element box painted at r
// in the current space: the last holding point is the hit, but for one
// with pointer-events: none.
func (p *painter) probed(b *box.Box, r geom.Rect) {
	if st := shown(b); !p.pointless && (st == nil || !st.PointerNone) && p.probe.holds(p.point, r) {
		p.found = b
	}
}

// probe is a Surface that draws nothing, keeping only the transform and
// the clips, for hit testing.
type probe struct {
	xf    geom.Transform2D
	clips []probeClip
}

type probeClip struct {
	rect  geom.Rect
	radii [4]float32
	back  geom.Transform2D
	shape *clipShape
}

// holds reports whether q, in the clip's space, is inside it: its rect,
// and with radii, its rounded corners, or with a shape, that shape.
func (c probeClip) holds(q geom.Vec2) bool {
	if c.shape != nil {
		return c.shape.holds(q)
	}
	if !hits(c.rect, q) {
		return false
	}
	r := c.rect
	for i, corner := range cornersOf(r) {
		k := c.radii[i]
		if k <= 0 {
			continue
		}
		sx, sy := float32(1), float32(1)
		if i == 1 || i == 2 {
			sx = -1
		}
		if i >= 2 {
			sy = -1
		}
		centre := geom.V2(corner.X+sx*k, corner.Y+sy*k)
		d := q.Sub(centre)
		if d.X*sx < 0 && d.Y*sy < 0 && d.X*d.X+d.Y*d.Y > k*k {
			return false
		}
	}
	return true
}

// holds reports whether p, in the root's space, is in r in the current
// space and in every clip in theirs.
func (s *probe) holds(p geom.Vec2, r geom.Rect) bool {
	for _, c := range s.clips {
		if !c.holds(c.back.Xform(p)) {
			return false
		}
	}
	back := s.xf.Inverse()
	if shifted(s.xf) {
		return hits(r, back.Xform(p))
	}
	return reaches(r, back, p)
}

// shifted reports whether t only moves by whole pixels, which Blink hits
// through as it hits outside any transform: as a point.
func shifted(t geom.Transform2D) bool {
	return t.X == geom.V2(1, 0) && t.Y == geom.V2(0, 1) &&
		t.Origin.X == geom.Floor(t.Origin.X) && t.Origin.Y == geom.Floor(t.Origin.Y)
}

// reaches is whether the pixel square from p, in the root's space, taken
// back into r's space, overlaps r: how Blink hits a point through any
// other transform, so a point up to the square's reach before a turned
// box's edge hits it.
func reaches(r geom.Rect, back geom.Transform2D, p geom.Vec2) bool {
	q := [4]geom.Vec2{back.Xform(p), back.Xform(p.Add(geom.V2(1, 0))), back.Xform(p.Add(geom.V2(1, 1))),
		back.Xform(p.Add(geom.V2(0, 1)))}
	box := cornersOf(r)
	for _, axis := range [4]geom.Vec2{geom.V2(1, 0), geom.V2(0, 1), perp(q[1].Sub(q[0])), perp(q[3].Sub(q[0]))} {
		a0, a1 := spread(axis, q[:])
		b0, b1 := spread(axis, box[:])
		if a1 <= b0 || b1 <= a0 {
			return false
		}
	}
	return true
}

func perp(v geom.Vec2) geom.Vec2 { return geom.V2(-v.Y, v.X) }

// spread is the least and the most of the points' projections on axis.
func spread(axis geom.Vec2, pts []geom.Vec2) (float32, float32) {
	lo, hi := pts[0].Dot(axis), pts[0].Dot(axis)
	for _, v := range pts[1:] {
		d := v.Dot(axis)
		lo, hi = min(lo, d), max(hi, d)
	}
	return lo, hi
}

// hits is whether p is in r as Chrome hits a box: its left and top
// edges in, its right and bottom ones out, so a point on a border between
// two boxes takes only the one starting there.
func hits(r geom.Rect, p geom.Vec2) bool {
	return p.X >= r.X && p.X < r.X+r.W && p.Y >= r.Y && p.Y < r.Y+r.H
}

func (s *probe) PushClip(r geom.Rect) {
	s.clips = append(s.clips, probeClip{rect: r, back: s.xf.Inverse()})
}

// PushClipRounded is a rounded clip, turned with what it cuts, as the
// draw list draws it (decision 0216).
func (s *probe) PushClipRounded(r geom.Rect, radii [4]float32) {
	s.clips = append(s.clips, probeClip{rect: r, radii: radii, back: s.xf.Inverse()})
}

// PushClipMask is a picture's clip, hit as its rect: paint's own masks
// come through pushShape, whose outline is hit exactly.
func (s *probe) PushClipMask(r geom.Rect, _ *twod.Texture) { s.PushClip(r) }

func (s *probe) pushShape(c *clipShape) {
	s.clips = append(s.clips, probeClip{rect: c.bounds, back: s.xf.Inverse(), shape: c})
}
func (s *probe) PopClip()                                               { s.clips = s.clips[:len(s.clips)-1] }
func (s *probe) Transform() geom.Transform2D                            { return s.xf }
func (s *probe) SetTransform(t geom.Transform2D)                        { s.xf = t }
func (s *probe) Rect(geom.Rect, color.Color)                            {}
func (s *probe) RoundRect(geom.Rect, float32, color.Color)              {}
func (s *probe) Border(geom.Rect, float32, float32, color.Color)        {}
func (s *probe) Glow(geom.Rect, float32, float32, color.Color)          {}
func (s *probe) Image(geom.Rect, *twod.Texture, geom.Rect, color.Color) {}
func (s *probe) NinePatch(geom.Rect, *twod.Texture, float32, float32, float32, float32, color.Color) {
}
func (s *probe) Line(geom.Vec2, geom.Vec2, float32, color.Color)           {}
func (s *probe) Circle(geom.Vec2, float32, color.Color)                    {}
func (s *probe) RoundRectCorners(geom.Rect, [4]float32, color.Color)       {}
func (s *probe) BorderCorners(geom.Rect, [4]float32, float32, color.Color) {}
func (s *probe) TrianglesColored([]geom.Vec2, []color.Color, []int)        {}
func (s *probe) Triangles([]geom.Vec2, []int, color.Color)                 {}
func (s *probe) Text(geom.Vec2, string, ...twod.TextOption) geom.Vec2      { return geom.Vec2{} }

// cornersOf are r's corners clockwise from the top-left, as radii and
// clips number them.
func cornersOf(r geom.Rect) [4]geom.Vec2 {
	return [4]geom.Vec2{{X: r.X, Y: r.Y}, {X: r.X + r.W, Y: r.Y}, {X: r.X + r.W, Y: r.Y + r.H}, {X: r.X, Y: r.Y + r.H}}
}
