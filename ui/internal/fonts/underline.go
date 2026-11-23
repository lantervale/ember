package fonts

import (
	"encoding/binary"

	"github.com/lantervale/ember/ui/internal/box"
)

// underlined sets m's underline from c's post table at scale points a
// unit: its top, the table's underlinePosition, below the baseline, as
// Skia reads it for Chrome; and its em share, OS/2's typographic ascender
// over the em it and the descender span, else the ascent's share of the
// ascent and descent.
func (c *Face) underlined(m *box.Metrics, scale float32) {
	if p := c.font.PostTable(); p != nil {
		m.Underline = -float32(p.UnderlinePosition) * scale
		m.UnderlineThickness = float32(p.UnderlineThickness) * scale
		m.Underlined = true
	}
	if !c.typoRead && c.TTF != nil {
		c.typoRead = true
		c.typo[0], c.typo[1], c.typoOK = typoMetrics(c.TTF)
	}
	if c.typoOK {
		m.EmAscent = c.typo[0] / (c.typo[0] - c.typo[1])
	} else if h := m.Ascent + m.Descent; h > 0 && m.Ascent >= 0 {
		m.EmAscent = m.Ascent / h
	}
}

// typoMetrics are the sTypoAscender and sTypoDescender of the OS/2 table
// of ttf, when it has one whose ascender is above the baseline.
func typoMetrics(ttf []byte) (float32, float32, bool) {
	be := binary.BigEndian
	if len(ttf) < 12 {
		return 0, 0, false
	}
	n := int(be.Uint16(ttf[4:]))
	for i := range n {
		rec := 12 + 16*i
		if rec+16 > len(ttf) {
			break
		}
		if string(ttf[rec:rec+4]) != "OS/2" {
			continue
		}
		off, size := int(be.Uint32(ttf[rec+8:])), int(be.Uint32(ttf[rec+12:]))
		const descender = 70
		if size < descender+2 || off < 0 || off+descender+2 > len(ttf) {
			return 0, 0, false
		}
		asc := float32(int16(be.Uint16(ttf[off+descender-2:]))) //nolint:gosec // a signed number
		desc := float32(int16(be.Uint16(ttf[off+descender:])))  //nolint:gosec // a signed number
		return asc, desc, asc > 0 && asc > desc
	}
	return 0, 0, false
}
