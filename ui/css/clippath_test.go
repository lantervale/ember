package css //nolint:testpackage // reads computed styles off dom with the package's test helpers

import (
	"strings"
	"testing"

	"github.com/lantervale/ember/ui/internal/style"
)

func TestClipPathComputesAsChromeWritesIt(t *testing.T) {
	elliptic := "inset(1px 2px 3px 4px round 1px 2px 3px 4px / 5px 6px)"
	xywhInset := "inset(20% calc(70% - 10px) calc(80% - 40px) 10px round 5px)"
	for in, want := range map[string]string{
		"none":        "none",
		"inset(10px)": "inset(10px)",
		"inset(10px 20px 10px 20px round 4px 4px)": "inset(10px 20px round 4px)",
		elliptic:                            elliptic,
		"inset(10% round 50%)":              "inset(10% round 50%)",
		"inset(10px round 0 / 5px)":         "inset(10px round 0px / 5px)",
		"inset(-10px)":                      "inset(-10px)",
		"circle()":                          "circle()",
		"circle(40px)":                      "circle(40px)",
		"circle(closest-side)":              "circle()",
		"circle(farthest-side at left top)": "circle(farthest-side at 0% 0%)",
		"circle(at 10px 20px)":              "circle(at 10px 20px)",
		"circle(at center)":                 "circle(at 50% 50%)",
		"circle(at right)":                  "circle(at 100% 50%)",
		"circle(at left 5px bottom 6px)":    "circle(at 5px calc(100% - 6px))",
		"circle(calc(10px + 5%))":           "circle(calc(5% + 10px))",
		"circle(1em)":                       "circle(10px)",
		"ellipse()":                         "ellipse()",
		"ellipse(10px 20%)":                 "ellipse(10px 20%)",
		"ellipse(closest-side farthest-side at 30% 40%)": "ellipse(closest-side farthest-side at 30% 40%)",
		"ellipse(at top)":                             "ellipse(at 50% 0%)",
		"polygon(0 0, 100% 0, 50% 100%)":              "polygon(0px 0px, 100% 0px, 50% 100%)",
		"polygon(evenodd, 0 0, 100% 0, 50% 100%)":     "polygon(evenodd, 0px 0px, 100% 0px, 50% 100%)",
		"polygon(nonzero, 0 0, 10px 0, 5px 10px)":     "polygon(0px 0px, 10px 0px, 5px 10px)",
		"polygon(0 0)":                                "polygon(0px 0px)",
		`path("M0 0 L 10 10 Z")`:                      `path("M 0 0 L 10 10 Z")`,
		`path(evenodd, "M 0,0 l 10 10 h 5 z")`:        `path(evenodd, "M 0 0 L 10 10 H 15 Z")`,
		`path("m 0 0 c 1 2 3 4 5 6 a 1 1 0 0 1 2 2")`: `path("M 0 0 C 1 2 3 4 5 6 A 1 1 0 0 1 7 8")`,
		"xywh(10px 20px 30px 40px)":                   "inset(20px calc(100% - 40px) calc(100% - 60px) 10px)",
		"xywh(10px 20% 30% 40px round 5px)":           xywhInset,
		"rect(10px 20px 30px 40px)":                   "inset(10px calc(100% - 20px) calc(100% - 30px) 40px)",
		"rect(auto auto auto auto)":                   "inset(0%)",
		"rect(10px auto 30% auto round 2px)":          "inset(10px 0% 70% round 2px)",
		"padding-box":                                 "padding-box",
		"border-box":                                  "border-box",
		"margin-box":                                  "margin-box",
		"view-box":                                    "view-box",
		"circle() padding-box":                        "circle() padding-box",
		"border-box circle(10px)":                     "circle(10px)",
		"inset(0) border-box":                         "inset(0px)",
		"content-box polygon(0 0, 1px 1px, 2px 0)":    "polygon(0px 0px, 1px 1px, 2px 0px) content-box",
	} {
		s := gridOf(t, "font-size: 10px; clip-path: "+in)
		if got := ClipPathCSS(s.ClipPath); got != want {
			t.Errorf("%s: %s, want %s", in, got, want)
		}
	}
}

func TestClipPathRefusesWhatChromeDoes(t *testing.T) {
	for _, in := range []string{
		"circle(at right 10px top)", "circle(-5px)", "polygon()", `path("M 0 0 L")`, `path("")`,
		"ellipse(10px)", "inset()", "circle() padding-box content-box", "nonsense", "polygon(0 0 0)",
	} {
		_, warns := Parse("t.css", "div { clip-path: "+in+" }")
		if len(warns) != 1 {
			t.Errorf("%s: warnings %v, want one", in, warns)
		}
	}
	_, warns := Parse("t.css", "div { clip-path: url(#clip) }")
	if len(warns) != 1 || !strings.Contains(warns[0].Error(), "url() is not supported") {
		t.Errorf("url(): %v", warns)
	}
}

func TestClipPathTransitionsBetweenShapesOfAKind(t *testing.T) {
	for from, c := range map[string]struct{ to, half string }{
		"inset(0 round 0)":      {"inset(20px 10% round 8px)", "inset(10px 5% round 4px)"},
		"circle(10px at 0% 0%)": {"circle(30px at 100% 50%)", "circle(20px at 50% 25%)"},
		"ellipse(10px 20px)":    {"ellipse(30px 40% at 50% 50%)", "ellipse(20px calc(20% + 10px) at 50% 50%)"},
		"polygon(0 0, 10px 0, 0 10px)": {"polygon(10px 10px, 30px 0, 0 50%)",
			"polygon(5px 5px, 20px 0px, 0px calc(25% + 5px))"},
		"circle(10px)":               {"inset(10px)", "inset(10px)"},
		"polygon(0 0, 1px 0, 0 1px)": {"polygon(0 0, 1px 0)", "polygon(0px 0px, 1px 0px)"},
		"polygon(evenodd, 0 0, 1px 0, 0 1px)": {"polygon(0 0, 3px 0, 0 3px)",
			"polygon(0px 0px, 3px 0px, 0px 3px)"},
		"circle(closest-side)":     {"circle(20px)", "circle(20px)"},
		"circle(10px) padding-box": {"circle(20px)", "circle(20px)"},
		"none":                     {"circle(20px)", "circle(20px)"},
	} {
		tr, root, b, sheets := transitionPage(t, "button { clip-path: "+from+"; transition: clip-path 1s linear } "+
			".on { clip-path: "+c.to+" }")
		b.AddClass("on")
		tr.Compute(root, sheets)
		tr.Step(0.5)
		if got := ClipPathCSS(shown(b).ClipPath); got != c.half {
			t.Errorf("%s to %s, half way: %s, want %s", from, c.to, got, c.half)
		}
	}
}

func TestClipPathKeyframesRunOnTheShape(t *testing.T) {
	tr, _, d, _ := animationPage(t, `@keyframes reveal { from { clip-path: circle(0px) } to { clip-path: circle(40px) } }
div { animation: reveal 1s linear }`)
	tr.Step(0.25)
	if got := ClipPathCSS(shown(d).ClipPath); got != "circle(10px)" {
		t.Errorf("a quarter in: %s", got)
	}
	tr2, _, d2, _ := animationPage(t, `@keyframes swap { from { clip-path: circle(0px) } to { clip-path: inset(5px) } }
div { animation: swap 1s linear }`)
	tr2.Step(0.4)
	if got := ClipPathCSS(shown(d2).ClipPath); got != "circle(0px)" {
		t.Errorf("shapes of two kinds swap half way, not before: %s", got)
	}
	tr2.Step(0.2)
	if got := ClipPathCSS(shown(d2).ClipPath); got != "inset(5px)" {
		t.Errorf("past half way: %s", got)
	}
}

func TestAClipPathIsNoneUntilSet(t *testing.T) {
	if s := gridOf(t, "color: red"); !s.ClipPath.None() || s.ClipPath.Shape != style.ShapeNone {
		t.Errorf("initial clip-path %+v", s.ClipPath)
	}
}
