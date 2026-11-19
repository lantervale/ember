# 0086. Media queries are answered for the Document's viewport

- Status: Accepted
- Date: 2026-11-17
- Area: api
- Owner: engine

## Context

ui refused `@media` (decision 0049's subset): a game's screen is laid out fluid, to its window. A screen that changes shape under a width (a side panel that stacks) had to do it in Go, and the web explorer answered each `@media` block once, for a fixed width, by rewriting the sheet's text.

## Decision

- `@media` is part of the subset, always on: it is standard CSS with one meaning, and a sheet that never writes it pays nothing.
- A query is answered for the Document's viewport: the size it is laid out in, in its own pixels (the window's points divided by `Zoom`), and the window's pixel ratio for `resolution`. The rest of the medium is fixed: a colour screen, a light scheme, a mouse, no preference. The `device-*` features are the viewport's, ui knowing no screen.
- Media types, `min-`/`max-` and range syntax, `and`, `,`, `not`, `only`, `or` and nesting are Chrome's, with Media Queries 4's three-valued logic: a feature ui doesn't know makes its query false, with a warning.
- A rule keeps the list of the `@media` rules it stands in; the cascade skips it while one fails. Each layout tells the cascade its viewport (`Transitions.SetMedia`), which compares which rules hold with the last answer: unchanged, nothing is restyled; changed, the whole tree is, once.
- With `ui.BrowserSheets()` (decision 0085), `-webkit-device-pixel-ratio` and its `min-`/`max-` are `resolution`, and another vendor's feature is false in silence.
- A page exported for Chrome carries its `@media` rules.

## Alternatives

- An option, as browser sheets are: nothing in `@media` conflicts with a game's sheet, and an option would leave the subset with two dialects of standard CSS.
- Keep refusing it: adaptive screens stay Go code that swaps classes on resize, and every tool showing a page rewrites text.
- Evaluate when the sheet loads, for one size: the first resize is wrong.
- Restyle the whole tree on every resize: a window dragged by its edge would restyle each frame; boundaries are crossed rarely.
- Follow the machine for `prefers-color-scheme` and `hover`: a game's look would change with the player's system where the game never asked, and Chrome comparisons would differ by machine. A later option can set them.

## Consequences

A sheet's `@media` warning is gone, so a game that kept one for a browser now gets its rules. Crossing a boundary costs one full restyle. Viewport units in a query's length are refused, and the scheme, pointer and motion answers can't be changed yet.
