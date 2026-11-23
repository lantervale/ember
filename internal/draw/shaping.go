package draw

import (
	"slices"
	"strings"
	"unicode/utf8"

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

// pending is a character of a run of one face waiting to be shaped:
// the first of its cluster, whose glyph is g, or one after it, glyph
// idx of the face.
type pending struct {
	r       rune
	g       glyph
	idx     sfnt.GlyphIndex
	cluster int32
	first   bool
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
// a picture or a character no face has breaking the run. Text right to
// left is shaped in its order and each line given from its end, its
// characters that have mirror images drawn as them.
func (f *Font) items(s, features string, rtl bool, fn func(item)) {
	if rtl {
		line := f.line[:0]
		f.items(mirrored(s), features, false, func(it item) {
			if !it.newline {
				line = append(line, it)
				return
			}
			for i := len(line) - 1; i >= 0; i-- {
				fn(line[i])
			}
			line = line[:0]
			fn(it)
		})
		for i := len(line) - 1; i >= 0; i-- {
			fn(line[i])
		}
		f.line = line[:0]
		return
	}
	run := f.pending[:0]
	face := -1
	var clusters int32
	script := shape.Script(s)
	flush := func() {
		if len(run) > 0 {
			f.shapeRun(face, run, clusters, script, features, fn)
		}
		run, clusters = run[:0], 0
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
		for _, r := range c {
			if sc, ok := shape.ScriptOf(r); ok {
				if sc != script {
					flush()
					script = sc
				}
				break
			}
		}
		g := f.glyph(first)
		if g.idx == 0 && len(f.layouts) > 0 && f.layouts[0].Decomposes(first) {
			g.face = 0
		} else if g.idx == 0 || f.plan(g.face, script, features).Inert() {
			flush()
			fn(item{r: first, g: g, advance: g.advance, clusters: 1})
			return
		}
		if g.face != face {
			flush()
			face = g.face
		}
		run = append(run, pending{r: first, g: g, idx: g.idx, cluster: clusters, first: true})
		for _, r := range c[utf8.RuneLen(first):] {
			if idx, err := f.sources[face].GlyphIndex(&f.buf, r); err == nil && (idx != 0 || r == zwj || r == zwnj) {
				run = append(run, pending{r: r, g: g, idx: idx, cluster: clusters})
			}
		}
		clusters++
	})
	flush()
	f.pending = run
}

// The joiners a run keeps when its face lacks them, for its shaping.
const (
	zwnj = 0x200C
	zwj  = 0x200D
)

// mirrored is s with each character that has a mirror image drawn as
// it, as text right to left draws it (UAX #9, L4).
func mirrored(s string) string {
	if !strings.ContainsFunc(s, func(r rune) bool { _, ok := shape.Mirror(r); return ok }) {
		return s
	}
	return strings.Map(func(r rune) rune {
		m, _ := shape.Mirror(r)
		return m
	}, s)
}

// shapeRun calls fn with the glyphs run, characters face draws of n
// clusters, is drawn as: each cluster its first character's glyph where
// the face's layout changes nothing.
func (f *Font) shapeRun(face int, run []pending, n int32, script shape.Tag, features string, fn func(item)) {
	p := f.plan(face, script, features)
	glyphs := f.run[:0]
	starts := f.starts[:0]
	for i, c := range run {
		if c.first {
			starts = append(starts, i)
		}
		glyphs = append(glyphs, shape.Glyph{ID: uint16(c.idx), Cluster: c.cluster, Char: c.r})
	}
	f.run, f.starts = glyphs, starts
	if !p.Touches(glyphs) {
		for _, c := range run {
			if c.first {
				fn(item{r: c.r, g: c.g, advance: c.g.advance, clusters: 1})
			}
		}
		return
	}
	shaped := f.shaped(p, glyphs)
	scale := f.size * f.looks[face].scale() / float32(f.sources[face].UnitsPerEm())
	v := f.looks[face].varied()
	for i, sg := range shaped {
		advance := float32(sg.Advance)
		if v != nil && advance != 0 {
			// GPOS's values stay the default instance's; the advance is the instance's.
			advance += v.Font.Advance(sg.ID, v.Coords) - v.Font.Advance(sg.ID, nil)
		}
		c := run[starts[sg.Cluster]]
		next := n
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

// shaped is glyphs shaped by p, kept for the next time the same run, of
// the same glyphs in the same clusters, is.
func (f *Font) shaped(p *shape.Plan, glyphs []shape.Glyph) []shape.Glyph {
	ids := f.ids[:0]
	for i, g := range glyphs {
		ids = append(ids, byte(g.ID>>8), byte(g.ID&0xff))
		if i > 0 && g.Cluster == glyphs[i-1].Cluster {
			ids = append(ids, 0)
		}
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
