# Time

The World's clock can slow down and pause the game, and tells each frame how far it sits between two fixed steps, so what moves on fixed steps is drawn smoothly. Here a ball takes one step each fixed step: a second at normal speed, at half speed and paused, then a few frames at 144 fps drawn smoothly between steps.

## A ball on fixed steps

The ball moves one unit each `FixedUpdate` step and remembers where it was. `Drawn(alpha)` is where to draw it in a frame: between its last two positions, `alpha` of the way.

[ball.go](../../examples/08-time/ball.go)

## Scale and alpha

`w.Time()`, or `f.Time` in a system, is the clock:

| Field | What it does |
|---|---|
| `Scale` | Multiplies simulated time: 0.5 is half speed, 0 pauses, so fixed steps stop and the ball stays put. |
| `Alpha` | How far the frame sits between the last fixed step and the next, from 0 to 1. |
| `RealElapsed` | The real time the frames took, never scaled or paused (each frame still clamped by `MaxDelta`), for what keeps running while the game is paused, such as a menu's animations. |

At 144 fps a frame is shorter than a fixed step, so most frames take no step. Drawing the ball at `Drawn(Alpha)` makes it glide instead of stutter.

[main.go](../../examples/08-time/main.go)

## Try it

[08-time](../../examples/08-time)
