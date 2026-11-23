package css

import (
	"fmt"
	"strings"

	"github.com/lantervale/ember/ui/internal/style"
)

// legacyComposite is the prefix the compositing words of
// -webkit-mask-composite take once read as mask-composite's, which
// refuses them written plain, as Chrome does.
const legacyComposite = "-ember-"

// maskComposites are mask-composite's words: its own, and those
// -webkit-mask-composite gives it.
//
//nolint:gochecknoglobals // lookup table
var maskComposites = func() map[string]style.MaskComposite {
	m := map[string]style.MaskComposite{}
	for i, w := range style.MaskCompositeWords {
		if c := style.MaskComposite(i); c <= style.CompositeExclude {
			m[w] = c
		} else {
			m[legacyComposite+w] = c
		}
	}
	return m
}()

// legacyCompositeWords are -webkit-mask-composite's words, read as their
// prefixed ones.
//
//nolint:gochecknoglobals // lookup table
var legacyCompositeWords = func() map[string]string {
	m := map[string]string{}
	for _, w := range style.MaskCompositeWords[style.CompositeSourceOver:] {
		m[w] = legacyComposite + w
	}
	return m
}()

func checkMaskComposite(c Component) error {
	if _, ok := maskComposites[strings.ToLower(c.Text)]; c.Kind == Ident && ok {
		return nil
	}
	return fmt.Errorf("expected add, subtract, intersect or exclude, found %q", c.String())
}

// webkitMask reads the value of -webkit-mask, -webkit-mask-composite or
// -webkit-mask-position, by rest, as the plain property's: the compositing
// operators of -webkit-mask-composite as mask-composite's words for them,
// which the shorthand refuses as Chrome's does; a position of three
// values, which
// Chrome's own longhands take and the plain property refuses, is written
// as the four it stands for, and -webkit-mask refuses no-clip, as Chrome
// does.
func webkitMask(rest string, value []Component) ([]Component, error) {
	if rest == "mask-composite" {
		for i := range value {
			if c := &value[i]; c.Kind == Ident {
				if as, ok := legacyCompositeWords[strings.ToLower(c.Text)]; ok {
					c.Text = as
				}
			}
		}
	}
	if rest != "mask" && rest != "mask-position" {
		return value, nil
	}
	var out []Component
	for i, layer := range splitOn(items(value), func(c Component) bool { return c.Kind == Comma }) {
		if i > 0 {
			out = append(out, Component{Token: Token{Kind: Comma, Text: ","}})
		}
		if rest == "mask" {
			for _, c := range layer {
				if isWord(c, "no-clip") {
					return nil, fmt.Errorf("-webkit-mask takes no no-clip")
				}
			}
		}
		start, end := positionRun(layer, rest == "mask-position")
		if end-start == 3 {
			four := fourValues(layer[start:end])
			layer = append(append(append([]Component{}, layer[:start]...), four...), layer[end:]...)
		}
		for k, c := range layer {
			if k > 0 {
				out = append(out, Component{Token: Token{Kind: Whitespace, Text: " "}})
			}
			out = append(out, c)
		}
	}
	return out, nil
}

// positionRun is where a mask layer's position stands in it: the whole
// layer for mask-position's, the first run of edges and lengths before a
// slash for mask's.
func positionRun(layer []Component, whole bool) (int, int) {
	if whole {
		return 0, len(layer)
	}
	start := -1
	for i, c := range layer {
		positional := isEdge(c) || c.Kind == Dimension || c.Kind == Percentage || c.Kind == Number && c.Num == 0
		switch {
		case positional && start < 0:
			start = i
		case !positional && start >= 0:
			return start, i
		}
		if isSlash(c) {
			break
		}
	}
	if start < 0 {
		return 0, 0
	}
	return start, len(layer)
}

// fourValues writes the position of three values v as four: the edge
// without an offset at 0% from itself, a center at 50% from its axis's
// start.
func fourValues(v []Component) []Component {
	edge := func(c, other Component) []Component {
		if !isWord(c, kwCenter) {
			return []Component{c, zeroPct}
		}
		start := kwLeft
		if isXEdge(other) {
			start = kwTop
		}
		return []Component{{Token: Token{Kind: Ident, Text: start}}, fifty}
	}
	if isEdge(v[1]) {
		return append(edge(v[0], v[1]), v[1], v[2])
	}
	return append([]Component{v[0], v[1]}, edge(v[2], v[0])...)
}
