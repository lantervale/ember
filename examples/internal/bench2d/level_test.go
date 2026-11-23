package main

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/internal/wm"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/twod/light"
	"github.com/lantervale/ember/world"
	"github.com/stretchr/testify/assert"
)

// A painted level, for what its size costs the CPU: square maps of 32-point
// cells, a third of them solid, with props, lights and tall tiles spread
// over the whole level and a 1280×720 view on its middle. No GPU:
//
//	go test -run xxx -bench Level -benchtime 1s ./examples/internal/bench2d

// level is a painted level's make-up.
type level struct {
	// side is the level's width and height in cells, layers its tile
	// layers, props its still y-sorted sprites, lights its point lights
	// and rocks its Tiled tile objects casting from a strip at their foot.
	side, layers, props, lights, rocks int
	// tall makes a sixth of the cells tall tiles, a sprite each; lit
	// lights the tile layers per pixel, their solid tiles round what they
	// block with, and around lights those from every side.
	tall, lit, around bool
}

// levelTile is the tile of cell x, y of a layer: 2 and 3 are solid, a
// sixth of the cells each.
func levelTile(x, y, layer int) int {
	switch uint(x*73856093^y*19349663^layer*83492791) % 6 {
	case 0:
		return 2
	case 1:
		return 3
	}
	return 1
}

func levelCells(side, layer int) [][]int {
	g := make([][]int, side)
	for y := range g {
		g[y] = make([]int, side)
		for x := range g[y] {
			g[y][x] = levelTile(x, y, layer)
		}
	}
	return g
}

func blank(w, h int) *twod.Texture { return twod.NewTexture(image.NewNRGBA(image.Rect(0, 0, w, h))) }

// levelTileset is four 16-pixel tiles, 2 and 3 solid, 3 tall when asked,
// both lit from every side when asked.
func levelTileset(tall, around bool) *twod.Tileset {
	opts := []twod.TilesetOption{twod.TileValue(2, "solid", true), twod.TileValue(3, "solid", true)}
	if tall {
		opts = append(opts, twod.Tall(3))
	}
	if around {
		opts = append(opts, light.TileLitAround(2, 3))
	}
	return twod.NewTileset(blank(64, 16), twod.Cell(16, 16), opts...)
}

// build is l in a World without a GPU, and its first tile layer.
func (l level) build() (*world.World, *twod.TileMap) {
	w := world.New().AddPlugins(wm.Plugin{}, twod.Plugin{}, light.Plugin{})
	win := platform.NewWindow(platform.Size(1280, 720))
	cam := twod.NewCamera(twod.Zoom(1))
	cam.SetPosition(geom.V2(float32(l.side)*16, float32(l.side)*16))
	if l.lights > 0 {
		light.Ambient(color.RGB(0.1, 0.1, 0.15))(cam)
	}
	ts := levelTileset(l.tall, l.around)
	ground, things := twod.NewLayer(0), twod.NewLayer(1, twod.SortByY())
	if l.lit {
		light.Lit()(ground)
	}
	var first *twod.TileMap
	for i := range l.layers {
		m := twod.NewTileMap(ts, twod.Cells(levelCells(l.side, i)), twod.CellSize(32, 32), twod.TallIn(things))
		ground.Add(m)
		if first == nil {
			first = m
		}
	}
	at := func(i, a, b int) geom.Vec2 {
		span := uint(l.side * 32)
		return geom.V2(float32(uint(i*2654435761+a)%span), float32(uint(i*40503+b)%span))
	}
	prop := blank(32, 48)
	for i := range l.props {
		things.Add(twod.NewSprite(prop, twod.At(at(i, 0, 7).X, at(i, 0, 7).Y)))
	}
	if l.rocks > 0 {
		rocks, err := twod.LoadTiled(rockFile(l.rocks, l.side*16), "rocks.tmj", twod.CellSize(32, 32),
			light.TileProperties())
		if err != nil {
			panic(err)
		}
		for _, o := range rocks.Objects("rocks") {
			things.Add(rocks.Sprite(o))
		}
	}
	for i := range l.lights {
		lamp := light.NewPoint(200, color.RGB(1, 0.8, 0.5))
		lamp.SetPosition(at(i, 13, 101))
		things.Add(lamp)
	}
	win.Add(cam, ground, things)
	w.AddEntities(win)
	w.RunFrames(3)
	return w, first
}

// BenchmarkLevelFrame is a frame of a level: its tile layers alone on a
// small and a large level, then the large one with props, lights or tall
// tiles. Only what is in view should count.
func BenchmarkLevelFrame(b *testing.B) {
	for _, c := range []struct {
		name string
		l    level
	}{
		{"40x40", level{side: 40, layers: 4}},
		{"400x400", level{side: 400, layers: 4}},
		{"400x400-20000props", level{side: 400, layers: 4, props: 20000}},
		{"400x400-300lights", level{side: 400, layers: 4, lights: 300}},
		{"400x400-300lights-lit", level{side: 400, layers: 4, lights: 300, lit: true}},
		{"400x400-300lights-lit-around", level{side: 400, layers: 4, lights: 300, lit: true, around: true}},
		{"400x400-tall", level{side: 400, layers: 1, tall: true}},
		{"400x400-300lights-2000rocks", level{side: 400, layers: 4, lights: 300, rocks: 2000}},
	} {
		b.Run(c.name, func(b *testing.B) {
			w, _ := c.l.build()
			b.ReportAllocs()
			for b.Loop() {
				w.Frame(1.0 / 60)
			}
		})
	}
}

// BenchmarkLevelSet is a cell set during play, as a floor that crumbles:
// the call alone, then with the lit frame after it.
func BenchmarkLevelSet(b *testing.B) {
	for _, side := range []int{40, 400} {
		for _, tall := range []bool{false, true} {
			b.Run(fmt.Sprintf("%dx%d-tall=%v", side, side, tall), func(b *testing.B) {
				_, m := level{side: side, layers: 1, tall: tall}.build()
				i := 0
				for b.Loop() {
					m.Set(i%side, i*7%side, 1+i%3)
					i++
				}
			})
		}
		b.Run(fmt.Sprintf("%dx%d-then-a-lit-frame", side, side), func(b *testing.B) {
			w, m := level{side: side, layers: 1, lights: 12}.build()
			i := 0
			for b.Loop() {
				m.Set(i%side, i*7%side, 1+i%3)
				w.Frame(1.0 / 60)
				i++
			}
		})
	}
}

// levelFile is a level of side cells as a Tiled map: its tile layers as
// plain arrays over two tilesets, and an object layer of tile objects.
func levelFile(side, layers, objects int) fstest.MapFS {
	var sb strings.Builder
	fmt.Fprintf(&sb, `{"orientation":"orthogonal","width":%d,"height":%d,"tilewidth":16,"tileheight":16,`, side, side)
	sb.WriteString(`"tilesets":[{"firstgid":1,"name":"t","image":"t.png","tilewidth":16,"tileheight":16},` +
		`{"firstgid":100,"name":"u","image":"u.png","tilewidth":16,"tileheight":16}],"layers":[`)
	for l := range layers {
		fmt.Fprintf(&sb, `{"type":"tilelayer","id":%d,"name":"l%d","width":%d,"height":%d,"data":[`, l+1, l, side, side)
		for i := range side * side {
			if i > 0 {
				sb.WriteByte(',')
			}
			g := levelTile(i%side, i/side, l)
			if l == 1 && i%5 == 0 {
				g = 100
			}
			fmt.Fprintf(&sb, "%d", g)
		}
		sb.WriteString("]},")
	}
	sb.WriteString(`{"type":"objectgroup","id":99,"name":"things","objects":[`)
	for i := range objects {
		if i > 0 {
			sb.WriteByte(',')
		}
		fmt.Fprintf(&sb, `{"id":%d,"gid":%d,"x":%d,"y":%d,"width":16,"height":16,`+
			`"properties":[{"name":"kind","type":"string","value":"torch"}]}`,
			i+1, 1+i%4, i*37%(side*16), i*91%(side*16))
	}
	sb.WriteString("]}]}")
	picture := func(w, h int) []byte {
		var buf bytes.Buffer
		_ = png.Encode(&buf, image.NewNRGBA(image.Rect(0, 0, w, h)))
		return buf.Bytes()
	}
	return fstest.MapFS{
		"level.tmj": {Data: []byte(sb.String())},
		"t.png":     {Data: picture(64, 16)},
		"u.png":     {Data: picture(64, 16)},
	}
}

// rockFile is a map of n rock tile objects spread over span pixels: a
// tall tile blocking light with a strip at its foot, 24 pixels high.
func rockFile(n, span int) fstest.MapFS {
	var sb strings.Builder
	sb.WriteString(`{"orientation":"orthogonal","width":1,"height":1,"tilewidth":16,"tileheight":16,` +
		`"tilesets":[{"firstgid":1,"name":"r","image":"r.png","tilewidth":16,"tileheight":16,"tiles":[{"id":0,` +
		`"properties":[{"name":"tall","type":"bool","value":true},{"name":"height","type":"float","value":24}],` +
		`"objectgroup":{"type":"objectgroup","objects":[{"id":1,"type":"shadow","x":2,"y":12,"width":12,"height":4}]}}]}],` +
		`"layers":[{"type":"objectgroup","id":1,"name":"rocks","objects":[`)
	for i := range n {
		if i > 0 {
			sb.WriteByte(',')
		}
		fmt.Fprintf(&sb, `{"id":%d,"gid":1,"x":%d,"y":%d,"width":16,"height":16}`, i+1, i*53%span, i*97%span)
	}
	sb.WriteString("]}]}")
	var buf bytes.Buffer
	_ = png.Encode(&buf, image.NewNRGBA(image.Rect(0, 0, 16, 16)))
	return fstest.MapFS{"rocks.tmj": {Data: []byte(sb.String())}, "r.png": {Data: buf.Bytes()}}
}

// BenchmarkLevelLoad is twod.LoadTiled of a level's file, its pictures
// apart: four tile layers and a tile object for every 80 cells.
func BenchmarkLevelLoad(b *testing.B) {
	for _, side := range []int{100, 400} {
		b.Run(fmt.Sprintf("%dx%d", side, side), func(b *testing.B) {
			fsys := levelFile(side, 4, side*side/80)
			b.SetBytes(int64(len(fsys["level.tmj"].Data)))
			b.ReportAllocs()
			for b.Loop() {
				if _, err := twod.LoadTiled(fsys, "level.tmj"); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// setCost is how long a Set and a SetFlip of a cell take on a level of side
// cells, averaged over at least calls of each and a tenth of a second, so
// a coarse clock (Windows') can't read it as nothing, or over those two
// seconds allow. A tall tile is set to itself and a plain one to another
// plain one, so no tall tile's sprite comes or goes: BenchmarkLevelSet
// times those.
func setCost(side int, tall bool, calls int) time.Duration {
	_, m := level{side: side, layers: 1, tall: tall}.build()
	began := time.Now()
	done := 0
	for (done < calls || time.Since(began) < 100*time.Millisecond) && time.Since(began) < 2*time.Second {
		x, y := done%side, done*7%side
		id := 1 + done%2
		if tall && m.At(x, y) == 3 {
			id = 3
		}
		m.Set(x, y, id)
		m.SetFlip(done*3%side, done%side, twod.TileFlip(done%2))
		done++
	}
	return time.Since(began) / time.Duration(done)
}

// TestSettingACellCostsTheSameOnALargeLevel holds TileMap.Set and SetFlip
// to the cell they change: on a level a hundred times the cells they may
// take ten times as long at most, where walking the map took a hundred
// times, two hundred with tall tiles. A tall tile coming or going is not
// held: taking its sprite out of its parent looks through the parent's
// children (node's detach), 26,700 on the large level.
func TestSettingACellCostsTheSameOnALargeLevel(t *testing.T) {
	if testing.Short() {
		t.Skip("times thousands of calls")
	}
	for _, tall := range []bool{false, true} {
		small, large := setCost(40, tall, 20000), setCost(400, tall, 20000)
		t.Logf("tall tiles %v: %s a cell on 40×40, %s on 400×400", tall, small, large)
		assert.Positive(t, small, "tall tiles %v: 40×40 timed", tall)
		assert.LessOrEqual(t, large, 10*small, "tall tiles %v: 400×400 against 40×40", tall)
	}
}
