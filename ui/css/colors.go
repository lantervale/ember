package css

import (
	"fmt"
	"math"
	"strings"
)

// space is a colour space a colour function writes its channels in.
type space uint8

const (
	spaceSRGB space = iota
	spaceSRGBLinear
	spaceDisplayP3
	spaceA98
	spaceProPhoto
	spaceRec2020
	spaceXYZD50
	spaceXYZD65
	spaceLab
	spaceLCH
	spaceOklab
	spaceOklch
	spaceHSL
	spaceHWB
)

// spaceInfo is how a space's channels read: each one's name in a
// relative colour, what 100% stands for (0 for a hue), and its kind, as
// color-mix() carries a missing channel to the analogous one of another
// space.
type spaceInfo struct {
	names [3]string
	pct   [3]float64
	kind  [3]channelKind
	// fn is the function that writes it, "" for color()'s spaces.
	fn string
}

type channelKind uint8

const (
	chanRed channelKind = iota + 1
	chanGreen
	chanBlue
	chanLightness
	chanColorfulness
	chanHue
	chanOpposingA
	chanOpposingB
)

//nolint:gochecknoglobals // the colour spaces' table
var (
	rgbInfo = spaceInfo{names: [3]string{"r", "g", "b"}, pct: [3]float64{1, 1, 1},
		kind: [3]channelKind{chanRed, chanGreen, chanBlue}}
	xyzInfo = spaceInfo{names: [3]string{"x", "y", "z"}, pct: [3]float64{1, 1, 1},
		kind: [3]channelKind{chanRed, chanGreen, chanBlue}}
	spaces = [...]spaceInfo{
		spaceSRGB: rgbInfo, spaceSRGBLinear: rgbInfo, spaceDisplayP3: rgbInfo, spaceA98: rgbInfo,
		spaceProPhoto: rgbInfo, spaceRec2020: rgbInfo, spaceXYZD50: xyzInfo, spaceXYZD65: xyzInfo,
		spaceLab: {names: [3]string{"l", "a", "b"}, pct: [3]float64{100, 125, 125}, fn: "lab",
			kind: [3]channelKind{chanLightness, chanOpposingA, chanOpposingB}},
		spaceLCH: {names: [3]string{"l", "c", "h"}, pct: [3]float64{100, 150, 0}, fn: "lch",
			kind: [3]channelKind{chanLightness, chanColorfulness, chanHue}},
		spaceOklab: {names: [3]string{"l", "a", "b"}, pct: [3]float64{1, 0.4, 0.4}, fn: "oklab",
			kind: [3]channelKind{chanLightness, chanOpposingA, chanOpposingB}},
		spaceOklch: {names: [3]string{"l", "c", "h"}, pct: [3]float64{1, 0.4, 0}, fn: "oklch",
			kind: [3]channelKind{chanLightness, chanColorfulness, chanHue}},
		spaceHSL: {names: [3]string{"h", "s", "l"}, pct: [3]float64{0, 100, 100}, fn: "hsl",
			kind: [3]channelKind{chanHue, chanColorfulness, chanLightness}},
		spaceHWB: {names: [3]string{"h", "w", "b"}, pct: [3]float64{0, 100, 100}, fn: "hwb",
			kind: [3]channelKind{chanHue, 0, 0}},
	}
	// predefinedSpaces are color()'s spaces by name, and the spaces color-mix()
	// interpolates in besides the functions' own.
	predefinedSpaces = map[string]space{
		"srgb": spaceSRGB, "srgb-linear": spaceSRGBLinear, "display-p3": spaceDisplayP3, "a98-rgb": spaceA98,
		"prophoto-rgb": spaceProPhoto, "rec2020": spaceRec2020, "xyz": spaceXYZD65, "xyz-d50": spaceXYZD50,
		"xyz-d65": spaceXYZD65,
	}
	functionSpaces = map[string]space{
		"rgb": spaceSRGB, "rgba": spaceSRGB, "hsl": spaceHSL, "hsla": spaceHSL, "hwb": spaceHWB, "lab": spaceLab,
		"lch": spaceLCH, "oklab": spaceOklab, "oklch": spaceOklch,
	}
)

// rgb() and rgba() write their channels from 0 to 255, where color()'s
// sRGB is 0 to 1.
const rgbScale = 255

// colour is a colour in a space: its channels, alpha, and the components
// written none, missing to color-mix().
type colour struct {
	sp      space
	ch      [3]float64
	alpha   float64
	missing [4]bool
}

func (c colour) polar() int {
	switch c.sp {
	case spaceLCH, spaceOklch:
		return 2
	case spaceHSL, spaceHWB:
		return 0
	}
	return -1
}

// colourParser reads a colour within others: cur is what currentcolor
// stands for, nil where it is not known yet, and taken as black.
type colourParser struct {
	cur *Color
}

// colorIn is parseColor with currentcolor standing for cur inside the
// colour functions, nil for black where it is not known yet.
func colorIn(c Component, cur *Color) (Color, error) {
	col, err := colourParser{cur: cur}.read(c, false)
	if err != nil {
		return Color{}, err
	}
	return col.sRGB(), nil
}

// read reads c as a colour, currentcolor among them when inner.
func (p colourParser) read(c Component, inner bool) (colour, error) {
	if inner && isWord(c, kwCurrentColor) {
		if p.cur == nil {
			return colour{alpha: 1}, nil
		}
		return colour{ch: [3]float64{p.cur.R, p.cur.G, p.cur.B}, alpha: p.cur.A}, nil
	}
	if c.Kind == Function {
		name := strings.ToLower(c.Text)
		if sp, ok := functionSpaces[name]; ok {
			return p.function(name, sp, c.Args)
		}
		switch name {
		case "color":
			return p.colorFunction(c.Args)
		case "color-mix":
			return p.mix(c.Args)
		case "light-dark":
			light, dark, ok := lightDark(c.Args)
			if !ok {
				return colour{}, fmt.Errorf("light-dark() takes two colours, separated by a comma")
			}
			if _, err := p.read(dark, true); err != nil {
				return colour{}, fmt.Errorf("light-dark(): %w", err)
			}
			col, err := p.read(light, true)
			if err != nil {
				return colour{}, fmt.Errorf("light-dark(): %w", err)
			}
			return col, nil
		}
	}
	s, err := plainColor(c)
	if err != nil {
		return colour{}, err
	}
	return colour{ch: [3]float64{s.R, s.G, s.B}, alpha: s.A}, nil
}

// function reads name(), a function of space sp: its channels in the
// modern form, rgb() and hsl() in the legacy one too, or relative to an
// origin colour after from.
func (p colourParser) function(name string, sp space, args []Component) (colour, error) {
	toks := items(args)
	if len(toks) > 0 && isWord(toks[0], "from") {
		if len(toks) < 2 {
			return colour{}, fmt.Errorf("%s(): from needs a colour", name)
		}
		origin, err := p.read(toks[1], true)
		if err != nil {
			return colour{}, fmt.Errorf("%s(): %w", name, err)
		}
		col, err := p.channels(sp, channelScale(sp), toks[2:], &origin)
		if err != nil {
			return colour{}, fmt.Errorf("%s(): %w", name, err)
		}
		return col, nil
	}
	var col colour
	var err error
	if (sp == spaceSRGB || sp == spaceHSL) && containsComma(toks) {
		col, err = p.legacy(sp, toks)
	} else {
		col, err = p.channels(sp, channelScale(sp), toks, nil)
	}
	if err != nil {
		return colour{}, fmt.Errorf("%s(): %w", name, err)
	}
	return col, nil
}

func containsComma(toks []Component) bool {
	for _, t := range toks {
		if t.Kind == Comma {
			return true
		}
	}
	return false
}

// colorFunction reads color(): a predefinedSpaces space and its channels,
// relative to an origin colour after from or not.
func (p colourParser) colorFunction(args []Component) (colour, error) {
	toks := items(args)
	var origin *colour
	if len(toks) > 0 && isWord(toks[0], "from") {
		if len(toks) < 2 {
			return colour{}, fmt.Errorf("color(): from needs a colour")
		}
		o, err := p.read(toks[1], true)
		if err != nil {
			return colour{}, fmt.Errorf("color(): %w", err)
		}
		origin, toks = &o, toks[2:]
	}
	if len(toks) == 0 || toks[0].Kind != Ident {
		return colour{}, fmt.Errorf("color() takes a colour space first")
	}
	sp, ok := predefinedSpaces[strings.ToLower(toks[0].Text)]
	if !ok {
		return colour{}, fmt.Errorf("color(): unknown colour space %q", toks[0].Text)
	}
	col, err := p.channels(sp, 1, toks[1:], origin)
	if err != nil {
		return colour{}, fmt.Errorf("color(): %w", err)
	}
	return col, nil
}

// channels reads three channels and an optional alpha after '/', in the
// modern form, each a number, a percentage, none, a hue's angle or a
// math function of them, written scale times the space's own; relative
// to origin, the channels' names stand for its channels in sp, as
// numbers.
func (p colourParser) channels(sp space, scale float64, toks []Component, origin *colour) (colour, error) {
	info := spaces[sp]
	var names map[string]float64
	alpha := 1.0
	if origin != nil {
		o := origin.to(sp)
		names = map[string]float64{"alpha": o.alpha}
		for i, n := range info.names {
			v := o.ch[i]
			if o.missing[i] || math.IsNaN(v) {
				v = 0
			}
			names[n] = v * scale
		}
		alpha = o.alpha
	}
	n := len(toks)
	var alphaTok *Component
	if n >= 2 && isSlash(toks[n-2]) {
		alphaTok, toks = &toks[n-1], toks[:n-2]
	}
	if len(toks) != 3 {
		return colour{}, fmt.Errorf("expected three channels, found %d", len(toks))
	}
	col := colour{sp: sp, alpha: alpha}
	for i, t := range toks {
		v, none, err := channelValue(t, info.pct[i]*scale, names)
		if err != nil {
			return colour{}, err
		}
		col.ch[i], col.missing[i] = v/scale, none
	}
	if alphaTok != nil {
		v, none, err := channelValue(*alphaTok, 1, names)
		if err != nil {
			return colour{}, err
		}
		col.alpha, col.missing[3] = v, none
	}
	col.alpha = clamp01(col.alpha)
	col.clampChannels()
	return col, nil
}

// channelScale is how many times the space's own its function writes the
// channels: rgb()'s 0–255.
func channelScale(sp space) float64 {
	if sp == spaceSRGB {
		return rgbScale
	}
	return 1
}

// clampChannels clamps what the space's function clamps when it reads
// its channels, and turns a hue into 0–360.
func (c *colour) clampChannels() {
	switch c.sp {
	case spaceLab:
		c.ch[0] = min(max(c.ch[0], 0), 100)
	case spaceLCH:
		c.ch[0] = min(max(c.ch[0], 0), 100)
		c.ch[1] = max(c.ch[1], 0)
	case spaceOklab:
		c.ch[0] = clamp01(c.ch[0])
	case spaceOklch:
		c.ch[0] = clamp01(c.ch[0])
		c.ch[1] = max(c.ch[1], 0)
	case spaceHSL:
		c.ch[1] = max(c.ch[1], 0)
	case spaceSRGB:
		for i := range c.ch {
			c.ch[i] = clamp01(c.ch[i])
		}
	case spaceSRGBLinear, spaceDisplayP3, spaceA98, spaceProPhoto, spaceRec2020, spaceXYZD50, spaceXYZD65, spaceHWB:
	}
	if c.polar() >= 0 {
		c.ch[c.polar()] = normalHue(c.ch[c.polar()])
	}
}

func normalHue(h float64) float64 {
	h = math.Mod(h, 360)
	if h < 0 {
		h += 360
	}
	return h
}

// channelValue reads one channel: a number, a percentage of pct, a hue
// (pct 0) in degrees, none, a channel's name where names has it, or a
// math function of them.
func channelValue(c Component, pct float64, names map[string]float64) (float64, bool, error) {
	switch c.Kind {
	case Number:
		return c.Num, false, nil
	case Percentage:
		if pct == 0 {
			return 0, false, fmt.Errorf("a hue takes no percentage")
		}
		return c.Num / 100 * pct, false, nil
	case Ident:
		if isWord(c, kwNone) {
			return 0, true, nil
		}
		if v, ok := names[strings.ToLower(c.Text)]; ok {
			return v, false, nil
		}
	case Dimension, Function:
		if c.Kind == Function {
			c = withChannels(c, names)
		}
		d, err := dimOf(c)
		if err != nil {
			return 0, false, fmt.Errorf("bad colour channel %q", c.String())
		}
		v, ok := d.constant()
		switch {
		case d.typ == typePercent && pct != 0:
			return d.Resolve(100) / 100 * pct, false, nil
		case d.typ == typeNumber && !d.pct:
			return v, false, okErr(ok, c)
		case d.typ == typeAngle && pct == 0 && !d.pct:
			return v, false, okErr(ok, c)
		}
	case EOF, AtKeyword, Hash, String, BadString, URL, BadURL, Delim, Whitespace, CDO, CDC, Colon, Semicolon, Comma,
		LBracket, RBracket, LParen, RParen, LBrace, RBrace:
		// nothing special for these tokens
	}
	return 0, false, fmt.Errorf("bad colour channel %q", c.String())
}

func okErr(ok bool, c Component) error {
	if ok {
		return nil
	}
	return fmt.Errorf("bad colour channel %q", c.String())
}

// withChannels is c with the channels' names in it, at any depth, made the
// numbers they stand for.
func withChannels(c Component, names map[string]float64) Component {
	if len(names) == 0 || len(c.Args) == 0 {
		return c
	}
	args := make([]Component, len(c.Args))
	for i, a := range c.Args {
		switch {
		case a.Kind == Ident:
			if v, ok := names[strings.ToLower(a.Text)]; ok {
				a = Component{Token: Token{Kind: Number, Num: v, Pos: a.Pos, Text: fmt.Sprint(v)}}
			}
		case len(a.Args) > 0:
			a = withChannels(a, names)
		}
		args[i] = a
	}
	c.Args = args
	return c
}

// legacy reads rgb()'s and hsl()'s channels separated by commas: rgb()'s
// all numbers or all percentages, hsl()'s a hue then percentages, and an
// optional alpha.
func (p colourParser) legacy(sp space, toks []Component) (colour, error) {
	if !commasBetweenItems(toks) || len(toks) != 5 && len(toks) != 7 {
		return colour{}, fmt.Errorf("bad commas in colour")
	}
	info := spaces[sp]
	col := colour{sp: sp, alpha: 1}
	var kinds [3]dimType
	for i := range 3 {
		t := toks[2*i]
		if isWord(t, kwNone) {
			return colour{}, fmt.Errorf("none can't be used with commas")
		}
		v, _, err := channelValue(t, info.pct[i]*channelScale(sp), nil)
		if err != nil {
			return colour{}, err
		}
		kinds[i] = typeNumber
		if d, derr := dimOf(t); derr == nil {
			kinds[i] = d.typ
		}
		col.ch[i] = v / channelScale(sp)
	}
	switch sp {
	case spaceSRGB:
		if kinds[1] != kinds[0] || kinds[2] != kinds[0] {
			return colour{}, fmt.Errorf("legacy channels must all be numbers or all percentages")
		}
	default:
		if kinds[1] != typePercent || kinds[2] != typePercent {
			return colour{}, fmt.Errorf("legacy saturation and lightness are percentages")
		}
	}
	if len(toks) == 7 {
		v, none, err := channelValue(toks[6], 1, nil)
		if err != nil || none {
			return colour{}, fmt.Errorf("bad alpha %q", toks[6].String())
		}
		col.alpha = v
	}
	col.alpha = clamp01(col.alpha)
	col.clampChannels()
	return col, nil
}

func commasBetweenItems(toks []Component) bool {
	for i, t := range toks {
		if (i%2 == 1) != (t.Kind == Comma) {
			return false
		}
	}
	return true
}

// hueMethods are color-mix()'s ways round the hue circle.
//
//nolint:gochecknoglobals // lookup table
var hueMethods = []string{"shorter", "longer", "increasing", "decreasing"}

// mix reads color-mix(): in a space and a way round the hue for a polar
// one, then two colours, each with a percentage or not.
func (p colourParser) mix(args []Component) (colour, error) {
	parts := splitOn(args, func(c Component) bool { return c.Kind == Comma })
	if len(parts) != 3 {
		return colour{}, fmt.Errorf("color-mix() takes a space and two colours")
	}
	head := items(parts[0])
	if len(head) < 2 || !isWord(head[0], "in") || head[1].Kind != Ident {
		return colour{}, fmt.Errorf("color-mix() begins with in and a colour space")
	}
	name := strings.ToLower(head[1].Text)
	sp, ok := predefinedSpaces[name]
	if !ok {
		sp, ok = functionSpaces[name]
		if !ok || name == "rgb" || name == "rgba" || name == "hsla" {
			return colour{}, fmt.Errorf("color-mix(): unknown colour space %q", head[1].Text)
		}
	}
	method := 0
	if len(head) > 2 {
		probe := colour{sp: sp}
		if len(head) != 4 || probe.polar() < 0 || !isWord(head[3], "hue") || !isWord(head[2], hueMethods...) {
			return colour{}, fmt.Errorf("color-mix(): %q is no hue interpolation of %s", join(parts[0]), name)
		}
		for i, m := range hueMethods {
			if isWord(head[2], m) {
				method = i
			}
		}
	}
	var cols [2]colour
	var pcts [2]float64
	var given [2]bool
	for i, part := range parts[1:] {
		toks := items(part)
		var err error
		switch {
		case len(toks) == 1:
			cols[i], err = p.read(toks[0], true)
		case len(toks) == 2:
			c, pc := toks[0], toks[1]
			if isPercentage(c) {
				c, pc = pc, c
			}
			cols[i], err = p.read(c, true)
			if err == nil {
				pcts[i], err = mixPercent(pc)
				given[i] = true
			}
		default:
			err = fmt.Errorf("expected a colour and a percentage, found %q", join(part))
		}
		if err != nil {
			return colour{}, fmt.Errorf("color-mix(): %w", err)
		}
	}
	switch {
	case !given[0] && !given[1]:
		pcts = [2]float64{50, 50}
	case !given[1]:
		pcts[1] = 100 - pcts[0]
	case !given[0]:
		pcts[0] = 100 - pcts[1]
	}
	return mixColours(cols, pcts, sp, method), nil
}

func isPercentage(c Component) bool {
	if c.Kind == Percentage {
		return true
	}
	if c.Kind != Function {
		return false
	}
	d, err := dimOf(c)
	return err == nil && d.typ == typePercent
}

// mixPercent reads a color-mix() percentage, 0% to 100%.
func mixPercent(c Component) (float64, error) {
	d, err := dimOf(c)
	if err != nil || d.typ != typePercent {
		return 0, fmt.Errorf("expected a percentage, found %q", c.String())
	}
	v := d.Resolve(100)
	if c.Kind == Percentage && (v < 0 || v > 100) {
		return 0, fmt.Errorf("a percentage of color-mix() is 0%% to 100%%, found %q", c.String())
	}
	return min(max(v, 0), 100), nil
}

// mixColours mixes cols in sp, pcts of each, round the hue by method
// (hueMethods): premultiplied by alpha, a missing component taking the
// other colour's, and alpha scaled down where pcts sum under 100.
func mixColours(cols [2]colour, pcts [2]float64, sp space, method int) colour {
	sum := pcts[0] + pcts[1]
	if sum == 0 {
		return colour{sp: sp}
	}
	t := pcts[1] / sum
	a, b := cols[0].to(sp), cols[1].to(sp)
	for i := range 4 {
		switch {
		case a.missing[i] && !b.missing[i]:
			a.set(i, b.get(i))
			a.missing[i] = false
		case b.missing[i] && !a.missing[i]:
			b.set(i, a.get(i))
			b.missing[i] = false
		}
	}
	out := colour{sp: sp}
	out.missing = a.missing
	out.alpha = a.alpha*(1-t) + b.alpha*t
	hue := a.polar()
	if hue >= 0 && !a.missing[hue] {
		a.ch[hue], b.ch[hue] = fixHues(normalHue(a.ch[hue]), normalHue(b.ch[hue]), method)
	}
	for i := range 3 {
		if i == hue {
			out.ch[i] = a.ch[i]*(1-t) + b.ch[i]*t
			continue
		}
		out.ch[i] = a.ch[i]*a.alpha*(1-t) + b.ch[i]*b.alpha*t
		if out.alpha != 0 {
			out.ch[i] /= out.alpha
		}
	}
	out.alpha *= min(sum, 100) / 100
	if hue >= 0 {
		out.ch[hue] = normalHue(out.ch[hue])
	}
	return out
}

func fixHues(h1, h2 float64, method int) (float64, float64) {
	d := h2 - h1
	switch hueMethods[method] {
	case "shorter":
		if d > 180 {
			h1 += 360
		} else if d < -180 {
			h2 += 360
		}
	case "longer":
		if d > 0 && d < 180 {
			h1 += 360
		} else if d > -180 && d <= 0 {
			h2 += 360
		}
	case "increasing":
		if d < 0 {
			h2 += 360
		}
	case "decreasing":
		if d > 0 {
			h1 += 360
		}
	}
	return h1, h2
}

func (c *colour) get(i int) float64 {
	if i == 3 {
		return c.alpha
	}
	return c.ch[i]
}

func (c *colour) set(i int, v float64) {
	if i == 3 {
		c.alpha = v
		return
	}
	c.ch[i] = v
}

// to is c in sp, its missing components carried to the analogous ones of
// sp, and a hue made missing where the colour has none.
func (c colour) to(sp space) colour {
	if c.sp == sp {
		return c
	}
	src := c
	for i := range 3 {
		if src.missing[i] {
			src.ch[i] = 0
		}
	}
	out := colour{sp: sp, alpha: c.alpha, ch: fromXYZ(sp, src.xyz())}
	out.missing[3] = c.missing[3]
	from, into := spaces[c.sp].kind, spaces[sp].kind
	for i, k := range from {
		if !c.missing[i] || k == 0 {
			continue
		}
		for j, kk := range into {
			if kk == k {
				out.missing[j] = true
			}
		}
	}
	if h := out.polar(); h >= 0 && out.powerless() {
		out.missing[h] = true
	}
	return out
}

// powerless reports whether the colour, in a polar space, is grey, its
// hue meaning nothing.
func (c colour) powerless() bool {
	const eps = 1e-6
	switch c.sp {
	case spaceLCH:
		return c.ch[1] <= 0.0015*150/100
	case spaceOklch:
		return c.ch[1] <= 0.000004
	case spaceHSL:
		return math.Abs(c.ch[1]) <= eps
	case spaceHWB:
		return c.ch[1]+c.ch[2] >= 100-eps
	case spaceSRGB, spaceSRGBLinear, spaceDisplayP3, spaceA98, spaceProPhoto, spaceRec2020, spaceXYZD50, spaceXYZD65,
		spaceLab, spaceOklab:
	}
	return false
}

// sRGB is c in sRGB, clipped to its gamut.
func (c colour) sRGB() Color {
	s := c.to(spaceSRGB)
	if c.sp == spaceSRGB {
		for i := range 3 {
			if s.missing[i] {
				s.ch[i] = 0
			}
		}
	}
	return Color{clamp01(s.ch[0]), clamp01(s.ch[1]), clamp01(s.ch[2]), clamp01(s.alpha)}
}

type mat3 [3][3]float64

func (m mat3) apply(v [3]float64) [3]float64 {
	var out [3]float64
	for i := range 3 {
		out[i] = m[i][0]*v[0] + m[i][1]*v[1] + m[i][2]*v[2]
	}
	return out
}

// The matrices of CSS Color 4's sample code.
//
//nolint:gochecknoglobals // constants
var (
	srgbToXYZ = mat3{{0.41239079926595934, 0.357584339383878, 0.1804807884018343},
		{0.21263900587151027, 0.715168678767756, 0.07219231536073371},
		{0.01933081871559182, 0.11919477979462598, 0.9505321522496607}}
	xyzToSRGB = mat3{{3.2409699419045226, -1.537383177570094, -0.4986107602930034},
		{-0.9692436362808796, 1.8759675015077202, 0.04155505740717559},
		{0.05563007969699366, -0.20397695888897652, 1.0569715142428786}}
	p3ToXYZ = mat3{{0.4865709486482162, 0.26566769316909306, 0.1982172852343625},
		{0.2289745640697488, 0.6917385218365064, 0.079286914093745},
		{0, 0.04511338185890264, 1.043944368900976}}
	xyzToP3 = mat3{{2.493496911941425, -0.9313836179191239, -0.40271078445071684},
		{-0.8294889695615747, 1.7626640603183463, 0.023624685841943577},
		{0.03584583024378447, -0.07617238926804182, 0.9568845240076872}}
	a98ToXYZ = mat3{{0.5766690429101305, 0.1855582379065463, 0.1882286462349947},
		{0.29734497525053605, 0.6273635662554661, 0.07529145849399788},
		{0.02703136138641234, 0.07068885253582723, 0.9913375368376388}}
	xyzToA98 = mat3{{2.0415879038107465, -0.5650069742788596, -0.34473135077832956},
		{-0.9692436362808795, 1.8759675015077202, 0.04155505740717557},
		{0.013444280632031142, -0.11836239223101838, 1.0151749943912054}}
	proPhotoToXYZD50 = mat3{{0.7977666449006423, 0.13518129740053308, 0.0313477341283922},
		{0.2880748288194013, 0.711835234241873, 0.00008993693872564},
		{0, 0, 0.8251046025104602}}
	xyzD50ToProPhoto = mat3{{1.3457868816471583, -0.25557208737979464, -0.05110186497554526},
		{-0.5446307051249019, 1.5082477428451468, 0.02052744743642139},
		{0, 0, 1.2119675456389452}}
	rec2020ToXYZ = mat3{{0.6369580483012914, 0.14461690358620832, 0.1688809751641721},
		{0.2627002120112671, 0.6779980715188708, 0.05930171646986196},
		{0, 0.028072693049087428, 1.060985057710791}}
	xyzToRec2020 = mat3{{1.7166511879712674, -0.35567078377639233, -0.25336628137365974},
		{-0.6666843518324892, 1.6164812366349395, 0.01576854581391113},
		{0.017639857445310783, -0.042770613257808524, 0.9421031212354738}}
	d65ToD50 = mat3{{1.0479297925449969, 0.022946870601609652, -0.05019226628920524},
		{0.02962780877005599, 0.9904344267538799, -0.017073799063418826},
		{-0.009243040646204504, 0.015055191490298152, 0.7518742814281371}}
	d50ToD65 = mat3{{0.955473421488075, -0.02309845494876471, 0.06325924320057072},
		{-0.0283697093338637, 1.0099953980813041, 0.021041441191917323},
		{0.012314014864481998, -0.020507649298898964, 1.330365926242124}}
	xyzToLMS = mat3{{0.8190224379967030, 0.3619062600528904, -0.1288737815209879},
		{0.0329836539323885, 0.9292868615863434, 0.0361446663506424},
		{0.0481771893596242, 0.2642395317527308, 0.6335478284694309}}
	lmsToOklab = mat3{{0.2104542683093140, 0.7936177747023054, -0.0040720430116193},
		{1.9779985324311684, -2.4285922420485799, 0.4505937096174110},
		{0.0259040424655478, 0.7827717124575296, -0.8086757660831043}}
	oklabToLMS = mat3{{1, 0.3963377773761749, 0.2158037573099136},
		{1, -0.1055613458156586, -0.0638541728258133},
		{1, -0.0894841775298119, -1.2914855480194092}}
	lmsToXYZ = mat3{{1.2268798758459243, -0.5578149944602171, 0.2813910456659647},
		{-0.0405757452148008, 1.1122868032803170, -0.0717110580655164},
		{-0.0763729366746601, -0.4214933324022432, 1.5869240198367816}}
	d50White = [3]float64{0.3457 / 0.3585, 1, (1 - 0.3457 - 0.3585) / 0.3585}
)

func each(v [3]float64, f func(float64) float64) [3]float64 {
	return [3]float64{f(v[0]), f(v[1]), f(v[2])}
}

// transfer is a signed power curve with a linear part near 0.
func signed(v float64, f func(float64) float64) float64 {
	if v < 0 {
		return -f(-v)
	}
	return f(v)
}

func srgbToLinear(v float64) float64 {
	return signed(v, func(a float64) float64 {
		if a <= 0.04045 {
			return a / 12.92
		}
		return math.Pow((a+0.055)/1.055, 2.4)
	})
}

func linearToSRGB(v float64) float64 {
	return signed(v, func(a float64) float64 {
		if a > 0.0031308 {
			return 1.055*math.Pow(a, 1/2.4) - 0.055
		}
		return 12.92 * a
	})
}

const (
	rec2020Alpha = 1.09929682680944
	rec2020Beta  = 0.018053968510807
	labEpsilon   = 216.0 / 24389
	labKappa     = 24389.0 / 27
)

// xyz is c in XYZ under D65.
func (c colour) xyz() [3]float64 {
	v := c.ch
	switch c.sp {
	case spaceSRGB:
		return srgbToXYZ.apply(each(v, srgbToLinear))
	case spaceSRGBLinear:
		return srgbToXYZ.apply(v)
	case spaceDisplayP3:
		return p3ToXYZ.apply(each(v, srgbToLinear))
	case spaceA98:
		return a98ToXYZ.apply(each(v, func(x float64) float64 {
			return signed(x, func(a float64) float64 { return math.Pow(a, 563.0/256) })
		}))
	case spaceProPhoto:
		return d50ToD65.apply(proPhotoToXYZD50.apply(each(v, func(x float64) float64 {
			return signed(x, func(a float64) float64 {
				if a <= 16.0/512 {
					return a / 16
				}
				return math.Pow(a, 1.8)
			})
		})))
	case spaceRec2020:
		return rec2020ToXYZ.apply(each(v, func(x float64) float64 {
			return signed(x, func(a float64) float64 {
				if a < rec2020Beta*4.5 {
					return a / 4.5
				}
				return math.Pow((a+rec2020Alpha-1)/rec2020Alpha, 1/0.45)
			})
		}))
	case spaceXYZD50:
		return d50ToD65.apply(v)
	case spaceXYZD65:
		return v
	case spaceLab:
		return d50ToD65.apply(labToXYZD50(v))
	case spaceLCH:
		return d50ToD65.apply(labToXYZD50(polarToLab(v)))
	case spaceOklab:
		return oklabToXYZ(v)
	case spaceOklch:
		return oklabToXYZ(polarToLab(v))
	case spaceHSL:
		return colour{sp: spaceSRGB, ch: hslToSRGB(v)}.xyz()
	case spaceHWB:
		return colour{sp: spaceSRGB, ch: hwbToSRGB(v)}.xyz()
	}
	return v
}

// fromXYZ is v, in XYZ under D65, in sp.
func fromXYZ(sp space, v [3]float64) [3]float64 {
	switch sp {
	case spaceSRGB:
		return each(xyzToSRGB.apply(v), linearToSRGB)
	case spaceSRGBLinear:
		return xyzToSRGB.apply(v)
	case spaceDisplayP3:
		return each(xyzToP3.apply(v), linearToSRGB)
	case spaceA98:
		return each(xyzToA98.apply(v), func(x float64) float64 {
			return signed(x, func(a float64) float64 { return math.Pow(a, 256.0/563) })
		})
	case spaceProPhoto:
		return each(xyzD50ToProPhoto.apply(d65ToD50.apply(v)), func(x float64) float64 {
			return signed(x, func(a float64) float64 {
				if a >= 1.0/512 {
					return math.Pow(a, 1/1.8)
				}
				return 16 * a
			})
		})
	case spaceRec2020:
		return each(xyzToRec2020.apply(v), func(x float64) float64 {
			return signed(x, func(a float64) float64 {
				if a > rec2020Beta {
					return rec2020Alpha*math.Pow(a, 0.45) - (rec2020Alpha - 1)
				}
				return 4.5 * a
			})
		})
	case spaceXYZD50:
		return d65ToD50.apply(v)
	case spaceXYZD65:
		return v
	case spaceLab:
		return xyzD50ToLab(d65ToD50.apply(v))
	case spaceLCH:
		return labToPolar(xyzD50ToLab(d65ToD50.apply(v)))
	case spaceOklab:
		return xyzToOklab(v)
	case spaceOklch:
		return labToPolar(xyzToOklab(v))
	case spaceHSL:
		return srgbToHSL(fromXYZ(spaceSRGB, v))
	case spaceHWB:
		return srgbToHWB(fromXYZ(spaceSRGB, v))
	}
	return v
}

func labToXYZD50(lab [3]float64) [3]float64 {
	f1 := (lab[0] + 16) / 116
	f0, f2 := lab[1]/500+f1, f1-lab[2]/200
	inv := func(f float64) float64 {
		if f*f*f > labEpsilon {
			return f * f * f
		}
		return (116*f - 16) / labKappa
	}
	y := lab[0] / labKappa
	if lab[0] > labKappa*labEpsilon {
		y = f1 * f1 * f1
	}
	return [3]float64{inv(f0) * d50White[0], y * d50White[1], inv(f2) * d50White[2]}
}

func xyzD50ToLab(v [3]float64) [3]float64 {
	var f [3]float64
	for i := range 3 {
		x := v[i] / d50White[i]
		if x > labEpsilon {
			f[i] = math.Cbrt(x)
		} else {
			f[i] = (labKappa*x + 16) / 116
		}
	}
	return [3]float64{116*f[1] - 16, 500 * (f[0] - f[1]), 200 * (f[1] - f[2])}
}

func oklabToXYZ(v [3]float64) [3]float64 {
	return lmsToXYZ.apply(each(oklabToLMS.apply(v), func(x float64) float64 { return x * x * x }))
}

func xyzToOklab(v [3]float64) [3]float64 {
	return lmsToOklab.apply(each(xyzToLMS.apply(v), math.Cbrt))
}

func polarToLab(v [3]float64) [3]float64 {
	h := v[2] * math.Pi / 180
	return [3]float64{v[0], v[1] * math.Cos(h), v[1] * math.Sin(h)}
}

func labToPolar(v [3]float64) [3]float64 {
	return [3]float64{v[0], math.Hypot(v[1], v[2]), normalHue(math.Atan2(v[2], v[1]) * 180 / math.Pi)}
}

// hslToSRGB is hsl(), its saturation and lightness 0–100, in sRGB.
func hslToSRGB(v [3]float64) [3]float64 {
	h, s, l := normalHue(v[0]), v[1]/100, v[2]/100
	f := func(n float64) float64 {
		k := math.Mod(n+h/30, 12)
		a := s * min(l, 1-l)
		return l - a*max(-1, min(k-3, 9-k, 1))
	}
	return [3]float64{f(0), f(8), f(4)}
}

func hwbToSRGB(v [3]float64) [3]float64 {
	w, b := v[1]/100, v[2]/100
	if w+b >= 1 {
		g := w / (w + b)
		return [3]float64{g, g, g}
	}
	rgb := hslToSRGB([3]float64{v[0], 100, 50})
	return each(rgb, func(x float64) float64 { return x*(1-w-b) + w })
}

func srgbToHSL(v [3]float64) [3]float64 {
	r, g, b := v[0], v[1], v[2]
	hi, lo := max(r, g, b), min(r, g, b)
	h, s, l := 0.0, 0.0, (hi+lo)/2
	if d := hi - lo; d != 0 {
		if l != 0 && l != 1 {
			s = (hi - l) / min(l, 1-l)
		}
		switch hi {
		case r:
			h = (g - b) / d
			if g < b {
				h += 6
			}
		case g:
			h = (b-r)/d + 2
		default:
			h = (r-g)/d + 4
		}
		h *= 60
	}
	if s < 0 {
		h += 180
		s = -s
	}
	return [3]float64{normalHue(h), s * 100, l * 100}
}

func srgbToHWB(v [3]float64) [3]float64 {
	h := srgbToHSL(v)[0]
	return [3]float64{h, min(v[0], v[1], v[2]) * 100, (1 - max(v[0], v[1], v[2])) * 100}
}
