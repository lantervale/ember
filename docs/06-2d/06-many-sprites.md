# Many sprites

A bunnymark: hundreds of bunnies falling and bouncing around the window, and a hundred more every frame while Space or the mouse button is held. The counter shows how many there are and the frame rate, so you can see how far your machine goes.

![Bunnies bouncing, and more pouring in from the corner](bunnies.gif "ember · many sprites")

## The program

The bunnies are drawn on one layer and the counter on a layer above it, so the counter stays readable. All the bunnies share one texture.

[main.go](../../examples/24-many-sprites/main.go)

## Bunnies

`Bunnies` adds them. `Add` makes each sprite a child of the field layer, and since the layer is already in the World, the sprite joins it too.

Every frame, each bunny:

1. falls with gravity;
2. moves by its speed with `Translate`;
3. bounces off the edges through `Position` and `SetPosition`.

`f.World.Stats().FPS` is the frame rate.

[bunnies.go](../../examples/24-many-sprites/bunnies.go)

## Measuring a frame with the profiler

F3 in any ember window opens the profiler. `twod.ShowProfiler(w, on)` is the same switch from code, so a game's own debug menu and F3 agree, and `twod.ProfilerShown(w)` reads it.

It shows a frame graph, the entity count and the slowest systems, and these lines:

| Line | What it says |
|---|---|
| `work  1.60ms  (~625 fps possible)` | The frame's own time, its tick and render without the wait for the display, and the frame rate that would allow. Under vsync the frame rate is the display's, so this is where the room to spare shows. `Stats().Work()` is the same time. |
| `2D draws N  binds N  verts Nk  passes N` | The last frame's 2D: draws are the GPU calls, binds the texture changes between them, verts the corners drawn, and passes the pictures drawn first offscreen (cameras with a `Resolution` or `Into` a texture, masks and blended layers). In code: `Stats().Draws2D`, `Binds2D`, `Vertices2D` and `Passes2D`. |
| `recorded` | Below the slowest systems: what no system covers. Collecting and drawing the 2D, and the GPU's own time for the window's 2D, the scene and each offscreen pass, named after its layer or camera: `twod.Name("light")` on a lighting layer shows `gpu pass light`. |
| `gpu frame` | The GPU's time for the whole frame. On Apple GPUs the passes overlap, so their rows can add up to more than it. |
| `swap` | The time spent handing the frame to the screen. |
| `counted` | The game's own counters, below. |

The GPU is timed only while the profiler shows, one frame in 30, and its times are read back four frames late, so measuring them never stalls the frame.

## Measuring without the overlay

`EMBER_STATS=1 go run …` prints, on quitting, averages over the run after a 60-frame warm-up:

- the frame's work and the frame rate it would allow;
- the frame, tick, render and swap times;
- the wait for the display, as `acquire`, which it leaves out of its reports of slow frames;
- each system's and each GPU pass's cost;
- the backend and the graphics card it drew with.

`EMBER_STATS=cpu` does the same without timing the GPU.

## The game's own measures

- `Stats().Record(name, d)` records a span of the game's own, smoothed like the rest; `Recorded()` lists them.
- `Stats().Count(name)` adds one to a running counter; `Counted()` lists them, and the profiler shows them under `counted`.

## How sprites share draw calls

One draw call takes sprites of up to eight textures, even interleaved: in a layer sorted by y, a hero can stand between trees between enemies of several sheets. A new call starts only for:

- a ninth texture;
- a change of clip, blend or mask.

The draw order stays as it was. A typical scene draws in a handful of calls; beyond eight sheets mixed together, more sheets mean more draws ([decision 0031](../99-decisions/0031-a-batch-draws-several-textures.md)).

## What no camera shows isn't drawn

A sprite, nine-patch, line or polygon is skipped when its bounds, grown by a 16-point margin, lie outside every camera that draws its layer: the main one and any drawing `Into` a texture, such as a minimap. A layer sorted by y sorts only what is in view. So a world can hold far more sprites than the screen shows, and the ones off screen cost nothing to draw.

These always draw:

- fixed layers and the UI;
- what reports no bounds: a custom `Draw`, text and particles.

Tile maps cut their own cells to the view.

## Try it

[24-many-sprites](../../examples/24-many-sprites)
