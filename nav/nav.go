// Package nav finds paths around the solid cells of a grid, such as a
// twod.TileMap: A* with diagonal moves that never cut a wall's corner, the
// path pulled straight wherever the way is clear, and routes that plan
// again when the grid changes. A grid whose ground is slower in places (a
// Coster) gets the quickest way rather than the shortest. It is pure Go,
// for games and servers alike.
//
//	m := nav.New(level)
//	route := m.Route(hero.Position(), click)
//	hero.Translate(route.Toward(hero.Position(), 4).Scale(speed * dt))
package nav

import (
	"math"

	"github.com/lantervale/ember/geom"
)

// Grid is cells, some of them solid, laid out in a plane: a twod.TileMap,
// or a server's own level. nav never compares grids, so a grid's type may
// hold funcs, maps or slices.
type Grid interface {
	Columns() int
	Rows() int
	// Solid reports whether a cell blocks the way; cells outside count as
	// solid.
	Solid(x, y int) bool
	// Cell is the cell under p.
	Cell(p geom.Vec2) (x, y int, ok bool)
	// Center is the middle of cell x, y.
	Center(x, y int) geom.Vec2
}

// Coster is a Grid whose ground is slower to cross in places. Cost is how
// many times longer cell x, y takes than quick ground: 1 for quick ground,
// 3 for ground crossed at a third of the speed, and anything below 1
// counts as 1. A Map then finds the quickest way, crossing slow ground
// only where going round takes longer, and pulls it straight only where
// that is no slower. A cell nothing may cross is Solid, not costly.
type Coster interface {
	Grid
	Cost(x, y int) float32
}

// versioned is a grid that says when its cells change.
type versioned interface{ Version() uint64 }

// Map finds paths over a grid, keeping the memory its searches use.
type Map struct {
	grid      Grid
	cost      Coster
	diagonals bool

	cols, rows int
	g          []float32
	parent     []int32
	stamp      []uint32
	closed     []uint32
	search     uint32
	open       openSet
	along      []float64
}

// Option sets up a Map.
type Option interface{ mapOption(*Map) }

// SharedOption sets up a Map or a FlowField.
type SharedOption interface {
	Option
	FlowOption
}

type noDiagonals struct{}

func (noDiagonals) mapOption(m *Map)        { m.diagonals = false }
func (noDiagonals) flowOption(f *FlowField) { f.diagonals = false }

// NoDiagonals keeps paths to the four straight directions.
func NoDiagonals() SharedOption { return noDiagonals{} }

// New returns a map finding paths over grid.
func New(grid Grid, opts ...Option) *Map {
	m := &Map{grid: grid, diagonals: true}
	m.cost, _ = grid.(Coster)
	for _, o := range opts {
		o.mapOption(m)
	}
	return m
}

// Diagonals reports whether paths may move diagonally.
func (m *Map) Diagonals() bool { return m.diagonals }

// Grid is the grid the map finds paths over.
func (m *Map) Grid() Grid { return m.grid }

func (m *Map) fit() {
	cols, rows := m.grid.Columns(), m.grid.Rows()
	if cols == m.cols && rows == m.rows {
		return
	}
	n := cols * rows
	m.cols, m.rows = cols, rows
	m.g, m.parent, m.stamp, m.closed = make([]float32, n), make([]int32, n), make([]uint32, n), make([]uint32, n)
	m.search = 0
}

// Find is the way from from to to, as waypoints after from, the last one
// to; ok is false when to is solid or out of reach, and the way then ends
// at the reachable cell nearest it.
func (m *Map) Find(from, to geom.Vec2) ([]geom.Vec2, bool) {
	cells, reached := m.cells(from, to)
	if len(cells) == 0 {
		return nil, false
	}
	points := make([]geom.Vec2, 0, len(cells)+1)
	points = append(points, from)
	for _, c := range cells[1:] {
		points = append(points, m.grid.Center(c%m.cols, c/m.cols))
	}
	if reached {
		if len(points) > 1 {
			points[len(points)-1] = to
		} else {
			points = append(points, to)
		}
	}
	return m.pull(points), reached
}

// cells is the A* path of cell indexes from from's cell towards to's,
// from's cell first, and whether it reaches to's cell.
func (m *Map) cells(from, to geom.Vec2) ([]int, bool) {
	m.fit()
	sx, sy, ok := m.grid.Cell(from)
	if !ok || m.cols == 0 {
		return nil, false
	}
	tx, ty, _ := m.grid.Cell(to)
	m.search++
	if m.search == 0 {
		clear(m.stamp)
		clear(m.closed)
		m.search = 1
	}
	start, goal := sy*m.cols+sx, ty*m.cols+tx
	goalIn := tx >= 0 && ty >= 0 && tx < m.cols && ty < m.rows && !m.grid.Solid(tx, ty)
	h := func(i int) float32 {
		dx, dy := abs(i%m.cols-tx), abs(i/m.cols-ty)
		if !m.diagonals {
			return float32(dx+dy) * tieBreak
		}
		return (float32(dx+dy) + (math.Sqrt2-2)*float32(min(dx, dy))) * tieBreak
	}
	m.open = m.open[:0]
	m.g[start], m.parent[start], m.stamp[start] = 0, -1, m.search
	m.open.push(node{start, h(start)})
	best, bestH := start, h(start)
	for len(m.open) > 0 {
		cur := m.open.pop().cell
		if m.closed[cur] == m.search {
			continue
		}
		m.closed[cur] = m.search
		if hc := h(cur); hc < bestH {
			best, bestH = cur, hc
		}
		if goalIn && cur == goal {
			best = goal
			break
		}
		cx, cy := cur%m.cols, cur/m.cols
		slow := m.cellCost(cx, cy)
		for _, d := range steps {
			if d.diagonal && !m.diagonals {
				continue
			}
			nx, ny := cx+d.x, cy+d.y
			if m.grid.Solid(nx, ny) || d.diagonal && (m.grid.Solid(cx+d.x, cy) || m.grid.Solid(cx, cy+d.y)) {
				continue
			}
			next := ny*m.cols + nx
			if m.closed[next] == m.search {
				continue
			}
			g := m.g[cur] + m.stepCost(d.cost, slow, nx, ny)
			if m.stamp[next] == m.search && g >= m.g[next] {
				continue
			}
			//nolint:gosec // a cell index, as parent keeps them
			m.g[next], m.parent[next], m.stamp[next] = g, int32(cur), m.search
			m.open.push(node{next, g + h(next)})
		}
	}
	return m.trace(best), goalIn && best == goal
}

// trace is the path of cell indexes the search took to end, start first.
func (m *Map) trace(end int) []int {
	var out []int
	for c := end; c >= 0; c = int(m.parent[c]) {
		out = append(out, c)
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}

// cellCost is the cost of cell x, y over a Coster, 0 over a plain grid.
func (m *Map) cellCost(x, y int) float32 {
	if m.cost == nil {
		return 0
	}
	return m.costAt(x, y)
}

// stepCost is what a step of length long from a cell costing from to cell
// x, y takes: its length over a plain grid, half of it in each cell over a
// Coster.
func (m *Map) stepCost(long, from float32, x, y int) float32 {
	if m.cost == nil {
		return long
	}
	return long * (from + m.costAt(x, y)) / 2
}

// costAt is the cost of cell x, y, never below 1 so that the search's
// estimate of what is left stays below the truth.
func (m *Map) costAt(x, y int) float32 {
	if c := m.cost.Cost(x, y); c >= 1 {
		return c
	}
	return 1
}

// pull keeps only the waypoints the straight way can't skip, going as far
// ahead from each as is clear, and drops points[0], where the walk starts.
// Over a Coster the straight way must also be no slower than the stretch
// of path it replaces.
func (m *Map) pull(points []geom.Vec2) []geom.Vec2 {
	if m.cost != nil {
		m.along = append(m.along[:0], 0)
		for i := 1; i < len(points); i++ {
			took, _ := m.time(points[i-1], points[i])
			m.along = append(m.along, m.along[i-1]+float64(took))
		}
	}
	var out []geom.Vec2
	at := 0
	for at < len(points)-1 {
		next := at + 1
		for next+1 < len(points) && m.skips(points, at, next+1) {
			next++
		}
		out = append(out, points[next])
		at = next
	}
	return out
}

// skips reports whether the straight way from points[at] to points[to] may
// replace the path between them.
func (m *Map) skips(points []geom.Vec2, at, to int) bool {
	if m.cost == nil {
		return m.Clear(points[at], points[to])
	}
	took, open := m.time(points[at], points[to])
	return open && float64(took) <= (m.along[to]-m.along[at])*pullSlack
}

// time is how long the straight way from a to b takes over a Coster, as a
// length on quick ground, and whether it is clear.
func (m *Map) time(a, b geom.Vec2) (float32, bool) {
	took, open := m.line(a, b, true)
	return took * a.Dist(b), open
}

// pullSlack lets a straight way that takes as long as the path replace it,
// whatever the sums round to.
const pullSlack = 1 + 1e-4

// Clear reports whether the straight way from a to b crosses no solid
// cell, nor slips between two diagonal ones.
func (m *Map) Clear(a, b geom.Vec2) bool {
	_, open := m.line(a, b, false)
	return open
}

// line walks the straight way from a to b cell by cell: whether it is
// clear, and when timed over a Coster, how long it takes in lengths of the
// way, each cell's part counted cost times.
func (m *Map) line(a, b geom.Vec2, timed bool) (float32, bool) {
	ax, ay, ok := m.grid.Cell(a)
	if !ok {
		return 0, false
	}
	o := m.grid.Center(0, 0)
	size := m.grid.Center(1, 1).Sub(o)
	if size.X == 0 || size.Y == 0 {
		return 0, false
	}
	// u, v are a and b in cells, the centre of cell 0, 0 at 0.5, 0.5.
	ua, va := (a.X-o.X)/size.X+0.5, (a.Y-o.Y)/size.Y+0.5
	ub, vb := (b.X-o.X)/size.X+0.5, (b.Y-o.Y)/size.Y+0.5
	x, y := ax, ay
	bx, by, _ := m.grid.Cell(b)
	du, dv := ub-ua, vb-va
	stepX, stepY := sign(du), sign(dv)
	tMaxX, tMaxY := boundary(ua, du), boundary(va, dv)
	tDeltaX, tDeltaY := inverse(du), inverse(dv)
	// t is how far along the cell x, y is entered, took the cost so far, in
	// fractions of the way.
	var t, took float32
	for range 2*(m.cols+m.rows) + 4 {
		if m.grid.Solid(x, y) {
			return took, false
		}
		leaves := min(tMaxX, tMaxY)
		last := x == bx && y == by || leaves > 1
		if timed {
			end := leaves
			if last {
				end = 1
			}
			took += (end - t) * m.costAt(x, y)
			t = end
		}
		if last {
			return took, true
		}
		switch {
		case tMaxX < tMaxY-1e-6:
			x += stepX
			tMaxX += tDeltaX
		case tMaxY < tMaxX-1e-6:
			y += stepY
			tMaxY += tDeltaY
		default:
			if m.grid.Solid(x+stepX, y) || m.grid.Solid(x, y+stepY) {
				return took, false
			}
			x, y = x+stepX, y+stepY
			tMaxX += tDeltaX
			tMaxY += tDeltaY
		}
	}
	return took, false
}

// boundary is the fraction of the way, d long from u, at which the first
// cell boundary is crossed.
func boundary(u, d float32) float32 {
	switch {
	case d > 0:
		return (float32(math.Floor(float64(u))) + 1 - u) / d
	case d < 0:
		return (u - float32(math.Floor(float64(u)))) / -d
	}
	return float32(math.Inf(1))
}

func inverse(d float32) float32 {
	if d == 0 {
		return float32(math.Inf(1))
	}
	return 1 / geom.Abs(d)
}

func sign(v float32) int {
	switch {
	case v > 0:
		return 1
	case v < 0:
		return -1
	}
	return 0
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// tieBreak nudges the heuristic so that among cells as promising, the one
// nearer the goal is opened first, and fewer cells are opened.
const tieBreak = 1 + 1.0/1024

type step struct {
	x, y     int
	cost     float32
	diagonal bool
}

//nolint:gochecknoglobals // a lookup table: the eight moves, read in the search's inner loop
var steps = [...]step{
	{1, 0, 1, false}, {-1, 0, 1, false}, {0, 1, 1, false}, {0, -1, 1, false},
	{1, 1, math.Sqrt2, true}, {1, -1, math.Sqrt2, true}, {-1, 1, math.Sqrt2, true}, {-1, -1, math.Sqrt2, true},
}

type node struct {
	cell int
	f    float32
}

// openSet is a binary min-heap of cells by their f score.
type openSet []node

func (s *openSet) push(n node) {
	*s = append(*s, n)
	h := *s
	for i := len(h) - 1; i > 0; {
		p := (i - 1) / 2
		if h[p].f <= h[i].f {
			break
		}
		h[p], h[i] = h[i], h[p]
		i = p
	}
}

func (s *openSet) pop() node {
	h := *s
	top := h[0]
	last := len(h) - 1
	h[0] = h[last]
	h = h[:last]
	for i := 0; ; {
		l, r, small := 2*i+1, 2*i+2, i
		if l < len(h) && h[l].f < h[small].f {
			small = l
		}
		if r < len(h) && h[r].f < h[small].f {
			small = r
		}
		if small == i {
			break
		}
		h[i], h[small] = h[small], h[i]
		i = small
	}
	*s = h
	return top
}
