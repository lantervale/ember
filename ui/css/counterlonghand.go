package css

import (
	"strings"

	"github.com/lantervale/ember/ui/internal/style"
)

// internalSystems are the systems of the complex predefined counter
// styles, which CSS Counter Styles defines by their algorithm: only the
// predefined sheet names them. Ethiopic's has no table.
//
//nolint:gochecknoglobals // lookup table
var internalSystems = map[string]*style.Longhand{
	"-ember-japanese-informal":     longhandTable(style.LonghandJapanese, true, "〇一二三四五六七八九", "十百千", ""),
	"-ember-japanese-formal":       longhandTable(style.LonghandJapanese, false, "零壱弐参四伍六七八九", "拾百阡", ""),
	"-ember-korean-hangul-formal":  longhandTable(style.LonghandKorean, false, "영일이삼사오육칠팔구", "십백천", "만억조"),
	"-ember-korean-hanja-informal": longhandTable(style.LonghandKorean, true, "零一二三四五六七八九", "十百千", "萬億兆"),
	"-ember-korean-hanja-formal":   longhandTable(style.LonghandKorean, false, "零壹貳參四五六七八九", "拾百仟", "萬億兆"),
	"-ember-simp-chinese-informal": longhandTable(style.LonghandChinese, true, "零一二三四五六七八九", "十百千", "万亿"),
	"-ember-simp-chinese-formal":   longhandTable(style.LonghandChinese, false, "零壹贰叁肆伍陆柒捌玖", "拾佰仟", "万亿"),
	"-ember-trad-chinese-informal": longhandTable(style.LonghandChinese, true, "零一二三四五六七八九", "十百千", "萬億兆"),
	"-ember-trad-chinese-formal":   longhandTable(style.LonghandChinese, false, "零壹貳參肆伍陸柒捌玖", "拾佰仟", "萬億兆"),
	"-ember-ethiopic-numeric":      nil,
}

// longhandTable is a longhand style of lang: its ten digits, its three
// markers and the markers of its groups, a simplified Chinese 10¹²
// written 万亿.
func longhandTable(lang style.LonghandLang, informal bool, digits, markers, groups string) *style.Longhand {
	l := &style.Longhand{Lang: lang, Informal: informal}
	for i, r := range []rune(digits) {
		l.Digits[i] = string(r)
	}
	for i, r := range []rune(markers) {
		l.Markers[i] = string(r)
	}
	for i, r := range []rune(groups) {
		l.Groups[i] = string(r)
	}
	if groups == "万亿" {
		l.Groups[2] = groups
	}
	return l
}

// internalSystem is the system of a complex predefined style v names,
// false when it names none.
func internalSystem(v []Component) (style.CounterSystem, *style.Longhand, bool) {
	if len(v) != 1 || v[0].Kind != Ident {
		return 0, nil, false
	}
	l, ok := internalSystems[strings.ToLower(v[0].Text)]
	if !ok {
		return 0, nil, false
	}
	if l == nil {
		return style.CounterEthiopic, nil, true
	}
	return style.CounterLonghand, l, true
}
