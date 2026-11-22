package css

import "strings"

// logical names the physical property each logical one is under
// direction ltr: ui lays text out in horizontal lines only, where a
// block's start is its top, and an inline start is its left, its right
// under rtl (Declaration.inline). A logical declaration takes its
// physical one's place in the cascade, as in a browser. The shorthands
// of two sides (margin-inline…) are in shorthands.
//
//nolint:gochecknoglobals,goconst // the logical properties' table
var logical = func() map[string]string {
	m := map[string]string{
		"inline-size": "width", "block-size": "height",
		"min-inline-size": "min-width", "min-block-size": "min-height",
		"max-inline-size": "max-width", "max-block-size": "max-height",
		"inset-block-start": "top", "inset-block-end": "bottom",
		"inset-inline-start": "left", "inset-inline-end": "right",
		"overflow-inline": "overflow-x", "overflow-block": "overflow-y",
		"border-start-start-radius": "border-top-left-radius", "border-start-end-radius": "border-top-right-radius",
		"border-end-start-radius": "border-bottom-left-radius", "border-end-end-radius": "border-bottom-right-radius",
	}
	for side, physical := range map[string]string{
		"block-start": "top", "block-end": "bottom", "inline-start": "left", "inline-end": "right",
	} {
		m["margin-"+side] = "margin-" + physical
		m["padding-"+side] = "padding-" + physical
		m["border-"+side] = "border-" + physical
		for _, part := range []string{"width", "style", "color"} {
			m["border-"+side+"-"+part] = "border-" + physical + "-" + part
		}
	}
	return m
}()

// inlineShorthands are the shorthands of both inline sides, written as
// left then right.
//
//nolint:gochecknoglobals // a fixed set
var inlineShorthands = func() map[string]bool {
	m := map[string]bool{shortPaddingInline: true, shortMarginInline: true, shortInsetInline: true}
	for name := range shorthands {
		if strings.HasPrefix(name, "border-inline") {
			m[name] = true
		}
	}
	return m
}()

// horizontal reports whether the physical property name is of a left or
// a right side or corner.
func horizontal(name string) bool {
	return strings.Contains(name, "left") || strings.Contains(name, "right")
}

// mirrored is the longhand name of the other horizontal side.
func mirrored(name string) string { return mirrors[name] }

//nolint:gochecknoglobals // derived from the properties
var mirrors = func() map[string]string {
	swap := strings.NewReplacer("left", "right", "right", "left")
	m := make(map[string]string, len(properties))
	for name := range properties {
		m[name] = swap.Replace(name)
	}
	return m
}()
