package css

// logical names the physical property each logical one is: ui lays text
// out left to right in horizontal lines only, where a block's start is
// its top and an inline start its left. A logical declaration takes its
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
