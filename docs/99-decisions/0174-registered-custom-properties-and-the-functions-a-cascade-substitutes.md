# 0174. Registered custom properties and the functions a cascade substitutes

- Status: Accepted
- Date: 2026-11-23
- Area: api
- Owner: engine

## Context

Chrome 157 types custom properties (`@property`, `CSS.registerProperty`), interpolates them, and substitutes `if()`, typed `attr()` and custom functions (`@function`) as it does `var()`. A custom property was text in `style.Computed.Vars`, inherited whole, and the animation machinery worked by longhand.

## Decision

- A registered property's computed value stays text in `Vars`, written as Chrome's getComputedStyle writes it (`22px`, `90deg`, `rgb(…)`): `var()` substitutes it, inspection reads it, and nothing outside `css` learns a new type. The root starts from every initial value; an element starts from its parent's, those that don't inherit put back to theirs.
- An animation or transition of a custom property is an `override` that names it and carries its text, applied when the element's custom properties are, before the longhands that read them. Interpolation reads the two texts back by the syntax, item by item.
- `if()`, `attr()` and calls of custom functions are replaced where `var()` is, at computed-value time; `sibling-index()`, `sibling-count()` and an unkeyed `random()` are replaced in a property's value (not in a custom property, as Chrome) before its math is read, `random()` by a fixed number hashed from its key, the element's place in the tree, the property and its place in the value.
- `doc.RegisterProperty(css.PropertyDefinition{…})` is `CSS.registerProperty`, over any rule of its name; the export writes it as that script.

## Alternatives

- Typed fields per registered property in `style.Computed`: the layout and paint never read them, and every registration would need its own storage and copying.
- A longhand slot per custom property in the transition machinery: the slots are fixed tables shared by every Document, where registrations are per Document.
- `random()` from a generator per restyle: a value that changes on every restyle, where Chrome keeps it on the element.

## Consequences

Registered values cost a parse back to interpolate, only while they move. `calc-size()` and `interpolate-size` stay missing: they need a size from layout the cascade cannot hold yet.
