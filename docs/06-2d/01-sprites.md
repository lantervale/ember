# Sprites

A sprite draws a picture. Here one 16 × 16 pixel-art crate is drawn three times: as it is, turned, and mirrored with a green tint.

![Three crates: plain, turned, and mirrored in green](sprites.png "ember · sprites")

## The program

`twod.LoadTexture` reads the crate from the example's own files, embedded with `go:embed`. `twod.Pixelated()` keeps its pixels sharp when it is scaled up.

Each sprite is made with options:

- `At` places its centre.
- `Scale` and `Rotation` size and turn it. At `Scale(1, 1)` one texture pixel is one point, so the 16-pixel crate needs `Scale(6, 6)` to be 96 points wide.
- `Tint` colours it.
- `Flip`, a sprite's own option, mirrors it.

The sprites are the window's children, so they show in it.

[main.go](../../examples/19-sprites/main.go)

## Hiding a sprite

- `Hidden()` keeps an entity from drawing, and its children with it.
- `SetChildrenHidden(true)` draws an entity without its children. They keep their own `Hidden`, as a folded UI panel does with what's inside it.

## Other colours and shapes

The same picture in other colours, or cut to a shape, is a new texture made at load: see [Recolours and masks](21-recolours-and-masks.md).

## Try it

[19-sprites](../../examples/19-sprites)
