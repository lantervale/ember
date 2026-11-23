package orient

import "testing"

func TestOfFollowsUAX50(t *testing.T) {
	for r, want := range map[rune]Orientation{
		'A': Rotated, '1': Rotated, 'é': Rotated, 'א': Rotated, '«': Rotated,
		'§': Upright, '©': Upright, '®': Upright, '±': Upright, '¼': Upright, '×': Upright, '÷': Upright,
		'一': Upright, 'あ': Upright, '한': Upright, '😀': Upright, '🌍': Upright, '\U00020000': Upright,
		'、': TransformedUpright, '。': TransformedUpright, 'ぁ': TransformedUpright,
		'「': TransformedRotated, '（': TransformedRotated, 'ー': TransformedRotated, '〈': TransformedRotated,
		'\U0010FFFF': Rotated,
	} {
		if got := Of(r); got != want {
			t.Errorf("%q (U+%04X): %d, want %d", r, r, got, want)
		}
	}
}

func TestOnlyRotatedTurns(t *testing.T) {
	if Rotated.Stands() || !Upright.Stands() || !TransformedUpright.Stands() || !TransformedRotated.Stands() {
		t.Error("only Rotated turns with the line")
	}
}
