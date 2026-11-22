package css

import (
	"fmt"
	"math"
	"strings"

	"github.com/lantervale/ember/ui/internal/style"
)

// systemColors are CSS's system colours and the deprecated ones that
// stand for them, in the light and the dark scheme, as Chrome resolves
// them (TestChromeResolvesSystemColoursInEachScheme), Highlight,
// SelectedItem and their texts as on macOS, where each system has its
// own: 0xRRGGBBAA.
//
//nolint:gochecknoglobals,goconst // the system colours' table
var systemColors = map[string][2]uint32{
	"canvas":              {0xffffffff, 0x121212ff},
	"canvastext":          {0x000000ff, 0xffffffff},
	"field":               {0xffffffff, 0x3b3b3bff},
	"fieldtext":           {0x000000ff, 0xffffffff},
	"buttonface":          {0xefefefff, 0x6b6b6bff},
	"buttontext":          {0x000000ff, 0xffffffff},
	"buttonborder":        {0x000000ff, 0xffffffff},
	"linktext":            {0x0000eeff, 0x9e9effff},
	"visitedtext":         {0x551a8bff, 0xd0adf0ff},
	"activetext":          {0xff0000ff, 0xff0000ff},
	"graytext":            {0x808080ff, 0x808080ff},
	"highlight":           {0x80bcfe99, 0xb3d7ffcc},
	"highlighttext":       {0x000000ff, 0x000000ff},
	"mark":                {0xffff00ff, 0xffff00ff},
	"marktext":            {0x000000ff, 0x000000ff},
	"selecteditem":        {0xb3d7ffff, 0x99c8ffff},
	"selecteditemtext":    {0x000000ff, 0x3b3b3bff},
	"accentcolor":         {0x0075ffff, 0x0075ffff},
	"accentcolortext":     {0xffffffff, 0xffffffff},
	"activeborder":        {0x000000ff, 0xffffffff},
	"activecaption":       {0xffffffff, 0x121212ff},
	"appworkspace":        {0xffffffff, 0x121212ff},
	"background":          {0xffffffff, 0x121212ff},
	"buttonhighlight":     {0xefefefff, 0x6b6b6bff},
	"buttonshadow":        {0xefefefff, 0x6b6b6bff},
	"captiontext":         {0x000000ff, 0xffffffff},
	"inactiveborder":      {0x000000ff, 0xffffffff},
	"inactivecaption":     {0xffffffff, 0x121212ff},
	"inactivecaptiontext": {0x808080ff, 0x808080ff},
	"infobackground":      {0xffffffff, 0x121212ff},
	"infotext":            {0x000000ff, 0xffffffff},
	"menu":                {0xffffffff, 0x121212ff},
	"menutext":            {0x000000ff, 0xffffffff},
	"scrollbar":           {0xffffffff, 0x121212ff},
	"threeddarkshadow":    {0x000000ff, 0xffffffff},
	"threedface":          {0xefefefff, 0x6b6b6bff},
	"threedhighlight":     {0x000000ff, 0xffffffff},
	"threedlightshadow":   {0x000000ff, 0xffffffff},
	"threedshadow":        {0x000000ff, 0xffffffff},
	"window":              {0xffffffff, 0x121212ff},
	"windowframe":         {0x000000ff, 0xffffffff},
	"windowtext":          {0x000000ff, 0xffffffff},
}

// systemColor is the system colour c names in the dark scheme or the
// light one, false when c names none.
func systemColor(c Component, dark bool) (Color, bool) {
	if c.Kind != Ident {
		return Color{}, false
	}
	v, ok := systemColors[strings.ToLower(c.Text)]
	if !ok {
		return Color{}, false
	}
	rgba := v[0]
	if dark {
		rgba = v[1]
	}
	ch := func(shift uint) float64 { return float64(rgba>>shift&0xff) / 255 }
	return Color{ch(24), ch(16), ch(8), ch(0)}, true
}

// lightDark splits light-dark()'s arguments into its two colours.
func lightDark(args []Component) (Component, Component, bool) {
	var parts [3]Component
	n := 0
	for _, a := range args {
		if a.Kind == Whitespace {
			continue
		}
		if n == len(parts) {
			return Component{}, Component{}, false
		}
		parts[n] = a
		n++
	}
	if n != 3 || parts[1].Kind != Comma || parts[0].Kind == Comma || parts[2].Kind == Comma {
		return Component{}, Component{}, false
	}
	return parts[0], parts[2], true
}

func parseLightDark(args []Component) (Color, error) {
	light, dark, ok := lightDark(args)
	if !ok {
		return Color{}, fmt.Errorf("light-dark() takes two colours, separated by a comma")
	}
	for _, c := range []Component{light, dark} {
		if isWord(c, kwCurrentColor) {
			continue
		}
		if _, err := parseColor(c); err != nil {
			return Color{}, fmt.Errorf("light-dark(): %w", err)
		}
	}
	if isWord(light, kwCurrentColor) {
		return Color{}, nil
	}
	return parseColor(light)
}

// inScheme is v, a value of a property that takes colours, with its
// system colours and light-dark() resolved in the dark scheme or the
// light one, as hex colours; v itself when it holds neither. Elsewhere a
// system colour's name is a word like any other (animation-name: mark).
func inScheme(v []Component, dark bool) []Component {
	var out []Component
	for i, c := range v {
		r, ok := resolved(c, dark)
		if !ok {
			continue
		}
		if out == nil {
			out = make([]Component, len(v))
			copy(out, v)
		}
		out[i] = r
	}
	if out == nil {
		return v
	}
	return out
}

// resolved is c resolved in a scheme, false when the scheme changes
// nothing of it.
func resolved(c Component, dark bool) (Component, bool) {
	switch {
	case c.Kind == Function && strings.EqualFold(c.Text, "light-dark"):
		light, d, ok := lightDark(c.Args)
		if !ok {
			return c, false
		}
		pick := light
		if dark {
			pick = d
		}
		if r, deeper := resolved(pick, dark); deeper {
			return r, true
		}
		return pick, true
	case c.Kind == Ident:
		col, ok := systemColor(c, dark)
		if !ok {
			return c, false
		}
		ch := func(v float64) int { return int(math.Round(v * 255)) }
		text := fmt.Sprintf("%02x%02x%02x", ch(col.R), ch(col.G), ch(col.B))
		if col.A < 1 {
			text += fmt.Sprintf("%02x", ch(col.A))
		}
		return Component{Token: Token{Kind: Hash, Pos: c.Pos, Text: text}}, true
	case len(c.Args) > 0:
		args := inScheme(c.Args, dark)
		if len(args) > 0 && &args[0] == &c.Args[0] {
			return c, false
		}
		c.Args = args
		return c, true
	}
	return c, false
}

// colorScheme reads color-scheme: normal, or the schemes named, words
// of the page's own among them, with only before or after.
func colorScheme(v []Component) style.ColorScheme {
	var s style.ColorScheme
	for _, c := range v {
		switch strings.ToLower(c.Text) {
		case "light":
			s |= style.SchemeLight
		case "dark":
			s |= style.SchemeDark
		case "only":
			s |= style.SchemeOnly
		}
	}
	return s
}

// checkColorScheme checks color-scheme: normal, or one name or more
// (light, dark or a word of the page's own) and only at most once,
// first or last.
func checkColorScheme(v []Component) error {
	if len(v) == 1 && isWord(v[0], "normal") {
		return nil
	}
	names, only := 0, 0
	for i, c := range v {
		switch {
		case c.Kind != Ident || isWord(c, "normal", kwNone, "inherit", "initial", "unset", "revert", "default"):
			return fmt.Errorf("color-scheme takes normal, or scheme names and only, not %q", c.String())
		case isWord(c, "only"):
			if only > 0 || i != 0 && i != len(v)-1 {
				return fmt.Errorf("only stands once, first or last")
			}
			only++
		default:
			names++
		}
	}
	if names == 0 {
		return fmt.Errorf("color-scheme names no scheme")
	}
	return nil
}
