# 0193. Vertical lines are selected in the flow of their lines

- Status: Accepted
- Date: 2026-11-23
- Area: api
- Owner: engine

## Context

Decision [0176](0176-vertical-writing-modes-are-laid-out-turned-into-their-own-frame.md) left a vertical line's fragments on the page, turned back from their frame, and paint draws each as the horizontal line it is in its frame. Selecting text read the fragments as horizontal ones: a press, a drag and a double click found characters across the line, the parts of a selection were measured across it, and paint, drawing a vertical line's parts apart, found no selection for them.

## Decision

- Finding a place, a press's, a drag's or a line move's, works in the flow of each fragment's lines: x along them from their start, y across them from where their blocks start (vertical-rl's right, vertical-lr's and sideways-lr's left, sideways-lr's lines running up). The search, the text's start before all its lines and its end after them, and the arrows' line moves are the horizontal ones, unchanged. A character standing measures its vertical advance along the line, as layout measures it.
- A selected part is measured along its line in the line's frame (`paint.LineTurn`, the frame paint draws in) and turned onto the page, so `SelectionRects` and paint share it; paint draws a vertical line's selection under the whole part, and each turned or standing piece of the line takes its colours where the part covers it.
- On vertical lines the arrows along them step through the text and those across go from line to line, by the block side: in vertical-rl Left is the next line, in vertical-lr and sideways-lr Right. A line move into a block of another writing mode goes to the next line in the text's order, at the place where the caret stands on the page, and looks for it on the page's whole points, as Chrome's do.

## Alternatives

- Hit-test and measure in each fragment's own frame alone: places of lines side by side would not compare, a press between two lines could not tell the nearer.
- Turn the point into each line's frame at paint's transform, as for CSS transforms: the same, and a second transform per fragment for what turning the rectangle does once.

## Consequences

A press, a drag, a double click and the arrows on vertical lines of every writing mode, turned or standing, take the places Chrome takes, and the selection is drawn where Chrome's range rectangles lie, a space past a line's end as on horizontal lines. The empty lines of preformatted vertical text, a br alone on a vertical line, and tabs kept in vertical lines are not marked yet (10-missing).
