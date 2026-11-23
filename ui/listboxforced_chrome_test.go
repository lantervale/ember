//go:build chrome

package ui_test

import (
	"testing"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
	"github.com/lantervale/ember/ui/internal/dom"
)

// forcedKeys are the system colours a forced listbox's rows are told
// apart by.
//
//nolint:gochecknoglobals // a test's table
var forcedKeys = []string{"Highlight", "HighlightText", "SelectedItem", "SelectedItemText", "GrayText",
	"Canvas", "CanvasText", "Field", "FieldText"}

// TestAForcedListBoxsSelectedRowsTakeTheSystemsColours: under forced
// colours a selected option's row is Highlight under HighlightText,
// focused or not, in either scheme, a disabled one's or a disabled
// listbox's GrayText. Chrome forces its own palette, ui none (the system
// colours stand as they are), so each row's colours are compared by the
// system colour they are: a span of each keyword is on the page, kept as
// it is by Chrome, which never forces a system colour.
func TestAForcedListBoxsSelectedRowsTakeTheSystemsColours(t *testing.T) {
	for _, scheme := range []string{"light", "dark"} {
		t.Run(scheme, func(t *testing.T) { forcedListBox(t, scheme) })
	}
}

func forcedListBox(t *testing.T, scheme string) {
	d := ui.NewDocument(ui.BrowserSheets(), ui.ForcedColors(), ui.CSS(":root { color-scheme: "+scheme+" }"))
	focused := ui.Select(multiple(), ui.ID("focused"), opt("A", ui.Selected()), opt("B"),
		opt("C", ui.Selected(), ui.Disabled()))
	d.Body().Append(
		ui.Select(multiple(), opt("A", ui.Selected()), opt("B"), opt("C", ui.Selected(), ui.Disabled())),
		focused,
		ui.Select(multiple(), ui.Disabled(), opt("A", ui.Selected()), opt("B")),
	)
	keys := map[*ui.Node]string{}
	for _, k := range forcedKeys {
		for _, p := range []string{"color", "background-color"} {
			s := ui.Span(ui.Style(p + ": " + k))
			keys[s] = k
			d.Body().Append(s)
		}
	}
	size := geom.V2(400, 200)
	win := platform.NewWindow(platform.Size(int(size.X), int(size.Y)))
	win.Add(d)
	w := ember.New().AddEntities(win)
	w.RunFrames(2)
	d.Focus(focused)
	w.RunFrames(2)
	props := []string{"color", "background-color"}
	styles, err := chrome.StylesAfter(d, size, props, `document.querySelector("#focused").focus();`)
	if err != nil {
		t.Fatal(err)
	}
	nodes := elementsOf(d)
	if len(nodes) != len(styles) {
		t.Fatalf("%d elements, Chrome styled %d", len(nodes), len(styles))
	}
	// named are the keywords each colour Chrome computed stands for, and
	// ui's value of each keyword.
	named := map[[4]int][]string{}
	ours := map[string][4]int{}
	for i, n := range nodes {
		k, ok := keys[n]
		if !ok {
			continue
		}
		p := 0
		if st, _ := n.Attr("style"); st[0] == 'b' {
			p = 1
		}
		c := chromeColour(t, styles[i][props[p]])
		named[c] = append(named[c], k)
		ours[k] = emberColour(ui.ShownInks(n)[p])
	}
	for i, n := range nodes {
		if _, on := n.Attr("selected"); n.Tag() != dom.Option || !on {
			continue
		}
		inks := ui.ShownInks(n)
		for k, p := range props {
			want := chromeColour(t, styles[i][p])
			if want[3] == 0 {
				continue
			}
			names, got, ok := named[want], emberColour(inks[k]), false
			for _, name := range names {
				ok = ok || ours[name] == got
			}
			if !ok {
				t.Errorf("element %d (option %q) %s: %v, Chrome %v, one of %v", i, n.Text(), p, got, want, names)
			}
		}
	}
}
