package css

import (
	"math"
	"slices"
	"strings"
)

// Media is what @media rules are answered for: the viewport's size in
// CSS pixels and the device pixels per CSS pixel, 1 when 0. The rest is
// fixed: a colour screen in a light scheme with a mouse, and no
// preference for less motion or more contrast. ui knows no screen, so
// the device-* features are the viewport's.
type Media struct {
	Width, Height float32
	Resolution    float32
}

func (m Media) dppx() float64 {
	if m.Resolution <= 0 {
		return 1
	}
	return float64(m.Resolution)
}

// tri is a media condition's answer, as Media Queries 4 has it: a
// feature ui doesn't know is neither true nor false, and stays so under
// not.
type tri int8

const (
	no tri = iota - 1
	unknown
	yes
)

func triOf(b bool) tri {
	if b {
		return yes
	}
	return no
}

// mediaList is an @media rule's queries, one of which must hold. text is
// the list as written.
type mediaList struct {
	text    string
	queries []mediaQuery
}

// mediaQuery is one query: [not | only] type [and condition], or a
// condition alone. A malformed one never holds.
type mediaQuery struct {
	bad, not bool
	// kind is the media type in lower case, "" for none.
	kind string
	cond *mediaCond
}

type mediaOp uint8

const (
	mediaFeature mediaOp = iota
	mediaNot
	mediaAnd
	mediaOr
	// mediaUnknown is a feature, a value or a form ui doesn't read.
	mediaUnknown
)

type mediaCond struct {
	op   mediaOp
	kids []mediaCond
	// spec and bounds are a feature's: every bound must hold; none asks
	// whether the feature is on at all.
	spec   *mediaSpec
	bounds []mediaBound
	// word is the value asked of a feature that has words.
	word string
}

// mediaBound is one side of a range: the feature compared with a value.
type mediaBound struct {
	// cmp is '<', '>' or '=', orEqual adding equality to the first two.
	cmp     byte
	orEqual bool
	value   mediaValue
}

// mediaValue is a number in the feature's own unit, or a share of the
// viewport (vw, vh, vmin, vmax).
type mediaValue struct {
	num  float64
	unit string
}

func (v mediaValue) at(m Media) float64 {
	w, h := float64(m.Width), float64(m.Height)
	switch v.unit {
	case "vw":
		return v.num * w / 100
	case "vh":
		return v.num * h / 100
	case "vmin":
		return v.num * min(w, h) / 100
	case "vmax":
		return v.num * max(w, h) / 100
	}
	return v.num
}

type mediaValueKind uint8

const (
	mediaLength mediaValueKind = iota
	mediaRatio
	mediaResolution
	mediaInteger
	mediaWord
)

// mediaSpec is a feature ui answers: a number of its kind, or one of its
// words, none of them the one that means off.
type mediaSpec struct {
	kind  mediaValueKind
	of    func(m Media) float64
	word  func(m Media) string
	words []string
	off   string
}

func fixedNumber(n float64) func(Media) float64 { return func(Media) float64 { return n } }
func fixedWord(w string) func(Media) string     { return func(Media) string { return w } }

func viewportRatio(m Media) float64 {
	if m.Height <= 0 {
		return math.Inf(1)
	}
	return float64(m.Width) / float64(m.Height)
}

const (
	mediaNone         = "none"
	mediaNoPreference = "no-preference"
)

// mediaSpecs are the features ui answers.
//
//nolint:gochecknoglobals // the media features' table
var mediaSpecs = map[string]*mediaSpec{
	"width":               {kind: mediaLength, of: func(m Media) float64 { return float64(m.Width) }},
	"height":              {kind: mediaLength, of: func(m Media) float64 { return float64(m.Height) }},
	"device-width":        {kind: mediaLength, of: func(m Media) float64 { return float64(m.Width) }},
	"device-height":       {kind: mediaLength, of: func(m Media) float64 { return float64(m.Height) }},
	"aspect-ratio":        {kind: mediaRatio, of: viewportRatio},
	"device-aspect-ratio": {kind: mediaRatio, of: viewportRatio},
	"resolution":          {kind: mediaResolution, of: Media.dppx},
	"color":               {kind: mediaInteger, of: fixedNumber(8)},
	"monochrome":          {kind: mediaInteger, of: fixedNumber(0)},
	"grid":                {kind: mediaInteger, of: fixedNumber(0)},
	"orientation": {kind: mediaWord, words: []string{"portrait", "landscape"}, word: func(m Media) string {
		if m.Height >= m.Width {
			return "portrait"
		}
		return "landscape"
	}},
	"prefers-color-scheme": {kind: mediaWord, words: []string{"light", "dark"}, word: fixedWord("light")},
	"prefers-reduced-motion": {kind: mediaWord, words: []string{mediaNoPreference, "reduce"},
		word: fixedWord(mediaNoPreference), off: mediaNoPreference},
	"prefers-contrast": {kind: mediaWord, words: []string{mediaNoPreference, "more", "less", "custom"},
		word: fixedWord(mediaNoPreference), off: mediaNoPreference},
	"forced-colors": {kind: mediaWord, words: []string{mediaNone, "active"}, word: fixedWord(mediaNone),
		off: mediaNone},
	"hover":     {kind: mediaWord, words: []string{mediaNone, "hover"}, word: fixedWord("hover"), off: mediaNone},
	"any-hover": {kind: mediaWord, words: []string{mediaNone, "hover"}, word: fixedWord("hover"), off: mediaNone},
	"pointer": {kind: mediaWord, words: []string{mediaNone, "coarse", "fine"}, word: fixedWord("fine"),
		off: mediaNone},
	"any-pointer": {kind: mediaWord, words: []string{mediaNone, "coarse", "fine"}, word: fixedWord("fine"),
		off: mediaNone},
}

// matches reports whether one of l's queries holds for m; a list with
// none always does.
func (l *mediaList) matches(m Media) bool {
	if len(l.queries) == 0 {
		return true
	}
	return slices.ContainsFunc(l.queries, func(q mediaQuery) bool { return q.matches(m) })
}

func (q mediaQuery) matches(m Media) bool {
	if q.bad {
		return false
	}
	result := triOf(q.kind == "" || q.kind == "all" || q.kind == "screen")
	if q.cond != nil {
		result = min(result, q.cond.eval(m))
	}
	if q.not {
		result = -result
	}
	return result == yes
}

func (c *mediaCond) eval(m Media) tri {
	switch c.op {
	case mediaNot:
		return -c.kids[0].eval(m)
	case mediaAnd:
		result := yes
		for i := range c.kids {
			result = min(result, c.kids[i].eval(m))
		}
		return result
	case mediaOr:
		result := no
		for i := range c.kids {
			result = max(result, c.kids[i].eval(m))
		}
		return result
	case mediaUnknown:
		return unknown
	case mediaFeature:
		// below
	}
	if c.spec.kind == mediaWord {
		is := c.spec.word(m)
		if c.word == "" {
			return triOf(is != c.spec.off)
		}
		return triOf(is == c.word)
	}
	is := c.spec.of(m)
	if len(c.bounds) == 0 {
		return triOf(is != 0)
	}
	for _, b := range c.bounds {
		v := b.value.at(m)
		holds := b.orEqual && is == v
		switch b.cmp {
		case '<':
			holds = holds || is < v
		case '>':
			holds = holds || is > v
		case '=':
			holds = is == v
		}
		if !holds {
			return no
		}
	}
	return yes
}

// mediaParser reads media queries, telling warn of what makes one
// unknown or malformed. browser reads the vendors' features as a sheet
// read as a browser's does.
type mediaParser struct {
	browser bool
	warn    func(pos Pos, format string, args ...any)
}

// list reads an @media rule's prelude.
func (p *mediaParser) list(prelude []Component) *mediaList {
	l := &mediaList{text: strings.TrimSpace(join(prelude))}
	if len(trim(prelude)) == 0 {
		return l
	}
	for _, part := range splitOn(prelude, func(c Component) bool { return c.Kind == Comma }) {
		q, ok := p.query(items(part))
		if !ok {
			pos := Pos{1, 1}
			if len(prelude) > 0 {
				pos = prelude[0].Pos
			}
			if len(trim(part)) > 0 {
				pos = trim(part)[0].Pos
			}
			p.warn(pos, "@media: %q is no media query, and never holds", strings.TrimSpace(join(part)))
			q = mediaQuery{bad: true}
		}
		l.queries = append(l.queries, q)
	}
	return l
}

func reservedMediaWord(c Component) bool { return isWord(c, "and", "or", "not", "only", "layer") }

func (p *mediaParser) query(toks []Component) (mediaQuery, bool) {
	var q mediaQuery
	if len(toks) == 0 {
		return q, false
	}
	typed := toks
	switch {
	case isWord(toks[0], "not") && len(toks) > 1 && toks[1].Kind == Ident:
		q.not, typed = true, toks[1:]
	case isWord(toks[0], "only"):
		typed = toks[1:]
		if len(typed) == 0 || typed[0].Kind != Ident {
			return q, false
		}
	}
	if typed[0].Kind != Ident || isWord(typed[0], "not") {
		cond, ok := p.condition(toks, true)
		q.cond = cond
		return q, ok
	}
	if reservedMediaWord(typed[0]) {
		return q, false
	}
	q.kind = strings.ToLower(typed[0].Text)
	if len(typed) == 1 {
		return q, true
	}
	if !isWord(typed[1], "and") || len(typed) == 2 {
		return q, false
	}
	cond, ok := p.condition(typed[2:], false)
	q.cond = cond
	return q, ok
}

// condition reads not (…), or (…) joined by and, or by or where orOK.
func (p *mediaParser) condition(toks []Component, orOK bool) (*mediaCond, bool) {
	if len(toks) == 0 {
		return nil, false
	}
	if isWord(toks[0], "not") {
		if len(toks) != 2 {
			return nil, false
		}
		kid, ok := p.inParens(toks[1])
		return &mediaCond{op: mediaNot, kids: []mediaCond{kid}}, ok
	}
	first, ok := p.inParens(toks[0])
	if !ok {
		return nil, false
	}
	if len(toks) == 1 {
		return &first, true
	}
	c := &mediaCond{op: mediaAnd, kids: []mediaCond{first}}
	if isWord(toks[1], "or") && orOK {
		c.op = mediaOr
	}
	for i := 1; i < len(toks); i += 2 {
		joins := isWord(toks[i], "and") && c.op == mediaAnd || isWord(toks[i], "or") && c.op == mediaOr
		if !joins || i+1 >= len(toks) {
			return nil, false
		}
		kid, kidOK := p.inParens(toks[i+1])
		if !kidOK {
			return nil, false
		}
		c.kids = append(c.kids, kid)
	}
	return c, true
}

func (p *mediaParser) inParens(c Component) (mediaCond, bool) {
	switch {
	case c.Kind == Function:
		p.warn(c.Pos, "@media: %s() is not supported; its query never holds", c.Text)
		return mediaCond{op: mediaUnknown}, true
	case c.Kind != LParen:
		return mediaCond{}, false
	}
	inner := items(c.Args)
	if len(inner) > 0 && (inner[0].Kind == LParen || inner[0].Kind == Function ||
		isWord(inner[0], "not") && len(inner) > 1) {
		cond, ok := p.condition(inner, true)
		if !ok {
			return mediaCond{}, false
		}
		return *cond, true
	}
	return p.feature(c, inner), true
}

// feature reads (name), (name: value) or a range such as
// (400px <= width < 900px). What ui doesn't read is unknown.
func (p *mediaParser) feature(at Component, toks []Component) mediaCond {
	unknownFeature := mediaCond{op: mediaUnknown}
	isCmp := func(c Component) bool {
		return c.Kind == Delim && strings.ContainsAny(c.Text, "<>=") && len(c.Text) == 1
	}
	var c mediaCond
	var name string
	switch {
	case len(toks) == 1 && toks[0].Kind == Ident:
		name = strings.ToLower(toks[0].Text)
	case len(toks) >= 3 && toks[0].Kind == Ident && toks[1].Kind == Colon:
		name = strings.ToLower(toks[0].Text)
	case slices.ContainsFunc(toks, isCmp):
		return p.rangeFeature(at, toks, isCmp)
	default:
		p.warn(at.Pos, "@media: %s is no media feature; its query never holds", at.String())
		return unknownFeature
	}
	plain, bare, known := p.vendorFeature(name)
	if !known {
		return unknownFeature
	}
	bound := byte(0)
	spec := mediaSpecs[plain]
	if rest, ok := strings.CutPrefix(plain, "min-"); ok && spec == nil {
		bound, spec = '>', mediaSpecs[rest]
	} else if rest, ok = strings.CutPrefix(plain, "max-"); ok && spec == nil {
		bound, spec = '<', mediaSpecs[rest]
	}
	if spec == nil || bound != 0 && (spec.kind == mediaWord || len(toks) == 1) {
		p.warn(at.Pos, "@media: the %s feature is not supported; its query never holds", name)
		return unknownFeature
	}
	c.spec = spec
	if len(toks) == 1 {
		return c
	}
	value := toks[2:]
	if spec.kind == mediaWord {
		if len(value) != 1 || !isWord(value[0], spec.words...) {
			p.warn(value[0].Pos, "@media: %s takes %s; its query never holds", name, strings.Join(spec.words, ", "))
			return unknownFeature
		}
		c.word = strings.ToLower(value[0].Text)
		return c
	}
	v, ok := mediaValueOf(spec.kind, value, bare)
	if !ok {
		p.warn(value[0].Pos, "@media: %s can't take %s; its query never holds", name, join(value))
		return unknownFeature
	}
	if bound == 0 {
		bound = '='
	}
	c.bounds = []mediaBound{{cmp: bound, orEqual: true, value: v}}
	return c
}

// vendorFeature is the feature name stands for, bare when its value is a
// bare number of dppx. It is not known when a sheet read as a browser's
// names a vendor's feature Chrome doesn't answer, which is false without
// a word.
func (p *mediaParser) vendorFeature(name string) (plain string, bare, known bool) {
	v, rest := vendorOf(name)
	if v == nil || !p.browser {
		return name, false, true
	}
	if as, read := v.feature(rest); as != "" {
		return as, true, true
	} else if read {
		return name, false, true
	}
	return "", false, false
}

// rangeFeature reads width > 400px, 400px < width and
// 400px < width <= 900px.
func (p *mediaParser) rangeFeature(at Component, toks []Component, isCmp func(Component) bool) mediaCond {
	unknownFeature := mediaCond{op: mediaUnknown}
	type operator struct {
		cmp     byte
		orEqual bool
	}
	var operands [][]Component
	var ops []operator
	start := 0
	for i := 0; i < len(toks); i++ {
		if !isCmp(toks[i]) {
			continue
		}
		operands = append(operands, toks[start:i])
		op := operator{cmp: toks[i].Text[0]}
		if op.cmp != '=' && i+1 < len(toks) && toks[i+1].Kind == Delim && toks[i+1].Text == "=" {
			op.orEqual = true
			i++
		}
		ops = append(ops, op)
		start = i + 1
	}
	operands = append(operands, toks[start:])
	named := slices.IndexFunc(operands, func(o []Component) bool { return len(o) == 1 && o[0].Kind == Ident })
	bad := named < 0 || len(ops) > 2 || len(ops) == 2 && (named != 1 || ops[0].cmp != ops[1].cmp || ops[0].cmp == '=')
	if bad || len(ops) == 1 && named > 1 {
		p.warn(at.Pos, "@media: %s is no media feature; its query never holds", at.String())
		return unknownFeature
	}
	name := strings.ToLower(operands[named][0].Text)
	spec := mediaSpecs[name]
	if spec == nil || spec.kind == mediaWord {
		p.warn(at.Pos, "@media: the %s feature is not supported in a range; its query never holds", name)
		return unknownFeature
	}
	c := mediaCond{spec: spec}
	for i, op := range ops {
		side := operands[i+1]
		if i >= named {
			// The feature stands on the left: as written.
		} else {
			side = operands[i]
			switch op.cmp {
			case '<':
				op.cmp = '>'
			case '>':
				op.cmp = '<'
			}
		}
		v, ok := mediaValueOf(spec.kind, side, false)
		if !ok {
			p.warn(at.Pos, "@media: %s can't take %s; its query never holds", name, join(side))
			return unknownFeature
		}
		c.bounds = append(c.bounds, mediaBound{cmp: op.cmp, orEqual: op.orEqual || op.cmp == '=', value: v})
	}
	return c
}

// lengthUnits are the lengths a media query measures in CSS pixels; an
// em is 16 of them whatever the page's font.
//
//nolint:gochecknoglobals // lookup table
var lengthUnits = map[string]float64{"px": 1, "em": 16, "rem": 16, "in": 96, "cm": 96 / 2.54, "mm": 96 / 25.4,
	"q": 96 / 101.6, "pt": 96.0 / 72, "pc": 16}

//nolint:gochecknoglobals // lookup table
var resolutionUnits = map[string]float64{"dppx": 1, "x": 1, "dpi": 1.0 / 96, "dpcm": 2.54 / 96}

// mediaValueOf reads a feature's value; bare takes a resolution as a
// bare number of dppx.
func mediaValueOf(kind mediaValueKind, v []Component, bare bool) (mediaValue, bool) {
	if len(v) == 0 {
		return mediaValue{}, false
	}
	unit := strings.ToLower(v[0].Unit)
	switch kind {
	case mediaLength:
		switch {
		case len(v) != 1:
		case v[0].Kind == Number && v[0].Num == 0:
			return mediaValue{}, true
		case v[0].Kind != Dimension:
		case slices.Contains([]string{"vw", "vh", "vmin", "vmax"}, unit):
			return mediaValue{num: v[0].Num, unit: unit}, true
		case lengthUnits[unit] != 0:
			return mediaValue{num: v[0].Num * lengthUnits[unit]}, true
		}
	case mediaRatio:
		switch {
		case len(v) == 1 && v[0].Kind == Number && v[0].Num >= 0:
			return mediaValue{num: v[0].Num}, true
		case len(v) == 3 && v[0].Kind == Number && isSlash(v[1]) && v[2].Kind == Number && v[0].Num >= 0 && v[2].Num > 0:
			return mediaValue{num: v[0].Num / v[2].Num}, true
		}
	case mediaResolution:
		switch {
		case len(v) != 1:
		case bare && v[0].Kind == Number:
			return mediaValue{num: v[0].Num}, true
		case v[0].Kind == Dimension && resolutionUnits[unit] != 0:
			return mediaValue{num: v[0].Num * resolutionUnits[unit]}, true
		}
	case mediaInteger:
		if len(v) == 1 && v[0].Kind == Number && v[0].Int {
			return mediaValue{num: v[0].Num}, true
		}
	case mediaWord:
		// a word is no number
	}
	return mediaValue{}, false
}

// mediaOff is, for each of sheets with @media rules, the rules whose
// queries don't all hold for m; nil when no sheet has any.
func mediaOff(sheets []*Sheet, m Media) [][]bool {
	var off [][]bool
	for si, s := range sheets {
		if len(s.media) == 0 {
			continue
		}
		if off == nil {
			off = make([][]bool, len(sheets))
		}
		holds := make([]bool, len(s.media))
		for i, l := range s.media {
			holds[i] = l.matches(m)
		}
		off[si] = make([]bool, len(s.Rules))
		for ri := range s.Rules {
			for _, i := range s.Rules[ri].media {
				if !holds[i] {
					off[si][ri] = true
				}
			}
		}
	}
	return off
}

func sameOff(a, b [][]bool) bool {
	return slices.EqualFunc(a, b, func(x, y []bool) bool { return slices.Equal(x, y) })
}
