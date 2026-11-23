package twod //nolint:testpackage // white-box: shares twod's test helpers, reaches unexported state

import (
	"os"
	"slices"
	"testing"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/stretchr/testify/assert"
)

func TestCanvasTextReturnsItsMeasureAndDrawsAsBefore(t *testing.T) {
	w, _, _ := app()
	w.RunFrames(1)
	c := CanvasFor(w, nil)
	for _, text := range []string{"Handgloves", "tile 12 · Edit", "x", ""} {
		for _, opts := range [][]TextOption{
			{FontSize(15)},
			{FontSize(15), Baseline()},
			{FontSize(13), Pivot(0, 0), Tracking(0.5)},
			{FontSize(16), Baseline(), TextShadow(color.Black)},
			{FontSize(16), Pivot(0.5, 0.5)},
			{FontSize(16), Pivot(0, 1), Baseline()},
		} {
			want := c.Measure(text, opts...)
			n := len(c.list.Vertices)
			assert.Equal(t, want, c.Text(geom.V2(10.3, 30), text, opts...), "%q", text)
			drawn := slices.Clone(c.list.Vertices[n:])

			tx := c.scratchText(text, opts)
			size := tx.style().Measure(tx.shown(text))
			off := geom.V2(size.X*tx.pivot.X, size.Y*tx.pivot.Y)
			if tx.baseline {
				off.Y = 0
			}
			n = len(c.list.Vertices)
			tx.paint(c, geom.V2(10.3, 30).Sub(off), tx.shown(text))
			assert.Equal(t, c.list.Vertices[n:], drawn, "%q: the same quads as measuring first", text)
		}
	}
}

func BenchmarkCanvasText(b *testing.B) {
	w, _, _ := app()
	w.RunFrames(1)
	c := CanvasFor(w, nil)
	opts := []TextOption{FontSize(15), Baseline(), Color(color.White)}
	b.ReportAllocs()
	for range b.N {
		c.list.Vertices, c.list.Indices, c.list.Batches = c.list.Vertices[:0], c.list.Indices[:0], c.list.Batches[:0]
		c.Text(geom.V2(10, 20), "tile 412 · walls · Edit", opts...)
	}
}

func TestATextIsShapedWithTheFeaturesItAsksFor(t *testing.T) {
	face, err := LoadFace(os.DirFS("../internal/shape/testdata"), "go-features.ttf")
	if err != nil {
		t.Fatal(err)
	}
	text := NewText("office", Features("-liga", "dlig"), Typeface(face))
	assert.Equal(t, "-liga,dlig", text.Features())
	tied, apart := Measure("office", Typeface(face)), Measure("office", Typeface(face), Features("-liga"))
	assert.Less(t, tied.X, apart.X, "ffi is narrower than f, f and i")
	text.SetFeatures()
	assert.Empty(t, text.Features())
}

func TestArabicIsJoinedWhetherDrawnRightToLeftOrMeasured(t *testing.T) {
	face, err := LoadFace(os.DirFS("../internal/shape/testdata"), "go-arabic.ttf")
	if err != nil {
		t.Fatal(err)
	}
	text := NewText("بتب", Typeface(face), RightToLeft())
	assert.True(t, text.RightToLeft())
	text.SetRightToLeft(false)
	assert.False(t, text.RightToLeft())
	joined := Measure("بتب", Typeface(face), FontSize(2048))
	assert.InDelta(t, 1399+1099+1399, joined.X, 0.5, "beh.init, teh.medi, beh.fina")
	assert.Equal(t, joined, Measure("بتب", Typeface(face), FontSize(2048), RightToLeft()))
}

func TestATextBetweenOthersJoinsWithThem(t *testing.T) {
	face, err := LoadFace(os.DirFS("../internal/shape/testdata"), "go-arabic.ttf")
	if err != nil {
		t.Fatal(err)
	}
	text := NewText("ت", Typeface(face), Between("ب", "ب"))
	before, after := text.Between()
	assert.Equal(t, [2]string{"ب", "ب"}, [2]string{before, after})
	text.SetBetween("", "")
	before, after = text.Between()
	assert.Empty(t, before+after)
	alone := Measure("ت", Typeface(face), FontSize(2048))
	medial := Measure("ت", Typeface(face), FontSize(2048), Between("ب", "ب"))
	assert.InDelta(t, 1099, medial.X, 0.5, "teh.medi, as in بتب")
	assert.NotEqual(t, alone.X, medial.X)
}
