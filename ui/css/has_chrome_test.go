//go:build chrome

package css_test

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/css"
	"github.com/lantervale/ember/ui/internal/chrome"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

// TestTheNewPseudoClassesMatchWhatChromeMatches asks Chrome which elements
// of a page :has(), :empty, :lang(), :read-only and :read-write match.
func TestTheNewPseudoClassesMatchWhatChromeMatches(t *testing.T) {
	d := ui.NewDocument(ui.Language("fr-BE"))
	d.Body().Append(
		ui.Div(ui.ID("card"), ui.Class("card"),
			ui.Img(ui.ID("img")),
			ui.P(ui.ID("p"), ui.Span(ui.ID("deep"), ui.Class("x"), ui.Text("deep"))),
			ui.P(ui.ID("blank"))),
		ui.P(ui.ID("next"), ui.Text(" ")),
		ui.Div(ui.ID("us"), ui.Attr("lang", "en-US"), ui.Span(ui.ID("inner"), ui.Text("in"))),
		ui.Div(ui.ID("ch"), ui.Attr("lang", "de-Latn-CH")),
		ui.Div(ui.ID("odd"), ui.Attr("lang", "en_US")),
		ui.Div(ui.ID("unset"), ui.Attr("lang", ""), ui.Text("x")),
		ui.Input(ui.ID("text")),
		ui.Input(ui.ID("ro"), ui.Attr("readonly", "")),
		ui.Input(ui.ID("off"), ui.Attr("disabled", "")),
		ui.Input(ui.ID("num"), ui.Type("number")),
		ui.Input(ui.ID("upper"), ui.Type("EMAIL")),
		ui.Input(ui.ID("check"), ui.Type("checkbox")),
		ui.Input(ui.ID("range"), ui.Type("range")),
		ui.Textarea(ui.ID("area")),
		ui.Div(ui.ID("edit"), ui.Attr("contenteditable", ""),
			ui.Span(ui.ID("editIn"), ui.Text("e")),
			ui.Div(ui.ID("still"), ui.Attr("contenteditable", "false"), ui.Span(ui.ID("stillIn"), ui.Text("s")))),
		ui.Div(ui.ID("plain"), ui.Attr("contenteditable", "plaintext-only"), ui.B(ui.ID("plainIn"), ui.Text("p"))),
		ui.Div(ui.ID("last"), ui.Class("tail")),
	)
	sels := []string{
		".card:has(img)", ":has(> img)", ":has(> span)", ":has(span)", ":has(p > .x)", ":has(> p > .x)", ":has(img + p)",
		":has(img ~ span)", ":has(+ p)", ":has(+ div)", ":has(~ .tail)", ":has(~ div)", ":has(+ p .x)", ":has(.card img)",
		":has(blink, img)", ":has(:is(body img))", ":has(img):has(.x)", "div:not(:has(*))", ":is(:has(> img), #text)",
		"body:has(.x) > div", ":has(+ p) ~ div", ":has(> :empty)", ":has(:not(span))", ":has(input:read-only)",
		":empty", "p:empty", "div:empty + *", ":not(:empty)",
		":lang(en)", ":lang(EN-us)", ":lang(e)", ":lang(de)", ":lang(de-CH)", `:lang(\*-CH)`, ":lang(fr)",
		":lang(fr-BE)", ":lang(en_US)", ":lang(en-)",
		":read-only", ":read-write", "input:read-write", "div:read-only", ":read-write > b",
	}
	list, _ := json.Marshal(sels)
	raw, err := chrome.Eval(d, geom.V2(400, 300),
		`return `+string(list)+`.map(s => [...document.querySelectorAll(s)].map(e => e.id).filter(id => id));`)
	if err != nil {
		t.Fatal(err)
	}
	var chromes [][]string
	if err = json.Unmarshal(raw, &chromes); err != nil {
		t.Fatalf("%v in %s", err, raw)
	}
	d.Layout(geom.V2(400, 300))
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
		if !slices.Equal(got, chromes[i]) {
			t.Errorf("%s: ui matches %v, Chrome %v", src, got, chromes[i])
		}
	}
}

// TestTheNewPseudoClassesWeighAsInChrome styles a page whose rules
// compete by the weight :has() and the others give.
func TestTheNewPseudoClassesWeighAsInChrome(t *testing.T) {
	d := ui.NewDocument(ui.CSS(`
div { color: #000; }
.card:has(#img) { color: #f00; }
.card.card { color: #0f0; }
.box:has(.a, #b) { background-color: #00f; }
.box.box.box { background-color: #ff0; }
p:where(:has(span)) { color: #0f0; }
p { color: #00f; }
span:lang(fr) { color: #f00; }
span.s { color: #0f0; }
input:read-write { background-color: #f00; }
input.i { background-color: #0f0; }
:empty { padding-left: 3px; }
i:empty { padding-left: 5px; }
`))
	d.Body().Append(
		ui.Div(ui.Class("card"), ui.Img(ui.ID("img"))),
		ui.Div(ui.Class("box"), ui.Span(ui.ID("b"), ui.Text("b"))),
		ui.P(ui.Span(ui.Class("s"), ui.Attr("lang", "fr"), ui.Text("s"))),
		ui.Input(ui.Class("i")),
		ui.I(),
	)
	props := map[string]func(*style.Computed) string{
		"color": compared["color"], "background-color": compared["background-color"],
		"padding-left": compared["padding-left"],
	}
	matchChromeProps(t, d, props, 0)
}

// TestHasFollowsTheTreeAsChrome changes a page under and beside the
// elements :has() reads from, in Chrome by a script and in ui by the same
// changes, and compares the styles each restyle leaves.
func TestHasFollowsTheTreeAsChrome(t *testing.T) {
	d := ui.NewDocument(ui.CSS(`
div, p, span, b, li { color: #000; }
.card:has(.on) { color: #f00; }
.card:has(> b) { background-color: #0f0; }
.card:not(:has(img)) span { color: #00f; }
li:has(+ .pick) { color: #f00; }
li:has(~ li .pick) { background-color: #ff0; }
ul:has(> li:nth-child(2n).pick) { color: #0f0; }
.card:has(p:empty) ~ ul li { padding-left: 2px; }
:has(> [data-x]) { padding-top: 3px; }
body:has(input:checked) b { color: #f0f; }
`))
	deep := ui.Span(ui.Class("deep"), ui.Text("deep"))
	img := ui.Img()
	p := ui.P(ui.Span(ui.Class("gone"), ui.Text("gone")))
	card := ui.Div(ui.Class("card"), img, ui.P(deep), p)
	items := []*ui.Node{ui.Li(ui.Text("one")), ui.Li(ui.Text("two")), ui.Li(ui.Span(ui.Text("three"))),
		ui.Li(ui.Text("four"))}
	list := ui.Ul(items[0], items[1], items[2], items[3])
	box := ui.Input(ui.Type("checkbox"))
	d.Body().Append(card, list, ui.Div(box, ui.B(ui.Text("b"))))
	d.Layout(geom.V2(800, 600))
	script := `
document.querySelector(".deep").classList.add("on");
document.querySelector(".card img").remove();
document.querySelector(".card").append(document.createElement("b"));
document.querySelector(".gone").remove();
const items = document.querySelectorAll("li");
items[3].classList.add("pick");
items[2].querySelector("span").classList.add("pick");
items[1].setAttribute("data-x", "");
document.querySelector("input").checked = true;
`
	props := []string{"color", "background-color", "padding-left", "padding-top"}
	names, _ := json.Marshal(props)
	raw, err := chrome.Eval(d, geom.V2(800, 600), script+`
return [document.documentElement, document.body, ...document.body.querySelectorAll("*")].map(e => {
	const s = getComputedStyle(e);
	return `+string(names)+`.map(p => s.getPropertyValue(p));
});`)
	if err != nil {
		t.Fatal(err)
	}
	var want [][]string
	if err = json.Unmarshal(raw, &want); err != nil {
		t.Fatalf("%v in %s", err, raw)
	}
	deep.AddClass("on")
	card.Remove(img)
	card.Append(ui.B())
	p.Remove(p.Children()[0])
	items[3].AddClass("pick")
	items[2].Children()[0].AddClass("pick")
	items[1].SetAttr("data-x", "")
	d.SetChecked(box, true)
	d.Layout(geom.V2(800, 600))
	var got []*dom.Node
	d.HTML().Walk(func(n *dom.Node) bool {
		if !n.IsText() {
			got = append(got, n)
		}
		return true
	})
	if len(got) != len(want) {
		t.Fatalf("ember has %d elements, Chrome %d", len(got), len(want))
	}
	for i, n := range got {
		for j, p := range props {
			if ours := compared[p](dom.Data(n).Style); !same(p, ours, want[i][j]) {
				t.Errorf("%s %s: ember %q, Chrome %q", where(n), p, ours, want[i][j])
			}
		}
	}
}

// TestTargetAndFocusWithinStyleAsInChrome goes to a fragment and focuses
// a field, in Chrome by a script and in ui through the Document.
func TestTargetAndFocusWithinStyleAsInChrome(t *testing.T) {
	d := ui.NewDocument(ui.CSS(`
div, p, span, form { color: #000; }
:target { color: #f00; }
:target > span { background-color: #0f0; }
form:focus-within { color: #00f; }
:focus-within > p { background-color: #ff0; }
section:has(:target) p { padding-left: 4px; }
`))
	field := ui.Input()
	target := ui.Div(ui.ID("t"), ui.Span(ui.Text("t")))
	d.Body().Append(
		ui.Section(target, ui.P(ui.Text("p"))),
		ui.Form(ui.Div(ui.P(ui.Text("q")), field), ui.P(ui.Text("r"))),
	)
	d.Layout(geom.V2(800, 600))
	then := func() {
		win := platform.NewWindow(platform.Size(800, 600))
		win.Add(d)
		w := ember.New().AddEntities(win)
		w.RunFrames(1)
		d.SetTarget(target)
		d.Focus(field)
		w.RunFrames(2)
		if d.Target() != target {
			t.Error("the Document's target is not the one set")
		}
	}
	props := map[string]func(*style.Computed) string{
		"color": compared["color"], "background-color": compared["background-color"],
		"padding-left": compared["padding-left"],
	}
	matchChromeShown(t, d, props, 2, `location.hash = "#t"; document.querySelector("input").focus();`, then,
		func(n *dom.Node) *style.Computed { return dom.Data(n).Style })
}
