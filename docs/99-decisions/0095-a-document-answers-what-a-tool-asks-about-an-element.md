# 0095. A Document answers what a tool asks about an element

- Status: Accepted
- Date: 2026-11-18
- Area: api
- Owner: engine

## Context

A tool that inspects a screen, such as the Inspector of light's developer tools, shows the rules that match an element, which declarations lost, its computed style and its boxes. The cascade kept none of that: it folds the winning declarations into a style and forgets where they came from.

## Decision

- A Document answers on demand, read-only: `doc.MatchedRules(n)` (`[]css.Match`, the strongest first, with origin, sheet, selector, place, `@media`, `@layer` and declarations marked `Overridden`), `doc.ComputedStyle(n)` and `doc.ComputedValue(n, name)` (`css.Property`), `doc.BoxesOf(n)` (`ui.Boxes`: margin, border, padding, content, in the points of `BoxOf`).
- They run the matching again for the one element asked about, in files of their own (`ui/inspect.go`, `ui/css/inspect.go`); the cascade of a frame records nothing more and is not changed.
- A computed value is the winning declaration's text with `var()` replaced and lengths in `px`, inherited or initial otherwise; colours and keywords stay as written.

## Alternatives

- Recording the origin of every declaration during the cascade: exact by construction, but paid by every element of every screen for a tool few programs open.
- Leaving it to the tool, over `doc.Sheets()`: the tool would copy the selector matching, the specificity, the layers and the hints, and drift from the engine.
- Computed values resolved as Chrome's `getComputedStyle` (colours as `rgb()`, used sizes): closer to a browser, but a second reading of every property kept in step with layout.

## Consequences

Tools read the cascade without the engine paying for it each frame; the price is a second walk of the sheets kept in step with the cascade by tests, and computed values that are the cascaded ones, not the used ones.
