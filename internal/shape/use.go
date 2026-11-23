package shape

import "sort"

// The categories of the Universal Shaping Engine (gen-use-table.py).
const (
	uO uint8 = iota
	uB
	uN
	uGB
	uCGJ
	uSUB
	uH
	uHN
	uZWNJ
	uWJ
	uR
	uCS
	uIS
	uSk
	uG
	uJ
	uSB
	uSE
	uHVM
	uHM
	uHR
	uRK
	uFAbv
	uFBlw
	uFPst
	uMAbv
	uMBlw
	uMPst
	uMPre
	uCMAbv
	uCMBlw
	uVAbv
	uVBlw
	uVPst
	uVPre
	uVMAbv
	uVMBlw
	uVMPst
	uVMPre
	uSMAbv
	uSMBlw
	uFMAbv
	uFMBlw
	uFMPst
)

type useRange struct {
	lo, hi rune
	cat    uint8
}

// useCategory is r's category for the Universal Shaping Engine.
func useCategory(r rune) uint8 {
	i := sort.Search(len(useCategories), func(i int) bool { return useCategories[i].hi >= r })
	if i < len(useCategories) && useCategories[i].lo <= r {
		return useCategories[i].cat
	}
	return uO
}

func (b *buffer) useCategories() {
	for i := range b.glyphs {
		b.glyphs[i].cat = useCategory(b.glyphs[i].Char)
	}
}

// Clusters the USE machine finds.
const (
	useViramaTerminatedCluster uint8 = iota
	useSakotTerminatedCluster
	useStandardCluster
	useNumberJoinerTerminatedCluster
	useNumeralCluster
	useSymbolCluster
	useHieroglyphCluster
	useBrokenCluster
	useNonCluster
)

//nolint:gochecknoglobals,goconst // a machine compiled once, its grammar as HarfBuzz writes it
var useMachine = newMachine(map[string]uint8{
	"O": uO, "B": uB, "N": uN, "GB": uGB, "CGJ": uCGJ, "SUB": uSUB, "H": uH, "HN": uHN, "ZWNJ": uZWNJ, "WJ": uWJ,
	"R": uR, "CS": uCS, "IS": uIS, "Sk": uSk, "G": uG, "J": uJ, "SB": uSB, "SE": uSE, "HVM": uHVM, "HM": uHM,
	"HR": uHR, "RK": uRK, "FAbv": uFAbv, "FBlw": uFBlw, "FPst": uFPst, "MAbv": uMAbv, "MBlw": uMBlw,
	"MPst": uMPst, "MPre": uMPre, "CMAbv": uCMAbv, "CMBlw": uCMBlw, "VAbv": uVAbv, "VBlw": uVBlw, "VPst": uVPst,
	"VPre": uVPre, "VMAbv": uVMAbv, "VMBlw": uVMBlw, "VMPst": uVMPst, "VMPre": uVMPre, "SMAbv": uSMAbv,
	"SMBlw": uSMBlw, "FMAbv": uFMAbv, "FMBlw": uFMBlw, "FMPst": uFMPst,
}, [][2]string{
	{"h", "H | HVM | IS | Sk"},
	{"consonant_modifiers", "CMAbv* CMBlw* ((h B | SUB) CMAbv* CMBlw*)*"},
	{"medial_consonants", "MPre? MAbv? MBlw? MPst?"},
	{"dependent_vowels", "VPre* VAbv* VBlw* VPst* | H"},
	{"vowel_modifiers", "HVM? VMPre* VMAbv* VMBlw* VMPst*"},
	{"final_consonants", "FAbv* FBlw* FPst*"},
	{"final_modifiers", "FMAbv* FMBlw* | FMPst?"},
	{"complex_syllable_start", "(R | CS)? (B | GB)"},
	{"complex_syllable_middle", "consonant_modifiers medial_consonants dependent_vowels vowel_modifiers (Sk B)*"},
	{"complex_syllable_tail", "complex_syllable_middle final_consonants final_modifiers"},
	{"number_joiner_terminated_cluster_tail", "(HN N)* HN"},
	{"numeral_cluster_tail", "(HN N)+"},
	{"symbol_cluster_tail", "SMAbv+ SMBlw* | SMBlw+"},
	{"virama_terminated_cluster_tail", "consonant_modifiers (IS | RK)"},
	{"virama_terminated_cluster", "complex_syllable_start virama_terminated_cluster_tail"},
	{"sakot_terminated_cluster_tail", "complex_syllable_middle Sk"},
	{"sakot_terminated_cluster", "complex_syllable_start sakot_terminated_cluster_tail"},
	{"standard_cluster", "complex_syllable_start complex_syllable_tail"},
	{"tail", "complex_syllable_tail | sakot_terminated_cluster_tail | symbol_cluster_tail | " +
		"virama_terminated_cluster_tail"},
	{"broken_cluster", "R? (tail | number_joiner_terminated_cluster_tail | numeral_cluster_tail)"},
	{"number_joiner_terminated_cluster", "N number_joiner_terminated_cluster_tail"},
	{"numeral_cluster", "N numeral_cluster_tail?"},
	{"symbol_cluster", "(O | GB | SB) tail?"},
	{"hieroglyph_cluster", "SB* G HR? HM? SE* (J SB* (G HR? HM? SE*)?)*"},
}, []rule{
	{"virama_terminated_cluster ZWNJ?", useViramaTerminatedCluster},
	{"sakot_terminated_cluster ZWNJ?", useSakotTerminatedCluster},
	{"standard_cluster ZWNJ?", useStandardCluster},
	{"number_joiner_terminated_cluster ZWNJ?", useNumberJoinerTerminatedCluster},
	{"numeral_cluster ZWNJ?", useNumeralCluster},
	{"symbol_cluster ZWNJ?", useSymbolCluster},
	{"hieroglyph_cluster ZWNJ?", useHieroglyphCluster},
	{"FMPst", useNonCluster},
	{"broken_cluster ZWNJ?", useBrokenCluster},
	{"any", useNonCluster},
})

// usePlan is what the USE keeps of a plan.
type usePlan struct{ rphf uint32 }

func collectUSE(c *collector) {
	c.pause(setupUSESyllables)
	c.add(true, perSyllable, "locl", "ccmp", "nukt")
	c.add(true, manualZWJ|perSyllable, "akhn")
	c.pause(clearSubstituted)
	c.add(false, manualZWJ|perSyllable, "rphf")
	c.pause(recordRphf)
	c.pause(clearSubstituted)
	c.add(true, manualZWJ|perSyllable, "pref")
	c.pause(recordPref)
	c.add(true, manualZWJ|perSyllable, "rkrf", "abvf", "blwf", "half", "pstf", "vatu", "cjct")
	c.pause(reorderUSE)
	c.pause(nil)
	c.add(false, 0, "isol", "init", "medi", "fina")
	c.pause(nil)
	c.add(true, manualZWJ, "abvs", "blws", "haln", "pres", "psts")
}

// setupUSESyllables finds the clusters, a combining grapheme joiner
// and a ZWNJ before a mark left out of the machine's view, and marks
// the glyphs reph may form of and the forms joined scripts take.
func setupUSESyllables(p *Plan, b *buffer) {
	var cats []uint8
	var index []int
	for i := range b.glyphs {
		c := b.glyphs[i].cat
		if c == uCGJ {
			continue
		}
		if c == uZWNJ {
			keep := true
			for j := i + 1; j < len(b.glyphs); j++ {
				if b.glyphs[j].cat != uCGJ {
					keep = !isMarkChar(b.glyphs[j].Char)
					break
				}
			}
			if !keep {
				continue
			}
		}
		cats = append(cats, c)
		index = append(index, i)
	}
	b.syllables(useMachine, cats, index)
	if mask := p.use.rphf; mask != 0 {
		b.eachSyllable(func(start, end int) {
			limit := min(3, end-start)
			if b.glyphs[start].cat == uR {
				limit = 1
			}
			for i := start; i < start+limit; i++ {
				b.glyphs[i].mask |= mask
			}
		})
	}
	b.topographical(p)
}

// topographical gives the glyphs of each cluster the mask of the form
// it takes, a cluster joining the one before it unless a cluster that
// does not join, or a ZWNJ, comes between, as the USE does for scripts
// it does not join as Arabic.
func (b *buffer) topographical(p *Plan) {
	var forms [4]uint32
	all := uint32(0)
	for i, t := range []string{"isol", "init", "medi", "fina"} {
		if m := p.masks[MakeTag(t)]; m != globalMask {
			forms[i] = m
			all |= m
		}
	}
	if all == 0 {
		return
	}
	const isol, initial, medi, fina, none = 0, 1, 2, 3, 4
	lastStart, last := 0, none
	b.eachSyllable(func(start, end int) {
		g := b.glyphs
		kind := g[start].syllable & 0x0F
		switch kind {
		case useHieroglyphCluster, useNonCluster:
			last = none
		default:
			join := last == fina || last == isol
			if join {
				if last == fina {
					last = medi
				} else {
					last = initial
				}
				for i := lastStart; i < start; i++ {
					g[i].mask = g[i].mask&^all | forms[last]
				}
			}
			last = isol
			if join {
				last = fina
			}
			for i := start; i < end; i++ {
				g[i].mask = g[i].mask&^all | forms[last]
			}
		}
		lastStart = start
	})
}

func clearSubstituted(_ *Plan, b *buffer) {
	for i := range b.glyphs {
		b.glyphs[i].props &^= substituted
	}
}

// recordRphf gives the first glyph of each cluster rphf formed the
// category of a repha.
func recordRphf(p *Plan, b *buffer) {
	mask := p.use.rphf
	if mask == 0 {
		return
	}
	b.eachSyllable(func(start, end int) {
		for i := start; i < end && b.glyphs[i].mask&mask != 0; i++ {
			if b.glyphs[i].props&substituted != 0 {
				b.glyphs[i].cat = uR
				break
			}
		}
	})
}

// recordPref gives the first glyph of each cluster pref formed the
// category of a pre-base vowel.
func recordPref(_ *Plan, b *buffer) {
	b.eachSyllable(func(start, end int) {
		for i := start; i < end; i++ {
			if b.glyphs[i].props&substituted != 0 {
				b.glyphs[i].cat = uVPre
				break
			}
		}
	})
}

func reorderUSE(_ *Plan, b *buffer) {
	b.insertDottedCircles(useBrokenCluster, uB, 0, int(uR))
	b.eachSyllable(func(start, end int) { b.reorderUSESyllable(start, end) })
}

func isUSEHalant(g *Glyph) bool {
	return (g.cat == uH || g.cat == uHVM || g.cat == uIS) && g.props&formed == 0
}

// usePostBase is whether a category comes after the base.
func usePostBase(c uint8) bool {
	switch c {
	case uFAbv, uFBlw, uFPst, uFMAbv, uFMBlw, uFMPst, uMAbv, uMBlw, uMPst, uMPre, uVAbv, uVBlw, uVPst, uVPre,
		uVMAbv, uVMBlw, uVMPst, uVMPre:
		return true
	}
	return false
}

// reorderUSESyllable moves a repha to before the first glyph after the
// base, and each pre-base vowel or vowel modifier to after the last
// halant before it.
func (b *buffer) reorderUSESyllable(start, end int) {
	switch b.glyphs[start].syllable & 0x0F {
	case useViramaTerminatedCluster, useSakotTerminatedCluster, useStandardCluster, useSymbolCluster,
		useBrokenCluster:
	default:
		return
	}
	g := b.glyphs
	if g[start].cat == uR && end-start > 1 {
		for i := start + 1; i < end; i++ {
			post := usePostBase(g[i].cat) || isUSEHalant(&g[i])
			if post || i == end-1 {
				if post {
					i--
				}
				b.mergeClusters(start, i+1)
				b.move(start, i)
				break
			}
		}
	}
	j := start
	for i := start; i < end; i++ {
		switch {
		case isUSEHalant(&g[i]):
			j = i + 1
		case (g[i].cat == uVPre || g[i].cat == uVMPre) && compOf(g[i]) == 0 && j < i:
			b.mergeClusters(j, i+1)
			b.move(i, j)
		}
	}
}
