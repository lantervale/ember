package css

// What the chrome suite (package css_test) asks Chrome about.

// WebkitTable is -webkit-'s property tables: the plain property each
// name is read as, and why each of Chrome's own stays as written.
func WebkitTable() (plain, own map[string]string) {
	plain = map[string]string{}
	for _, name := range webkit.same {
		plain[name] = name
	}
	for name, as := range webkit.renamed {
		plain[name] = as
	}
	return plain, webkit.own
}

// WebkitValues are the -webkit- values read as a plain one.
func WebkitValues() map[string]string { return webkit.values }

// VendorPrefixes are the prefixes with a table.
func VendorPrefixes() []string {
	var out []string
	for _, v := range vendors {
		out = append(out, v.prefix)
	}
	return out
}

// MediaCase is a media query and Chrome's answer to it in ChromeViewport.
type MediaCase struct {
	Query string
	Want  bool
}

// MediaCases are the queries of TestMediaQueriesAreAnsweredAsChromeAnswers.
func MediaCases() []MediaCase {
	var out []MediaCase
	for _, c := range mediaCases {
		out = append(out, MediaCase{c.query, c.want})
	}
	return out
}

// ChromeViewport is the viewport MediaCases were answered in.
func ChromeViewport() Media { return chromeViewport }

// Supported answers an @supports condition as a sheet read as a
// browser's does.
func Supported(cond string) bool { return supported(componentsOf(cond)) }

// PseudoElement is one pseudo-element of the table, as a selector with
// an argument where it takes one, and whether ui refuses it.
type PseudoElement struct {
	Selector string
	Refused  bool
}

// PseudoElements are the pseudo-elements of the table.
func PseudoElements() []PseudoElement {
	var out []PseudoElement
	for name, pe := range pseudoElements {
		sel := "::" + name
		if pe.form == formFunction {
			arg := "x"
			switch name {
			case "slotted":
				arg = "*"
			case "scroll-button":
				arg = "left"
			case "picker":
				arg = "select"
			}
			sel += "(" + arg + ")"
		}
		out = append(out, PseudoElement{sel, pe.status == pseudoRefused})
	}
	return out
}
