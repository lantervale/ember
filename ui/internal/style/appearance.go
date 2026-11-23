package style

// Appearance is appearance: none draws a control from its CSS alone, as
// Chrome draws it; any other value keeps ui's own look of the control,
// the compatibility values (checkbox, menulist…) standing for auto as in
// Chrome.
type Appearance uint8

const (
	AppearanceNone Appearance = iota
	AppearanceAuto
	AppearanceTextfield
	AppearanceMenulistButton
	AppearanceCheckbox
	AppearanceRadio
	AppearanceButton
	AppearanceListbox
	AppearanceMenulist
	AppearanceMeter
	AppearanceProgressBar
	AppearanceSliderVertical
	AppearanceSearchfield
	AppearanceTextarea
	AppearanceBaseSelect
)

// AppearanceNames are appearance's keywords, by value.
//
//nolint:gochecknoglobals // the keywords' table
var AppearanceNames = [...]string{
	AppearanceNone: "none", AppearanceAuto: "auto", AppearanceTextfield: "textfield",
	AppearanceMenulistButton: "menulist-button", AppearanceCheckbox: "checkbox", AppearanceRadio: "radio",
	AppearanceButton: "button", AppearanceListbox: "listbox", AppearanceMenulist: "menulist",
	AppearanceMeter: "meter", AppearanceProgressBar: "progress-bar", AppearanceSliderVertical: "slider-vertical",
	AppearanceSearchfield: "searchfield", AppearanceTextarea: "textarea", AppearanceBaseSelect: "base-select",
}

func (a Appearance) String() string { return AppearanceNames[a] }

// Native reports whether a keeps the control's own look.
func (a Appearance) Native() bool { return a != AppearanceNone }

// ControlParts are a range's, a progress's or a meter's parts its sheets
// style, as Chrome lays them out where the control's appearance is none
// (decision 0160); a part no rule styles is nil.
type ControlParts struct {
	// Inner is a progress's or a meter's inner element, filling its
	// content box, and Bar fills the inner's.
	Inner, Bar *Computed
	// Value is a progress's value, the bar's content box cut to the
	// value; Regions are a meter's, by the region its value is in:
	// optimum, suboptimum and even less good.
	Value   *Computed
	Regions [3]*Computed
	// Track and Thumb are a range's.
	Track, Thumb *Computed
}

// Equal reports whether p and o style the same parts the same way.
func (p *ControlParts) Equal(o *ControlParts) bool {
	if p == o {
		return true
	}
	if p == nil || o == nil {
		return false
	}
	return p.Inner.Equal(o.Inner) && p.Bar.Equal(o.Bar) && p.Value.Equal(o.Value) &&
		p.Regions[0].Equal(o.Regions[0]) && p.Regions[1].Equal(o.Regions[1]) && p.Regions[2].Equal(o.Regions[2]) &&
		p.Track.Equal(o.Track) && p.Thumb.Equal(o.Thumb)
}
