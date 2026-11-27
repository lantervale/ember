# Tile maps

A tile map builds a level from small square tiles laid out on a grid. Here a meadow with a pond, a sand path and a walled yard is drawn from one sheet of 16 × 16 tiles, written as a picture in text; the pond's water ripples.

The tiles, numbered from 1, left to right then top to bottom:

![The tile sheet: grass, wall, water, ripple, tree, pillar, sand and floor](../../examples/26-tilemap/tiles.png)

![A meadow with a rippling pond, a sand path and a walled yard](tilemap.gif "ember · tile map")

## The level

The level is a picture in text, one character a cell, and a legend that says which tile each character stands for. Naming the tiles with constants keeps the numbers readable.

[level.go](../../examples/26-tilemap/level.go)

## The map

- `twod.NewTileset` cuts the sheet into 16 × 16 tiles.
- `TileAnimation` makes the water alternate with its ripple twice a second.
- `twod.NewTileMap` lays the tiles out with `ASCII(level, legend)`.
- `CellSize` draws each cell 32 points wide, so the pixel art is doubled.

A map's position is the top-left corner of its first cell, so `Pivot` doesn't apply to it.

A map draws only the cells in view: a map a thousand cells wide costs no more than one screen. Isometric maps are the exception: they draw every cell and have no collision.

[main.go](../../examples/26-tilemap/main.go)

## How a tile is drawn

Every tile draws whole, never resized or cut, at its tileset's one scale: the scale at which the tileset's grid tile fits the map's cell width, here 2.

A tile's bottom-left sits on its cell's. So a tile bigger than the grid rises above its cell or spreads to its right, and the map draws the cells whose tiles reach into view, not only those in it ([decision 0034](../99-decisions/0034-every-tile-draws-whole.md)).

## Tilesets with tiles of their own sizes

- `twod.LoadTiledTileset(files, "walls.tsj")` reads a tileset saved by [Tiled](11-tiled.md): a grid, or a collection with a picture per tile or a rectangle of a shared one.
- `twod.MergeTilesets(floor, walls)` puts several together, numbering each set's tiles on from the last one's, so one map can draw from all of them.

What a tileset tells about a tile:

| Method | Answers |
|---|---|
| `tileset.Properties(id)` | its custom properties |
| `tileset.ID("pillar")` | the tile whose `name` property is `pillar` |
| `tileset.TileSize(id)` | its size in pixels |
| `tileset.Picture(id)` | its texture and rectangle in it (the first frame of an animated tile), to draw it elsewhere as a sprite's `Picture` does |
| `tileset.Bounds(id)` | its non-transparent part, in its own pixels |

A map's `Tileset()` is the one it draws from.

## Asking a map where things are

| Method | Answers |
|---|---|
| `Cell(p)` | the cell under a point, and whether it is on the map |
| `Center(x, y)` | the middle of a cell |
| `Picture(x, y)` | where what's drawn in a cell really is, trimmed to the tile's pixels; false for an empty cell |

`Picture` is a tall column's picture rising above its cell, or the part of a flat tile its pixels cover. A game hits what's drawn with it, such as an arrow striking a column's picture rather than its base. An endless map has `Picture` too.

## Endless maps

`twod.NewInfiniteMap(tileset, …)` is a map with no edge, made in chunks.

| Option | What it does |
|---|---|
| `ChunkSize(n)` | A chunk is n cells square, 16 by default. |
| `Generator(func(cx, cy, cells))` | Fills a chunk when it is first needed. It must give a chunk the same cells each time, since the map forgets chunks and makes them again; a seed and the chunk's place make that easy. |
| `KeepChunks(n)` | Past n chunks (64 by default), the map forgets the least recently used. |
| `CellTint(func(x, y) color.Color)` | Colours each cell's tile, white leaving it as it is, such as a cooler floor in one region of the world. It runs once per cell when the chunk is made, after the `Generator`, and the colour stays with the chunk, so drawing costs nothing more. Tall tiles stand untinted. |
| `CellSize`, `TallIn` | As for a tile map. |

When chunks are made:

- Each frame, before anything draws, the map makes the chunks every camera of its window shows, drawn or not, including one drawing `Into` a minimap's texture, and the ring of chunks around them, so tall tiles such as crates stand before they scroll into view.
- Collision makes the chunks it touches too.

Changing it:

- Its solid tiles block in a collision space as `tiles.Of(m)`.
- `Set` edits a cell, whose chunk is then never forgotten.
- `Reset()` forgets every chunk, edited ones and their tall tiles too, so the next draw or query makes them again from the `Generator` and `CellTint` as they read now. Call it from a system after changing what the generator reads, never while drawing.
- Both change `Version`, so routes and flow fields plan again.

```go
floor := twod.NewInfiniteMap(tiles, twod.CellSize(32, 32), twod.Generator(func(cx, cy int, cells [][]int) {
	r := rand.New(rand.NewPCG(seed, uint64(cx)<<32|uint64(uint32(cy))))
	for y := range cells {
		for x := range cells[y] {
			cells[y][x] = stone + r.IntN(3)
		}
	}
}))
```

## Try it

[26-tilemap](../../examples/26-tilemap)
