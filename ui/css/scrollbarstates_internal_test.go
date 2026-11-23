package css

import (
	"testing"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

func TestAPartsHoverAndActiveFollowThePartThePointerIsOn(t *testing.T) {
	n := el(dom.Div, "", "a")
	s, warns := Parse("t.css", `div { overflow: scroll; }
		.a::-webkit-scrollbar { width: 12px; height: 12px; background-color: #111; }
		.a::-webkit-scrollbar:hover { background-color: #222; }
		.a::-webkit-scrollbar-track:hover { background-color: #333; }
		.a::-webkit-scrollbar-thumb { background-color: #444; }
		.a::-webkit-scrollbar-thumb:hover { background-color: #555; }
		.a::-webkit-scrollbar-thumb:active { background-color: #666; }
		.a::-webkit-scrollbar-track-piece:decrement:no-button { background-color: #777; }
		.a::-webkit-scrollbar-button:end:increment { display: block; height: 9px; }
		.a::-webkit-scrollbar-button:hover { background-color: #888; }
		.a::-webkit-scrollbar-corner:enabled { background-color: #999; }`)
	if len(warns) > 0 {
		t.Fatalf("warnings:\n%s", warnings(warns))
	}
	sheets := []*Sheet{s}
	root := page(n)
	dom.Data(n).BarHover = style.BarThumb
	Compute(root, sheets)
	p := styleOf(n).Scrollbar
	for name, c := range map[string][2]color.Color{
		"the vertical bar, a part of it hovered": {p.V.Bar.Background, color.Hex("#222")},
		"its track, its thumb hovered":           {p.V.Track.Background, color.Hex("#333")},
		"the thumb":                              {p.V.Thumb.Background, color.Hex("#555")},
		"the horizontal bar":                     {p.H.Bar.Background, color.Hex("#111")},
		"the horizontal thumb":                   {p.H.Thumb.Background, color.Hex("#444")},
		"the piece before the thumb":             {p.V.Start.Background, color.Hex("#777")},
	} {
		if c[0] != c[1] {
			t.Errorf("%s: %v, want %v", name, c[0], c[1])
		}
	}
	if p.V.End != nil || p.Corner != nil {
		t.Errorf("the piece after the thumb isn't a decrement %v, a corner has no :enabled %v", p.V.End, p.Corner)
	}
	if b := p.V.Buttons; b[0] != nil || b[1] != nil || b[2] != nil || b[3] == nil || b[3].Display != style.Block {
		t.Errorf("buttons %v, want the one going forward at the end alone", b)
	}
	data := dom.Data(n)
	for _, p := range []style.BarPart{data.BarHover, style.BarThumb | style.BarHorizontal} {
		data.BarsStale.Mark(p)
	}
	data.BarHover = style.BarThumb | style.BarHorizontal
	data.BarPressed = style.BarThumb | style.BarHorizontal
	dom.Invalidate(n, ":scrollbar")
	Compute(root, sheets)
	p = styleOf(n).Scrollbar
	if p.V.Thumb.Background != color.Hex("#444") || p.H.Thumb.Background != color.Hex("#666") {
		t.Errorf("thumbs %v and %v, the horizontal one pressed", p.V.Thumb.Background, p.H.Thumb.Background)
	}
}

func TestScrollbarWidthThinAsksForTheSystemsBars(t *testing.T) {
	n := el(dom.Div, "", "a")
	s, warns := Parse("t.css", `div { overflow: scroll; scrollbar-width: thin; }
		.a::-webkit-scrollbar { width: 12px; }`)
	if len(warns) > 0 {
		t.Fatalf("warnings:\n%s", warnings(warns))
	}
	Compute(page(n), []*Sheet{s})
	if c := styleOf(n); c.Scrollbar != nil || !c.ScrollbarOverlay || c.ScrollbarWidth != 0 {
		t.Errorf("parts %v, overlay %v, gutter %v", c.Scrollbar, c.ScrollbarOverlay, c.ScrollbarWidth)
	}
}

func TestAResizerIsStyledOnABoxTheUserMayResize(t *testing.T) {
	clips, visible := el(dom.Div, "", "a"), el(dom.Div, "", "b")
	s, warns := Parse("t.css", `div { resize: both; } .a { overflow: auto; }
		div::-webkit-resizer { background-color: #f00; }
		div::-webkit-resizer:window-inactive { background-color: #00f; }`)
	if len(warns) > 0 {
		t.Fatalf("warnings:\n%s", warnings(warns))
	}
	Compute(page(clips, visible), []*Sheet{s})
	if c := styleOf(clips); !c.Resizable() || c.Resizer == nil || c.Resizer.Background != color.Hex("#f00") {
		t.Errorf("resizable %v, resizer %v", c.Resize, c.Resizer)
	}
	if c := styleOf(visible); c.Resizable() || c.Resizer != nil {
		t.Errorf("a box that doesn't clip: resizable %v, resizer %v", c.Resize, c.Resizer)
	}
}

// A box's :hover restyles its bars' parts only as Chrome's: none while
// the pointer is off them, the part reached with its bar and track, all
// of them once the box's own style, or its bar's, changes.
func TestABoxsHoverRestylesItsPartsAsThePointerReachesThem(t *testing.T) {
	n := el(dom.Div, "", "a")
	s, warns := Parse("t.css", `div { overflow: scroll; }
		.a::-webkit-scrollbar { width: 12px; height: 12px; }
		.a::-webkit-scrollbar-thumb { background-color: #111; }
		.a:hover::-webkit-scrollbar-thumb { background-color: #222; }
		.a::-webkit-scrollbar-track-piece { background-color: #333; }
		.a:hover::-webkit-scrollbar-track-piece { background-color: #444; }
		.b:hover { color: red; }`)
	if len(warns) > 0 {
		t.Fatalf("warnings:\n%s", warnings(warns))
	}
	sheets := []*Sheet{s}
	root := page(n)
	Compute(root, sheets)
	n.SetState(dom.Hover, true)
	Compute(root, sheets)
	p := styleOf(n).Scrollbar
	if p.V.Thumb.Background != color.Hex("#111") || p.V.End.Background != color.Hex("#333") {
		t.Errorf("box hovered, pointer off the bars: thumb %v, piece %v", p.V.Thumb.Background, p.V.End.Background)
	}
	data := dom.Data(n)
	data.BarsStale.Mark(style.BarForwardTrack)
	data.BarHover = style.BarForwardTrack
	dom.Invalidate(n, ":scrollbar")
	Compute(root, sheets)
	p = styleOf(n).Scrollbar
	if p.V.Thumb.Background != color.Hex("#111") || p.V.End.Background != color.Hex("#444") ||
		p.H.End.Background != color.Hex("#333") {
		t.Errorf("pointer on the piece: thumb %v, piece %v, the other bar's %v", p.V.Thumb.Background,
			p.V.End.Background, p.H.End.Background)
	}
	n.AddClass("b")
	Compute(root, sheets)
	if p = styleOf(n).Scrollbar; p.V.Thumb.Background != color.Hex("#222") || p.H.End.Background != color.Hex("#444") {
		t.Errorf("the box's own style changed: thumb %v, the other bar's piece %v", p.V.Thumb.Background,
			p.H.End.Background)
	}
}
