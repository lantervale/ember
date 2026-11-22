package css //nolint:testpackage // reads the vendors' tables and the package's unexported test helpers

import (
	"fmt"
	"maps"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

// asBrowser parses src as a browser's sheet and returns it with its
// warnings as text.
func asBrowser(src string) (*Sheet, string) {
	s, warns := Parse("t.css", src, Browser())
	return s, warnings(warns)
}

// read is what a sheet keeps of one declaration and says of it.
func read(decl string, opts ...ParseOption) string {
	s, warns := Parse("t.css", "p { "+decl+" }", opts...)
	var out []string
	if len(s.Rules) == 1 {
		for _, d := range s.Rules[0].Decls {
			out = append(out, d.String())
		}
	}
	return strings.Join(append(out, warnings(warns)), " | ")
}

func TestWebkitPropertiesAreReadAsTheirPlainOnes(t *testing.T) {
	if len(webkit.same) != 74 || len(webkit.renamed) != 30 || len(webkit.own) != 45 || len(webkit.legacy) != 2 {
		t.Errorf("-webkit-: %d of the same name, %d renamed, %d its own, %d its own read as plain ones",
			len(webkit.same), len(webkit.renamed), len(webkit.own), len(webkit.legacy))
	}
	plain := map[string]string{}
	for _, name := range webkit.same {
		plain[name] = name
	}
	for name := range webkit.legacy {
		plain[name] = name
	}
	for name, as := range webkit.renamed {
		plain[name] = as
	}
	for name, as := range plain {
		if _, both := webkit.own[name]; both {
			t.Errorf("-webkit-%s is both read as %s and Chrome's own", name, as)
		}
		got := read("-webkit-"+name+": inherit", Browser())
		want := strings.ReplaceAll(read(as+": inherit", Browser()), "t.css:1:5: ", "t.css:1:5: -webkit-"+name+": ")
		if got != want {
			t.Errorf("-webkit-%s\n got %s\nwant %s, as %s is read", name, got, want, as)
		}
	}
	if got := read("-WebKit-Transform: rotate(90deg)", Browser()); got != "transform: rotate(90deg) | " {
		t.Errorf("in any case: %s", got)
	}
}

func TestWebkitsOwnPropertiesStayAsWritten(t *testing.T) {
	for name, why := range webkit.own {
		if why == "" {
			t.Errorf("-webkit-%s says no why", name)
		}
		got, strict := read("-webkit-"+name+": inherit", Browser()), read("-webkit-"+name+": inherit")
		if got != strict {
			t.Errorf("-webkit-%s\n got %s\nwant %s, as without the option", name, got, strict)
		}
	}
	if got := read("-webkit-font-smoothing: none", Browser()); got != "-webkit-font-smoothing: none | " {
		t.Errorf("ui's own: %s", got)
	}
	if got := read("-webkit-line-clamp: 2", Browser()); !strings.Contains(got, `"-webkit-line-clamp" is not supported`) {
		t.Errorf("Chrome's own keeps its warning: %s", got)
	}
}

func TestWebkitPropertiesChromeDoesNotKnowGo(t *testing.T) {
	for _, decl := range []string{"-webkit-touch-callout: none", "-webkit-overflow-scrolling: touch",
		"-webkit-hyphens: auto"} {
		if got := read(decl, Browser()); got != " | " && got != "" {
			t.Errorf("%s: %q, want nothing and no warning", decl, got)
		}
	}
}

func TestAnAliasKeepsItsPlaceInTheCascade(t *testing.T) {
	turned := func(src string) bool {
		p := el(dom.P, "p")
		s, warns := asBrowser(src)
		if warns != "" {
			t.Fatal(warns)
		}
		Compute(page(p), []*Sheet{s})
		return len(styleOf(p).Transform) > 0
	}
	for src, want := range map[string]bool{
		"p { -webkit-transform: rotate(90deg) }":                                                              true,
		"p { -webkit-transform: rotate(90deg); transform: none }":                                             false,
		"p { transform: none; -webkit-transform: rotate(90deg) }":                                             true,
		"p { transform: none !important; -webkit-transform: rotate(90deg) }":                                  false,
		"p { -webkit-transform: rotate(90deg) !important; transform: none }":                                  true,
		"#p { transform: none } p { -webkit-transform: rotate(90deg) }":                                       false,
		"p { -webkit-transform: rotate(90deg) } p { -moz-transform: none }":                                   true,
		"p { transform: none } @supports (-webkit-transform: none) { p { -webkit-transform: rotate(9deg) } }": true,
	} {
		if got := turned(src); got != want {
			t.Errorf("%s: turned %v, want %v", src, got, want)
		}
	}
}

func TestWebkitValues(t *testing.T) {
	for decl, want := range map[string]string{
		"width: -webkit-calc(1px + 2px)":   "width: calc(1px + 2px) | ",
		"width: -webkit-min-content":       "width: min-content | ",
		"height: -webkit-max-content":      "height: max-content | ",
		"max-width: -webkit-fit-content":   "max-width: fit-content | ",
		"display: -webkit-flex":            "display: flex | ",
		"display: -WebKit-Inline-Flex":     "display: inline-flex | ",
		"cursor: -webkit-grab":             "cursor: grab | ",
		"cursor: -webkit-grabbing":         "cursor: grabbing | ",
		"transition: -webkit-transform 1s": "transition: transform 1s | ",
		"--x: -webkit-flex":                "--x: -webkit-flex | ",
		"text-align: -webkit-center":       "text-align: -webkit-center | ",
	} {
		if got := read(decl, Browser()); got != want {
			t.Errorf("%s\n got %s\nwant %s", decl, got, want)
		}
	}
	for name := range webkit.values {
		if _, known := webkit.values["-webkit-"+name]; known || strings.HasPrefix(name, "-") {
			t.Errorf("%s: values are written without the prefix", name)
		}
	}
	// Chrome's own values keep their warning.
	for _, decl := range []string{"display: -webkit-box", "width: -webkit-fill-available",
		"background-image: -webkit-linear-gradient(top, #fff, #000)"} {
		got, strict := read(decl, Browser()), read(decl)
		if got != strict || !strings.Contains(got, "t.css:1:") {
			t.Errorf("%s\n got %s\nwant %s, as without the option", decl, got, strict)
		}
	}
}

func TestWebkitSelectors(t *testing.T) {
	for src, want := range map[string]string{
		"input::-webkit-input-placeholder { color: #f00 }":         "input::placeholder { color: #f00; }\n",
		"a:-webkit-any-link { color: #f00 }":                       "a:any-link { color: #f00; }\n",
		"::-webkit-scrollbar { width: 8px }":                       "",
		"p, ::-webkit-scrollbar-thumb:vertical, a { color: #f00 }": "p, a { color: #f00; }\n",
		"::-webkit-search-decoration, p { color: #f00 }":           "p { color: #f00; }\n",
		"input:-webkit-autofill, p { color: #f00 }":                "p { color: #f00; }\n",
		"p:-webkit-full-screen { color: #f00 }":                    "",
		"p:-webkit-read-only, a { color: #f00 }":                   "",
	} {
		s, warns := asBrowser(src)
		if got := plainRules(s); got != want || warns != "" {
			t.Errorf("%s\n got %q %s\nwant %q and no warning", src, got, warns, want)
		}
	}
	// :-webkit-any() is Chrome's: :is() with another weight, so not read as it.
	if _, warns := asBrowser("p:-webkit-any(a, b) { color: #f00 }"); !strings.Contains(warns, ":-webkit-any()") {
		t.Errorf(":-webkit-any(): %q", warns)
	}
}

// plainRules writes s's rules as a sheet not read as a browser's does.
func plainRules(s *Sheet) string {
	return (&Sheet{Rules: s.Rules, media: s.media}).String()
}

func TestWebkitAtRules(t *testing.T) {
	s, warns := asBrowser("@-webkit-keyframes spin { from { opacity: 0 } } p { color: #f00 }")
	if warns != "" || len(s.Rules) != 1 || len(s.Keyframes) != 1 || s.Keyframes[0].Name != "spin" {
		t.Errorf("@-webkit-keyframes is @keyframes: %q, %d rules, %+v", warns, len(s.Rules), s.Keyframes)
	}
	for _, src := range []string{"@-webkit-viewport { width: device-width }",
		"@-webkit-region x { p { color: #f00 } }", "@-webkit-supports (display: flex) { p { color: #f00 } }"} {
		if s, warns = asBrowser(src + " a { color: #f00 }"); warns != "" || plainRules(s) != "a { color: #f00; }\n" {
			t.Errorf("%s: %q %q, want it gone without a word", src, plainRules(s), warns)
		}
	}
}

// foreign checks a vendor Chrome reads nothing of: every kind of name
// goes without a word, and what stands beside it stays.
func foreign(t *testing.T, v *vendor) {
	t.Helper()
	if v.properties != dropped || v.otherValues != dropped || v.otherElements != dropped ||
		v.otherClasses != dropped || v.otherAtRules != dropped {
		t.Errorf("%s: %+v", v.prefix, v)
	}
	if len(v.same)+len(v.renamed)+len(v.own)+len(v.values)+len(v.elements)+len(v.classes)+len(v.never)+
		len(v.ownClasses)+len(v.atRules)+len(v.features)+len(v.ownFeatures) != 0 {
		t.Errorf("%s reads names Chrome doesn't: %+v", v.prefix, v)
	}
	x := v.prefix
	for src, want := range map[string]string{
		"p { " + x + "transform: rotate(90deg); color: #f00 }":         "p { color: #f00; }\n",
		"p { " + x + "box-sizing: border-box; " + x + "foo: 1 }":       "p { }\n",
		"p { display: " + x + "box; color: #f00 }":                     "p { color: #f00; }\n",
		"p { width: " + x + "calc(1px + 2px); color: #f00 }":           "p { color: #f00; }\n",
		"p { background: " + x + "linear-gradient(top, #fff, #000) }":  "p { }\n",
		"p { --x: " + x + "box }":                                      "p { --x: " + x + "box; }\n",
		"button::" + x + "focus-inner { border: 0 } a { color: #f00 }": "a { color: #f00; }\n",
		"p, input:" + x + "placeholder { color: #f00 }":                "",
		"p:not(:" + x + "any-link) { color: #f00 }":                    "",
		"@" + x + "keyframes spin { from { opacity: 0 } }":             "",
		"@" + x + "document url-prefix() { p { color: #f00 } }":        "",
		"@" + x + "viewport { width: device-width } p { color: #f00 }": "p { color: #f00; }\n",
		"@media (" + x + "min-device-pixel-ratio: 2) { p { color: #f00 } }": "@media (" + x +
			"min-device-pixel-ratio: 2) { p { color: #f00; } }\n",
	} {
		s, warns := asBrowser(src)
		if got := plainRules(s); got != want || warns != "" {
			t.Errorf("%s\n got %q %s\nwant %q and no warning", src, got, warns, want)
		}
		if _, strict := Parse("t.css", src); len(strict) == 0 && !strings.Contains(src, "--x") {
			t.Errorf("%s: no warning without the option", src)
		}
	}
	s, _ := asBrowser("@media (" + x + "min-device-pixel-ratio: 0) { p { color: #f00 } }")
	if s.media[0].matches(Media{Width: 800, Height: 600}) {
		t.Errorf("a query on a %s feature holds", x)
	}
	decls, warns := ParseInline(x+"transform: none; color: #f00", Browser())
	if len(decls) != 1 || decls[0].Name != "color" || len(warns) != 0 {
		t.Errorf("a style attribute: %v %s", decls, warnings(warns))
	}
}

func TestMozNamesGoAsChromeDropsThem(t *testing.T)   { foreign(t, &moz) }
func TestMsNamesGoAsChromeDropsThem(t *testing.T)    { foreign(t, &ms) }
func TestOperaNamesGoAsChromeDropsThem(t *testing.T) { foreign(t, &opera) }
func TestKhtmlNamesGoAsChromeDropsThem(t *testing.T) { foreign(t, &khtml) }

func TestAnotherVendorsValueGoesOnlyWhereUIRefusesTheDeclaration(t *testing.T) {
	if got := read("font-family: -moz-fixed, monospace", Browser()); got != "font-family: -moz-fixed, monospace | " {
		t.Errorf("a family is any name: %s", got)
	}
}

func TestEveryVendorHasItsTable(t *testing.T) {
	var prefixes []string
	for _, v := range vendors {
		prefixes = append(prefixes, v.prefix)
	}
	if got := strings.Join(prefixes, " "); got != "-webkit- -moz- -ms- -o- -khtml-" {
		t.Errorf("vendors: %s", got)
	}
	// A prefix ui has no table for keeps its warning.
	if got := read("-apple-trailing-word: inline", Browser()); !strings.Contains(got, "vendor-prefixed property") {
		t.Errorf("-apple-: %s", got)
	}
}

func TestAStyleAttributeIsReadAsABrowsers(t *testing.T) {
	p := el(dom.P, "p")
	p.SetStyle("-webkit-transform: rotate(90deg); -moz-opacity: 0.5; display: -webkit-flex")
	tr := NewTransitions()
	tr.BrowserSheets()
	if warns := tr.Compute(page(p), nil); len(warns) > 0 {
		t.Errorf("warnings: %s", warnings(warns))
	}
	if s := styleOf(p); len(s.Transform) != 1 || s.Display != style.FlexDisplay {
		t.Errorf("transform %v, display %v", s.Transform, s.Display)
	}
	q := el(dom.P, "q")
	q.SetStyle("-webkit-transform: rotate(90deg)")
	if warns := NewTransitions().Compute(page(q), nil); len(warns) != 1 {
		t.Errorf("without the option: %s", warnings(warns))
	}
}

// vendorGuide is the guide's section for v, as docs/05-ui/03-css.md
// must hold it.
func vendorGuide(v *vendor) string {
	var b strings.Builder
	code := func(names []string, prefix string) string {
		out := make([]string, len(names))
		for i, n := range names {
			out[i] = "`" + prefix + n + "`"
		}
		return strings.Join(out, " ")
	}
	keys := func(m map[string]string) []string {
		var out []string
		for k := range m {
			out = append(out, k)
		}
		slices.Sort(out)
		return out
	}
	pairs := func(m map[string]string, from, to string) string {
		var out []string
		for _, k := range keys(m) {
			out = append(out, "`"+from+k+"` as `"+to+m[k]+"`")
		}
		return strings.Join(out, ", ")
	}
	others := map[treatment]string{
		warned:    "keeps its warning",
		dropped:   "goes without a word, as Chrome drops it",
		unmatched: "matches nothing: its selector alone goes, without a word",
	}
	fmt.Fprintf(&b, "#### `%s`\n\n| | |\n|---|---|\n", v.prefix)
	if len(v.same) > 0 {
		fmt.Fprintf(&b, "| properties read as the plain one of the same name (%d) | %s |\n", len(v.same),
			code(v.same, v.prefix))
	}
	if len(v.renamed) > 0 {
		fmt.Fprintf(&b, "| properties read as a plain one of another name (%d) | %s |\n", len(v.renamed),
			pairs(v.renamed, v.prefix, ""))
	}
	if len(v.own) > 0 {
		byWhy := map[string][]string{}
		for _, name := range keys(v.own) {
			byWhy[v.own[name]] = append(byWhy[v.own[name]], name)
		}
		var whys []string
		for why := range byWhy {
			whys = append(whys, why)
		}
		slices.Sort(whys)
		var rows []string
		for _, why := range whys {
			rows = append(rows, code(byWhy[why], v.prefix)+": "+why)
		}
		fmt.Fprintf(&b, "| properties left as written (%d), so each keeps its warning unless ui reads it | %s |\n",
			len(v.own), strings.Join(rows, "; "))
	}
	if len(v.legacy) > 0 {
		var rows []string
		for _, name := range keys(v.legacy) {
			rows = append(rows, "`"+v.prefix+name+"` as `"+name+"`, "+v.legacy[name])
		}
		fmt.Fprintf(&b, "| Chrome's own properties read as the plain one (%d) | %s |\n", len(v.legacy),
			strings.Join(rows, "; "))
	}
	if len(v.words) > 0 {
		var rows []string
		for _, name := range slices.Sorted(maps.Keys(v.words)) {
			rows = append(rows, "`"+v.prefix+name+"`: "+pairs(v.words[name], "", ""))
		}
		fmt.Fprintf(&b, "| words read as the plain property's | %s; its other words keep their warning |\n",
			strings.Join(rows, "; "))
	}
	fmt.Fprintf(&b, "| any %sproperty | %s |\n", map[bool]string{true: "other "}[len(v.same) > 0],
		others[v.properties])
	if len(v.values) > 0 {
		fmt.Fprintf(&b, "| values read as the plain one | %s, and the properties above in `transition` |\n",
			pairs(v.values, v.prefix, ""))
	}
	if v.otherValues == dropped {
		b.WriteString("| a value ui refuses that holds one of its words or functions | its declaration goes without a " +
			"word, as Chrome drops it |\n")
	} else {
		b.WriteString("| any other value | keeps its warning, the declaration dropped |\n")
	}
	if len(v.elements) > 0 {
		fmt.Fprintf(&b, "| pseudo-elements read as the plain one | %s |\n", pairs(v.elements, "::"+v.prefix, "::"))
	}
	if v.otherElements == dropped {
		b.WriteString("| any pseudo-element or pseudo-class | its whole rule goes without a word, as Chrome drops it |\n")
	} else {
		fmt.Fprintf(&b, "| any other pseudo-element | %s |\n", others[v.otherElements])
		fmt.Fprintf(&b, "| pseudo-classes read as the plain one | %s |\n", pairs(v.classes, ":"+v.prefix, ":"))
		fmt.Fprintf(&b, "| pseudo-classes nothing in ui is ever in | %s: the selector alone goes, without a word |\n",
			code(v.never, ":"+v.prefix))
		fmt.Fprintf(&b, "| pseudo-classes Chrome reads and ui can't | %s: the rule keeps its warning |\n",
			code(v.ownClasses, ":"+v.prefix))
		b.WriteString("| any other pseudo-class | its whole rule goes without a word, as Chrome drops it |\n")
	}
	if len(v.atRules) > 0 {
		fmt.Fprintf(&b, "| at-rules read as the plain one | %s |\n",
			pairs(v.atRules, "@"+v.prefix, "@"))
	}
	fmt.Fprintf(&b, "| any %sat-rule | %s |\n", map[bool]string{true: "other "}[len(v.atRules) > 0],
		others[v.otherAtRules])
	if len(v.features) > 0 {
		fmt.Fprintf(&b, "| media features read as the plain one, a bare number as dppx | %s |\n",
			pairs(v.features, v.prefix, ""))
		fmt.Fprintf(&b, "| media features Chrome answers and ui can't | %s: the query is false, with a warning |\n",
			code(v.ownFeatures, v.prefix))
	}
	fmt.Fprintf(&b, "| any %smedia feature | its query is false without a word, as in Chrome |\n",
		map[bool]string{true: "other "}[len(v.features) > 0])
	return b.String()
}

// TestTheGuideHasEveryVendorsTable keeps the guide's vendor sections the
// tables of vendors.go; EMBER_VENDOR_GUIDE=1 prints them to paste there.
func TestTheGuideHasEveryVendorsTable(t *testing.T) {
	guide, err := os.ReadFile("../../docs/05-ui/03-css.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range vendors {
		section := vendorGuide(v)
		if os.Getenv("EMBER_VENDOR_GUIDE") != "" {
			fmt.Fprintln(os.Stdout, section)
		}
		if !strings.Contains(string(guide), section) {
			t.Errorf("docs/05-ui/03-css.md doesn't hold the table of %s as vendors.go has it:\n%s", v.prefix, section)
		}
	}
}
