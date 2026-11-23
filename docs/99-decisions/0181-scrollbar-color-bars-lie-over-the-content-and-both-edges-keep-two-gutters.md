# 0181. scrollbar-color's bars lie over the content, and both-edges keeps two gutters

- Status: Accepted
- Date: 2026-11-23
- Area: api
- Owner: engine

## Context

Decision 0158 read `scrollbar-color` as a colouring of ui's own bars, in their 15-point gutter, and `scrollbar-gutter` as `stable` alone. Chrome reads `scrollbar-color` as a request for the system's bars, its `::-webkit-scrollbar` rules then ignored; headless Chrome on macOS draws those as overlay bars, keeping no gutter and hidden at rest, and on Linux and Windows as classic bars of their own look. `stable both-edges` keeps a gutter at each inline edge, and Chrome clips content only under a bar that stands, not under an empty gutter.

## Decision

- A box whose `scrollbar-color` isn't `auto` has overlay bars, as Chrome's system bars on macOS: no gutter, its thumbs drawn over the content's edge in the thumb colour, as under `ui.OverlayScrollbars()`; its `::-webkit-scrollbar` rules are ignored, as in Chrome.
- The Chrome comparisons turn on Chrome's `OverlayScrollbar` feature off macOS, so Linux and Windows lay these bars out as macOS does.
- `stable both-edges` keeps the bar's width at the inline start too, part of the box's gutter in every layout (block, flex, grid, columns), the bar itself still the end one.
- A box clips its content to its padding box less the bars that stand: overflow: `scroll`'s always, `auto`'s once the content overflows; a stable or start gutter with no bar clips nothing.

## Alternatives

- Keep ui's coloured classic bars: they match Chrome's gutter on Linux and Windows but not on macOS, and the comparisons would differ by platform.
- Draw scrollbar-color's bars only while the box scrolls, as macOS: the colours would rarely show in a game, which has no system to show them; the thumbs stay drawn, a few pixels off Chrome's picture at rest.
- Add the start gutter to the box's padding: it would change the size `width` gives a content-box box, where Chrome's gutters come out of the content.

## Consequences

A coloured bar's box matches Chrome's on every platform, and its picture differs by the thumbs Chrome hides at rest. A both-edges box lays out as Chrome's, but its scrollport takes in the start gutter, so it scrolls sideways one gutter further; the viewport and textareas keep their end gutter alone.
