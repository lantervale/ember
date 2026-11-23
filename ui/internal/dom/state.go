package dom

// State is the set of an element's dynamic pseudo-classes.
type State uint32

const (
	Hover State = 1 << iota
	Focus
	FocusVisible
	Active
	Disabled
	Checked
	Open
	// Modal is a dialog shown modal (HTML's :modal).
	Modal
	// FocusWithin is an element holding the focus or above one that does,
	// Target the Document's target (the element a fragment names).
	FocusWithin
	Target
	// Empty is an element with no element and no text in it (:empty),
	// which the tree keeps itself.
	Empty
	// Indeterminate is a checkbox the program set so, or a radio of a
	// group none of which is checked; PlaceholderShown a field showing
	// its placeholder; Default a checkbox or radio checked by its
	// attribute, an option selected by its own, or its form's first
	// submit button.
	Indeterminate
	PlaceholderShown
	Default
	// Valid and Invalid are an element constraint validation reads
	// (:valid, :invalid), UserValid and UserInvalid the same once the
	// player has changed it; InRange and OutOfRange a number's value
	// against its min and max. ui keeps them, as Checked.
	Valid
	Invalid
	UserValid
	UserInvalid
	InRange
	OutOfRange
	// PopoverOpen is an element with the popover attribute shown
	// (:popover-open).
	PopoverOpen
)

func (s State) Has(f State) bool { return s&f == f }
