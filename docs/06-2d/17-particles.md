# Particles

A particle emitter throws out many small pictures that move, change and fade on their own: smoke, fire, sparks. Here a campfire smokes, burns and sparks at night, and clicking throws a burst of embers that falls and is gone.

![A campfire with flames, smoke and sparks, and two bursts of embers where the window is clicked](particles.gif "ember · particles")

## The campfire

The fire is three emitters on one layer, around the logs. `twod.NewParticles` takes a texture, or `nil` for round dots, and options:

| Option | What it sets |
|---|---|
| `Rate` | how many particles a second |
| `Lifetime`, `Velocity` | a minimum and a maximum; each particle picks its own between them |
| `Direction` | an angle and a spread, in radians, clockwise from the right, so `-geom.Pi/2` is up |
| `Gravity` | a pull, a vector in points a second per second |
| `Spin` | each particle turns at a speed picked between two |
| `ColorOverLife`, `SizeOverLife` | from a start to an end over each particle's life; a size is the side of its box in points |
| `LocalSpace()` | particles move with their emitter; without it they stay where they were thrown even if it moves |
| `MaxParticles` | caps how many live at once |
| `Seed(n)` | makes a run repeat exactly |

The smoke is drawn normally. The flames and the sparks `Blend(Add)`, so where they overlap they glow brighter.

While the game runs, `Emit(n)` throws n particles and `SetEmitting` starts or stops the flow.

[main.go](../../examples/34-particles/main.go)

## The puff

The smoke and the flames share a soft disc made in code, as in [Blend modes and masks](14-blend-modes.md).

[glow.go](../../examples/34-particles/glow.go)

## Bursts

A click adds an emitter where the mouse is:

- `Burst(80)` and `OneShot()`: it throws eighty embers at once and stops;
- `Drag` slows them as they fall, taking that fraction of their speed away each second.

When its last particle is gone, the World sends a `twod.ParticlesFinished` event, and `Bursts` removes the emitter.

[bursts.go](../../examples/34-particles/bursts.go)

## Try it

[34-particles](../../examples/34-particles)
