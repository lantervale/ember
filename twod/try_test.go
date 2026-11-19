package twod //nolint:testpackage // white-box: shares twod's test helpers, reaches unexported state

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/geom"
)

type tried struct {
	Item
	panics          bool
	panicked        any
	stack           []byte
	clips, depth    int
	before, after   geom.Transform2D
	scoped, scoping string
}

func (p *tried) Draw(c *Canvas) {
	p.before, p.clips, p.scoped = c.Transform(), c.list.Clips(), c.ID("x")
	p.panicked, p.stack = c.Try(func() {
		c.PushClip(geom.Rect{W: 10, H: 10})
		c.Push("inside")
		c.SetTransform(c.Transform().Mul(geom.Transform2DFrom(geom.V2(40, 40), 0, 0, geom.V2(1, 1))))
		if p.panics {
			panic("halfway")
		}
		c.Pop()
		c.PopClip()
		c.SetTransform(p.before)
	})
	p.after, p.depth, p.scoping = c.Transform(), c.list.Clips(), c.ID("x")
}

func TestACanvasIsPutBackAfterAPanicItTried(t *testing.T) {
	w, win, _ := app()
	p := &tried{Item: Base()}
	win.Add(Build(p, "tried"))
	w.RunFrames(1)
	assert.Nil(t, p.panicked, "nothing panicked")
	assert.Nil(t, p.stack)

	p.panics = true
	w.RunFrames(1)
	assert.Equal(t, "halfway", p.panicked)
	assert.True(t, strings.Contains(string(p.stack), "try_test.go"), "the stack is where it panicked")
	assert.Equal(t, p.clips, p.depth, "the clips pushed are popped")
	assert.Equal(t, p.before, p.after, "the transform is the one before")
	assert.Equal(t, p.scoped, p.scoping, "the scope is the one before")
	w.RunFrames(1)
}
