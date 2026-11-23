package css

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/lantervale/ember/ui/internal/style"
)

// kerningWords are font-kerning's keywords.
//
//nolint:gochecknoglobals // a grammar table
var kerningWords = map[string]style.FontKerning{kwAuto: style.KerningAuto, kwNormal: style.KerningNormal,
	kwNone: style.KerningNone}

// checkFeatures is font-feature-settings' check: normal, or tags each
// with on, off or a whole number.
func checkFeatures(v []Component) error {
	if _, ok := fontFeatures(v); !ok {
		return errors.New(`expected normal, or "tag" with on, off or a number, comma separated`)
	}
	return nil
}

// featureSettings is font-feature-settings as Chrome writes it back: each
// tag once, its last value, in the order of the tags; each quoted, its
// value after it unless 1; "" for normal.
func featureSettings(_ *ctx, v []Component) string {
	feats, _ := fontFeatures(v)
	values := map[string]int{}
	for _, f := range feats {
		values[f.Tag] = f.Value
	}
	parts := make([]string, 0, len(values))
	for _, tag := range slices.Sorted(maps.Keys(values)) {
		part := fmt.Sprintf("%q", tag)
		if values[tag] != 1 {
			part += fmt.Sprintf(" %d", values[tag])
		}
		parts = append(parts, part)
	}
	return strings.Join(parts, ", ")
}

// kwRuby is font-variant-east-asian's ruby, whose feature is ruby too;
// kwSub and kwSuper are vertical-align's and font-variant-position's.
const (
	kwRuby  = "ruby"
	kwSub   = "sub"
	kwSuper = "super"
)

// variantFeatures are the OpenType features each keyword of
// font-variant's longhands turns on, or off for the features after a
// minus, as Chrome asks HarfBuzz for them.
//
//nolint:gochecknoglobals // a lookup table
var variantFeatures = map[string]string{
	kwNone:                       "-liga,-clig,-dlig,-hlig,-calt",
	"common-ligatures":           "liga,clig",
	"no-common-ligatures":        "-liga,-clig",
	"discretionary-ligatures":    "dlig",
	"no-discretionary-ligatures": "-dlig",
	"historical-ligatures":       "hlig",
	"no-historical-ligatures":    "-hlig",
	"contextual":                 "calt",
	"no-contextual":              "-calt",
	"lining-nums":                "lnum",
	"oldstyle-nums":              "onum",
	"proportional-nums":          "pnum",
	"tabular-nums":               "tnum",
	"diagonal-fractions":         "frac",
	"stacked-fractions":          "afrc",
	"ordinal":                    "ordn",
	"slashed-zero":               "zero",
	"jis78":                      "jp78",
	"jis83":                      "jp83",
	"jis90":                      "jp90",
	"jis04":                      "jp04",
	"simplified":                 "smpl",
	"traditional":                "trad",
	"full-width":                 "fwid",
	"proportional-width":         "pwid",
	kwRuby:                       kwRuby,
	"historical-forms":           "hist",
	kwSuper:                      "sups",
	kwSub:                        "subs",
}

// shapingFeatures are the features t's text is shaped with, HarfBuzz's
// way: the common ligatures and contextual alternates text-rendering:
// optimizeSpeed turns off where font-variant-ligatures names neither,
// font-variant's longhands but its capitals, then
// font-kerning, then the optional ligatures letter-spacing turns off,
// then, after a '|' where a face's @font-face features go
// (shape.Splice), font-feature-settings, a later one overriding an
// earlier, and font-variant-position last, which Chrome's settings do not
// turn off.
func shapingFeatures(t *style.Text) string {
	var out, settings []string
	if t.TextRendering == style.OptimizeSpeed {
		lig := " " + t.FontVariant.Ligatures + " "
		if !strings.Contains(lig, "common-ligatures ") && lig != " none " {
			out = append(out, variantFeatures["no-common-ligatures"])
		}
		if !strings.Contains(lig, "contextual ") && lig != " none " {
			out = append(out, variantFeatures["no-contextual"])
		}
	}
	for _, v := range []string{t.FontVariant.Ligatures, t.FontVariant.Numeric, t.FontVariant.EastAsian,
		t.FontVariant.Alternates} {
		for _, w := range strings.Fields(v) {
			if f := variantFeatures[w]; f != "" {
				out = append(out, f)
			}
		}
	}
	switch t.Kerning {
	case style.KerningNone:
		out = append(out, "-kern")
	case style.KerningNormal:
		out = append(out, "kern")
	case style.KerningAuto:
	}
	if t.LetterSpacing != (style.Spacing{}) {
		out = append(out, variantFeatures[kwNone])
	}
	for part := range strings.SplitSeq(t.FeatureSettings, ", ") {
		tag, value, _ := strings.Cut(part, " ")
		tag = strings.Trim(tag, `"`)
		switch {
		case tag == "":
		case value == "":
			settings = append(settings, tag)
		case value == "0":
			settings = append(settings, "-"+tag)
		default:
			settings = append(settings, tag+"="+value)
		}
	}
	if f := variantFeatures[t.FontVariant.Position]; f != "" {
		settings = append(settings, f)
	}
	if len(out) == 0 {
		return strings.Join(settings, ",")
	}
	return strings.Join(out, ",") + "|" + strings.Join(settings, ",")
}
