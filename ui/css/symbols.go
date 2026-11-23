package css

import (
	"fmt"
	"strings"
	"sync"

	"github.com/lantervale/ember/ui/internal/style"
)

// anonymousStyles are the counter styles symbols() made, by the name
// list-style-type and counter() keep for them: the function as
// getComputedStyle writes it.
//
//nolint:gochecknoglobals // shared by every Document: a style is its text
var anonymousStyles sync.Map

const kwSymbols = "symbols"

// isSymbolsFunction reports whether c is symbols().
func isSymbolsFunction(c Component) bool { return c.Kind == Function && strings.EqualFold(c.Text, kwSymbols) }

// checkCounterStyleRef checks a counter style where a value names one: a
// name or symbols().
func checkCounterStyleRef(v []Component) error {
	if len(v) == 1 && isSymbolsFunction(v[0]) {
		_, _, err := symbolsOf(v[0])
		return err
	}
	return checkStyleName(v)
}

// counterStyleRef is the name a value keeps for the counter style c
// names, or makes with symbols().
func counterStyleRef(c Component) string {
	if !isSymbolsFunction(c) {
		return styleName(c.Text)
	}
	cs, name, _ := symbolsOf(c)
	anonymousStyles.LoadOrStore(name, cs)
	return name
}

// symbolsOf reads symbols(): a system, symbolic when left out, and the
// strings it writes with, as an anonymous counter style has them: no
// prefix, a space after, decimal for what it can't write. Chrome takes
// no picture for a symbol.
func symbolsOf(c Component) (*style.CounterStyle, string, error) {
	args := items(c.Args)
	cs := &style.CounterStyle{System: style.CounterSymbolic, First: 1, Negative: [2]string{"-"}, Suffix: " ",
		Fallback: kwDecimal}
	var b strings.Builder
	b.WriteString("symbols(")
	if len(args) > 0 && args[0].Kind == Ident {
		word := strings.ToLower(args[0].Text)
		switch word {
		case "cyclic", "numeric", "alphabetic", "symbolic", kwFixed:
			cs.System, _ = systemOf(args[:1])
		default:
			return nil, "", fmt.Errorf("symbols() takes cyclic, numeric, alphabetic, symbolic or fixed, found %q", word)
		}
		if cs.System != style.CounterSymbolic {
			b.WriteString(word + " ")
		}
		args = args[1:]
	}
	if len(args) == 0 {
		return nil, "", fmt.Errorf("symbols() takes at least one string")
	}
	for i, a := range args {
		if a.Kind != String {
			return nil, "", fmt.Errorf("symbols() takes strings, found %q; ui, as Chrome, takes no picture", a.String())
		}
		if i > 0 {
			b.WriteString(" ")
		}
		b.WriteString(a.String())
		cs.Symbols = append(cs.Symbols, a.Text)
	}
	if !enoughSymbols(cs) {
		return nil, "", fmt.Errorf("symbols(): an alphabetic or a numeric system takes at least two strings")
	}
	b.WriteString(")")
	cs.Name = b.String()
	return cs, cs.Name, nil
}

// symbolsStyle is the style symbols() made under name, nil for any other
// name.
func symbolsStyle(name string) *style.CounterStyle {
	if !strings.HasPrefix(name, kwSymbols+"(") {
		return nil
	}
	if cs, ok := anonymousStyles.Load(name); ok {
		return cs.(*style.CounterStyle) //nolint:forcetypeassert // the map holds nothing else
	}
	return nil
}
