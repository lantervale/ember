package ui

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/image/font/gofont/gobold"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/internal/draw"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/world"
)

// heldFont is a Document whose one face, of display, is fetched only
// once release is closed, shown in a window of a World of its own.
func heldFont(t *testing.T, display string) (*Document, *world.World, *platform.Window, chan struct{}) {
	t.Helper()
	release := make(chan struct{})
	d := NewDocument(FetchFonts(func(string, string) ([]byte, error) {
		<-release
		return gobold.TTF, nil
	}), CSS(`@font-face { font-family: Heavy; src: url(heavy.ttf); font-display: `+display+`; }
div { display: inline-block; font: 25px/1 Heavy, monospace; }`))
	d.Body().Append(Div(ID("text"), Text("1 2 3 4")))
	win := platform.NewWindow(platform.Size(400, 300))
	win.Add(d)
	w := ember.New().AddEntities(win)
	t.Cleanup(func() {
		select {
		case <-release:
		default:
			close(release)
		}
	})
	return d, w, win, release
}

// until runs frames, at the fixed step, until the World's real time is
// at least seconds.
func until(w *world.World, seconds float32) {
	for w.Time().RealElapsed < seconds {
		w.RunFrames(1)
	}
}

// land lets the held font come and runs frames until it has landed.
func land(t *testing.T, d *Document, w *world.World, release chan struct{}) {
	t.Helper()
	close(release)
	require.Eventually(t, func() bool {
		w.RunFrames(1)
		return d.FontsLoading() == 0
	}, 5*time.Second, time.Millisecond)
	w.RunFrames(2)
}

func TestAFaceInItsBlockPeriodDrawsItsTextInvisibleInItsRoom(t *testing.T) {
	d, w, win, release := heldFont(t, "block")
	until(w, 1)
	assert.Zero(t, glyphQuads(draw.FrontFor(w, win)), "invisible for 3 s")
	wide := d.ByID("text").Box().W
	assert.Positive(t, wide, "laid out in the next family")
	until(w, 3.1)
	assert.Equal(t, 4, glyphQuads(draw.FrontFor(w, win)), "then in the next family")
	assert.InDelta(t, wide, d.ByID("text").Box().W, 0.01)
	land(t, d, w, release)
	assert.Equal(t, 4, glyphQuads(draw.FrontFor(w, win)))
	assert.NotEqual(t, wide, d.ByID("text").Box().W, "swapped in at last")
}

func TestFontDisplayPeriodsAreTimedOnTheWorldsRealTime(t *testing.T) {
	for _, k := range []struct {
		display      string
		hiddenUntil  float32
		landAt       float32
		swapped      bool
		hiddenAtOnce bool
	}{
		{"auto", 3, 2, true, true},
		{"swap", 0, 10, true, false},
		{"fallback", 0.1, 2.5, true, true},
		{"fallback", 0.1, 3.5, false, true},
		{"optional", 0.1, 0.5, false, true},
	} {
		d, w, win, release := heldFont(t, k.display)
		w.RunFrames(2)
		assert.Equal(t, k.hiddenAtOnce, glyphQuads(draw.FrontFor(w, win)) == 0, "%s at once", k.display)
		until(w, k.hiddenUntil+0.05)
		assert.Equal(t, 4, glyphQuads(draw.FrontFor(w, win)), "%s after its block period", k.display)
		wide := d.ByID("text").Box().W
		until(w, k.landAt)
		land(t, d, w, release)
		assert.Equal(t, k.swapped, d.ByID("text").Box().W != wide, "%s landing at %vs", k.display, k.landAt)
	}
}
