# 0194. A float after text is placed as its lines are broken

- Status: Accepted
- Date: 2026-11-23
- Area: rendering
- Owner: engine

## Context

A float written among a block's text (`words <span style="float: left">…</span> more`) was lifted out of its lines and placed before them, at the block's top, as a float before the text is. CSS 2's float rules put its top no higher than the line box holding its place in the text, and Chrome places it on that line when it fits beside what stands before it there, else below the line, which keeps its room. A float is laid out by its container's block layout, while the lines are broken by the inline content's own, run several times per layout as a block is measured.

## Decision

- A float with shown content before it in its lines keeps a place among them (the run it stands before) and is laid out by its container as before, but placed by the lines: as they are broken, on the line its place falls on (at its break, the line before unless that is a forced break) when what stands there before it, its trailing spaces left out, and its margin box fit the line's room, or nothing stands before it; the line is then broken again round it. Otherwise it goes below the line, at the line's bottom, and every float after it on that line too, as Chrome's.
- The float context the lines start from is copied before them, and each breaking of the lines starts from the copy, so measuring a block several times places its floats once.
- A float on the first line rises with a trimmed first line (decision [0192](0192-text-box-trim-follows-chromes-lines-into-columns-and-floats.md)); one placed below it is already where the trimmed lines are.

## Alternatives

- Placing the floats in the block layout and breaking the lines in pieces between them: each piece would need the room of the lines before, which only the breaking knows.
- Marking floats in the pieces, as the anchors of positioned boxes are: a float would then take part in breaking and in bidi reordering, where Chrome's takes no room in the line.

## Consequences

Floats after text, several on a line, those too wide for the rest of their line and floats after a `br` stand where Chrome puts them, left, right and right to left. A float with only collapsible space before it is still placed before the lines, as it was.
