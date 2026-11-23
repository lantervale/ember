//go:build chrome

package chrome

import (
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/stretchr/testify/assert"
)

func TestANarrowOrShortPagesPictureOpensAWindowAtLeastFiveHundredPointsEachWay(t *testing.T) {
	page := []byte("<html><head></head><body></body></html>")
	for _, c := range []struct {
		size, window geom.Vec2
	}{
		{geom.V2(160, 104), geom.V2(500, 500)},
		{geom.V2(520, 80), geom.V2(520, 500)},
		{geom.V2(560, 140), geom.V2(560, 500)},
		{geom.V2(640, 160), geom.V2(640, 160)},
		{geom.V2(420, 160), geom.V2(500, 500)},
		{geom.V2(640, 520), geom.V2(640, 520)},
	} {
		got, window := fit(page, c.size, true)
		assert.Equal(t, c.window, window, "%v", c.size)
		if window != c.size {
			assert.Contains(t, string(got), "html { width: ", "%v keeps its size on the root", c.size)
			assert.Contains(t, string(got), "position: relative", "%v", c.size)
		}
	}
}

func TestABoxDumpKeepsItsWindowAsTall(t *testing.T) {
	_, window := fit([]byte("<head></head>"), geom.V2(160, 104), false)
	assert.Equal(t, geom.V2(500, 104), window, "only pictures get the taller window")
}

func TestThePageOfAScrollingViewportKeepsItsRootAndIsPicturedInAFrame(t *testing.T) {
	page := []byte(`<html data-ember-scrolling-viewport><head></head><body>"a"</body></html>`)
	got, window := fit(page, geom.V2(640, 400), false)
	assert.Equal(t, page, got, "its boxes are dumped with the root as it is: its own height, its bar")
	assert.Equal(t, geom.V2(640, 400), window)

	got, window = fit(page, geom.V2(320, 260), true)
	assert.Equal(t, geom.V2(500, 500), window)
	assert.Contains(t, string(got), `<iframe style="display: block; border: 0; width: 320px; height: 260px" srcdoc="`)
	assert.Contains(t, string(got), "&lt;html data-ember-scrolling-viewport&gt;")
	assert.Contains(t, string(got), "&#34;a&#34;")
	assert.NotContains(t, string(got), "scrollbar-width")
}
