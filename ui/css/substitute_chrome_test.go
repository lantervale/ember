//go:build chrome

package css_test

import (
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/style"
)

// TestSubstitutionFunctionsComputeAsInChrome compares what if(), typed
// attr(), custom functions, sibling-index() and sibling-count(),
// progress() and random() with a fixed key compute to with Chrome.
func TestSubstitutionFunctionsComputeAsInChrome(t *testing.T) {
	d := ui.NewDocument()
	d.AddSheet(`
@property --len { syntax: "<length>"; inherits: false; initial-value: 0px; }
@function --double(--x) { result: calc(var(--x) * 2); }
@function --pick(--a <length>, --b: 3px) returns <length> {
  --loc: calc(var(--a) + var(--b));
  result: var(--loc);
  @media (width > 1px) { result: calc(var(--loc) * 10); }
}
@function --outer(--v) { --k: 5; result: --inner(); }
@function --inner() { result: calc(var(--k) * var(--v)); }
@function --self() { result: --self(); }
.a { --n: 3; --s: foo; font-size: 10px; height: 4px;
  width: calc(10px * sibling-index());
  margin-right: calc(1px * sibling-count());
  z-index: calc(sibling-index() * 2);
  --w: if(style(--n: 3): yes; else: no);
  --w2: if(style(--n: 4): yes; else: no);
  --w3: if(media(width > 100px): wide; else: narrow);
  --w4: if(supports(display: grid): g; else: ng);
  --w5: if(style(--n: 4): a; style(--s: foo): b; else: c);
  --w6: if(not style(--n: 4): x);
  --w7: if(style(--n: 4): x);
  --w8: if(style(--n: 3) and media(width > 1px): both; else: no);
  --w9: if(style(--s): set; else: unset);
  --w10: if(style(--zz): set; else: unset);
  --w11: if(style(--len: 0px): zero; else: more);
  margin-left: attr(data-x px);
  margin-top: attr(data-l type(<length>));
  padding-left: attr(data-bad px, 7px);
  padding-right: attr(data-l type(<number>), 2px);
  --at: attr(data-x);
  --at2: attr(data-x type(<number>));
  --at3: attr(data-l type(<length>));
  --at4: attr(data-missing type(<length>), 1em);
  opacity: progress(5, 0, 10);
  flex-grow: progress(15, 0, 10);
  flex-shrink: progress(1em, 0px, 40px);
  padding-bottom: random(fixed 0.5, 0px, 100px);
  margin-bottom: random(fixed 0.25, 0px, 100px, 10px);
  --sib: sibling-index();
  --f1: --double(4);
  --f2: --pick(1em);
  --f3: --pick(1px, 2px);
  --f4: --double({1, 2});
  --f5: --outer(2);
  --f6: --self();
  padding-top: --pick(1px);
}
.a + .a { --n: 4; }
`)
	list := ui.Div()
	for range 3 {
		list.Append(ui.Div(ui.Class("a"), ui.Attr("data-x", "12"), ui.Attr("data-l", "3em")))
	}
	d.Body().Append(list)
	props := varsCompared([]string{"--w", "--w2", "--w3", "--w4", "--w5", "--w6", "--w7", "--w8", "--w9", "--w10",
		"--w11", "--at", "--at2", "--at3", "--at4", "--sib", "--f1", "--f2", "--f3", "--f4", "--f5", "--f6"})
	for _, p := range []string{"width", "margin-right", "z-index", "margin-left", "margin-top", "padding-left",
		"padding-right", "opacity", "flex-grow", "flex-shrink", "padding-bottom", "margin-bottom", "padding-top"} {
		props[p] = everything[p]
	}
	matchChromeProps(t, d, props, 3)
}

// TestSiblingFunctionsFollowTheTree has sibling-index() and
// sibling-count() read again as elements come and go.
func TestSiblingFunctionsFollowTheTree(t *testing.T) {
	d := ui.NewDocument()
	d.AddSheet(`li { width: calc(10px * sibling-index()); height: calc(1px * sibling-count()); }`)
	list := ui.Ul()
	for range 3 {
		list.Append(ui.Li(ui.Text("x")))
	}
	d.Body().Append(list)
	d.Layout(geom.V2(800, 600))
	list.Remove(list.Children()[0])
	list.Append(ui.Li(), ui.Li())
	props := map[string]func(*style.Computed) string{"width": everything["width"], "height": everything["height"]}
	matchChromeProps(t, d, props, 3)
}
