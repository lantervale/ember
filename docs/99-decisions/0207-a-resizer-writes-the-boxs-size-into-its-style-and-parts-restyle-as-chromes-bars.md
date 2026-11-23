# 0207. A resizer writes the box's size into its style, and parts restyle as Chrome's bars

- Status: Accepted
- Date: 2026-11-23
- Area: api
- Owner: engine

## Context

Decision 0198 left a box's resizer without Chrome's grip and without its drag, restyled a bar's parts as soon as a rule such as `.box:hover::-webkit-scrollbar-thumb` matched, where Chrome waits, and kept a vertical box's bars in its own frame, so one with both gutters and a vertical bar scrolled a gutter off Chrome's range.

## Decision

- `resize` takes `both`, `horizontal`, `vertical`, `block` and `inline`, the last two turned by the writing mode. A press in the resizer's corner drags the box's size as Chrome's: the size at the press plus how far the pointer moved, mirrored on a right-to-left box, no smaller than the corner (or the size at the press when smaller), written into the element's style attribute as Chrome writes it, `width` and `height` in whole pixels of the size `box-sizing` names, a form control's margins beside them, each in place of its own declaration. The program reads the size with `Node.Style()` and takes it back with `SetStyle`; layout applies min and max sizes as for any width.
- An unstyled resizer draws Chrome's grip: two diagonal strokes, dark over light, mirrored on the left; a resize cursor shows over it and while it drags.
- A bar's parts are kept as last shown, as Chrome's `CustomScrollbar` keeps them: a change of the part under the pointer or pressed restyles the part left, the part reached, their bar and its track; the element's own style changing, its `::-webkit-scrollbar` with it, or the window's focus restyles them all.
- An orthogonal box's bars are its own frame's, turned onto the page, and a vertical box with a stable gutter scrolls as far as Chrome's: across its lines, the content less the padding box; along them, less the padding box but the horizontal bar that stands.

## Alternatives

- Hold the dragged size on the Document, read by a getter: a page's script and Chrome's export read the style attribute, so a second place to look would differ from them, and `SetStyle` would no longer undo a drag.
- Restyle the parts with the element (ui's way so far): simpler, but a page relying on Chrome's late restyle looks different under the pointer.

## Consequences

A resizable box behaves and looks as in Chrome on macOS, and a drag is visible to code that reads or rewrites the style attribute. The viewport's bars still restyle with their element at once; the cached parts cost one more match of `::-webkit-scrollbar` per restyle of a box with styled bars.
