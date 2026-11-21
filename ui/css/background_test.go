package css //nolint:testpackage // white-box: reads computed styles

import (
	"testing"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBackgroundAttachmentOriginAndClip(t *testing.T) {
	a, b, c := el(dom.Div, "a"), el(dom.Div, "b"), el(dom.Div, "c")
	Compute(page(a, b, c), []*Sheet{sheet(t, `
#a { background-attachment: fixed; background-origin: content-box; background-clip: padding-box }
#b { background-attachment: local; background-origin: border-box; background-clip: content-box }
#c { background-attachment: scroll; background-origin: padding-box; background-clip: border-box }`)})
	st := styleOf(a)
	assert.Equal(t, style.AttachFixed, st.BackgroundLayer(0).Attachment)
	assert.Equal(t, style.UnderContent, st.BackgroundLayer(0).Origin)
	assert.Equal(t, style.UnderPadding, st.BackgroundLayer(0).Clip)
	st = styleOf(b)
	assert.Equal(t, style.AttachLocal, st.BackgroundLayer(0).Attachment)
	assert.Equal(t, style.UnderBorder, st.BackgroundLayer(0).Origin)
	assert.Equal(t, style.UnderContent, st.BackgroundLayer(0).Clip)
	st = styleOf(c)
	assert.Equal(t, style.AttachScroll, st.BackgroundLayer(0).Attachment)
	assert.Equal(t, style.InitialBox, st.BackgroundLayer(0).Origin, "the padding box is where a picture starts")
	assert.Equal(t, style.InitialBox, st.BackgroundLayer(0).Clip, "the border box is what a background fills")
}

func TestTheBackgroundShorthandTakesItsPartsInAnyOrder(t *testing.T) {
	cases := map[string]func(t *testing.T, st *style.Computed){
		"fixed url(a.png) red": func(t *testing.T, st *style.Computed) {
			assert.Equal(t, style.AttachFixed, st.BackgroundLayer(0).Attachment)
			assert.Equal(t, "a.png", st.BackgroundLayer(0).Image.Source)
			assert.Equal(t, color.RGB(1, 0, 0), st.Background)
		},
		"red no-repeat local 10px 20px url(a.png)": func(t *testing.T, st *style.Computed) {
			assert.Equal(t, style.AttachLocal, st.BackgroundLayer(0).Attachment)
			assert.Equal(t, style.NoRepeat, st.BackgroundLayer(0).Repeat)
			assert.Equal(t, style.BackgroundPosition{X: style.Px(10), Y: style.Px(20)}, st.BackgroundLayer(0).Position)
		},
		"url(a.png) scroll center / 20px 10px repeat-x content-box": func(t *testing.T, st *style.Computed) {
			assert.Equal(t, style.AttachScroll, st.BackgroundLayer(0).Attachment)
			assert.Equal(t, repeatX, st.BackgroundLayer(0).Repeat)
			assert.Equal(t, style.UnderContent, st.BackgroundLayer(0).Origin, "one box is the origin")
			assert.Equal(t, style.UnderContent, st.BackgroundLayer(0).Clip, "and the clip")
			assert.Equal(t, style.BackgroundSize{Width: style.Px(20), Height: style.Px(10)}, st.BackgroundLayer(0).Size)
		},
		"border-box padding-box url(a.png)": func(t *testing.T, st *style.Computed) {
			assert.Equal(t, style.UnderBorder, st.BackgroundLayer(0).Origin, "the first of two boxes is the origin")
			assert.Equal(t, style.UnderPadding, st.BackgroundLayer(0).Clip, "the second the clip")
		},
		"url(a.png) repeat no-repeat": func(t *testing.T, st *style.Computed) {
			assert.Equal(t, repeatX, st.BackgroundLayer(0).Repeat, "two words: across, then down")
		},
		"url(a.png) top left no-repeat, url(b.png) fixed, blue": func(t *testing.T, st *style.Computed) {
			require.Equal(t, 3, st.BackgroundLayers(), "the colour alone is a layer of none")
			top, under := st.BackgroundLayer(0), st.BackgroundLayer(1)
			assert.Equal(t, "a.png", top.Image.Source, "the first layer is on top")
			assert.Equal(t, style.NoRepeat, top.Repeat)
			assert.Equal(t, style.AttachScroll, top.Attachment)
			assert.Equal(t, "b.png", under.Image.Source)
			assert.Equal(t, style.BackgroundRepeat{}, under.Repeat, "each layer resets what it leaves out")
			assert.Equal(t, style.AttachFixed, under.Attachment)
			assert.Equal(t, color.RGB(0, 0, 1), st.Background, "the colour is the last layer's")
		},
		"none, url(b.png) fixed": func(t *testing.T, st *style.Computed) {
			require.Equal(t, 2, st.BackgroundLayers())
			assert.Empty(t, st.BackgroundLayer(0).Image.Source, "a layer of none is still a layer")
			assert.Equal(t, "b.png", st.BackgroundLayer(1).Image.Source)
			assert.Equal(t, style.AttachFixed, st.BackgroundLayer(1).Attachment)
		},
		"url(a.png) space round, linear-gradient(red, blue) content-box padding-box": func(t *testing.T,
			st *style.Computed) {
			assert.Equal(t, style.BackgroundRepeat{X: style.TileSpace, Y: style.TileRound}, st.BackgroundLayer(0).Repeat)
			under := st.BackgroundLayer(1)
			assert.NotNil(t, under.Image.Gradient)
			assert.Equal(t, style.UnderContent, under.Origin)
			assert.Equal(t, style.UnderPadding, under.Clip)
		},
	}
	for value, check := range cases {
		t.Run(value, func(t *testing.T) {
			e := el(dom.Div, "e")
			Compute(page(e), []*Sheet{sheet(t, `#e { background-attachment: fixed; background-clip: content-box;
background: `+value+` }`)})
			check(t, styleOf(e))
		})
	}
	e := el(dom.Div, "e")
	Compute(page(e), []*Sheet{sheet(t, `#e { background-attachment: fixed; background-clip: content-box;
background: red }`)})
	assert.Equal(t, style.AttachScroll, styleOf(e).BackgroundLayer(0).Attachment,
		"the shorthand resets what it leaves out")
	assert.Equal(t, style.InitialBox, styleOf(e).BackgroundLayer(0).Clip)
}

func TestABackgroundThatIsNoneIsRefusedWhole(t *testing.T) {
	for _, value := range []string{"red blue", "url(a.png) fixed local", "red, url(a.png)", "url(a.png) / 10px", "wavy",
		"url(a.png),", ", url(a.png)", "url(a.png) red, url(b.png)"} {
		_, err := expand("background", items(componentsOf(value)))
		assert.Error(t, err, value)
	}
}

//nolint:gochecknoglobals // a test value
var repeatX = style.BackgroundRepeat{X: style.TileRepeat, Y: style.TileNoRepeat}

func TestTheBackgroundLonghandsTakeLists(t *testing.T) {
	e := el(dom.Div, "e")
	Compute(page(e), []*Sheet{sheet(t, `#e {
background-image: url(a.png), linear-gradient(red, blue), none;
background-position: right 4px top, 10px 20px;
background-size: cover, 4px auto, contain;
background-repeat: repeat-y, space, round no-repeat;
background-attachment: fixed, local;
background-origin: content-box, border-box;
background-clip: padding-box;
}`)})
	st := styleOf(e)
	require.Equal(t, 3, st.BackgroundLayers())
	assert.Equal(t, "a.png", st.BackgroundLayer(0).Image.Source)
	assert.NotNil(t, st.BackgroundLayer(1).Image.Gradient)
	assert.Equal(t, style.BackgroundPosition{X: style.Px(10), Y: style.Px(20)}, st.BackgroundLayer(1).Position)
	assert.Equal(t, st.BackgroundLayer(0).Position, st.BackgroundLayer(2).Position, "the list starts again")
	assert.Equal(t, style.BackgroundSize{Scale: style.ScaleContain}, st.BackgroundLayer(2).Size)
	assert.Equal(t, style.BackgroundRepeat{X: style.TileNoRepeat, Y: style.TileRepeat}, st.BackgroundLayer(0).Repeat)
	assert.Equal(t, style.BackgroundRepeat{X: style.TileSpace, Y: style.TileSpace}, st.BackgroundLayer(1).Repeat)
	assert.Equal(t, style.BackgroundRepeat{X: style.TileRound, Y: style.TileNoRepeat}, st.BackgroundLayer(2).Repeat)
	assert.Equal(t, style.AttachFixed, st.BackgroundLayer(2).Attachment)
	assert.Equal(t, style.UnderBorder, st.BackgroundLayer(1).Origin)
	assert.Equal(t, style.UnderPadding, st.BackgroundLayer(2).Clip)
	for prop, bad := range map[string]string{
		"background-image": "url(a.png),", "background-position": "left, , right", "background-size": "1px, cover 2px",
		"background-repeat": "space round round", "background-clip": "padding-box, text",
		"background-attachment": "fixed local", "background-color": "red, blue",
	} {
		assert.Error(t, validate(prop, componentsOf(bad)), "%s: %s", prop, bad)
	}
}
