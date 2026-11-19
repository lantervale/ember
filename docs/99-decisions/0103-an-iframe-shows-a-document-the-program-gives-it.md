# 0103. An iframe shows a Document the program gives it

- Status: Accepted
- Date: 2026-11-18
- Area: api
- Owner: engine

## Context

A browser built on ui shows pages inside pages. A Document drawn in another's `ui.Canvas` laid out and painted, but the two knew nothing of each other: an element over the canvas did not keep the pointer from the inner one, the wheel scrolled the outer one alone, Tab, the keys and the selection were each Document's own, and no `iframe` rule matched.

## Decision

- `ui.Iframe(child, args…)` is an `iframe` element holding a `*ui.Document` the program built; `doc.SetFrame(n, child)` swaps it, `doc.Frame(n)` reads it. ember fetches nothing: `src` and `srcdoc` are the program's to read.
- It is a replaced box as Chrome's: 300 by 150 with no ratio of its own, a 2px inset border, `width`, `height` and `frameborder` read as on old pages. The child lays out at its content box, which its `@media` rules are answered for, is clipped to it, and paints where the element stands, at its Document's zoom.
- The Document around it gates the input: the pointer is the child's only where the element is what the hit test finds, the wheel is the child's while it can scroll that way, the keys are the child's while the element holds the focus, Tab walks in and out both ways, and one Document of the tree holds a selection at a time.
- A frame out of view is not drawn (decision 0101), and a frame nothing changed in replays inside a Document that replays. Automation ids of the child are under the element's.
- The element holds its Document only while it is in one: taken out, it lets go of it at the next frame, as `SetFrame(n, nil)`.
- A panic of the child is the program's, as a panic of any system or canvas: it reaches the frame loop and the crash log. Only an iframe built with `ui.Guarded()` stops it: the frame empties, `doc.FrameBroken(n)` gives the panic, its stack is a line of the error log, and the canvas is put back as it was (`twod.Canvas.Try`). The guard covers what runs inside the child's frame: layout, input, callbacks, paint.

## Alternatives

- Two hooks on a Document, a pointer gate and a wheel hand-over, with the program drawing the child in a canvas: less in the engine, but every program would write the focus, Tab and selection rules again, and CSS would still not see an `iframe`.
- ember loading the child from `src`: the engine has no network, no page parser and no say on what a program may load.
- Every iframe guarded: a bug of the program's own screens would show as an empty box and a line in a log instead of a crash at the line that caused it.
- The program recovering round the child itself: it cannot, the child is drawn by the Document round it, and a recover higher up loses that Document's frame and leaves the canvas with the child's clips and transform.
- One Document holding both trees: styles, scripts' views and scrolling of two pages must stay apart, which is what a Document already is.

## Consequences

A page view, a help pane or a tool is a Document in a frame, nested as deep as the program likes, at the cost of one more thing the interaction code asks about: whether a Document is framed. What is left open is in the guide's missing list: popups and modal dialogs of a framed Document stay inside its box, arrows and Back do not cross a frame.
