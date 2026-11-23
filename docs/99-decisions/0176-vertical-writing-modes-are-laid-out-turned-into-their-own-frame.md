# 0176. Vertical writing modes are laid out turned into their own frame

- Status: Accepted
- Date: 2026-11-23
- Area: rendering
- Owner: engine

## Context

ui laid text out in horizontal lines only: `writing-mode`, `text-orientation` and `text-combine-upright` were unknown, and the logical properties were placed by `direction` alone. Block, inline, flex, grid, table, multi-column and anchor layout are all written, after Taffy, in physical x and y, with width the inline size and height the block size.

## Decision

- A box whose lines run down it is laid out turned: the algorithms run unchanged in its frame, where x runs along its lines and y across them from the side its blocks start on. Each node's style is turned into the frame of its parent's content (sizes across, sides those of the frame, `position-area` and anchor sides with them), copies kept from layout to layout; writing the boxes turns each place, size, edge, scroll overflow and line fragment back onto the page from its parent's frame. vertical-lr mirrors what stands on each line, its glyphs' tops facing its block end.
- A box of another frame than its parent's (an orthogonal flow) lays its content out in its own: the space it is given and the size it takes are turned across at that one call, its lines as long as its container's definite height or else the initial containing block's, and it shrinks to fit in a block container, as Chrome's. An inline element of another writing mode is an inline-block. The root's writing mode turns the initial containing block.
- The cascade places a logical declaration by the element's writing mode and direction, read first as direction is; ui's sheet takes Chrome's logical margins and paddings.
- Paint draws a vertical line's fragment as the horizontal line it is in its frame, turned a quarter onto the page; a character standing upright (the CJK scripts under `mixed`, all under `upright`) is drawn unturned in a room of its font's ascent and descent, and `text-combine-upright: all` sets its text across one em, pressed into it when wider.

## Alternatives

- Write the algorithms in logical coordinates, as Blink's LayoutNG: every algorithm rewritten, where turning them keeps each one as it is and as Chrome-matched as it was.
- Draw a vertical box as a rotated horizontal one, a transform at paint: its boxes would not be where Chrome's are, nor size as an orthogonal flow sizes.
- Read the vertical metrics of `vhea` and `vmtx`: the faces ui ships have none, and Chrome falls back to the ascent and descent as ui does.

## Consequences

Blocks, lines, flex, grid, tables, columns, floats, positioned and anchored boxes, scrollers and orthogonal flows in every writing mode land where Chrome 157 puts them. What turning doesn't reach yet (selecting text of vertical lines, a viewport that scrolls under a vertical root, a positioned box escaping an orthogonal flow, vertical alternates of punctuation) is in 10-missing.
