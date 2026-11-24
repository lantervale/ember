# 0221. A vertical root scrolls its viewport along its blocks

- Status: Accepted
- Date: 2026-11-23
- Area: api
- Owner: engine

## Context

A root of a vertical writing mode was laid out in a viewport that didn't scroll: a `vertical-rl` page wider than its window could not be read past its first screen, and `body { writing-mode: vertical-rl }` left `html` horizontal, where Chrome turns the whole viewport.

## Decision

- The root takes the writing mode and `direction` of its first `body`, as CSS Writing Modes asks and Chrome does; its computed style keeps its own, as Chrome's `getComputedStyle` reports.
- The viewport lays the root out turned in what its bars leave, the initial containing block, and scrolls what overflows towards its blocks' and lines' ends: a root whose blocks start on the right (`vertical-rl`, `sideways-rl`) stands against the viewport's right edge and scrolls left, its offset from zero below zero as Chrome's `scrollX`; `vertical-lr` scrolls right from zero. The viewport box carries that start (`Box.ScrollStart`), so `doc.Scroll`, `SetScroll`, the bars, fixed and sticky boxes take it as an element scroller's.
- The keys that page and end a scroller move along its blocks: Home, End and Space, and Page Up and Page Down but on macOS, where Chrome pages down the page whatever its writing mode. The arrows and the wheel scroll as they point. `ScrollToStart` puts an element's block start at the scroller's (its right in `vertical-rl`).

## Alternatives

- Taking body's writing mode into the root's computed style: simpler for layout, but `getComputedStyle` and inheritance would differ from Chrome's.
- Mirroring the viewport so a `vertical-rl` page scrolls from zero upwards: offsets would disagree with every browser's `scrollLeft`.

## Consequences

A vertical page reads as in Chrome, by the wheel, the keys and the program's calls. The root's style as laid out is a copy whenever body's mode or direction differs, kept while both are the same.
