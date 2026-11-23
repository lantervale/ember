package css

import (
	"testing"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/ui/internal/dom"
)

func TestAStyledScrollbarTakesItsPartsAndThicknesses(t *testing.T) {
	styled, plain, hidden := el(dom.Div, "", "a"), el(dom.Div, "", "b"), el(dom.Div, "", "c")
	colored, clipped := el(dom.Div, "", "a", "d"), el(dom.Div, "", "a", "e")
	s, warns := Parse("t.css", `div { overflow: scroll; }
		.a::-webkit-scrollbar { width: 12px; height: 8px; }
		.a::-webkit-scrollbar-thumb { background-color: #00f; }
		.a::-webkit-scrollbar-thumb:horizontal { background-color: #f00; }
		.a::-webkit-scrollbar-track-piece:end { background-color: #0f0; }
		.b::-webkit-scrollbar-thumb { background-color: #00f; }
		.c::-webkit-scrollbar { display: none; }
		.d { scrollbar-color: #123 #456; }
		.e { overflow: hidden; }`)
	if len(warns) > 0 {
		t.Fatalf("warnings:\n%s", warnings(warns))
	}
	Compute(page(styled, plain, hidden, colored, clipped), []*Sheet{s})
	c := styleOf(styled)
	if c.ScrollbarWidth != 12 || c.ScrollbarHeight != 8 || c.Scrollbar == nil {
		t.Fatalf("styled: gutters %v×%v, parts %v", c.ScrollbarWidth, c.ScrollbarHeight, c.Scrollbar)
	}
	p := c.Scrollbar
	if p.V.Thumb.Background != color.Hex("#00f") || p.H.Thumb.Background != color.Hex("#f00") {
		t.Errorf("thumbs: vertical %v, horizontal %v", p.V.Thumb.Background, p.H.Thumb.Background)
	}
	if p.V.Start != nil || p.V.End == nil || p.V.End.Background != color.Hex("#0f0") || p.Corner != nil {
		t.Errorf("pieces: start %v, end %v; corner %v", p.V.Start, p.V.End, p.Corner)
	}
	if pl := styleOf(plain); pl.Scrollbar != nil || pl.ScrollbarWidth != barGutter {
		t.Errorf("a thumb alone styles nothing, as Chrome keeps its own bars: %v, %v", pl.Scrollbar, pl.ScrollbarWidth)
	}
	if h := styleOf(hidden); !h.ScrollbarNone || h.ScrollbarWidth != 0 || h.ScrollbarHeight != 0 {
		t.Errorf("display: none: none %v, gutters %v×%v", h.ScrollbarNone, h.ScrollbarWidth, h.ScrollbarHeight)
	}
	if co := styleOf(colored); co.Scrollbar != nil || !co.ScrollbarColored || co.ScrollbarThumb != color.Hex("#123") ||
		co.ScrollbarTrack != color.Hex("#456") {
		t.Errorf("scrollbar-color wins over the parts: %v, %v %v %v", co.Scrollbar, co.ScrollbarColored,
			co.ScrollbarThumb, co.ScrollbarTrack)
	}
	if cl := styleOf(clipped); cl.Scrollbar != nil || cl.ScrollbarWidth != 0 {
		t.Errorf("overflow hidden shows no bar: %v, %v", cl.Scrollbar, cl.ScrollbarWidth)
	}
}

func TestAPartsPseudoClassesWeighAndPrint(t *testing.T) {
	sels, err := ParseSelectors("div::-webkit-scrollbar-thumb:vertical:hover")
	if err != nil {
		t.Fatal(err)
	}
	if got := sels[0].String(); got != "div::-webkit-scrollbar-thumb:vertical:hover" {
		t.Errorf("printed %q", got)
	}
	if got := sels[0].Specificity; got != (Specificity{0, 2, 2}) {
		t.Errorf("specificity %v", got)
	}
	for _, bad := range []string{"div::before:vertical", "div::-webkit-scrollbar:nope", "div:vertical"} {
		if _, bad2 := ParseSelectors(bad); bad2 == nil {
			t.Errorf("%s read, which Chrome refuses", bad)
		}
	}
}

func TestAStableGutterIsKeptWhereTheBoxClips(t *testing.T) {
	auto, hidden, visible := el(dom.Div, "", "a"), el(dom.Div, "", "h"), el(dom.Div, "", "v")
	s, warns := Parse("t.css", `div { scrollbar-gutter: stable; } .a { overflow: auto; } .h { overflow: hidden; }
		.both { scrollbar-gutter: both-edges stable; }`)
	if len(warns) > 0 {
		t.Fatalf("warnings:\n%s", warnings(warns))
	}
	Compute(page(auto, hidden, visible), []*Sheet{s})
	for n, want := range map[*dom.Node]float32{auto: barGutter, hidden: barGutter, visible: 0} {
		if c := styleOf(n); !c.GutterStable || c.ScrollbarWidth != want {
			t.Errorf("%s: stable %v, gutter %v, want %v", describe(n), c.GutterStable, c.ScrollbarWidth, want)
		}
	}
}
