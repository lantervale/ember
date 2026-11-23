package css

import (
	"strconv"
	"strings"
)

// FontPaletteValues is an @font-palette-values rule: a palette, named
// for font-palette, of the colour fonts of Families, Base or the first
// for a light or dark background, its entries changed by Overrides.
type FontPaletteValues struct {
	Name     string
	Families []string
	// Base is base-palette's index, 0 without one.
	Base        int
	Light, Dark bool
	Overrides   []PaletteOverride
}

// PaletteOverride is a palette entry's colour in place of the font's.
type PaletteOverride struct {
	Index int
	Color Color
}

// FontPaletteValues lists the sheet's @font-palette-values rules in
// order.
func (s *Sheet) FontPaletteValues() []FontPaletteValues { return s.palettes }

// For reports whether the palette is one of family's.
func (v FontPaletteValues) For(family string) bool {
	for _, f := range v.Families {
		if strings.EqualFold(f, family) {
			return true
		}
	}
	return false
}

// String writes the rule back as CSS.
func (v FontPaletteValues) String() string {
	var b strings.Builder
	b.WriteString("@font-palette-values " + v.Name + " { font-family: ")
	for i, f := range v.Families {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(strconv.Quote(f))
	}
	switch {
	case v.Light:
		b.WriteString("; base-palette: light")
	case v.Dark:
		b.WriteString("; base-palette: dark")
	default:
		b.WriteString("; base-palette: " + strconv.Itoa(v.Base))
	}
	for i, o := range v.Overrides {
		if i == 0 {
			b.WriteString("; override-colors: ")
		} else {
			b.WriteString(", ")
		}
		b.WriteString(strconv.Itoa(o.Index) + " " + o.Color.String())
	}
	b.WriteString("; }\n")
	return b.String()
}

func (p *parser) fontPaletteValues(t Token) {
	var name string
	var block []Component
	for {
		if p.peek().Kind == EOF {
			p.warnf(t.Pos, "@font-palette-values has no { block }")
			return
		}
		c := p.component()
		if c.Kind == LBrace {
			block = c.Args
			break
		}
		switch {
		case c.Kind == Whitespace:
		case c.Kind == Ident && strings.HasPrefix(c.Text, "--") && name == "":
			name = c.Text
		default:
			p.warnf(c.Pos, "@font-palette-values: expected a --name")
			name = "\x00"
		}
	}
	if name == "" || name == "\x00" {
		p.warnf(t.Pos, "@font-palette-values needs a --name; dropped")
		return
	}
	v := FontPaletteValues{Name: name}
	for _, part := range splitOn(block, func(c Component) bool { return c.Kind == Semicolon }) {
		part = trim(part)
		if len(part) == 0 {
			continue
		}
		desc := part[0]
		rest := trim(part[1:])
		if desc.Kind != Ident || len(rest) < 2 || rest[0].Kind != Colon {
			p.warnf(desc.Pos, "expected a descriptor in @font-palette-values")
			continue
		}
		raw := trim(rest[1:])
		if !p.paletteDescriptor(&v, strings.ToLower(desc.Text), raw) {
			p.warnf(desc.Pos, "@font-palette-values: %s: value not understood", desc.Text)
		}
	}
	if len(v.Families) == 0 {
		p.warnf(t.Pos, "@font-palette-values %s needs font-family; dropped", name)
		return
	}
	p.sheet.palettes = append(p.sheet.palettes, v)
}

func (p *parser) paletteDescriptor(v *FontPaletteValues, desc string, raw []Component) bool {
	switch desc {
	case propFontFamily:
		var families []string
		for _, fam := range splitOn(raw, func(c Component) bool { return c.Kind == Comma }) {
			name := familyName(items(trim(fam)))
			if name == "" {
				return false
			}
			families = append(families, name)
		}
		v.Families = families
	case "base-palette":
		it := items(raw)
		switch {
		case len(it) != 1:
			return false
		case isWord(it[0], "light"):
			v.Light, v.Dark = true, false
		case isWord(it[0], "dark"):
			v.Light, v.Dark = false, true
		case it[0].Kind == Number:
			n, err := strconv.Atoi(it[0].Text)
			if err != nil || n < 0 {
				return false
			}
			v.Base, v.Light, v.Dark = n, false, false
		default:
			return false
		}
	case "override-colors":
		var out []PaletteOverride
		for _, entry := range splitOn(raw, func(c Component) bool { return c.Kind == Comma }) {
			it := items(trim(entry))
			if len(it) != 2 || it[0].Kind != Number {
				return false
			}
			n, err := strconv.Atoi(it[0].Text)
			if err != nil || n < 0 || isWord(it[1], kwCurrentColor) {
				return false
			}
			c, err := parseColor(it[1])
			if err != nil {
				return false
			}
			out = append(out, PaletteOverride{n, c})
		}
		v.Overrides = out
	default:
		return false
	}
	return true
}

// colourTech reports whether ui draws a font of the colour technology
// tech (color-COLRv0…), as Chrome does: all of them but SVG's.
func colourTech(tech string) bool {
	switch strings.ToLower(tech) {
	case "color-colrv0", "color-colrv1", "color-sbix", "color-cbdt":
		return true
	}
	return false
}
