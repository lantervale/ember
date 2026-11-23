package css

import (
	"strings"
	"testing"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

func input(typ, class, inline string) *dom.Node {
	n := el(dom.Input, "", strings.Fields(class)...)
	n.SetAttr("type", typ)
	if inline != "" {
		n.SetStyle(inline)
	}
	return n
}

// A control keeps its own look while its appearance isn't none; with none
// the user agent's rules of that look hold no more, and it is drawn from
// the rules Chrome's sheet gives every control.
func TestAppearanceNoneTakesAControlsOwnLookOff(t *testing.T) {
	box, bare := input("checkbox", "", ""), input("checkbox", "bare", "")
	checked := input("checkbox", "bare", "")
	checked.SetState(dom.Checked, true)
	sized := input("checkbox", "", "appearance: none; width: 20px")
	compat := input("checkbox", "", "appearance: checkbox")
	div := el(dom.Div, "", "")
	Compute(page(box, bare, checked, sized, compat, div),
		[]*Sheet{sheet(t, `.bare { appearance: none; }`)})

	if c := styleOf(box); c.Appearance != style.AppearanceAuto || c.Size.Width != style.Px(13) || c.Border.Top != 1 {
		t.Errorf("a checkbox: %v, width %v, border %v", c.Appearance, c.Size.Width, c.Border.Top)
	}
	for name, n := range map[string]*dom.Node{"bare": bare, "checked": checked} {
		c := styleOf(n)
		if c.Appearance != style.AppearanceNone || c.Size.Width.Kind != style.Auto || c.Border.Top != 0 ||
			c.Background.A != 0 || c.Radius.TopLeft.Width != style.Px(0) {
			t.Errorf("%s checkbox of appearance none: width %v, border %v, background %v, radius %v", name,
				c.Size.Width, c.Border.Top, c.Background, c.Radius.TopLeft.Width)
		}
	}
	if c := styleOf(sized); c.Size.Width != style.Px(20) || c.Size.Height.Kind != style.Auto {
		t.Errorf("sized: %v × %v", c.Size.Width, c.Size.Height)
	}
	if c := styleOf(compat); c.Appearance != style.AppearanceCheckbox || c.Size.Width != style.Px(13) {
		t.Errorf("appearance: checkbox keeps the look: %v, width %v", c.Appearance, c.Size.Width)
	}
	if c := styleOf(div); c.Appearance != style.AppearanceNone {
		t.Errorf("a div: %v", c.Appearance)
	}
}

// A field, a select and a button of appearance none take Chrome's sheet's
// borders and backgrounds, a select no chevron.
func TestAppearanceNoneTakesAFieldsOwnLookOff(t *testing.T) {
	field, flat := input("text", "", ""), input("text", "bare", "")
	sel, plain := el(dom.Select, "", ""), el(dom.Select, "", "bare")
	btn, flatBtn := el(dom.Button, "", ""), el(dom.Button, "", "bare")
	Compute(page(field, flat, sel, plain, btn, flatBtn), []*Sheet{sheet(t, `.bare { appearance: none; }`)})
	if c := styleOf(field); c.Border.Top != 1 || c.Radius.TopLeft.Width != style.Px(2) {
		t.Errorf("a field: border %v, radius %v", c.Border.Top, c.Radius.TopLeft.Width)
	}
	if c := styleOf(flat); c.Border.Top != 2 || c.BorderStyle.Top != style.BorderInset ||
		c.Radius.TopLeft.Width != style.Px(0) || c.Background != color.White {
		t.Errorf("a field of appearance none: border %v %v, radius %v, background %v", c.Border.Top,
			c.BorderStyle.Top, c.Radius.TopLeft.Width, c.Background)
	}
	if c := styleOf(plain); len(c.BackgroundImages) != 1 || c.BackgroundImages[0].Source != "" ||
		c.Padding.Right != style.Px(2) {
		t.Errorf("a select of appearance none has no chevron: %v, padding %v", c.BackgroundImages, c.Padding.Right)
	}
	if c := styleOf(sel); len(c.BackgroundImages) != 1 || c.BackgroundImages[0].Source == "" ||
		c.Padding.Right != style.Px(20) {
		t.Errorf("a select: %v, padding %v", c.BackgroundImages, c.Padding.Right)
	}
	if c := styleOf(flatBtn); c.Border.Top != 2 || c.BorderStyle.Top != style.BorderOutset ||
		c.Background != color.Hex("#efefef") {
		t.Errorf("a button of appearance none: border %v %v, background %v", c.Border.Top, c.BorderStyle.Top,
			c.Background)
	}
	if c := styleOf(btn); c.Border.Top != 1 || c.BorderStyle.Top != style.BorderSolid {
		t.Errorf("a button: border %v %v", c.Border.Top, c.BorderStyle.Top)
	}
}

func TestTheUserAgentsNativeClassWeighsAsAClassAndPrints(t *testing.T) {
	sels, err := ParseSelectors(`input[type="checkbox"]:checked:-ember-native`)
	if err != nil {
		t.Fatal(err)
	}
	if !sels[0].native || sels[0].Specificity != (Specificity{0, 3, 1}) {
		t.Errorf("native %v, specificity %v", sels[0].native, sels[0].Specificity)
	}
	if got := sels[0].String(); got != `input[type="checkbox"]:checked:-ember-native` {
		t.Errorf("printed %q", got)
	}
}

// A range, a progress and a meter get their parts' styles, each
// inheriting from the part it is in; a range of appearance none is at
// least as tall as its track, as tall as its thumb unless sized.
func TestTheControlsPartsAreStyledAsChromes(t *testing.T) {
	prog, meter := el(dom.Progress, "", ""), el(dom.Meter, "", "")
	plain, short, tall := input("range", "none", ""), input("range", "none four", ""), input("range", "none thumb", "")
	own := input("range", "thumb", "")
	Compute(page(prog, meter, plain, short, tall, own), []*Sheet{sheet(t, `.none { appearance: none; }
progress { color: #123; }
.four::-webkit-slider-runnable-track { height: 4px; margin: 2px 0; }
.thumb::-webkit-slider-thumb { appearance: none; height: 20px; border: 2px solid; }
meter::-webkit-meter-suboptimum-value { background-color: #f00; }`)})
	p := styleOf(prog).Parts
	if p == nil || p.Bar.Background != color.Hex("#808080") || p.Value.Background != color.Hex("#008000") ||
		p.Value.Color != color.Hex("#123") {
		t.Fatalf("a progress's parts: %+v", p)
	}
	m := styleOf(meter).Parts
	if m == nil || m.Bar == nil || len(m.Bar.BackgroundImages) != 1 || m.Regions[1].Background != color.Hex("#f00") {
		t.Fatalf("a meter's parts: %+v", m)
	}
	for name, c := range map[string]struct {
		n *dom.Node
		h style.Length
	}{"plain": {plain, style.Px(16)}, "short": {short, style.Px(8)}, "tall": {tall, style.Px(20)}} {
		if got := styleOf(c.n).MinSize.Height; got != c.h {
			t.Errorf("%s: min-height %v, want %v", name, got, c.h)
		}
	}
	if st := styleOf(own); st.Size.Height.Kind != style.Auto || st.Parts.Thumb.Appearance.Native() {
		t.Errorf("a range of its own look: height %v, thumb %v", st.Size.Height, st.Parts.Thumb.Appearance)
	}
}

// A part's :hover and :active hold while the pointer is over it and
// while a press on it lasts, as the element's part states say; the
// parts they hold are in them too.
func TestAControlsPartsTakeTheirHoverAndActive(t *testing.T) {
	sels, err := ParseSelectors("input::-webkit-slider-thumb:hover:active")
	if err != nil {
		t.Fatal(err)
	}
	if got := sels[0].String(); got != "input::-webkit-slider-thumb:hover:active" {
		t.Errorf("printed %q", got)
	}
	for _, bad := range []string{"input::-webkit-slider-thumb:vertical", "progress::-webkit-progress-value:checked"} {
		if _, e := ParseSelectors(bad); e == nil {
			t.Errorf("%s read", bad)
		}
	}
	rest, over, pressed := input("range", "none", ""), input("range", "none", ""), input("range", "none", "")
	over.SetState(dom.ContainerHover|dom.TrackHover, true)
	pressed.SetState(dom.PartHover|dom.PartActive, true)
	bar := el(dom.Progress, "", "")
	bar.SetState(dom.PartHover, true)
	Compute(page(rest, over, pressed, bar), []*Sheet{sheet(t, `.none { appearance: none; }
::-webkit-slider-container { background-color: #100; }
::-webkit-slider-container:hover { background-color: #200; }
::-webkit-slider-runnable-track:hover { background-color: #300; }
::-webkit-slider-runnable-track:active { background-color: #400; }
::-webkit-slider-thumb { appearance: none; background-color: #500; }
::-webkit-slider-thumb:hover { background-color: #600; }
::-webkit-slider-thumb:hover:active { background-color: #700; }
progress::-webkit-progress-value:hover { background-color: #800; }`)})
	for name, c := range map[string]struct {
		n                       *dom.Node
		container, track, thumb color.Color
	}{
		"at rest": {rest, color.Hex("#100"), color.Color{}, color.Hex("#500")},
		"over":    {over, color.Hex("#200"), color.Hex("#300"), color.Hex("#500")},
		"a press": {pressed, color.Hex("#200"), color.Hex("#400"), color.Hex("#700")},
	} {
		p := styleOf(c.n).Parts
		var track color.Color
		if p.Track != nil {
			track = p.Track.Background
		}
		if p.Container.Background != c.container || track != c.track || p.Thumb.Background != c.thumb {
			t.Errorf("%s: container %v, track %v, thumb %v", name, p.Container.Background, track, p.Thumb.Background)
		}
	}
	if v := styleOf(bar).Parts.Value; v.Background != color.Hex("#800") {
		t.Errorf("a hovered progress value: %v", v.Background)
	}
}

// A range of appearance none is as thick as its slider container round
// its track, down it, or across it where its lines are vertical.
func TestARangesContainerAndVerticalLinesSizeIt(t *testing.T) {
	padded, upright := input("range", "none pad", ""), input("range", "none up", "")
	own := input("range", "up", "")
	sv := input("range", "", "appearance: slider-vertical")
	Compute(page(padded, upright, own, sv), []*Sheet{sheet(t, `.none { appearance: none; }
.pad::-webkit-slider-container { padding: 3px; border: 1px solid; }
.pad::-webkit-slider-runnable-track { height: 8px; }
.up { writing-mode: vertical-lr; }
.up::-webkit-slider-thumb { appearance: none; width: 12px; height: 6px; }`)})
	if got := styleOf(padded).MinSize.Height; got != style.Px(16) {
		t.Errorf("a padded container: min-height %v, want 16px", got)
	}
	if st := styleOf(upright); st.MinSize.Width != style.Px(12) || st.Size.Height.Kind != style.Auto ||
		st.MinSize.Height.Kind != style.Auto {
		t.Errorf("vertical lines: min-width %v, height %v, min-height %v", st.MinSize.Width, st.Size.Height,
			st.MinSize.Height)
	}
	if st := styleOf(own); st.Size.Width.Kind != style.Auto || st.Size.Height.Kind != style.Auto {
		t.Errorf("its own look in vertical lines, its sizes natural ones layout gives: %v × %v", st.Size.Width,
			st.Size.Height)
	}
	if a := styleOf(sv).Appearance; a != style.AppearanceSliderVertical {
		t.Errorf("appearance: slider-vertical computes as %v", a)
	}
}
