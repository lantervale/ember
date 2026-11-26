//go:build chrome

package css_test

import (
	"fmt"
	"testing"
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/css"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

//nolint:gochecknoglobals // what each case's class holds, a test table
var clipPaths = []string{
	"none", "inset(10px)", "inset(10px 20px 10px 20px round 4px 4px)",
	"inset(1px 2px 3px 4px round 1px 2px 3px 4px / 5px 6px)", "inset(10% round 50%)", "inset(-10px)",
	"inset(10px round 0 / 5px)", "circle()", "circle(40px)", "circle(closest-side)",
	"circle(farthest-side at left top)", "circle(at 10px 20px)", "circle(at center)", "circle(at right)",
	"circle(at left 5px bottom 6px)", "circle(calc(10px + 5%))", "circle(1em)", "ellipse()", "ellipse(10px 20%)",
	"ellipse(closest-side farthest-side at 30% 40%)", "ellipse(closest-side closest-side)", "ellipse(at top)",
	"polygon(0 0, 100% 0, 50% 100%)", "polygon(evenodd, 0 0, 100% 0, 50% 100%)",
	"polygon(nonzero, 0 0, 10px 0, 5px 10px)", `path("M0 0 L 10 10 Z")`, `path(evenodd, "M 0,0 l 10 10 h 5 z")`,
	`path("m 0 0 c 1 2 3 4 5 6 a 1 1 0 0 1 2 2 s 1 1 2 2 q 3 3 4 4 t 1 1 v 5 H 0 z m 4 4 l 1 1")`,
	"xywh(10px 20px 30px 40px)", "xywh(10px 20% 30% 40px round 5px)", "rect(10px 20px 30px 40px)",
	"rect(auto auto auto auto)", "rect(10px auto 30% auto round 2px)", "padding-box", "border-box",
	"content-box", "margin-box", "fill-box", "stroke-box", "view-box", "circle() padding-box",
	"border-box circle(10px)", "inset(0) border-box", "content-box polygon(0 0, 1px 1px, 2px 0)",
}

func TestClipPathsComputeAsInChrome(t *testing.T) {
	src := "div { width: 100px; height: 50px; font-size: 10px; }\n"
	for i, c := range clipPaths {
		src += fmt.Sprintf(".c%d { clip-path: %s; }\n", i, c)
	}
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"c.css": {Data: []byte(src)}}, "c.css"))
	for i := range clipPaths {
		d.Body().Append(ui.Div(ui.Class(fmt.Sprint("c", i))))
	}
	matchChromeProps(t, d, map[string]func(*style.Computed) string{
		"clip-path": func(s *style.Computed) string { return css.ClipPathCSS(s.ClipPath) },
	}, 2)
}

func TestClipComputesAsInChrome(t *testing.T) {
	clips := []string{"auto", "rect(0, 50px, 50px, 0)", "rect(0px 100px 100px 0px)", "rect( 0 auto auto 0 )",
		"rect(-50px, auto, auto, -50px)", "rect(1em, 2em, 3em, 0.75em)", "rect(1in, 72pt, 6pc, 2.54cm)",
		"rect(auto auto auto auto)"}
	src := "div { position: absolute; font-size: 10px; }\n"
	for i, c := range clips {
		src += fmt.Sprintf(".c%d { clip: %s; }\n", i, c)
	}
	src += ".in { clip: inherit; }\n"
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"c.css": {Data: []byte(src)}}, "c.css"))
	for i := range clips {
		d.Body().Append(ui.Div(ui.Class(fmt.Sprint("c", i)), ui.Div(ui.Class("in"))))
	}
	matchChromeProps(t, d, map[string]func(*style.Computed) string{
		"clip": func(s *style.Computed) string { return css.ClipCSS(s.Clip) },
	}, 2)
}

func TestClipPathTransitionsAndAnimationsMatchChrome(t *testing.T) {
	src := `
@keyframes reveal { from { clip-path: circle(0px at 0% 0%) } to { clip-path: circle(80px at 100% 50%) } }
@keyframes swap { from { clip-path: inset(0) } to { clip-path: circle(10px) } }
div { width: 100px; height: 50px; transition: clip-path 1s linear; }
.a { clip-path: inset(0 round 0); }
.a.on { clip-path: inset(20px 10% 5px round 8px 2px / 4px); }
.b { clip-path: ellipse(10px 20px at 10% 10%); }
.b.on { clip-path: ellipse(30px 40% at 50% 80%); }
.c { clip-path: polygon(0 0, 10px 0, 0 10px); }
.c.on { clip-path: polygon(10px 10px, 30px 0, 0 50%); }
.d { clip-path: circle(10px); }
.d.on { clip-path: inset(10px); }
.e { clip-path: circle(10px) padding-box; }
.e.on { clip-path: circle(20px); }
.f { clip-path: polygon(evenodd, 0 0, 1px 0, 0 1px); }
.f.on { clip-path: polygon(0 0, 3px 0, 0 3px); }
.g { transition: none; animation: reveal 1s linear; }
.h { transition: none; animation: swap 1s linear; }
`
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"t.css": {Data: []byte(src)}}, "t.css"))
	for _, c := range []string{"a", "b", "c", "d", "e", "f", "g", "h"} {
		d.Body().Append(ui.Div(ui.Class(c)))
	}
	tr := css.NewTransitions()
	tr.Compute(d.HTML(), d.Sheets())
	script := holdAtStart + `els.forEach(e => getComputedStyle(e).clipPath);
	document.querySelectorAll("div").forEach(e => e.classList.add("on"));
	els.forEach(e => getComputedStyle(e).clipPath);
	document.getAnimations().forEach(a => { a.pause(); a.currentTime = 350; });`
	then := func() {
		d.HTML().Walk(func(n *dom.Node) bool {
			if n.Tag() == dom.Div {
				n.AddClass("on")
			}
			return true
		})
		tr.Compute(d.HTML(), d.Sheets())
		tr.Step(0.35)
		tr.Compute(d.HTML(), d.Sheets())
	}
	matchChromeShown(t, d, map[string]func(*style.Computed) string{
		"clip-path": func(s *style.Computed) string { return css.ClipPathCSS(s.ClipPath) },
	}, 2, script, then, func(n *dom.Node) *style.Computed { return dom.Data(n).Shown })
}
