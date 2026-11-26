package css //nolint:testpackage // reads computed styles off dom with the package's test helpers

import (
	"testing"

	"github.com/lantervale/ember/ui/internal/dom"
)

func TestClipComputesAsChromeWritesIt(t *testing.T) {
	for in, want := range map[string]string{
		"auto":                           "auto",
		"rect(0, 50px, 50px, 0)":         "rect(0px, 50px, 50px, 0px)",
		"rect(0px 100px 100px 0px)":      "rect(0px, 100px, 100px, 0px)",
		"rect( 0 auto auto 0 )":          "rect(0px, auto, auto, 0px)",
		"rect(-50px, auto, auto, -50px)": "rect(-50px, auto, auto, -50px)",
		"rect(1em, 2em, 3em, 0.75em)":    "rect(10px, 20px, 30px, 7.5px)",
		"rect(1in, 72pt, 6pc, 2.54cm)":   "rect(96px, 96px, 96px, 96px)",
		"RECT(auto, auto, auto, auto)":   "rect(auto, auto, auto, auto)",
	} {
		s := gridOf(t, "font-size: 10px; clip: "+in)
		if got := ClipCSS(s.Clip); got != want {
			t.Errorf("%s: %s, want %s", in, got, want)
		}
	}
}

func TestClipRefusesWhatChromeDoes(t *testing.T) {
	for _, in := range []string{
		"rect(0,0 0,0)", "rect(0 0,0,0)", "rect(0,0,0 0)", "rect(0 0,0 0)", "rect(0, 0, 0)", "rect(0 0 0 0 0)",
		"rect(10%, 0, 0, 0)", "circle(10px, 25px)", "inset(0)", "none", "rect(1, 0, 0, 0)",
	} {
		_, warns := Parse("t.css", "div { clip: "+in+" }")
		if len(warns) != 1 {
			t.Errorf("%s: warnings %v, want one", in, warns)
		}
	}
}

func TestClipIsInheritedOnlyByInherit(t *testing.T) {
	for decls, want := range map[string]string{
		"#p { clip: rect(-50px, auto, auto, -50px) } #c { clip: inherit }": "rect(-50px, auto, auto, -50px)",
		"#p { clip: rect(0, 1px, 1px, 0) }":                                "auto",
	} {
		parent, child := el(dom.Div, "p"), el(dom.Div, "c")
		parent.Append(child)
		Compute(page(parent), []*Sheet{sheet(t, decls)})
		if got := ClipCSS(styleOf(child).Clip); got != want {
			t.Errorf("%s: %s, want %s", decls, got, want)
		}
	}
}

func TestClipTransitionsEdgeByEdge(t *testing.T) {
	for from, c := range map[string]struct{ to, half string }{
		"rect(0, 10px, 10px, 0)": {"rect(10px, 30px, 50px, 20px)", "rect(5px, 20px, 30px, 10px)"},
		"rect(0, auto, 10px, 0)": {"rect(10px, auto, 30px, 20px)", "rect(5px, auto, 20px, 10px)"},
		"rect(0, 4px, 10px, 0)":  {"rect(10px, auto, 30px, 20px)", "rect(10px, auto, 30px, 20px)"},
		"auto":                   {"rect(10px, 20px, 30px, 20px)", "rect(10px, 20px, 30px, 20px)"},
	} {
		tr, root, b, sheets := transitionPage(t, "button { clip: "+from+"; transition: clip 1s linear } "+
			".on { clip: "+c.to+" }")
		b.AddClass("on")
		tr.Compute(root, sheets)
		tr.Step(0.5)
		if got := ClipCSS(shown(b).Clip); got != c.half {
			t.Errorf("%s to %s, half way: %s, want %s", from, c.to, got, c.half)
		}
	}
}
