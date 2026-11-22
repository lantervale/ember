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

// The types of input ui's code tells apart.
const (
	typeText     = "text"
	typeSearch   = "search"
	typePassword = "password"
	typeEmail    = "email"
	typeURL      = "url"
	typeNumber   = "number"
	typeCheckbox = "checkbox"
	typeRadio    = "radio"
	typeHidden   = "hidden"
	typeButton   = "button"
	typeSubmit   = "submit"
	typeImage    = "image"
)

// A tooltip's or a suggestion list's side of its anchor.
const (
	sideAbove = "above"
	sideLeft  = "left"
	sideRight = "right"
)
