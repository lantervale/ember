package css

import (
	"strings"

	"github.com/lantervale/ember/ui/internal/style"
)

// langChanged is a lang attribute's key in dom.Pipeline.Changed.
const langChanged = "[lang"

// strictness is the rules Chrome breaks the element's lines by: those
// of line-break's value in its language, as ICU's locales have them.
// Chinese, Japanese and Korean take the CJK rules of normal and loose,
// Chinese those of normal for auto too, and Korean has no strict rules.
func (x *ctx) strictness() style.Strictness {
	lang := ""
	if x.node != nil {
		lang, _, _ = strings.Cut(strings.ToLower(language(x.node)), "-")
	}
	cjk := lang == "ja" || lang == "zh" || lang == "ko"
	switch x.c.LineBreak {
	case style.LineBreakAuto:
		if lang == "zh" {
			return style.StrictnessNormalCJK
		}
	case style.LineBreakNormal:
		if cjk {
			return style.StrictnessNormalCJK
		}
	case style.LineBreakStrict:
		if lang != "ko" {
			return style.StrictnessStrict
		}
	case style.LineBreakLoose:
		if cjk {
			return style.StrictnessLooseCJK
		}
		return style.StrictnessLoose
	case style.LineBreakAnywhere:
	}
	return style.StrictnessNormal
}
