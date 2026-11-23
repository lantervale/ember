package css

import (
	"fmt"
	"slices"
	"strings"
)

func word(s string) Component { return Component{Token: Token{Kind: Ident, Text: s}} }

//nolint:gochecknoglobals // derived from the properties
var initials = func() map[string][]Component {
	m := make(map[string][]Component, len(properties))
	for name, p := range properties {
		m[name] = items(componentsOf(p.initial))
	}
	return m
}()

// initialValue is a longhand's initial value as components.
func initialValue(name string) []Component { return initials[name] }

func componentsOf(src string) []Component {
	p := parser{toks: Tokenize(src)}
	var out []Component
	for p.peek().Kind != EOF {
		out = append(out, p.component())
	}
	return out
}

// validate checks a declaration's value, without var(), against its
// property's grammar.
func validate(name string, value []Component) error {
	v := items(value)
	if globalKeyword(v) != "" {
		return nil
	}
	if _, ok := shorthands[name]; ok {
		_, err := expand(name, v)
		return err
	}
	if c, ok := checks[name]; ok {
		return c(v)
	}
	return nil
}

// expand splits a shorthand's value, given without whitespace, into its
// longhands; a longhand the value leaves out gets its initial value.
//
//nolint:funlen,cyclop,gocyclo // one case a shorthand
func expand(name string, v []Component) (map[string][]Component, error) {
	long := shorthands[name]
	out := make(map[string][]Component, len(long))
	if k := globalKeyword(v); k != "" {
		for _, l := range long {
			out[l] = []Component{word(k)}
		}
		return out, nil
	}
	var err error
	switch name {
	case "margin", "padding", kwInset, "border-width", "border-style", propBorderColor, "border-radius",
		"scroll-padding", "scroll-margin":
		err = expandBox(long, v, out)
	case shortGap, "overflow", "place-items", "place-content", "place-self", shortMarginInline, shortMarginBlock,
		shortPaddingInline, shortPaddingBlock, shortInsetInline, shortInsetBlock, "border-inline-width",
		"border-inline-style", "border-inline-color", "border-block-width", "border-block-style", "border-block-color",
		shortScrollPaddingInline, shortScrollPaddingBlock, shortScrollMarginInline, shortScrollMarginBlock,
		"overscroll-behavior":
		err = expandPair(long, v, out)
	case kwFlex:
		err = expandFlex(v, out)
	case "flex-flow":
		err = expandAnyOrder(long, v, out)
	case propPositionTry:
		err = expandPositionTry(v, out)
	case "grid-column", "grid-row":
		err = expandGridLines(long, v, out)
	case "grid-area":
		err = expandGridArea(long, v, out)
	case shortGrid:
		err = expandGrid(long, v, out)
	case "grid-template":
		err = expandGridTemplate(v, out)
	case "border-top", "border-right", "border-bottom", "border-left", "outline", "column-rule", shortStroke:
		err = expandAnyOrder(long, v, out)
	case "columns":
		err = expandColumns(v, out)
	case shortWhiteSpace:
		err = expandWhiteSpace(v, out)
	case "text-wrap":
		err = expandAnyOrder(long, v, out)
	case "text-box":
		err = expandTextBox(v, out)
	case kwContainer:
		err = expandContainer(v, out)
	case shortBackground:
		err = expandBackground(v, out)
	case propMaskPosition:
		err = expandMaskPosition(v, out)
	case shortMask:
		err = expandMask(v, out)
	case kwBorder:
		err = expandBorder(long, v, out)
	case "border-inline", "border-block":
		err = expandSides(long, v, out)
	case shortBorderImage, shortMaskBox:
		err = expandBorderImage(name, long, v, out)
	case shortTransition:
		err = expandTransition(v, out)
	case shortAnimation:
		err = expandAnimation(v, out)
	case shortDecoration:
		err = expandTextDecoration(v, out)
	case shortFont:
		err = expandFont(v, out)
	case shortVariant:
		err = expandVariant(v, out)
	case "list-style":
		err = expandListStyle(v, out)
	default:
		err = fmt.Errorf("%s can't be expanded", name)
	}
	if err != nil {
		return nil, err
	}
	for _, l := range long {
		if _, ok := out[l]; !ok {
			out[l] = initialValue(l)
		}
	}
	return out, nil
}

// expandBorder gives the four sides one side's width, style and colour,
// and the border image its initial values.
func expandBorder(long []string, v []Component, out map[string][]Component) error {
	side := shorthands["border-top"]
	one := map[string][]Component{}
	if err := expandAnyOrder(side, v, one); err != nil {
		return err
	}
	for _, l := range side {
		if one[l] == nil {
			one[l] = initialValue(l)
		}
	}
	for i, l := range long[:12] {
		out[l] = one[side[i/4]]
	}
	for _, l := range long[12:] {
		out[l] = initialValue(l)
	}
	return nil
}

// expandSides gives two sides, three longhands each, one side's width,
// style and colour.
func expandSides(long []string, v []Component, out map[string][]Component) error {
	one := map[string][]Component{}
	if err := expandAnyOrder(long[:3], v, one); err != nil {
		return err
	}
	for i, l := range long {
		if c := one[long[i%3]]; c != nil {
			out[l] = c
		}
	}
	return nil
}

// expandBox fills four sides (or corners) from one to four values.
func expandBox(long []string, v []Component, out map[string][]Component) error {
	if len(v) < 1 || len(v) > 4 {
		return fmt.Errorf("expected 1 to 4 values, found %d", len(v))
	}
	for _, c := range v {
		if isSlash(c) {
			return fmt.Errorf("elliptical corners are not supported")
		}
		if err := checks[long[0]]([]Component{c}); err != nil {
			return err
		}
	}
	pick := [4][4]int{{0, 0, 0, 0}, {0, 1, 0, 1}, {0, 1, 2, 1}, {0, 1, 2, 3}}[len(v)-1]
	for i, l := range long {
		out[l] = []Component{v[pick[i]]}
	}
	return nil
}

// expandPair fills two longhands from one or two values.
func expandPair(long []string, v []Component, out map[string][]Component) error {
	if len(v) < 1 || len(v) > 2 {
		return fmt.Errorf("expected 1 or 2 values, found %d", len(v))
	}
	for i, l := range long {
		c := v[min(i, len(v)-1)]
		if err := checks[l]([]Component{c}); err != nil {
			return err
		}
		out[l] = []Component{c}
	}
	return nil
}

// expandAnyOrder gives each value to the first longhand that takes it and
// has none yet.
func expandAnyOrder(long []string, v []Component, out map[string][]Component) error {
	if len(v) == 0 || len(v) > len(long) {
		return fmt.Errorf("expected 1 to %d values, found %d", len(long), len(v))
	}
next:
	for _, c := range v {
		for _, l := range long {
			if _, taken := out[l]; !taken && checks[l]([]Component{c}) == nil {
				out[l] = []Component{c}
				continue next
			}
		}
		return fmt.Errorf("%q fits none of %s", c.String(), strings.Join(long, ", "))
	}
	return nil
}

// expandColumns takes a column-width and a column-count in either order; auto stands for
// whichever is left.
func expandColumns(v []Component, out map[string][]Component) error {
	if len(v) < 1 || len(v) > 2 {
		return fmt.Errorf("columns takes a width and a count, found %d values", len(v))
	}
	for _, c := range v {
		one := []Component{c}
		switch {
		case isWord(c, "auto"):
		case out["column-count"] == nil && checks["column-count"](one) == nil:
			out["column-count"] = one
		case out["column-width"] == nil && checks["column-width"](one) == nil:
			out["column-width"] = one
		default:
			return fmt.Errorf("columns takes one width and one count, found %q", c.String())
		}
	}
	return nil
}

func expandFlex(v []Component, out map[string][]Component) error {
	num := func(s string) Component {
		return Component{Token: Token{Kind: Number, Num: parseNum(s), Int: true, Text: s}}
	}
	switch {
	case len(v) == 1 && isWord(v[0], kwNone):
		out["flex-grow"], out["flex-shrink"] = []Component{num("0")}, []Component{num("0")}
		out["flex-basis"] = []Component{word("auto")}
		return nil
	case len(v) == 1 && isWord(v[0], "auto"):
		out["flex-grow"], out["flex-shrink"] = []Component{num("1")}, []Component{num("1")}
		out["flex-basis"] = []Component{word("auto")}
		return nil
	case len(v) < 1 || len(v) > 3:
		return fmt.Errorf("expected 1 to 3 values, found %d", len(v))
	}
	var factors []Component
	var basis []Component
	for _, c := range v {
		if c.Kind == Number && len(factors) < 2 && (len(factors) == 0 || basis == nil) {
			if c.Num < 0 {
				return fmt.Errorf("flex factors can't be negative")
			}
			factors = append(factors, c)
			continue
		}
		if basis != nil || checks["flex-basis"]([]Component{c}) != nil {
			return fmt.Errorf("bad flex value %q", c.String())
		}
		basis = []Component{c}
	}
	out["flex-grow"] = []Component{num("1")}
	out["flex-shrink"] = []Component{num("1")}
	out["flex-basis"] = []Component{{Token: Token{Kind: Percentage, Text: "0"}}}
	if len(factors) > 0 {
		out["flex-grow"] = factors[:1]
	}
	if len(factors) > 1 {
		out["flex-shrink"] = factors[1:2]
	}
	if basis != nil {
		out["flex-basis"] = basis
	}
	return nil
}

func parseNum(s string) float64 {
	t := Tokenize(s)[0]
	return t.Num
}

func expandGridLines(long []string, v []Component, out map[string][]Component) error {
	parts := splitOn(v, isSlash)
	if len(parts) > 2 {
		return fmt.Errorf("expected start / end")
	}
	for i, p := range parts {
		if err := checkGridLine(p); err != nil {
			return err
		}
		out[long[i]] = p
	}
	if len(parts) == 1 {
		out[long[1]] = []Component{word("auto")}
		if lineName(parts[0]) {
			out[long[1]] = parts[0]
		}
	}
	return nil
}

// expandImageSlice reads a border image's slice from v[i] into long[1],
// then its width and outset after slashes into long[2] and long[3], and
// returns where it stopped.
func expandImageSlice(long []string, v []Component, i int, out map[string][]Component) (int, error) {
	j := i
	for j < len(v) && (v[j].Kind == Number || v[j].Kind == Percentage || isWord(v[j], "fill")) {
		j++
	}
	if err := checkImageSlice(v[i:j]); err != nil {
		return i, err
	}
	out[long[1]] = v[i:j]
	i = j
	for k, l := range long[2:4] {
		if i >= len(v) || !isSlash(v[i]) {
			break
		}
		i++
		j = i
		for j < len(v) && !isSlash(v[j]) && checks[l]([]Component{v[j]}) == nil {
			j++
		}
		if j > i {
			out[l] = v[i:j]
		} else if k == 1 || j >= len(v) || !isSlash(v[j]) {
			return i, fmt.Errorf("expected %s after '/'", [...]string{"a width", "an outset"}[k])
		}
		i = j
	}
	return i, nil
}

func expandBorderImage(name string, long []string, v []Component, out map[string][]Component) error {
	source, slice, repeat := long[0], long[1], long[4]
	i := 0
	for i < len(v) {
		c := v[i]
		switch {
		case isWord(c, kwNone) || urlTerm(c) == nil:
			if out[source] != nil {
				return fmt.Errorf("%s has two sources", name)
			}
			out[source] = []Component{c}
			i++
		case isWord(c, "stretch", "repeat", "round", "space"):
			if out[repeat] != nil {
				return fmt.Errorf("%s has two repeats", name)
			}
			j := i + 1
			if j < len(v) && isWord(v[j], "stretch", "repeat", "round", "space") {
				j++
			}
			out[repeat] = v[i:j]
			i = j
		case c.Kind == Number || c.Kind == Percentage || isWord(c, "fill"):
			if out[slice] != nil {
				return fmt.Errorf("%s has two slices", name)
			}
			var err error
			if i, err = expandImageSlice(long, v, i, out); err != nil {
				return err
			}
		default:
			return fmt.Errorf("bad %s part %q", name, c.String())
		}
	}
	if name == shortMaskBox && out[slice] != nil && !slices.ContainsFunc(out[slice], func(c Component) bool {
		return isWord(c, "fill")
	}) {
		// Chrome's shorthand always fills the middle.
		out[slice] = append(slices.Clone(out[slice]), word("fill"))
	}
	return nil
}

func expandTransition(v []Component, out map[string][]Component) error {
	long := shorthands["transition"]
	lists := make(map[string][]Component, len(long))
	for n, part := range splitOn(v, func(c Component) bool { return c.Kind == Comma }) {
		if len(part) == 0 {
			return fmt.Errorf("empty transition")
		}
		one := map[string]Component{}
		for _, c := range part {
			switch {
			case timingFunction(c) == nil && one["transition-timing-function"].Kind == EOF:
				one["transition-timing-function"] = c
			case c.Kind != Ident && timeTerm(false)(c) == nil:
				switch {
				case one["transition-duration"].Kind == EOF:
					if timeTerm(true)(c) != nil {
						return fmt.Errorf("a duration can't be negative")
					}
					one["transition-duration"] = c
				case one["transition-delay"].Kind == EOF:
					one["transition-delay"] = c
				default:
					return fmt.Errorf("a transition has at most two times")
				}
			case isWord(c, kwNormal, kwAllowDiscrete) && one[propTransitionBehavior].Kind == EOF:
				one[propTransitionBehavior] = c
			case c.Kind == Ident && one["transition-property"].Kind == EOF:
				if !isWord(c, kwAll, kwNone) {
					if err := transitionTarget(c); err != nil {
						return err
					}
				}
				one["transition-property"] = c
			default:
				return fmt.Errorf("bad transition part %q", c.String())
			}
		}
		for _, l := range long {
			c, ok := one[l]
			if !ok {
				c = initialValue(l)[0]
			}
			if n > 0 {
				lists[l] = append(lists[l], Component{Token: Token{Kind: Comma}})
			}
			lists[l] = append(lists[l], c)
		}
	}
	for l, c := range lists {
		out[l] = c
	}
	return nil
}

// expandListStyle splits list-style into its position, its type and its
// image, in any order. A none goes to the one of the two not given, to
// both when neither is.
func expandListStyle(v []Component, out map[string][]Component) error {
	var typ, image, position []Component
	nones := 0
	for _, c := range v {
		switch {
		case isWord(c, kwNone):
			nones++
		case isWord(c, "outside", "inside") && position == nil:
			position = []Component{c}
		case checkListImage([]Component{c}) == nil && image == nil:
			image = []Component{c}
		case checkListStyle([]Component{c}) == nil && typ == nil:
			typ = []Component{c}
		default:
			return fmt.Errorf("list-style takes a list-style-position, a list-style-image and a list-style-type, found %q",
				c.String())
		}
	}
	none := []Component{{Token: Token{Kind: Ident, Text: kwNone}}}
	switch {
	case nones > 2, nones == 2 && (typ != nil || image != nil), nones == 1 && typ != nil && image != nil:
		return fmt.Errorf("list-style takes none for its type and its image at most")
	case nones == 2, nones == 1 && typ == nil && image == nil:
		typ, image = none, none
	case nones == 1 && typ == nil:
		typ = none
	case nones == 1:
		image = none
	}
	if typ == nil {
		typ = []Component{{Token: Token{Kind: Ident, Text: kwDisc}}}
	}
	if image == nil {
		image = none
	}
	if position == nil {
		position = []Component{{Token: Token{Kind: Ident, Text: kwOutside}}}
	}
	out["list-style-type"], out["list-style-image"], out["list-style-position"] = typ, image, position
	return nil
}

// expandAnimation splits the animation shorthand item by item: the
// first time is the duration and the second the delay, and an
// identifier that is a keyword of a longhand not yet given is that
// longhand's, as CSS has it, so the name is the one left over.
func expandAnimation(v []Component, out map[string][]Component) error {
	long := shorthands[shortAnimation]
	lists := make(map[string][]Component, len(long))
	for n, part := range splitOn(v, func(c Component) bool { return c.Kind == Comma }) {
		if len(part) == 0 {
			return fmt.Errorf("empty animation")
		}
		one := map[string]Component{}
		for _, c := range part {
			name, err := animationPart(c, func(l string) bool { return one[l].Kind == EOF })
			if err != nil {
				return err
			}
			one[name] = c
		}
		for _, l := range long {
			c, ok := one[l]
			if !ok {
				c = initialValue(l)[0]
			}
			if n > 0 {
				lists[l] = append(lists[l], Component{Token: Token{Kind: Comma}})
			}
			lists[l] = append(lists[l], c)
		}
	}
	for l, c := range lists {
		out[l] = c
	}
	return nil
}

// animationPart is the longhand c of an animation item stands for, given
// which longhands are still free.
func animationPart(c Component, free func(string) bool) (string, error) {
	switch {
	case c.Kind != Ident && timeTerm(false)(c) == nil:
		switch {
		case free(propAnimationDuration):
			if timeTerm(true)(c) != nil {
				return "", fmt.Errorf("a duration can't be negative")
			}
			return propAnimationDuration, nil
		case free(propAnimationDelay):
			return propAnimationDelay, nil
		}
		return "", fmt.Errorf("an animation has at most two times")
	case c.Kind == Number && free(propAnimationCount):
		return propAnimationCount, iterationCount(c)
	case timingFunction(c) == nil && free(propAnimationTiming):
		return propAnimationTiming, nil
	}
	for _, k := range []struct {
		name  string
		words []string
	}{
		{propAnimationCount, []string{kwInfinite}},
		{propAnimationDir, animationDirections},
		{propAnimationFill, animationFills},
		{propAnimationPlay, []string{kwRunning, kwPaused}},
	} {
		if isWord(c, k.words...) && free(k.name) {
			return k.name, nil
		}
	}
	if free(propAnimationName) && (isWord(c, kwNone) || keyframesName(c) == nil) {
		return propAnimationName, nil
	}
	return "", fmt.Errorf("bad animation part %q", c.String())
}

// expandBackground splits the background shorthand: a colour, an image, a
// repeat of one or two words, an attachment, one box for the origin and
// the clip or two, and a position with a size after a slash, in any
// order, for each layer; the last layer alone may have the colour. Each
// other longhand is the list of the layers' values, those a layer leaves
// out initial.
func expandBackground(v []Component, out map[string][]Component) error {
	layers := splitOn(v, func(c Component) bool { return c.Kind == Comma })
	for i, l := range layers {
		layer := map[string][]Component{}
		if len(l) == 0 {
			return fmt.Errorf("background: an empty layer")
		}
		if err := backgroundLayer(l, layer); err != nil {
			return err
		}
		if c := layer["background-color"]; c != nil {
			if i != len(layers)-1 {
				return fmt.Errorf("background: only the last layer takes a colour")
			}
			out["background-color"] = c
		}
		for _, name := range shorthands[shortBackground][1:] {
			value, ok := layer[name]
			if !ok {
				value = initialValue(name)
			}
			if i > 0 {
				out[name] = append(out[name], Component{Token: Token{Kind: Comma}})
			}
			out[name] = append(out[name], value...)
		}
	}
	return nil
}

func backgroundLayer(v []Component, out map[string][]Component) error {
	var pos, boxes []Component
	posEnd := -1
	for i := 0; i < len(v); i++ {
		c := v[i]
		one := []Component{c}
		switch {
		case isSlash(c):
			if posEnd != i || out["background-size"] != nil {
				return fmt.Errorf("background: a size follows a position, as in center / cover")
			}
			n := 0
			for n < 2 && i+1+n < len(v) && checkBackgroundSize(v[i+1:i+2+n]) == nil {
				n++
			}
			if n == 0 {
				return fmt.Errorf("background: expected a size after /")
			}
			out["background-size"] = v[i+1 : i+1+n]
			i += n
		case out["background-color"] == nil && colorTerm(c) == nil:
			out["background-color"] = one
		case out["background-image"] == nil && (isWord(c, kwNone) || imageTerm(c) == nil):
			out["background-image"] = one
		case out["background-repeat"] == nil && checkTiling(one) == nil:
			out["background-repeat"] = one
			if i+1 < len(v) && checkTiling(v[i:i+2]) == nil {
				out["background-repeat"] = v[i : i+2]
				i++
			}
		case out["background-attachment"] == nil && checks["background-attachment"](one) == nil:
			out["background-attachment"] = one
		case len(boxes) < 3 && (backgroundBox(c) == nil || isWord(c, "text")):
			boxes = append(boxes, c)
		case out["background-size"] != nil && posEnd >= 0:
			return fmt.Errorf("background: %s after the size", c.String())
		case posEnd >= 0 && posEnd != i:
			return fmt.Errorf("background: %s is not part of a background", c.String())
		default:
			pos, posEnd = append(pos, c), i+1
		}
	}
	if len(pos) > 0 {
		if err := checkPosition(pos); err != nil {
			return err
		}
		out["background-position"] = pos
	}
	return backgroundBoxes(boxes, out)
}

// backgroundBoxes are a background layer's boxes: one the origin and the
// clip, two the origin then the clip; text the clip alone, beside one box
// the origin.
func backgroundBoxes(all []Component, out map[string][]Component) error {
	var boxes, text []Component
	for i, c := range all {
		switch {
		case !isWord(c, "text"):
			boxes = append(boxes, c)
		case text != nil:
			return fmt.Errorf("background: text twice")
		default:
			text = all[i : i+1]
		}
	}
	switch {
	case len(boxes) > 2:
		return fmt.Errorf("background: three boxes")
	case text != nil && len(boxes) > 1:
		return fmt.Errorf("background: text after two boxes")
	case text != nil:
		out["background-clip"] = text
		if len(boxes) > 0 {
			out["background-origin"] = boxes
		}
	case len(boxes) > 0:
		out["background-origin"], out["background-clip"] = boxes[:1], boxes[len(boxes)-1:]
	}
	return nil
}

// expandMask splits the mask shorthand: an image, a position with a size
// after a slash, a repeat of one or two words, one box for the origin and
// the clip or two, the second of which may be no-clip, a compositing word
// and a mode, in any order, for each layer. Each longhand is the list of
// the layers' values, those a layer leaves out initial.
func expandMask(v []Component, out map[string][]Component) error {
	for i, l := range splitOn(v, func(c Component) bool { return c.Kind == Comma }) {
		layer := map[string][]Component{}
		if len(l) == 0 {
			return fmt.Errorf("mask: an empty layer")
		}
		if err := maskLayer(l, layer); err != nil {
			return err
		}
		for _, name := range shorthands[shortMask] {
			value, ok := layer[name]
			if !ok {
				value = initialValue(name)
			}
			if i > 0 {
				out[name] = append(out[name], Component{Token: Token{Kind: Comma}})
			}
			out[name] = append(out[name], value...)
		}
	}
	return nil
}

func maskLayer(v []Component, out map[string][]Component) error {
	var pos, boxes []Component
	posEnd := -1
	for i := 0; i < len(v); i++ {
		c := v[i]
		one := []Component{c}
		switch {
		case isSlash(c):
			if posEnd != i || out["mask-size"] != nil {
				return fmt.Errorf("mask: a size follows a position, as in center / contain")
			}
			n := 0
			for n < 2 && i+1+n < len(v) && checkBackgroundSize(v[i+1:i+2+n]) == nil {
				n++
			}
			if n == 0 {
				return fmt.Errorf("mask: expected a size after /")
			}
			out["mask-size"] = v[i+1 : i+1+n]
			i += n
		case out["mask-image"] == nil && (isWord(c, kwNone) || imageTerm(c) == nil):
			out["mask-image"] = one
		case out["mask-repeat"] == nil && checkTiling(one) == nil:
			out["mask-repeat"] = one
			if i+1 < len(v) && checkTiling(v[i:i+2]) == nil {
				out["mask-repeat"] = v[i : i+2]
				i++
			}
		case out["mask-composite"] == nil && checks["mask-composite"](one) == nil:
			out["mask-composite"] = one
		case out["mask-mode"] == nil && checks["mask-mode"](one) == nil:
			out["mask-mode"] = one
		case len(boxes) < 2 && maskBox(c) == nil, len(boxes) < 2 && isWord(c, "no-clip"):
			boxes = append(boxes, c)
		case out["mask-size"] != nil && posEnd >= 0:
			return fmt.Errorf("mask: %s after the size", c.String())
		case posEnd >= 0 && posEnd != i:
			return fmt.Errorf("mask: %s is not part of a mask", c.String())
		default:
			pos, posEnd = append(pos, c), i+1
		}
	}
	if len(pos) > 0 {
		if err := checkMaskPosition(pos); err != nil {
			return err
		}
		out[propMaskPositionX], out[propMaskPositionY] = maskAxes(pos)
	}
	return maskOriginAndClip(boxes, out)
}

// maskOriginAndClip sets a mask layer's origin and clip from the boxes it names:
// one is both, of two the first is the origin, and no-clip is a clip.
func maskOriginAndClip(boxes []Component, out map[string][]Component) error {
	switch {
	case len(boxes) == 0:
	case isWord(boxes[0], "no-clip"):
		if len(boxes) > 1 {
			return fmt.Errorf("mask: no-clip is the clip alone")
		}
		out["mask-clip"] = boxes
	default:
		out["mask-origin"], out["mask-clip"] = boxes[:1], boxes[len(boxes)-1:]
	}
	return nil
}

// expandFont splits the font shorthand: style, small-caps, weight and stretch in any order, a
// size, a line height after a slash, then the families. What it leaves
// out is reset, as CSS says.
func expandFont(v []Component, out map[string][]Component) error {
	i := 0
	for ; i < len(v); i++ {
		one := []Component{v[i]}
		switch {
		case isWord(v[i], "normal"):
		case out[propCaps] == nil && isWord(v[i], "small-caps"):
			out[propCaps] = one
		case isWord(v[i], "small-caps"):
			return fmt.Errorf("font: %s is not supported", v[i].String())
		case out["font-stretch"] == nil && checks["font-stretch-word"](one) == nil:
			out["font-stretch"] = one
		case out["font-style"] == nil && checks["font-style"](one) == nil:
			out["font-style"] = one
		case out["font-weight"] == nil && checks["font-weight"](one) == nil:
			out["font-weight"] = one
		default:
			goto size
		}
	}
size:
	if i >= len(v) || checks["font-size"]([]Component{v[i]}) != nil {
		if len(v) == 1 && v[0].Kind == Ident {
			return fmt.Errorf("font takes a size and a family; system fonts such as %s are not supported", v[0].Text)
		}
		return fmt.Errorf("font takes a size then a family, such as 12px/1.5 Georgia")
	}
	out["font-size"] = []Component{v[i]}
	i++
	if i < len(v) && isSlash(v[i]) {
		if i+1 >= len(v) || checks["line-height"]([]Component{v[i+1]}) != nil {
			return fmt.Errorf("font: expected a line height after /")
		}
		out["line-height"] = []Component{v[i+1]}
		i += 2
	}
	if i >= len(v) {
		return fmt.Errorf("font takes a family after its size")
	}
	if err := checkFontFamily(v[i:]); err != nil {
		return err
	}
	out["font-family"] = v[i:]
	return nil
}

// expandContainer splits container: a name list, then a type after a
// slash.
func expandContainer(v []Component, out map[string][]Component) error {
	parts := splitOn(v, isSlash)
	if len(parts) > 2 || len(parts[0]) == 0 {
		return fmt.Errorf("container takes names, then a type after a slash")
	}
	if err := checkContainerName(parts[0]); err != nil {
		return err
	}
	out["container-name"] = parts[0]
	if len(parts) == 2 {
		if err := checkContainerType(parts[1]); err != nil {
			return err
		}
		out["container-type"] = parts[1]
	}
	return nil
}
