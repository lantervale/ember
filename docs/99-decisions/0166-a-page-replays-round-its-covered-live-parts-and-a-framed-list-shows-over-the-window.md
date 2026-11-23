# 0166. A page replays round its covered live parts, and a framed list shows over the window

- Status: Accepted
- Date: 2026-11-23
- Area: rendering
- Owner: engine

## Context

Under [0052](0052-uiv2-paints-live-parts-over-a-replayed-frame.md), a live
part something covers (an iframe, a canvas or a caret under a box, turned,
or in an opacity's, a filter's or a mask's picture) repainted the whole
page every frame. And the open list of a select in an iframe was the
framed Document's paint, cut at the frame's box, where Chrome shows it
whole over the page.

## Decision

- A paint drawing live parts in their place marks where each lies in the
  canvas's drawing; a part in a picture marks instead the outermost
  stacking context drawing that picture. The page is recorded round those
  gaps (`twod.Canvas.KeepAround`), and a still frame replays the ranges in
  turn, each gap drawn anew between them (`ReplayAround`, `paint.FillGap`):
  the part under its own clips and transform, or the stacking context
  whole, its pictures with it. F3 counts "ui replays round live".
- A gap must leave the canvas's state as it found it; where one does not,
  what the replay drew is taken back and the page repaints.
- A Document in an iframe paints its open select's list apart; the
  window's Document draws it last, over everything, kept inside the
  window rather than the frame (`paint.Lists`, `paint.DrawLists`). Where
  the pointer is over such a list it is that Document's alone; while its
  list is open a framed Document hears the pointer anywhere, so a press
  outside closes the list.

## Alternatives

- Draw a covered part on top of the replay: wrong wherever it is covered.
- Record the page in per-subtree ranges and repaint any changed subtree in
  place: the general form, with range bookkeeping for every box, where
  gaps only exist round what is live.
- Hand a framed list to the host as an element of its own: it would have
  to mirror the select's options, state and events across Documents.

## Consequences

A page with a box over a frame, or a faded frame, costs a replay and the
frame's own (replayed) drawing: 18 µs and 35 allocations to 12 µs and 4 on
the benchmark page. A part in a picture redraws its whole stacking context
each frame, which is cheaper than the page but more than the part. A
framed list is drawn every frame by the window's Document, after its
replay.
