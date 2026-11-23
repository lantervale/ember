//go:build js

package web

import (
	"syscall/js"

	"github.com/lantervale/ember/internal/sysprefs"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/world"
)

// pagePrefs keeps the World's preferences as the browser has them: its
// media queries answer what the user prefers and say when it changes, as
// languagechange says the languages did.
type pagePrefs struct {
	ls      listeners
	changed bool
	system  sysprefs.System
}

// preferenceQueries are the media queries whose change has the
// preferences read again.
//
//nolint:gochecknoglobals // a fixed list
var preferenceQueries = []string{
	"(prefers-color-scheme: dark)", "(prefers-reduced-motion: reduce)", "(prefers-contrast: more)",
	"(prefers-contrast: less)", "(prefers-contrast: custom)", "(forced-colors: active)",
}

// watchPreferences listens for the browser's changes; the func it gives
// removes and releases its handlers.
func watchPreferences() (*pagePrefs, func()) {
	p := &pagePrefs{changed: true}
	changed := func(js.Value) { p.changed = true }
	for _, q := range preferenceQueries {
		p.ls.on(js.Global().Call("matchMedia", q), "change", changed)
	}
	p.ls.on(js.Global(), "languagechange", changed)
	return p, p.ls.release
}

// sync gives every preferences of the World what the browser last said;
// nil pagePrefs, a capture's, give nothing.
func (p *pagePrefs) sync(prefs *world.Group[platform.Preferred]) {
	if p == nil {
		return
	}
	if p.changed {
		p.changed, p.system = false, pageSystem()
	}
	s := p.system
	for e := range prefs.All() {
		pr := e.GetPreferences()
		pr.SyncAppearance(s.Appearance)
		pr.SyncMotion(s.Motion)
		pr.SyncContrast(s.Contrast)
		pr.SyncForcedColors(s.ForcedColors)
		pr.SyncHighlight(s.Highlight)
		pr.SyncLanguages(s.Languages...)
	}
}

// pageSystem is what the browser says: its media queries' answers, the
// colour it gives Highlight, and navigator.languages.
func pageSystem() sysprefs.System {
	var s sysprefs.System
	matches := func(q string) bool { return js.Global().Call("matchMedia", q).Get("matches").Truthy() }
	switch {
	case matches("(prefers-color-scheme: dark)"):
		s.Appearance = platform.AppearanceDark
	case matches("(prefers-color-scheme: light)"):
		s.Appearance = platform.AppearanceLight
	}
	switch {
	case matches("(prefers-reduced-motion: reduce)"):
		s.Motion = platform.MotionReduced
	case matches("(prefers-reduced-motion: no-preference)"):
		s.Motion = platform.MotionFull
	}
	for word, c := range map[string]platform.Contrast{
		"no-preference": platform.ContrastNoPreference, "more": platform.ContrastMore,
		"less": platform.ContrastLess, "custom": platform.ContrastCustom,
	} {
		if matches("(prefers-contrast: " + word + ")") {
			s.Contrast = c
		}
	}
	switch {
	case matches("(forced-colors: active)"):
		s.ForcedColors = platform.ForcedColorsActive
	case matches("(forced-colors: none)"):
		s.ForcedColors = platform.ForcedColorsNone
	}
	s.Highlight.Background, _ = sysprefs.ComputedColor(systemColor("Highlight"))
	languages := js.Global().Get("navigator").Get("languages")
	for i := range languages.Length() {
		s.Languages = append(s.Languages, languages.Index(i).String())
	}
	return s
}

// systemColor is the colour the browser computes for a CSS system colour,
// read off an element shown nowhere.
func systemColor(name string) string {
	doc := js.Global().Get("document")
	probe := doc.Call("createElement", "span")
	probe.Get("style").Set("display", "none")
	probe.Get("style").Set("color", name)
	doc.Get("body").Call("appendChild", probe)
	defer probe.Call("remove")
	return js.Global().Call("getComputedStyle", probe).Get("color").String()
}
