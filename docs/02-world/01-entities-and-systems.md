# Entities and systems

All game logic in ember is written the same way: components hold data, an entity is made of components, and a system updates every entity that has the components it needs. Here a candle and a campfire cool down until both are out.

## Components

A component is a plain struct, with a getter that lets a system ask for it.

[components.go](../../examples/02-entities-and-systems/components.go)

## An entity

An entity is a struct that embeds `*world.BasicEntity` and its components.

[flame.go](../../examples/02-entities-and-systems/flame.go)

## A system

`Burning` lists the getters the system needs, and `world.Group[Burning]` hands it every entity that has them. `Update` runs once per frame.

[cooling.go](../../examples/02-entities-and-systems/cooling.go)

### Several groups in one system

A system can have more than one group: the embedded one, and named fields such as `prey world.Group[Prey]`. The World fills them all.

### Keeping one kind of entity out

A group takes every entity that has the getters, whatever else it is. When a hero and an enemy both have health, a system meant only for enemies asks for a tag, an empty component that marks the kind:

```go
type Hostile struct{}

func (h *Hostile) GetHostile() *Hostile { return h }
```

Embed `Hostile` in the enemy and add `GetHostile() *Hostile` to the system's interface: the hero stays out.

### When an entity is missing from a system

A component's fields and getters sit on the entity as if they were its own, so a name the entity already has hides the other. A field called `Position` on a component hides the `Position()` method an embedded `twod.Item` gives the entity, and the entity then quietly leaves every group that needs that method. Give fields and getters names of their own.

To find out why an entity is left out, print `w.Explain(entity)`. It lists every system and view, whether the entity is in it, and, for the others, what it lacks: a missing getter, or a method a field hides.

Groups are matched when an entity is added. When a later change makes it match a group, or stop matching one, `world.Regroup(entity)` offers it to its groups again, once the running system returns. `Explain` says when it's needed.

## The World

`newWorld` adds the entities and the system, and a function added with `On` stops the World once it is empty. `main` runs it in real time with `MustRun`.

The World is built in a function of its own so the example's test can run the same World with `RunFrames`: frame by frame at the fixed step, without waiting on the clock.

[main.go](../../examples/02-entities-and-systems/main.go)

## Try it

[02-entities-and-systems](../../examples/02-entities-and-systems)
