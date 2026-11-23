package paint //nolint:testpackage // white-box: builds box trees by hand and reads the painter's own state

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/render"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/ui/internal/style"
)

var _ Pictures = (*twod.Canvas)(nil)

// picturing is a recorder that also writes down its pictures: begin, end
// with its margin, show with its effects by shader.
type picturing struct {
	*recorder
	effects []*render.Effect
}

func newPicturing() *picturing { return &picturing{recorder: newRecorder()} }

func (r *picturing) BeginPicture() { r.add(op{Kind: "begin"}) }
func (r *picturing) EndPicture(margin float32) *twod.Picture {
	r.add(op{Kind: "end", Width: margin})
	return &twod.Picture{}
}

func (r *picturing) DrawPicture(_ *twod.Picture, effects ...*render.Effect) {
	var names []string
	for _, e := range effects {
		names = append(names, shaderName(e.Shader()))
	}
	r.add(op{Kind: "show", Text: strings.Join(names, " ")})
	r.effects = append(r.effects, effects...)
}

func (r *picturing) DrawPictureOver(_, _ *twod.Picture, m twod.Mix, effects ...*render.Effect) {
	names := []string{fmt.Sprint(m)}
	for _, e := range effects {
		names = append(names, shaderName(e.Shader()))
	}
	r.add(op{Kind: "mix", Text: strings.Join(names, " ")})
	r.effects = append(r.effects, effects...)
}

func (r *picturing) Effect(shader string) *render.Effect { return render.NewEffect(shader) }

func shaderName(s string) string {
	switch s {
	case matrixShader:
		return "matrix"
	case blurShader:
		return "blur"
	case shadowShader:
		return "shadow"
	case mirroredBlurShader:
		return "mirrored"
	}
	return "?"
}

// steps are r's ops as words: pictures, clips and fills.
func (r *picturing) steps() string {
	var out []string
	for _, o := range r.ops {
		switch o.Kind {
		case "begin", "pop":
			out = append(out, o.Kind)
		case "end":
			out = append(out, fmt.Sprintf("end %g", o.Width))
		case "show":
			out = append(out, "show("+o.Text+")")
		case "mix":
			out = append(out, "mix("+o.Text+")")
		case "push-round":
			out = append(out, fmt.Sprintf("round %g", o.Rect.X))
		case "push":
			out = append(out, fmt.Sprintf("push %g", o.Rect.X))
		case "push-mask":
			out = append(out, fmt.Sprintf("mask %g", o.Rect.X))
		case "rect":
			out = append(out, "rect")
		}
	}
	return strings.Join(out, ", ")
}

func filtered(filters ...style.Filter) func(s *style.Computed) {
	return func(s *style.Computed) {
		s.Background = blue
		s.Filters = filters
	}
}

func TestAFilteredElementAndItsContentAreOnePictureShownThroughItsFilters(t *testing.T) {
	r := newPicturing()
	Paint(r, tree(el(0, 0, 400, 300, nil,
		el(10, 10, 100, 100, filtered(style.Filter{Kind: style.Blur, Amount: 2}, style.Filter{Kind: style.Grayscale,
			Amount: 1}), el(10, 10, 20, 20, bg(red))))), Scale(2))
	assert.Equal(t, "begin, rect, rect, end 6, show(blur blur matrix)", r.steps())
	require.Len(t, r.effects, 3)
	across, down := r.effects[0], r.effects[1]
	sigma, _ := across.Uniform("uSigma")
	assert.InDelta(t, 4, sigma, 1e-6, "two points, in pixels")
	step, _ := across.Uniform("uStep")
	assert.Equal(t, geom.V2(1, 0), step)
	step, _ = down.Uniform("uStep")
	assert.Equal(t, geom.V2(0, 1), step)
}

func TestAFilterUnderATransformBlursAsFarAsItIsScaled(t *testing.T) {
	r := newPicturing()
	Paint(r, tree(el(0, 0, 400, 300, nil, el(10, 10, 100, 100, func(s *style.Computed) {
		filtered(style.Filter{Kind: style.Blur, Amount: 2})(s)
		s.Transform = []style.TransformOp{{Kind: style.ScaleOp, A: [6]float32{3, 3}}}
	}))))
	sigma, _ := r.effects[0].Uniform("uSigma")
	assert.InDelta(t, 6, sigma, 1e-5)
	assert.Contains(t, r.steps(), "end 18", "the margin, in the surface's points")
}

func TestADropShadowShowsThePictureTwiceAndFiltersAfterItTheTwo(t *testing.T) {
	shadow := style.Filter{Kind: style.DropShadow, Shadow: style.Shadow{X: 4, Y: -2, Blur: 6, Color: red}}
	r := newPicturing()
	Paint(r, tree(el(0, 0, 400, 300, nil, el(10, 10, 100, 100, filtered(shadow)))))
	assert.Equal(t, "begin, rect, end 13, show(shadow blur blur), show()", r.steps())
	at, _ := r.effects[0].Uniform("uOffset")
	assert.Equal(t, geom.V2(4, -2), at)
	sigma, _ := r.effects[1].Uniform("uSigma")
	assert.InDelta(t, 3, sigma, 1e-6, "a drop shadow's blur is a radius, twice the deviation")

	r = newPicturing()
	Paint(r, tree(el(0, 0, 400, 300, nil, el(10, 10, 100, 100, filtered(
		style.Filter{Kind: style.Invert, Amount: 1}, shadow, style.Filter{Kind: style.Blur, Amount: 1})))))
	assert.Equal(t, "begin, begin, rect, end 13, show(matrix shadow blur blur), show(matrix), end 3, show(blur blur)",
		r.steps(), "what comes before the shadow under both, the blur after it over the two")
}

func TestAFilteredPictureIsDrawnWithoutTheClipsAroundItAndShownInThem(t *testing.T) {
	r := newPicturing()
	Paint(r, tree(el(0, 0, 400, 300, nil, el(5, 5, 200, 200, func(s *style.Computed) {
		s.Overflow = style.Size[style.Overflow]{Width: style.Hidden, Height: style.Hidden}
	}, el(10, 10, 100, 100, filtered(style.Filter{Kind: style.Sepia, Amount: 1}),
		el(0, 0, 300, 50, func(s *style.Computed) {
			s.Background = red
			s.Overflow = style.Size[style.Overflow]{Width: style.Hidden, Height: style.Hidden}
		}, el(0, 0, 400, 10, bg(green))))))))
	assert.Equal(t, "begin, rect, rect, push 15, rect, pop, end 0, push 5, show(matrix), pop", r.steps())
}

func TestAFilteredElementIsCutByItsClipPathAfterItsFilters(t *testing.T) {
	d := el(50, 50, 100, 100, func(s *style.Computed) {
		filtered(style.Filter{Kind: style.Blur, Amount: 2})(s)
		s.ClipPath = circleOf(40)
	}, el(-20, -20, 140, 140, bg(red)))
	r := newPicturing()
	Paint(r, tree(el(0, 0, 400, 300, nil, d)))
	assert.Equal(t, "begin, rect, rect, end 6, mask 60, show(blur blur), pop", r.steps(),
		"drawn whole and blurred, then cut to the circle, its blur cut too")
	assert.Equal(t, geom.Rect{X: 10, Y: 10, W: 80, H: 80}, d.Ink, "the blur's ink cut to the circle")
}

func TestFiltersStartAStackingContext(t *testing.T) {
	r := newPicturing()
	Paint(r, tree(el(0, 0, 400, 300, nil,
		el(0, 0, 100, 100, filtered(style.Filter{Kind: style.Invert, Amount: 1}), el(0, 0, 10, 10, func(s *style.Computed) {
			s.Background = red
			s.Position = style.Relative
			s.ZIndex, s.ZAuto = -1, false
		})),
		el(0, 0, 100, 100, bg(green)))))
	assert.Equal(t, "rect, begin, rect, rect, end 0, show(matrix)", r.steps(),
		"painted over the flow, as a context, the child under z-index -1 in its picture")
	assert.Equal(t, "green blue red", r.order(map[color.Color]string{blue: "blue", red: "red", green: "green"}))
}

func TestASurfaceWithoutPicturesDrawsAFilteredElementUnfiltered(t *testing.T) {
	r := newRecorder()
	Paint(r, tree(el(0, 0, 400, 300, nil, el(10, 10, 100, 100, filtered(style.Filter{Kind: style.Blur, Amount: 2})))))
	assert.Len(t, r.of("rect"), 1)
}

func TestFiltersGrowTheInkTheyBlurOrShadow(t *testing.T) {
	blurred := el(10, 10, 100, 50, filtered(style.Filter{Kind: style.Blur, Amount: 2}))
	shadowed := el(10, 100, 100, 50, filtered(style.Filter{Kind: style.DropShadow,
		Shadow: style.Shadow{X: 10, Y: 20, Blur: 4, Color: red}}))
	t1 := tree(el(0, 0, 400, 300, nil, blurred, shadowed))
	inked(t1)
	assert.Equal(t, geom.Rect{X: -6, Y: -6, W: 112, H: 62}, blurred.Ink)
	assert.Equal(t, geom.Rect{X: 0, Y: 0, W: 116, H: 76}, shadowed.Ink)
}

func TestAFilterBlurringIntoViewIsPaintedThoughItsBoxIsOut(t *testing.T) {
	r := newPicturing()
	Paint(r, tree(el(0, 0, 400, 300, nil, el(0, 305, 100, 100, filtered(style.Filter{Kind: style.Blur, Amount: 4}),
		el(0, 0, 10, 10, bg(red))))), Viewport(geom.Rect{W: 400, H: 300}))
	assert.Equal(t, "begin, rect, rect, end 12, show(blur blur)", r.steps(),
		"its blur reaches 12 points up, and its child too")
}

// apply is c through m, unpremultiplied and clamped as the shader does.
func apply(m [4][5]float32, c [4]float32) [4]float32 {
	var out [4]float32
	for i := range out {
		v := m[i][4]
		for k := range c {
			v += m[i][k] * c[k]
		}
		out[i] = float32(math.Round(float64(min(max(v, 0), 1))*1000) / 1000)
	}
	return out
}

func TestColourFiltersAreFilterEffectsMatrices(t *testing.T) {
	red := [4]float32{1, 0, 0, 1}
	for _, c := range []struct {
		f    style.Filter
		in   [4]float32
		want [4]float32
	}{
		{style.Filter{Kind: style.Grayscale, Amount: 1}, red, [4]float32{0.213, 0.213, 0.213, 1}},
		{style.Filter{Kind: style.Grayscale}, red, red},
		{style.Filter{Kind: style.Sepia, Amount: 1}, red, [4]float32{0.393, 0.349, 0.272, 1}},
		{style.Filter{Kind: style.Saturate, Amount: 0}, red, [4]float32{0.213, 0.213, 0.213, 1}},
		{style.Filter{Kind: style.Saturate, Amount: 1}, red, red},
		{style.Filter{Kind: style.HueRotate, Amount: 0}, red, red},
		{style.Filter{Kind: style.HueRotate, Amount: 360}, red, red},
		{style.Filter{Kind: style.HueRotate, Amount: 180}, red, [4]float32{0, 0.426, 0.426, 1}},
		{style.Filter{Kind: style.Invert, Amount: 1}, red, [4]float32{0, 1, 1, 1}},
		{style.Filter{Kind: style.Invert, Amount: 0.5}, red, [4]float32{0.5, 0.5, 0.5, 1}},
		{style.Filter{Kind: style.OpacityFilter, Amount: 0.25}, red, [4]float32{1, 0, 0, 0.25}},
		{style.Filter{Kind: style.Brightness, Amount: 0.5}, red, [4]float32{0.5, 0, 0, 1}},
		{style.Filter{Kind: style.Contrast, Amount: 0}, red, [4]float32{0.5, 0.5, 0.5, 1}},
		{style.Filter{Kind: style.Contrast, Amount: 2}, [4]float32{0.6, 0.4, 0.5, 1}, [4]float32{0.7, 0.3, 0.5, 1}},
	} {
		assert.Equal(t, c.want, apply(colorMatrix(c.f), c.in), "%+v", c.f)
	}
}
