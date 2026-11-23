package css

import (
	"fmt"
	"strings"

	"github.com/lantervale/ember/ui/internal/style"
)

const (
	shortWhiteSpace  = "white-space"
	propCollapse     = "white-space-collapse"
	propTextWrapMode = "text-wrap-mode"
	propBoxTrim      = "text-box-trim"
	propBoxEdge      = "text-box-edge"
	kwText           = "text"
	kwAlphabetic     = "alphabetic"
	kwWrap           = "wrap"
	// kwCollapseSpaces is white-space-collapse's collapse.
	kwCollapseSpaces = "collapse"
)

// quoted is a string.
func quoted(c Component) error {
	if c.Kind == String {
		return nil
	}
	return fmt.Errorf("expected a string, found %q", c.String())
}

//nolint:gochecknoglobals,goconst // lookup tables
var (
	whiteSpaces = map[string]style.WhiteSpace{"normal": style.Normal, "nowrap": style.NoWrapSpace, "pre": style.Pre,
		"pre-wrap": style.PreWrap, "pre-line": style.PreLine}
	collapses = map[string]style.Collapse{kwCollapseSpaces: style.CollapseSpaces, "preserve": style.Preserve,
		"preserve-breaks": style.PreserveBreaks}
	overEdges  = map[string]style.TextEdge{kwText: style.EdgeText, "cap": style.EdgeCap, "ex": style.EdgeEx}
	underEdges = map[string]style.TextEdge{kwText: style.EdgeText, kwAlphabetic: style.EdgeAlphabetic}
)

// checkTextBoxEdge takes auto, text, or an edge over the text (text, cap,
// ex) and one under it (text, alphabetic), as Chrome 157 reads it.
func checkTextBoxEdge(v []Component) error {
	switch {
	case len(v) == 1 && isWord(v[0], kwAuto, kwText):
		return nil
	case len(v) == 2 && isWord(v[0], kwText, "cap", "ex") && isWord(v[1], kwText, kwAlphabetic):
		return nil
	}
	return fmt.Errorf("expected auto, text, or an edge over the text and one under it")
}

// expandWhiteSpace reads white-space: one of CSS 2's keywords, or a
// white-space-collapse and a text-wrap-mode in either order.
func expandWhiteSpace(v []Component, out map[string][]Component) error {
	if len(v) == 1 && v[0].Kind == Ident {
		if ws, ok := whiteSpaces[strings.ToLower(v[0].Text)]; ok {
			out[propCollapse] = []Component{word(collapseWord(ws.Collapse()))}
			out[propTextWrapMode] = []Component{word(map[bool]string{true: kwWrap, false: "nowrap"}[ws.Wraps()])}
			return nil
		}
	}
	return expandAnyOrder(shorthands[shortWhiteSpace], v, out)
}

func collapseWord(c style.Collapse) string {
	for w, k := range collapses {
		if k == c {
			return w
		}
	}
	return kwCollapseSpaces
}

// expandTextBox reads text-box: normal, or a text-box-trim and a
// text-box-edge in either order, an edge alone trimming both ends.
func expandTextBox(v []Component, out map[string][]Component) error {
	if len(v) == 1 && isWord(v[0], "normal") {
		return nil
	}
	trim := -1
	for i, c := range v {
		if checks[propBoxTrim]([]Component{c}) == nil {
			trim = i
			break
		}
	}
	var edge []Component
	switch trim {
	case -1:
		edge = v
		out[propBoxTrim] = []Component{word("trim-both")}
	case 0:
		edge = v[1:]
		out[propBoxTrim] = v[:1]
	case len(v) - 1:
		edge = v[:trim]
		out[propBoxTrim] = v[trim:]
	default:
		return fmt.Errorf("text-box takes its trim before or after its edges")
	}
	if len(edge) > 0 {
		if err := checkTextBoxEdge(edge); err != nil {
			return err
		}
		out[propBoxEdge] = edge
	}
	return nil
}

// textEdgeOf is text-box-edge's value.
func textEdgeOf(_ *ctx, v []Component) style.TextBoxEdge {
	v = items(v)
	if isWord(v[0], kwAuto) {
		return style.TextBoxEdge{}
	}
	if len(v) == 1 {
		return style.TextBoxEdge{Named: true}
	}
	return style.TextBoxEdge{Over: overEdges[strings.ToLower(v[0].Text)],
		Under: underEdges[strings.ToLower(v[1].Text)], Named: true}
}

// textWrapProps are the longhands of white-space, text-wrap and text-box,
// and the text properties beside them.
//
//nolint:goconst // the longhands' table
func textWrapProps() map[string]prop {
	wraps := map[string]bool{kwWrap: true, "nowrap": false}
	return map[string]prop{
		propCollapse: {
			set: func(x *ctx, v []Component) {
				x.c.WhiteSpace = style.WhiteSpaceOf(collapses[strings.ToLower(v[0].Text)], x.c.WhiteSpace.Wraps())
			},
			inherit: func(dst, src *style.Computed) {
				dst.WhiteSpace = style.WhiteSpaceOf(src.WhiteSpace.Collapse(), dst.WhiteSpace.Wraps())
			},
			anim: anim{kind: discrete},
		},
		propTextWrapMode: {
			set: func(x *ctx, v []Component) {
				x.c.WhiteSpace = style.WhiteSpaceOf(x.c.WhiteSpace.Collapse(), wraps[strings.ToLower(v[0].Text)])
			},
			inherit: func(dst, src *style.Computed) {
				dst.WhiteSpace = style.WhiteSpaceOf(dst.WhiteSpace.Collapse(), src.WhiteSpace.Wraps())
			},
			anim: anim{kind: discrete},
		},
		"text-wrap-style": keyword(func(c *style.Computed) *style.WrapStyle { return &c.WrapStyle },
			map[string]style.WrapStyle{"auto": style.WrapAuto, "balance": style.WrapBalance,
				"pretty": style.WrapPretty, "stable": style.WrapStable}),
		"hyphens": keyword(func(c *style.Computed) *style.Hyphens { return &c.Hyphens },
			map[string]style.Hyphens{"manual": style.HyphensManual, "none": style.HyphensNone,
				"auto": style.HyphensAuto}),
		"hyphenate-character": {
			set: func(x *ctx, v []Component) {
				x.c.HyphenChosen = v[0].Kind == String
				x.c.HyphenChar = ""
				if x.c.HyphenChosen {
					x.c.HyphenChar = v[0].Text
				}
			},
			inherit: func(dst, src *style.Computed) { dst.HyphenChar, dst.HyphenChosen = src.HyphenChar, src.HyphenChosen },
			anim:    anim{kind: discrete},
		},
		propBoxTrim: keyword(func(c *style.Computed) *style.TextBoxTrim { return &c.TextBoxTrim },
			map[string]style.TextBoxTrim{"none": style.TrimNone, "trim-start": style.TrimStart,
				"trim-end": style.TrimEnd, "trim-both": style.TrimBoth}),
		propBoxEdge: field(func(c *style.Computed) *style.TextBoxEdge { return &c.TextBoxEdge }, textEdgeOf,
			flips[style.TextBoxEdge]()),
		"text-spacing-trim": keyword(func(c *style.Computed) *style.SpacingTrim { return &c.SpacingTrim },
			map[string]style.SpacingTrim{"normal": style.SpacingTrimNormal, "space-all": style.SpaceAll,
				"space-first": style.SpaceFirst, "trim-start": style.TrimStartSpacing}),
		"text-autospace": keyword(func(c *style.Computed) *bool { return &c.Autospace },
			map[string]bool{"normal": true, "no-autospace": false}),
		"interpolate-size": keyword(func(c *style.Computed) *bool { return &c.KeywordSizes },
			map[string]bool{"numeric-only": false, "allow-keywords": true}),
	}
}
