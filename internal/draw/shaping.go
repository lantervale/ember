package draw

import (
	"slices"

	"golang.org/x/image/font/sfnt"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/internal/shape"
	"github.com/lantervale/ember/render"
)

// maxShapedRuns bounds the shaped runs a font keeps.
const maxShapedRuns = 1 << 12

// indexKey is a glyph of a face by its index, rasterized at a phase.
type indexKey struct {
	face  int
	idx   sfnt.GlyphIndex
	phase int
}

// item is what a line of text is drawn as, one after the other: a
// glyph, a picture, or the end of the line.
type item struct {
	newline bool
	pic     *render.Texture
	// r is the first character of the glyph's cluster, and g its glyph:
	// what is drawn unless shaped is set, when idx of g's face is.
	r      rune
	g      glyph
	shaped bool
	idx    sfnt.GlyphIndex
	// advance is how far the glyph moves the pen, kerning included, and
	// offset how far it is drawn from the pen, y down, in the font's
	// points.
	advance float32
	offset  geom.Vec2
	// clusters is how many grapheme clusters the item draws.
	clusters int
}

// pending is a cluster of a run of one face waiting to be shaped.
type pending struct {
	r rune
	g glyph
}

// planMemo is the plan a face shapes with last asked for.
type planMemo struct {
	script   shape.Tag
	features string
	plan     *shape.Plan
}

// plan is face's plan for runs in script with features after its own.
func (f *Font) plan(face int, script shape.Tag, features string) *shape.Plan {
	if f.memos == nil {
		f.memos = make([]planMemo, len(f.layouts))
	}
	m := &f.memos[face]
	if m.plan == nil || m.script != script || m.features != features {
		*m = planMemo{script, features,
			f.layouts[face].Plan(script, shape.Splice(features, f.looks[face].Features))}
	}
	return m.plan
}

// items calls fn with what s is drawn as, shaped with features after
// each face's own: each run of clusters one face draws shaped together,
// a picture or a character no face has breaking the run.
func (f *Font) items(s, features string, fn func(item)) {
	run := f.pending[:0]
	face := -1
	script := shape.Script(s)
	flush := func() {
		if len(run) > 0 {
			f.shapeRun(face, run, script, features, fn)
		}
		run = run[:0]
	}
	eachCluster(s, func(c string, first rune) {
		if first == '\n' || first == '\r' {
			flush()
			fn(item{newline: true})
			return
		}
		if tex, ok := f.picture(c); ok {
			flush()
			fn(item{pic: tex, advance: f.size, clusters: 1})
			return
		}
		g := f.glyph(first)
		if g.idx == 0 || f.plan(g.face, script, features).Inert() {
			flush()
			fn(item{r: first, g: g, advance: g.advance, clusters: 1})
			return
		}
		if g.face != face {
			flush()
			face = g.face
		}
		run = append(run, pending{first, g})
	})
	flush()
	f.pending = run
}

// shapeRun calls fn with the glyphs run, clusters face draws, is drawn
// as: each its own glyph where the face's layout changes nothing.
func (f *Font) shapeRun(face int, run []pending, script shape.Tag, features string, fn func(item)) {
	p := f.plan(face, script, features)
	glyphs := f.run[:0]
	for i, c := range run {
		glyphs = append(glyphs, shape.Glyph{ID: uint16(c.g.idx), Cluster: int32(i)})
	}
	f.run = glyphs
	if !p.Touches(glyphs) {
		for _, c := range run {
			fn(item{r: c.r, g: c.g, advance: c.g.advance, clusters: 1})
		}
		return
	}
	shaped := f.shaped(p, glyphs)
	scale := f.size * f.looks[face].scale() / float32(f.sources[face].UnitsPerEm())
	v := f.looks[face].varied()
	for i, sg := range shaped {
		advance := float32(sg.Advance)
		if v != nil {
			// GPOS's values stay the default instance's; the advance is the instance's.
			advance += v.Font.Advance(sg.ID, v.Coords) - v.Font.Advance(sg.ID, nil)
		}
		c := run[sg.Cluster]
		next := int32(len(run)) //nolint:gosec // a run of a line
		if i+1 < len(shaped) {
			next = shaped[i+1].Cluster
		}
		fn(item{
			r: c.r, g: c.g, shaped: sg.ID != uint16(c.g.idx), idx: sfnt.GlyphIndex(sg.ID),
			advance:  advance * scale,
			offset:   geom.V2(float32(sg.XOffset)*scale, -float32(sg.YOffset)*scale),
			clusters: max(int(next-sg.Cluster), 0),
		})
	}
}

// shaped is glyphs shaped by p, kept for the next time the same run is.
func (f *Font) shaped(p *shape.Plan, glyphs []shape.Glyph) []shape.Glyph {
	ids := f.ids[:0]
	for _, g := range glyphs {
		ids = append(ids, byte(g.ID>>8), byte(g.ID&0xff))
	}
	f.ids = ids
	runs := f.shapedRuns[p]
	if out, ok := runs[string(ids)]; ok {
		return out
	}
	out := p.Shape(slices.Clone(glyphs))
	if f.shapedRuns == nil || f.shapedCount >= maxShapedRuns {
		f.shapedRuns, f.shapedCount = map[*shape.Plan]map[string][]shape.Glyph{}, 0
	}
	if f.shapedRuns[p] == nil {
		f.shapedRuns[p] = map[string][]shape.Glyph{}
	}
	f.shapedRuns[p][string(ids)] = out
	f.shapedCount++
	return out
}

// drawn is the glyph it draws at phase: its character's, or the one
// shaping put in its place.
func (f *Font) drawn(it item, phase int) glyph {
	if !it.shaped {
		return f.glyphAt(it.r, phase)
	}
	key := indexKey{it.g.face, it.idx, phase}
	if g, ok := f.indexed[key]; ok {
		return g
	}
	g := f.rasterIndex(it.g.face, it.idx, -1, phase)
	f.indexed[key] = g
	return g
}
