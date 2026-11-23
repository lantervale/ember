//go:build chrome

package css_test

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/css"
	"github.com/lantervale/ember/ui/internal/chrome"
)

// TestFormPseudoClassesMatchWhatChromeMatches asks Chrome which elements
// of a form the pseudo-classes of fields' values and states match, then
// again once a script has changed the values as the Document's program
// does, and compares ui's answers both times.
func TestFormPseudoClassesMatchWhatChromeMatches(t *testing.T) {
	d := ui.NewDocument()
	radio := func(id, name string, args ...ui.Arg) *ui.Node {
		return ui.Input(append([]ui.Arg{ui.ID(id), ui.Type("radio"), ui.Attr("name", name)}, args...)...)
	}
	req := ui.Attr("required", "")
	d.Body().Append(
		ui.Form(ui.ID("f"),
			ui.Input(ui.ID("name"), req, ui.Placeholder("Name")), ui.Span(ui.ID("tip"), ui.Text("!")),
			ui.Input(ui.ID("mail"), ui.Type("email"), ui.Placeholder("")),
			ui.Input(ui.ID("mails"), ui.Type("email"), ui.Attr("multiple", ""), ui.Value("a@b.c, d@e")),
			ui.Input(ui.ID("site"), ui.Type("url"), ui.Value("https://example.com/x")),
			ui.Input(ui.ID("code"), ui.Attr("pattern", "[A-Z]{3}"), ui.Value("ABC")),
			ui.Input(ui.ID("age"), ui.Type("number"), ui.Attr("min", "0"), ui.Attr("max", "120"), ui.Placeholder("Age")),
			ui.Input(ui.ID("qty"), ui.Type("number"), ui.Attr("step", "5"), ui.Value("10")),
			ui.Input(ui.ID("free"), ui.Type("number")),
			ui.Input(ui.ID("tiny"), ui.Type("number"), ui.Attr("min", "1"), ui.Value("0.5"), ui.Attr("step", "any")),
			ui.Input(ui.ID("vol"), ui.Type("range")),
			ui.Input(ui.ID("agree"), ui.Type("checkbox"), req),
			ui.Input(ui.ID("news"), ui.Type("checkbox"), ui.Checked(true)),
			ui.Input(ui.ID("mixed"), ui.Type("CHECKBOX")),
			radio("a1", "size", req), radio("a2", "size"),
			radio("b1", "shape", ui.Checked(true)), radio("b2", "shape"),
			ui.Input(ui.ID("lone"), ui.Type("radio")),
			ui.Select(ui.ID("pick"), req,
				ui.Option(ui.ID("p0"), ui.Value(""), ui.Selected(), ui.Text("Choose")), ui.Option(ui.ID("p1"), ui.Text("One"))),
			ui.Select(ui.ID("plain"),
				ui.Option(ui.ID("q0"), ui.Selected(), ui.Text("a")), ui.Option(ui.ID("q1"), ui.Disabled(), ui.Text("b")),
				ui.Option(ui.ID("q2"), ui.Text("c"))),
			ui.Textarea(ui.ID("bio"), req, ui.Placeholder("About you")),
			ui.Input(ui.ID("ro"), req, ui.Attr("readonly", "")),
			ui.Input(ui.ID("off"), req, ui.Disabled()),
			ui.Fieldset(ui.ID("fs"), ui.Disabled(),
				ui.Legend(ui.ID("lg"), ui.Input(ui.ID("inlegend"), req)),
				ui.Input(ui.ID("infs"), req), ui.Button(ui.ID("fsbtn"), ui.Text("x"))),
			ui.Fieldset(ui.ID("fs2"), ui.Input(ui.ID("inner"), req)),
			ui.Button(ui.ID("plainbtn"), ui.Type("button"), ui.Text("b")),
			ui.Button(ui.ID("go"), ui.Text("Go")),
			ui.Button(ui.ID("also"), ui.Type("SUBMIT"), ui.Text("Also")),
			ui.Input(ui.ID("reset"), ui.Type("reset")),
			ui.Input(ui.ID("hide"), ui.Type("hidden"), req),
		),
		ui.Form(ui.ID("f2"), radio("c1", "size", ui.Checked(true)), radio("c2", "shape"),
			ui.Input(ui.ID("c4"), ui.Type("radio"), ui.Attr("name", "size"), ui.Attr("form", "f"))),
		radio("c3", "size", ui.Attr("form", "f2")), radio("c5", "size", ui.Checked(true)),
		ui.Button(ui.ID("out"), ui.Text("Out")),
		ui.Progress(ui.ID("prog")),
		ui.Progress(ui.ID("done"), ui.Value("1")),
		ui.Div(ui.ID("rtl"), ui.Attr("dir", "RTL"), ui.Span(ui.ID("inrtl"), ui.Text("x")),
			ui.Div(ui.ID("he"), ui.Attr("dir", "auto"), ui.Span(ui.Text("123 ")), ui.Text("שלום")),
			ui.Div(ui.ID("digits"), ui.Attr("dir", "auto"), ui.Text("123")),
			ui.Div(ui.ID("skip"), ui.Attr("dir", "auto"), ui.Bdi(ui.Text("abc")), ui.Text("שלום")),
			ui.Bdi(ui.ID("bdi"), ui.Text("abc")),
			ui.Div(ui.ID("bogus"), ui.Attr("dir", "up"))),
		ui.Input(ui.ID("rtlin"), ui.Attr("dir", "auto"), ui.Value("abc")),
		ui.Div(ui.ID("al"), ui.Attr("align", "CENTER"), ui.Attr("title", "Hi")),
		ui.A(ui.ID("rel"), ui.Attr("rel", "NoFollow"), ui.Text("a")),
	)
	sels := []string{
		":valid", ":invalid", ":user-valid", ":user-invalid", ":required", ":optional", ":in-range", ":out-of-range",
		":placeholder-shown", ":default", ":indeterminate", ":checked", ":disabled", ":enabled", ":dir(rtl)",
		":dir(ltr)", ":autofill", ":-webkit-autofill", "form:invalid", "fieldset:valid", ":has(> :invalid)",
		":invalid + span", "input:not(:placeholder-shown)", "option:checked", "option:default", ":default:checked",
		"[type=checkbox]", "[type=Email]", "[align=center]", "[title=hi]", "[rel=nofollow]", "[dir=rtl]",
		"[type=submit]",
	}
	list, _ := json.Marshal(sels)
	raw, err := chrome.Eval(d, geom.V2(800, 1200), `
const sels = `+string(list)+`;
const ask = () => sels.map(s => [...document.querySelectorAll(s)].map(e => e.id).filter(id => id));
const before = ask();
const $ = id => document.getElementById(id);
$("name").value = "Ada";
$("mail").value = "ada@example";
$("mails").value = "a@b.c, x";
$("site").value = "notaurl";
$("code").value = "abc";
$("age").value = "130";
$("qty").value = "12";
$("free").value = "7";
$("tiny").value = "2";
$("agree").checked = true;
$("mixed").indeterminate = true;
$("a2").checked = true;
$("c3").checked = true;
$("pick").selectedIndex = 1;
$("plain").selectedIndex = 2;
$("bio").value = "x";
$("inner").value = "y";
$("rtlin").value = "שלום";
return [before, ask()];`)
	if err != nil {
		t.Fatal(err)
	}
	var chromes [2][][]string
	if err = json.Unmarshal(raw, &chromes); err != nil {
		t.Fatalf("%v in %s", err, raw)
	}
	compare := func(phase string, want [][]string) {
		t.Helper()
		d.Layout(geom.V2(800, 1200))
		for i, src := range sels {
			parsed, unread := css.ParseSelectors(src)
			if unread != nil {
				t.Errorf("%s: %v", src, unread)
				continue
			}
			var got []string
			d.HTML().Walk(func(n *ui.Node) bool {
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
	compare("before", chromes[0])
	for id, v := range map[string]string{
		"name": "Ada", "mail": "ada@example", "mails": "a@b.c, x", "site": "notaurl", "code": "abc", "age": "130",
		"qty": "12", "free": "7", "tiny": "2", "bio": "x", "inner": "y", "rtlin": "שלום",
	} {
		d.SetValue(d.ByID(id), v)
	}
	d.SetChecked(d.ByID("agree"), true)
	d.SetIndeterminate(d.ByID("mixed"), true)
	d.SetChecked(d.ByID("a2"), true)
	d.SetChecked(d.ByID("c3"), true)
	d.SetSelected(d.ByID("pick"), 1)
	d.SetSelected(d.ByID("plain"), 2)
	compare("after", chromes[1])
}
