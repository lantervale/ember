package css

import (
	"fmt"
	"slices"
	"strings"

	"github.com/lantervale/ember/ui/internal/style"
)

// dashed reports whether c is a dashed ident, an anchor or try name.
func dashed(c Component) bool {
	return c.Kind == Ident && strings.HasPrefix(c.Text, "--") && len(c.Text) > 2
}

// checkDashedList takes none (or all, with all), or dashed idents
// between commas.
func checkDashedList(all bool) check {
	return func(v []Component) error {
		if len(v) == 1 && (isWord(v[0], kwNone) || all && isWord(v[0], "all")) {
			return nil
		}
		for _, part := range splitOn(v, func(c Component) bool { return c.Kind == Comma }) {
			if len(part) != 1 || !dashed(part[0]) {
				return fmt.Errorf("expected none or --names between commas, found %q", join(v))
			}
		}
		return nil
	}
}

func checkPositionAnchor(v []Component) error {
	if len(v) == 1 && (isWord(v[0], "normal", kwNone, "auto") || dashed(v[0])) {
		return nil
	}
	return fmt.Errorf("position-anchor takes normal, none, auto or an --anchor-name, found %q", join(v))
}

func checkPositionArea(v []Component) error {
	if len(v) == 1 && isWord(v[0], kwNone) {
		return nil
	}
	_, err := positionArea(v, false)
	return err
}

func checkTryOrder(v []Component) error {
	if len(v) == 1 && isWord(v[0], tryOrders...) {
		return nil
	}
	return fmt.Errorf("position-try-order takes one of %s, found %q", strings.Join(tryOrders, ", "), join(v))
}

func checkTryFallbacks(v []Component) error {
	if len(v) == 1 && isWord(v[0], kwNone) {
		return nil
	}
	_, err := tryFallbacks(v, false)
	return err
}

func checkPositionVisibility(v []Component) error {
	if len(v) == 1 && isWord(v[0], "always") {
		return nil
	}
	seen := map[string]bool{}
	for _, c := range v {
		w := strings.ToLower(c.Text)
		if c.Kind != Ident || seen[w] || !isWord(c, "anchors-valid", "anchors-visible", "no-overflow") {
			return fmt.Errorf("position-visibility takes always, or anchors-valid, anchors-visible and "+
				"no-overflow, found %q", join(v))
		}
		seen[w] = true
	}
	return nil
}

// expandPositionTry splits position-try into a position-try-order, if it
// starts with one, and the fallbacks after it.
func expandPositionTry(v []Component, out map[string][]Component) error {
	if len(v) > 1 && isWord(v[0], tryOrders...) {
		out["position-try-order"], v = v[:1], v[1:]
	}
	if len(v) == 0 || checkTryFallbacks(v) != nil {
		return fmt.Errorf("position-try takes an order and fallbacks, found %q", join(v))
	}
	out["position-try-fallbacks"] = v
	return nil
}

const propPositionTry = "position-try"

//nolint:gochecknoglobals // position-try-order's keywords
var tryOrders = []string{kwNormal, "most-width", "most-height", "most-block-size", "most-inline-size"}

// areaWord is one keyword of a position-area: the span it gives its axis
// and which axis that is.
type areaWord struct {
	span style.AreaSpan
	axis areaAxis
}

type areaAxis uint8

const (
	// anyAxis is center, span-all, and start and end alone: the axis
	// their place in the value gives them.
	anyAxis areaAxis = iota
	xAxis
	yAxis
)

//nolint:gochecknoglobals // the span each track keyword stands for
var (
	startTrack  = style.AreaSpan{From: 0, To: 0}
	centerTrack = style.AreaSpan{From: 1, To: 1}
	endTrack    = style.AreaSpan{From: 2, To: 2}
	spanStart   = style.AreaSpan{From: 0, To: 1}
	spanEnd     = style.AreaSpan{From: 1, To: 2}
	spanAll     = style.AreaSpan{From: 0, To: 2}
)

func mirror(s style.AreaSpan) style.AreaSpan { return style.AreaSpan{From: 2 - s.To, To: 2 - s.From} }

// areaWordOf is the keyword w of a position-area, for an element of
// direction rtl, where x-start and inline-start are its right.
//
//nolint:goconst // the area keywords
func areaWordOf(w string, rtl bool) (areaWord, bool) {
	w = strings.ToLower(w)
	switch w {
	case "center":
		return areaWord{centerTrack, anyAxis}, true
	case "span-all":
		return areaWord{spanAll, anyAxis}, true
	case "left":
		return areaWord{startTrack, xAxis}, true
	case "right":
		return areaWord{endTrack, xAxis}, true
	case "span-left":
		return areaWord{spanStart, xAxis}, true
	case "span-right":
		return areaWord{spanEnd, xAxis}, true
	case "top":
		return areaWord{startTrack, yAxis}, true
	case "bottom":
		return areaWord{endTrack, yAxis}, true
	case "span-top":
		return areaWord{spanStart, yAxis}, true
	case "span-bottom":
		return areaWord{spanEnd, yAxis}, true
	}
	span := startTrack
	var rest string
	plain, spans := strings.CutPrefix(w, "span-")
	plain = strings.TrimPrefix(plain, "self-")
	switch {
	case strings.HasSuffix(plain, "start"):
		rest = strings.TrimSuffix(plain, "start")
		if spans {
			span = spanStart
		}
	case strings.HasSuffix(plain, "end"):
		rest, span = strings.TrimSuffix(plain, "end"), endTrack
		if spans {
			span = spanEnd
		}
	default:
		return areaWord{}, false
	}
	switch rest {
	case "":
		return areaWord{span, anyAxis}, true
	case "x-", "inline-", "self-x-", "self-inline-":
		if rtl {
			span = mirror(span)
		}
		return areaWord{span, xAxis}, true
	case "y-", "block-", "self-y-", "self-block-":
		return areaWord{span, yAxis}, true
	}
	return areaWord{}, false
}

// positionArea reads a position-area of one or two keywords: one of an
// axis alone spans the whole other axis, one of either axis stands for
// both, and two of either axis are the block axis's then the inline
// axis's.
func positionArea(v []Component, rtl bool) (style.PositionArea, error) {
	bad := fmt.Errorf("position-area takes none or one or two area keywords, found %q", join(v))
	if len(v) < 1 || len(v) > 2 {
		return style.PositionArea{}, bad
	}
	var ws []areaWord
	for _, c := range v {
		if c.Kind != Ident {
			return style.PositionArea{}, bad
		}
		w, ok := areaWordOf(c.Text, rtl)
		if !ok {
			return style.PositionArea{}, bad
		}
		ws = append(ws, w)
	}
	inline := func(s style.AreaSpan) style.AreaSpan {
		if rtl {
			return mirror(s)
		}
		return s
	}
	a := style.PositionArea{Set: true}
	if len(ws) == 1 {
		switch ws[0].axis {
		case xAxis:
			a.X, a.Y = ws[0].span, spanAll
		case yAxis:
			a.X, a.Y = spanAll, ws[0].span
		case anyAxis:
			a.X, a.Y = inline(ws[0].span), ws[0].span
		}
		return a, nil
	}
	first, second := ws[0], ws[1]
	switch {
	case first.axis == xAxis && second.axis != xAxis:
		a.X, a.Y = first.span, second.span
	case second.axis == xAxis && first.axis != xAxis:
		a.X, a.Y = second.span, first.span
	case first.axis == yAxis && second.axis == anyAxis:
		a.Y, a.X = first.span, inline(second.span)
	case first.axis == anyAxis && second.axis == yAxis:
		a.Y, a.X = second.span, inline(first.span)
	case first.axis == anyAxis && second.axis == anyAxis:
		a.Y, a.X = first.span, inline(second.span)
	default:
		return style.PositionArea{}, bad
	}
	return a, nil
}

// tryFallbacks reads position-try-fallbacks: between commas, an
// @position-try name and flips in any order, or a position-area.
func tryFallbacks(v []Component, rtl bool) ([]style.PositionTry, error) {
	var out []style.PositionTry
	for _, part := range splitOn(v, func(c Component) bool { return c.Kind == Comma }) {
		if len(part) == 0 {
			return nil, fmt.Errorf("position-try-fallbacks has an empty fallback")
		}
		var t style.PositionTry
		tactics := true
		for _, c := range part {
			f, flip := tryFlips[strings.ToLower(c.Text)]
			switch {
			case c.Kind == Ident && flip && !slices.Contains(t.Flips, f):
				t.Flips = append(t.Flips, f)
			case dashed(c) && t.Name == "":
				t.Name = c.Text
			default:
				tactics = false
			}
		}
		if !tactics {
			a, err := positionArea(part, rtl)
			if err != nil {
				return nil, fmt.Errorf("position-try-fallbacks: %q is no fallback", join(part))
			}
			t = style.PositionTry{Area: a}
		}
		out = append(out, t)
	}
	return out, nil
}

//nolint:gochecknoglobals // the try tactics
var tryFlips = map[string]style.Flip{
	"flip-block": style.FlipBlock, "flip-inline": style.FlipInline, "flip-start": style.FlipStart,
	"flip-x": style.FlipInline, "flip-y": style.FlipBlock,
}

func dashedNames(v []Component) []string {
	var names []string
	for _, c := range v {
		if dashed(c) {
			names = append(names, c.Text)
		}
	}
	return names
}

func anchorProp(set func(x *ctx, v []Component), copyOf func(dst, src *style.Anchoring)) prop {
	return prop{
		set:     set,
		inherit: func(dst, src *style.Computed) { copyOf(&dst.Anchoring, &src.Anchoring) },
		anim:    anim{kind: discrete},
	}
}

//nolint:goconst // the anchor properties' table
func anchorProps() map[string]prop {
	return map[string]prop{
		"anchor-name": anchorProp(func(x *ctx, v []Component) { x.c.Anchoring.Names = dashedNames(v) },
			func(dst, src *style.Anchoring) { dst.Names = src.Names }),
		"anchor-scope": anchorProp(func(x *ctx, v []Component) {
			a := &x.c.Anchoring
			a.Scope, a.ScopeNames = style.ScopeNone, nil
			switch {
			case isWord(v[0], "all"):
				a.Scope = style.ScopeAll
			case !isWord(v[0], kwNone):
				a.Scope, a.ScopeNames = style.ScopeNames, dashedNames(v)
			}
		}, func(dst, src *style.Anchoring) { dst.Scope, dst.ScopeNames = src.Scope, src.ScopeNames }),
		"position-anchor": anchorProp(func(x *ctx, v []Component) {
			a := &x.c.Anchoring
			a.Anchor, a.AnchorName = style.AnchorNormal, ""
			switch {
			case isWord(v[0], kwNone):
				a.Anchor = style.AnchorNone
			case isWord(v[0], "auto"):
				a.Anchor = style.AnchorAuto
			case dashed(v[0]):
				a.Anchor, a.AnchorName = style.AnchorNamed, v[0].Text
			}
		}, func(dst, src *style.Anchoring) { dst.Anchor, dst.AnchorName = src.Anchor, src.AnchorName }),
		"position-area": anchorProp(func(x *ctx, v []Component) {
			x.c.Anchoring.Area = style.PositionArea{}
			if !isWord(v[0], kwNone) {
				x.c.Anchoring.Area, _ = positionArea(v, x.rtl)
			}
		}, func(dst, src *style.Anchoring) { dst.Area = src.Area }),
		"position-try-fallbacks": anchorProp(func(x *ctx, v []Component) {
			x.c.Anchoring.Fallbacks = nil
			if !isWord(v[0], kwNone) {
				x.c.Anchoring.Fallbacks, _ = tryFallbacks(v, x.rtl)
			}
		}, func(dst, src *style.Anchoring) { dst.Fallbacks = src.Fallbacks }),
		"position-try-order": anchorProp(func(x *ctx, v []Component) {
			switch strings.ToLower(v[0].Text) {
			case "most-width", "most-inline-size":
				x.c.Anchoring.Order = style.TryMostWidth
			case "most-height", "most-block-size":
				x.c.Anchoring.Order = style.TryMostHeight
			default:
				x.c.Anchoring.Order = style.TryNormal
			}
		}, func(dst, src *style.Anchoring) { dst.Order = src.Order }),
		"position-visibility": anchorProp(func(x *ctx, v []Component) {
			x.c.Anchoring.Visibility = style.VisibleAlways
			for _, c := range v {
				switch {
				case isWord(c, "anchors-visible"):
					x.c.Anchoring.Visibility |= style.AnchorsVisible
				case isWord(c, "no-overflow"):
					x.c.Anchoring.Visibility |= style.NoOverflow
				}
			}
		}, func(dst, src *style.Anchoring) { dst.Visibility = src.Visibility }),
	}
}
