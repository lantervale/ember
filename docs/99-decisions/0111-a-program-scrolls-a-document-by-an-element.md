# 0111. A program scrolls a Document by an element

- Status: Accepted
- Date: 2026-11-19
- Area: api
- Owner: user

## Context

The web explorer goes to the place a fragment names, and comes back through its history to where a page was left. `doc.ScrollIntoView(n)` scrolls by the least that shows an element, so a heading below the view came to its bottom edge, and a program could neither read nor set how far anything was scrolled.

## Decision

- `doc.ScrollToStart(n)` puts n's top at the top of each scroller round it, as far as each scrolls, as the DOM's `scrollIntoView()` and a browser going to a fragment do; sideways it comes in by its nearest edge. It waits for an element not laid out yet, as `ScrollIntoView` does, which stays as it is.
- `doc.Scroll(n)` and `doc.SetScroll(n, off)` read and set how far an element's content is scrolled, in the Document's own points, before its zoom, kept within what it scrolls. The viewport's scroll is the root element's, `doc.HTML()`, as `document.documentElement.scrollTop`.
- An offset set before the first layout is kept, and clamped then: a page built off the frame is shown where it was left.

## Alternatives

- Options on `ScrollIntoView(n, opts…)` for `block` and `inline`, as the DOM's: a new option type for two words, where `nearest` and `start` are what the keyboard and a fragment ask for; `center` and `end` can come the same way when something needs them.
- `doc.ViewportScroll()` beside a scroll per element: two paths to the one offset layout keeps on a node, and a page's own scroller (a pane, a list) would have no way.
- A scroll kept by the program in window points: it would not survive a zoom or a resize, and layout already clamps the node's own.

## Consequences

A Document that calls none of them scrolls as before. `SetScroll` moves one scroller and nothing round it; a program restoring a page sets each scroller it kept.
