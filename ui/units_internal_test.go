package ui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/image/font/gofont/gomono"

	"github.com/lantervale/ember/geom"
)

func TestLengthsOfTheViewportFollowIt(t *testing.T) {
	box, plain := Div(Class("half")), Div(Class("plain"))
	d := NewDocument(CSS(`body { margin: 0; } .half { width: 50vw; height: 10vh; } .plain { width: min(100%, 300px); }`))
	d.Body().Append(box, plain)
	d.Layout(geom.V2(400, 300))
	assert.Equal(t, geom.V2(200, 30), geom.V2(box.Box().W, box.Box().H))
	assert.InDelta(t, 300, plain.Box().W, 0.001)

	d.Layout(geom.V2(200, 600))
	assert.Equal(t, geom.V2(100, 60), geom.V2(box.Box().W, box.Box().H))
	assert.InDelta(t, 200, plain.Box().W, 0.001)
}

func TestAViewportThatChangesStylesNothingWhereNoLengthReadsIt(t *testing.T) {
	d := NewDocument(CSS(`div { width: 50%; height: 2em; }`))
	d.Body().Append(Div(), Div())
	d.Layout(geom.V2(400, 300))
	styled := d.trans.Restyled()
	d.Layout(geom.V2(500, 300))
	assert.Equal(t, styled, d.trans.Restyled())
}

func TestLengthsOfAFontFollowTheFontThatLands(t *testing.T) {
	held := make(chan struct{})
	d := NewDocument(FetchFonts(func(string, string) ([]byte, error) {
		<-held
		return gomono.TTF, nil
	}), CSS(`@font-face { font-family: Landing; src: url(landing.ttf); }
div { font: 20px Landing; width: 10ch; height: 5px; }`))
	box := Div(Text("0"))
	d.Body().Append(box)
	d.Layout(geom.V2(400, 300))
	before := box.Box().W
	require.Positive(t, before)

	close(held)
	d.pageFonts.Wait()
	d.Layout(geom.V2(400, 300))
	assert.InDelta(t, 10*0.6*20, box.Box().W, 0.05, "ten zeros of the font that landed")
	assert.NotEqual(t, before, box.Box().W)
}
