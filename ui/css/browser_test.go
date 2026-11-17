package css //nolint:testpackage // uses the package's unexported test helpers (el, page, asBrowser, styleOf)

import (
	"strings"
	"testing"

	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

func TestSupportsAppliesWhereUISupportsWhatItAsks(t *testing.T) {
	for cond, want := range map[string]bool{
		"(display: grid)":                                  true,
		"(display:flex)":                                   true,
		"((display: flex))":                                true,
		"(display: flex !important)":                       true,
		"(--x: y)":                                         true,
		"(position: sticky)":                               true,
		"(-webkit-appearance: none)":                       true,
		"(-moz-appearance: none)":                          false,
		"(display: -webkit-box)":                           false,
		"(backdrop-filter: blur(4px))":                     false,
		"(foo: bar)":                                       false,
		"not (display: grid)":                              false,
		"not (foo: bar)":                                   true,
		"(display: flex) and (foo: bar)":                   false,
		"(display: flex) or (foo: bar)":                    true,
		"(display: flex) and (color: red) or (foo: bar)":   false,
		"not ((foo: bar) or (display: flex))":              false,
		"(display: grid) and (not (display: inline-grid))": false,
		"selector(a > b)":                                  true,
		"selector(a + b)":                                  false,
		"selector(::-moz-x)":                               false,
		"selector(::-webkit-foo)":                          false,
		"font-tech(color-COLRv1)":                          false,
		"(display: flex) and (foo)":                        false,
		"":                                                 false,
	} {
		s, warns := asBrowser("@supports " + cond + " { p { color: #f00 } } a { color: #00f }")
		if warns != "" {
			t.Errorf("@supports %s: %s", cond, warns)
		}
		if got := len(s.Rules) == 2; got != want {
			t.Errorf("@supports %s: applies %v, want %v", cond, got, want)
		}
		if len(s.Rules) == 0 || s.Rules[len(s.Rules)-1].Selectors[0].String() != "a" {
			t.Errorf("@supports %s: the rule after it is lost", cond)
		}
	}
}

func TestWhatAnUnsupportedBlockHoldsSaysNothing(t *testing.T) {
	s, warns := asBrowser("@supports (foo: bar) { p { colour: red } @media (bogus) { a { -x: 1 } } } b { color: #f00 }")
	if warns != "" || plainRules(s) != "b { color: #f00; }\n" {
		t.Errorf("got %q %s", plainRules(s), warns)
	}
}

// coloured computes p under the browser sheets srcs and says which of
// the colours #100, #200… it got, as 1, 2…
func coloured(t *testing.T, srcs ...string) int {
	t.Helper()
	var sheets []*Sheet
	for _, src := range srcs {
		s, warns := asBrowser(src)
		if warns != "" {
			t.Fatal(warns)
		}
		sheets = append(sheets, s)
	}
	p := el(dom.P, "p", "c")
	Compute(page(p), sheets)
	for i := 1; i < 10; i++ {
		if styleOf(p).Color == hex("#"+string(rune('0'+i))+"00") {
			return i
		}
	}
	return 0
}

func TestLayersOrderTheCascade(t *testing.T) {
	cases := []struct {
		name string
		want int
		srcs []string
	}{
		{"a later layer wins whatever the selectors weigh", 2,
			[]string{"@layer a { #p { color: #100 } } @layer b { p { color: #200 } }"}},
		{"no layer wins over any", 1,
			[]string{"p { color: #100 } @layer a { #p.c { color: #200 } }"}},
		{"layers are ordered where they are first named", 1,
			[]string{"@layer b, a; @layer a { p { color: #100 } } @layer b { #p { color: #200 } }"}},
		{"a layer named again keeps its place", 2,
			[]string{"@layer a { p { color: #100 } } @layer b { p { color: #200 } } @layer a { #p { color: #300 } }"}},
		{"!important turns the layers round", 1,
			[]string{"@layer a { p { color: #100 !important } } @layer b { #p { color: #200 !important } }"}},
		{"!important in a layer wins over one in none", 1,
			[]string{"@layer a { p { color: #100 !important } } #p { color: #200 !important }"}},
		{"a layer's own rules win over the layers inside it", 1,
			[]string{"@layer a { p { color: #100 } @layer b { #p { color: #200 } } }"}},
		{"a.b names the layer b inside a", 3,
			[]string{"@layer a.b { #p { color: #100 } } @layer c { p { color: #200 } } @layer a { @layer d { p { color: " +
				"#400 } } } @layer c { p { color: #300 } }"}},
		{"a layer inside another stands with it", 2,
			[]string{"@layer a { @layer b { #p { color: #100 } } } @layer c { p { color: #200 } }"}},
		{"layers without a name are each their own", 2,
			[]string{"@layer { #p { color: #100 } } @layer { p { color: #200 } }"}},
		{"layers are the Document's, across its sheets", 1,
			[]string{"@layer b, a;", "@layer a { p { color: #100 } } @layer b { #p { color: #200 } }"}},
		{"a sheet's unnamed layer is not another sheet's", 2,
			[]string{"@layer { #p { color: #100 } }", "@layer { p { color: #200 } }"}},
		{"inside one layer, weight then order", 1,
			[]string{"@layer a { #p { color: #100 } p { color: #200 } }"}},
		{"a layer in an @media rule", 2,
			[]string{"@layer a { #p { color: #100 } } @media (max-width: 9px) { @layer b { p { color: #200 } } }"}},
	}
	for _, c := range cases {
		if got := coloured(t, c.srcs...); got != c.want {
			t.Errorf("%s: colour %d, want %d\n%s", c.name, got, c.want, strings.Join(c.srcs, "\n"))
		}
	}
}

func TestALayerWithNoNameOrTooManyIsAWarning(t *testing.T) {
	for src, want := range map[string]string{
		"@layer a, b { p { color: #f00 } } a { color: #f00 }": "t.css:1:1: @layer: a block takes one name",
		"@layer 1px { p { color: #f00 } } a { color: #f00 }":  `t.css:1:1: @layer: "1px" is no layer name`,
		"@layer a.; a { color: #f00 }":                        `t.css:1:1: @layer: "a." is no layer name`,
	} {
		s, warns := asBrowser(src)
		if warns != want || plainRules(s) != "a { color: #f00; }\n" {
			t.Errorf("%s\n got %q %s\nwant %s", src, plainRules(s), warns, want)
		}
	}
}

func TestABrowsersSheetsStartFromContentBoxWidths(t *testing.T) {
	kinds := map[string]*dom.Node{}
	var kids []*dom.Node
	add := func(name string, n *dom.Node) {
		kinds[name] = n
		kids = append(kids, n)
	}
	for _, tag := range []dom.Tag{dom.Div, dom.P, dom.Textarea, dom.Img, dom.Hr, dom.Input, dom.Button, dom.Select,
		dom.Table, dom.Progress, dom.Meter} {
		add(tag.String(), dom.NewElement(tag))
	}
	for _, typ := range []string{"text", "range", "checkbox", "radio", "button", "submit", "reset", "search"} {
		in := dom.NewElement(dom.Input)
		in.SetAttr("type", typ)
		add("input "+typ, in)
	}
	root := page(kids...)
	border := map[string]bool{"button": true, "select": true, "table": true, "progress": true, "meter": true,
		"input checkbox": true, "input radio": true, "input button": true, "input submit": true, "input reset": true,
		"input search": true}
	tr := NewTransitions()
	tr.BrowserSheets()
	tr.Compute(root, nil)
	for name, n := range kinds {
		if got := styleOf(n).BoxSizing == style.BorderBox; got != border[name] {
			t.Errorf("%s: border-box %v, want %v", name, got, border[name])
		}
	}
	root.Walk(func(n *dom.Node) bool { dom.Data(n).Style = nil; return true })
	NewTransitions().Compute(root, nil)
	for name, n := range kinds {
		if styleOf(n).BoxSizing != style.BorderBox {
			t.Errorf("%s: without the option every box is sized by its border", name)
		}
	}
}

func TestABrowsersSheetIsItsOwnTextForChrome(t *testing.T) {
	src := "@layer a { p { -webkit-transform: none; -moz-foo: 1 } } @supports (foo: bar) { a { color: red } }"
	s, _ := asBrowser(src)
	if !s.Browser() || s.String() != src {
		t.Errorf("got %q", s.String())
	}
	if plain, _ := Parse("t.css", "p { color: #f00 }"); plain.Browser() {
		t.Error("a sheet is no browser's unless asked")
	}
}

func TestASheetResolvesItsURLsFromItsAddress(t *testing.T) {
	shown := func(ref string, opts ...ParseOption) string {
		s, warns := Parse("style", "p { background-image: url("+ref+") }", opts...)
		if len(warns) > 0 {
			t.Fatal(warnings(warns))
		}
		p := el(dom.P, "p")
		Compute(page(p), []*Sheet{s})
		return styleOf(p).BackgroundImage.Source
	}
	from := From("https://lantervale.example/css/site/main.css")
	for ref, want := range map[string]string{
		"door.png":                      "https://lantervale.example/css/site/door.png",
		"../img/door.png":               "https://lantervale.example/css/img/door.png",
		"/img/door.png":                 "https://lantervale.example/img/door.png",
		"//cdn.example/door.png":        "https://cdn.example/door.png",
		"https://other.example/a.png":   "https://other.example/a.png",
		"data:image/png;base64,iVBORw0": "data:image/png;base64,iVBORw0",
		"ember:select-arrow":            "ember:select-arrow",
		"'quoted door.png'":             "https://lantervale.example/css/site/quoted%20door.png",
	} {
		if got := shown(ref, from); got != want {
			t.Errorf("url(%s) from the sheet's address: %s, want %s", ref, got, want)
		}
	}
	if got := shown("img/door.png"); got != "img/door.png" {
		t.Errorf("without an address: %s", got)
	}
	s, _ := Parse("style", "p { color: #f00 }", from)
	if s.From() != "https://lantervale.example/css/site/main.css" {
		t.Errorf("From: %q", s.From())
	}
	if _, warns := Parse("style", "p { color: #f00 }", From("https://bad host/")); len(warns) != 1 {
		t.Errorf("a bad address: %s", warnings(warns))
	}
	p := el(dom.P, "p")
	p.SetStyle("background-image: url(door.png)")
	Compute(page(p), []*Sheet{s})
	if got := styleOf(p).BackgroundImage.Source; got != "door.png" {
		t.Errorf("a style attribute's url is the page's to resolve: %s", got)
	}
}
