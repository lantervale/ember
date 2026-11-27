# Events

An event lets one system tell any other that something happened, without either knowing the other. Here `Cooling` says a flame went out, and `Log` prints it.

## Sending one

`WentOut` is a plain struct, and `world.Emit` sends it.

[cooling.go](../../examples/04-events/cooling.go)

## Reading them

A system holding a `world.Reader[WentOut]` reads each event once, whatever phase it runs in.

An event lives two frames. While the World has `FixedUpdate` systems, it also waits until a fixed step has begun after it. So a fixed-step reader never misses an event sent in `Update`, however fast the frames run ([decision 0023](../99-decisions/0023-events-wait-for-a-fixed-step.md)).

### Only whether something happened

When each event doesn't matter, only that there was one:

- `Any(w)` says whether there was an event;
- `Count(w)` says how many.

Like `Read`, they use up the events the reader hasn't seen yet, so the next call starts after them. `if hits.Any(w) { play(hit) }` plays one sound for a frame's many hits.

[log.go](../../examples/04-events/log.go)

## The World

[main.go](../../examples/04-events/main.go)

## As before

[components.go](../../examples/04-events/components.go)

[flame.go](../../examples/04-events/flame.go)

## Try it

[04-events](../../examples/04-events)
