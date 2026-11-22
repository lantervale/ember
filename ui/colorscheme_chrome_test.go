//go:build chrome

package ui_test

import (
	"fmt"
	"math"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
)

// systemColourNames are the system colours compared with Chrome's. ui's
// are Chrome's on macOS, the same on every system; Highlight,
// HighlightText, SelectedItem and SelectedItemText are each system's own
// in Chrome, and lightOwn are Windows' in its light scheme.
//
//nolint:gochecknoglobals // a test's table
var (
	systemColourNames = []string{
		"Canvas", "CanvasText", "Field", "FieldText", "ButtonFace", "ButtonText", "ButtonBorder", "LinkText",
		"VisitedText", "ActiveText", "GrayText", "Mark", "MarkText", "AccentColor", "AccentColorText", "ActiveBorder",
		"ActiveCaption", "AppWorkspace", "Background", "ButtonHighlight", "ButtonShadow", "CaptionText",
		"InactiveBorder", "InactiveCaption", "InactiveCaptionText", "InfoBackground", "InfoText", "Menu", "MenuText",
		"Scrollbar", "ThreeDDarkShadow", "ThreeDFace", "ThreeDHighlight", "ThreeDLightShadow", "ThreeDShadow",
		"Window", "WindowFrame", "WindowText",
	}
	lightOwn = []string{
		"ButtonFace", "LinkText", "VisitedText", "ActiveText", "GrayText", "ButtonHighlight", "ButtonShadow",
		"ThreeDFace",
	}
)

// sharedColours reports whether Chrome answers the system colour name in
// scheme as on macOS here.
func sharedColours(scheme, name string) bool {
	return runtime.GOOS == "darwin" || scheme == "dark" || !slices.Contains(lightOwn, name)
}

// schemePage holds elements whose colours differ by scheme: every
// system colour in both, light-dark() alone, in a gradient's place and
// through a variable, the color-scheme values, and what a child inherits
// from a parent of the other scheme.
func schemePage(doc *ui.Document) {
	body := doc.Body()
	for _, scheme := range []string{"light", "dark"} {
		for _, name := range systemColourNames {
			if !sharedColours(scheme, name) {
				continue
			}
			body.Append(ui.Div(ui.Style(fmt.Sprintf("color-scheme: %s; color: %s; background-color: %s",
				scheme, name, name))))
		}
	}
	for _, scheme := range []string{"normal", "light", "dark", "light dark", "dark light", "only dark", "light only",
		"sepia", "sepia dark"} {
		body.Append(ui.Div(ui.Style("color-scheme: " + scheme +
			"; color: light-dark(#f00, #00f); background-color: Canvas; border-top-color: light-dark(Field, LinkText)")))
	}
	body.Append(
		ui.Div(ui.Style("color-scheme: dark; color: CanvasText"), ui.Span(ui.Style("color-scheme: light"))),
		ui.Div(ui.Style("color-scheme: dark; color: light-dark(red, CanvasText)"),
			ui.Span(ui.Style("color-scheme: light"))),
		ui.Div(ui.Style("color-scheme: dark; --ink: light-dark(red, blue)"),
			ui.Span(ui.Style("color-scheme: light; color: var(--ink)")),
			ui.Span(ui.Style("color: var(--ink); background-color: light-dark(rgb(1 2 3 / 50%), Mark)"))),
		ui.Div(ui.Style("color-scheme: dark"), ui.Span(ui.Style("color: initial"))),
	)
}

// chromeColour reads a colour getComputedStyle wrote.
func chromeColour(t *testing.T, s string) [4]int {
	t.Helper()
	inner := s[strings.IndexByte(s, '(')+1 : strings.LastIndexByte(s, ')')]
	parts := strings.Split(inner, ",")
	out := [4]int{3: 100}
	for i, p := range parts {
		v, err := strconv.ParseFloat(strings.TrimSpace(p), 64)
		if err != nil {
			t.Fatalf("%q: %v", s, err)
		}
		if i == 3 {
			v *= 100
		}
		out[i] = int(math.Round(v))
	}
	return out
}

// elementsOf are html, body and body's elements, in the order Chrome's
// styles come.
func elementsOf(doc *ui.Document) []*ui.Node {
	out := []*ui.Node{doc.HTML(), doc.Body()}
	doc.Body().Walk(func(n *ui.Node) bool {
		if n != doc.Body() && !n.IsText() {
			out = append(out, n)
		}
		return true
	})
	return out
}

func emberColour(c color.Color) [4]int {
	s := c.SRGB()
	return [4]int{int(math.Round(float64(s[0]) * 255)), int(math.Round(float64(s[1]) * 255)),
		int(math.Round(float64(s[2]) * 255)), int(math.Round(float64(c.A) * 100))}
}

// TestChromeResolvesSystemColoursInEachScheme has Chrome compute every
// element's colours for a Document that prefers light and one that
// prefers dark, under a root of each color-scheme, and ember match each
// to the unit: system colours, light-dark(), the root's text and what
// children inherit.
func TestChromeResolvesSystemColoursInEachScheme(t *testing.T) {
	for _, prefers := range []platform.Appearance{platform.AppearanceLight, platform.AppearanceDark} {
		for _, root := range []string{"normal", "light dark", "dark"} {
			t.Run(fmt.Sprintf("%v %s", prefers, root), func(t *testing.T) {
				doc := ui.NewDocument(ui.ColorScheme(prefers), ui.CSS(":root { color-scheme: "+root+" }"))
				schemePage(doc)
				size := geom.V2(400, 300)
				doc.Layout(size)
				props := []string{"color", "background-color", "border-top-color"}
				styles, err := chrome.Styles(doc, size, props)
				if err != nil {
					t.Fatal(err)
				}
				nodes := elementsOf(doc)
				if len(nodes) != len(styles) {
					t.Fatalf("%d elements, Chrome styled %d", len(nodes), len(styles))
				}
				for i, n := range nodes {
					inks := ui.ShownInks(n)
					for k, p := range props {
						if i < 2 && k == 2 {
							continue
						}
						want := chromeColour(t, styles[i][p])
						if got := emberColour(inks[k]); got != want {
							style, _ := n.Attr("style")
							t.Errorf("element %d (%s) %s: %v, Chrome %v", i, style, p, got, want)
						}
					}
				}
			})
		}
	}
}

// TestChromeColoursABrowsersControlsInTheirScheme has Chrome compute the
// colours of controls in a page of each scheme, their user agent sheet's
// light-dark() and system colours resolved, and ember match them.
func TestChromeColoursABrowsersControlsInTheirScheme(t *testing.T) {
	for _, scheme := range []string{"light", "dark"} {
		t.Run(scheme, func(t *testing.T) {
			if !sharedColours(scheme, "ButtonFace") {
				t.Skip("Chrome's light controls take the system's colours here")
			}
			doc := ui.NewDocument(ui.BrowserSheets(), ui.CSS(":root { color-scheme: "+scheme+" } body { color: #c00 }"))
			off := ui.Attr("disabled", "")
			doc.Body().Append(
				ui.Button(ui.Text("Go")), ui.Button(off, ui.Text("Go")), ui.Input(), ui.Input(off),
				ui.Input(ui.Type("checkbox"), ui.Attr("checked", "")), ui.Input(ui.Type("radio"), ui.Attr("checked", "")),
				ui.Input(ui.Type("range")), ui.Input(ui.Type("submit")), ui.Textarea(), ui.Select(ui.Option(ui.Text("a"))),
				ui.A(ui.Attr("href", "#x"), ui.Text("link")), ui.Mark(ui.Text("mark")), ui.Progress(),
				ui.Input(ui.Type("checkbox"), ui.Attr("switch", ""), ui.Attr("checked", "")),
			)
			size := geom.V2(600, 300)
			doc.Layout(size)
			props := []string{"color", "background-color", "border-top-color"}
			styles, err := chrome.Styles(doc, size, props)
			if err != nil {
				t.Fatal(err)
			}
			elements := elementsOf(doc)
			if len(elements) != len(styles) {
				t.Fatalf("%d elements, Chrome styled %d", len(elements), len(styles))
			}
			for i, n := range elements {
				inks := ui.ShownInks(n)
				for k, p := range props {
					if got, want := emberColour(inks[k]), chromeColour(t, styles[i][p]); got != want {
						t.Errorf("%s %s: %v, Chrome %v", n.Tag(), p, got, want)
					}
				}
			}
		})
	}
}
