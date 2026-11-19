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
	used uint16
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

// calcOf is d, absolute and with terms left, as layout reads it; false
// when it takes more steps than a Calc holds.
func calcOf(d Dim) (unique.Handle[style.Calc], bool) {
	steps := calcSteps(nil, d)
	if len(steps) > style.CalcSteps {
		return unique.Handle[style.Calc]{}, false
	}
	c := style.Calc{N: len(steps)}
	copy(c.Steps[:], steps)
	return unique.Make(c), true
}

// calcSteps adds to steps those that leave d's value on the stack.
func calcSteps(steps []style.CalcStep, d Dim) []style.CalcStep {
	summed := 0
	if d.Px != 0 || d.Pct != 0 || len(d.terms) == 0 {
		steps = append(steps, style.CalcStep{Op: style.CalcPush, Px: float32(d.Px), Percent: float32(d.Pct)})
		summed++
	}
	for _, t := range d.terms {
		for _, a := range t.args {
			steps = calcSteps(steps, a)
		}
		op := style.CalcMin
		if t.most {
			op = style.CalcMax
		}
		if len(t.args) > 1 {
			steps = append(steps, style.CalcStep{Op: op, N: len(t.args)})
		}
		if t.factor != 1 {
			steps = append(steps, style.CalcStep{Op: style.CalcScale, Px: float32(t.factor)})
		}
		summed++
	}
	if summed > 1 {
		steps = append(steps, style.CalcStep{Op: style.CalcSum, N: summed})
	}
	return steps
}
