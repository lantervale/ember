# A 2D camera

A camera shows a level wider than the window and follows the hero along it. Here the far hills scroll at half the speed of the ground, for depth, and a line of help stays put in the corner.

![The camera following the hero right, then left, with hills scrolling slower behind](follow.gif "ember · 2D camera")

## Layers and the camera

The level stands on three layers:

| Layer | Holds | Option |
|---|---|---|
| -1 | the hills, behind | `Parallax(0.5)`: moves at half the camera's speed |
| 0 | the ground and the hero | |
| 1 | the help, in front | `Fixed()`: stays where it is whatever the camera does |

`twod.NewCamera` takes:

- `Follow(hero)`: the camera follows the hero;
- `Smoothing`: it catches up with him rather than snapping;
- `Bounds`: it never shows outside the level. The level is as tall as the window, so the camera only scrolls sideways.

The camera follows where the hero is drawn. When he moves in fixed steps, that is up to a step behind his `Position`, so a test that reads both sees a small gap.

[main.go](../../examples/23-2d-camera/main.go)

## The level

The ground is a polygon the whole level wide, and the hero stands on it.

Parallax works on both axes. A layer at half speed also shows lower than the rest when the view is centred below its origin, so the hills are drawn that much higher to sit on the horizon.

[level.go](../../examples/23-2d-camera/level.go)

## As before

The hero walks as in [Sprite sheets](02-sprite-sheets.md).

[walk.go](../../examples/23-2d-camera/walk.go)

## When the window resizes

This window has a fixed size. In one that can be resized:

- By default, a camera shows more of the world as the window grows, at the same scale.
- `twod.ViewSize(w, h)` keeps at least that much world in view, scaling to fit. `ViewSize(0, 360)` always shows 360 units of height and lets the width follow.
- A camera with a `Resolution` scales by whole factors only, as on the [Pixel-perfect](13-pixel-perfect.md) page, and ignores `ViewSize`.
- A fixed layer's `Anchor(x, y)` puts its origin at that fraction of the window. With `Anchor(1, 0)`, a HUD at `At(-20, 20)` stays 20 points from the top-right corner.
- A camera's `Layers(…)` makes it draw only those layers, as a minimap does. Fixed layers and the UI still draw.

## Nudging the view

`twod.Offset(x, y)`, or `SetOffset` while the game runs, moves what the camera shows by that much of the world.

- It applies last, after `Follow`, `Smoothing` and `Bounds`, so it may show past the bounds.
- It never feeds back into the follow: setting it back to 0, 0 puts the view where following alone would have it.
- `ScreenToWorld` and `Center` include it.

A camera shake is game code built on it: when the hero is hurt, a system sets a small random offset each frame for a moment, then 0, 0.

## Try it

[23-2d-camera](../../examples/23-2d-camera)
