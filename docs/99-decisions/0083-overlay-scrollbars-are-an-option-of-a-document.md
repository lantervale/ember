# 0083. Overlay scrollbars are an option of a Document

- Status: Accepted
- Date: 2026-11-17
- Area: api
- Owner: engine

## Context

A box that scrolls keeps a 15-point gutter for its bar, as Chrome's classic scrollbars do, and the Chrome comparisons are written with that gutter. A browser on macOS or a phone draws its bars over the content, in no room: a page shown in the web explorer was 15 points narrower than in such a browser, and its lines broke elsewhere.

## Decision

- `ui.OverlayScrollbars()` is a Document option, off by default: every box of the Document that shows scrollbars keeps no gutter for them, and they are drawn over its content, where a classic bar lies.
- The bars work as classic ones: the wheel, a drag of the thumb, a click on the track, which takes the press from the content under it. They show whenever the content overflows, as a dark thumb with a light edge, seen on a light or a dark page; they do not fade when idle.
- It is not CSS. Exported for Chrome, such a Document's bars are written zero wide (`::-webkit-scrollbar { width: 0; height: 0; }`), so Chrome lays the page out the same on any machine.

## Alternatives

- A CSS value: none means it. `scrollbar-gutter` can only add room (`auto`, `stable`), `overflow: overlay` is now another word for `auto` in Chrome, and `scrollbar-width: none` removes the bar. Whether bars overlay is the browser's and the system's choice, not the page's.
- A property of ember's own (`-ember-scrollbar: overlay`): a sheet with it would mean nothing in Chrome, and ui's CSS is what Chrome reads the same way.
- No gutter for the root element only, as the exported page already hides the root's bar in Chrome: screens that scroll their root would change, and a page's inner scrollers would still differ from the browser's.
- Follow the system's setting: a screen would be laid out differently from one machine to the next.

## Consequences

Classic bars stay the default and every existing comparison is unchanged. An overlay bar covers the last 8 points of its box's content along that edge, and a click there goes to the bar. Chrome draws no bar for the exported page, so a pixel comparison of an overlay Document differs by its thumbs; boxes compare exactly.
