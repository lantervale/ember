package layout //nolint:testpackage // tests the layout tree and its unexported helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

// Each fragment says where its text starts in its node's, in characters,
// through collapsed spaces, wrapped lines, words broken anywhere and
// characters of several bytes.
func TestFragmentsSayWhereTheirTextStartsInTheNode(t *testing.T) {
	pre := func(st *style.Computed) { width(400)(st); st.WhiteSpace = style.Pre }
	anywhere := func(st *style.Computed) { width(20)(st); st.BreakWord = true }
	cases := []struct {
		name string
		edit func(*style.Computed)
		text string
		want []int32
	}{
		{"one line", width(400), "  HH   HHH  HH ", []int32{2}},
		{"wrapped", width(40), "HHHH HHHH   HHHH", []int32{0, 5, 12}},
		{"several bytes", width(30), "été çà   là", []int32{0, 4, 9}},
		{"pre", pre, "H  H\n H", []int32{0, 5}},
		{"anywhere", anywhere, "HHHHHHH", []int32{0, 2, 4, 6}},
	}
	for _, c := range cases {
		node := text(c.text)
		tr := lay(el(dom.Div, block, el(dom.P, c.edit, node)), 800)
		var got []int32
		for _, f := range tr.Of[node].Lines {
			got = append(got, f.Start)
		}
		assert.Equal(t, c.want, got, c.name)
	}
}
