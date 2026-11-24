package css

import (
	"strconv"
	"strings"

	"github.com/lantervale/ember/ui/internal/dom"
)

// Declared is one property as a rule or a style attribute declares it,
// its value written as Chrome writes a declaration block's text: keywords
// in lower case, a length of 0 as 0px, a hex or functional colour as
// rgb(), commas and spaces as Chrome puts them. A shorthand of sides
// stands as its longhands, so that a block folds them back as Chrome
// does (decision 0224).
type Declared struct {
	Name, Value string
	Important   bool
	// Percent is set on a value that is one percentage, which a copy
	// writes as the element's computed value.
	Percent bool
}

// Declared is what the rules that match the element n declare, ember's
// own and the Document's, and what its style attribute declares: each
// property once, where its first declaration puts it, with the value of
// the last, the rules taken from the weakest. It is what Chrome's copy
// merges into an element's style attribute, where ember's own sheet
// stands for the page's: a paste knows nothing of it. Both are empty
// for a text node and for an element not yet styled.
func (t *Transitions) Declared(n *dom.Node) ([]Declared, []Declared) {
	if !t.ready(n) {
		return nil, nil
	}
	var rules, inline []Declared
	for _, f := range t.inspect(n).found(n) {
		switch {
		case f.hint:
		case f.inline:
			inline = declare(inline, f.decl)
		default:
			rules = declare(rules, f.decl)
		}
	}
	return rules, inline
}

// declare sets the longhands of d in list, in place where list has one,
// else after it.
func declare(list []Declared, d *Declaration) []Declared {
	if strings.HasPrefix(d.Name, "--") {
		return list
	}
	for _, p := range declaredSides(d) {
		p.Important = d.Important
		i := 0
		for i < len(list) && list[i].Name != p.Name {
			i++
		}
		if i == len(list) {
			list = append(list, p)
		} else {
			list[i] = p
		}
	}
	return list
}

// sideNames are the longhands of the shorthands Chrome folds back from
// one to four values, top, right, bottom, left, or start and end, each
// logical one named as written.
//
//nolint:gochecknoglobals // a lookup table
var sideNames = func() map[string][]string {
	out := map[string][]string{}
	for _, name := range []string{shortMargin, shortPadding, kwInset} {
		out[name] = shorthands[name]
		for _, axis := range []string{"-block", "-inline"} {
			out[name+axis] = []string{name + axis + "-start", name + axis + "-end"}
		}
	}
	for _, name := range []string{shortBorderWidth, shortBorderStyle, propBorderColor} {
		out[name] = shorthands[name]
	}
	return out
}()

// SideShorthands are the shorthands a Declared list holds as longhands,
// each with its longhands in order.
func SideShorthands() map[string][]string { return sideNames }

// declaredSides is d as Declared longhands: a shorthand of sides cut into its
// sides, anything else whole.
func declaredSides(d *Declaration) []Declared {
	name := d.Name
	if d.logical != "" {
		name = d.logical
	}
	values := topLevel(d.Value)
	long, ok := sideNames[name]
	if !ok || hasVar(d.Value) || len(values) == 0 || len(values) > len(long) {
		return []Declared{declared(name, d.Value)}
	}
	pick := func(i int) []Component {
		switch {
		case i < len(values):
			return values[i]
		case len(long) == 2 || i == 2:
			return values[0]
		case i == 3 && len(values) > 1:
			return values[1]
		}
		return values[0]
	}
	out := make([]Declared, len(long))
	for i, name := range long {
		out[i] = declared(name, pick(i))
	}
	return out
}

// topLevel splits a value at its top-level white space.
func topLevel(v []Component) [][]Component {
	var out [][]Component
	var cur []Component
	for _, c := range v {
		if c.Kind == Whitespace {
			if cur != nil {
				out = append(out, cur)
				cur = nil
			}
			continue
		}
		cur = append(cur, c)
	}
	if cur != nil {
		out = append(out, cur)
	}
	return out
}

func declared(name string, v []Component) Declared {
	v = trimSpace(v)
	return Declared{Name: name, Value: specified(name, v), Percent: len(v) == 1 && v[0].Kind == Percentage}
}

func trimSpace(v []Component) []Component {
	for len(v) > 0 && v[0].Kind == Whitespace {
		v = v[1:]
	}
	for len(v) > 0 && v[len(v)-1].Kind == Whitespace {
		v = v[:len(v)-1]
	}
	return v
}

// specified writes v, the value of the property name, as Chrome
// serialises a specified value.
func specified(name string, v []Component) string {
	if name == propFontFamily {
		return familyList(v)
	}
	var b strings.Builder
	space := false
	for _, c := range v {
		if c.Kind == Whitespace {
			space = true
			continue
		}
		if c.Kind == Comma {
			b.WriteString(", ")
			space = false
			continue
		}
		if space && b.Len() > 0 && !strings.HasSuffix(b.String(), " ") {
			b.WriteByte(' ')
		}
		space = false
		b.WriteString(specifiedComponent(name, c))
	}
	return b.String()
}

func specifiedComponent(name string, c Component) string {
	switch c.Kind {
	case Ident:
		if keepsCase(name) {
			return c.Text
		}
		return strings.ToLower(c.Text)
	case Hash:
		if col, err := parseHex(c.Text); err == nil {
			return col.String()
		}
	case Number:
		if c.Num == 0 && takesLength(name) {
			return "0px"
		}
		return specifiedNumber(c.Num)
	case Percentage:
		return specifiedNumber(c.Num) + "%"
	case Dimension:
		return specifiedNumber(c.Num) + strings.ToLower(c.Unit)
	case String:
		return c.String()
	case Function:
		fn := strings.ToLower(c.Text)
		if col, err := parseColor(c); err == nil && legacyColorFunction(fn) {
			return col.String()
		}
		return fn + "(" + specified(name, c.Args) + ")"
	case EOF, AtKeyword, BadString, URL, BadURL, Delim, Whitespace, CDO, CDC, Colon, Semicolon, Comma, LBracket,
		RBracket, LParen, RParen, LBrace, RBrace:
		// as written
	}
	return c.String()
}

// specifiedNumber writes v as Chrome does, at most six significant digits.
func specifiedNumber(v float64) string {
	return strconv.FormatFloat(v, 'g', 6, 64)
}

// keepsCase reports a property whose identifiers are names, written as
// given.
func keepsCase(name string) bool {
	if name == propFontFamily || name == shortFont || strings.HasSuffix(name, "-name") ||
		strings.HasSuffix(name, "-anchor") {
		return true
	}
	for _, p := range []string{"animation", "counter-", "container", "grid-"} {
		if strings.HasPrefix(name, p) {
			return true
		}
	}
	return false
}

// takesLength reports a property a bare 0 is a length of.
func takesLength(name string) bool {
	if strings.HasSuffix(name, "-style") || strings.HasSuffix(name, "-color") || strings.HasPrefix(name, "line-") {
		return false
	}
	for _, end := range []string{propWidth, propHeight, "-radius", "-offset", "spacing", "gap", "-thickness",
		"-indent", "-basis"} {
		if strings.HasSuffix(name, end) {
			return true
		}
	}
	for _, p := range []string{shortMargin, shortPadding, kwInset, kwTop, kwRight, kwBottom, kwLeft, propFontSize,
		"vertical-"} {
		if strings.HasPrefix(name, p) {
			return true
		}
	}
	return false
}

// familyList writes a font-family value as Chrome serialises it: its
// names one by one, each quoted only where it is not one identifier.
func familyList(v []Component) string {
	var names []string
	var cur []string
	flush := func() {
		if cur != nil {
			names = append(names, strings.Join(cur, " "))
			cur = nil
		}
	}
	for _, c := range v {
		switch c.Kind {
		case Comma:
			flush()
		case Ident:
			cur = append(cur, c.Text)
		case String:
			cur = append(cur, c.Text)
		case EOF, AtKeyword, Hash, BadString, URL, BadURL, Delim, Number, Percentage, Dimension, Whitespace, CDO, CDC,
			Colon, Semicolon, LBracket, RBracket, LParen, RParen, LBrace, RBrace, Function:
			// between the names
		}
	}
	flush()
	return FontFamily(names)
}

// FontFamily writes a list of font families as Chrome serialises one:
// the generic families and a name that is one identifier as they are,
// any other name quoted.
func FontFamily(names []string) string {
	out := make([]string, len(names))
	for i, n := range names {
		out[i] = n
		if !isIdentifier(n) {
			out[i] = Token{Kind: String, Text: n}.String()
		}
	}
	return strings.Join(out, ", ")
}

// isIdentifier reports s one CSS identifier.
func isIdentifier(s string) bool {
	toks := Tokenize(s)
	return len(toks) == 2 && toks[0].Kind == Ident && toks[0].Text == s
}

// legacyColorFunction reports a colour function Chrome writes back as
// rgb().
func legacyColorFunction(fn string) bool {
	return strings.HasPrefix(fn, "rgb") || strings.HasPrefix(fn, "hsl") || strings.HasPrefix(fn, "hw")
}
