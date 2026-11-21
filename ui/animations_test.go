package ui_test

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/world"
)

const fadeIn = `@keyframes in { from { opacity: 0 } to { opacity: 1 } }
.card { height: 20px; opacity: 0; animation: in 1s linear forwards }
@media (prefers-reduced-motion: reduce) { .card { animation: none; opacity: 1 } }`

func TestAnimationsRunOnTheWorldsRealTime(t *testing.T) {
	win := platform.NewWindow(platform.Size(320, 200))
	doc := ui.NewDocument(ui.CSS(fadeIn))
	card := ui.Div(ui.Class("card"))
	doc.Body().Append(card)
	win.Add(doc)
	w := ember.New().AddEntities(win)
	w.RunFrames(1)
	assert.InDelta(t, 0, ui.ShownOpacity(card), 0.001)
	w.Time().Scale = 0
	w.RunFrames(30)
	mid := ui.ShownOpacity(card)
	assert.Greater(t, mid, float32(0.3))
	assert.Less(t, mid, float32(0.7))
	w.RunFrames(60)
	assert.InDelta(t, 1, ui.ShownOpacity(card), 0.001, "filled forwards to the last keyframe")
}

func TestAnimationEventsAreWorldEventsAndCallbacks(t *testing.T) {
	win := platform.NewWindow(platform.Size(320, 200))
	doc := ui.NewDocument(ui.CSS(`@keyframes in { from { opacity: 0 } } .card { animation: in 0.25s 2 }`))
	var ended []ui.Event
	card := ui.Div(ui.Class("card"), ui.OnAnimationEnd(func(e ui.Event) { ended = append(ended, e) }))
	doc.Body().Append(card)
	win.Add(doc)
	w := ember.New().AddEntities(win)
	var started []ui.AnimationStarted
	var iters []ui.AnimationIterated
	var ends []ui.AnimationEnded
	var r1 world.Reader[ui.AnimationStarted]
	var r2 world.Reader[ui.AnimationIterated]
	var r3 world.Reader[ui.AnimationEnded]
	for range 40 {
		w.RunFrames(1)
		started = slices.AppendSeq(started, r1.Read(w))
		iters = slices.AppendSeq(iters, r2.Read(w))
		ends = slices.AppendSeq(ends, r3.Read(w))
	}
	assert.Equal(t, []ui.AnimationStarted{{Node: card, Name: "in"}}, started)
	assert.Equal(t, []ui.AnimationIterated{{Node: card, Name: "in", Elapsed: 0.25}}, iters)
	assert.Equal(t, []ui.AnimationEnded{{Node: card, Name: "in", Elapsed: 0.5}}, ends)
	if assert.Len(t, ended, 1) {
		assert.Equal(t, ui.EventAnimationEnd, ended[0].Type)
		assert.Equal(t, "in", ended[0].Animation)
	}
}

func TestADocumentAnswersReducedMotion(t *testing.T) {
	doc := ui.NewDocument(ui.CSS(fadeIn), ui.ReducedMotion())
	card := ui.Div(ui.Class("card"))
	doc.Body().Append(card)
	doc.Layout(geom.V2(320, 200))
	assert.True(t, doc.ReducedMotion())
	assert.InDelta(t, 1, ui.ShownOpacity(card), 0.001, "the page's reduce rule stops the fade")
	doc.SetReducedMotion(false)
	doc.Layout(geom.V2(320, 200))
	assert.InDelta(t, 0, ui.ShownOpacity(card), 0.001, "no preference: the fade runs")
}
