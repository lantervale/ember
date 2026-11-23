package shape

import (
	"slices"
	"strconv"
	"strings"
)

// Feature is an OpenType feature a run turns on (1, or the alternate to
// take, from 1) or off (0).
type Feature struct {
	Tag   Tag
	Value uint32
}

// ParseFeatures reads features written as HarfBuzz writes them, comma
// separated: "smcp", "-liga" (off), "+kern" (on), "salt=2". What it
// cannot read it leaves out; a '|' (Splice) separates as a comma does.
func ParseFeatures(s string) []Feature {
	var out []Feature
	for part := range strings.SplitSeq(strings.ReplaceAll(s, "|", ","), ",") {
		part = strings.TrimSpace(part)
		f := Feature{Value: 1}
		switch {
		case strings.HasPrefix(part, "-"):
			f.Value, part = 0, part[1:]
		case strings.HasPrefix(part, "+"):
			part = part[1:]
		}
		if name, v, ok := strings.Cut(part, "="); ok {
			n, err := strconv.ParseUint(v, 10, 32)
			if err != nil {
				continue
			}
			f.Value, part = uint32(n), name
		}
		if len(part) == 0 || len(part) > 4 {
			continue
		}
		f.Tag = MakeTag(part)
		out = append(out, f)
	}
	return out
}

// Splice puts a face's own features, its @font-face's, among a text's:
// where the text's have a '|', after font-variant's and before
// font-feature-settings', as Chrome orders them, else first.
func Splice(text, face string) string {
	if face == "" {
		return text
	}
	before, after, marked := strings.Cut(text, "|")
	if !marked {
		before, after = "", text
	}
	var parts []string
	for _, p := range []string{before, face, after} {
		if p != "" {
			parts = append(parts, p)
		}
	}
	return strings.Join(parts, ",")
}

// Plan is what shaping a run of a font in one script with some features
// applies: lookups in stages, each stage's in the order of the font's
// lists.
type Plan struct {
	font *Font
	gsub []planned
	gpos []planned
	// first are the glyphs one of its lookups may start at.
	first glyphSet
	// kerns is whether the run is kerned.
	kerns bool
	// rtl is whether the run is written right to left, and joins whether
	// its letters join, each glyph then marked with the mask of its form's
	// feature in forms.
	rtl, joins bool
	forms      [len(formTags)]uint32
	// mentioned are the glyphs its lookups take after the first, read
	// when Mentions is first asked.
	mentioned glyphSet
}

type planKey struct {
	script   Tag
	features string
}

// Plan is the plan for runs in script with features, HarfBuzz's written
// comma separated (ParseFeatures) after those the script's shaper has
// on, a feature written again overriding it.
func (f *Font) Plan(script Tag, features string) *Plan {
	f.mu.Lock()
	defer f.mu.Unlock()
	key := planKey{script, features}
	if p, ok := f.plans[key]; ok {
		return p
	}
	asked := requests(script, ParseFeatures(features))
	m := masks(asked)
	p := &Plan{font: f, gsub: f.gsub.plan(script, asked, m), gpos: f.gpos.plan(script, asked, m),
		rtl: RightToLeft(script), kerns: m[kern] != 0}
	if slices.Contains(joiningScripts, script) {
		p.joins = true
		for i, t := range formTags {
			p.forms[i] = m[t]
		}
	}
	for _, l := range append(slices.Clip(p.gsub), p.gpos...) {
		p.first.union(l.lk.first)
	}
	f.plans[key] = p
	return p
}

// Mentions reports whether one of p's lookups may start at glyph g,
// follow a glyph it starts at in a pair it adjusts, or join it into a
// ligature: whether g, a space, ties the words on either side of it.
func (p *Plan) Mentions(g uint16) bool {
	if p.first.has(g) || p.KernTable() {
		return true
	}
	p.font.mu.Lock()
	defer p.font.mu.Unlock()
	if p.mentioned == nil {
		p.mentioned = glyphSet{}
		for _, l := range append(slices.Clip(p.gsub), p.gpos...) {
			for _, s := range l.lk.subs {
				l.lk.seconds(s, &p.mentioned)
			}
		}
	}
	return p.mentioned.has(g)
}

// seconds adds the glyphs after the first that subtable s of lk may
// take, as the second of a pair or a ligature's component, to set.
func (lk *lookup) seconds(s data, set *glyphSet) {
	switch {
	case lk.kind == ligature && s.u16(0) == 1:
		for k := range int(s.u16(4)) {
			ls := s.sub(6 + 2*k)
			for n := range int(ls.u16(0)) {
				lig := ls.sub(2 + 2*n)
				for c := 1; c < int(lig.u16(2)); c++ {
					set.add(lig.u16(4 + 2*(c-1)))
				}
			}
		}
	case lk.kind == pair && s.u16(0) == 1:
		rec := 2 + valueSize(s.u16(4)) + valueSize(s.u16(6))
		for k := range int(s.u16(8)) {
			ps := s.sub(10 + 2*k)
			for n := range int(ps.u16(0)) {
				set.add(ps.u16(2 + n*rec))
			}
		}
	case lk.kind == pair && s.u16(0) == 2:
		classes := s.sub(10)
		switch classes.u16(0) {
		case 1:
			start := classes.u16(2)
			for i := range int(classes.u16(4)) {
				if classes.u16(6+2*i) != 0 {
					set.add(start + uint16(i))
				}
			}
		case 2:
			for i := range int(classes.u16(2)) {
				if classes.u16(4+6*i+4) == 0 {
					continue
				}
				for g := int(classes.u16(4 + 6*i)); g <= int(classes.u16(4+6*i+2)); g++ {
					set.add(uint16(g)) //nolint:gosec // a glyph of a 16-bit range
				}
			}
		}
	}
}

// Has reports whether the font has feature t for runs in script.
func (f *Font) Has(script Tag, t Tag) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.gsub.has(script, t) || f.gpos.has(script, t)
}

// langSys is the default language system of script in l: of script, of
// DFLT, or of Latin, the first l has.
func (l *layout) langSys(script Tag) data {
	n := int(l.scripts.u16(0))
	for _, want := range []Tag{script, dflt, MakeTag("dflt"), latn} {
		for i := range n {
			if Tag(l.scripts.u32(2+6*i)) == want {
				s := l.scripts.sub(2 + 6*i + 4)
				return s.sub(0)
			}
		}
	}
	return nil
}

// features calls fn with each feature script's language system has in
// l, its required one included, and its lookups' list.
func (l *layout) features(script Tag, fn func(tag Tag, f data, required bool)) {
	ls := l.langSys(script)
	if ls == nil {
		return
	}
	each := func(i int, required bool) {
		if i >= int(l.feats.u16(0)) {
			return
		}
		fn(Tag(l.feats.u32(2+6*i)), l.feats.sub(2+6*i+4), required)
	}
	if req := ls.u16(2); req != 0xFFFF {
		each(int(req), true)
	}
	for k := range int(ls.u16(4)) {
		each(int(ls.u16(6+2*k)), false)
	}
}

func (l *layout) has(script Tag, t Tag) bool {
	found := false
	l.features(script, func(tag Tag, _ data, _ bool) { found = found || tag == t })
	return found
}
