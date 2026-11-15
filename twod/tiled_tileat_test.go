package twod //nolint:testpackage // white-box: shares twod's test helpers

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// gapped is a map of two tilesets with a gap between their first ids (the
// floor's four tiles from 3, the walls' two from 10): a layer of both, a
// layer of the walls alone, and layers of no tiles.
func gapped(t *testing.T) *Tiled {
	t.Helper()
	tmj := `{"orientation": "orthogonal", "tilewidth": 16, "tileheight": 16,
	 "tilesets": [{"firstgid": 3, "name": "floor", "image": "floor.png", "tilewidth": 16, "tileheight": 16},
	   {"firstgid": 10, "name": "walls", "image": "walls.png", "tilewidth": 16, "tileheight": 16}],
	 "layers": [
	   {"type": "tilelayer", "name": "both", "width": 4, "height": 1, "data": [4, 0, 11, 2147483658]},
	   {"type": "tilelayer", "name": "walls", "width": 2, "height": 1, "data": [0, 11]},
	   {"type": "objectgroup", "name": "things", "objects": []},
	   {"type": "group", "name": "rooms", "layers": []}]}`
	m, err := LoadTiled(fstest.MapFS{
		"map.tmj":   {Data: []byte(tmj)},
		"floor.png": {Data: pngOf(t, 32, 32)},
		"walls.png": {Data: pngOf(t, 32, 16)},
	}, "map.tmj")
	require.NoError(t, err)
	return m
}

func TestACellOfATileLayerGivesItsOwnTilesetAndLocalID(t *testing.T) {
	m := gapped(t)
	floor, walls := m.Tilesets()[0], m.Tilesets()[1]
	both, _ := m.Layer("both")
	require.Empty(t, both.TileMap().Tileset().Source(), "a layer of two tilesets draws from the merged one")
	for _, c := range []struct {
		name string
		x, y int
		want TiledTile
	}{
		{"the first tileset's", 0, 0, TiledTile{GID: 4, Tileset: floor, ID: 2}},
		{"the second's, past the gap", 2, 0, TiledTile{GID: 11, Tileset: walls, ID: 2}},
		{"a flipped one", 3, 0, TiledTile{GID: 2147483658, Tileset: walls, ID: 1, Flip: TileFlipX}},
	} {
		got, ok := both.TileAt(c.x, c.y)
		assert.True(t, ok, c.name)
		assert.Equal(t, c.want, got, c.name)
	}

	alone, _ := m.Layer("walls")
	require.Same(t, walls, alone.TileMap().Tileset(), "a layer of one tileset draws from it")
	got, ok := alone.TileAt(1, 0)
	assert.True(t, ok)
	assert.Equal(t, TiledTile{GID: 11, Tileset: walls, ID: 2}, got, "the same on a layer of one tileset")
}

func TestAnEmptyCellACellOutsideAndALayerOfNoTilesGiveNoTile(t *testing.T) {
	m := gapped(t)
	for _, c := range []struct {
		name, layer string
		x, y        int
	}{
		{"an empty cell", "both", 1, 0},
		{"an empty cell of a layer of one tileset", "walls", 0, 0},
		{"left of the map", "both", -1, 0},
		{"right of the map", "both", 4, 0},
		{"under the map", "both", 0, 1},
		{"an object layer", "things", 0, 0},
		{"a group", "rooms", 0, 0},
	} {
		l, ok := m.Layer(c.layer)
		require.True(t, ok, c.name)
		got, ok := l.TileAt(c.x, c.y)
		assert.False(t, ok, c.name)
		assert.Equal(t, TiledTile{}, got, c.name)
	}
}

func TestACellSetAfterLoadingGivesTheTileItHoldsNow(t *testing.T) {
	m := gapped(t)
	walls := m.Tilesets()[1]
	both, _ := m.Layer("both")
	both.TileMap().Set(1, 0, 10)
	both.TileMap().SetFlip(1, 0, TileFlipY|TileFlipDiagonal)
	got, ok := both.TileAt(1, 0)
	assert.True(t, ok)
	turned := TileFlipY | TileFlipDiagonal
	assert.Equal(t, TiledTile{GID: 10 | tiledFlipY | tiledFlipDiag, Tileset: walls, ID: 1, Flip: turned}, got,
		"a merged layer's cell is set by global id")

	both.TileMap().Set(1, 0, 2)
	_, ok = both.TileAt(1, 0)
	assert.False(t, ok, "a global id under the map's first is no tile")

	alone, _ := m.Layer("walls")
	alone.TileMap().Set(0, 0, 1)
	got, ok = alone.TileAt(0, 0)
	assert.True(t, ok)
	assert.Equal(t, TiledTile{GID: 10, Tileset: walls, ID: 1}, got, "a layer of one tileset's by that tileset's id")
}
