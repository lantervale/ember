//go:build chrome

package ui_test

import (
	"encoding/json"
	"fmt"
	"slices"
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/css"
	"github.com/lantervale/ember/ui/internal/chrome"
)

// chromeValid asks Chrome, for each element of d with an id starting
// with prefix, whether its value meets its constraints, in tree order.
func chromeValid(t *testing.T, d *ui.Document, prefix string) []bool {
	t.Helper()
	raw, err := chrome.Eval(d, geom.V2(800, 600), fmt.Sprintf(
		`return [...document.querySelectorAll("[id^=%s]")].map(e => e.validity.valid);`, prefix))
	if err != nil {
		t.Fatal(err)
	}
	var valid []bool
	if err = json.Unmarshal(raw, &valid); err != nil {
		t.Fatalf("%v in %s", err, raw)
	}
	return valid
}

// TestAPatternConstrainsAsInChrome has Chrome and ui test values against
// patterns of JavaScript's v flag: lookarounds, backreferences, named
// groups, modifiers, properties, class set operations and strings, and
// patterns Chrome refuses, which constrain nothing.
func TestAPatternConstrainsAsInChrome(t *testing.T) {
	pairs := [][2]string{
		{"(?=a)a.", "ab"}, {"(?=a)a.", "bb"}, {"(a)\\1", "aa"}, {"(a)\\1", "ab"}, {"\\d+", "12"}, {"\\p{L}+", "été"},
		{"[\\p{L}--[a-z]]", "É"}, {"[\\p{L}--[a-z]]", "e"}, {"(?<n>a)\\k<n>", "aa"}, {"(?<n>a)\\k<n>", "ab"},
		{"[\\q{ab|c}]", "ab"}, {"[\\q{ab|c}]", "c"}, {"[\\q{ab|c}]", "abc"}, {".", "\U0001F600"}, {"..", "\U0001F600"},
		{"\\w", "é"}, {"(?<=a)b", "b"}, {"a(?<=a)b", "ab"}, {"a(?<!a)b", "ab"}, {"a|", "x"}, {"[^]", "x"},
		{"\\u{61}", "a"}, {"\\x61", "a"}, {"[]", "x"}, {"\\/", "/"}, {"/", "/"}, {"(?i:A)", "a"}, {"(?i:A)", "b"},
		{"(?i:a)A", "aa"}, {"x*?y", "xxy"}, {"[a&&b]", "a"}, {"[\\d&&[0-4]]", "3"}, {"[\\d&&[0-4]]", "7"}, {"\\b", "x"},
		{"\\bx\\b", "x"}, {"^a$", "a"}, {"a{2,}", "aaa"}, {"a{2,}", "a"}, {"a{1,2}", "aaa"}, {"\\p{Script=Greek}", "α"},
		{"\\p{sc=Greek}", "a"}, {"\\P{Lu}", "a"}, {"[[a-c]--b]", "b"}, {"[[a-c]--b]", "c"}, {"(?:)", "9"},
		{"[\\w--\\d]", "5"}, {"[\\w--\\d]", "a"}, {"[^\\p{L}]", "a"}, {"\\s", "\u00a0"}, {"$", "9"}, {"[\\(]", "("},
		{"[a\\-z]", "-"}, {"[a\\-z]", "b"}, {"(?i:[^a])", "A"}, {"(?i:\\W)", "ſ"}, {"(?i:\\w)", "K"}, {"(a*)*b", "aab"},
		{"(a|ab)(c|bcd)(d*)", "abcd"}, {"(?!a)\\w", "b"}, {"(?!a)\\w", "a"}, {"(?<!a)b", "b"}, {"(.)(?<=\\1\\1)", "x"},
		{"\\u0041", "A"}, {"\\uD83D\\uDE00", "\U0001F600"}, {"(?i:[\\q{AB}])", "ab"}, {"[\\p{Lu}&&\\p{Script=Latin}]", "Ω"},
		{"[\\p{Lu}&&\\p{Script=Latin}]", "Q"}, {"[0-9]{3}-[0-9]{4}", "555-1234"}, {"[A-Za-z]{3}", "ab"},
		{"\\p{Lowercase}", "ª"}, {"\\p{Alpha}", "Ⅳ"}, {"\\p{ID_Start}", "℘"}, {"\\p{gc=Nd}", "٣"},
		{"(?<y>\\d{4})-\\k<y>", "2024-2024"}, {"(\\w+)\\s\\1", "hey hey"}, {"(?i:straße)", "STRASSE"},
		{"(?i:ǆ)", "ǅ"}, {"(?i:\\u{10400})", "\U00010428"},
		// Chrome refuses these: they constrain nothing.
		{"[a-z-]", "9"}, {"a{", "9"}, {"a)", "9"}, {"\\k<n>", "9"}, {"\\1", "9"}, {"(a)\\2", "9"}, {"\\p{Foo}", "9"},
		{"[(]", "9"}, {"[!!]", "9"}, {"[a-]", "9"}, {"x{2,1}", "9"}, {"\\M", "9"}, {"(?=a)*", "9"}, {"(?i)a", "9"},
		{"\\-", "9"}, {"[^\\q{ab}]", "9"}, {"(?<a>x)(?<a>y)", "9"}, {"[a&&b--c]", "9"},
	}
	d := ui.NewDocument()
	for i, p := range pairs {
		d.Body().Append(ui.Input(ui.ID(fmt.Sprint("p", i)), ui.Attr("pattern", p[0]), ui.Value(p[1])))
	}
	want := chromeValid(t, d, "p")
	d.Layout(geom.V2(800, 600))
	for i, p := range pairs {
		if got := d.Valid(d.ByID(fmt.Sprint("p", i))); got != want[i] {
			t.Errorf("pattern %q on %q: ui valid %v, Chrome %v", p[0], p[1], got, want[i])
		}
	}
}

// TestAnEmailsDomainIsTurnedToPunycodeAsInChrome has Chrome and ui test
// the addresses of value attributes whose domains are not in ASCII.
func TestAnEmailsDomainIsTurnedToPunycodeAsInChrome(t *testing.T) {
	addresses := []string{
		"x@bücher.de", "x@BÜCHER.de", "ü@a.de", "x@xn--bcher-kva.de", "x@ex\u00e4mple", "x@-ä.de", "x@\u0661\u0662.com",
		"x@a\u200db.c", "x@ＡＢ.com", "x@ä..de", "x@日本.jp", "x@ß.de", "x@\u05d0a.com", "x@ä_b.de", "x@ä.de, y@b.c",
	}
	d := ui.NewDocument()
	for i, a := range addresses {
		d.Body().Append(ui.Input(ui.ID(fmt.Sprint("m", i)), ui.Type("email"), ui.Value(a)))
	}
	d.Body().Append(ui.Input(ui.ID("many"), ui.Type("email"), ui.Attr("multiple", ""), ui.Value("x@bücher.de, y@é.fr")))
	want := chromeValid(t, d, "m")
	d.Layout(geom.V2(800, 600))
	for i, a := range addresses {
		if got := d.Valid(d.ByID(fmt.Sprint("m", i))); got != want[i] {
			t.Errorf("%q: ui valid %v, Chrome %v", a, got, want[i])
		}
	}
	if got := d.Valid(d.ByID("many")); got != want[len(want)-1] {
		t.Errorf("multiple: ui valid %v, Chrome %v", got, want[len(want)-1])
	}
}

// TestASubmittedFormsFieldsAreUserInvalidAsInChrome submits forms, by a
// script's requestSubmit in Chrome and RequestSubmit in ui, and compares
// what :user-invalid and :user-valid match, where the focus goes and
// which forms were submitted.
func TestASubmittedFormsFieldsAreUserInvalidAsInChrome(t *testing.T) {
	d := ui.NewDocument()
	req := ui.Attr("required", "")
	nodes := []*ui.Node{
		ui.Form(ui.ID("f"),
			ui.Input(ui.ID("name"), req), ui.Input(ui.ID("code"), ui.Attr("pattern", "[a-z]+"), ui.Value("9")),
			ui.Input(ui.ID("mail"), ui.Type("email"), ui.Value("x@bücher.de")), ui.Input(ui.ID("note"), ui.Value("ok")),
			ui.Input(ui.ID("agree"), ui.Type("checkbox"), req), ui.Textarea(ui.ID("bio"), req),
			ui.Select(ui.ID("pick"), req, ui.Option(ui.Value(""), ui.Text("Choose")), ui.Option(ui.Text("One"))),
			ui.Input(ui.ID("off"), req, ui.Disabled()), ui.Input(ui.ID("ro"), req, ui.Attr("readonly", "")),
			ui.Button(ui.ID("go"), ui.Text("Go")), ui.Button(ui.ID("plain"), ui.Type("button"), ui.Text("b"))),
		ui.Input(ui.ID("owned"), req, ui.Attr("form", "f")),
		ui.Input(ui.ID("outside"), req),
		ui.Form(ui.ID("g"), ui.Attr("novalidate", ""), ui.Input(ui.ID("loose"), req), ui.Input(ui.ID("fine"))),
		ui.Form(ui.ID("h"), ui.Input(ui.ID("good"), ui.Value("a"))),
	}
	d.Body().Append(nodes...)
	sels := []string{":user-invalid", ":user-valid", "form:user-invalid", "form:user-valid"}
	list, _ := json.Marshal(sels)
	raw, err := chrome.Eval(d, geom.V2(800, 600), `
const sels = `+string(list)+`;
const ask = () => sels.map(s => [...document.querySelectorAll(s)].map(e => e.id).filter(id => id));
const submitted = [];
for (const f of document.forms) f.addEventListener("submit", e => { e.preventDefault(); submitted.push(f.id); });
const before = ask();
for (const f of document.forms) f.requestSubmit();
return [before, ask(), document.activeElement.id, submitted];`)
	if err != nil {
		t.Fatal(err)
	}
	var want struct {
		before, after [][]string
		focus         string
		submitted     []string
	}
	var parts []json.RawMessage
	if err = json.Unmarshal(raw, &parts); err == nil {
		err = json.Unmarshal(parts[0], &want.before)
		err = jsonOr(err, parts[1], &want.after)
		err = jsonOr(err, parts[2], &want.focus)
		err = jsonOr(err, parts[3], &want.submitted)
	}
	if err != nil {
		t.Fatalf("%v in %s", err, raw)
	}
	l := liveDoc(t, nodes...)
	compare := func(phase string, want [][]string) {
		t.Helper()
		for i, src := range sels {
			parsed, unread := css.ParseSelectors(src)
			if unread != nil {
				t.Fatal(unread)
			}
			var got []string
			l.doc.HTML().Walk(func(n *ui.Node) bool {
				if !n.IsText() && n.ID() != "" && parsed[0].Matches(n) {
					got = append(got, n.ID())
				}
				return true
			})
			if !slices.Equal(got, want[i]) {
				t.Errorf("%s %s: ui matches %v, Chrome %v", phase, src, got, want[i])
			}
		}
	}
	compare("before", want.before)
	for _, id := range []string{"f", "g", "h"} {
		l.doc.RequestSubmit(l.doc.ByID(id))
	}
	l.frames(1)
	compare("after", want.after)
	if !l.doc.Focused(l.doc.ByID(want.focus)) {
		t.Errorf("the focus is not on #%s, as in Chrome", want.focus)
	}
	var got []string
	for _, s := range l.submitted() {
		got = append(got, s.ID())
	}
	if !slices.Equal(got, want.submitted) {
		t.Errorf("ui submitted %v, Chrome %v", got, want.submitted)
	}
}

func jsonOr(err error, raw json.RawMessage, v any) error {
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, v)
}
