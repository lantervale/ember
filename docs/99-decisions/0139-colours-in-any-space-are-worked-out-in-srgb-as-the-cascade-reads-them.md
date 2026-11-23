# 0139. Colours in any space are worked out in sRGB as the cascade reads them

- Status: Accepted
- Date: 2026-11-23
- Area: api
- Owner: engine

## Context

ui read hex, `rgb()`, `oklch()` and names only, refused math in channels and times, and fixed `prefers-contrast`, `forced-colors` and `any-hover`. Pages use `hsl()`, `color-mix()`, relative colours and `calc()` everywhere, and ui draws in sRGB only.

## Decision

- Every colour function (`rgb()`, `hsl()`, `hwb()`, `lab()`, `lch()`, `oklab()`, `oklch()`, `color()` in its nine spaces), relative colours (`from`) and `color-mix()` are read into a colour of their own space, converted through XYZ by CSS Color 4's matrices, and the computed value is sRGB clipped to its gamut, as Chrome draws on an sRGB display. Nothing past the cascade changes.
- Channels, alpha and mix percentages take math, a relative colour's channel names standing for the origin's numbers. `currentColor` inside a function is the element's colour where the cascade reads it (its parent's in `color`).
- Times take math as a new dimension type of math; a negative duration is 0.
- Media queries measure their font units in the initial font, the Document's `sans-serif` at 16px. `prefers-contrast`, `forced-colors` and a touch-only `any-hover`/`any-pointer` are the program's answers (`ui.PrefersContrast`, `ui.ForcedColors`, `ui.TouchOnly`), as no system is read for them; the Chrome harness emulates them.
- Where Chrome holds no condition, ui doesn't either: a `style()` query of a standard property fails its whole `@container` condition, and `@keyframes` inside `@container` apply unconditionally.

## Alternatives

- Keep colours in their space to the GPU: a wide-gamut pipeline ui doesn't have; clipping is what Chrome shows on the same display.
- Gamut-map out-of-range colours by chroma as CSS Color 4 suggests: Chrome clips, and the comparison is with Chrome.
- Add contrast and forced colours to `platform.Preferences`: no backend reads them yet, and the setting would be an empty promise.
- Measure media query font units in Chrome's default font: ui has no Times, and a Document's own initial font is the honest analogue.

## Consequences

A page's colours compute as Chrome's within a step (`TestColourFunctionsComputeAsInChrome`), at the cost of resolving `currentColor` in a function where it is declared rather than where it is inherited. Font units in media queries differ from Chrome's near a bound, since the two default fonts differ.
