# Pixel-perfect

Pixel art looks right when every one of its pixels is a square on screen, the same size as the others. Here the meadow is drawn at 320 × 180 pixels, then scaled up by a whole number to fill as much of the window as it can, with bars around it.

![A pixel-art meadow scaled up whole, with thin bars at the window's edges](pixel-perfect.png "ember · pixel perfect")

## The camera's resolution

`twod.Resolution(320, 180)` on the camera draws the world at that size. It then scales it up by the largest whole factor that fits the window, centred, with the window's background as bars around it, so every art pixel stays square. Resize the window and the factor follows.

| Option or method | What it does |
|---|---|
| `Upscale(twod.Linear)` | Smooths the picture instead of the default `Nearest`. |
| `FitWindow()` | Fills the window at any factor, keeping its shape. |
| `cam.SetResolution(w, h)` | Changes it while the game runs. |
| `cam.SetResolution(0, 0)` | Goes back to the window's own resolution, with no offscreen picture, for a setting the player toggles. `Resolution()` reads 0, 0 then. |
| `cam.ScreenToWorld(p)` | Turns a window point, such as the mouse, into a world one. |

Fixed layers and the UI, such as the hint at the top, draw at the window's own resolution, so text stays smooth.

The scaled world is drawn as one picture at its first layer. So a fixed layer numbered between two world layers ends up beneath it.

[main.go](../../examples/30-pixel-perfect/main.go)

## The level

A tile map from a picture in text, with tiles at their own 16 pixels: the camera does the scaling.

[level.go](../../examples/30-pixel-perfect/level.go)

## As before

The hero walks as in [Sprite sheets](02-sprite-sheets.md). With a camera's resolution, a unit of the world is one art pixel, so he walks 60 of them a second.

[walk.go](../../examples/30-pixel-perfect/walk.go)

## Try it

[30-pixel-perfect](../../examples/30-pixel-perfect)
