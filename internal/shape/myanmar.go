package shape

// Syllables the Myanmar machine finds.
const (
	myanmarConsonantSyllable uint8 = iota
	myanmarBrokenCluster
	myanmarNonMyanmarCluster
)

//nolint:gochecknoglobals,goconst // a machine compiled once, its grammar as HarfBuzz writes it
var myanmarMachine = newMachine(map[string]uint8{
	"C": iC, "IV": iV, "DB": iN, "H": iH, "ZWNJ": iZWNJ, "ZWJ": iZWJ, "SM": iSM, "GB": iPLACEHOLDER,
	"DOTTEDCIRCLE": iDOTTEDCIRCLE, "A": iA, "Ra": iRa, "CS": iCS, "SMPst": iSMPst, "VAbv": iVAbv, "VBlw": iVBlw,
	"VPre": iVPre, "VPst": iVPst, "As": iAs, "MH": iMH, "MR": iMR, "MW": iMW, "MY": iMY, "PT": iPT, "VS": iVS,
	"ML": iML,
}, [][2]string{
	{"j", "ZWJ|ZWNJ"},
	{"k", "(Ra As H)"},
	{"sm", "SM | SMPst"},
	{"c", "C|Ra"},
	{"medial_group", "MY? As? MR? ((MW MH? ML? | MH ML? | ML) As?)?"},
	{"main_vowel_group", "(VPre.VS?)* VAbv* VBlw* A* (DB As?)?"},
	{"post_vowel_group", "VPst MH? ML? As* VAbv* A* (DB As?)?"},
	{"tone_group", "sm | PT A* DB? As?"},
	{"complex_syllable_tail", "As* medial_group main_vowel_group post_vowel_group* tone_group* j?"},
	{"syllable_tail", "(H (c|IV).VS?)* (H | complex_syllable_tail)"},
	{"consonant_syllable", "(k|CS)? (c|IV|GB|DOTTEDCIRCLE).VS? syllable_tail"},
	{"broken_cluster", "k? VS? syllable_tail"},
}, []rule{
	{"consonant_syllable", myanmarConsonantSyllable},
	{"j | SMPst", myanmarNonMyanmarCluster},
	{"broken_cluster", myanmarBrokenCluster},
	{"any", myanmarNonMyanmarCluster},
})

//nolint:goconst // its features by their tags
func collectMyanmar(c *collector) {
	c.pause(setupMyanmarSyllables)
	c.add(true, perSyllable, "locl", "ccmp")
	c.pause(reorderMyanmar)
	for _, t := range []string{"rphf", "pref", "blwf", "pstf"} {
		c.add(true, manualZWJ|perSyllable, t)
		c.pause(nil)
	}
	c.pause(nil)
	c.add(true, manualZWJ, "pres", "abvs", "blws", "psts")
}

func setupMyanmarSyllables(_ *Plan, b *buffer) {
	cats := make([]uint8, len(b.glyphs))
	for i, g := range b.glyphs {
		cats[i] = g.cat
	}
	b.syllables(myanmarMachine, cats, nil)
}

func reorderMyanmar(_ *Plan, b *buffer) {
	b.insertDottedCircles(myanmarBrokenCluster, iDOTTEDCIRCLE, 0, -1)
	b.eachSyllable(func(start, end int) {
		if k := b.glyphs[start].syllable & 0x0F; k == myanmarConsonantSyllable || k == myanmarBrokenCluster {
			b.reorderMyanmarSyllable(start, end)
		}
	})
}

func isMyanmarConsonant(g *Glyph) bool {
	return oneOf(g, iC, iCS, iRa, iV, iPLACEHOLDER, iDOTTEDCIRCLE)
}

// reorderMyanmarSyllable sorts a syllable by position: a kinzi after
// its base, medial Ra and pre-base vowels before it.
//
//nolint:funlen // HarfBuzz's steps, in their order
func (b *buffer) reorderMyanmarSyllable(start, end int) {
	g := b.glyphs
	base := end
	hasReph := false
	limit := start
	if start+3 <= end && g[start].cat == iRa && g[start+1].cat == iAs && g[start+2].cat == iH {
		limit += 3
		base, hasReph = start, true
	}
	if !hasReph {
		base = limit
	}
	for i := limit; i < end; i++ {
		if isMyanmarConsonant(&g[i]) {
			base = i
			break
		}
	}
	i := start
	if hasReph {
		for ; i < start+3; i++ {
			g[i].pos = posAfterMain
		}
	}
	for ; i < base; i++ {
		g[i].pos = posPreC
	}
	if i < end {
		g[i].pos = posBaseC
		i++
	}
	pos := posAfterMain
	for ; i < end; i++ {
		switch c := g[i].cat; {
		case c == iMR:
			g[i].pos = posPreC
		case c == iVPre:
			g[i].pos = posPreM
		case c == iVS:
			g[i].pos = g[i-1].pos
		case pos == posAfterMain && c == iVBlw:
			pos = posBelowC
			g[i].pos = pos
		case pos == posBelowC && c == iA:
			g[i].pos = posBeforeSub
		case pos == posBelowC && c == iVBlw:
			g[i].pos = pos
		case pos == posBelowC:
			pos = posAfterSub
			g[i].pos = pos
		default:
			g[i].pos = pos
		}
	}
	b.sort(start, end, func(x, y *Glyph) int { return int(x.pos) - int(y.pos) })
	first, last := end, end
	for k := start; k < end; k++ {
		if g[k].pos == posPreM {
			if first == end {
				first = k
			}
			last = k
		}
	}
	if first < last {
		b.reverseRange(first, last+1)
		from := first
		for j := from; j <= last; j++ {
			if g[j].cat == iVPre {
				b.reverseRange(from, j+1)
				from = j + 1
			}
		}
	}
}
