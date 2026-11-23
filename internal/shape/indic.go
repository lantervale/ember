package shape

import (
	"sort"
	"unicode"
)

//go:generate go run ./gen -ucd https://www.unicode.org/Public/17.0.0/ucd -msuse https://raw.githubusercontent.com/harfbuzz/harfbuzz/14.2.1/src/ms-use

// The categories HarfBuzz's Indic, Khmer and Myanmar shapers give
// characters (gen-indic-table.py); a nukta is Myanmar's dot below, a
// placeholder its generic base and a vowel its independent vowel.
const (
	iX uint8 = iota
	iC
	iV
	iN
	iH
	iZWNJ
	iZWJ
	iM
	iSM
	iA
	iPLACEHOLDER
	iDOTTEDCIRCLE
	iRS
	iMPst
	iRepha
	iRa
	iCM
	iSymbol
	iCS
	iSMPst
	iVAbv
	iVBlw
	iVPre
	iVPst
	iRobatic
	iXgroup
	iYgroup
	iAs
	iMH
	iMR
	iMW
	iMY
	iPT
	iVS
	iML
)

// The positions in a syllable the Indic shapers sort its characters by.
const (
	posStart uint8 = iota
	posRaToBecomeReph
	posPreM
	posPreC
	posBaseC
	posAfterMain
	posAboveC
	posBeforeSub
	posBelowC
	posAfterSub
	posBeforePost
	posPostC
	posAfterPost
	posSMVD
	posEnd
)

type indicRange struct {
	lo, hi   rune
	cat, pos uint8
}

// indicCategory is r's category and position for the Indic shapers.
func indicCategory(r rune) (uint8, uint8) {
	i := sort.Search(len(indicCategories), func(i int) bool { return indicCategories[i].hi >= r })
	if i < len(indicCategories) && indicCategories[i].lo <= r {
		return indicCategories[i].cat, indicCategories[i].pos
	}
	return iX, posEnd
}

// indicCategories gives each glyph its character's Indic category and
// position.
func (b *buffer) indicCategories() {
	for i := range b.glyphs {
		g := &b.glyphs[i]
		g.cat, g.pos = indicCategory(g.Char)
	}
}

// Syllables the Indic machine finds.
const (
	indicConsonantSyllable uint8 = iota
	indicVowelSyllable
	indicStandaloneCluster
	indicSymbolCluster
	indicBrokenCluster
	indicNonIndicCluster
)

//nolint:gochecknoglobals,goconst // a machine compiled once, its grammar as HarfBuzz writes it
var indicMachine = newMachine(map[string]uint8{
	"X": iX, "C": iC, "V": iV, "N": iN, "H": iH, "ZWNJ": iZWNJ, "ZWJ": iZWJ, "M": iM, "SM": iSM, "A": iA, "VD": iA,
	"PLACEHOLDER": iPLACEHOLDER, "DOTTEDCIRCLE": iDOTTEDCIRCLE, "RS": iRS, "MPst": iMPst, "Repha": iRepha,
	"Ra": iRa, "CM": iCM, "Symbol": iSymbol, "CS": iCS, "SMPst": iSMPst,
}, [][2]string{
	{"c", "(C | Ra)"},
	{"n", "((ZWNJ?.RS)? (N.N?)?)"},
	{"z", "ZWJ|ZWNJ"},
	{"reph", "(Ra H | Repha)"},
	{"sm", "SM | SMPst"},
	{"cn", "c.ZWJ?.n?"},
	{"symbol", "Symbol.N?"},
	{"matra_group", "z*.(M | sm? MPst).N?.H?"},
	{"syllable_tail", "(z?.sm.sm?.ZWNJ?)? (A | VD)*"},
	{"halant_group", "(z?.H.(ZWJ.N?)?)"},
	{"final_halant_group", "halant_group | H.ZWNJ"},
	{"medial_group", "CM?"},
	{"halant_or_matra_group", "(final_halant_group | matra_group*)"},
	{"complex_syllable_tail", "(halant_group.cn)* medial_group halant_or_matra_group syllable_tail"},
	{"consonant_syllable", "(Repha|CS)? cn complex_syllable_tail"},
	{"vowel_syllable", "reph? V.n? (ZWJ | complex_syllable_tail)"},
	{"standalone_cluster", "((Repha|CS)? PLACEHOLDER | reph? DOTTEDCIRCLE).n? complex_syllable_tail"},
	{"symbol_cluster", "symbol syllable_tail"},
	{"broken_cluster", "reph? n? complex_syllable_tail"},
}, []rule{
	{"consonant_syllable", indicConsonantSyllable},
	{"vowel_syllable", indicVowelSyllable},
	{"standalone_cluster", indicStandaloneCluster},
	{"symbol_cluster", indicSymbolCluster},
	{"SMPst", indicNonIndicCluster},
	{"broken_cluster", indicBrokenCluster},
	{"any", indicNonIndicCluster},
})

// Where a script's reph goes, and how it is written.
const (
	rephImplicit = iota // Ra, Halant
	rephExplicit        // Ra, Halant, ZWJ
	rephLogRepha        // a repha character
)

// indicConfig is what HarfBuzz's Indic shaper knows of a script.
type indicConfig struct {
	script   Tag
	oldSpec  bool
	virama   rune
	rephPos  uint8
	rephMode uint8
	// blwfPost is whether below forms apply only after the base.
	blwfPost bool
}

//nolint:gochecknoglobals // a fixed table
var indicConfigs = [...]indicConfig{
	{0, false, 0, posBeforePost, rephImplicit, false},
	{MakeTag("deva"), true, 0x094D, posBeforePost, rephImplicit, false},
	{MakeTag("beng"), true, 0x09CD, posAfterSub, rephImplicit, false},
	{MakeTag("guru"), true, 0x0A4D, posBeforeSub, rephImplicit, false},
	{MakeTag("gujr"), true, 0x0ACD, posBeforePost, rephImplicit, false},
	{MakeTag("orya"), true, 0x0B4D, posAfterMain, rephImplicit, false},
	{MakeTag("taml"), true, 0x0BCD, posAfterPost, rephImplicit, false},
	{MakeTag("telu"), true, 0x0C4D, posAfterPost, rephExplicit, true},
	{MakeTag("knda"), true, 0x0CCD, posAfterPost, rephImplicit, true},
	{MakeTag("mlym"), true, 0x0D4D, posAfterMain, rephLogRepha, false},
}

// The Indic features: the basic ones apply one after the other after
// the first reordering, the others together after the last.
//
//nolint:gochecknoglobals,goconst // a fixed table
var indicFeatures = [...]struct {
	tag    string
	global bool
	flags  uint8
}{
	{"nukt", true, manualJoiners | perSyllable},
	{"akhn", true, manualJoiners | perSyllable},
	{"rphf", false, manualJoiners | perSyllable},
	{"rkrf", true, manualJoiners | perSyllable},
	{"pref", false, manualJoiners | perSyllable},
	{"blwf", false, manualJoiners | perSyllable},
	{"abvf", false, manualJoiners | perSyllable},
	{"half", false, manualJoiners | perSyllable},
	{"pstf", false, manualJoiners | perSyllable},
	{"vatu", true, manualJoiners | perSyllable},
	{"cjct", true, manualJoiners | perSyllable},
	{"init", false, manualJoiners | perSyllable},
	{"pres", true, manualJoiners | perSyllable},
	{"abvs", true, manualJoiners | perSyllable},
	{"blws", true, manualJoiners | perSyllable},
	{"psts", true, manualJoiners | perSyllable},
	{"haln", true, manualJoiners | perSyllable},
}

const (
	indicRphf = 2
	indicPref = 4
	indicBlwf = 5
	indicAbvf = 6
	indicHalf = 7
	indicPstf = 8
	indicInit = 11
	// indicBasic is how many of the features are basic.
	indicBasic = 11
)

func collectIndic(c *collector) {
	c.pause(setupIndicSyllables)
	c.add(true, perSyllable, "locl", "ccmp")
	c.pause(initialIndicReordering)
	for _, f := range indicFeatures[:indicBasic] {
		c.add(f.global, f.flags, f.tag)
		c.pause(nil)
	}
	c.pause(finalIndicReordering)
	for _, f := range indicFeatures[indicBasic:] {
		c.add(f.global, f.flags, f.tag)
	}
}

// indicPlan is what the Indic shaper keeps of a plan.
type indicPlan struct {
	config  indicConfig
	oldSpec bool
	// zeroContext is whether the would-substitute tests of the features
	// below leave out rules that look round the glyphs.
	zeroContext                  bool
	rphf, pref, blwf, pstf, vatu []*lookup
	masks                        [len(indicFeatures)]uint32
	virama                       uint16
	// positions are the positions of consonants by their glyphs.
	positions map[uint16]uint8
}

func newIndicPlan(p *Plan) *indicPlan {
	ip := &indicPlan{config: indicConfigs[0], positions: map[uint16]uint8{}}
	for _, c := range indicConfigs[1:] {
		if c.script == p.script {
			ip.config = c
		}
	}
	ip.oldSpec = ip.config.oldSpec && p.chosen&0xff != '2'
	ip.zeroContext = !ip.oldSpec && p.script != MakeTag("mlym")
	ip.rphf = p.stageLookups(MakeTag("rphf"))
	ip.pref = p.stageLookups(MakeTag("pref"))
	ip.blwf = p.stageLookups(MakeTag("blwf"))
	ip.pstf = p.stageLookups(MakeTag("pstf"))
	ip.vatu = p.stageLookups(MakeTag("vatu"))
	for i, f := range indicFeatures {
		if !f.global {
			ip.masks[i] = p.masks[MakeTag(f.tag)]
		}
	}
	if ip.config.virama != 0 {
		ip.virama = p.font.nominal(ip.config.virama)
	}
	return ip
}

func (ip *indicPlan) would(f *Font, lks []*lookup, glyphs ...uint16) bool {
	return f.wouldSubstitute(lks, glyphs, ip.zeroContext)
}

// consonantPosition is where consonant sits when it follows a virama,
// as the font's below, post-base and pre-base forms say.
func (ip *indicPlan) consonantPosition(f *Font, consonant uint16) uint8 {
	if pos, ok := ip.positions[consonant]; ok {
		return pos
	}
	v := ip.virama
	pos := posBaseC
	switch {
	case ip.would(f, ip.blwf, v, consonant) || ip.would(f, ip.blwf, consonant, v) ||
		ip.would(f, ip.vatu, v, consonant) || ip.would(f, ip.vatu, consonant, v):
		pos = posBelowC
	case ip.would(f, ip.pstf, v, consonant) || ip.would(f, ip.pstf, consonant, v),
		ip.would(f, ip.pref, v, consonant) || ip.would(f, ip.pref, consonant, v):
		pos = posPostC
	}
	ip.positions[consonant] = pos
	return pos
}

// oneOf reports whether g, unless a ligature formed it, is of one of
// cats.
func oneOf(g *Glyph, cats ...uint8) bool {
	if g.props&formed != 0 {
		return false
	}
	for _, c := range cats {
		if g.cat == c {
			return true
		}
	}
	return false
}

func isConsonant(g *Glyph) bool {
	return oneOf(g, iC, iCS, iRa, iCM, iV, iPLACEHOLDER, iDOTTEDCIRCLE)
}

func isJoiner(g *Glyph) bool { return oneOf(g, iZWJ, iZWNJ) }

func isHalant(g *Glyph) bool { return oneOf(g, iH) }

// formedOnce reports whether a ligature formed g, not a multiple
// substitution after it.
func formedOnce(g *Glyph) bool { return g.props&(formed|multiplied) == formed }

func setupIndicSyllables(_ *Plan, b *buffer) {
	cats := make([]uint8, len(b.glyphs))
	for i, g := range b.glyphs {
		cats[i] = g.cat
	}
	b.syllables(indicMachine, cats, nil)
}

func initialIndicReordering(p *Plan, b *buffer) {
	ip := p.indic
	if ip.virama != 0 {
		for i := range b.glyphs {
			if g := &b.glyphs[i]; g.pos == posBaseC {
				g.pos = ip.consonantPosition(p.font, g.ID)
			}
		}
	}
	b.insertDottedCircles(indicBrokenCluster, iDOTTEDCIRCLE, posEnd, int(iRepha))
	b.eachSyllable(func(start, end int) {
		switch b.glyphs[start].syllable & 0x0F {
		case indicVowelSyllable, indicConsonantSyllable, indicBrokenCluster, indicStandaloneCluster:
			b.reorderIndicSyllable(p, start, end)
		}
	})
}

// reorderIndicSyllable is the first reordering of a syllable: its base
// found, a reph marked, its matras and the consonants after its base
// sorted by position, and each glyph given the masks of the features
// that form it.
//
//nolint:gocognit,gocyclo,cyclop,funlen,maintidx // HarfBuzz's steps, kept in their order
func (b *buffer) reorderIndicSyllable(p *Plan, start, end int) {
	ip := p.indic
	g := b.glyphs
	if p.script == MakeTag("knda") && start+3 <= end && oneOf(&g[start], iRa) && oneOf(&g[start+1], iH) &&
		oneOf(&g[start+2], iZWJ) {
		b.mergeClusters(start+1, start+3)
		g[start+1], g[start+2] = g[start+2], g[start+1]
	}
	base := end
	hasReph := false
	limit := start
	mode := ip.config.rephMode
	if ip.masks[indicRphf] != 0 && start+3 <= end &&
		(mode == rephImplicit && !isJoiner(&g[start+2]) || mode == rephExplicit && g[start+2].cat == iZWJ) {
		third := uint16(0)
		if mode == rephExplicit {
			third = g[start+2].ID
		}
		if ip.would(p.font, ip.rphf, g[start].ID, g[start+1].ID) ||
			mode == rephExplicit && ip.would(p.font, ip.rphf, g[start].ID, g[start+1].ID, third) {
			limit += 2
			for limit < end && isJoiner(&g[limit]) {
				limit++
			}
			base, hasReph = start, true
		}
	} else if mode == rephLogRepha && g[start].cat == iRepha {
		limit++
		for limit < end && isJoiner(&g[limit]) {
			limit++
		}
		base, hasReph = start, true
	}
	seenBelow := false
	for i := end - 1; ; i-- {
		if isConsonant(&g[i]) {
			if g[i].pos != posBelowC && (g[i].pos != posPostC || seenBelow) {
				base = i
				break
			}
			if g[i].pos == posBelowC {
				seenBelow = true
			}
			base = i
		} else if start < i && g[i].cat == iZWJ && g[i-1].cat == iH {
			break
		}
		if i <= limit {
			break
		}
	}
	if hasReph && base == start && limit-base <= 2 {
		hasReph = false
	}
	for i := start; i < base; i++ {
		g[i].pos = min(posPreC, g[i].pos)
	}
	if base < end {
		g[base].pos = posBaseC
	}
	if hasReph {
		g[start].pos = posRaToBecomeReph
	}
	if ip.oldSpec {
		noDouble := p.script == MakeTag("knda")
		for i := base + 1; i < end; i++ {
			if g[i].cat != iH {
				continue
			}
			j := end - 1
			for ; j > i; j-- {
				if isConsonant(&g[j]) || noDouble && g[j].cat == iH {
					break
				}
			}
			if g[j].cat != iH && j > i {
				b.move(i, j)
			}
			break
		}
	}
	last := posStart
	for i := start; i < end; i++ {
		switch {
		case g[i].cat == iZWJ || g[i].cat == iZWNJ || g[i].cat == iN || g[i].cat == iRS || g[i].cat == iCM ||
			g[i].cat == iH:
			g[i].pos = last
			if g[i].cat == iH && g[i].pos == posPreM {
				for j := i; j > start; j-- {
					if g[j-1].pos != posPreM {
						g[i].pos = g[j-1].pos
						break
					}
				}
			}
		case g[i].pos != posSMVD:
			if g[i].cat == iMPst && i > start && g[i-1].cat == iSM {
				g[i-1].pos = g[i].pos
			}
			last = g[i].pos
		}
	}
	{
		last := base
		for i := base + 1; i < end; i++ {
			if isConsonant(&g[i]) {
				for j := last + 1; j < i; j++ {
					if g[j].pos < posSMVD {
						g[j].pos = g[i].pos
					}
				}
				last = i
			} else if g[i].cat == iM || g[i].cat == iMPst {
				last = i
			}
		}
	}
	syllable := g[start].syllable
	for i := start; i < end; i++ {
		g[i].syllable = uint8(min(i-start, 255)) //nolint:gosec // at most 255
	}
	b.stablePositions(start, end)
	first, lastLeft := end, end
	base = end
	for i := start; i < end; i++ {
		if g[i].pos == posBaseC {
			base = i
			break
		}
		if g[i].pos == posPreM {
			if first == end {
				first = i
			}
			lastLeft = i
		}
	}
	if first < lastLeft {
		b.reverseRange(first, lastLeft+1)
		i := first
		for j := i; j <= lastLeft; j++ {
			if g[j].cat == iM || g[j].cat == iMPst {
				b.reverseRange(i, j+1)
				i = j + 1
			}
		}
	}
	if ip.oldSpec || end-start > 127 {
		b.mergeClusters(base, end)
	} else {
		for i := base; i < end; i++ {
			if g[i].syllable == 255 {
				continue
			}
			lo, hi := i, i
			j := start + int(g[i].syllable)
			for j != i {
				lo, hi = min(lo, j), max(hi, j)
				next := start + int(g[j].syllable)
				g[j].syllable = 255
				j = next
			}
			b.mergeClusters(max(base, lo), hi+1)
		}
	}
	for i := start; i < end; i++ {
		g[i].syllable = syllable
	}
	for i := start; i < end && g[i].pos == posRaToBecomeReph; i++ {
		g[i].mask |= ip.masks[indicRphf]
	}
	mask := ip.masks[indicHalf]
	if !ip.oldSpec && !ip.config.blwfPost {
		mask |= ip.masks[indicBlwf]
	}
	for i := start; i < base; i++ {
		g[i].mask |= mask
	}
	mask = ip.masks[indicBlwf] | ip.masks[indicAbvf] | ip.masks[indicPstf]
	for i := base + 1; i < end; i++ {
		g[i].mask |= mask
	}
	if ip.oldSpec && p.script == MakeTag("deva") {
		for i := start; i+1 < base; i++ {
			if g[i].cat == iRa && g[i+1].cat == iH && (i+2 == base || g[i+2].cat != iZWJ) {
				g[i].mask |= ip.masks[indicBlwf]
				g[i+1].mask |= ip.masks[indicBlwf]
			}
		}
	}
	if ip.masks[indicPref] != 0 && base+2 < end {
		for i := base + 1; i+1 < end; i++ {
			if ip.would(p.font, ip.pref, g[i].ID, g[i+1].ID) {
				g[i].mask |= ip.masks[indicPref]
				g[i+1].mask |= ip.masks[indicPref]
				break
			}
		}
	}
	for i := start + 1; i < end; i++ {
		if !isJoiner(&g[i]) {
			continue
		}
		nonJoiner := g[i].cat == iZWNJ
		j := i
		for {
			j--
			if nonJoiner {
				g[j].mask &^= ip.masks[indicHalf]
			}
			if j <= start || isConsonant(&g[j]) {
				break
			}
		}
	}
}

// stablePositions sorts the glyphs from start to end stably by their
// positions, clusters left as they are.
func (b *buffer) stablePositions(start, end int) {
	s := b.glyphs[start:end]
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j-1].pos > s[j].pos; j-- {
			s[j-1], s[j] = s[j], s[j-1]
		}
	}
}

func finalIndicReordering(p *Plan, b *buffer) {
	if len(b.glyphs) == 0 {
		return
	}
	b.eachSyllable(func(start, end int) { b.finalIndicSyllable(p, start, end) })
}

// finalIndicSyllable is the final reordering of a syllable, once the
// basic features formed what they form: a pre-base matra moved after
// the half forms, a reph to its place, a pre-base reordering consonant
// before the base, and init on a matra starting a word.
//
//nolint:gocognit,gocyclo,cyclop,funlen,maintidx,nestif // HarfBuzz's steps, kept in their order
func (b *buffer) finalIndicSyllable(p *Plan, start, end int) {
	ip := p.indic
	g := b.glyphs
	if ip.virama != 0 {
		for i := start; i < end; i++ {
			if g[i].ID == ip.virama && g[i].props&(formed|multiplied) == formed|multiplied {
				g[i].cat = iH
				g[i].props &^= formed | multiplied
			}
		}
	}
	tryPref := ip.masks[indicPref] != 0
	tamilLike := p.script == MakeTag("mlym") || p.script == MakeTag("taml")
	base := start
	for ; base < end; base++ {
		if g[base].pos < posBaseC {
			continue
		}
		if tryPref && base+1 < end {
			for i := base + 1; i < end; i++ {
				if g[i].mask&ip.masks[indicPref] == 0 {
					continue
				}
				if g[i].props&substituted == 0 || !formedOnce(&g[i]) {
					base = i
					for base < end && isHalant(&g[base]) {
						base++
					}
					if base < end {
						g[base].pos = posBaseC
					}
					tryPref = false
				}
				break
			}
			if base == end {
				break
			}
		}
		if p.script == MakeTag("mlym") {
			for i := base + 1; i < end; i++ {
				for i < end && isJoiner(&g[i]) {
					i++
				}
				if i == end || !isHalant(&g[i]) {
					break
				}
				i++
				for i < end && isJoiner(&g[i]) {
					i++
				}
				if i < end && isConsonant(&g[i]) && g[i].pos == posBelowC {
					base = i
					g[base].pos = posBaseC
				}
			}
		}
		if start < base && g[base].pos > posBaseC {
			base--
		}
		break
	}
	if base == end && start < base && oneOf(&g[base-1], iZWJ) {
		base--
	}
	if base < end {
		for start < base && oneOf(&g[base], iN, iH) {
			base--
		}
	}
	if start+1 < end && start < base {
		newPos := base - 1
		if base == end {
			newPos = base - 2
		}
		for !tamilLike {
			for newPos > start && !oneOf(&g[newPos], iM, iMPst, iH) {
				newPos--
			}
			if isHalant(&g[newPos]) && g[newPos].pos != posPreM {
				if newPos+1 < end && g[newPos+1].cat == iZWJ && newPos > start {
					newPos--
					continue
				}
			} else {
				newPos = start
			}
			break
		}
		if start < newPos && g[newPos].pos != posPreM {
			for i := newPos; i > start; i-- {
				if g[i-1].pos != posPreM {
					continue
				}
				old := i - 1
				if old < base && base <= newPos {
					base--
				}
				b.move(old, newPos)
				b.mergeClusters(newPos, min(end, base+1))
				newPos--
			}
		} else {
			for i := start; i < base; i++ {
				if g[i].pos == posPreM {
					b.mergeClusters(i, min(end, base+1))
					break
				}
			}
		}
	}
	if start+1 < end && g[start].pos == posRaToBecomeReph && (g[start].cat == iRepha) != formedOnce(&g[start]) {
		newPos := b.rephTarget(ip, start, end, base)
		b.mergeClusters(start, newPos+1)
		b.move(start, newPos)
		if start < base && base <= newPos {
			base--
		}
	}
	if tryPref && base+1 < end {
		for i := base + 1; i < end; i++ {
			if g[i].mask&ip.masks[indicPref] == 0 {
				continue
			}
			if formedOnce(&g[i]) {
				newPos := base
				if !tamilLike {
					for newPos > start && !oneOf(&g[newPos-1], iM, iMPst, iH) {
						newPos--
					}
				}
				if newPos > start && isHalant(&g[newPos-1]) && newPos < end && isJoiner(&g[newPos]) {
					newPos++
				}
				b.mergeClusters(newPos, i+1)
				b.move(i, newPos)
			}
			break
		}
	}
	if g[start].pos == posPreM && (start == 0 || startsWord(g[start-1].Char)) {
		g[start].mask |= ip.masks[indicInit]
	}
}

// startsWord reports whether what follows r starts a word: r is a
// control, a space, a number, punctuation or a symbol.
func startsWord(r rune) bool {
	return unicode.In(r, unicode.Cc, unicode.Z, unicode.N, unicode.P, unicode.S)
}

// rephTarget is where the reph at start of the syllable to end moves
// to, its base at base.
//
//nolint:gocognit,nestif // HarfBuzz's steps, in their order
func (b *buffer) rephTarget(ip *indicPlan, start, end, base int) int {
	g := b.glyphs
	afterHalant := func() (int, bool) {
		at := start + 1
		for at < base && !isHalant(&g[at]) {
			at++
		}
		if at < base && isHalant(&g[at]) {
			if at+1 < base && isJoiner(&g[at+1]) {
				at++
			}
			return at, true
		}
		return 0, false
	}
	pos := ip.config.rephPos
	if pos != posAfterPost {
		if at, ok := afterHalant(); ok {
			return at
		}
		if pos == posAfterMain {
			at := base
			for at+1 < end && g[at+1].pos <= posAfterMain {
				at++
			}
			if at < end {
				return at
			}
		}
		if pos == posAfterSub {
			at := base
			for at+1 < end && g[at+1].pos != posPostC && g[at+1].pos != posAfterPost && g[at+1].pos != posSMVD {
				at++
			}
			if at < end {
				return at
			}
		}
	}
	if at, ok := afterHalant(); ok {
		return at
	}
	at := end - 1
	for at > start && g[at].pos == posSMVD {
		at--
	}
	if isHalant(&g[at]) {
		for i := base + 1; i < at; i++ {
			if g[i].cat == iM || g[i].cat == iMPst {
				at--
			}
		}
	}
	return at
}
