//go:build chrome

package css_test

import (
	"encoding/json"
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

func TestMasksComputeAsInChrome(t *testing.T) {
	fsys := fstest.MapFS{"m.css": {Data: []byte(`
div { width: 20px; height: 20px; background-color: #36c; }
.a { mask-image: url(img/panel.png); }
.b { mask-image: url(img/panel.png); mask-size: 12px; mask-position: center; mask-repeat: no-repeat; }
.c { mask-image: url(img/panel.png); mask-size: 50% 8px; mask-position: right 2px bottom; mask-repeat: repeat-x; }
.f { mask-image: url(img/panel.png); mask-position: right 2px bottom 4px; }
.d { mask-image: url(img/panel.png); mask-size: contain; mask-repeat: repeat-y; mask-image: none; }
.e { -webkit-mask-image: url(img/panel.png); -webkit-mask-size: cover; -webkit-mask-repeat: no-repeat; }
`)}}
	d := ui.NewDocument(ui.Stylesheet(fsys, "m.css"), ui.BrowserSheets())
	for _, class := range []string{"a", "b", "c", "d", "e", "f", ""} {
		d.Body().Append(ui.Div(ui.Class(class)))
	}
	raw, err := chrome.Eval(d, geom.V2(800, 600), `return Array.from(document.body.querySelectorAll("div")).map(e => {
		const s = getComputedStyle(e);
		return [s.maskImage, s.maskSize, s.maskRepeat, s.maskPosition];
	});`)
	if err != nil {
		t.Fatal(err)
	}
	var want [][4]string
	if err = json.Unmarshal(raw, &want); err != nil {
		t.Fatal(err)
	}
	d.Layout(geom.V2(800, 600))
	for i, n := range d.Body().Children() {
		s := dom.Data(n).Style
		repeat := "repeat"
		switch {
		case s.MaskNoRepeat:
			repeat = "no-repeat"
		case s.MaskRepeat == style.RepeatX:
			repeat = "repeat-x"
		case s.MaskRepeat == style.RepeatY:
			repeat = "repeat-y"
		}
		ours := [4]string{image(s.MaskImage.Source), sizeCSS(s.MaskSize), repeat,
			positionCSS(s.MaskPosition.X, s.MaskPosition.FromRight) + " " +
				positionCSS(s.MaskPosition.Y, s.MaskPosition.FromBottom)}
		for k := range ours {
			if !same("", ours[k], want[i][k]) {
				t.Errorf("div %d, value %d: ember %q, Chrome %q", i, k, ours[k], want[i][k])
			}
		}
	}
}
