package ui

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/internal/draw"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/world"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// repaintsIn counts the frames of w's Documents that painted rather than
// replayed.
func repaintsIn(w *world.World) int {
	n := 0
	for _, c := range w.Stats().Counted() {
		if strings.HasPrefix(c.Name, "ui repaints") {
			n += c.N
		}
	}
	return n
}

// replaysRound counts the frames of w's Documents replayed round their
// live parts.
func replaysRound(w *world.World) int {
	n := 0
	for _, c := range w.Stats().Counted() {
		if c.Name == replayedAround {
			n += c.N
		}
	}
	return n
}

// drawing is what win shows this frame: its front list's vertices, then
// each pass's.
func drawing(w *world.World, win *platform.Window) [][]draw.Vertex {
	all := [][]draw.Vertex{slices.Clone(draw.FrontFor(w, win).Vertices)}
	for _, p := range draw.PassesFor(w, win) {
		all = append(all, slices.Clone(p.List.Vertices))
	}
	return all
}

// coveredFrame is a page whose iframe a box lies over, run until it
// settles.
func coveredFrame(t testing.TB, css string) (*Document, *Document, *platform.Window, *world.World) {
	t.Helper()
	child := NewDocument(CSS(`body { margin: 0; background: #def; } p { margin: 4px; }`))
	for i := range 3 {
		child.Body().Append(P(Text(fmt.Sprintf("line %d of the frame", i))))
	}
	outer := NewDocument(CSS(`iframe { width: 300px; height: 120px; border: 0; }
.over { position: absolute; left: 20px; top: 20px; width: 60px; height: 40px; background: red; }` + css))
	outer.Body().Append(P(Text("before")), Iframe(child), Div(Class("over")), P(Text("after")))
	win := platform.NewWindow(platform.Size(400, 300))
	win.Add(outer)
	w := ember.New().AddEntities(win)
	win.Input().SetMouse(geom.V2(390, 290))
	w.RunFrames(6)
	return outer, child, win, w
}

func TestAFrameUnderABoxReplaysRoundIt(t *testing.T) {
	outer, _, win, w := coveredFrame(t, "")
	require.True(t, outer.replay.inPlace, "the box covers the frame")
	before := repaintsIn(w)
	w.RunFrames(5)
	assert.Equal(t, before, repaintsIn(w), "no frame repaints the page")
	assert.True(t, outer.replay.replayedLast)

	replayed := drawing(w, win)
	outer.replay.repaint = true
	w.RunFrames(1)
	assert.Equal(t, replayed, drawing(w, win), "the frame in its place, the box over it, as a whole paint")
}

func TestATurnedFrameReplaysRoundIt(t *testing.T) {
	outer, _, win, w := coveredFrame(t, "iframe { transform: rotate(3deg); } .over { display: none; }")
	require.True(t, outer.replay.inPlace, "a turned frame is drawn in its place")
	before := repaintsIn(w)
	w.RunFrames(5)
	assert.Equal(t, before, repaintsIn(w))

	replayed := drawing(w, win)
	outer.replay.repaint = true
	w.RunFrames(1)
	assert.Equal(t, replayed, drawing(w, win))
}

func TestAFadedFrameReplaysRoundItsPicture(t *testing.T) {
	for name, css := range map[string]string{
		"on it":     "iframe { opacity: .5; } .over { display: none; }",
		"round it":  "body { opacity: .7; } .over { display: none; }",
		"filtered":  "iframe { filter: blur(1px); } .over { display: none; }",
		"and boxed": "iframe { opacity: .5; }",
	} {
		t.Run(name, func(t *testing.T) {
			outer, _, win, w := coveredFrame(t, css)
			require.True(t, outer.replay.inPlace, "a frame in a picture is drawn in its place")
			before := repaintsIn(w)
			w.RunFrames(5)
			assert.Equal(t, before, repaintsIn(w), "no frame repaints the page")
			assert.Positive(t, replaysRound(w), "replayed round the picture")

			replayed := drawing(w, win)
			outer.replay.repaint = true
			w.RunFrames(1)
			assert.Equal(t, replayed, drawing(w, win), "the picture drawn again as a whole paint draws it")
		})
	}
}

func TestAFrameChangingUnderABoxShowsTheChange(t *testing.T) {
	outer, child, win, w := coveredFrame(t, "")
	w.RunFrames(2)
	text := glyphQuads(draw.FrontFor(w, win))
	child.Body().Append(P(Text("more")))
	w.RunFrames(2)
	assert.Equal(t, text+4, glyphQuads(draw.FrontFor(w, win)), "the frame's new line, the page replayed round it")
	assert.True(t, outer.replay.replayedLast)
}

func BenchmarkAFrameUnderABox(b *testing.B) {
	_, _, _, w := coveredFrame(b, "")
	b.ReportAllocs()
	for b.Loop() {
		w.RunFrames(1)
	}
}
