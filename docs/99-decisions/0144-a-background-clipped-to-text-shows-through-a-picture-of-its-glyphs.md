# 0144. A background clipped to text shows through a picture of its glyphs

- Status: Accepted
- Date: 2026-11-23
- Area: rendering
- Owner: user

## Context

`background-clip: text`, the way the web draws gradient headings, was refused: ui could clip a background to a box, a rounded box or a shape, never to the glyphs of the text an element holds.

## Decision

- An element with a background layer clipped to text paints the glyphs and decorations of its text, and of all it holds, opaque into a picture first (as a mask's, [0133](0133-a-masked-element-is-a-picture-cut-by-the-picture-of-its-mask.md)), and draws each layer clipped to text, its colour too when the last layer is, cut by that picture's alpha. The text itself draws after, over it, as in Chrome.
- `-webkit-text-fill-color` is read, under its own name, as the colour glyphs are filled with; a decoration of `currentColor` takes it, as Chrome draws it. `-webkit-background-clip` is `background-clip` under `ui.BrowserSheets()`.
- A range of draws cut by a picture is never recorded for replay: its picture is drawn only in the frame that made it.
- On a surface without pictures, and in hit testing, what is clipped to text is left out.

## Alternatives

- A glyph atlas sampled as the mask: no picture, but each layer drawn once a glyph, and a gradient spread across the element would have to be cut glyph by glyph.
- Drawing the text with the background as its fill: one pass, but text of other sizes and colours in the element, and its decorations, would each need the layer placed in the element's box.

## Consequences

Gradient and picture text draws as in Chrome, at the cost of one offscreen picture an element, painted again each frame it shows.
