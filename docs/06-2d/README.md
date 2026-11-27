# 2D

`twod` draws a 2D game: sprites, text, lines, shapes, tile maps and cameras. Each is an entity like any other:

- you make it with options, such as `twod.NewSprite(tex, twod.At(100, 50))`;
- you add it as a window's child to show it there; one that belongs to no window shows in the primary one;
- you change it while the game runs with its setters.

## Where things are drawn

- With no camera, one unit is one point. The origin is the window's top-left corner, and y grows down.
- A camera with a `Resolution` makes one unit one pixel of that resolution instead, as in [Pixel-perfect](13-pixel-perfect.md).

`At` places an entity's pivot:

| Entity | What `At` places |
|---|---|
| Anything with a size: a sprite, a text, a nine-patch | Its centre by default. `Pivot(0, 0)` makes it the top-left corner, `Pivot(0.5, 1)` the feet. |
| A tile map | The top-left corner of its first cell. |
| A line or a polygon | The origin of its points. |

## Sizing, turning and colouring

| Option | Works on | What it does |
|---|---|---|
| `Size(w, h)` | sprites, nine-patches | Fits it to a box, in points. `Scale` still multiplies it. |
| `Scale`, `Rotation` | any entity | Stretches or turns it, with its children. |
| `Color` | texts, lines, polygons | The colour it draws in. |
| `Tint` | any entity | Multiplies whatever it draws, its children included: flashes, fades, team colours. A white text tinted red shows red. |

## Children

A child is placed from its parent. `Add` makes children that move, turn and leave with their parent. It returns the parent, so a window or a layer is built in one call.

## Draw order

- Layers draw in number order: `twod.NewLayer(1)` is in front of layer 0.
- Within a layer, `Z` orders entities, higher on top; a child's `Z` adds to its parent's. Without `Z`, entities draw in the order they were added.
- `SortByY` draws a parent's children by their y instead, for top-down scenes.
- 2D works alone or over a 3D scene in the same window. Layers draw over the 3D scene, and negative layers behind it, for a backdrop.

## The pages

They go from drawing one thing to whole levels:

1. Drawing: sprites, sprite sheets, lines and shapes, nine-patches.
2. Layers and cameras.
3. Levels made of tiles that the hero collides with.
4. Effects a layer or a camera applies.
5. Motion: easing and paths.
6. Light.
7. Pictures the game paints a piece at a time, and variants of a picture, recoloured or masked, made at load.

Text is a 2D entity too, but fonts have their own section: [Text](../07-text/README.md).
