package css

import (
	"strings"

	"github.com/lantervale/ember/ui/internal/style"
)

// logical names the physical property each logical one is in horizontal
// left-to-right lines, where a block's start is its top and an inline
// start its left. A logical declaration takes the place of the physical
// one its element's writing-mode and direction give it in the cascade
// (Declaration.flow), as in a browser. The shorthands of two sides
// (margin-inline…) are in shorthands.
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
		"overscroll-behavior-inline": propOverscrollX, "overscroll-behavior-block": propOverscrollY,
		"border-start-start-radius": "border-top-left-radius", "border-start-end-radius": "border-top-right-radius",
		"border-end-start-radius": "border-bottom-left-radius", "border-end-end-radius": "border-bottom-right-radius",
	}
	for side, physical := range map[string]string{
		"block-start": "top", "block-end": "bottom", "inline-start": "left", "inline-end": "right",
	} {
		m["margin-"+side] = "margin-" + physical
		m["padding-"+side] = "padding-" + physical
		m["border-"+side] = "border-" + physical
		m["scroll-padding-"+side] = "scroll-padding-" + physical
		m["scroll-margin-"+side] = "scroll-margin-" + physical
		for _, part := range []string{"width", "style", "color"} {
			m["border-"+side+"-"+part] = "border-" + physical + "-" + part
		}
	}
	return m
}()

// flowShorthands are the shorthands of both inline sides, written as left
// then right, and of both block sides, written as top then bottom.
//
//nolint:gochecknoglobals // a fixed set
var flowShorthands = func() map[string]bool {
	m := map[string]bool{shortPaddingInline: true, shortMarginInline: true, shortInsetInline: true,
		shortScrollPaddingInline: true, shortScrollMarginInline: true, "padding-block": true, "margin-block": true,
		"inset-block": true, "scroll-padding-block": true, "scroll-margin-block": true}
	for name := range shorthands {
		if strings.HasPrefix(name, "border-inline") || strings.HasPrefix(name, "border-block") {
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

// flow is what the physical property of a logical one depends on: an
// element's writing-mode and whether its direction is rtl.
type flow struct {
	mode style.WritingMode
	rtl  bool
}

func flowOf(s *style.Computed) flow { return flow{mode: s.Mode, rtl: s.Dir == style.RTL} }

// physical is the property a logical declaration stands for in f, name
// being the one it stands for in horizontal left-to-right lines.
func (f flow) physical(name string) string {
	if as, ok := flowNames[f][name]; ok {
		return as
	}
	return name
}

// written is the inverse of physical: the name, in horizontal
// left-to-right lines, of a logical declaration standing for name in f;
// the longhand whose value a shorthand's expansion holds for it.
func (f flow) written(name string) string {
	if as, ok := flowWritten[f][name]; ok {
		return as
	}
	return name
}

// place is the longhand d gives its value to in f, name being one it names.
func (f flow) place(name string, d *Declaration) string {
	if !d.flow {
		return name
	}
	return f.physical(name)
}

// key is the longhand of d's expansion holding what d gives name in f.
func (f flow) key(name string, d *Declaration) string {
	if !d.flow {
		return name
	}
	return f.written(name)
}

// sides are the physical sides block-start, block-end, inline-start and
// inline-end stand on in f.
func (f flow) sides() [4]string {
	is, ie := "left", "right"
	if f.mode.Vertical() {
		is, ie = "top", "bottom"
		if f.mode.Upward() {
			is, ie = ie, is
		}
	}
	if f.rtl {
		is, ie = ie, is
	}
	switch {
	case !f.mode.Vertical():
		return [4]string{"top", "bottom", is, ie}
	case f.mode.BlockFromRight():
		return [4]string{"right", "left", is, ie}
	}
	return [4]string{"left", "right", is, ie}
}

//nolint:gochecknoglobals // a fixed set
var axisSwaps = map[string]string{
	"width": "height", "height": "width", "min-width": "min-height", "min-height": "min-width",
	"max-width": "max-height", "max-height": "max-width", "overflow-x": "overflow-y", "overflow-y": "overflow-x",
	propOverscrollX: propOverscrollY, propOverscrollY: propOverscrollX,
}

// flowName is name, a property of a side, a corner or an axis as
// horizontal left-to-right lines have it, in f.
func flowName(f flow, name string) string {
	if f.mode.Vertical() {
		if as, ok := axisSwaps[name]; ok {
			return as
		}
	}
	side := f.sides()
	to := map[string]string{"top": side[0], "bottom": side[1], "left": side[2], "right": side[3]}
	parts := strings.Split(name, "-")
	var turned []int
	for i, p := range parts {
		if as, ok := to[p]; ok {
			parts[i] = as
			turned = append(turned, i)
		}
	}
	if len(turned) == 2 && horizontal(parts[turned[0]]) {
		// a corner is named by its top or bottom side first
		parts[turned[0]], parts[turned[1]] = parts[turned[1]], parts[turned[0]]
	}
	return strings.Join(parts, "-")
}

// flowNames and flowWritten are flow's physical and written over every
// property ui knows, for each flow.
//
//nolint:gochecknoglobals // derived from the properties
var flowNames, flowWritten = func() (map[flow]map[string]string, map[flow]map[string]string) {
	names, written := map[flow]map[string]string{}, map[flow]map[string]string{}
	for mode := style.HorizontalTB; mode <= style.SidewaysLR; mode++ {
		for _, rtl := range []bool{false, true} {
			f := flow{mode: mode, rtl: rtl}
			to, from := map[string]string{}, map[string]string{}
			add := func(name string) {
				if as := flowName(f, name); as != name {
					to[name], from[as] = as, name
				}
			}
			for name := range properties {
				add(name)
			}
			for name := range shorthands {
				add(name)
			}
			names[f], written[f] = to, from
		}
	}
	return names, written
}()

// physical is the property name stands for in f: a logical one's
// physical one.
func physical(name string, f flow) string {
	as, ok := logical[name]
	if !ok {
		return name
	}
	return f.physical(as)
}
