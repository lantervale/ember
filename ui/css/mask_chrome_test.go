//go:build chrome

package css_test

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

func sizeCSS(s style.BackgroundSize) string {
	switch s.Scale {
	case style.ScaleCover:
		return "cover"
	case style.ScaleContain:
		return "contain"
	case style.ScaleNone: // sized by its width and height
	}
	if s.Height.Kind == style.Auto && s.Width.Kind == style.Auto {
		return "auto"
	}
	return lengthCSS(s.Width) + " " + lengthCSS(s.Height)
}

// repeatCSS is a background-repeat as Chrome computes it: one word when
// it says both axes, two otherwise.
func repeatCSS(r style.BackgroundRepeat) string {
	words := [...]string{"repeat", "no-repeat", "space", "round"}
	switch {
	case r.X == r.Y:
		return words[r.X]
	case r.X == style.TileRepeat && r.Y == style.TileNoRepeat:
		return "repeat-x"
	case r.X == style.TileNoRepeat && r.Y == style.TileRepeat:
		return "repeat-y"
	}
	return words[r.X] + " " + words[r.Y]
}

// gradients finds each gradient of a computed value, to be written by
// its function's name alone.
var gradients = regexp.MustCompile(`((?:linear|radial|conic)-gradient)\((?:[^()]|\([^()]*\))*\)`)

// masksCSS writes each layer of s's mask as getComputedStyle does: the
// longhands' lists as long as the pictures', a gradient by its name.
func masksCSS(s *style.Computed) [8]string {
	var parts [8][]string
	boxes := [...]string{"border-box", "border-box", "padding-box", "content-box", "fill-box", "stroke-box", "view-box",
		"no-clip"}
	for i := range s.MaskLayers() {
		l := s.MaskLayer(i)
		img := image(l.Image.Source)
		if g := l.Image.Gradient; g != nil {
			img = [...]string{"linear-gradient", "conic-gradient", "radial-gradient"}[g.Kind]
		}
		mode := [...]string{"match-source", "alpha", "luminance"}[l.Mode]
		composite := style.MaskCompositeWords[l.Composite]
		for k, v := range []string{img, positionCSS(l.Position.X, l.Position.FromRight) + " " +
			positionCSS(l.Position.Y, l.Position.FromBottom), sizeCSS(l.Size), repeatCSS(l.Repeat), boxes[l.Origin],
			boxes[l.Clip], mode, composite} {
			parts[k] = append(parts[k], v)
		}
	}
	var out [8]string
	for k := range parts {
		out[k] = strings.Join(parts[k], ", ")
	}
	return out
}

func TestMasksComputeAsInChrome(t *testing.T) {
	fsys := fstest.MapFS{"m.css": {Data: []byte(`
div { width: 20px; height: 20px; background-color: #36c; }
.a { mask-image: url(img/panel.png); }
.b { mask-image: url(img/panel.png); mask-size: 12px; mask-position: center; mask-repeat: no-repeat; }
.c { mask-image: url(img/panel.png); mask-size: 50% 8px; mask-position: right 2px bottom; mask-repeat: repeat-x; }
.f { mask-image: url(img/panel.png); mask-position: right 2px bottom 4px; }
.d { mask-image: url(img/panel.png); mask-size: contain; mask-repeat: repeat-y; mask-image: none; }
.e { -webkit-mask-image: url(img/panel.png); -webkit-mask-size: cover; -webkit-mask-repeat: no-repeat; }
.g { mask-image: url(img/panel.png), linear-gradient(#000, transparent), radial-gradient(#000, #fff 4px);
  mask-repeat: space round, round; mask-mode: luminance, alpha; mask-origin: padding-box, content-box, fill-box;
  mask-clip: no-clip, stroke-box; mask-composite: subtract, intersect, exclude; mask-position: 1px 2px, 30% bottom; }
.h { mask: linear-gradient(red, blue) center / 10px no-repeat padding-box content-box subtract luminance,
  url(img/panel.png) view-box; }
.i { mask-mode: luminance; mask-composite: intersect; mask: url(img/panel.png) border-box no-clip; }
.j { mask-image: url(img/panel.png); mask: none; }
.k { -webkit-mask: conic-gradient(#000, #fff) 4px 5px / auto 6px repeat-y content-box; }
.l { -webkit-mask-position: 3px 4px; -webkit-mask-clip: padding-box; -webkit-mask-origin: content-box; }
.m { mask-composite: subtract; -webkit-mask-composite: xor; }
.n { mask-image: none, url(img/panel.png); mask-size: 1px, 2px, 3px; }
.o { -webkit-mask: url(img/panel.png) right 4px top / 10px, linear-gradient(red, blue) center bottom 2px;
  -webkit-mask-composite: destination-in, plus-lighter; }
.p { -webkit-mask-image: url(img/panel.png), url(img/panel.png); -webkit-mask-position: left 3px bottom, center top 20%;
  -webkit-mask-composite: source-atop, xor; mask-composite: copy; }
.q { -webkit-mask: url(img/panel.png) no-clip; -webkit-mask-composite: plus-darker; }
.r { -webkit-mask: url(img/panel.png) xor; }
.s { -webkit-mask: url(img/panel.png) add; }
`)}}
	d := ui.NewDocument(ui.Stylesheet(fsys, "m.css"), ui.BrowserSheets())
	classes := []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l", "m", "n", "o", "p", "q", "r", "s", ""}
	for _, class := range classes {
		d.Body().Append(ui.Div(ui.Class(class)))
	}
	raw, err := chrome.Eval(d, geom.V2(800, 600), `return Array.from(document.body.querySelectorAll("div")).map(e => {
		const s = getComputedStyle(e);
		return [s.maskImage, s.maskPosition, s.maskSize, s.maskRepeat, s.maskOrigin, s.maskClip, s.maskMode,
			s.maskComposite];
	});`)
	if err != nil {
		t.Fatal(err)
	}
	var want [][8]string
	if err = json.Unmarshal(raw, &want); err != nil {
		t.Fatal(err)
	}
	d.Layout(geom.V2(800, 600))
	for i, n := range d.Body().Children() {
		ours := masksCSS(dom.Data(n).Style)
		want[i][0] = gradients.ReplaceAllString(want[i][0], "$1")
		for k := range ours {
			if !same("", ours[k], want[i][k]) {
				t.Errorf("div %q, value %d: ember %q, Chrome %q", classes[i], k, ours[k], want[i][k])
			}
		}
	}
}
