package draw //nolint:testpackage // white-box: reaches draw's unexported state

import (
	"os"
	"runtime"
	"testing"
	"time"

	"golang.org/x/image/font/gofont/goregular"
)

func kerningEntries() int {
	kerningMu.Lock()
	defer kerningMu.Unlock()
	return len(kerning)
}

func TestAFaceNoLongerHeldLeavesTheKerningCache(t *testing.T) {
	before := kerningEntries()
	func() {
		f, err := ParseFace("page-font", goregular.TTF)
		if err != nil {
			t.Fatal(err)
		}
		f.Size(16)
		if kerningEntries() != before+1 {
			t.Fatalf("%d kerning tables, want %d", kerningEntries(), before+1)
		}
	}()
	deadline := time.Now().Add(5 * time.Second)
	for kerningEntries() != before && time.Now().Before(deadline) {
		runtime.GC()
		time.Sleep(10 * time.Millisecond)
	}
	if got := kerningEntries(); got != before {
		t.Errorf("%d kerning tables after the face is gone, want %d: the cache keeps its font", got, before)
	}
}

func TestAFaceDrawsOnlyTheCharactersOfItsRanges(t *testing.T) {
	latin := Bold.Only([]Range{{'a', 'z'}})
	f := latin.With([]*Face{Regular}, nil).Size(16)
	if g := f.glyph('b'); g.face != 0 {
		t.Errorf("b is drawn by face %d, want the ranged face", g.face)
	}
	if g := f.glyph('B'); g.face != 1 {
		t.Errorf("B, outside the ranges, is drawn by face %d, want the fallback", g.face)
	}
	if !latin.With([]*Face{Regular}, nil).Draws("aB") || latin.Draws("B") {
		t.Error("Draws: a face draws a character outside its ranges, or its fallback does not")
	}
}

func TestACharacterOnItsWayDrawsAsNothing(t *testing.T) {
	const icon = ''
	if g := Regular.Size(16).glyph(icon); g.advance == 0 {
		t.Fatal("a character no face has takes no room: it shows as '?'")
	}
	g := Regular.Blank([]Range{{0xe000, 0xf8ff}}).Size(16).glyph(icon)
	if g.advance != 0 || !g.rect.Empty() {
		t.Errorf("a character on its way takes %v of room and draws %v", g.advance, g.rect)
	}
	if other := Regular.Blank([]Range{{0xe000, 0xf8ff}}).Size(16).glyph('一'); other.advance == 0 {
		t.Error("a character outside the ranges on their way shows as nothing")
	}
}

func TestAFaceKernedByGPOSDrawsItsPairsCloser(t *testing.T) {
	data, err := os.ReadFile("testdata/go-kerned.ttf")
	if err != nil {
		t.Fatal(err)
	}
	face, err := ParseFace("kerned", data)
	if err != nil {
		t.Fatal(err)
	}
	f := face.Size(20)
	a, v, b := f.Measure("A").X, f.Measure("V").X, f.Measure("B").X
	if got := f.Measure("AB").X; got != a+b {
		t.Errorf("AB is %v wide, want %v: the pair has no kerning", got, a+b)
	}
	// 200 units of 2048 closer, at 20 px.
	if got, want := f.Measure("AV").X, a+v-200.0/2048*20; got < want-0.01 || got > want+0.01 {
		t.Errorf("AV is %v wide, want %v: a pair GPOS does not hold must not hide the one it does", got, want)
	}
}
