# 0141. Keys and the arrows cross a frame; the frame fades and moves with its iframe

- Status: Accepted
- Date: 2026-11-23
- Area: api
- Owner: engine

## Context

Decision 0103 left a framed Document an island for the keys: an Escape pressed in a help pane never reached the `OnBack` of the screen showing it, and the arrows and a pad's stick, a game's way round a screen, stopped at the frame's edge. Its iframe was also drawn as a canvas, in its own colours whatever the `opacity` round it, and an iframe moved to another Document lost what it showed.

## Decision

- A key or a Back no callback of the framed Document stopped goes on from the iframe in the Document round it, `e.Node` there the iframe, as an event retargeted at a shadow root. The World event is emitted once, by the framed Document. A key or a Back with nothing focused in the frame is its body's, then the iframe's.
- An arrow or a stick that finds nothing of the frame that way moves the page's focus from the iframe's box; landing on an iframe, it goes on into its Document, to the element nearest the side it came in by (`control.Focus.MoveFrom`). Tab is as it was.
- A modal dialog of a framed Document holds that Document alone, as Chrome's.
- An element that is or holds a canvas or an iframe and has an `opacity` below 1 is drawn as one picture faded as a whole (`box.Box.Canvas`); a live part inside a group picture counts as covered, so it is painted in its place.
- An iframe moved to another Document within a frame keeps the Document it showed, as Chrome loads a moved frame again; one left out of every Document lets go of it as before.
- With an `aspect-ratio` and no size, an iframe fills its line as a picture of proportions alone does, the ratio holding between the boxes `box-sizing` names; a percentage height of an atomic inline is of its block's height where that is definite.

## Alternatives

- Keys kept inside each Document, as Chrome keeps a frame's `keydown` from its parent: a program would put the same `OnBack` on every Document it frames, and a framed pane would trap a pad's player.
- A World event per Document the key crosses: systems reading `KeyPressed` would see one key several times.
- Spatial navigation entering a frame from the element the focus left, mapped into the frame: more exact, but it needs each Document's boxes in the window's space, which the frame's own scroll and zoom move.

## Consequences

A screen answers Back and moves its focus the same whether a pane of it is framed or not, at the cost of one difference from Chrome's event model, which the guide states. A faded frame, like one covered, has its page painted each frame.
