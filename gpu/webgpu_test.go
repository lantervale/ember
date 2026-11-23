package gpu //nolint:testpackage // tests the unexported blend and clamp tables

import (
	"image"
	"image/color"
	"testing"
	"unsafe"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/internal/draw"
	"github.com/lantervale/ember/internal/wgpu"
	"github.com/lantervale/ember/render"
	"github.com/stretchr/testify/assert"
)

func TestEachBlendModeHasItsBlendState(t *testing.T) {
	alpha := wgpu.BlendComponent{Op: wgpu.Add, Src: wgpu.One, Dst: wgpu.OneMinusSrcAlpha}
	for m, want := range map[draw.Blend]wgpu.Blend{
		draw.Alpha: {
			Color: wgpu.BlendComponent{Op: wgpu.Add, Src: wgpu.SrcAlpha, Dst: wgpu.OneMinusSrcAlpha},
			Alpha: alpha,
		},
		draw.Add: {Color: wgpu.BlendComponent{Op: wgpu.Add, Src: wgpu.SrcAlpha, Dst: wgpu.One}, Alpha: alpha},
		draw.Multiply: {
			Color: wgpu.BlendComponent{Op: wgpu.Add, Src: wgpu.Zero, Dst: wgpu.Src},
			Alpha: wgpu.BlendComponent{Op: wgpu.Add, Src: wgpu.Zero, Dst: wgpu.One},
		},
		draw.Screen:  {Color: wgpu.BlendComponent{Op: wgpu.Add, Src: wgpu.One, Dst: wgpu.OneMinusSrc}, Alpha: alpha},
		draw.Lighten: {Color: wgpu.BlendComponent{Op: wgpu.Max, Src: wgpu.One, Dst: wgpu.One}, Alpha: alpha},
		draw.Occlude: {
			Color: wgpu.BlendComponent{Op: wgpu.Max, Src: wgpu.One, Dst: wgpu.One},
			Alpha: wgpu.BlendComponent{Op: wgpu.Max, Src: wgpu.One, Dst: wgpu.One},
		},
		draw.Plus: {
			Color: wgpu.BlendComponent{Op: wgpu.Add, Src: wgpu.One, Dst: wgpu.One},
			Alpha: wgpu.BlendComponent{Op: wgpu.Add, Src: wgpu.One, Dst: wgpu.One},
		},
	} {
		assert.Equal(t, want, blendOf(m), "blend %d", m)
	}
}

func TestAnInvertedCutIsFlaggedInItsUniforms(t *testing.T) {
	b := draw.Batch{Cut: draw.Cut{Mask: &render.Texture{}, ToMask: geom.Identity2D()}}
	assert.Zero(t, batchUniforms(&b, target2D{}, geom.V2(10, 10), 1, 0, 0).CutV[3])
	b.Cut.Inverted = true
	assert.Equal(t, float32(1), batchUniforms(&b, target2D{}, geom.V2(10, 10), 1, 0, 0).CutV[3])
}

func TestTheUniformsLieAsUIWGSLsDrawStruct(t *testing.T) {
	var u drawUniforms
	for name, at := range map[string][2]uintptr{
		"screen":   {unsafe.Offsetof(u.Screen), 0},
		"scale":    {unsafe.Offsetof(u.Scale), 8},
		"time":     {unsafe.Offsetof(u.Time), 12},
		"hdr":      {unsafe.Offsetof(u.HDR), 16},
		"exposure": {unsafe.Offsetof(u.Exposure), 20},
		"masked":   {unsafe.Offsetof(u.Masked), 24},
		"blend":    {unsafe.Offsetof(u.Blend), 28}, "targets": {unsafe.Offsetof(u.Targets), 32},
	} {
		assert.Equal(t, at[1], at[0], name)
	}
	assert.LessOrEqual(t, unsafe.Sizeof(u), uintptr(uniformStride))
}

func TestAClipIsKeptToItsTarget(t *testing.T) {
	assert.Equal(t, 0, clampTo(-3, 100))
	assert.Equal(t, 100, clampTo(140, 100))
	assert.Equal(t, 42, clampTo(41.6, 100))
}

func TestAPictureOverTheLargestTextureIsUploadedSmaller(t *testing.T) {
	wide := image.NewRGBA(image.Rect(0, 0, maxTextureSide+808, 4))
	for x := range wide.Rect.Dx() {
		wide.SetRGBA(x, 0, color.RGBA{R: 255, A: 255})
	}
	got := uploadable(wide)
	assert.Equal(t, image.Rect(0, 0, maxTextureSide, 4), got.Rect,
		"its long side is the largest a device takes, the other kept")
	assert.Equal(t, color.RGBA{R: 255, A: 255}, got.RGBAAt(maxTextureSide-1, 0), "and it is the whole picture, scaled")

	tall := uploadable(image.NewRGBA(image.Rect(0, 0, 3, 2*maxTextureSide)))
	assert.Equal(t, image.Rect(0, 0, 3, maxTextureSide), tall.Rect)

	fits := image.NewRGBA(image.Rect(0, 0, maxTextureSide, 16))
	assert.Same(t, fits, uploadable(fits), "one that fits is uploaded as it is")
	assert.Nil(t, uploadable(image.NewRGBA(image.Rect(0, 0, 0, 16))), "an empty one has nothing to upload")
}
