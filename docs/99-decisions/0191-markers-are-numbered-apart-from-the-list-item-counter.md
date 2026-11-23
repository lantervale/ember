# 0191. Markers are numbered apart from the list-item counter

- Status: Accepted
- Date: 2026-11-23
- Area: rendering
- Owner: engine

## Context

ui numbered a list item's marker with the `list-item` counter itself, as CSS Lists has it, with `value` setting the counter and a reversed list's reset counting its items. Chrome numbers the markers apart, and its counter differs from them: a reversed `ol` without `start` counts from 0 down, `value` leaves the counter alone, and a `counter-reset` nests in a counter of its name the parent has, for the element's descendants alone.

## Decision

- Each list keeps its own marker numbers, as Chrome's list-item ordinals: from `start` (a reversed list's from its number of items), an item's `counter-set` of `list-item`, else its `value`, else the number before it plus its step, which its own `counter-increment` (with its `counter-reset`) replaces.
- The `list-item` counter takes nothing implicit from a list or item whose counter properties name it; a reversed `ol` resets it to one over its `start`, 1 without one, and `value` never sets it.
- A `counter-reset` nests in a counter of its name in the parent's scope for the element's descendants alone, as Chrome and CSS Lists' inheritance of counters do.

## Alternatives

- Keep one counter for both, as Firefox and the spec's `reversed(list-item)`: markers right, but `counter()` never as Chrome's.
- Chrome's second count, once something has it count again (the item count, `value` as a counter of the item's own): what Chrome shows depends on timing, so ui follows the first count, the one a static page keeps.

## Consequences

`counter(list-item)` reads as Chrome's on a page Chrome counts once; the cases its later recount changes stay recorded in [missing](../05-ui/10-missing.md). Heading counters reset on a heading whose parent already resets them no longer reach the siblings after it, as in Chrome.
