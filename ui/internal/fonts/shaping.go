package fonts

import (
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"golang.org/x/image/font"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"

	"github.com/lantervale/ember/internal/shape"
	"github.com/lantervale/ember/internal/varfont"
	"github.com/lantervale/ember/ui/internal/style"
)

// maxShaped bounds the widths of shaped runs a set keeps.
const maxShaped = 1 << 13

// smallCapitals are the features of small capitals drawn from both cases.
const smallCapitals = "smcp,c2sc"

// Features of capitals, as Caps tells them apart.
const (
	tagSmcp = "smcp"
	tagUnic = "unic"
)

// featureSettings is features, HarfBuzz's, as font-feature-settings
// writes them.
func featureSettings(features string) string {
	fs := shape.ParseFeatures(features)
	parts := make([]string, len(fs))
	for i, f := range fs {
		parts[i] = fmt.Sprintf("%q %d", f.Tag.String(), f.Value)
	}
	return strings.Join(parts, ", ")
}

// layout is c's layout tables, read once.
func (c *Face) layout() *shape.Font {
	if c.shaper == nil {
		c.shaper = shape.Parse(c.font, c.TTF)
	}
	return c.shaper
}

// joined is features with the face's own among them (shape.Splice).
func (c *Face) joined(features string) string { return shape.Splice(features, c.Look.Features) }

// plan is c's plan for runs in script with features at instance at of
// a variable font, nil for its default.
func (c *Face) plan(script shape.Tag, features string, at varfont.Coords) *shape.Plan {
	if at == nil {
		return c.layout().Plan(script, c.joined(features))
	}
	return c.layout().PlanAt(script, c.joined(features), c.Variable(), at)
}

// shapedRun is a run shaped: its width in its face's units, and each of
// its glyphs' advance and the glyph of the run its cluster starts at.
type shapedRun struct {
	width    int32
	advances []int32
	starts   []int32
}

// part is the bytes [from, to) of a text measured whose glyphs its width
// counts, a glyph a part's where its cluster starts: all of them where to
// is negative.
type part struct{ from, to int }

// all is the part that is the whole text.
var all = part{0, -1} //nolint:gochecknoglobals // a constant of a struct type

func (p part) has(i int) bool { return p.to < 0 || i >= p.from && i < p.to }

// in is p in the text from byte at of the one it is a part of.
func (p part) in(at int) part {
	if p.to < 0 {
		return p
	}
	return part{p.from - at, p.to - at}
}

// covers reports whether every glyph of run, clustered by bytes, is p's.
func (p part) covers(run []shape.Glyph) bool {
	return p.to < 0 || len(run) == 0 || p.has(int(run[0].Cluster)) && p.has(int(run[len(run)-1].Cluster))
}

// shaped is the advance of the glyphs own has of run, glyphs of c, in
// c's units at instance at of a variable font (nil for its default):
// shaped with the features, in script, when its layout does anything to
// them, else the sum of their advances.
func (s *Set) shaped(c *Face, script shape.Tag, features string, run []shape.Glyph, at varfont.Coords,
	ctx context, own part) float32 {
	p := c.plan(script, features, at)
	if !p.Touches(run) {
		return s.unshaped(c, run, at, own)
	}
	r := s.shapedRun(p, run, ctx)
	if own.covers(run) {
		return float32(r.width)
	}
	var w float32
	for k, a := range r.advances {
		if own.has(int(run[r.starts[k]].Cluster)) {
			w += float32(a)
		}
	}
	return w
}

// unshaped is the sum of the advances of the glyphs own has of run, in
// c's units at instance at.
func (s *Set) unshaped(c *Face, run []shape.Glyph, at varfont.Coords, own part) float32 {
	if at != nil {
		var w float32
		for _, g := range run {
			if own.has(int(g.Cluster)) {
				w += c.web.vf.Advance(g.ID, at)
			}
		}
		return w
	}
	ppem := fixed.I(int(c.upem))
	var w fixed.Int26_6
	for _, g := range run {
		if !own.has(int(g.Cluster)) {
			continue
		}
		if a, err := c.font.GlyphAdvance(&s.buf, sfnt.GlyphIndex(g.ID), ppem, font.HintingNone); err == nil {
			w += a
		}
	}
	return float32(w >> 6)
}

// shapedRun is run shaped by p between ctx, kept for the next time the
// same glyphs are.
func (s *Set) shapedRun(p *shape.Plan, run []shape.Glyph, ctx context) shapedRun {
	ids := s.ids[:0]
	for _, g := range run {
		ids = append(ids, byte(g.ID>>8), byte(g.ID&0xff))
	}
	if !p.Joins() {
		ctx = context{}
	}
	ids = append(append(append(append(ids, 0xff), ctx.before...), 0xff), ctx.after...)
	s.ids = ids
	r, ok := s.shapedRuns[p][string(ids)]
	if ok {
		return r
	}
	k := 0
	for _, g := range p.ShapeIn(slices.Clone(run), ctx.before, ctx.after) {
		r.width += g.Advance
		r.advances = append(r.advances, g.Advance)
		for k > 0 && run[k].Cluster > g.Cluster {
			k--
		}
		for k+1 < len(run) && run[k+1].Cluster <= g.Cluster {
			k++
		}
		r.starts = append(r.starts, int32(k))
	}
	if s.shapedRuns == nil || s.shapedCount >= maxShaped {
		s.shapedRuns, s.shapedCount = map[*shape.Plan]map[string]shapedRun{}, 0
	}
	if s.shapedRuns[p] == nil {
		s.shapedRuns[p] = map[string]shapedRun{}
	}
	s.shapedRuns[p][string(ids)] = r
	s.shapedCount++
	return r
}

// context is the text before a run of glyphs and after it, as much as
// shaping reads.
type context struct{ before, after string }

// contextBytes is more than the characters of context shaping reads.
const contextBytes = 32

// around is the context of run, glyphs of text clustered by their bytes,
// in text between before and after.
func around(text string, run []shape.Glyph, before, after string) context {
	if len(run) == 0 || before == "" && after == "" && len(run) == utf8.RuneCountInString(text) {
		return context{}
	}
	last := run[len(run)-1]
	start, end := int(run[0].Cluster), int(last.Cluster)+utf8.RuneLen(last.Char)
	b, a := before+text[:start], text[end:]+after
	if len(b) > contextBytes {
		b = b[len(b)-contextBytes:]
	}
	if len(a) > contextBytes {
		a = a[:contextBytes]
	}
	return context{strings.ToValidUTF8(b, ""), strings.ToValidUTF8(a, "")}
}

// SpaceTies reports whether f's face shapes a space with the words on
// either side of it, kerning or joining them, as Chrome then shapes the
// whole run: a word's width takes its kerning with the space after it,
// and the space's its kerning with the word after it.
func (s *Set) SpaceTies(f style.Font) bool {
	c := s.Face(f)
	if ch := s.chained(f); ch != nil {
		c = ch.first()
	}
	if c == nil || c.font == nil {
		return false
	}
	g, err := c.font.GlyphIndex(&s.buf, ' ')
	if err != nil || g == 0 {
		return false
	}
	return c.plan(shape.MakeTag("latn"), f.Features, c.Instance(f)).Mentions(uint16(g))
}

// capsFeatures are the features that draw f's capitals with c, as
// Chrome takes them from a face that has them: "" when c has none of
// them, and f's capitals are synthesised.
func capsFeatures(c *Face, f style.Font, script shape.Tag) string {
	if c == nil || c.font == nil {
		return ""
	}
	l := c.layout()
	has := func(tags ...string) bool {
		for _, t := range tags {
			if !l.Has(script, shape.MakeTag(t)) {
				return false
			}
		}
		return true
	}
	switch f.Caps {
	case style.SmallCaps:
		if has(tagSmcp) {
			return tagSmcp
		}
	case style.AllSmallCaps:
		if has(tagSmcp, "c2sc") {
			return smallCapitals
		}
	case style.PetiteCaps:
		if has("pcap") {
			return "pcap"
		}
		if has(tagSmcp) {
			return tagSmcp
		}
	case style.AllPetiteCaps:
		if has("pcap", "c2pc") {
			return "pcap,c2pc"
		}
		if has(tagSmcp, "c2sc") {
			return smallCapitals
		}
	case style.Unicase:
		if has(tagUnic) {
			return tagUnic
		}
		if has(tagSmcp, "c2sc") {
			return smallCapitals
		}
	case style.TitlingCaps:
		if has("titl") {
			return "titl"
		}
	case style.NormalCaps, style.UnicaseFeatures:
	}
	return ""
}

// Caps is f as its text is drawn: its capitals turned into the
// features of its face that draw them, when the face has them, else
// left for SmallCaps to synthesise.
func (s *Set) Caps(f style.Font, text string) style.Font {
	if f.Caps == style.NormalCaps {
		return f
	}
	c := s.Face(f)
	if ch := s.chained(f); ch != nil {
		c = ch.first()
	}
	feats := capsFeatures(c, f, shape.Script(text))
	switch {
	case feats == "":
		return f
	case f.Caps == style.Unicase && feats != tagUnic && f.Synthesis&style.NoSyntheticSmallCaps != 0:
		// Nothing made up: the lowercase takes the face's small capitals,
		// as Chrome draws it.
		feats = tagSmcp
	case f.Caps == style.Unicase && feats != tagUnic:
		// What is not lowercase takes them, the rest as it is (SmallCaps).
		f.Caps = style.UnicaseFeatures
		return f
	}
	f.Caps = style.NormalCaps
	if strings.Contains(f.Features, "|") {
		feats += "," + f.Features
	} else {
		feats += "|" + f.Features
	}
	f.Features = feats
	return f
}
