//go:build chrome

package css_test

import (
	"encoding/json"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

// layersCSS writes each layer of s's background as getComputedStyle
// does: the longhands' lists as long as the pictures'.
func layersCSS(s *style.Computed) [8]string {
	var parts [8][]string
	for i := range s.BackgroundLayers() {
		l := s.BackgroundLayer(i)
		attach := [...]string{"scroll", "fixed", "local"}[l.Attachment]
		origin := [...]string{"padding-box", "border-box", "padding-box", "content-box"}[l.Origin]
		clip := [...]string{"border-box", "border-box", "padding-box", "content-box"}[l.Clip]
		for k, v := range []string{image(l.Image.Source), positionCSS(l.Position.X, l.Position.FromRight),
			positionCSS(l.Position.Y, l.Position.FromBottom), sizeCSS(l.Size), repeatCSS(l.Repeat), attach, origin,
			clip} {
			parts[k] = append(parts[k], v)
		}
	}
	var out [8]string
	for k := range parts {
		out[k] = strings.Join(parts[k], ", ")
	}
	return out
}

func TestBackgroundLayersComputeAsInChrome(t *testing.T) {
	fsys := fstest.MapFS{"l.css": {Data: []byte(`
div { width: 20px; height: 20px; }
.short { background: url(img/a.png) no-repeat center / 10px, url(img/b.png) fixed,
  url(img/c.png) space round content-box padding-box #f00; }
.lists { background-image: url(img/a.png), url(img/b.png), url(img/c.png);
  background-position: 1px 2px, right 3px bottom; background-repeat: space;
  background-clip: padding-box, content-box, border-box, padding-box; background-size: cover, 4px 50%; }
.none { background-image: none, url(img/b.png); background-attachment: local, fixed, scroll; }
.repeats { background-image: url(img/a.png), url(img/b.png), url(img/c.png);
  background-repeat: repeat no-repeat, round space, space space; }
.over { background: url(img/a.png) round; background-origin: border-box, content-box; }
.lone { background-repeat: no-repeat, repeat-y; }
`)}}
	d := ui.NewDocument(ui.Stylesheet(fsys, "l.css"))
	for _, class := range []string{"short", "lists", "none", "repeats", "over", "lone"} {
		d.Body().Append(ui.Div(ui.Class(class)))
	}
	raw, err := chrome.Eval(d, geom.V2(800, 600), `return Array.from(document.body.querySelectorAll("div")).map(e => {
		const s = getComputedStyle(e);
		return [s.backgroundImage, s.backgroundPositionX, s.backgroundPositionY, s.backgroundSize, s.backgroundRepeat,
			s.backgroundAttachment, s.backgroundOrigin, s.backgroundClip, s.backgroundColor];
	});`)
	if err != nil {
		t.Fatal(err)
	}
	var want [][9]string
	if err = json.Unmarshal(raw, &want); err != nil {
		t.Fatal(err)
	}
	d.Layout(geom.V2(800, 600))
	for i, n := range d.Body().Children() {
		s := dom.Data(n).Style
		layers := layersCSS(s)
		ours := append(layers[:], rgb(s.Background))
		for k := range ours {
			if !same("", ours[k], want[i][k]) {
				t.Errorf("div %d, value %d: ember %q, Chrome %q", i, k, ours[k], want[i][k])
			}
		}
	}
}
