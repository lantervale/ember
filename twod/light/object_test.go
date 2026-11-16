package light //nolint:testpackage // beside light's other tests

import (
	"bytes"
	"fmt"
	"image"
	imagepng "image/png"
	"testing"
	"testing/fstest"

	"github.com/lantervale/ember/collide/tiles"
	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/internal/draw"
	"github.com/lantervale/ember/twod"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// objectTiles are four 16×16 tiles: a tall rock casting from a strip at
// its foot, 24 pixels high, in occluder group 1; a solid wall casting its
// whole cell; a plain floor casting nothing; a solid wall top, lit from
// every side.
const objectTiles = `{"name": "set", "firstgid": 1, "image": "tiles.png", "imagewidth": 64, "imageheight": 16,
	"tilewidth": 16, "tileheight": 16, "columns": 4, "tilecount": 4,
	"tiles": [
		{"id": 0, "properties": [
			{"name": "tall", "type": "bool", "value": true},
			{"name": "height", "type": "float", "value": 24},
			{"name": "occluder_group", "type": "int", "value": 1}],
		 "objectgroup": {"type": "objectgroup", "objects": [
			{"id": 1, "type": "shadow", "x": 2, "y": 12, "width": 8, "height": 4}]}},
		{"id": 1, "properties": [{"name": "solid", "type": "bool", "value": true}]},
		{"id": 3, "properties": [
			{"name": "solid", "type": "bool", "value": true},
			{"name": "lit", "type": "string", "value": "around"}]}
	]}`

// objectFS is a 4×4 map, one.tmj, with tile gid on cell 2, 2 of its tile
// layer and, standing on that same cell, a tile object of it.
func objectFS(t *testing.T, gid uint32) fstest.MapFS {
	t.Helper()
	var png bytes.Buffer
	require.NoError(t, imagepng.Encode(&png, image.NewRGBA(image.Rect(0, 0, 64, 16))))
	cells := make([]uint32, 16)
	cells[2*4+2] = gid
	data := fmt.Sprint(cells)
	tmj := `{"orientation": "orthogonal", "width": 4, "height": 4, "tilewidth": 16, "tileheight": 16,
		"tilesets": [` + objectTiles + `],
		"layers": [
			{"type": "tilelayer", "name": "tiles", "width": 4, "height": 4, "visible": true, "opacity": 1,
			 "data": [` + commas(data) + `]},
			{"type": "objectgroup", "name": "things", "visible": true, "opacity": 1, "objects": [
				{"id": 7, "gid": ` + fmt.Sprint(gid) + `, "x": 32, "y": 48, "width": 16, "height": 16, "visible": true}]}
		]}`
	return fstest.MapFS{"one.tmj": {Data: []byte(tmj)}, "tiles.png": {Data: png.Bytes()}}
}

// commas is a printed slice's elements as a JSON list's.
func commas(printed string) string {
	out := bytes.ReplaceAll([]byte(printed[1:len(printed)-1]), []byte(" "), []byte(", "))
	return string(out)
}

// litAlone is a frame of e alone on a lit layer, with a lamp: what blocks light
// and how what is drawn reads it.
func litAlone(t *testing.T, e twod.Entity) ([]draw.Vertex, []draw.Vertex) {
	t.Helper()
	w, win, _ := scene(t)
	layer := twod.NewLayer(0, Lit(), twod.SortByY()).Add(e, NewPoint(200, color.White, At(-100, -100)))
	win.Add(layer)
	w.AddEntities(layer)
	w.RunFrames(2)
	l := draw.LightingFor(w, win)
	require.NotNil(t, l)
	return l.Occluders.Vertices, draw.FrontFor(w, win).Vertices
}

func TestATileObjectCastsAndIsLitAsItsTileDoesInAMap(t *testing.T) {
	const flipX = 0x80000000
	for _, c := range []struct {
		name string
		gid  uint32
		lit  int
	}{
		{"a tall rock, from its foot strip, lit round it from every side", 1, 2},
		{"the rock flipped, its strip flipped with it", 1 | flipX, 2},
		{"a solid wall, from its whole cell, lit by its face", 2, 3},
		{"a solid wall top, from its whole cell, lit from every side", 4, 2},
	} {
		load := func() *twod.Tiled {
			m, err := twod.LoadTiled(objectFS(t, c.gid), "one.tmj", twod.CellSize(32, 32), tiles.Properties(),
				TileProperties())
			require.NoError(t, err)
			return m
		}
		inMap, drawnInMap := litAlone(t, load().TileLayers()[0])
		m := load()
		o, _ := m.Object(7)
		asObject, drawnAsObject := litAlone(t, m.Sprite(o))
		require.Len(t, inMap, 4, c.name)
		assert.Equal(t, inMap, asObject, "%s: casts the same", c.name)
		require.Len(t, drawnAsObject, 4, c.name)
		require.Len(t, drawnInMap, 4, c.name)
		for i, v := range drawnAsObject {
			assert.Equal(t, c.lit, v.Lit(), c.name)
			assert.InDelta(t, drawnInMap[i].LightUV.X, v.LightUV.X, 1e-5, "%s: lit round the same point", c.name)
			assert.InDelta(t, drawnInMap[i].LightUV.Y, v.LightUV.Y, 1e-5, c.name)
			assert.InDelta(t, drawnInMap[i].LightSpread.X, v.LightSpread.X, 1e-5, "%s: as far round it", c.name)
			assert.InDelta(t, drawnInMap[i].LightSpread.Y, v.LightSpread.Y, 1e-5, c.name)
		}
	}
	m, err := twod.LoadTiled(objectFS(t, 1), "one.tmj", twod.CellSize(32, 32), tiles.Properties(), TileProperties())
	require.NoError(t, err)
	o, _ := m.Object(7)
	occluders, _ := litAlone(t, m.Sprite(o))
	assert.Equal(t, draw.OccluderColor(0b10, 24*2*2), occluders[0].Color,
		"in group 1, 24 of its pixels high, two points each, zoomed twice")
}

func TestATileObjectCastsNothingWithNoShadowsOrATileThatCastsNothing(t *testing.T) {
	m, err := twod.LoadTiled(objectFS(t, 1), "one.tmj", twod.CellSize(32, 32), tiles.Properties(), TileProperties())
	require.NoError(t, err)
	o, _ := m.Object(7)
	occluders, _ := litAlone(t, m.Sprite(o, NoShadows()))
	assert.Empty(t, occluders, "NoShadows, as on a map")
	m, err = twod.LoadTiled(objectFS(t, 3), "one.tmj", twod.CellSize(32, 32), tiles.Properties(), TileProperties())
	require.NoError(t, err)
	o, _ = m.Object(7)
	occluders, drawn := litAlone(t, m.Sprite(o))
	assert.Empty(t, occluders, "a floor tile casts nothing")
	for _, v := range drawn {
		assert.Equal(t, 1, v.Lit(), "and is lit per pixel, as before")
	}
}
