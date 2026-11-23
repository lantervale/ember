# 0158. Styled scrollbars are boxes of their parts, as Chrome's

- Status: Accepted
- Date: 2026-11-23
- Area: api
- Owner: engine

## Context

Real pages style their scrollbars through Chrome's `::-webkit-scrollbar` and its parts, which ui refused, so their bars kept ui's own look and its 15-point gutter. The standard `scrollbar-color` and `scrollbar-gutter` weren't read either.

## Decision

- `::-webkit-scrollbar`, `-track`, `-track-piece`, `-thumb` and `-corner` are pseudo-elements of their own, read in every sheet under their prefixed names, with the pseudo-classes Chrome gives them (`:vertical`, `:horizontal`, `:start`, `:end`…) as states of the part.
- As in Chrome, a box's bars are drawn from its sheets alone once a rule styles its `::-webkit-scrollbar` and its `scrollbar-color` is `auto`: the bar's `width` and `height` are its gutters (`display: none` none), and each part is a box laid along the bar as Chrome lays it (the thumb sized by the share in view, rounded, no shorter than its `min-height` or 26 points), painted by the same code as an element's box.
- Unstyled bars keep ui's own look; `scrollbar-color` colours that look, and `scrollbar-gutter: stable` keeps the vertical gutter wherever the box clips.
- The exported page keeps its look for ui's bars off the boxes whose bars are styled (`data-ember-bars`), so Chrome draws those from the same sheets.

## Alternatives

- Draw every bar, ui's own included, as styled parts from rules in the user agent's sheet: those rules would cascade into an author's partial styling (a thumb given a colour alone would keep ui's insets), where Chrome's styled bars start from nothing.
- Keep refusing the parts and read only `scrollbar-color`: the standard covers two colours, not the thickness, rounding and insets pages ask for.

## Consequences

A styled bar matches Chrome's to the pixel; the parts' `:hover` and `:active`, the buttons and the resizer are read and never hold or draw, and `scrollbar-color` can't match Chrome's system bars, which differ by platform.
