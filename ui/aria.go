package ui

// The roles and attribute values the elements read and write, named once.
const (
	roleMenu             = "menu"
	roleMenubar          = "menubar"
	roleMenuItemCheckbox = "menuitemcheckbox"
	roleMenuItemRadio    = "menuitemradio"
	roleTablist          = "tablist"
	roleOption           = "option"
	roleProgressbar      = "progressbar"

	// attrTrue is "true", as aria-selected, aria-checked and draggable hold it.
	attrTrue = "true"

	// typeReset is the type of a button that resets its form.
	typeReset = "reset"
)

// A tooltip's or a suggestion list's side of its anchor.
const (
	sideAbove = "above"
	sideLeft  = "left"
	sideRight = "right"
)
