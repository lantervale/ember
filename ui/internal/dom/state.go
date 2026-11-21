package dom

// State is the set of an element's dynamic pseudo-classes.
type State uint16

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
)

func (s State) Has(f State) bool { return s&f == f }
