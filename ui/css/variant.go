package css

import (
	"fmt"
	"strings"

	"github.com/lantervale/ember/ui/internal/style"
)

// variantGroups are, for each font-variant longhand kept as written, its
// keywords by group: a value takes at most one of each group, written
// back in this order.
//
//nolint:gochecknoglobals // a grammar table
var variantGroups = map[string][][]string{
	"font-variant-ligatures": {{"common-ligatures", "no-common-ligatures"},
		{"discretionary-ligatures", "no-discretionary-ligatures"}, {"historical-ligatures", "no-historical-ligatures"},
		{"contextual", "no-contextual"}},
	"font-variant-numeric": {{"lining-nums", "oldstyle-nums"}, {"proportional-nums", "tabular-nums"},
		{"diagonal-fractions", "stacked-fractions"}, {"ordinal"}, {"slashed-zero"}},
	"font-variant-east-asian": {{"jis78", "jis83", "jis90", "jis04", "simplified", "traditional"},
		{"full-width", "proportional-width"}, {"ruby"}},
	"font-variant-alternates": {{"historical-forms"}},
	"font-variant-position":   {{"sub", "super"}},
}

// capsWords are font-variant-caps' keywords.
//
//nolint:gochecknoglobals // a grammar table
var capsWords = map[string]style.FontCaps{"normal": style.NormalCaps, "small-caps": style.SmallCaps,
	"all-small-caps": style.AllSmallCaps, "petite-caps": style.PetiteCaps, "all-petite-caps": style.AllPetiteCaps,
	"unicase": style.Unicase, "titling-caps": style.TitlingCaps}

// variantLonghands are font-variant's longhands, in the order they
// serialise.
//
//nolint:gochecknoglobals // a shorthand's longhands
var variantLonghands = []string{"font-variant-ligatures", "font-variant-caps", "font-variant-alternates",
	"font-variant-numeric", "font-variant-east-asian", "font-variant-position"}

// variantGroup is the group of the longhand name word belongs to, -1 for
// none.
func variantGroup(name, word string) int {
	for g, ws := range variantGroups[name] {
		for _, w := range ws {
			if strings.EqualFold(w, word) {
				return g
			}
		}
	}
	return -1
}

// checkVariant is the check of a font-variant longhand kept as written:
// normal, none for the ligatures, or keywords of distinct groups.
func checkVariant(name string) check {
	return func(v []Component) error {
		if len(v) == 1 && (isWord(v[0], "normal") || name == "font-variant-ligatures" && isWord(v[0], kwNone)) {
			return nil
		}
		var seen []int
		for _, c := range v {
			g := -1
			if c.Kind == Ident {
				g = variantGroup(name, c.Text)
			}
			if g < 0 {
				return fmt.Errorf("%s: %q is not a keyword it takes", name, c.String())
			}
			for _, s := range seen {
				if s == g {
					return fmt.Errorf("%s: %q repeats a choice", name, c.String())
				}
			}
			seen = append(seen, g)
		}
		return nil
	}
}

// variantText is a longhand's value as Chrome writes it back: its
// keywords in their groups' order, "" for normal.
func variantText(name string, v []Component) string {
	if len(v) == 1 && isWord(v[0], "normal") {
		return ""
	}
	if len(v) == 1 && isWord(v[0], kwNone) {
		return kwNone
	}
	words := make([]string, len(variantGroups[name]))
	for _, c := range v {
		if g := variantGroup(name, c.Text); g >= 0 {
			words[g] = strings.ToLower(c.Text)
		}
	}
	var out []string
	for _, w := range words {
		if w != "" {
			out = append(out, w)
		}
	}
	return strings.Join(out, " ")
}

// expandVariant splits font-variant: normal, none (no ligatures), or
// each keyword given to the longhand that takes it.
func expandVariant(v []Component, out map[string][]Component) error {
	if len(v) == 1 && isWord(v[0], "normal", kwNone) {
		for _, l := range variantLonghands {
			out[l] = []Component{word("normal")}
		}
		if isWord(v[0], kwNone) {
			out["font-variant-ligatures"] = v
		}
		return nil
	}
	for _, c := range v {
		into := ""
		switch {
		case c.Kind != Ident:
		case capsWords[strings.ToLower(c.Text)] != style.NormalCaps:
			if out["font-variant-caps"] != nil {
				return fmt.Errorf("font-variant: %q is a second capitals keyword", c.String())
			}
			into = "font-variant-caps"
		default:
			for _, l := range variantLonghands {
				if variantGroup(l, c.Text) >= 0 {
					into = l
				}
			}
		}
		if into == "" {
			return fmt.Errorf("font-variant: %q is not a keyword it takes", c.String())
		}
		out[into] = append(out[into], c)
	}
	for _, l := range variantLonghands {
		if out[l] == nil {
			out[l] = []Component{word("normal")}
		} else if err := checks[l](out[l]); err != nil {
			return err
		}
	}
	return nil
}

// variantField computes the font-variant longhand name as written back.
func variantField(name string) func(_ *ctx, v []Component) string {
	return func(_ *ctx, v []Component) string { return variantText(name, v) }
}
