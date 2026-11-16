package nav //nolint:testpackage // white-box: the A* cells and a line's time

import (
	"math"
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/lantervale/ember/geom"
)

// marsh is a level whose ~ cells cost slow to cross.
type marsh struct {
	*level
	slow float32
}

func (m marsh) Cost(x, y int) float32 {
	if !m.Solid(x, y) && m.cells[y][x] == '~' {
		return m.slow
	}
	return 1
}

// wades reports whether a path from from crosses a ~ cell.
func wades(l *level, from geom.Vec2, path []geom.Vec2) bool {
	for _, p := range path {
		for k := float32(0); k <= 1; k += 0.01 {
			if x, y, ok := l.Cell(from.Add(p.Sub(from).Scale(k))); ok && l.cells[y][x] == '~' {
				return true
			}
		}
		from = p
	}
	return false
}

// walkTime is how long a path from from takes, slow ground counted.
func walkTime(t *testing.T, m *Map, from geom.Vec2, path []geom.Vec2) float64 {
	t.Helper()
	var total float64
	for _, p := range path {
		took, open := m.time(from, p)
		if !open {
			t.Fatalf("the leg %v → %v crosses a wall: %v", from, p, path)
		}
		total += float64(took)
		from = p
	}
	return total
}

func TestARouteGoesRoundSlowGroundWhenRoundIsQuicker(t *testing.T) {
	l := newLevel(
		"...........",
		"....~~~....",
		"....~~~....",
		"....~~~....",
		"...........",
	)
	from, to := at(0, 2), at(10, 2)
	if path, _ := New(l).Find(from, to); len(path) != 1 {
		t.Fatalf("a grid with no cost goes straight across: %v", path)
	}
	path, ok := New(marsh{l, 3}).Find(from, to)
	if !ok || path[len(path)-1] != to {
		t.Fatalf("round the pond: %v %v", path, ok)
	}
	if wades(l, from, path) {
		t.Fatalf("round is quicker, and no leg is pulled back across the pond: %v", path)
	}
	if len(path) > 4 {
		t.Fatalf("pulled to a few corners, not a cell each: %d waypoints %v", len(path), path)
	}
}

func TestARouteCrossesSlowGroundWhenRoundIsLonger(t *testing.T) {
	rows := slices.Repeat([]string{".....~....."}, 28)
	rows = append(rows, "...........", "...........")
	l := newLevel(rows...)
	from, to := at(0, 0), at(10, 0)
	path, ok := New(marsh{l, 3}).Find(from, to)
	if !ok || len(path) != 1 || path[0] != to {
		t.Fatalf("straight through the stream, its end is too far: %v %v", path, ok)
	}
	path, ok = New(marsh{l, 100}).Find(from, to)
	if !ok || wades(l, from, path) {
		t.Fatalf("slow enough, the same stream is gone round: %v %v", path, ok)
	}
}

func TestSlowGroundIsCrossedTheShortWay(t *testing.T) {
	l := newLevel(
		"............",
		"............",
		"~~~~~~~~~~~~",
		"~~~~~~~~~~~~",
		"............",
		"............",
	)
	m := New(marsh{l, 4})
	from, to := at(0, 0), at(11, 5)
	path, ok := m.Find(from, to)
	if !ok || path[len(path)-1] != to {
		t.Fatalf("across the ford: %v %v", path, ok)
	}
	if len(path) < 2 {
		t.Fatalf("the straight line stays in the water longer than the path does: %v", path)
	}
	straight := walkTime(t, m, from, []geom.Vec2{to})
	if took := walkTime(t, m, from, path); took >= straight {
		t.Fatalf("the path takes %.1f, the straight line %.1f", took, straight)
	}
}

func TestACostBelowOneCountsAsOne(t *testing.T) {
	l := newLevel(
		".....",
		"~~~~~",
		".....",
	)
	from, to := at(0, 0), at(4, 0)
	for _, cost := range []float32{0, 0.25, -3, float32(math.NaN())} {
		path, ok := New(marsh{l, cost}).Find(from, to)
		if !ok || len(path) != 1 || path[0] != to {
			t.Fatalf("no detour by ground costing %v: %v", cost, path)
		}
	}
}

// swamp is bigLevel with a third of its free cells slow.
func swamp() *level {
	l := bigLevel()
	random := rand.New(rand.NewPCG(5, 6))
	for y := range l.cells {
		for x := range l.cells[y] {
			if l.cells[y][x] == '.' && random.Float32() < 0.35 {
				l.cells[y][x] = '~'
			}
		}
	}
	return l
}

func TestACostOfOneEverywhereRoutesAsNoCost(t *testing.T) {
	l := bigLevel()
	random := rand.New(rand.NewPCG(7, 8))
	for _, opts := range [][]Option{nil, {NoDiagonals()}} {
		plain, costed := New(l, opts...), New(marsh{l, 1}, opts...)
		for range 200 {
			from, to := at(random.IntN(128), random.IntN(128)), at(random.IntN(128), random.IntN(128))
			want, wantOK := plain.Find(from, to)
			got, ok := costed.Find(from, to)
			if ok != wantOK || !slices.Equal(got, want) {
				t.Fatalf("%v → %v: %v %v, with no cost %v %v", from, to, got, ok, want, wantOK)
			}
		}
	}
}

// quickest is the least time from cell from to every cell, by Dijkstra
// without a heap.
func quickest(m *Map, from int) []float64 {
	m.fit()
	n := m.cols * m.rows
	dist, done := make([]float64, n), make([]bool, n)
	for i := range dist {
		dist[i] = math.Inf(1)
	}
	dist[from] = 0
	for {
		cur := -1
		for i := range dist {
			if !done[i] && !math.IsInf(dist[i], 1) && (cur < 0 || dist[i] < dist[cur]) {
				cur = i
			}
		}
		if cur < 0 {
			return dist
		}
		done[cur] = true
		cx, cy := cur%m.cols, cur/m.cols
		for _, d := range steps {
			nx, ny := cx+d.x, cy+d.y
			if d.diagonal && !m.diagonals || m.grid.Solid(nx, ny) ||
				d.diagonal && (m.grid.Solid(cx+d.x, cy) || m.grid.Solid(cx, cy+d.y)) {
				continue
			}
			step := float64(d.cost) * float64(m.costAt(cx, cy)+m.costAt(nx, ny)) / 2
			if next := ny*m.cols + nx; dist[cur]+step < dist[next] {
				dist[next] = dist[cur] + step
			}
		}
	}
}

func TestTheSearchFindsTheQuickestWayAndThePullIsNoSlower(t *testing.T) {
	random := rand.New(rand.NewPCG(11, 12))
	for range 40 {
		rows := make([]string, 12)
		for y := range rows {
			b := make([]byte, 14)
			for x := range b {
				b[x] = "..~~#"[random.IntN(5)]
			}
			rows[y] = string(b)
		}
		l := newLevel(rows...)
		for _, opts := range [][]Option{nil, {NoDiagonals()}} {
			m := New(marsh{l, 1 + 4*random.Float32()}, opts...)
			from := random.IntN(14 * 12)
			if l.Solid(from%14, from/14) {
				continue
			}
			best := quickest(m, from)
			for to, want := range best {
				a, b := at(from%14, from/14), at(to%14, to/14)
				cells, reached := m.cells(a, b)
				if reached == math.IsInf(want, 1) {
					t.Fatalf("%v → %v reached: %v, the quickest takes %v\n%v", a, b, reached, want, rows)
				}
				if !reached {
					continue
				}
				path := make([]geom.Vec2, 0, len(cells))
				for _, c := range cells[1:] {
					path = append(path, at(c%14, c/14))
				}
				// The search's tie-break may cost it a thousandth.
				took := walkTime(t, m, a, path)
				if took > want*10*1.002+1e-3 {
					t.Fatalf("%v → %v takes %.3f, the quickest %.3f\n%v", a, b, took, want*10, rows)
				}
				pulled, _ := m.Find(a, b)
				if after := walkTime(t, m, a, pulled); after > took*1.001+1e-3 {
					t.Fatalf("%v → %v pulled takes %.3f, the path %.3f: %v\n%v", a, b, after, took, pulled, rows)
				}
			}
		}
	}
}

func BenchmarkWorstPath128Cost(b *testing.B) {
	m := New(marsh{swamp(), 3})
	for b.Loop() {
		m.Find(at(0, 0), at(127, 127))
	}
}
