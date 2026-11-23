package fonts

import (
	"fmt"
	"strings"

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

// shapedRun is a run shaped: its width in its face's units at the
// default instance, and the glyphs it is drawn with.
type shapedRun struct {
	width  int32
	glyphs []uint16
}

// shaped is the advance of run, glyphs of c, in c's units at instance
// at of a variable font (nil for its default): shaped with the
// features, in script, when its layout does anything to them, else the
// sum of their advances. GPOS's values stay the default instance's.
func (s *Set) shaped(c *Face, script shape.Tag, features string, run []shape.Glyph, at varfont.Coords) float32 {
	p := c.layout().Plan(script, c.joined(features))
	if !p.Touches(run) {
		if at != nil {
			var w float32
			for _, g := range run {
				w += c.web.vf.Advance(g.ID, at)
			}
			return w
		}
		ppem := fixed.I(int(c.upem))
		var w fixed.Int26_6
		for _, g := range run {
			if a, err := c.font.GlyphAdvance(&s.buf, sfnt.GlyphIndex(g.ID), ppem, font.HintingNone); err == nil {
				w += a
			}
		}
		return float32(w >> 6)
	}
	ids := s.ids[:0]
	for _, g := range run {
		ids = append(ids, byte(g.ID>>8), byte(g.ID&0xff))
	}
	s.ids = ids
	r, ok := s.shapedRuns[p][string(ids)]
	if !ok {
		for _, g := range p.Shape(run) {
			r.width += g.Advance
			r.glyphs = append(r.glyphs, g.ID)
		}
		if s.shapedRuns == nil || s.shapedCount >= maxShaped {
			s.shapedRuns, s.shapedCount = map[*shape.Plan]map[string]shapedRun{}, 0
		}
		if s.shapedRuns[p] == nil {
			s.shapedRuns[p] = map[string]shapedRun{}
		}
		s.shapedRuns[p][string(ids)] = r
		s.shapedCount++
	}
	w := float32(r.width)
	if at != nil {
		for _, g := range r.glyphs {
			w += c.web.vf.Advance(g, at) - c.web.vf.Advance(g, nil)
		}
	}
	return w
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
	return c.layout().Plan(shape.MakeTag("latn"), c.joined(f.Features)).Mentions(uint16(g))
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
		if has("smcp") {
			return "smcp"
		}
	case style.AllSmallCaps:
		if has("smcp", "c2sc") {
			return smallCapitals
		}
	case style.PetiteCaps:
		if has("pcap") {
			return "pcap"
		}
		if has("smcp") {
			return "smcp"
		}
	case style.AllPetiteCaps:
		if has("pcap", "c2pc") {
			return "pcap,c2pc"
		}
		if has("smcp", "c2sc") {
			return smallCapitals
		}
	case style.Unicase:
		if has("unic") {
			return "unic"
		}
		if has("smcp", "c2sc") {
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
	case f.Caps == style.Unicase && feats != "unic" && f.Synthesis&style.NoSyntheticSmallCaps != 0:
		// Nothing made up: the lowercase takes the face's small capitals,
		// as Chrome draws it.
		feats = "smcp"
	case f.Caps == style.Unicase && feats != "unic":
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
