//go:build chrome

package css_test

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/css"
	"github.com/lantervale/ember/ui/internal/chrome"
	"github.com/lantervale/ember/ui/internal/dom"
)

// ask evaluates expr in a bare Chrome page and reads its value into v.
func ask(t *testing.T, size geom.Vec2, expr string, v any) {
	t.Helper()
	raw, err := chrome.Eval(ui.NewDocument(), size, expr)
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(raw, v); err != nil {
		t.Fatalf("%v in %s", err, raw)
	}
}

// TestChromeReadsWebkitPropertiesAsTheTable asks Chrome which plain
// property each -webkit- name of its style object sets, so the table
// can't drift from the Chrome the suite runs.
func TestChromeReadsWebkitPropertiesAsTheTable(t *testing.T) {
	var chromes map[string]string
	ask(t, geom.V2(800, 600), `
const el = document.createElement("div");
const hyphens = k => k.replace(/[A-Z]/g, c => "-" + c.toLowerCase());
const names = new Set();
for (const k in el.style) {
	if (/^[a-z]/.test(k) && !k.includes("-") && typeof el.style[k] === "string") names.add(hyphens(k));
}
const stored = name => {
	el.style.cssText = name + ": inherit";
	return Array.from(el.style).sort().join(" ");
};
const plain = new Map();
for (const n of names) {
	if (n.startsWith("webkit-")) continue;
	const s = stored(n);
	if (s && (!plain.has(s) || s === n)) plain.set(s, n);
}
const out = {};
for (const n of names) {
	if (!n.startsWith("webkit-")) continue;
	const s = stored("-" + n);
	out[n.slice(7)] = !s ? "(refused)" : s.includes("-webkit-") ? "(own)" : plain.get(s) || "(unknown)";
}
return out;`, &chromes)
	plain, own := css.WebkitTable()
	if len(chromes) < 100 {
		t.Fatalf("Chrome lists %d -webkit- properties", len(chromes))
	}
	stricter := []string{"column-break-after", "column-break-before", "column-break-inside", "mask-composite"}
	for _, name := range slices.Sorted(maps.Keys(chromes)) {
		as, why := chromes[name], own[name]
		switch {
		case plain[name] != "" && plain[name] != as:
			t.Errorf("-webkit-%s: Chrome reads it as %s, the table as %s", name, as, plain[name])
		case plain[name] != "":
		case why == "":
			t.Errorf("-webkit-%s: Chrome reads it as %s, and the table has no word of it", name, as)
		case as != "(own)" && !slices.Contains(stricter, name):
			t.Errorf("-webkit-%s: Chrome reads it as %s, the table as its own: %s", name, as, why)
		}
	}
	for name := range plain {
		if chromes[name] == "" {
			t.Errorf("-webkit-%s is in the table and not in Chrome", name)
		}
	}
	for name := range own {
		if chromes[name] == "" {
			t.Errorf("-webkit-%s is in the table as Chrome's own and not in Chrome", name)
		}
	}
	t.Logf("%d -webkit- properties in Chrome: %d read as a plain one, %d its own", len(chromes), len(plain), len(own))

	// Those four are another name of a plain property in Chrome, with
	// values the plain one refuses.
	var widened []bool
	ask(t, geom.V2(800, 600), `return [
	CSS.supports("-webkit-column-break-after", "always") && !CSS.supports("break-after", "always"),
	CSS.supports("-webkit-column-break-before", "always") && !CSS.supports("break-before", "always"),
	CSS.supports("-webkit-column-break-inside", "avoid") && CSS.supports("-webkit-column-break-inside", "auto"),
	CSS.supports("-webkit-mask-composite", "source-over") && !CSS.supports("mask-composite", "source-over"),
];`, &widened)
	if !widened[0] || !widened[1] || !widened[3] {
		t.Errorf("Chrome no longer reads other values under them: %v", widened)
	}
}

// TestChromeKnowsNoOtherVendorsNames checks what the -moz-, -ms-, -o-
// and -khtml- tables say: Chrome has no property, value, selector or
// at-rule of theirs.
func TestChromeKnowsNoOtherVendorsNames(t *testing.T) {
	for _, prefix := range css.VendorPrefixes() {
		if prefix == "-webkit-" {
			continue
		}
		var got struct {
			Properties []string
			Kept       []string
		}
		ask(t, geom.V2(800, 600), fmt.Sprintf(`
const x = %q, camel = x.slice(1, -1);
const el = document.createElement("div");
const properties = [];
for (const k in el.style) {
	if (k.toLowerCase().startsWith(camel) && /[A-Z]/.test(k[camel.length])) properties.push(k);
}
const kept = [];
for (const [prop, value] of [[x + "transform", "none"], [x + "box-sizing", "border-box"], [x + "opacity", "1"],
	["display", x + "box"], ["display", x + "flexbox"], ["width", x + "calc(1px + 2px)"],
	["width", x + "available"], ["background-image", x + "linear-gradient(top, red, blue)"]]) {
	if (CSS.supports(prop, value)) kept.push(prop + ": " + value);
}
const rules = src => { const s = document.createElement("style"); s.textContent = src;
	document.head.appendChild(s); const n = s.sheet.cssRules.length; s.remove(); return n; };
for (const src of ["button::" + x + "focus-inner { border: 0 }", "p, input:" + x + "placeholder { color: red }",
	"p:not(:" + x + "any-link) { color: red }", "@" + x + "keyframes spin { from { opacity: 0 } }",
	"@" + x + "document url-prefix() { p { color: red } }", "@" + x + "viewport { width: device-width }"]) {
	if (rules(src)) kept.push(src);
}
if (matchMedia("(" + x + "min-device-pixel-ratio: 0)").matches) kept.push("the media feature");
return {Properties: properties, Kept: kept};`, prefix), &got)
		if len(got.Properties) > 0 || len(got.Kept) > 0 {
			t.Errorf("%s: Chrome has the properties %v and keeps %v", prefix, got.Properties, got.Kept)
		}
	}
}

// TestChromeReadsWebkitValuesSelectorsAndAtRulesAsTheTable asks Chrome
// about the rest of -webkit-'s table.
func TestChromeReadsWebkitValuesSelectorsAndAtRulesAsTheTable(t *testing.T) {
	on := map[string]string{"calc": "width: -webkit-calc(1px + 2px)", "min-content": "width: -webkit-min-content",
		"max-content": "width: -webkit-max-content", "fit-content": "width: -webkit-fit-content",
		"flex": "display: -webkit-flex", "inline-flex": "display: -webkit-inline-flex", "grab": "cursor: -webkit-grab",
		"grabbing": "cursor: -webkit-grabbing"}
	var decls []string
	for _, name := range slices.Sorted(maps.Keys(css.WebkitValues())) {
		if on[name] == "" {
			t.Fatalf("-webkit-%s: the test has no declaration to try it in", name)
		}
		decls = append(decls, on[name])
	}
	// Chrome's own values, which keep their warning but text-align's, and one it refuses.
	decls = append(decls, "display: -webkit-box", "width: -webkit-fill-available", "text-align: -webkit-center",
		"background-image: -webkit-linear-gradient(top, red, blue)", "position: -webkit-sticky")
	selectors := []string{"input::-webkit-input-placeholder", "a:-webkit-any-link", "::-webkit-scrollbar",
		"::-webkit-search-decoration", "::-webkit-anything-at-all", "input:-webkit-autofill", "p:-webkit-drag",
		"p:-webkit-full-page-media", "p:-webkit-full-screen", "p:-webkit-full-screen-ancestor", "p:-webkit-any(a, b)",
		"p:-webkit-read-only", "p:-webkit-input-placeholder", "p:-webkit-anything-at-all"}
	atRules := []string{"@-webkit-keyframes spin { from { opacity: 0 } }", "@-webkit-viewport { width: device-width }",
		"@-webkit-region x { p { color: red } }", "@-webkit-supports (display: flex) { p { color: red } }"}
	var got struct {
		Decls, Selectors, AtRules []bool
		Placeholder, AnyLink      string
		Features                  []bool
	}
	asJSON := func(v any) string { b, _ := json.Marshal(v); return string(b) }
	ask(t, geom.V2(800, 600), `
const rules = src => { const s = document.createElement("style"); s.textContent = src;
	document.head.appendChild(s); const n = s.sheet.cssRules.length; s.remove(); return n > 0; };
const s = document.createElement("style");
s.textContent = "::-webkit-input-placeholder { color: rgb(1, 2, 3) } a:-webkit-any-link { color: rgb(4, 5, 6) }";
document.head.appendChild(s);
const input = document.createElement("input"), a = document.createElement("a");
input.placeholder = "x"; a.href = "#";
document.body.append(input, a);
return {
	Decls: `+asJSON(decls)+`.map(d => CSS.supports(d.split(": ")[0], d.split(": ")[1])),
	Selectors: `+asJSON(selectors)+`.map(sel => rules(sel + " { color: red }")),
	AtRules: `+asJSON(atRules)+`.map(rules),
	Placeholder: getComputedStyle(input, "::placeholder").color, AnyLink: getComputedStyle(a).color,
	Features: ["(-webkit-min-device-pixel-ratio: 0)", "(-webkit-max-device-pixel-ratio: 0.5)",
		"(-webkit-device-pixel-ratio: 1)", "(-webkit-transform-3d)", "(-webkit-animation)"].map(q => matchMedia(q).matches),
};`, &got)
	for i, d := range decls {
		if want := d != "position: -webkit-sticky"; got.Decls[i] != want {
			t.Errorf("%s: Chrome keeps it %v, want %v", d, got.Decls[i], want)
		}
	}
	if got.Placeholder != "rgb(1, 2, 3)" || got.AnyLink != "rgb(4, 5, 6)" {
		t.Errorf("::-webkit-input-placeholder colours %s, :-webkit-any-link %s", got.Placeholder, got.AnyLink)
	}
	for i, sel := range selectors {
		// Beside a selector ui matches, the rule stays where Chrome keeps
		// the vendor's selector and goes where Chrome drops the rule.
		src := sel + ", #probe { color: red }"
		if strings.Contains(sel, "placeholder") {
			// ::placeholder shares no rule in ui.
			src = sel + " { color: red }"
		}
		s, warns := css.Parse("t.css", src, css.Browser())
		ours := len(s.Rules) == 1
		if sel == "p:-webkit-any(a, b)" {
			if !got.Selectors[i] || len(warns) != 1 {
				t.Errorf("%s: Chrome keeps it %v, and ui warns %v", sel, got.Selectors[i], warns)
			}
			continue
		}
		if ours != got.Selectors[i] || len(warns) > 0 {
			t.Errorf("%s: Chrome keeps the rule %v, ui %v %v", sel, got.Selectors[i], ours, warns)
		}
	}
	for i, rule := range atRules {
		_, warns := css.Parse("t.css", rule, css.Browser())
		if got.AtRules[i] != (len(warns) > 0) {
			t.Errorf("%s: Chrome keeps it %v, ui warns %v", rule, got.AtRules[i], warns)
		}
	}
	if want := []bool{true, false, true, true, false}; !slices.Equal(got.Features, want) {
		t.Errorf("Chrome's -webkit- media features: %v, want %v", got.Features, want)
	}
}

// TestChromeAnswersMediaQueriesTheSame asks Chrome's matchMedia the
// queries of TestMediaQueriesAreAnsweredAsChromeAnswers.
func TestChromeAnswersMediaQueriesTheSame(t *testing.T) {
	cases := css.MediaCases()
	var queries []string
	for _, c := range cases {
		queries = append(queries, c.Query)
	}
	list, _ := json.Marshal(queries)
	m := css.ChromeViewport()
	var got struct {
		Width, Height float32
		Matches       []bool
	}
	ask(t, geom.V2(m.Width, m.Height),
		`return {Width: innerWidth, Height: innerHeight, Matches: `+string(list)+`.map(q => matchMedia(q).matches)};`,
		&got)
	if got.Width != m.Width || got.Height != m.Height {
		t.Fatalf("Chrome's viewport is %g by %g, not %g by %g", got.Width, got.Height, m.Width, m.Height)
	}
	for i, c := range cases {
		if got.Matches[i] != c.Want {
			t.Errorf("@media %s: Chrome %v, ui %v", c.Query, got.Matches[i], c.Want)
		}
	}
	t.Logf("%d queries answered as Chrome answers them at %g by %g", len(cases), m.Width, m.Height)
}

// TestChromeReadsTheQueriesItAsksTheMachine asks Chrome whether it reads
// the queries ui gives a fixed answer: Chrome's own answer is the
// machine's, its pointing devices and its user's preferences, so only
// the reading is compared (decision 0086). A query Chrome reads holds
// or its negation does; one it doesn't read is false both ways.
func TestChromeReadsTheQueriesItAsksTheMachine(t *testing.T) {
	const unknown = "(unknown-thing)"
	queries := append(css.MachineMediaQueries(), unknown)
	list, _ := json.Marshal(queries)
	var read []bool
	ask(t, geom.V2(800, 513),
		`return `+string(list)+`.map(q => matchMedia(q).matches !== matchMedia("not " + q).matches);`, &read)
	for i, q := range queries {
		if read[i] == (q == unknown) {
			t.Errorf("@media %s: Chrome reads it %v", q, read[i])
		}
	}
}

// TestChromeSupportsWhatUISupportsWhereBothKnowIt asks Chrome's
// CSS.supports conditions on what both read, or both don't.
func TestChromeSupportsWhatUISupportsWhereBothKnowIt(t *testing.T) {
	conds := []string{"(display: grid)", "(display:flex)", "((display: flex))", "(display: flex !important)",
		"(--x: y)", "(position: sticky)", "(-webkit-appearance: none)", "(-moz-appearance: none)", "(foo: bar)",
		"not (display: grid)", "not (foo: bar)", "(display: flex) and (foo: bar)", "(display: flex) or (foo: bar)",
		"not ((foo: bar) or (display: flex))", "(display: grid) and (not (display: inline-grid))",
		"selector(a > b)", "selector(::-moz-x)", "selector(::-webkit-foo)", "(display: flex) and (foo)",
		"(position: -webkit-sticky)", "(-webkit-touch-callout: none)", "(display: -ms-grid)", "unknown(x)"}
	list, _ := json.Marshal(conds)
	var got []bool
	ask(t, geom.V2(800, 600), `return `+string(list)+`.map(c => CSS.supports(c));`, &got)
	for i, c := range conds {
		if ours := css.Supported(c); ours != got[i] {
			t.Errorf("@supports %s: Chrome %v, ui %v", c, got[i], ours)
		}
	}
}

// TestChromeSizesTheSameBoxesByTheirBorder asks a page with Chrome's own
// sheet alone which elements it sizes by their border box.
func TestChromeSizesTheSameBoxesByTheirBorder(t *testing.T) {
	tags := []string{"div", "p", "span", "a", "button", "select", "option", "textarea", "table", "tr", "td", "th",
		"caption", "progress", "meter", "img", "hr", "fieldset", "legend", "details", "summary", "dialog", "ul", "li",
		"pre", "h1", "form", "label", "canvas"}
	types := []string{"text", "checkbox", "radio", "range", "button", "submit", "reset", "search", "color", "number",
		"password", "email"}
	var chromes []string
	tagList, _ := json.Marshal(tags)
	typeList, _ := json.Marshal(types)
	ask(t, geom.V2(800, 600), `
const frame = document.createElement("iframe");
document.body.appendChild(frame);
const doc = frame.contentDocument, out = [];
doc.open(); doc.write("<!DOCTYPE html><body>"); doc.close();
const sizing = (e, name) => { doc.body.appendChild(e);
	if (frame.contentWindow.getComputedStyle(e).boxSizing === "border-box") out.push(name); };
for (const tag of `+string(tagList)+`) sizing(doc.createElement(tag), tag);
for (const type of `+string(typeList)+`) { const e = doc.createElement("input"); e.type = type;
	sizing(e, 'input[type="' + type + '"]'); }
return out;`, &chromes)
	_, rule, _ := strings.Cut(css.BrowserDefault, "}")
	selectors, _, _ := strings.Cut(rule, "{")
	list := strings.Join(strings.Fields(selectors), " ")
	if want := strings.Join(chromes, ", "); list != want {
		t.Errorf("css.BrowserDefault sizes by their border\n     %s\nChrome %s", list, want)
	}
}

// TestChromeAlignsTheTextOfControlsAsABrowsersSheetDoes asks a page with
// Chrome's own sheet alone how each control in a center aligns its text.
func TestChromeAlignsTheTextOfControlsAsABrowsersSheetDoes(t *testing.T) {
	types := []string{"text", "search", "password", "number", "email", "button", "submit", "reset"}
	var chromes []string
	typeList, _ := json.Marshal(types)
	ask(t, geom.V2(800, 600), `
const frame = document.createElement("iframe");
document.body.appendChild(frame);
const doc = frame.contentDocument, out = [];
doc.open(); doc.write("<!DOCTYPE html><body><center id=c>"); doc.close();
const align = e => { doc.getElementById("c").appendChild(e);
	out.push(frame.contentWindow.getComputedStyle(e).textAlign); };
for (const type of `+string(typeList)+`) { const e = doc.createElement("input"); e.type = type; align(e); }
for (const tag of ["textarea", "select", "button"]) align(doc.createElement(tag));
return out;`, &chromes)
	d := ui.NewDocument(ui.BrowserSheets())
	center := ui.Center()
	for _, typ := range types {
		center.Append(ui.Input(ui.Type(typ)))
	}
	center.Append(ui.Textarea(), ui.Select(), ui.Button())
	d.Body().Append(center)
	d.Layout(geom.V2(800, 600))
	names := []string{"start", "end", "left", "right", "center", "justify", "-webkit-left", "-webkit-right",
		"-webkit-center"}
	for i, n := range center.Children() {
		if got := names[dom.Data(n).Style.TextAlign]; got != chromes[i] {
			typ, _ := n.Attr("type")
			t.Errorf("%s %s: ember %s, Chrome %s", n.Tag(), typ, got, chromes[i])
		}
	}
}

// TestChromeSpacesTheTextOfControlsAsABrowsersSheetDoes asks a page with
// Chrome's own sheet alone what each control keeps of a body's letter
// and word spacing, case and text shadow.
func TestChromeSpacesTheTextOfControlsAsABrowsersSheetDoes(t *testing.T) {
	types := []string{"text", "search", "password", "number", "email", "button", "submit", "reset", "checkbox"}
	const look = "font-size: 20px; word-spacing: 1em; letter-spacing: 2px; text-transform: uppercase; " +
		"text-shadow: 1px 1px red"
	var chromes [][4]string
	typeList, _ := json.Marshal(types)
	ask(t, geom.V2(800, 600), `
const frame = document.createElement("iframe");
document.body.appendChild(frame);
const doc = frame.contentDocument, out = [];
doc.open(); doc.write("<!DOCTYPE html><body style='`+look+`'>"); doc.close();
const read = e => { doc.body.appendChild(e); const s = frame.contentWindow.getComputedStyle(e);
	out.push([s.wordSpacing, s.letterSpacing, s.textTransform, s.textShadow]); };
for (const type of `+string(typeList)+`) { const e = doc.createElement("input"); e.type = type; read(e); }
for (const tag of ["textarea", "select", "button", "p"]) read(doc.createElement(tag));
return out;`, &chromes)
	d := ui.NewDocument(ui.BrowserSheets())
	d.Body().SetStyle(look)
	for _, typ := range types {
		d.Body().Append(ui.Input(ui.Type(typ)))
	}
	d.Body().Append(ui.Textarea(), ui.Select(), ui.Button(), ui.P())
	d.Layout(geom.V2(800, 600))
	px := func(v float32) string { return fmt.Sprintf("%gpx", v) }
	for i, n := range d.Body().Children() {
		s := dom.Data(n).Style
		got := [4]string{px(s.WordGap()), px(s.LetterSpacing), "none", "none"}
		if s.LetterSpacing == 0 {
			got[1] = "normal"
		}
		if s.TextTransform != 0 {
			got[2] = "uppercase"
		}
		if len(s.TextShadows) > 0 {
			got[3] = "rgb(255, 0, 0) 1px 1px 0px"
		}
		if got != chromes[i] {
			typ, _ := n.Attr("type")
			t.Errorf("%s %s: ember %v, Chrome %v", n.Tag(), typ, got, chromes[i])
		}
	}
}
