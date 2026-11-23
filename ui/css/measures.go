package css

import (
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
}

// fontUnits are f's, or what CSS has for a font that says nothing: half
// an em for ex and ch, an em for cap and ic, and lines 1.2 em apart.
func (m *measures) fontUnits(f style.Font) FontUnits {
	if m == nil || m.font == nil {
		return FontUnits{Ex: f.Size / 2, Ch: f.Size / 2, Cap: f.Size, Ic: f.Size, Line: 1.2 * f.Size}
	}
	return m.font(f)
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
		steps = append(steps, style.CalcStep{Op: t.op, N: len(t.args)})
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
