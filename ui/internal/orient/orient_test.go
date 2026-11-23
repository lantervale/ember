package orient_test

import (
	"testing"

	"github.com/lantervale/ember/ui/internal/orient"
)

func TestOfFollowsUAX50(t *testing.T) {
	const (
		r  = orient.Rotated
		u  = orient.Upright
		tu = orient.TransformedUpright
		tr = orient.TransformedRotated
	)
	for c, want := range map[rune]orient.Orientation{
		'A': r, '1': r, 'é': r, 'א': r, '«': r, '\U0010FFFF': r,
		'§': u, '©': u, '®': u, '±': u, '¼': u, '×': u, '÷': u,
		'一': u, 'あ': u, '한': u, '😀': u, '🌍': u, '\U00020000': u,
		'、': tu, '。': tu, 'ぁ': tu,
		'「': tr, '（': tr, 'ー': tr, '〈': tr,
	} {
		if got := orient.Of(c); got != want {
			t.Errorf("%q (U+%04X): %d, want %d", c, c, got, want)
		}
	}
}

func TestOnlyRotatedTurns(t *testing.T) {
	if orient.Rotated.Stands() || !orient.Upright.Stands() || !orient.TransformedUpright.Stands() ||
		!orient.TransformedRotated.Stands() {
		t.Error("only Rotated turns with the line")
	}
}
