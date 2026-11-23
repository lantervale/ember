# 0163. Scrolls snap and glide as Chrome's strategies, on the Document's clock

- Status: Accepted
- Date: 2026-11-23
- Area: api
- Owner: engine

## Context

Carousels, paged sections and pages under a sticky header rely on `scroll-snap-type`, `scroll-snap-align`, `scroll-snap-stop`, `scroll-padding` and `scroll-margin`, on `scroll-behavior: smooth` and on `overscroll-behavior`, which ui refused: its scrolls landed wherever the wheel, a key or the program left them, at once, and a scroll at a box's end always went on to the box round it. Where a snap lands depends on what moved the scroll, which the spec leaves to the browser.

## Decision

- A scroll is snapped as it lands, by one of two of Chrome's strategies: an absolute scroll (`doc.SetScroll`, a thumb let go, `ScrollIntoView` and `ScrollToStart`, a layout) takes the snap position nearest where it was sent; a relative one (the new `doc.ScrollBy`, the wheel once its gesture ends, the page keys and a click on the track) the nearest of those ahead of where it began, stopping at a `scroll-snap-stop: always` on the way, and a mandatory container with none ahead stays put. A proximity container snaps within a third of its snapport, and an area bigger than the snapport lets the scroll rest anywhere it covers it, its ends snap positions too. A layout that moves a container's areas snaps it again, to the areas it was snapped to.
- `scroll-behavior: smooth` glides the program's scrolls, and the wheel's snap always glides, on Chrome's desktop curve, `cubic-bezier(0.4, 0, 0, 1)` over √distance frames of 1/60 s (at most 1.5 s), stepped on the Document's clock. The wheel, the keys and the thumb still move at once.
- `overscroll-behavior: contain` or `none` on an axis keeps the wheel and the page keys that reach a box's end from the boxes round it, an iframe's host included.
- `scroll-state(snapped)` holds on the areas a container is snapped to; `scroll-state(scrolled)` on the way its last relative scroll went on each axis, as the WPT tests have it: `SetScroll` and `ScrollIntoView` don't count.

## Alternatives

- One strategy for every scroll, the nearest position: a mouse notch or a PageDown short of half an item would snap back where it began, and a stop could be jumped over.
- Snap the wheel on each turn, as Chrome does for a mouse: a trackpad's many small turns would each move a whole item. Ending the gesture after two frames without the wheel serves both.
- Glide the wheel and the keys too, as Chrome does by the system's setting: ui's scrolls by the user have always moved at once, and its tests read them the next frame.

## Consequences

The final offsets of scrollTo, scrollBy and scrollIntoView match Chrome's, the viewport's included. Headless Chrome runs no scroll animation and doesn't update `snapped` or `scrolled` after a script's scroll, so the glide's curve is Chromium's own code, and those two queries are tested on WPT's semantics, Chrome checked only at the first layout's snap.
