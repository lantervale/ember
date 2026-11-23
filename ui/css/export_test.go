package css

import (
	"maps"
	"strconv"
	"strings"

	"github.com/lantervale/ember/ui/internal/style"
)

// What the chrome suite (package css_test) asks Chrome about.

// WebkitTable is -webkit-'s property tables: the plain property each
// name is read as, and why each of Chrome's own stays as written.
func WebkitTable() (map[string]string, map[string]string) {
	plain := map[string]string{}
	for _, name := range webkit.same {
		plain[name] = name
	}
	for name, as := range webkit.renamed {
		plain[name] = as
	}
	own := maps.Clone(webkit.own)
	maps.Copy(own, webkit.legacy)
	return plain, own
}

// WebkitValues are the -webkit- values read as a plain one.
func WebkitValues() map[string]string { return webkit.values }

// VendorPrefixes are the prefixes with a table.
func VendorPrefixes() []string {
	var out []string
	for _, v := range vendors {
		out = append(out, v.prefix)
	}
	return out
}

// MediaCase is a media query and Chrome's answer to it in ChromeViewport.
type MediaCase struct {
	Query string
	Want  bool
}

// MediaCases are the queries of TestMediaQueriesAreAnsweredAsChromeAnswers.
func MediaCases() []MediaCase {
	var out []MediaCase
	for _, c := range mediaCases {
		out = append(out, MediaCase{c.query, c.want})
	}
	return out
}

// MachineMediaQueries are the queries of
// TestWhatChromeAsksTheMachineFollowsTheMedium.
func MachineMediaQueries() []string { return machineQueries }

// MachineMedium is a medium of
// TestWhatChromeAsksTheMachineFollowsTheMedium and the queries that hold
// for it.
type MachineMedium struct {
	Media   Media
	Holding []string
}

// MachineMedia are the media of
// TestWhatChromeAsksTheMachineFollowsTheMedium.
func MachineMedia() []MachineMedium {
	var out []MachineMedium
	for _, m := range machineMedia {
		out = append(out, MachineMedium{m.m, m.holding})
	}
	return out
}

// ChromeViewport is the viewport MediaCases were answered in.
func ChromeViewport() Media { return chromeViewport }

// Supported answers an @supports condition as a sheet read as a
// browser's does.
func Supported(cond string) bool { return supported(componentsOf(cond)) }

// PseudoElement is one pseudo-element of the table, as a selector with
// an argument where it takes one, and whether ui refuses it.
type PseudoElement struct {
	Selector string
	Refused  bool
}

// PseudoElements are the pseudo-elements of the table.
func PseudoElements() []PseudoElement {
	var out []PseudoElement
	for name, pe := range pseudoElements {
		sel := "::" + name
		if pe.form == formFunction {
			arg := "x"
			switch name {
			case "slotted":
				arg = "*"
			case "scroll-button":
				arg = "left"
			case "picker":
				arg = "select"
			}
			sel += "(" + arg + ")"
		}
		out = append(out, PseudoElement{sel, pe.status == pseudoRefused})
	}
	return out
}

// ClipPathCSS is c as Chrome's getComputedStyle writes it.
func ClipPathCSS(c style.ClipPath) string {
	if c.None() {
		return kwNone
	}
	var box string
	for name, b := range shapeBoxes {
		if b == c.Box {
			box = name
		}
	}
	if c.Shape == style.ShapeNone {
		return box
	}
	var in []string
	switch c.Shape {
	case style.ShapeInset:
		in = append(in, shortest(c.Inset.Top, c.Inset.Right, c.Inset.Bottom, c.Inset.Left))
		r := c.Round
		w := shortest(r.TopLeft.Width, r.TopRight.Width, r.BottomRight.Width, r.BottomLeft.Width)
		h := shortest(r.TopLeft.Height, r.TopRight.Height, r.BottomRight.Height, r.BottomLeft.Height)
		zero := style.Size[style.Length]{Width: style.Px(0), Height: style.Px(0)}
		if r.TopLeft != zero || r.TopRight != zero || r.BottomRight != zero || r.BottomLeft != zero {
			in = append(in, "round", w)
			if h != w {
				in = append(in, "/", h)
			}
		}
	case style.ShapeCircle, style.ShapeEllipse:
		n := 1
		if c.Shape == style.ShapeEllipse {
			n = 2
		}
		if c.Radius[0].Side != style.RadiusClosest || n == 2 && c.Radius[1].Side != style.RadiusClosest {
			for _, r := range c.Radius[:n] {
				in = append(in, [...]string{"closest-side", "farthest-side", lengthText(r.Length)}[r.Side])
			}
		}
		if c.AtSet {
			in = append(in, "at", lengthText(c.At[0]), lengthText(c.At[1]))
		}
	case style.ShapePolygon:
		var pts []string
		if c.EvenOdd {
			pts = append(pts, "evenodd")
		}
		for _, p := range c.Points {
			pts = append(pts, lengthText(p[0])+" "+lengthText(p[1]))
		}
		in = append(in, strings.Join(pts, ", "))
	case style.ShapePath:
		if c.EvenOdd {
			in = append(in, "evenodd,")
		}
		in = append(in, `"`+c.Path+`"`)
	case style.ShapeNone:
	}
	name := [...]string{"", "inset", "circle", "ellipse", "polygon", "path"}[c.Shape]
	out := name + "(" + strings.Join(in, " ") + ")"
	if c.Box != style.ShapeBoxUnset {
		out += " " + box
	}
	return out
}

func shortest(t, r, b, l style.Length) string {
	vs := []style.Length{t, r, b, l}
	switch {
	case l != r:
	case b != t:
		vs = vs[:3]
	case r != t:
		vs = vs[:2]
	default:
		vs = vs[:1]
	}
	out := make([]string, len(vs))
	for i, v := range vs {
		out[i] = lengthText(v)
	}
	return strings.Join(out, " ")
}

func lengthText(l style.Length) string {
	n := func(v float32) string { return strconv.FormatFloat(float64(v), 'f', -1, 32) }
	switch {
	case !l.IsPercent:
		return n(l.Px) + "px"
	case l.Px == 0:
		return n(l.Percent) + "%"
	case l.Px < 0:
		return "calc(" + n(l.Percent) + "% - " + n(-l.Px) + "px)"
	}
	return "calc(" + n(l.Percent) + "% + " + n(l.Px) + "px)"
}
