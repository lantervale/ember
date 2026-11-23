//go:build chrome

package css_test

import (
	"testing"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/style"
)

const cycleProperties = `
@property --l { syntax: "<length>"; inherits: false; initial-value: 3px; }
@property --li { syntax: "<length>"; inherits: true; initial-value: 4px; }
@property --l2 { syntax: "<length>"; inherits: false; initial-value: 5px; }
@property --n { syntax: "<number>"; inherits: false; initial-value: 2; }
`

func cycleCompared() map[string]func(*style.Computed) string {
	props := varsCompared([]string{"--l", "--li", "--l2", "--n", "--u", "--v", "--w"})
	for _, p := range []string{"font-size", "line-height", "padding-left", "padding-right"} {
		props[p] = everything[p]
	}
	return props
}

// TestUnitCyclesComputeAsInChrome has registered custom properties read
// units of the font or of the line height in the font-size or the
// line-height that sets them, as CSS Properties and Values' unit cycles
// do, and compares what each side of the cycle computes to with Chrome:
// the property reading the custom one unset, the custom one invalid,
// and every custom property between them invalid too.
func TestUnitCyclesComputeAsInChrome(t *testing.T) {
	d := ui.NewDocument()
	d.AddSheet(cycleProperties + `
body { font-size: 20px; line-height: 30px; --li: 6px; --u: 7px; }
#em { --l: 2em; font-size: var(--l); }
#ex { --l: 2ex; font-size: var(--l); }
#ch { --l: 2ch; font-size: var(--l); }
#cap { --l: 2cap; font-size: var(--l); }
#ic { --l: 2ic; font-size: var(--l); }
#lh { --l: 2lh; font-size: var(--l); }
#rem { --l: 2rem; font-size: var(--l); }
#rlh { --l: 2rlh; line-height: var(--l); }
#px { --l: 10px; font-size: var(--l); }
#inh { --li: 2em; font-size: var(--li); }
#calc { --l: calc(1em + 2px); font-size: calc(var(--l) + 1px); }
#typed { --l: calc(10px * (2em / 1em)); font-size: var(--l); }
#fallback { --l: 2em; font-size: var(--l, 33px); }
#chain { --l: 2em; --u: var(--l); font-size: var(--u); }
#via { --u: 2em; --l: var(--u); font-size: var(--l); }
#unreg { --u: 2em; font-size: var(--u); }
#other { --l: 2em; --l2: var(--l); padding-left: var(--l); padding-right: var(--l2); font-size: 12px; }
#reader { --l: 2em; font-size: var(--l); padding-left: var(--l); --u: var(--l); --l2: var(--l); }
#lhc { --l: 2lh; line-height: var(--l); }
#lhem { --l: 2em; line-height: var(--l); }
#lhex { --l: 2ex; line-height: var(--l); }
#lhw { --l: 2lh; padding-left: var(--l); line-height: 7px; }
#num { --n: calc(2em / 1em); font-size: calc(var(--n) * 10px); }
#both { --l: 2em; --l2: 3lh; font-size: var(--l); line-height: var(--l2); }
#vc { --u: var(--v); --v: var(--u); --w: var(--u, 1px); padding-left: var(--v, 8px); }
#miss { --u: var(--missing); --v: var(--u, 9px); }
#two { --l: 2em; --l2: 7px; font-size: calc(var(--l) + var(--l2)); }
#vc2 { --u: var(--v) var(--w); --v: var(--u); --w: 1px; }
#vc3 { --u: var(--w) var(--v); --v: var(--u); --w: 1px; }
#rc { --l: var(--l2); --l2: var(--l); padding-left: var(--l); }
`)
	ids := []string{"em", "ex", "ch", "cap", "ic", "lh", "rem", "rlh", "px", "inh", "calc", "typed", "fallback",
		"chain", "via", "unreg", "other", "reader", "lhc", "lhem", "lhex", "lhw", "num", "both", "vc", "miss", "two",
		"vc2", "vc3", "rc"}
	for _, id := range ids {
		d.Body().Append(ui.Div(ui.ID(id), ui.Span(ui.Text("x"))))
	}
	matchChromeProps(t, d, cycleCompared(), 2)
}

// TestRootUnitCyclesComputeAsInChrome is TestUnitCyclesComputeAsInChrome
// on the root, where the root units are its own font's and line's.
func TestRootUnitCyclesComputeAsInChrome(t *testing.T) {
	for name, rule := range map[string]string{
		"rem":    `--l: 2rem; font-size: var(--l);`,
		"rlh":    `--l: 2rlh; line-height: var(--l);`,
		"rem-lh": `--l: 2rem; line-height: var(--l);`,
		"rlh-fs": `--l: 2rlh; font-size: var(--l);`,
		"rex":    `--l: 2rex; font-size: var(--l);`,
		"em":     `--l: 2em; font-size: var(--l); --li: 1rem;`,
	} {
		t.Run(name, func(t *testing.T) {
			d := ui.NewDocument()
			d.AddSheet(cycleProperties + `:root { line-height: 20px; ` + rule + ` }
span { display: block; padding-left: 1rem; padding-right: 1rlh; }`)
			d.Body().Append(ui.Span(ui.Text("x")))
			matchChromeProps(t, d, cycleCompared(), 0)
		})
	}
}
