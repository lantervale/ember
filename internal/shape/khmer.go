package shape

// Syllables the Khmer machine finds.
const (
	khmerConsonantSyllable uint8 = iota
	khmerBrokenCluster
	khmerNonKhmerCluster
)

//nolint:gochecknoglobals,goconst // a machine compiled once, its grammar as HarfBuzz writes it
var khmerMachine = newMachine(map[string]uint8{
	"C": iC, "V": iV, "H": iH, "ZWNJ": iZWNJ, "ZWJ": iZWJ, "PLACEHOLDER": iPLACEHOLDER,
	"DOTTEDCIRCLE": iDOTTEDCIRCLE, "Ra": iRa, "VAbv": iVAbv, "VBlw": iVBlw, "VPre": iVPre, "VPst": iVPst,
	"Robatic": iRobatic, "Xgroup": iXgroup, "Ygroup": iYgroup,
}, [][2]string{
	{"c", "(C | Ra | V)"},
	{"cn", "c.((ZWJ|ZWNJ)?.Robatic)?"},
	{"joiner", "(ZWJ | ZWNJ)"},
	{"xgroup", "(joiner*.Xgroup)*"},
	{"ygroup", "Ygroup*"},
	{"matra_group", "VPre? xgroup VBlw? xgroup (joiner?.VAbv)? xgroup VPst?"},
	{"syllable_tail", "xgroup matra_group xgroup (H.c)? ygroup"},
	{"broken_cluster", "Robatic? (H.cn)* (H | syllable_tail)"},
	{"consonant_syllable", "(cn|PLACEHOLDER|DOTTEDCIRCLE) broken_cluster"},
}, []rule{
	{"consonant_syllable", khmerConsonantSyllable},
	{"broken_cluster", khmerBrokenCluster},
	{"any", khmerNonKhmerCluster},
})

// The Khmer features: the basic ones per syllable, the others after.
const (
	khmerPref = iota
	khmerBlwf
	khmerAbvf
	khmerPstf
	khmerCfar
)

//nolint:gochecknoglobals,goconst // a fixed list
var khmerFeatures = [...]string{"pref", "blwf", "abvf", "pstf", "cfar"}

func collectKhmer(c *collector) {
	c.pause(setupKhmerSyllables)
	c.pause(reorderKhmer)
	c.add(true, perSyllable, "locl", "ccmp")
	for _, t := range khmerFeatures {
		c.add(false, manualJoiners|perSyllable, t)
	}
	c.pause(nil)
	c.add(true, manualJoiners, "pres", "abvs", "blws", "psts")
}

// khmerPlan are the masks of the Khmer features.
type khmerPlan struct{ masks [len(khmerFeatures)]uint32 }

func newKhmerPlan(p *Plan) *khmerPlan {
	kp := &khmerPlan{}
	for i, t := range khmerFeatures {
		kp.masks[i] = p.masks[MakeTag(t)]
	}
	return kp
}

func setupKhmerSyllables(_ *Plan, b *buffer) {
	cats := make([]uint8, len(b.glyphs))
	for i, g := range b.glyphs {
		cats[i] = g.cat
	}
	b.syllables(khmerMachine, cats, nil)
}

func reorderKhmer(p *Plan, b *buffer) {
	b.insertDottedCircles(khmerBrokenCluster, iDOTTEDCIRCLE, 0, -1)
	b.eachSyllable(func(start, end int) {
		if k := b.glyphs[start].syllable & 0x0F; k == khmerConsonantSyllable || k == khmerBrokenCluster {
			b.reorderKhmerSyllable(p.khmer, start, end)
		}
	})
}

// reorderKhmerSyllable marks the glyphs after the first for the forms
// below, above and after it, moves a Coeng Ro and what follows it in
// cfar, and a pre-base vowel, to the start.
func (b *buffer) reorderKhmerSyllable(kp *khmerPlan, start, end int) {
	g := b.glyphs
	mask := kp.masks[khmerBlwf] | kp.masks[khmerAbvf] | kp.masks[khmerPstf]
	for i := start + 1; i < end; i++ {
		g[i].mask |= mask
	}
	coengs := 0
	for i := start + 1; i < end; i++ {
		switch {
		case g[i].cat == iH && coengs <= 2 && i+1 < end:
			coengs++
			if g[i+1].cat != iRa {
				continue
			}
			g[i].mask |= kp.masks[khmerPref]
			g[i+1].mask |= kp.masks[khmerPref]
			b.mergeClusters(start, i+2)
			b.move(i, start)
			b.move(i+1, start+1)
			if kp.masks[khmerCfar] != 0 {
				for j := i + 2; j < end; j++ {
					g[j].mask |= kp.masks[khmerCfar]
				}
			}
			coengs = 2
		case g[i].cat == iVPre:
			b.mergeClusters(start, i+1)
			b.move(i, start)
		}
	}
}
