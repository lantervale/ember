# 0198. Scrollbar parts take the pointer's states, and gutters scroll as Chrome's

- Status: Accepted
- Date: 2026-11-23
- Area: api
- Owner: engine

## Context

Decision 0158 read the parts' `:hover`, `:active` and `:window-inactive`, `::-webkit-scrollbar-button` and `::-webkit-resizer`, and matched none of them; it styled no viewport's bars and drew styled parts under `scrollbar-width: thin`. Decision 0181 kept the start gutter of `stable both-edges` inside the scrollport, so such a box scrolled sideways one gutter further than Chrome's, and a right-to-left one started at its edge where Chrome starts it scrolled.

## Decision

- The part of a box's styled bars under the pointer and the one the left button presses are kept on its node, as Chrome's `Scrollbar` keeps them, and restyle its parts: `:hover` and `:active` hold on that part, on the bar while any part of it does and on the track while a piece or the thumb does; the thumb stays hovered and pressed while dragged. `:window-inactive` holds while the window isn't focused, as `::selection`'s.
- Buttons are boxes of `display: block` only, as Chrome on macOS lays them (its own bars have none), two at each end; the track lies between them, and a press scrolls 40 points. The track pieces meet under the thumb's middle, and a thumb whose least length passes the track fills it.
- `resize` other than `none` on a box that clips keeps a corner for its `::-webkit-resizer`, which shortens a lone bar as Chrome's.
- `scrollbar-width: thin` asks for the system's bars, as `scrollbar-color`: overlay bars, the parts ignored.
- The viewport's bars are styled by body's rules, else the root's (Chrome's `ScrollbarStyleSource`); export keeps ui's look off both.
- Across a box keeping a stable gutter whose content overflows sideways, the scroll range is the content less the padding box but the bar that stands, whatever gutter is empty, and a right-to-left box starts scrolled by its empty gutters, as Chrome.
- The comparisons drive Chrome's pointer for these states (`chrome.PixelsUnderMouse`); export leaves a part's state pseudo-classes to it.

## Alternatives

- Hover by the element's own `:hover`: Chrome tells the part, and `.box:hover::-webkit-scrollbar-thumb` doesn't restyle a bar the pointer isn't on.
- Lay buttons out as Linux and Windows place their own (one at each end without `display: block`): the comparisons would differ by platform, as for `scrollbar-color` (decision 0181).
- Keep ui's range and start offset: the gap was a gutter's width, visible on every overflowing right-to-left box with a stable gutter.

## Consequences

Styled bars, their buttons and resizers match Chrome's on macOS, under the pointer too. A button without `display: block` shows on Linux and Windows Chrome and not in ui; an unstyled resizer draws no grip and no box is resized by dragging; a box's `:hover` restyles its parts at once where Chrome waits for its bar to change.
