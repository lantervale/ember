package collide

import (
	"math"

	"github.com/lantervale/ember/geom"
)

// capacity is how much of a crowd cell soft bodies cover before it
// spills: a little over the most that bodies touching without overlap
// fill, so only a crowd pressed into itself spreads by density.
const capacity = 1.25

// maxNudge is the most a spread pushes a body in one step, as a share of
// its smaller side, so a dense pile opens over a few steps instead of
// bursting.
const maxNudge = 0.25

// margin is how many cells the crowd's grid reaches past its bodies, for
// the moves until the next rebuild.
const margin = 4

// contacts is how many of the bodies near it, the nearest, a filed body
// keeps from walking into, and near how far it looks for them, as a share
// of the distance at which the ellipses two bodies' boxes hold touch.
const (
	contacts = 8
	near     = 1.25
)

// neighbours is how many of the bodies filed after it in its cell, and
// at the start of each cell past it, each body is kept apart from, so a
// crowded cell costs a bounded number of pairs.
const neighbours = 8

// maxCells is the most cells the crowd's grid holds. Its cells stay at its
// bodies' scale over a span up to about 512 of them a side; only past that
// do they grow, and bodies piled on one point then count as room.
const maxCells = 1 << 18

// crowd files the soft bodies, rebuilt once a step rather than kept up to
// date move by move: each body once, in the cell of its centre, in one
// array grouped by cell, the cells in the order their first body is
// listed. Each cell also counts how much of it soft bodies cover, kept up
// to date as they move. The grid spans every body, however far apart, but
// a step costs what its bodies touch: only the cells written since the
// last rebuild are cleared, listed in touched and marked by stamp. Bodies
// made soft since the rebuild wait in loose, and slack is how far any
// filed body has moved since, so a query widened by it misses none.
type crowd struct {
	cell       float32
	x0, y0     int32
	cols, rows int32
	half       float32
	starts     []int32
	counts     []int32
	entries    []*Body
	boxes      []geom.Rect
	density    []float32
	over       []float32
	centres    []geom.Vec2
	pushes     []geom.Vec2
	stamps     []uint32
	stamp      uint32
	touched    []int32
	loose      []*Body
	slack      float32
	outs       []geom.Vec2
	rooms      []float32
	nears      []uint8
}

// touch lists cell slot among those to clear at the next rebuild.
func (c *crowd) touch(slot int32) {
	if c.stamps[slot] != c.stamp {
		c.stamps[slot] = c.stamp
		c.touched = append(c.touched, slot)
	}
}

// wipe clears the cells written since the last rebuild.
func (c *crowd) wipe() {
	for _, slot := range c.touched {
		c.starts[slot], c.counts[slot], c.density[slot], c.over[slot], c.centres[slot] = 0, 0, 0, 0, geom.Vec2{}
	}
	c.touched = c.touched[:0]
	if c.stamp++; c.stamp == 0 {
		clear(c.stamps)
		c.stamp = 1
	}
}

func (c *crowd) cellOf(p geom.Vec2) (int32, int32) {
	return int32(math.Floor(float64(p.X / c.cell))), int32(math.Floor(float64(p.Y / c.cell)))
}

// slotOf is the cell p is in, -1 off the grid.
func (c *crowd) slotOf(p geom.Vec2) int32 {
	cx, cy := c.cellOf(p)
	cx, cy = cx-c.x0, cy-c.y0
	if cx < 0 || cy < 0 || cx >= c.cols || cy >= c.rows {
		return -1
	}
	return cy*c.cols + cx
}

// rebuild files every soft body of s where it is now, in the order s
// lists them. Cells are twice the largest soft body's side, larger only
// when the bodies are spread too far apart for maxCells of that size.
func (c *crowd) rebuild(s *Space) {
	c.wipe()
	n := 0
	var lo, hi geom.Vec2
	c.half = 0
	for _, b := range s.bodies {
		b.indexed = b.soft() && b.owner != nil
		if !b.indexed {
			b.slot = -1
			continue
		}
		b.box = b.Bounds()
		at := b.box.Center()
		if n == 0 {
			lo, hi = at, at
		}
		lo, hi = geom.V2(min(lo.X, at.X), min(lo.Y, at.Y)), geom.V2(max(hi.X, at.X), max(hi.Y, at.Y))
		c.half = max(c.half, b.box.W/2, b.box.H/2)
		n++
	}
	c.cell = max(4*c.half, 1)
	for {
		x0, y0 := c.cellOf(lo)
		x1, y1 := c.cellOf(hi)
		c.x0, c.y0 = x0-margin, y0-margin
		c.cols, c.rows = x1-x0+1+2*margin, y1-y0+1+2*margin
		if n == 0 || int(c.cols)*int(c.rows) <= maxCells {
			break
		}
		c.cell *= 2
	}
	cells := 0
	if n > 0 {
		cells = int(c.cols) * int(c.rows)
	}
	c.starts, c.counts = grown(c.starts, cells), grown(c.counts, cells)
	c.density, c.over = grown(c.density, cells), grown(c.over, cells)
	c.centres, c.stamps = grown(c.centres, cells), grown(c.stamps, cells)
	c.entries = resized(c.entries, n)
	c.pushes = resized(c.pushes, n)
	c.outs, c.rooms, c.nears = grown(c.outs, n*contacts), grown(c.rooms, n*contacts), resized(c.nears, n)
	c.boxes = resized(c.boxes, n)
	c.file(s)
	for _, b := range c.loose {
		b.waiting = false
	}
	clear(c.loose)
	c.loose, c.slack = c.loose[:0], 0
}

// file files the indexed bodies of s in the cells of their centres, in
// entries grouped by cell, and counts each cell's density and centre.
func (c *crowd) file(s *Space) {
	area := c.cell * c.cell
	for _, b := range s.bodies {
		if b.indexed {
			b.slot = c.slotOf(b.box.Center())
			b.share = b.area() / area
			c.touch(b.slot)
			c.counts[b.slot]++
			c.density[b.slot] += b.share
			c.centres[b.slot] = c.centres[b.slot].Add(b.box.Center())
		}
	}
	var next int32
	for _, slot := range c.touched {
		c.centres[slot] = c.centres[slot].Scale(1 / float32(c.counts[slot]))
		c.starts[slot] = next
		next += c.counts[slot]
	}
	for _, b := range s.bodies {
		if b.indexed {
			i := c.starts[b.slot]
			c.entries[i], c.boxes[i] = b, b.box
			b.entry = i
			c.starts[b.slot]++
		}
	}
	for _, slot := range c.touched {
		c.starts[slot] -= c.counts[slot]
	}
}

// moved notes that b, filed, now covers box: it widens the queries' slack
// and moves b's share to the cell it is in now.
func (c *crowd) moved(b *Body, box geom.Rect) {
	c.slack = max(c.slack, geom.Abs(box.X-b.box.X), geom.Abs(box.Y-b.box.Y))
	if box.W != b.box.W || box.H != b.box.H {
		c.slack = max(c.slack, geom.Abs(box.X+box.W-b.box.X-b.box.W), geom.Abs(box.Y+box.H-b.box.Y-b.box.H))
	}
	if slot := c.slotOf(box.Center()); slot != b.slot {
		c.leave(b)
		if b.slot = slot; slot >= 0 {
			c.touch(slot)
			c.density[slot] += b.share
		}
	}
}

// leave takes b's share out of its cell.
func (c *crowd) leave(b *Body) {
	if b.slot >= 0 {
		c.density[b.slot] -= b.share
	}
	b.slot = -1
}

// contact is the part of delta that keeps b, filed, from walking into the
// bodies that were near it when the crowd was filed further than the room
// it had to each, or deeper if it already overlapped one: the part along
// each one's way out past that is dropped, the rest kept, so b slides
// along and round them.
func (c *crowd) contact(b *Body, delta geom.Vec2) geom.Vec2 {
	if !b.indexed || delta == (geom.Vec2{}) {
		return delta
	}
	first := int(b.entry) * contacts
	for k := first; k < first+int(c.nears[b.entry]); k++ {
		if in := -delta.Dot(c.outs[k]); in > c.rooms[k] {
			delta = delta.Add(c.outs[k].Scale(in - c.rooms[k]))
		}
	}
	return delta
}

// noteNear notes for filed body i the way out of a body near it and the
// room it has towards it, keeping the nearest contacts of them.
func (c *crowd) noteNear(i int32, out geom.Vec2, room float32) {
	first := int(i) * contacts
	k := first + int(c.nears[i])
	if c.nears[i] < contacts {
		c.nears[i]++
	} else {
		k = first
		for m := first + 1; m < first+contacts; m++ {
			if c.rooms[m] > c.rooms[k] {
				k = m
			}
		}
		if c.rooms[k] <= room {
			return
		}
	}
	c.outs[k], c.rooms[k] = out, room
}

// room is the part of a spreading push delta that keeps b, filed, out of
// cells that would then cover more than limit: a push along an axis into
// such a cell is dropped, so overflow drains to the crowd's edge and not
// into cells as full.
func (c *crowd) room(b *Body, delta geom.Vec2, limit float32) geom.Vec2 {
	at := b.Bounds().Center()
	full := func(d geom.Vec2) bool {
		slot := c.slotOf(at.Add(d))
		return slot >= 0 && slot != b.slot && c.density[slot]+b.share > limit
	}
	if delta.X != 0 && full(geom.V2(delta.X, 0)) {
		delta.X = 0
	}
	if delta.Y != 0 && full(geom.V2(0, delta.Y)) {
		delta.Y = 0
	}
	if delta != (geom.Vec2{}) && full(delta) {
		delta = geom.Vec2{}
	}
	return delta
}

// collect appends to out the soft bodies whose bounds touch area: those
// filed in the cells under it, or, when they are more cells than bodies,
// every filed body.
func (c *crowd) collect(out []*Body, area geom.Rect) []*Body {
	if len(c.entries) > 0 {
		pad := c.half + c.slack
		wide := geom.Rect{X: area.X - c.slack, Y: area.Y - c.slack, W: area.W + 2*c.slack, H: area.H + 2*c.slack}
		take := func(i int32) {
			b := c.entries[i]
			if touches(wide, c.boxes[i]) && b.indexed && (c.slack == 0 || touches(area, b.Bounds())) {
				out = append(out, b)
			}
		}
		cx0, cy0 := c.cellOf(geom.V2(area.X-pad, area.Y-pad))
		cx1, cy1 := c.cellOf(geom.V2(area.X+area.W+pad, area.Y+area.H+pad))
		cx0, cy0 = max(cx0-c.x0, 0), max(cy0-c.y0, 0)
		cx1, cy1 = min(cx1-c.x0, c.cols-1), min(cy1-c.y0, c.rows-1)
		if cx0 <= cx1 && cy0 <= cy1 && int(cx1-cx0+1)*int(cy1-cy0+1) > len(c.entries) {
			for i := range c.entries {
				take(int32(i))
			}
			cy1 = -1
		}
		for cy := cy0; cy <= cy1 && cx0 <= cx1; cy++ {
			for slot := cy*c.cols + cx0; slot <= cy*c.cols+cx1; slot++ {
				for i := c.starts[slot]; i < c.starts[slot]+c.counts[slot]; i++ {
					take(i)
				}
			}
		}
	}
	for _, b := range c.loose {
		if b.waiting && !b.indexed && b.soft() && touches(area, b.Bounds()) {
			out = append(out, b)
		}
	}
	return out
}

// spread files the crowd again and pushes each soft body out of the
// bodies it overlaps and, where cells cover more than capacity, down the
// slope of their overflow and out from the middle of an over-full cell
// into cells with room, sliding each push against solids and bodies that
// aren't soft. Pushes are worked out from where the step found the bodies
// and applied in the order they are filed, so the same steps always give
// the same positions.
func (s *Space) spread() {
	c := &s.crowd
	c.rebuild(s)
	spilling := false
	for _, slot := range c.touched {
		d := c.density[slot]
		c.over[slot] = max(d-capacity, 0)
		spilling = spilling || d > capacity
	}
	c.separate()
	for i, b := range c.entries {
		if !b.indexed || b.owner == nil {
			continue
		}
		push := c.pushes[i]
		if spilling {
			limit := float32(capacity)
			if b.slot >= 0 {
				limit = max(limit, c.density[b.slot]-b.share)
			}
			push = push.Add(c.room(b, c.drain(b), limit))
		}
		if push == (geom.Vec2{}) {
			continue
		}
		if most := maxNudge * min(b.shape.size.X, b.shape.size.Y); push.LenSq() > most*most {
			push = push.Normalize().Scale(most)
		}
		to, _ := s.slide(b, b.owner.Position(), push, false)
		b.owner.SetPosition(to)
		s.file(b)
	}
}

// drain is the push down the slope of the overflow at b, and out of the
// middle of its cell when that is over-full.
func (c *crowd) drain(b *Body) geom.Vec2 {
	at := b.Bounds().Center()
	dir := c.slope(at).Scale(-1)
	if b.slot >= 0 && c.over[b.slot] > 0 {
		over := c.over[b.slot]
		out := at.Sub(c.centres[b.slot])
		if out.LenSq() < 1e-6 {
			out = geom.FromAngle(float32(b.ID()) * 2.39996)
		}
		dir = dir.Add(out.Normalize().Scale(over / 2))
	}
	return dir.Scale(c.cell * b.push)
}

// separate adds to pushes what moves apart two filed bodies that
// overlap, each half the way out, read as the ellipses their boxes hold
// so that a crowd settles in no rows, and notes for each body the bodies
// near it that its moves keep clear of. Each body is weighed against the
// next ones filed in its cell and the first ones of the cells right and
// below it that it reaches, so each pair once.
func (c *crowd) separate() {
	for _, slot := range c.touched {
		x, y := slot%c.cols, slot/c.cols
		from, to := c.starts[slot], c.starts[slot]+c.counts[slot]
		for i := from; i < to; i++ {
			box := &c.boxes[i]
			for j := i + 1; j < min(to, i+1+neighbours); j++ {
				if !far(box, &c.boxes[j]) {
					c.keepApart(i, j)
				}
			}
			at, rx, ry := box.Center(), near*(box.W/2+c.half), near*(box.H/2+c.half)
			u0, u1 := (at.X-rx)/c.cell-float32(c.x0+x), (at.X+rx)/c.cell-float32(c.x0+x)
			v1 := (at.Y+ry)/c.cell - float32(c.y0+y)
			right, left, below := u1 > 1, u0 < 0, v1 > 1
			for _, d := range [...]struct {
				dx, dy int32
				near   bool
			}{{1, 0, right}, {-1, 1, below && left}, {0, 1, below}, {1, 1, below && right}} {
				nx, ny := x+d.dx, y+d.dy
				if !d.near || nx < 0 || nx >= c.cols || ny >= c.rows {
					continue
				}
				n := ny*c.cols + nx
				for j := c.starts[n]; j < c.starts[n]+min(c.counts[n], neighbours); j++ {
					if !far(box, &c.boxes[j]) {
						c.keepApart(i, j)
					}
				}
			}
		}
	}
}

// far reports whether boxes p and q are too far apart, along x or y, to
// be near each other.
func far(p, q *geom.Rect) bool {
	rx, dx := near*(p.W+q.W)/2, q.X+q.W/2-p.X-p.W/2
	ry, dy := near*(p.H+q.H)/2, q.Y+q.H/2-p.Y-p.H/2
	return dx >= rx || dx <= -rx || dy >= ry || dy <= -ry
}

// keepApart notes filed bodies i and j as near each other when they are,
// and adds to pushes what moves them apart, each half the way, when they
// overlap.
func (c *crowd) keepApart(i, j int32) {
	p, q := c.boxes[i], c.boxes[j]
	reach := geom.V2((p.W+q.W)/2, (p.H+q.H)/2)
	gap := geom.V2(q.X+q.W/2-p.X-p.W/2, q.Y+q.H/2-p.Y-p.H/2)
	u := geom.V2(gap.X/reach.X, gap.Y/reach.Y)
	l := u.Len()
	if l >= near {
		return
	}
	dir := u.Scale(1 / l)
	if l < 1e-4 {
		dir = geom.FromAngle(float32(i+j) * 2.39996)
	}
	out := geom.V2(dir.X/reach.X, dir.Y/reach.Y).Normalize()
	room := max(l-1, 0) * geom.V2(dir.X*reach.X, dir.Y*reach.Y).Len() / 2
	c.noteNear(i, out.Scale(-1), room)
	c.noteNear(j, out, room)
	if l >= 1 {
		return
	}
	a, b := c.entries[i], c.entries[j]
	step := geom.V2(dir.X*reach.X, dir.Y*reach.Y).Scale((1 - l) / 2)
	c.pushes[i] = c.pushes[i].Sub(step.Scale(a.push))
	c.pushes[j] = c.pushes[j].Add(step.Scale(b.push))
}

// slope is how the overflow rises at p, per cell, interpolated between the
// slopes at the centres of the cells around it, each read from the
// overflow of its four neighbours as the step found it.
func (c *crowd) slope(p geom.Vec2) geom.Vec2 {
	u, v := p.X/c.cell-0.5-float32(c.x0), p.Y/c.cell-0.5-float32(c.y0)
	i, j := int32(math.Floor(float64(u))), int32(math.Floor(float64(v)))
	fx, fy := u-float32(i), v-float32(j)
	over := func(x, y int32) float32 {
		if x < 0 || y < 0 || x >= c.cols || y >= c.rows {
			return 0
		}
		return c.over[y*c.cols+x]
	}
	at := func(x, y int32) geom.Vec2 {
		if x < 0 || y < 0 || x >= c.cols || y >= c.rows {
			return geom.Vec2{}
		}
		return geom.V2(over(x+1, y)-over(x-1, y), over(x, y+1)-over(x, y-1)).Scale(0.5)
	}
	return at(i, j).Scale((1 - fx) * (1 - fy)).Add(at(i+1, j).Scale(fx * (1 - fy))).
		Add(at(i, j+1).Scale((1 - fx) * fy)).Add(at(i+1, j+1).Scale(fx * fy))
}

// grown is s at length n, its elements kept, a new zeroed array when s
// is too short.
func grown[T any](s []T, n int) []T {
	if cap(s) < n {
		return make([]T, n)
	}
	return s[:n]
}

func resized[T any](s []T, n int) []T {
	if cap(s) < n {
		return make([]T, n)
	}
	s = s[:n]
	clear(s)
	return s
}
