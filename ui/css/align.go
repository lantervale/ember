package css

import (
	"errors"
	"fmt"
	"strings"

	"github.com/lantervale/ember/ui/internal/style"
)

// alignGrammar is what one of the alignment properties takes, as Chrome reads it: words
// alone, positions an overflow position (safe or unsafe) may lead, and a baseline, first or
// last where last is taken.
type alignGrammar struct {
	alone, positions []string
	baseline, last   bool
}

//nolint:gochecknoglobals // the grammars
var (
	contentPositions = []string{kwCenter, kwStart, kwEnd, "flex-start", "flex-end"}
	selfPositions    = append([]string{kwSelfStart, kwSelfEnd}, contentPositions...)
	distributions    = []string{"space-between", "space-around", "space-evenly", kwStretch}
	alignItems       = alignGrammar{alone: []string{kwNormal, kwStretch}, positions: selfPositions, baseline: true,
		last: true}
	alignSelf = alignGrammar{alone: []string{kwAuto, kwNormal, kwStretch},
		positions: append([]string{kwAnchorCenter}, selfPositions...), baseline: true, last: true}
	justifyItems = alignGrammar{alone: []string{kwNormal, kwStretch, kwLegacy},
		positions: append([]string{kwLeft, kwRight}, selfPositions...), baseline: true, last: true}
	justifySelf = alignGrammar{alone: []string{kwAuto, kwNormal, kwStretch},
		positions: append([]string{kwLeft, kwRight, kwAnchorCenter}, selfPositions...), baseline: true, last: true}
	alignContent = alignGrammar{alone: append([]string{kwNormal}, distributions...), positions: contentPositions,
		baseline: true}
	justifyContent = alignGrammar{alone: append([]string{kwNormal}, distributions...),
		positions: append([]string{kwLeft, kwRight}, contentPositions...)}
)

const (
	kwAnchorCenter = "anchor-center"
	kwLegacy       = "legacy"
	kwSelfStart    = "self-start"
	kwSelfEnd      = "self-end"
)

// check takes a whole value of g.
func (g alignGrammar) check(v []Component) error {
	switch {
	case len(v) == 1 && isWord(v[0], g.alone...):
		return nil
	case g.baseline && len(v) == 1 && isWord(v[0], kwBaseline):
		return nil
	case g.baseline && len(v) == 2 && isWord(v[1], kwBaseline) &&
		(isWord(v[0], "first") || g.last && isWord(v[0], "last")):
		return nil
	case len(v) == 2 && isWord(v[0], "safe", "unsafe"):
		v = v[1:]
	}
	if len(v) == 1 && isWord(v[0], g.positions...) {
		return nil
	}
	if len(v) == 0 {
		return errors.New("expected an alignment")
	}
	return fmt.Errorf("not an alignment: %q", spelled(v))
}

// alignOf is the alignment a value of a checked grammar computes to: first baseline is
// baseline, and anchor-center drops its overflow position.
func alignOf(v []Component) style.Align {
	if len(v) == 2 && isWord(v[1], "baseline") {
		if isWord(v[0], "last") {
			return style.AlignLastBaseline
		}
		return style.AlignBaseline
	}
	var overflow style.Align
	if len(v) == 2 {
		if isWord(v[0], "safe") {
			overflow = style.AlignSafe
		} else {
			overflow = style.AlignUnsafe
		}
		v = v[1:]
	}
	k := alignWords4[strings.ToLower(v[0].Text)]
	if k == style.AlignAnchorCenter {
		return k
	}
	return k | overflow
}

func alignField(get func(*style.Computed) *style.Align) prop {
	return field(get, func(_ *ctx, v []Component) style.Align { return alignOf(v) }, flips[style.Align]())
}

// expandPlace splits a place-* value between its two longhands, as Chrome does: the first
// takes the longest start of the value it reads, the second the rest, or the same as the
// first when nothing is left (start for a justify-content the first's baseline can't be).
func expandPlace(long []string, v []Component, out map[string][]Component) error {
	for i := len(v); i > 0; i-- {
		first, rest := v[:i], v[i:]
		if checks[long[0]](first) != nil {
			continue
		}
		if len(rest) == 0 {
			rest = first
			if checks[long[1]](rest) != nil {
				rest = []Component{word("start")}
			}
		}
		if checks[long[1]](rest) != nil {
			continue
		}
		out[long[0]], out[long[1]] = first, rest
		return nil
	}
	return fmt.Errorf("not a %s value: %q", strings.Join(long, " and "), spelled(v))
}

func spelled(v []Component) string {
	parts := make([]string, len(v))
	for i, c := range v {
		parts[i] = c.String()
	}
	return strings.Join(parts, " ")
}
