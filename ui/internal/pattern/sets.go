package pattern

import (
	"slices"
	"strings"
	"unicode"
)

// set is a class as the v flag has it: characters, and strings of other
// than one character. Under the i flag every set is closed under case
// folding, its in true for all of a character's case variants or none,
// so that its operations and its complement work on folded characters
// as JavaScript's do.
type set struct {
	in   func(rune) bool
	strs []string
	fold bool
}

// leaf is the set of the characters in reads, closed under case folding
// when fold.
func leaf(in func(rune) bool, fold bool) *set {
	if !fold {
		return &set{in: in}
	}
	return &set{in: func(r rune) bool {
		for f := r; ; {
			if in(f) {
				return true
			}
			if f = unicode.SimpleFold(f); f == r {
				return false
			}
		}
	}, fold: true}
}

func single(c rune, fold bool) *set { return leaf(func(r rune) bool { return r == c }, fold) }

func (s *set) union(o *set) *set {
	a, b := s.in, o.in
	strs := slices.Clone(s.strs)
	for _, t := range o.strs {
		if !s.holds(t) {
			strs = append(strs, t)
		}
	}
	return &set{in: func(r rune) bool { return a(r) || b(r) }, strs: strs, fold: s.fold}
}

func (s *set) intersect(o *set) *set {
	a, b := s.in, o.in
	var strs []string
	for _, t := range s.strs {
		if o.holds(t) {
			strs = append(strs, t)
		}
	}
	return &set{in: func(r rune) bool { return a(r) && b(r) }, strs: strs, fold: s.fold}
}

func (s *set) subtract(o *set) *set {
	a, b := s.in, o.in
	var strs []string
	for _, t := range s.strs {
		if !o.holds(t) {
			strs = append(strs, t)
		}
	}
	return &set{in: func(r rune) bool { return a(r) && !b(r) }, strs: strs, fold: s.fold}
}

func (s *set) complement() *set {
	a := s.in
	return &set{in: func(r rune) bool { return !a(r) }, fold: s.fold}
}

// holds reports whether s has the string t, folded under the i flag.
func (s *set) holds(t string) bool {
	return slices.ContainsFunc(s.strs, func(u string) bool { return sameRunes([]rune(u), []rune(t), s.fold) })
}

// node is the set as a node to match: its strings longest first, then a
// character, then the empty string, as JavaScript tries them.
func (s *set) node() node {
	c := &class{in: s.in}
	for _, t := range s.strs {
		if t == "" {
			c.empty = true
		} else {
			c.strs = append(c.strs, []rune(t))
		}
	}
	slices.SortStableFunc(c.strs, func(a, b []rune) int { return len(b) - len(a) })
	c.fold = s.fold
	return c
}

func sameRunes(a, b []rune, fold bool) bool {
	return slices.EqualFunc(a, b, func(x, y rune) bool { return same(x, y, fold) })
}

func same(a, b rune, fold bool) bool { return a == b || fold && canon(a) == canon(b) }

// canon is the least of r's case variants: two characters are the same
// under the i flag when theirs are.
func canon(r rune) rune {
	least := r
	for f := unicode.SimpleFold(r); f != r; f = unicode.SimpleFold(f) {
		least = min(least, f)
	}
	return least
}

func isDigit(r rune) bool { return r >= '0' && r <= '9' }

func isWord(r rune) bool {
	return r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || isDigit(r) || r == '_'
}

func isSpace(r rune) bool {
	switch r {
	case '\t', '\n', '\v', '\f', '\r', ' ', 0xa0, 0x1680, 0x2028, 0x2029, 0x202f, 0x205f, 0x3000, 0xfeff:
		return true
	}
	return r >= 0x2000 && r <= 0x200a
}

func isLineEnd(r rune) bool { return r == '\n' || r == '\r' || r == 0x2028 || r == 0x2029 }

// property is what \p{body} names: a general category, a script, or a
// binary property, by the names JavaScript takes.
func property(body string) (func(rune) bool, error) {
	name, value, pair := strings.Cut(body, "=")
	if pair {
		switch name {
		case "General_Category", "gc":
			if in, ok := category(value); ok {
				return in, nil
			}
		case "Script", "sc":
			if t, ok := unicode.Scripts[value]; ok {
				return func(r rune) bool { return unicode.Is(t, r) }, nil
			}
			return nil, ErrUnsupported
		case "Script_Extensions", "scx":
			return nil, ErrUnsupported
		}
		return nil, errorf("unknown property %q", body)
	}
	if in, ok := category(body); ok {
		return in, nil
	}
	if long, ok := binaryAliases[body]; ok {
		body = long
	}
	if in, ok := derived[body]; ok {
		return in, nil
	}
	if t, ok := unicode.Properties[body]; ok {
		return func(r rune) bool { return unicode.Is(t, r) }, nil
	}
	if slices.Contains(unread, body) {
		return nil, ErrUnsupported
	}
	return nil, errorf("unknown property %q", body)
}

func category(v string) (func(rune) bool, bool) {
	if long, ok := unicode.CategoryAliases[v]; ok {
		v = long
	}
	t, ok := unicode.Categories[v]
	if !ok {
		return nil, false
	}
	return func(r rune) bool { return unicode.Is(t, r) }, true
}

func in(tables ...*unicode.RangeTable) func(rune) bool {
	return func(r rune) bool { return unicode.IsOneOf(tables, r) }
}

// derived are the binary properties Go's unicode tables have the parts
// of, but not the whole.
//
//nolint:gochecknoglobals // lookup table
var derived = map[string]func(rune) bool{
	"Any":             func(rune) bool { return true },
	"ASCII":           func(r rune) bool { return r < 0x80 },
	"Assigned":        func(r rune) bool { return !unicode.Is(unicode.Cn, r) },
	"Alphabetic":      in(unicode.L, unicode.Nl, unicode.Other_Alphabetic),
	"Lowercase":       in(unicode.Ll, unicode.Other_Lowercase),
	"Uppercase":       in(unicode.Lu, unicode.Other_Uppercase),
	"Math":            in(unicode.Sm, unicode.Other_Math),
	"Grapheme_Extend": in(unicode.Me, unicode.Mn, unicode.Other_Grapheme_Extend),
	"ID_Start": func(r rune) bool {
		return unicode.IsOneOf([]*unicode.RangeTable{unicode.L, unicode.Nl, unicode.Other_ID_Start}, r) &&
			!unicode.In(r, unicode.Pattern_Syntax, unicode.Pattern_White_Space)
	},
	"ID_Continue": func(r rune) bool {
		return unicode.IsOneOf([]*unicode.RangeTable{unicode.L, unicode.Nl, unicode.Other_ID_Start, unicode.Mn,
			unicode.Mc, unicode.Nd, unicode.Pc, unicode.Other_ID_Continue}, r) &&
			!unicode.In(r, unicode.Pattern_Syntax, unicode.Pattern_White_Space)
	},
}

// binaryAliases are the short names of the binary properties.
//
//nolint:gochecknoglobals // lookup table
var binaryAliases = map[string]string{
	"AHex": "ASCII_Hex_Digit", "Alpha": "Alphabetic", "Bidi_C": "Bidi_Control", "Bidi_M": "Bidi_Mirrored",
	"CI": "Case_Ignorable", "CWCF": "Changes_When_Casefolded", "CWCM": "Changes_When_Casemapped",
	"CWKCF": "Changes_When_NFKC_Casefolded", "CWL": "Changes_When_Lowercased", "CWT": "Changes_When_Titlecased",
	"CWU": "Changes_When_Uppercased", "DI": "Default_Ignorable_Code_Point", "Dep": "Deprecated", "Dia": "Diacritic",
	"EBase": "Emoji_Modifier_Base", "EComp": "Emoji_Component", "EMod": "Emoji_Modifier",
	"EPres": "Emoji_Presentation", "Ext": "Extender", "ExtPict": "Extended_Pictographic", "Gr_Base": "Grapheme_Base",
	"Gr_Ext": "Grapheme_Extend", "Hex": "Hex_Digit", "IDC": "ID_Continue", "IDS": "ID_Start",
	"IDSB": "IDS_Binary_Operator", "IDST": "IDS_Trinary_Operator", "Ideo": "Ideographic", "Join_C": "Join_Control",
	"LOE": "Logical_Order_Exception", "Lower": "Lowercase", "NChar": "Noncharacter_Code_Point",
	"Pat_Syn": "Pattern_Syntax", "Pat_WS": "Pattern_White_Space", "QMark": "Quotation_Mark",
	"RI": "Regional_Indicator", "SD": "Soft_Dotted", "Term": "Terminal_Punctuation", "UIdeo": "Unified_Ideograph",
	"Upper": "Uppercase", "VS": "Variation_Selector", "XIDC": "XID_Continue", "XIDS": "XID_Start",
	"space": "White_Space",
}

// unread are the properties JavaScript knows that Go's tables do not
// give: the emoji ones, the case mappings', and the properties of
// strings.
//
//nolint:gochecknoglobals // lookup table
var unread = []string{
	"Bidi_Mirrored", "Case_Ignorable", "Cased", "Changes_When_Casefolded", "Changes_When_Casemapped",
	"Changes_When_Lowercased", "Changes_When_NFKC_Casefolded", "Changes_When_Titlecased", "Changes_When_Uppercased",
	"Default_Ignorable_Code_Point", "Emoji", "Emoji_Component", "Emoji_Modifier", "Emoji_Modifier_Base",
	"Emoji_Presentation", "Extended_Pictographic", "Grapheme_Base", "XID_Continue", "XID_Start", "Basic_Emoji",
	"Emoji_Keycap_Sequence", "RGI_Emoji_Modifier_Sequence", "RGI_Emoji_Flag_Sequence", "RGI_Emoji_Tag_Sequence",
	"RGI_Emoji_ZWJ_Sequence", "RGI_Emoji",
}
