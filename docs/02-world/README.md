# The World

Everything in ember happens in one place: the **World**. It holds the entities, the systems that update them, the resources they share and the presenters that draw them. The same World runs in a window or headless on a server.

This page is the map. Each page after it walks through one idea with a small example.

| Concept | What it is | Made with | Page |
|---|---|---|---|
| Entity | A Go struct that embeds `*world.BasicEntity` and its components | `AddEntities(e…)` | [Entities and systems](01-entities-and-systems.md) |
| Component | Plain data with a `GetX()` getter | a struct field of the entity | [Entities and systems](01-entities-and-systems.md) |
| System | Logic run on every entity that has the components it needs | `AddSystems(s…)`, or `On(phase, name, fn)` for a function | [Entities and systems](01-entities-and-systems.md) |
| Resource | One value that belongs to no entity, such as the weather or the game's settings | `Insert(&Weather{})`, read with `world.Resource[Weather](w)` | [Resources](02-resources.md) |
| Event | A value one system sends and any other reads, without either knowing the other | `world.Emit(w, WentOut{})`, read with a `world.Reader[WentOut]` | [Events](03-events.md) |
| Child | An entity owned by another: added and removed with its parent | `parent.AppendChild(child)` | [Parents and children](04-children.md) |
| View | A `world.Group[T]` the World keeps in sync but never updates | `AddView(view)` | [Views](05-views.md) |
| Presenter | What draws the World after each frame | `AddPresenter(p)` | below |

## Systems

A system embeds `world.Group[T]`, where `T` is an interface listing the getters it needs. The World hands it every entity that matches, as entities are added and removed.

## Phases

Each frame runs the systems phase by phase:

1. `Startup`, once, before the first frame's other phases;
2. `PreUpdate`;
3. `FixedUpdate`, at sixty steps per simulated second, whatever the frame rate ([Fixed step](06-fixed-step.md));
4. `Update`, the default;
5. `PostUpdate`.

A system runs in `Update` unless its `Phase()` method says otherwise. `FixedUpdate` follows simulated time, not frames, so the simulation is the same on a slow and a fast machine.

The World's clock is `w.Time()`, or `f.Time` in a system. It holds the fixed step, the alpha between two steps and the time scale, which pauses the game at 0 ([Time](07-time.md)).

## Presenters

Drawing is not a system. After each frame is simulated, the World calls its presenters, which observe the entities they need through views. `ember.New` adds the renderer; a headless server has no presenter at all. [Windows](../03-windows/README.md) shows one at work.
