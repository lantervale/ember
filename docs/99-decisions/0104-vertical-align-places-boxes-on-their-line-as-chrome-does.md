# 0104. vertical-align places boxes on their line as Chrome does

- Status: Accepted
- Date: 2026-11-18
- Area: api
- Owner: user

## Context

ui read `vertical-align: baseline | sub | super` and refused the rest, a shift worked out once per run; pages of the web align pictures, inline-blocks and cells with `top`, `middle`, `bottom` and lengths, and the web-platform-tests' references do too. CSS 2 section 10.8 leaves open where a line grows when boxes on its top and its bottom are both taller than it, and Chrome departs from it in places.

## Decision

- One alignment, per line (`layout.lineBox`): every inline box open on the line counts by its line height and every atomic inline by its margin box, each from its parent's baseline (`baseline`, `sub`, `super`, a length, a percentage of the box's own line height, `middle`), inner boxes first; `text-top` and `text-bottom` wait for their parent, `top` and `bottom` for the line. `sub` and `super` go through it, no path of their own.
- Where the spec is open or Chrome departs, Chrome's layout is the reference: the tallest `top` or `bottom` box says where the line grows (below for a top one, and when they tie), a `top` or `bottom` box inside another aligns to that one and not to the line, and one inside a shifted box moves with it after the line is sized. An inline-block with no line of its own, or one that scrolls, stands on its bottom margin edge.
- An inline box with nothing on a line it is open on still counts there (the strut of section 10.8), which ui left out before.
- On a table cell, `vertical-align` places the content: `top`, `middle`, `bottom`, the rest as `top`. The user agent sheet gives row groups `middle` and has rows and cells inherit, and `valign` hints `vertical-align` on the element that carries it. This supersedes the line of [0092](0092-presentational-attributes-are-hints-under-every-rule.md) that made `valign` an `align-content`; `align-content` on a cell still wins.
- A longhand a sheet sets to `inherit` restyles the children only when that longhand changes: the user agent sheet's own `inherit` no longer has every change restyle a subtree.

## Alternatives

- The aligned subtree of the spec to the letter (every `top` and `bottom` box against the line): simpler to say, but pages are checked in Chrome, and the difference shows as boxes a few points off.
- Keeping `sub` and `super` as a shift per run beside a second path for the rest: two answers to how tall a line is.
- `valign` as `align-content`: a sheet's `vertical-align` on a cell would lose to an attribute, the reverse of a browser.

## Consequences

Lines are sized once per pass over their pieces and the spans open on them, with no allocation; a line's shifts live in its runs and spans until the next line is asked for. Baseline alignment between the cells of a row is still missing ([what is missing](../05-ui/10-missing.md)).
