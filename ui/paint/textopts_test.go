package paint //nolint:testpackage // tests the unexported cache of text options

import (
	"runtime"
	"testing"
	"time"
	"weak"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/twod"
)

func TestATextsOptionsAreMadeOnceAndGoWithTheirFace(t *testing.T) {
	held := twod.Regular().With(twod.Fallback())
	first := textOptions(held, 16, 0, color.White, smoothDarkened)
	again := textOptions(held, 16, 0, color.White, smoothDarkened)
	if &first[0] != &again[0] {
		t.Fatal("the same face, size and colour share one slice of options")
	}
	if other := textOptions(held, 18, 0, color.White, smoothDarkened); &other[0] == &first[0] {
		t.Fatal("another size has its own")
	}
	if none := textOptions(nil, 16, 0, color.White, smoothDarkened); len(none) == 0 || &none[0] == &first[0] {
		t.Fatal("a text without a face has options of its own")
	}
	var gone []weak.Pointer[twod.Face]
	func() {
		for range 20 {
			face := twod.Regular().With(twod.Fallback())
			textOptions(face, 16, 0, color.White, smoothDarkened)
			gone = append(gone, weak.Make(face))
		}
	}()
	left := len(gone)
	deadline := time.Now().Add(5 * time.Second)
	for left > 0 && time.Now().Before(deadline) {
		runtime.GC()
		time.Sleep(10 * time.Millisecond)
		left = 0
		for _, p := range gone {
			if p.Value() != nil {
				left++
			}
		}
	}
	if left > 0 {
		t.Fatalf("%d of %d faces nothing holds are kept by their texts' options", left, len(gone))
	}
	runtime.KeepAlive(held)
}
