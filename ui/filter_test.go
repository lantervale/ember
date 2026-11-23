package ui_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/ui"
)

func TestAFilteredElementHoldsItsFixedAndAbsoluteDescendants(t *testing.T) {
	p := newScrollingPage(t, "body { filter: grayscale(1); }")
	assert.Equal(t, geom.Rect{X: 320, Y: pageHeight - 30, W: 50, H: 20}, p.badge.Box(),
		"a fixed box is placed in the filtered body, not the viewport")
	p.key(input.KeyEnd)
	assert.InDelta(t, pageHeight-30-p.scrolled(), p.badge.Box().Y, 1e-3, "and scrolls with it")

	q := newTallPage(t, "#lead { filter: blur(1px); margin-left: 7px; } #lead > div { position: absolute; top: 3px; }")
	in := ui.Div(ui.Style("width: 5px; height: 5px"))
	q.doc.Body().Children()[0].Append(in)
	q.w.RunFrames(2)
	assert.Equal(t, geom.Rect{X: 7, Y: 3, W: 5, H: 5}, in.Box(), "an absolute box too, though it is static")
}

func TestAnElementFilteringItsBackdropHoldsItsFixedDescendants(t *testing.T) {
	held := geom.Rect{X: 320, Y: pageHeight - 30, W: 50, H: 20}
	p := newScrollingPage(t, "body { backdrop-filter: invert(1); }")
	assert.Equal(t, held, p.badge.Box(), "a fixed box is placed in the body, not the viewport")
	m := newScrollingPage(t, "body { mix-blend-mode: multiply; }")
	assert.NotEqual(t, held, m.badge.Box(), "a blend mode holds none")
}
