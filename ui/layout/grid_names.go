package layout

import (
	"slices"

	"github.com/lantervale/ember/ui/internal/style"
)

// gridLines is one axis of a grid's explicit grid: its track count, and the lines from zero
// each name is on, in order, those of its track list's names and of its areas' edges.
type gridLines struct {
	explicit int
	named    map[string][]int
	// listed are the names its track list gives each line, in its order.
	listed [][]string
}

func (g *gridLines) add(line int, names []string) {
	if line > g.explicit {
		return
	}
	if len(names) > 0 {
		for len(g.listed) <= line {
			g.listed = append(g.listed, nil)
		}
		g.listed[line] = append(g.listed[line], names...)
	}
	for _, n := range names {
		if g.named == nil {
			g.named = map[string][]int{}
		}
		g.named[n] = append(g.named[n], line)
	}
}

// finish names the edges of st's areas, and sorts each name's lines.
func (g *gridLines) finish(st *style.Computed, a axis) {
	for _, area := range st.Areas {
		edges := area.Row
		if a == horizontal {
			edges = area.Column
		}
		g.add(edges[0]-1, []string{area.Name + "-start"})
		g.add(edges[1]-1, []string{area.Name + "-end"})
	}
	for n, lines := range g.named {
		slices.Sort(lines)
		g.named[n] = slices.Compact(lines)
	}
}

// subgridLines are the lines of a subgrid in an axis it takes its grid's tracks in: the names
// its grid gives them, those of its own list, its repeat(auto-fill) as many times as its lines
// hold, and those of its areas' edges.
func subgridLines(st *style.Computed, a axis, sub *subgridAxis) gridLines {
	g := gridLines{explicit: sub.span}
	for n, lines := range sub.names {
		for _, l := range lines {
			g.add(l, []string{n})
		}
	}
	list := st.Columns
	if a == vertical {
		list = st.Rows
	}
	fixed := 0
	for _, r := range list {
		if r.Count != style.AutoFill {
			fixed += r.Count * len(r.Names)
		}
	}
	line := 0
	for _, r := range list {
		count := r.Count
		if count == style.AutoFill {
			count = max(sub.span+1-fixed, 0) / len(r.Names)
		}
		for range min(count, maxGridTracks) {
			for _, names := range r.Names {
				g.add(line, names)
				line++
			}
		}
	}
	g.finish(st, a)
	return g
}

// within are the names of the lines from s.start to s.end, from s.start.
func (g gridLines) within(s lineSpan) map[string][]int {
	var out map[string][]int
	for n, lines := range g.named {
		for _, l := range lines {
			if l < s.start || l > s.end {
				continue
			}
			if out == nil {
				out = map[string][]int{}
			}
			out[n] = append(out[n], l-s.start)
		}
	}
	return out
}

func newGridLines(st *style.Computed, a axis, autoRepetitions, explicit int) gridLines {
	g := gridLines{explicit: explicit}
	add := g.add
	template, _, _ := templateOf(st, a)
	line := 0
	for _, r := range template {
		add(line, r.Before)
		count := r.Count
		if isAutoRepeat(r) {
			count = autoRepetitions
		}
		for range min(count, maxGridTracks) {
			for i := range r.Tracks {
				if r.Names != nil {
					add(line, r.Names[i])
				}
				line++
			}
			if r.Names != nil {
				add(line, r.Names[len(r.Tracks)])
			}
			if line > explicit {
				break
			}
		}
		add(line, r.After)
	}
	g.finish(st, a)
	return g
}

// templateAreaTracks is how many tracks of an axis the areas of grid-template-areas take.
func templateAreaTracks(areas []style.GridArea, a axis) int {
	n := 0
	for _, area := range areas {
		if a == horizontal {
			n = max(n, area.Column[1]-1)
		} else {
			n = max(n, area.Row[1]-1)
		}
	}
	return n
}

// placement is lines from zero: a name alone is its -start or -end line on that side, else its
// first line; when there are fewer lines of a name than counted, every implicit line on the
// side counted towards bears it. A span to a name counts from the other end, a span of 1 when
// that end is auto.
func (g gridLines) placement(lines [2]style.GridLine) ozLines {
	p := ozLines{g.line(lines[0], "-start"), g.line(lines[1], "-end")}
	if p.start.kind == placeSpan && p.end.kind == placeSpan {
		p.end = ozPlacement{}
		if lines[0].Name != "" {
			p.start.v = 1
		}
		return p
	}
	switch {
	case lines[0].Name != "" && lines[0].Span > 0:
		p.start = g.spanTo(lines[0], p.end, false)
	case lines[1].Name != "" && lines[1].Span > 0:
		p.end = g.spanTo(lines[1], p.start, true)
	}
	return p
}

func (g gridLines) line(l style.GridLine, side string) ozPlacement {
	switch {
	case l.Name == "":
		return toOriginZero(l, g.explicit)
	case l.Span > 0:
		return ozPlacement{placeSpan, min(l.Span, maxGridTracks)}
	case l.Line == 0:
		if at := g.named[l.Name+side]; len(at) > 0 {
			return ozPlacement{placeLine, at[0]}
		}
		l.Line = 1
	}
	at := g.named[l.Name]
	if l.Line > 0 {
		if l.Line <= len(at) {
			return ozPlacement{placeLine, at[l.Line-1]}
		}
		return ozPlacement{placeLine, clampOZ(g.explicit + l.Line - len(at))}
	}
	if k := -l.Line; k <= len(at) {
		return ozPlacement{placeLine, at[len(at)-k]}
	}
	return ozPlacement{placeLine, clampOZ(len(at) + l.Line)}
}

// spanTo is the line a span to a name reaches from the other end, forward from a start or
// back from an end.
func (g gridLines) spanTo(l style.GridLine, from ozPlacement, forward bool) ozPlacement {
	if from.kind != placeLine {
		return ozPlacement{placeSpan, 1}
	}
	at := g.named[l.Name]
	n := l.Span
	if forward {
		for _, line := range at {
			if line > from.v {
				if n--; n == 0 {
					return ozPlacement{placeLine, line}
				}
			}
		}
		return ozPlacement{placeLine, clampOZ(max(from.v, g.explicit) + n)}
	}
	for _, line := range slices.Backward(at) {
		if line < from.v {
			if n--; n == 0 {
				return ozPlacement{placeLine, line}
			}
		}
	}
	return ozPlacement{placeLine, clampOZ(min(from.v, 0) - n)}
}
