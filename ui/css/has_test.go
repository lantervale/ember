package css //nolint:testpackage // reads computed styles off dom with the package's test helpers

import (
	"testing"

	"github.com/lantervale/ember/ui/internal/dom"
)

func TestHasMatchesByWhatIsBelowOrAfter(t *testing.T) {
	img := el(dom.Img, "img")
	deep := el(dom.Span, "deep", "x")
	card := with(el(dom.Div, "card", "card"), img, with(el(dom.P, "p"), deep))
	next := el(dom.P, "next")
	last := el(dom.Div, "last", "tail")
	plain := el(dom.Div, "plain")
	page(card, next, plain, last)

	cases := []struct {
		src  string
		e    *dom.Node
		want bool
	}{
		{".card:has(img)", card, true},
		{"div:has(img)", plain, false},
		{":has(> img)", card, true},
		{":has(> span)", card, false},
		{":has(span)", card, true},
		{":has(p > .x)", card, true},
		{":has(> p > .x)", card, true},
		{":has(> .x)", card, false},
		{":has(img + p)", card, true},
		{":has(img ~ span)", card, false},
		{":has(+ p)", card, true},
		{":has(+ div)", card, false},
		{":has(~ .tail)", card, true},
		{":has(~ div)", next, true},
		{":has(+ p .x)", card, false},
		{":has(~ div.tail)", last, false},
		{":has(.card)", card, false},
		{":has(blink, img)", card, true},
		{":has(blink)", card, false},
		{":has(:is(body img))", card, true},
		{":has(img):has(.x)", card, true},
		{":not(:has(img))", plain, true},
		{":is(:has(> img), .none)", card, true},
		{"body:has(.x) > div", plain, true},
		{":has(+ p) ~ div", plain, true},
		{":has(.card img)", card.Parent(), true},
		{":has(.card img)", card, false},
	}
	for _, c := range cases {
		if got := mustSelector(t, c.src).Matches(c.e); got != c.want {
			t.Errorf("%q on #%s: got %v", c.src, c.e.ID(), got)
		}
	}
}

func TestHasRefusesWhatChromeRefuses(t *testing.T) {
	for _, src := range []string{":has()", ":has(:has(img))", ":has(::before)", ":has(> )", ":has(img,)", ":has(> > img)",
		":not(:has(:not(:has(p))))", "::before:has(img)", ":lang()", ":lang(en, fr)", `:lang("en")`, ":lang(12)",
		`[x=y s]`} {
		if _, err := ParseSelectors(src); err == nil {
			t.Errorf("%q: no error", src)
		}
	}
	for _, src := range []string{":has(:is(:has(img)))", ":has(blink)", ":lang(\\*-CH)", ":has(:not(img))"} {
		if _, err := ParseSelectors(src); err != nil {
			t.Errorf("%q: %v", src, err)
		}
	}
}

func TestTheNewPseudoClassesWeighAsChromeWeighsThem(t *testing.T) {
	for _, c := range []struct {
		src  string
		want Specificity
	}{
		{":has(img)", Specificity{0, 0, 1}},
		{"div:has(> #a, .b)", Specificity{1, 0, 1}},
		{":has(+ p .x)", Specificity{0, 1, 1}},
		{":has(:where(#a))", Specificity{0, 0, 0}},
		{"p:empty", Specificity{0, 1, 1}},
		{":target", Specificity{0, 1, 0}},
		{":focus-within", Specificity{0, 1, 0}},
		{":lang(en)", Specificity{0, 1, 0}},
		{"input:read-only", Specificity{0, 1, 1}},
		{":read-write", Specificity{0, 1, 0}},
	} {
		if got := mustSelector(t, c.src).Specificity; got != c.want {
			t.Errorf("%q: got %v, want %v", c.src, got, c.want)
		}
	}
}

func TestEmptyIsAnElementWithNothingButEmptyText(t *testing.T) {
	bare, blank, spaced, full := el(dom.Div, "bare"), el(dom.Div, "blank"), el(dom.Div, "spaced"), el(dom.Div, "full")
	blank.Append(dom.NewText(""))
	spaced.Append(dom.NewText(" "))
	full.Append(el(dom.Span, ""))
	page(bare, blank, spaced, full)
	s := mustSelector(t, ":empty")
	for n, want := range map[*dom.Node]bool{bare: true, blank: true, spaced: false, full: false} {
		if got := s.Matches(n); got != want {
			t.Errorf("#%s: :empty %v", n.ID(), got)
		}
	}
	full.Remove(full.Children()[0])
	spaced.Children()[0].SetText("")
	blank.Children()[0].SetText("x")
	bare.SetText("")
	for n, want := range map[*dom.Node]bool{bare: true, blank: false, spaced: true, full: true} {
		if got := s.Matches(n); got != want {
			t.Errorf("changed, #%s: :empty %v", n.ID(), got)
		}
	}
}

func TestLangFollowsTheAttributeUpToTheDocumentsLanguage(t *testing.T) {
	inner := el(dom.Span, "inner")
	us := with(el(dom.Div, "us"), inner)
	us.SetAttr("lang", "en-US")
	blank := el(dom.Div, "blank")
	blank.SetAttr("lang", "")
	ch := el(dom.Div, "ch")
	ch.SetAttr("lang", "de-Latn-CH")
	odd := el(dom.Div, "odd")
	odd.SetAttr("lang", "en_US")
	star := el(dom.Div, "star")
	star.SetAttr("lang", "*")
	none := el(dom.Div, "none")
	root := page(us, blank, ch, odd, star, none)

	cases := []struct {
		src  string
		e    *dom.Node
		want bool
	}{
		{":lang(en)", inner, true},
		{":lang(EN-us)", inner, true},
		{":lang(en-US)", us, true},
		{":lang(e)", us, false},
		{":lang(en-)", us, false},
		{":lang(de)", ch, true},
		{":lang(de-CH)", ch, false},
		{":lang(\\*-CH)", ch, false},
		{":lang(\\*)", star, false},
		{":lang(en)", odd, false},
		{":lang(en_US)", odd, false},
		{":lang(fr)", none, false},
		{":lang(fr)", blank, false},
	}
	for _, c := range cases {
		if got := mustSelector(t, c.src).Matches(c.e); got != c.want {
			t.Errorf("%q on #%s: got %v", c.src, c.e.ID(), got)
		}
	}
	dom.SetLanguage(root, "fr-BE")
	if !mustSelector(t, ":lang(fr)").Matches(none) || mustSelector(t, ":lang(fr)").Matches(blank) ||
		mustSelector(t, ":lang(fr)").Matches(inner) {
		t.Error("the Document's language is the language of what no lang attribute names one for")
	}
}

func TestReadWriteIsWhatTheUserCanEdit(t *testing.T) {
	input := func(id, kind string, attrs ...string) *dom.Node {
		n := el(dom.Input, id)
		if kind != "" {
			n.SetAttr("type", kind)
		}
		for _, a := range attrs {
			n.SetAttr(a, "")
		}
		return n
	}
	text, number, date, odd := input("text", ""), input("number", "number"), input("date", "date"), input("odd", "nope")
	upper := input("upper", "EMAIL")
	ro, off := input("ro", "", "readonly"), input("off", "", "disabled")
	check, rng, color, hidden := input("check", "checkbox"), input("range", "range"), input("color", "color"),
		input("hidden", "hidden")
	area, areaOff := el(dom.Textarea, "area"), el(dom.Textarea, "areaOff")
	areaOff.SetState(dom.Disabled, true)
	editIn, plainIn, stillIn := el(dom.Span, "editIn"), el(dom.Span, "plainIn"), el(dom.Span, "stillIn")
	edit, plain := with(el(dom.Div, "edit"), editIn), with(el(dom.Div, "plain"), plainIn)
	edit.SetAttr("contenteditable", "")
	plain.SetAttr("contenteditable", "plaintext-only")
	still := with(el(dom.Div, "still"), stillIn)
	still.SetAttr("contenteditable", "false")
	bad := with(el(dom.Div, "bad"), el(dom.Span, "badIn"))
	bad.SetAttr("contenteditable", "maybe")
	roInEdit := input("roInEdit", "", "readonly")
	edit.Append(roInEdit, with(el(dom.Div, "nested"), still))
	sel, div := el(dom.Select, "sel"), el(dom.Div, "div")
	page(text, number, date, odd, upper, ro, off, check, rng, color, hidden, area, areaOff, edit, plain, bad, sel, div)

	rw := map[string]bool{"text": true, "number": true, "date": true, "odd": true, "upper": true, "area": true,
		"edit": true, "editIn": true, "plain": true, "plainIn": true, "nested": true}
	readWrite, readOnly := mustSelector(t, ":read-write"), mustSelector(t, ":read-only")
	text.Parent().Parent().Walk(func(n *dom.Node) bool {
		if n.IsText() || n.ID() == "" {
			return true
		}
		if got := readWrite.Matches(n); got != rw[n.ID()] {
			t.Errorf("#%s: :read-write %v", n.ID(), got)
		}
		if got := readOnly.Matches(n); got == rw[n.ID()] {
			t.Errorf("#%s: :read-only %v", n.ID(), got)
		}
		return true
	})
}

func TestTargetAndFocusWithinAreStates(t *testing.T) {
	n := el(dom.Div, "x")
	page(n)
	n.SetState(dom.Target|dom.FocusWithin, true)
	if !mustSelector(t, "div:target:focus-within").Matches(n) {
		t.Error("a node in Target and FocusWithin matches :target and :focus-within")
	}
}

// What :has() reads below or after an element restyles that element, and
// what it reads beside or below it, as the selector holding it asks.
func TestAChangeHasReadsRestylesItsAnchor(t *testing.T) {
	red := hex("#f00")
	type tree struct{ root, card, img, p, deep, next, after, inner *dom.Node }
	build := func() tree {
		var x tree
		x.img, x.deep = el(dom.Img, "img"), el(dom.Span, "deep")
		x.p = with(el(dom.P, "p"), x.deep)
		x.card = with(el(dom.Div, "card", "card"), x.img, x.p)
		x.next = el(dom.P, "next")
		x.inner = el(dom.Span, "inner")
		x.after = with(el(dom.Div, "after"), x.inner)
		x.root = page(x.card, x.next, x.after)
		return x
	}
	steps := []struct {
		name, src string
		do        func(x tree)
		red       func(x tree) *dom.Node
	}{
		{"a class below", ".card:has(.on) { color: #f00 }", func(x tree) { x.deep.AddClass("on") },
			func(x tree) *dom.Node { return x.card }},
		{"an attribute below", ".card:has([data-x]) { color: #f00 }", func(x tree) { x.deep.SetAttr("data-x", "1") },
			func(x tree) *dom.Node { return x.card }},
		{"a state below", ".card:has(:hover) { color: #f00 }", func(x tree) { x.deep.SetState(dom.Hover, true) },
			func(x tree) *dom.Node { return x.card }},
		{"a child added", ".card:has(> b) { color: #f00 }", func(x tree) { x.card.Append(el(dom.B, "")) },
			func(x tree) *dom.Node { return x.card }},
		{"a deep child added", ".card:has(b) { color: #f00 }", func(x tree) { x.deep.Append(el(dom.B, "")) },
			func(x tree) *dom.Node { return x.card }},
		{"a child gone", ".card:not(:has(img)) { color: #f00 }", func(x tree) { x.card.Remove(x.img) },
			func(x tree) *dom.Node { return x.card }},
		{"emptied", ".card:has(p:empty) { color: #f00 }", func(x tree) { x.p.Remove(x.deep) },
			func(x tree) *dom.Node { return x.card }},
		{"a sibling after", ":has(+ .on) { color: #f00 }", func(x tree) { x.next.AddClass("on") },
			func(x tree) *dom.Node { return x.card }},
		{"below a sibling after", "div:has(~ div .on) { color: #f00 }", func(x tree) { x.inner.AddClass("on") },
			func(x tree) *dom.Node { return x.card }},
		{"a sibling's child gone", "p:has(+ div > span) { color: #f00 }", func(x tree) { x.after.Remove(x.inner) },
			func(tree) *dom.Node { return nil }},
		{"under the anchor", ".card:has(.on) span { color: #f00 }", func(x tree) { x.img.AddClass("on") },
			func(x tree) *dom.Node { return x.deep }},
		{"after the anchor", ".card:has(.on) ~ div span { color: #f00 }", func(x tree) { x.deep.AddClass("on") },
			func(x tree) *dom.Node { return x.inner }},
		{"inside :is()", ":is(.card:has(:checked), .none) p { color: #f00 }",
			func(x tree) { x.img.SetState(dom.Checked, true) }, func(x tree) *dom.Node { return x.p }},
	}
	for _, step := range steps {
		x := build()
		sheets := []*Sheet{sheet(t, "div, p, span, img, b { color: #000 }\n"+step.src)}
		tr := NewTransitions()
		tr.Compute(x.root, sheets)
		want := step.red(x)
		if want != nil && styleOf(want).Color == red {
			t.Fatalf("%s: red before the change", step.name)
		}
		step.do(x)
		tr.Compute(x.root, sheets)
		x.root.Walk(func(n *dom.Node) bool {
			if n.IsText() || n.ID() == "" {
				return true
			}
			if got := styleOf(n).Color == red; got != (n == want) {
				t.Errorf("%s: #%s red %v", step.name, n.ID(), got)
			}
			return true
		})
	}
}

// A change :has() reads restyles only the anchors whose other parts it
// can match, and nothing when no :has() reads what changed.
func TestAChangeHasReadsRestylesOnlyItsAnchors(t *testing.T) {
	const rows = 300
	for _, c := range []struct {
		name, src string
		want      int
	}{
		{"no :has()", ".row .item { background-color: #f00 }", 1},
		{"a :has() reading another state", ".row:has(:focus) { background-color: #f00 }", 1},
		{"its row", ".row:has(.item:hover) { background-color: #f00 }", 2},
		{"its row and the menu", ".row:has(:hover), .menu:has(:hover) { background-color: #f00 }", 3},
		{"every ancestor", ":has(:hover) { background-color: #f00 }", 5},
		{"the rows before it", ".row:has(~ .row :hover) { background-color: #f00 }", 1 + rows/2},
		{"under its row", ".row:has(:hover) .label { background-color: #f00 }", 3},
	} {
		root, menu := wide(rows)
		item := menu.Children()[rows/2].Children()[0]
		tr := NewTransitions()
		sheets := []*Sheet{sheet(t, c.src)}
		tr.Compute(root, sheets)
		for _, on := range []bool{true, false} {
			was := tr.computed
			item.SetState(dom.Hover, on)
			tr.Compute(root, sheets)
			if got := tr.computed - was; got != c.want {
				t.Errorf("%s, hover %v: %d elements restyled, want %d", c.name, on, got, c.want)
			}
		}
	}
}

func TestLangAndEditingReadAboveTheSubject(t *testing.T) {
	red := hex("#f00")
	span := el(dom.Span, "s")
	div := with(el(dom.Div, "d"), with(el(dom.P, "p"), span))
	root := page(div)
	sheets := []*Sheet{sheet(t, "span { color: #000 } span:lang(fr) { color: #f00 } "+
		"span:read-write { background-color: #f00 }")}
	tr := NewTransitions()
	tr.Compute(root, sheets)
	div.SetAttr("lang", "fr")
	div.SetAttr("contenteditable", "true")
	tr.Compute(root, sheets)
	if styleOf(span).Color != red || styleOf(span).Background != red {
		t.Error("a lang or a contenteditable set above restyles what reads it")
	}
	div.RemoveAttr("lang")
	div.RemoveAttr("contenteditable")
	dom.SetLanguage(root, "fr-CA")
	tr.Compute(root, sheets)
	if styleOf(span).Color != red || styleOf(span).Background == red {
		t.Error("the Document's language restyles what reads it")
	}
}

func TestHasInvalidationKeepsUpWithManyChanges(t *testing.T) {
	root, menu := wide(50)
	sheets := []*Sheet{sheet(t, ".row:has(.item.on) { color: #f00 } .menu:has(> .row:nth-child(3n).on) { color: #00f }")}
	tr := NewTransitions()
	tr.Compute(root, sheets)
	red, blue := hex("#f00"), hex("#00f")
	for step := range 40 {
		row := menu.Children()[(step*7)%len(menu.Children())]
		switch step % 4 {
		case 0:
			row.Children()[0].AddClass("on")
		case 1:
			row.AddClass("on")
		case 2:
			menu.Remove(row)
			menu.Append(row)
		case 3:
			row.Children()[0].RemoveClass("on")
		}
		tr.Compute(root, sheets)
		for i, r := range menu.Children() {
			if got, want := styleOf(r).Color == red, r.Children()[0].HasClass("on"); got != want {
				t.Fatalf("step %d, row %d: red %v, want %v", step, i, got, want)
			}
		}
		want := false
		for i, r := range menu.Children() {
			want = want || (i+1)%3 == 0 && r.HasClass("on")
		}
		if got := styleOf(menu).Color == blue; got != want {
			t.Fatalf("step %d: menu blue %v, want %v", step, got, want)
		}
	}
}

func BenchmarkRestyleHasBigPage(b *testing.B) {
	root, menu := wide(2000)
	s, warns := Parse("bench.css", ".row:has(.item:hover) { color: #f00 } .menu:has(> .row.on) { color: #00f } "+
		"li:has(> a) { padding: 2px } .side:has(+ .menu .label:hover) { color: #f00 } :has(> img) { display: block }")
	if len(warns) > 0 {
		b.Fatal(warnings(warns))
	}
	sheets := []*Sheet{s}
	item := menu.Children()[1000].Children()[0]
	tr := NewTransitions()
	tr.Compute(root, sheets)
	on := false
	b.ReportAllocs()
	for b.Loop() {
		on = !on
		item.SetState(dom.Hover, on)
		tr.Compute(root, sheets)
		laidOut(root)
	}
}

func BenchmarkComputeAllHas(b *testing.B) {
	root, _ := wide(2000)
	s, warns := Parse("bench.css", ".row:has(.item:hover) { color: #f00 } .menu:has(> .row.on) { color: #00f } "+
		"div:has(.nope) { padding: 2px } .side:has(~ .menu .label:hover) { color: #f00 } :has(> img) { display: block }")
	if len(warns) > 0 {
		b.Fatal(warnings(warns))
	}
	b.ReportAllocs()
	for b.Loop() {
		root.Walk(func(n *dom.Node) bool { dom.Data(n).Style = nil; return true })
		Compute(root, []*Sheet{s})
	}
}
