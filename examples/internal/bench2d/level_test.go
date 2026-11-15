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
	// layers, props its still y-sorted sprites and lights its point lights.
	side, layers, props, lights int
	// tall makes a sixth of the cells tall tiles, a sprite each.
	tall bool
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

// levelTileset is four 16-pixel tiles, 2 and 3 solid, 3 tall when asked.
func levelTileset(tall bool) *twod.Tileset {
	opts := []twod.TilesetOption{twod.TileValue(2, "solid", true), twod.TileValue(3, "solid", true)}
	if tall {
		opts = append(opts, twod.Tall(3))
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
	ts := levelTileset(l.tall)
	ground, things := twod.NewLayer(0), twod.NewLayer(1, twod.SortByY())
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
		{"400x400-tall", level{side: 400, layers: 1, tall: true}},
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
// cells, averaged over calls of each, or over those two seconds allow.
func setCost(side int, tall bool, calls int) time.Duration {
	_, m := level{side: side, layers: 1, tall: tall}.build()
	began := time.Now()
	done := 0
	for done < calls && time.Since(began) < 2*time.Second {
		m.Set(done%side, done*7%side, 1+done%3)
		m.SetFlip(done*3%side, done%side, twod.TileFlip(done%2))
		done++
	}
	return time.Since(began) / time.Duration(done)
}

// TestSettingACellCostsTheSameOnALargeLevel holds TileMap.Set and SetFlip
// to the cell they change: on a level a hundred times the cells, a plain
// tile may take ten times as long at most, where walking the map took a
// hundred. With tall tiles it may take fifty: a tall tile set to a plain
// one takes its sprite out of its parent, which looks through the parent's
// children (node's detach), 26,700 here; walking the map took two hundred.
func TestSettingACellCostsTheSameOnALargeLevel(t *testing.T) {
	if testing.Short() {
		t.Skip("times thousands of calls")
	}
	for _, c := range []struct {
		tall bool
		most int
	}{{false, 10}, {true, 50}} {
		small, large := setCost(40, c.tall, 20000), setCost(400, c.tall, 20000)
		t.Logf("tall tiles %v: %s a cell on 40×40, %s on 400×400", c.tall, small, large)
		assert.LessOrEqual(t, large, time.Duration(c.most)*small, "tall tiles %v: 400×400 against 40×40", c.tall)
	}
}
