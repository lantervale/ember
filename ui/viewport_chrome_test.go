//go:build chrome

package ui_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
	"github.com/lantervale/ember/ui/internal/dom"
)

const chromePage = `html { margin: 0 0 30px; border: 4px solid #335; padding: 6px; }
#head { position: sticky; top: 10px; height: 20px; }
#badge { position: fixed; right: 30px; bottom: 10px; width: 50px; height: 20px; }
#note { position: absolute; top: 1500px; left: 20px; width: 80px; height: 40px; }
#inner { position: fixed; top: 5px; left: 5px; width: 10px; height: 10px; }
#list { overflow: auto; width: 200px; height: 100px; }
.row { height: 30px; }
`

// chromeDoc is a page some 1,400 points high with a sticky head, a fixed
// badge holding another fixed box, an absolute note below everything in
// flow and a list that scrolls on its own, its viewport scrolled to y.
func chromeDoc(y float32, sheet string, opts ...ui.DocumentOption) *ui.Document {
	d := ui.NewDocument(append([]ui.DocumentOption{ui.ScrollingViewport(), ui.CSS(chromePage + sheet)}, opts...)...)
	list := ui.Div(ui.ID("list"), ui.Div(ui.Style("height: 400px")))
	d.Body().Append(ui.P(ui.Text("Above the head")), ui.Div(ui.ID("head"), ui.Text("Head")),
		ui.Div(ui.ID("badge"), ui.Div(ui.ID("inner"))), ui.Div(ui.ID("note")), list)
	for i := range 40 {
		d.Body().Append(ui.Div(ui.Class("row"), ui.Text(fmt.Sprint("row ", i))))
	}
	dom.Data(d.HTML()).Scroll = geom.V2(0, y)
	dom.Data(list).Scroll = geom.V2(0, 120)
	return d
}

func TestAScrolledViewportMatchesChrome(t *testing.T) {
	for name, c := range map[string]struct {
		y     float32
		sheet string
		opts  []ui.DocumentOption
	}{
		"at the top":                    {0, "", nil},
		"scrolled":                      {700, "", nil},
		"past its end":                  {1e6, "", nil},
		"overlay bars":                  {700, "", []ui.DocumentOption{ui.OverlayScrollbars()}},
		"a page that fits":              {0, ".row, #note { display: none; }", nil},
		"html's overflow hidden":        {300, "html { overflow: hidden; }", nil},
		"body's overflow-y scroll":      {300, "body { overflow-y: scroll; } .row, #note { display: none; }", nil},
		"html and body a viewport high": {900, "html, body { height: 100%; }", nil},
		"a stable gutter, the page fitting": {0, "html { scrollbar-gutter: stable; } .row, #note { display: none; }",
			nil},
		"both edges": {700, "html { scrollbar-gutter: stable both-edges; }", nil},
		"both edges, the page fitting": {0,
			"html { scrollbar-gutter: stable both-edges; } .row, #note { display: none; }", nil},
		"both edges, right to left":     {700, "html { scrollbar-gutter: stable both-edges; direction: rtl; }", nil},
		"body's gutter, not the root's": {700, "body { scrollbar-gutter: stable both-edges; }", nil},
	} {
		t.Run(name, func(t *testing.T) {
			r := chrome.Match(t, chromeDoc(c.y, c.sheet, c.opts...), geom.V2(640, 400))
			t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
		})
	}
}

func TestAViewportScrollsAsFarAsChromes(t *testing.T) {
	for name, sheet := range map[string]string{
		"a tall page":                   "",
		"html and body a viewport high": "html, body { height: 100%; }",
		"a wide box":                    "#note { left: 900px; }",
	} {
		t.Run(name, func(t *testing.T) {
			d := chromeDoc(1e6, sheet)
			out, err := chrome.Eval(d, geom.V2(640, 400), `const e = document.documentElement;
				return [e.scrollWidth, e.scrollHeight, e.clientWidth, e.clientHeight, scrollX, scrollY];`)
			require.NoError(t, err)
			var want [6]float32
			require.NoError(t, json.Unmarshal(out, &want))
			d.Layout(geom.V2(640, 400))
			got := ui.ScrollSize(d, d.HTML())
			assert.InDelta(t, want[0], got.X, 1, "scrollWidth (Chrome %v)", want)
			assert.InDelta(t, want[1], got.Y, 1, "scrollHeight (Chrome %v)", want)
			assert.InDelta(t, want[5], -d.HTML().Box().Y, 1, "scrolled to its end (Chrome %v)", want)
		})
	}
}

func TestAScrolledViewportIsHitAsInChrome(t *testing.T) {
	points := []geom.Vec2{{X: 100, Y: 15}, {X: 590, Y: 380}, {X: 300, Y: 200}, {X: 8, Y: 8}, {X: 50, Y: 300}}
	script := "return ["
	for _, p := range points {
		script += fmt.Sprintf(`(e => e.id || e.className || e.tagName)(document.elementFromPoint(%g, %g)),`, p.X, p.Y)
	}
	d := chromeDoc(1205, "", ui.OverlayScrollbars())
	out, err := chrome.Eval(d, geom.V2(640, 400), script+"];")
	require.NoError(t, err)
	var want []string
	require.NoError(t, json.Unmarshal(out, &want))
	d.Layout(geom.V2(640, 400))
	for i, p := range points {
		n := ui.HitAt(d, p)
		require.NotNil(t, n, "%v: Chrome hits %s", p, want[i])
		got := n.ID()
		if got == "" && len(n.Classes()) > 0 {
			got = n.Classes()[0]
		}
		if got == "" {
			got = map[bool]string{true: "HTML", false: "BODY"}[n == d.HTML()]
		}
		assert.Equal(t, want[i], got, "at %v", p)
	}
	assert.Contains(t, want, "head")
	assert.Contains(t, want, "badge")
}
