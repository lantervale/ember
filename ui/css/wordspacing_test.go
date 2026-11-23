package css //nolint:testpackage // uses the unexported test helpers (sheet, page) and reads computed styles off dom

import (
	"strings"
	"testing"

	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

func TestWordSpacingIsALengthAPercentageOrNormal(t *testing.T) {
	ids := []string{"px", "em", "rem", "neg", "pct", "calc", "normal", "zero"}
	var kids []*dom.Node
	for _, id := range ids {
		kids = append(kids, el(dom.P, id))
	}
	Compute(page(kids...), []*Sheet{sheet(t, `
html { font-size: 10px; }
body { word-spacing: 7px; }
p { font-size: 20px; }
#px { word-spacing: 3px; }
#em { word-spacing: 0.5em; }
#rem { word-spacing: 2rem; }
#neg { word-spacing: -4px; }
#pct { word-spacing: 50%; }
#calc { word-spacing: calc(25% + 1em - 2px); }
#normal { word-spacing: normal; }
#zero { word-spacing: 0; }
`)})
	want := map[string]style.Spacing{
		"px": {Px: 3}, "em": {Px: 10}, "rem": {Px: 20}, "neg": {Px: -4}, "pct": {Percent: 50},
		"calc": {Px: 18, Percent: 25}, "normal": {}, "zero": {},
	}
	for i, id := range ids {
		if got := styleOf(kids[i]).WordSpacing; got != want[id] {
			t.Errorf("#%s: %+v, want %+v", id, got, want[id])
		}
	}
}

func TestWordSpacingIsInheritedAsItsLengthAndItsPercentage(t *testing.T) {
	em, pct, reset := el(dom.Span, "em"), el(dom.Span, "pct"), el(dom.Span, "reset")
	outer := with(el(dom.Div, "out"), em, with(el(dom.Div, "half"), pct), reset)
	Compute(page(outer), []*Sheet{sheet(t, `
#out { font-size: 10px; word-spacing: 2em; }
#half { word-spacing: 50%; }
span { font-size: 40px; }
#reset { word-spacing: initial; }
`)})
	if s := styleOf(em); s.WordSpacing != (style.Spacing{Px: 20}) || s.WordGap() != 20 {
		t.Errorf("the parent's ems, not the child's: %+v", s.WordSpacing)
	}
	if s := styleOf(pct); s.WordSpacing != (style.Spacing{Percent: 50}) || s.WordGap() != 20 {
		t.Errorf("a percentage of the child's own font size: %+v, %v", s.WordSpacing, s.WordGap())
	}
	if s := styleOf(reset); s.WordSpacing != (style.Spacing{}) {
		t.Errorf("initial is normal: %+v", s.WordSpacing)
	}
}

func TestLetterSpacingTakesAPercentageOfEachInheritorsFontSize(t *testing.T) {
	pct, sum := el(dom.Span, "pct"), el(dom.Span, "sum")
	Compute(page(with(el(dom.Div, "out"), pct, sum)), []*Sheet{sheet(t, `
#out { font-size: 10px; letter-spacing: 10%; }
span { font-size: 40px; }
#sum { letter-spacing: calc(5% + 1px); }
`)})
	if s := styleOf(pct); s.LetterSpacing != (style.Spacing{Percent: 10}) || s.LetterGap() != 4 {
		t.Errorf("a percentage of the child's own font size: %+v, %v", s.LetterSpacing, s.LetterGap())
	}
	if s := styleOf(sum); s.LetterGap() != 3 {
		t.Errorf("calc(5%% + 1px) of 40px: %v", s.LetterGap())
	}
}

func TestWordSpacingRefusesWhatIsNoLength(t *testing.T) {
	for _, v := range []string{"5", "wide", "1px 2px", "1fr"} {
		if _, warns := Parse("t.css", "p { word-spacing: "+v+" }"); len(warns) == 0 {
			t.Errorf("word-spacing: %s is taken", v)
		}
	}
}

func TestAMarkerTakesWordSpacing(t *testing.T) {
	li := el(dom.Li, "li")
	warns := Compute(page(with(el(dom.Ol, ""), li)), []*Sheet{sheet(t, `li::marker { word-spacing: 4px; }`)})
	if len(warns) > 0 {
		t.Fatal(warnings(warns))
	}
	if m := dom.Data(li).Marker; m == nil || m.WordSpacing.Px != 4 {
		t.Errorf("marker: %+v", m)
	}
}

func TestControlsTakeThePagesWordSpacingAsItsLetterSpacing(t *testing.T) {
	if !strings.Contains(Default, "word-spacing: inherit") {
		t.Fatal("the agent's sheet leaves the controls' word-spacing out of what they inherit")
	}
	in := el(dom.Input, "in")
	sheets := []*Sheet{sheet(t, Default), sheet(t, `body { word-spacing: 6px; }`)}
	Compute(page(in), sheets)
	if got := styleOf(in).WordSpacing; got != (style.Spacing{Px: 6}) {
		t.Errorf("input: %+v", got)
	}
}
