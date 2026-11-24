package shape

import (
	"slices"
	"strconv"
	"strings"

	"github.com/lantervale/ember/internal/varfont"
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
	// arabic are the lookups of HarfBuzz's Arabic fallback, applied after
	// rlig's stage, for Arabic in a face without its joined forms.
	arabic []planned
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

	script Tag
	shaper shaper
	// chosen is the script tag of GSUB's language system, 0 for none.
	chosen Tag
	// requests are the features asked for, masks their masks, and pauses
	// what the shaper does after the GSUB stages they end.
	requests []request
	masks    map[Tag]uint32
	pauses   map[int]pause
	// positions is whether the font's GPOS positions the run (Hebrew's
	// only when GPOS has the Hebrew script), found whether GSUB has the
	// run's script, and marks whether the language systems have a mark
	// feature.
	positions, found, marks bool
	indic                   *indicPlan
	khmer                   *khmerPlan
	use                     *usePlan

	// vf and coords are the variable font and the instance its runs are
	// shaped at, coords nil for the default instance.
	vf     *varfont.Font
	coords varfont.Coords
}

type planKey struct {
	script   Tag
	features string
	coords   string
}

// Plan is the plan for runs in script with features, HarfBuzz's written
// comma separated (ParseFeatures) after those the script's shaper has
// on, a feature written again overriding it.
func (f *Font) Plan(script Tag, features string) *Plan { return f.PlanAt(script, features, nil, nil) }

// PlanAt is Plan for runs drawn at instance at of variable font vf, the
// font's file: the features its feature variations choose there, GPOS's
// values moved by their deltas and the glyphs' advances the instance's.
func (f *Font) PlanAt(script Tag, features string, vf *varfont.Font, at varfont.Coords) *Plan {
	f.mu.Lock()
	defer f.mu.Unlock()
	if vf == nil {
		at = nil
	}
	key := planKey{script, features, coordsKey(at)}
	if p, ok := f.plans[key]; ok {
		return p
	}
	chosen, gsubLS := f.gsub.langSys(script)
	gposChosen, gposLS := f.gpos.langSys(script)
	sh := shaperFor(script, chosen)
	asked, pauses := requests(script, sh, ParseFeatures(features))
	m := masks(asked)
	marks := f.gsub.inLangSys(gsubLS, mark) || f.gpos.inLangSys(gposLS, mark)
	fallback := sh == arabicShaper && f.arabicFallback(script, gsubLS, gposLS)
	positions := f.gpos.scripts != nil && (sh != hebrewShaper || gposChosen == hebr)
	if !positions {
		gposLS = nil
	}
	p := &Plan{font: f, gsub: f.gsub.plan(gsubLS, f.gsub.substitution(at), asked, m),
		gpos: f.gpos.plan(gposLS, f.gpos.substitution(at), asked, m),
		rtl:  RightToLeft(script), kerns: m[kern] != 0, script: script, shaper: sh, chosen: chosen,
		requests: asked, masks: m, pauses: pauses, positions: positions, marks: marks,
		found: slices.Contains(candidates(script), chosen), vf: vf, coords: at}
	if s := scriptByTag(script); sh == arabicShaper || sh == useShaper && s != nil && s.joins {
		p.joins = true
		for i, t := range formTags {
			p.forms[i] = m[t]
		}
	}
	if fallback {
		p.arabic = f.arabicPlanned(m)
	}
	switch sh { //nolint:exhaustive // the shapers that do something here
	case indicShaper:
		p.indic = newIndicPlan(p)
	case khmerShaper:
		p.khmer = newKhmerPlan(p)
	case useShaper:
		p.use = &usePlan{rphf: m[MakeTag("rphf")]}
	}
	for _, l := range p.lookups() {
		p.first.union(l.lk.first)
	}
	f.plans[key] = p
	return p
}

// lookups are all of p's lookups: GSUB's, the Arabic fallback's and
// GPOS's.
func (p *Plan) lookups() []planned {
	return slices.Concat(p.gsub, p.arabic, p.gpos)
}

// stageOf is the GSUB stage feature t applies at, -1 when the plan has
// no lookup of it.
func (p *Plan) stageOf(t Tag) int {
	for _, r := range p.requests {
		if r.tag == t && p.masks[t] != 0 {
			return r.stage
		}
	}
	return -1
}

// stageLookups are the GSUB lookups of the stage feature t applies at.
func (p *Plan) stageLookups(t Tag) []*lookup {
	s := p.stageOf(t)
	if s < 0 || !p.font.gsub.has(p.script, t) {
		return nil
	}
	var out []*lookup
	for _, l := range p.gsub {
		if l.stage == s {
			out = append(out, l.lk)
		}
	}
	return out
}

// Mentions reports whether one of p's lookups may start at glyph g,
// follow a glyph it starts at in a pair it adjusts, join it into a
// ligature or read it round the glyph it starts at in a context: whether
// g, a space, ties the words on either side of it.
func (p *Plan) Mentions(g uint16) bool {
	if p.first.has(g) || p.KernTable() {
		return true
	}
	p.font.mu.Lock()
	defer p.font.mu.Unlock()
	if p.mentioned == nil {
		p.mentioned = glyphSet{}
		for _, l := range p.lookups() {
			for _, s := range l.lk.subs {
				l.lk.seconds(s, &p.mentioned)
			}
		}
	}
	return p.mentioned.has(g)
}

// seconds adds the glyphs after the first that subtable s of lk may
// take, as the second of a pair or a ligature's component, to set, and
// those a context reads besides the glyph it starts at; of a context by
// classes, the glyphs its classes name.
func (lk *lookup) seconds(s data, set *glyphSet) {
	switch {
	case lk.kind == context || lk.kind == chain:
		contextGlyphs(lk.kind == chain, s, set)
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

// contextGlyphs adds to set the glyphs context subtable s, chained or
// not, reads besides the first of its input.
func contextGlyphs(chained bool, s data, set *glyphSet) {
	switch format := s.u16(0); {
	case format == 2 && chained:
		for _, at := range []int{4, 6, 8} {
			s.sub(at).eachClassed(set.add)
		}
	case format == 2:
		s.sub(4).eachClassed(set.add)
	case format == 1:
		for k := range int(s.u16(4)) {
			rules := s.sub(6 + 2*k)
			for n := range int(rules.u16(0)) {
				ruleGlyphs(chained, rules.sub(2+2*n), set)
			}
		}
	case format == 3 && chained:
		back := int(s.u16(2))
		in := 4 + 2*back
		count := int(s.u16(in))
		ahead := in + 2 + 2*count
		for c := range back {
			s.sub(4 + 2*c).eachCovered(set.add)
		}
		for c := 1; c < count; c++ {
			s.sub(in + 2 + 2*c).eachCovered(set.add)
		}
		for c := range int(s.u16(ahead)) {
			s.sub(ahead + 2 + 2*c).eachCovered(set.add)
		}
	case format == 3:
		for c := 1; c < int(s.u16(2)); c++ {
			s.sub(6 + 2*c).eachCovered(set.add)
		}
	}
}

// ruleGlyphs adds to set the glyphs a rule of a context by glyphs reads:
// its input's after the first, and a chained one's backtrack and
// lookahead.
func ruleGlyphs(chained bool, rule data, set *glyphSet) {
	if !chained {
		for c := 1; c < int(rule.u16(0)); c++ {
			set.add(rule.u16(4 + 2*(c-1)))
		}
		return
	}
	at := 0
	for part := range 3 {
		count, from := int(rule.u16(at)), 0
		if part == 1 {
			from = 1
		}
		for c := from; c < count; c++ {
			set.add(rule.u16(at + 2 + 2*(c-from)))
		}
		at += 2 + 2*(count-from)
	}
}

// Has reports whether the font has feature t for runs in script.
func (f *Font) Has(script Tag, t Tag) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.gsub.has(script, t) || f.gpos.has(script, t)
}

// langSys is the default language system of script in l, and the tag
// it is of: of script by its tags (candidates), of DFLT, or of Latin,
// the first l has.
func (l *layout) langSys(script Tag) (Tag, data) {
	n := int(l.scripts.u16(0))
	for _, want := range append(candidates(script), dflt, MakeTag("dflt"), latn) {
		for i := range n {
			if Tag(l.scripts.u32(2+6*i)) == want {
				s := l.scripts.sub(2 + 6*i + 4)
				return want, s.sub(0)
			}
		}
	}
	return 0, nil
}

// features calls fn with each feature language system ls has in l, its
// required one included, and its lookups' list, that of the feature
// subst puts in its place where it does.
func (l *layout) features(ls, subst data, fn func(tag Tag, f data, required bool)) {
	if ls == nil {
		return
	}
	each := func(i int, required bool) {
		if i >= int(l.feats.u16(0)) {
			return
		}
		fn(Tag(l.feats.u32(2+6*i)), l.feature(i, subst), required)
	}
	if req := ls.u16(2); req != 0xFFFF {
		each(int(req), true)
	}
	for k := range int(ls.u16(4)) {
		each(int(ls.u16(6+2*k)), false)
	}
}

func (l *layout) has(script Tag, t Tag) bool {
	_, ls := l.langSys(script)
	return l.inLangSys(ls, t)
}

// inLangSys reports whether language system ls of l has feature t.
func (l *layout) inLangSys(ls data, t Tag) bool {
	found := false
	l.features(ls, nil, func(tag Tag, _ data, _ bool) { found = found || tag == t })
	return found
}
