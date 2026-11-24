//go:build chrome

package ui_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
	"github.com/lantervale/ember/ui/internal/style"
)

// Ranges whose lines are vertical, and of appearance slider-vertical, are
// sized as Chrome's: their track's length down the box, a range of
// appearance none as thick as its slider container round its track.
func TestVerticalRangesAreSizedAsChromes(t *testing.T) {
	const sheet = `body { font: 16px "Go"; } .none { appearance: none; }
.lr { writing-mode: vertical-lr; } .rl { writing-mode: vertical-rl; direction: rtl; }
.sl { writing-mode: sideways-lr; } .sv { appearance: slider-vertical; width: 20px; height: 100px; }
.c::-webkit-slider-container { padding: 3px; border: 1px solid #000; }
.t::-webkit-slider-runnable-track { width: 6px; height: 6px; }
.th::-webkit-slider-thumb { appearance: none; width: 12px; height: 4px; }`
	d := ui.NewDocument(ui.CSS(sheet))
	rng := func(class ...string) *ui.Node { return ui.Input(ui.Type("range"), ui.Class(class...)) }
	d.Body().Append(
		ui.Div(ui.Text("x"), rng("lr"), rng("rl"), rng("sl"), rng("sv"), ui.Text("x")),
		ui.Div(ui.Text("x"), rng("none", "lr"), rng("none", "rl", "t"), rng("none", "lr", "th", "c"), ui.Text("x")),
		ui.Div(ui.Text("x"), rng("none", "c"), rng("none", "c", "t", "th"), ui.Text("x")),
	)
	r := chrome.Match(t, d, geom.V2(500, 500))
	t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
}

// unhovered is s with its hover left out.
func unhovered(s style.Shade) style.Shade {
	if s == style.ShadeHovered {
		return style.ShadeRest
	}
	return s
}

// TestAccentShadesAreChromes hovers and presses Chrome's own ranges and
// checkboxes of several accents and reads the colours it fills them
// with: the range's fill and thumb, the checkbox's box, each within a
// level of style.AccentShade's.
func TestAccentShadesAreChromes(t *testing.T) {
	accents := []string{"#ee0044", "#33aa77", "#808080", "#8b8b8b", "#204020", "#ffe080", "#3060c0", "#ff8800",
		"#000000", "#ffffff", "#0075ff"}
	var page strings.Builder
	page.WriteString(`<!DOCTYPE html><html><head><style>body { margin: 0; }
input { position: absolute; margin: 0; } [type=checkbox] { width: 20px; height: 20px; }</style></head><body>`)
	for i, a := range accents {
		fmt.Fprintf(&page, `<input type="range" value="40" style="left: 10px; top: %dpx; accent-color: %s">`, i*30, a)
		fmt.Fprintf(&page, `<input type="checkbox" checked style="left: 200px; top: %dpx; accent-color: %s">`, i*30, a)
	}
	page.WriteString("</body></html>")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(page.String()))
	}))
	defer srv.Close()
	s := chrome.Open(t, srv.URL, geom.V2(400, float32(len(accents)*30)), 1)
	at := func(at geom.Vec2) color.Color {
		s.Eval(`new Promise(r => requestAnimationFrame(() => requestAnimationFrame(r)))`, nil)
		r, g, b, _ := s.Shot().At(int(at.X), int(at.Y)).RGBA()
		return color.RGB8(uint8(r>>8), uint8(g>>8), uint8(b>>8))
	}
	near := func(what string, got, want color.Color) {
		g, w := got.SRGB(), want.SRGB()
		for i := range g {
			if d := (g[i] - w[i]) * 255; d > 1.5 || d < -1.5 {
				t.Errorf("%s: Chrome's %s, ui's %s", what, got.Hex(), want.Hex())
				return
			}
		}
	}
	away := geom.V2(390, 5)
	for i, hex := range accents {
		a := color.Hex(hex)
		y := float32(i*30) + 8
		// the thumb's middle at 40, and the fill nearer the start
		thumb, fill, box := geom.V2(10+8+0.4*113, y), geom.V2(20, y), geom.V2(203, float32(i*30)+3)
		for _, c := range []struct {
			name             string
			pointer          geom.Vec2
			press            bool
			thumb, fill, box style.Shade
		}{
			{"over the thumb", thumb, false, style.ShadeHovered, style.ShadeHovered, style.ShadeRest},
			{"over the track", geom.V2(120, y), false, style.ShadeRest, style.ShadeHovered, style.ShadeRest},
			{"a press on the thumb", thumb, true, style.ShadePressed, style.ShadePressed, style.ShadeRest},
			{"over the checkbox", box, false, style.ShadeRest, style.ShadeRest, style.ShadeHovered},
			{"a press on the checkbox", box, true, style.ShadeRest, style.ShadeRest, style.ShadePressed},
		} {
			if runtime.GOOS == "linux" {
				// Chrome on Linux draws a hovered accent as at rest
				// (docs/05-ui/10-missing.md).
				c.thumb, c.fill, c.box = unhovered(c.thumb), unhovered(c.fill), unhovered(c.box)
			}
			s.Mouse("mouseMoved", away, 0)
			s.Mouse("mouseMoved", c.pointer, 0)
			if c.press {
				s.Mouse("mousePressed", c.pointer, 1)
			}
			near(fmt.Sprintf("%s %s: the thumb", hex, c.name), at(thumb), style.AccentShade(a, c.thumb, style.LightRange(a)))
			near(fmt.Sprintf("%s %s: the fill", hex, c.name), at(fill), style.AccentShade(a, c.fill, style.LightRange(a)))
			near(fmt.Sprintf("%s %s: the box", hex, c.name), at(box), style.AccentShade(a, c.box, style.LightCheck(a)))
			if c.press {
				s.Mouse("mouseReleased", c.pointer, 1)
				if c.pointer == box {
					// the click unchecked it; a second checks it again
					s.Mouse("mousePressed", c.pointer, 1)
					s.Mouse("mouseReleased", c.pointer, 1)
				}
			}
		}
	}
}
