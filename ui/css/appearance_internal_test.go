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
	if st := styleOf(own); st.Size.Height != style.Px(16) || st.Parts.Thumb.Appearance.Native() {
		t.Errorf("a range of its own look: height %v, thumb %v", st.Size.Height, st.Parts.Thumb.Appearance)
	}
}
