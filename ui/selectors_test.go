package ui_test

import (
	"testing"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/world"
)

func shown(t *testing.T, d *ui.Document) *world.World {
	t.Helper()
	win := platform.NewWindow(platform.Size(400, 300))
	win.Add(d)
	w := ember.New().AddEntities(win)
	w.RunFrames(1)
	return w
}

func padded(d *ui.Document, n *ui.Node) bool {
	v, _ := d.ComputedValue(n, "padding-left")
	return v == "7px"
}

func TestFocusWithinFollowsTheFocusUpItsAncestors(t *testing.T) {
	d := ui.NewDocument(ui.CSS("form:focus-within, div:focus-within { padding-left: 7px }"))
	field, other := ui.Input(), ui.Button(ui.Text("other"))
	box := ui.Div(field)
	form := ui.Form(box)
	d.Body().Append(form, ui.Div(other))
	w := shown(t, d)
	if padded(d, form) {
		t.Fatal("no focus: the form is in :focus-within")
	}
	d.Focus(field)
	w.RunFrames(2)
	if !padded(d, form) || !padded(d, box) {
		t.Error("a field focused: its form and its box are not in :focus-within")
	}
	d.Focus(other)
	w.RunFrames(2)
	if padded(d, form) || padded(d, box) || !padded(d, other.Parent()) {
		t.Error("the focus moved: :focus-within did not move with it")
	}
}

func TestFocusWithinReachesThroughAnIframe(t *testing.T) {
	child := ui.NewDocument(ui.CSS("body:focus-within { padding-left: 7px }"))
	field := ui.Input()
	child.Body().Append(field)
	d := ui.NewDocument(ui.CSS("section:focus-within { padding-left: 7px }"))
	section := ui.Section(ui.Iframe(child))
	d.Body().Append(section, ui.Input())
	w := shown(t, d)
	child.Focus(field)
	w.RunFrames(3)
	if !padded(d, section) || !padded(child, child.Body()) {
		t.Error("a field focused in an iframe: the elements round the iframe are not in :focus-within")
	}
}

func TestTheTargetIsTheElementTheDocumentSays(t *testing.T) {
	d := ui.NewDocument(ui.CSS(":target { padding-left: 7px }"))
	a, b := ui.H2(ui.ID("a"), ui.Text("a")), ui.H2(ui.ID("b"), ui.Text("b"))
	d.Body().Append(a, b)
	w := shown(t, d)
	if d.Target() != nil || padded(d, a) {
		t.Fatal("a Document has a target before one is set")
	}
	d.SetTarget(a)
	w.RunFrames(1)
	if d.Target() != a || !padded(d, a) || padded(d, b) {
		t.Error("the target set does not match :target")
	}
	d.SetTarget(b)
	w.RunFrames(1)
	if padded(d, a) || !padded(d, b) {
		t.Error("the target moved: :target did not move with it")
	}
	d.SetTarget(nil)
	w.RunFrames(1)
	if d.Target() != nil || padded(d, b) {
		t.Error("the target cleared: something still matches :target")
	}
}

func TestTheDocumentsLanguageIsWhatLangFallsBackTo(t *testing.T) {
	d := ui.NewDocument(ui.Language("fr-BE"), ui.CSS(":lang(fr) { padding-left: 7px }"))
	p, en := ui.P(ui.Text("p")), ui.P(ui.Attr("lang", "en"), ui.Text("en"))
	d.Body().Append(p, en)
	w := shown(t, d)
	if d.Language() != "fr-BE" || !padded(d, p) || padded(d, en) {
		t.Error("the Document's language is not the language of an element no lang attribute names one for")
	}
	d.SetLanguage("de")
	w.RunFrames(1)
	if padded(d, p) {
		t.Error("the Document's language changed: :lang() still reads the old one")
	}
	d.HTML().SetAttr("lang", "fr")
	w.RunFrames(1)
	if !padded(d, p) {
		t.Error("the root's lang attribute does not win over the Document's language")
	}
}
