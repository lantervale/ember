//go:build chrome

package ui_test

import "testing"

func TestArabicJoinsAndLigaturesCrossAnEdgeRightToLeftAsInChrome(t *testing.T) {
	matchWhole(t,
		"rtl,arabic;ب|c:ب|ب", "rtl,arabic;ل|c:ا", "rtl,arabic;c:ل|ا", "rtl,arabic;بل|c:ا|ب", "rtl,arabic;يل|c:ا",
		"rtl,arabic;ب|c:م", "rtl,arabic;ب|c:مم", "rtl,arabic;ب|c:َ|ب", "rtl,arabic;ل|c:اَ", "rtl,arabic;سس|c:س",
		"rtl,arabic;ل|pad:ا", "rtl,arabic;ل|marg:ا", "rtl,arabic;ل|end:ا", "rtl,arabic;ل|sub:ا",
		"rtl,arabic;لا|c: |ب", "arabic;ل|c:ا", "arabic;ab ل|c:ا| cd", "arabic;ab ل|c:ا|cd",
	)
}

func TestKerningCrossesAnEdgeRightToLeftAsInChrome(t *testing.T) {
	matchWhole(t,
		"rtl,hebrew;ד|c:ך", "rtl,hebrew;c:ד|ך", "rtl,hebrew;בד|c:ב|ד", "rtl,hebrew;c:ד|c:ב|c:ד", "rtl,hebrew;ד|pad:ך",
		"rtl,hebrew;ד|end:ך", "rtl,hebrew;ד|bold:ך", "rtl,hebrew;ד|lang=he:ך", "hebrew;ד|c:ך", "hebrew;AV ד|c:ך AV",
	)
}

func TestMixedDirectionsShapeEachRunWholeAsInChrome(t *testing.T) {
	matchWhole(t,
		"rtl;A|c:V", "rtl;of|c:fi|ce", "rtl,hebrew;דך|c: AV|A", "rtl,hebrew;AV|c:ד|ך", "hebrew;A|c:Vד|ך",
		"rtl,hebrew;ד|c:ך AV|ד", "rtl,arabic;ل|c:ا ab|c:cd",
	)
}

func TestLinesBreakRightToLeftOutsideALigatureAsChrome(t *testing.T) {
	lines := []string{
		"rtl,hebrew,wrap,all;דךדך", "rtl,hebrew,wrap,all,w40;דךד|c:ך", "rtl,hebrew,wrap;דך דך",
		"rtl,arabic,wrap,all;لالا", "rtl,arabic,wrap,any,w40;بلالا", "rtl,arabic,wrap,any,w40;ببببب",
		"arabic,wrap,any,w40;ببببب", "rtl,arabic,wrap,any,w60;ب|c:بب|ب",
	}
	matchWhole(t, lines...)
	matchCharacters(t, lines...)
}

func TestKerningAndLigaturesCrossAnEdgeInVerticalLinesAsInChrome(t *testing.T) {
	matchWhole(t,
		"vrl;A|c:V", "vrl;of|c:fi|ce", "vlr;A|c:V", "vlr;of|c:fi|ce", "vrl,sideways;A|c:V", "vrl,sideways;of|c:fi|ce",
		"vrl,upright;A|c:V", "vrl,upright;of|c:fi|ce", "vrl;A|pad:V", "vrl;f|end:i", "srl;A|c:V", "srl;of|c:fi|ce",
		"slr;A|c:V", "slr;of|c:fi|ce", "vrl,sideways;A|pad:V", "vrl,sideways;f|end:i", "srl;A|pad:V", "srl;f|end:i",
	)
}
