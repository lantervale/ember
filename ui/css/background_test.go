package css //nolint:testpackage // white-box: reads computed styles

import (
	"testing"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
	"github.com/stretchr/testify/assert"
)

func TestBackgroundAttachmentOriginAndClip(t *testing.T) {
	a, b, c := el(dom.Div, "a"), el(dom.Div, "b"), el(dom.Div, "c")
	Compute(page(a, b, c), []*Sheet{sheet(t, `
#a { background-attachment: fixed; background-origin: content-box; background-clip: padding-box }
#b { background-attachment: local; background-origin: border-box; background-clip: content-box }
#c { background-attachment: scroll; background-origin: padding-box; background-clip: border-box }`)})
	st := styleOf(a)
	assert.Equal(t, style.AttachFixed, st.BackgroundAttachment)
	assert.Equal(t, style.UnderContent, st.BackgroundOrigin)
	assert.Equal(t, style.UnderPadding, st.BackgroundClip)
	st = styleOf(b)
	assert.Equal(t, style.AttachLocal, st.BackgroundAttachment)
	assert.Equal(t, style.UnderBorder, st.BackgroundOrigin)
	assert.Equal(t, style.UnderContent, st.BackgroundClip)
	st = styleOf(c)
	assert.Equal(t, style.AttachScroll, st.BackgroundAttachment)
	assert.Equal(t, style.InitialBox, st.BackgroundOrigin, "the padding box is where a picture starts")
	assert.Equal(t, style.InitialBox, st.BackgroundClip, "the border box is what a background fills")
}

func TestTheBackgroundShorthandTakesItsPartsInAnyOrder(t *testing.T) {
	cases := map[string]func(t *testing.T, st *style.Computed){
		"fixed url(a.png) red": func(t *testing.T, st *style.Computed) {
			assert.Equal(t, style.AttachFixed, st.BackgroundAttachment)
			assert.Equal(t, "a.png", st.BackgroundImage.Source)
			assert.Equal(t, color.RGB(1, 0, 0), st.Background)
		},
		"red no-repeat local 10px 20px url(a.png)": func(t *testing.T, st *style.Computed) {
			assert.Equal(t, style.AttachLocal, st.BackgroundAttachment)
			assert.True(t, st.BackgroundNoRepeat)
			assert.Equal(t, style.BackgroundPosition{X: style.Px(10), Y: style.Px(20)}, st.BackgroundPosition)
		},
		"url(a.png) scroll center / 20px 10px repeat-x content-box": func(t *testing.T, st *style.Computed) {
			assert.Equal(t, style.AttachScroll, st.BackgroundAttachment)
			assert.Equal(t, style.RepeatX, st.BackgroundRepeat)
			assert.Equal(t, style.UnderContent, st.BackgroundOrigin, "one box is the origin")
			assert.Equal(t, style.UnderContent, st.BackgroundClip, "and the clip")
			assert.Equal(t, style.BackgroundSize{Width: style.Px(20), Height: style.Px(10)}, st.BackgroundSize)
		},
		"border-box padding-box url(a.png)": func(t *testing.T, st *style.Computed) {
			assert.Equal(t, style.UnderBorder, st.BackgroundOrigin, "the first of two boxes is the origin")
			assert.Equal(t, style.UnderPadding, st.BackgroundClip, "the second the clip")
		},
		"url(a.png) repeat no-repeat": func(t *testing.T, st *style.Computed) {
			assert.False(t, st.BackgroundNoRepeat)
			assert.Equal(t, style.RepeatX, st.BackgroundRepeat, "two words: across, then down")
		},
		"url(a.png) top left no-repeat, url(b.png) fixed, blue": func(t *testing.T, st *style.Computed) {
			assert.Equal(t, "a.png", st.BackgroundImage.Source, "the layer on top is the one drawn")
			assert.True(t, st.BackgroundNoRepeat)
			assert.Equal(t, style.AttachScroll, st.BackgroundAttachment)
			assert.Equal(t, color.RGB(0, 0, 1), st.Background, "the colour is the last layer's")
		},
		"none, url(b.png) fixed": func(t *testing.T, st *style.Computed) {
			assert.Equal(t, "b.png", st.BackgroundImage.Source, "the first layer with a picture")
			assert.Equal(t, style.AttachFixed, st.BackgroundAttachment)
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
	assert.Equal(t, style.AttachScroll, styleOf(e).BackgroundAttachment, "the shorthand resets what it leaves out")
	assert.Equal(t, style.InitialBox, styleOf(e).BackgroundClip)
}

func TestABackgroundThatIsNoneIsRefusedWhole(t *testing.T) {
	for _, value := range []string{"red blue", "url(a.png) fixed local", "red, url(a.png)", "url(a.png) / 10px", "wavy"} {
		_, err := expand("background", items(componentsOf(value)))
		assert.Error(t, err, value)
	}
}
