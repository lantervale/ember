package css

import "strings"

// Sizes is the sizes attribute of an img or a source, read: the width
// of the slot its picture is chosen for, by media conditions.
type Sizes struct{ entries []sourceSize }

// sourceSize is one entry of a sizes attribute: a length, or auto, used
// where cond holds or has none.
type sourceSize struct {
	cond *mediaCond
	size Dim
	auto bool
}

// ParseSizes reads a sizes attribute as HTML does: entries apart by
// commas, each a media condition and a length that is no percentage
// and not less than 0, or auto; an entry that is no such pair is left
// out.
func ParseSizes(text string) Sizes {
	var s Sizes
	media := mediaParser{warn: func(Pos, string, ...any) {}}
	for _, part := range splitOn(componentsOf(text), func(c Component) bool { return c.Kind == Comma }) {
		part = trim(part)
		if len(part) == 0 {
			continue
		}
		var e sourceSize
		last := part[len(part)-1]
		if isWord(last, "auto") {
			e.auto = true
		} else if d, ok := sourceSizeValue(last); ok {
			e.size = d
		} else {
			continue
		}
		if rest := items(part[:len(part)-1]); len(rest) > 0 {
			cond, ok := media.condition(rest, true)
			if !ok {
				continue
			}
			e.cond = cond
		}
		s.entries = append(s.entries, e)
	}
	return s
}

// sourceSizeValue reads a length of a sizes attribute: no percentage,
// not below 0, a function only of CSS's maths.
func sourceSizeValue(c Component) (Dim, bool) {
	if c.Kind == Number && c.Num == 0 {
		return Dim{kind: dimLength}, true
	}
	if c.Kind != Dimension && c.Kind != Function {
		return Dim{}, false
	}
	d, err := parseDim(c)
	if err != nil || d.kind != dimLength || c.Kind == Dimension && c.Num < 0 {
		return Dim{}, false
	}
	return d, true
}

// Width is the slot's width s gives in m, in CSS pixels: the size of
// its first entry whose condition holds, auto standing for auto when
// it is not below 0 and skipped otherwise, the viewport's width when
// none holds. Font units are those of the initial font, 16 pixels.
func (s Sizes) Width(m Media, auto float32) float32 {
	for _, e := range s.entries {
		if e.auto && auto < 0 || e.cond != nil && e.cond.eval(m) != yes {
			continue
		}
		if e.auto {
			return auto
		}
		return float32(max(e.size.Absolute(initialUnits(m)).Resolve(0), 0))
	}
	return m.Width
}

// initialUnits are what lengths stand for in a media query: the
// initial font's units, by CSS's fallbacks off its size, and m's
// viewport.
func initialUnits(m Media) *Units {
	const em = 16
	u := &Units{Em: em, Rem: em}
	in := [...]float64{unitEx: em / 2, unitCh: em / 2, unitCap: em, unitIc: em, unitLh: 1.2 * em}
	for i, v := range in {
		u.Rel[i], u.Rel[unitRex+relUnit(i)] = v, v
	}
	w, h := float64(m.Width), float64(m.Height)
	u.Rel[unitVw], u.Rel[unitVh], u.Rel[unitVmin], u.Rel[unitVmax] = w/100, h/100, min(w, h)/100, max(w, h)/100
	return u
}

// MediaQueries is a media query list read outside a sheet, such as the
// media of a source.
type MediaQueries struct{ list *mediaList }

// ParseMediaQueries reads a media query list, as @media's prelude; one
// that is malformed never holds.
func ParseMediaQueries(text string) MediaQueries {
	media := mediaParser{warn: func(Pos, string, ...any) {}}
	return MediaQueries{media.list(componentsOf(strings.TrimSpace(text)))}
}

// Hold reports whether one of q's queries holds in m; an empty list
// does.
func (q MediaQueries) Hold(m Media) bool { return q.list.matches(m) }
