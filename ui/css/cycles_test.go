package css //nolint:testpackage // reads computed styles off dom

import (
	"testing"

	"github.com/lantervale/ember/ui/internal/dom"
)

func TestAPropertyInACycleComputesAsInvalid(t *testing.T) {
	s := sheet(t, `@property --l { syntax: "<length>"; inherits: false; initial-value: 3px; }
@property --l2 { syntax: "<length>"; inherits: false; initial-value: 5px; }
body { font-size: 20px; line-height: 30px; --u: 7px; }
div { --l: calc(10px * (2em / 1em)); --u: var(--l); font-size: var(--u); line-height: var(--l2); --l2: 2lh; }
span { --l: var(--l2); --l2: var(--l); --u: var(--missing); --v: var(--u, 9px); }
p { --l: 2lh; padding-left: var(--l); line-height: 7px; }`)
	d, sp, p := el(dom.Div, ""), el(dom.Span, ""), el(dom.P, "")
	root := page(d, sp, p)
	Compute(root, []*Sheet{s})
	for n, want := range map[*dom.Node]map[string]string{
		d:  {"--l": "3px", "--u": "", "--l2": "5px"},
		sp: {"--l": "3px", "--l2": "5px", "--u": "", "--v": "9px"},
		p:  {"--l": "14px"},
	} {
		for name, v := range want {
			if got := styleOf(n).Vars[name]; got != v {
				t.Errorf("%s %s = %q, want %q", n.Tag(), name, got, v)
			}
		}
	}
	if st := styleOf(d); st.Font.Size != 20 || st.LineHeight.Px != 30 {
		t.Errorf("div font-size %v, line-height %+v; want both inherited, 20 and 30", st.Font.Size, st.LineHeight)
	}
}
