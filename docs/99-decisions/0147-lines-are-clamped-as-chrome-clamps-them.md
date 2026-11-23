# 0147. Lines are clamped as Chrome clamps them

- Status: Accepted
- Date: 2026-11-23
- Area: api
- Owner: user

## Context

The web clamps a card's text to a few lines with Chrome's old flexbox: `display: -webkit-box; -webkit-box-orient: vertical; -webkit-line-clamp: n`, which Chrome 157 lays out as `flow-root` and still draws the lines past the clamp. The standard `line-clamp` does the same but hides what follows; Chrome reads it only behind its `CSSLineClamp` flag.

## Decision

- ui reads `-webkit-box`, `-webkit-inline-box`, `-webkit-box-orient` and `-webkit-line-clamp` as properties of its own, in any sheet, and `line-clamp` as the standard one: a vertical `-webkit-box` clamping its lines is a `flow-root`, any other is `flex` along its orient.
- The clamp travels down the layout as part of a box's input, the lines left before it, and comes back in its output, the lines it took and whether the clamp fell in it: a block in the flow ends below the clamped line, and those after it are laid out where they would be. The layout cache keys on it.
- The clamped line makes room for an ellipsis when anything follows it. `-webkit-line-clamp` draws what follows, as Chrome 157; `line-clamp` does not, as Chrome behind its flag, which the comparison turns on (`chrome.MatchWith`).

## Alternatives

- Clamp at paint, cutting the drawn lines: the box would keep its unclamped height, where every page clamping a card relies on the shorter one.
- Read `line-clamp` only, the `-webkit-` form as an alias: pages write the `-webkit-` form, and Chrome draws past it, so its `overflow: hidden` is what the pages count on.

## Consequences

Cards from the web clamp as in Chrome, and ember's own sheets have the standard property. A clamp holds through the blocks of its flow only: one that starts a formatting context counts no line, as in Chrome.
