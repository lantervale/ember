# 0084. A scrolling viewport is an option of a Document

- Status: Accepted
- Date: 2026-11-17
- Area: api
- Owner: engine

## Context

A Document's viewport did not scroll: `html` was the outermost box, and a page taller than its window was cut. To scroll a fetched page the web explorer gave it `html { height: 100%; overflow-y: auto; }`, which made `html` a scroll container one window high. The canvas background is placed in the root element's box, so a page's `body { background: url(tree.jpg) no-repeat bottom right }` showed its picture at the bottom right of the window and held it there while the page scrolled; in a browser it is at the page's end. `position: fixed` boxes scrolled away with the content for the same reason.

## Decision

- `ui.ScrollingViewport()` is a Document option, off by default. With it the viewport is the scrollport, as in a browser: `html` keeps its own height, percentages on `html` and `body` resolve against the viewport, and what overflows it scrolls.
- `overflow` on `html`, or on `body` while `html`'s is visible, is the viewport's, as CSS Overflow has it: that element's own overflow is visible, `visible` scrolls as `auto` and `clip` is `hidden`. `scrollbar-width` on that element is the viewport's too.
- The viewport is a box of its own (`box.Viewport`) above the root element's, a scroll container as big as the window whose scroll offset the root element keeps. The wheel, the page keys, the thumb, the bars, overlay or classic with their 15-point room, reveal and sticky boxes work through the code every scroller uses, and every position read through `Box.Rect` (`Node.Box`, `BoxOf`, hit testing, automation, `ui.Follow`) is the scrolled one.
- A `position: fixed` box moves by the viewport's offset each time it scrolls, so it stays put. What scrolls reaches as far as the root's margin box, what overflows it and the absolute boxes of the initial containing block; fixed boxes add nothing.
- The canvas background covers the window, and its picture is placed in the root element's padding box, where Chrome places it. The whole viewport takes the pointer.
- Exported for Chrome, the page keeps its bar and `html` carries `data-ember-scrolling-viewport`; the offset travels as any scroller's. Pictures of such a page are taken in a frame: headless Chrome's screenshot of a scrolled window is blank.

## Alternatives

- Always on: a game's screen taller than its window would show a bar, lose 15 points of width to it, and take the wheel and every click of the window from the game. Screens are laid out to their window; pages are not.
- Keep `html` as the scroll container and special-case the canvas background and fixed boxes when the root scrolls: `html`'s height would still be the window's, so `height: 100%` chains, `bottom` on absolute boxes and the root's own box would differ from a browser's, each needing its own exception.
- An offset on the tree, the root's box moved by it, in place of a box: the bars, the wheel, the keys, reveal, sticky boxes and the clip would each need a second path beside the scrollers' own.
- A CSS property: CSS has none, since a browser's viewport always scrolls.

## Consequences

Documents without the option are unchanged: `html` is the root box, and the canvas picture is placed in the window. With it, `tree.Root` is the viewport's box and the root element's is its child, so code walking boxes from the root meets one box more. A fixed box inside a box that scrolls on its own still moves with that box; there is no `background-attachment`. Headless Chrome hides a window's and a frame's own bar in a picture, so pixel pages of a scrolling viewport set `scrollbar-width: none`, and the bar's room is checked by boxes.
