package layout //nolint:testpackage // tests the layout tree and its unexported helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

func TestPreservedSpacesEndingTheContentTakeRoom(t *testing.T) {
	pre := func(st *style.Computed) { st.Display, st.WhiteSpace = style.Block, style.Pre }
	sizes := map[string]geom.Vec2{
		"HH   ":     geom.V2(50, 10),
		"   ":       geom.V2(30, 10),
		"HH\n   ":   geom.V2(30, 20),
		"HH\n   \n": geom.V2(30, 20),
	}
	for s, want := range sizes {
		p := el(dom.Pre, pre, text(s))
		f := el(dom.Div, func(st *style.Computed) { st.Display, st.Float = style.Block, style.FloatLeft }, p)
		tr := lay(el(dom.Div, block, f), 400)
		assert.Equal(t, want, tr.Of[p].Size, "%q", s)
	}
}

func TestSpacesThatWrapOrCollapseStillHangAtTheContentsEnd(t *testing.T) {
	for _, ws := range []style.WhiteSpace{style.Normal, style.PreWrap, style.PreLine, style.NoWrapSpace} {
		p := el(dom.P, func(st *style.Computed) { st.Display, st.WhiteSpace = style.Block, ws }, text("HH   "))
		f := el(dom.Div, func(st *style.Computed) { st.Display, st.Float = style.Block, style.FloatLeft }, p)
		tr := lay(el(dom.Div, block, f), 400)
		assert.Equal(t, []string{"HH"}, lines(tr.Of[p]), "white-space %v", ws)
	}
}
