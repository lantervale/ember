package css //nolint:testpackage // builds trees with the package's test helpers

import (
	"testing"

	"github.com/lantervale/ember/ui/internal/dom"
)

func TestTheStatesOfFieldsMatchTheirPseudoClasses(t *testing.T) {
	field := el(dom.Input, "field")
	page(field)
	for name, st := range map[string]dom.State{
		":placeholder-shown": dom.PlaceholderShown, ":default": dom.Default, ":valid": dom.Valid,
		":invalid": dom.Invalid, ":user-valid": dom.UserValid, ":user-invalid": dom.UserInvalid,
		":in-range": dom.InRange, ":out-of-range": dom.OutOfRange, ":indeterminate": dom.Indeterminate,
	} {
		sel := mustSelector(t, name)
		if sel.Matches(field) {
			t.Errorf("%s matches before the state", name)
		}
		field.SetState(st, true)
		if !sel.Matches(field) {
			t.Errorf("%s does not match in the state", name)
		}
		field.SetState(st, false)
	}
}

func TestRequiredAndOptionalReadTheAttributeWhereItApplies(t *testing.T) {
	text := attrs(el(dom.Input, "text"), "required", "")
	rng := attrs(el(dom.Input, "range"), "type", "RANGE", "required", "")
	hidden := attrs(el(dom.Input, "hidden"), "type", "hidden")
	area := attrs(el(dom.Textarea, "area"), "required", "")
	sel := el(dom.Select, "sel")
	btn := attrs(el(dom.Button, "btn"), "required", "")
	div := attrs(el(dom.Div, "div"), "required", "")
	page(text, rng, hidden, area, sel, btn, div)
	for _, c := range []struct {
		e                  *dom.Node
		required, optional bool
	}{
		{text, true, false}, {rng, false, true}, {hidden, false, true}, {area, true, false}, {sel, false, true},
		{btn, false, true}, {div, false, false},
	} {
		if got := mustSelector(t, ":required").Matches(c.e); got != c.required {
			t.Errorf(":required on #%s: %v", c.e.ID(), got)
		}
		if got := mustSelector(t, ":optional").Matches(c.e); got != c.optional {
			t.Errorf(":optional on #%s: %v", c.e.ID(), got)
		}
	}
}

func TestIndeterminateTakesAProgressWithoutValue(t *testing.T) {
	waiting, done := el(dom.Progress, "waiting"), attrs(el(dom.Progress, "done"), "value", "1")
	page(waiting, done)
	sel := mustSelector(t, "progress:indeterminate")
	if !sel.Matches(waiting) || sel.Matches(done) {
		t.Errorf("waiting %v, done %v", sel.Matches(waiting), sel.Matches(done))
	}
}

func TestDirFollowsTheDirAttributeAndTheText(t *testing.T) {
	inner := el(dom.Span, "inner")
	hebrew := with(attrs(el(dom.Div, "hebrew"), "dir", "auto"), with(el(dom.Span, ""), dom.NewText("12 ")),
		dom.NewText("שלום"))
	skipped := with(attrs(el(dom.P, "skipped"), "dir", "auto"), with(el(dom.Bdi, ""), dom.NewText("abc")),
		dom.NewText("שלום"))
	digits := with(attrs(el(dom.P, "digits"), "dir", "auto"), dom.NewText("42"))
	bogus := attrs(el(dom.Div, "bogus"), "dir", "up")
	rtl := with(attrs(el(dom.Div, "rtl"), "dir", "RTL"), inner, hebrew, skipped, digits, bogus)
	plain := el(dom.Div, "plain")
	page(rtl, plain)
	for _, c := range []struct {
		e   *dom.Node
		rtl bool
	}{
		{rtl, true}, {inner, true}, {hebrew, true}, {skipped, true}, {digits, false}, {bogus, true}, {plain, false},
	} {
		if got := mustSelector(t, ":dir(rtl)").Matches(c.e); got != c.rtl {
			t.Errorf(":dir(rtl) on #%s: %v", c.e.ID(), got)
		}
		if got := mustSelector(t, ":dir(ltr)").Matches(c.e); got == c.rtl {
			t.Errorf(":dir(ltr) on #%s: %v", c.e.ID(), got)
		}
	}
	if mustSelector(t, ":dir(up)").Matches(bogus) {
		t.Error(":dir(up) matches")
	}
}

func TestAutofillMatchesNothing(t *testing.T) {
	field := el(dom.Input, "field")
	page(field)
	for _, src := range []string{":autofill", ":-webkit-autofill", "input:not(:autofill)"} {
		if got := mustSelector(t, src).Matches(field); got != (src == "input:not(:autofill)") {
			t.Errorf("%s: %v", src, got)
		}
	}
}

func TestEnumeratedAttributesCompareWhateverTheCase(t *testing.T) {
	box := attrs(el(dom.Input, "box"), "type", "CheckBox", "title", "Hi")
	link := attrs(el(dom.A, "link"), "rel", "NoFollow", "lang", "EN-us")
	pic := attrs(el(dom.Svg, "pic"), "type", "Logo")
	page(box, link, pic)
	for _, c := range []struct {
		src  string
		e    *dom.Node
		want bool
	}{
		{"[type=checkbox]", box, true},
		{"[type^=check]", box, true},
		{"[title=hi]", box, false},
		{"[title=hi i]", box, true},
		{"[rel=nofollow]", link, true},
		{"[lang|=en]", link, true},
		{"[type=logo]", pic, false},
	} {
		if got := mustSelector(t, c.src).Matches(c.e); got != c.want {
			t.Errorf("%s on #%s: %v", c.src, c.e.ID(), got)
		}
	}
}

func TestTheFormPseudoClassesWeighAsAClass(t *testing.T) {
	for _, src := range []string{":valid", ":invalid", ":user-valid", ":user-invalid", ":required", ":optional",
		":in-range", ":out-of-range", ":placeholder-shown", ":default", ":indeterminate", ":dir(rtl)", ":autofill"} {
		if got := mustSelector(t, src).Specificity; got != (Specificity{0, 1, 0}) {
			t.Errorf("%s weighs %v", src, got)
		}
	}
	for _, src := range []string{":dir()", ":dir(ltr, rtl)", ":dir(1)", `:dir("rtl")`} {
		if _, err := ParseSelectors(src); err == nil {
			t.Errorf("%q: no error", src)
		}
	}
}
