package css

import (
	"math"
	"strconv"
	"strings"
	"unique"

	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

// FontUnits are what the units that read a font stand for in one, in
// points: its x-height (ex), the advance of its zero (ch), the height of
// its capitals (cap), the advance of its 水 (ic) and the height of its
// normal line (lh).
type FontUnits struct{ Ex, Ch, Cap, Ic, Line float32 }

// measures are what lengths are measured against besides font sizes:
// the fonts, the viewport and the root's style.
type measures struct {
	font          func(style.Font) FontUnits
	width, height float32
	root          *dom.Node
	// used are the relative units read since the tree was last styled whole.
	used uint32
	// dppx is the device pixels a point the widths snap to.
	dppx float32
	// dark is whether the Document prefers the dark scheme, and
	// prefersRead whether an element took its scheme from it since then.
	dark, prefersRead bool
	// inactive is whether the Document's window is inactive, and
	// inactiveRead whether a :window-inactive was matched since the tree
	// was last styled whole.
	inactive, inactiveRead bool
	// containers are the elements styled as size containers, sizes the
	// content boxes of those laid out, read those a container unit
	// measured, and requery those whose descendants the next Compute
	// restyles (Transitions.Requery).
	containers, read, requery map[*dom.Node]bool
	sizes                     map[*dom.Node][2]float32
	// scroll is the state of the scroll-state containers
	// (Transitions.RequeryScroll).
	scroll map[*dom.Node]ScrollState
	// titleBar is the env(titlebar-area-*) values, and titleBarRead
	// whether one was read since the tree was last styled whole.
	titleBar     TitleBarArea
	titleBarRead bool
}

// TitleBarArea is the part of a window's title bar a Document drawn under
// it has for its own, beside the system's buttons, in its points: Chrome's
// window controls overlay, read by env(titlebar-area-x), -y, -width and
// -height. Its zero value is none, where the system keeps its title bar
// above the page and those are undefined, as in Chrome without the
// overlay.
type TitleBarArea struct {
	X, Y, Width, Height float32
}

// zeroEnv are the environment variables Chrome defines as 0px on a
// desktop: no notch and no virtual keyboard.
//
//nolint:gochecknoglobals // a read-only table
var zeroEnv = map[string]bool{
	"safe-area-inset-top": true, "safe-area-inset-right": true, "safe-area-inset-bottom": true,
	"safe-area-inset-left": true, "keyboard-inset-top": true, "keyboard-inset-right": true,
	"keyboard-inset-bottom": true, "keyboard-inset-left": true, "keyboard-inset-width": true,
	"keyboard-inset-height": true,
}

// variable is the value of env(name), false when it is undefined: names
// are case-sensitive, as custom properties.
func (m *measures) variable(name string) ([]Component, bool) {
	if zeroEnv[name] {
		return parsedVar("0px"), true
	}
	at, ok := strings.CutPrefix(name, "titlebar-area-")
	if !ok || m == nil {
		return nil, false
	}
	m.titleBarRead = true
	a := m.titleBar
	if a.Height <= 0 {
		return nil, false
	}
	var v float32
	switch at {
	case "x":
		v = a.X
	case "y":
		v = a.Y
	case propWidth:
		v = a.Width
	case propHeight:
		v = a.Height
	default:
		return nil, false
	}
	return parsedVar(strconv.FormatFloat(float64(v), 'f', -1, 32) + "px"), true
}

// fontUnits are f's, or what CSS has for a font that says nothing: half
// an em for ex and ch, an em for cap and ic, and lines 1.2 em apart.
func (m *measures) fontUnits(f style.Font) FontUnits {
	if m == nil || m.font == nil {
		return FontUnits{Ex: f.Size / 2, Ch: f.Size / 2, Cap: f.Size, Ic: f.Size, Line: 1.2 * f.Size}
	}
	return m.font(f)
}

// snap is v towards zero to a whole device pixel, a whole point on a
// display of fewer; a width under one is one when least is set, as
// Chrome snaps border widths.
func (m *measures) snap(v float32, least bool) float32 {
	d := float32(1)
	if m != nil {
		d = max(m.dppx, 1)
	}
	if least && v > 0 && v*d < 1 {
		return 1 / d
	}
	return float32(math.Trunc(float64(v*d))) / d
}

func (m *measures) viewport() (float32, float32) {
	if m == nil {
		return 0, 0
	}
	return m.width, m.height
}

func (m *measures) rootStyle() *style.Computed {
	if m == nil || m.root == nil {
		return nil
	}
	return dom.Data(m.root).Style
}

// calcOf is d, absolute and with terms left, as layout reads it.
func calcOf(d Dim) unique.Handle[style.Calc] {
	return unique.Make(style.NewCalc(calcSteps(nil, d)))
}

// calcSteps adds to steps those that leave d's value on the stack.
func calcSteps(steps []style.CalcStep, d Dim) []style.CalcStep {
	summed := 0
	if v := d.value(); v != 0 || d.Pct != 0 || len(d.terms) == 0 {
		steps = append(steps, style.CalcStep{Op: style.CalcPush, Px: v, Percent: d.Pct})
		summed++
	}
	for _, t := range d.terms {
		for _, a := range t.args {
			steps = calcSteps(steps, a)
		}
		steps = append(steps, style.CalcStep{Op: t.op, N: len(t.args), Anchor: t.anchor})
		if t.factor != 1 {
			steps = append(steps, style.CalcStep{Op: style.CalcScale, Px: t.factor})
		}
		summed++
	}
	if summed > 1 {
		steps = append(steps, style.CalcStep{Op: style.CalcSum, N: summed})
	}
	return steps
}
