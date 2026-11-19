# 0101. Paint and hit testing skip the boxes out of view

- Status: Accepted
- Date: 2026-11-18
- Area: rendering
- Owner: engine

## Context

Paint culled each thing it drew against the view, and the hit test each box against the pointer, yet both walked every box to do it: on a page of 14,600 boxes many screens high, 4 of the 10.5 ms of a scroll frame, growing with the page and not with what shows.

## Decision

- Each box has an `Ink`: the bounds, in its own space, of every rectangle paint culls a part of it or of the boxes under it by (border boxes and fragments, shadows, outlines, text lines with their overhang and shadows). A box whose overflow cuts both axes bounds what it holds by its own.
- A box no rectangle can speak for is `Loose`, and so is every box above it, so the walk reaches it: an absolutely or fixed positioned one (its clips are not its parent's), a sticky one (a scroll moves it), a transformed one (drawn where the viewport doesn't cull), one cutting a single axis, a select and a dialog (their open list and modal layer are looked for wherever they lie). Its neighbours are still skipped.
- The walk skips a child that is not loose where its `Ink` misses the view its clip leaves, or a scan's point: the same test, in the same space, as the one that culls what it draws, so what is drawn, its order, its clips and the box hit are the same. Two things changed to make that exact: clips that don't meet show nothing, and a scroller's bars are culled by its box.
- `Ink` is worked out by paint for the whole tree, bottom-up, when the tree says it does not stand (`box.Tree.Inked`): a layout's tree starts without, and the Document clears it when css styled an element or a transition moved, since shadows, outlines and transforms ask for no layout. A scroll changes none of it.
- A frame that scrolled repaints, as before, now only what shows; a frame in which nothing changed replays the last, as before. Nothing is kept from one scroll offset for another.
- `EMBER_UI_CULL=off` walks every box; `=check` has every paint and hit test done both ways and a difference reported. The tests hold the skipping to the whole walk with it.

## Alternatives

- Bounds kept up by layout as it writes each box: layout doesn't read the styles paint shows (a transition's values) nor is it run for a change only paint sees, so the bounds would be stale exactly where they matter.
- Bounds refreshed for the changed elements only, told by css: the walk a hover costs would shrink from every box to a path, but each way a shown style changes (transitions, generated content, text) must then report itself, and one missed is a box not drawn. The whole pass is one walk reading each box once, 1.4 ms on the long page of the paint benchmark on a loaded machine, and runs only on frames that restyle or lay out.
- Fold positioned boxes into the bounds of the block that places them: fewer loose paths, but their place depends on the scroll of what lies between, and a fixed one on the viewport's.
- Paint a margin above and below the view and move the kept drawing while a scroll stays in it: a scroll frame would replay, but the drawing is not kept in a form that moves (the vertices are already placed), and what sticks or is fixed moves against it.

## Consequences

A scroll frame and a pointer frame cost what shows, not the page. A frame that restyles pays one pass over the boxes for their ink; a page with many positioned boxes walks the path to each.
