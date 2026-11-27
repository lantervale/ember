# Text

Text is a 2D entity: words drawn with a typeface, at a size and a colour. Here a title, every face the engine ships, a pixel font read from its file, a paragraph wrapped to a width, and a clock that changes every frame.

![A title, six faces, a wrapped paragraph and a clock](text.png "ember · text")

## The program

`twod.NewText` makes white 16-point `twod.Regular` text. Here `Pivot(0, 0)` places each text by its top-left corner.

[main.go](../../examples/44-text/main.go)

### Options

| Option | What it does |
|---|---|
| `Typeface(face)` | The face to draw with. The engine ships `twod.Regular()`, `Bold()`, `Italic()`, `SmallCaps()` and `Mono()`; a game's own font is a face too, here the pixel font of the last label and the clock. |
| `FontSize(points)` | The size in points, fractions too: glyphs are rasterized at the size drawn, to a sixteenth of a point, so 13.44 is as sharp as 13. |
| `Color(c)` | The colour. |
| `Wrap(width)` | Breaks the text between words so no line is wider than `width` points. |
| `Baseline()` | Places the text by the start of its first baseline instead of its top-left; the pivot then only moves it across. Texts of different sizes on one line share the baseline. |
| `Tracking(points)` | Adds that much room after each character, as CSS `letter-spacing` does; a whole emoji counts as one, and a negative value tightens the text. The letters, digits and punctuation of a script whose letters join, such as Arabic, take none, as Chrome leaves them; their spaces do. `SetTracking` changes it. |
| `TextStroke(width)` | Draws each glyph as its outline, `width` points wide, in place of the filled glyph. The text drawn filled, then again stroked in another colour, is outlined text, as CSS's `-webkit-text-stroke`. |
| `TextShadow(c)` | Draws a shadow in `c`, a little down and right. |

### Measuring

- `Measure()` is the text's size in points once wrapped, and `Lines()` its lines.
- `twod.Measure(text, opts…)` measures a text without making an entity.

Measuring, wrapping and drawing all count tracking, so `Measure` agrees with what's drawn.

A space character a face lacks (an em space, a thin space…) takes the room a browser gives it in that face, an em, a fifth of one…, rather than a missing glyph's.

## A font of your own

`twod.LoadFace` reads a TrueType font, a `.ttf`, from a file system, here the example's embedded `fonts` folder. `twod.NewFace` takes the same bytes from anywhere else.

- TrueType is the one format: OpenType with CFF outlines (`.otf`), font collections and web fonts such as WOFF are refused with an error saying so.
- A colour font's glyphs (COLR layers and gradients, sbix and CBDT bitmaps, and SVG documents once `ui/svg` is imported) draw in colour, from its first palette, the text's colour standing for its foreground.
- A font's licence travels with it. Silkscreen comes under the SIL Open Font Licence, so its licence file sits next to it.

[font.go](../../examples/44-text/font.go)

### How glyphs are drawn

| Option | What it does |
|---|---|
| `twod.Pixelated()` | Draws the glyphs unsmoothed on the pixel grid, for a pixel font such as Silkscreen at its own sizes. `Pixelated()` reads it back. |
| `twod.ExactCoverage()` | Draws the glyphs as covered, without the stem darkening that gives text the weight of a browser's on macOS. `ExactCoverage()` reads it back. |

### Shaping

A face's OpenType features shape its text as a browser shapes it.

| Option | What it does |
|---|---|
| `twod.Features("smcp", "-liga", "salt=2")` | Turns features on or off, written as HarfBuzz writes them. Kerning, the common ligatures and contextual alternates are on unless turned off; a feature the face lacks does nothing. `SetFeatures` changes them. |
| `twod.RightToLeft()` | Draws a line of Arabic or Hebrew from its end, given in the order it is read: its letters joined, its marks on their bases, its brackets and other mirrored characters turned round. A line mixing directions is cut into runs first. `SetRightToLeft` changes it. |
| `twod.Between(before, after)` | Shapes the text as the piece of a longer one between those two, which are never drawn: an Arabic word drawn as two texts of different colours keeps its letters joined across the cut. `SetBetween` changes it. |
| `twod.Amid(lead, trail)` | Shapes the text as one with `lead` before it and `trail` after it, as a browser shapes elements of one font side by side, and draws and measures only its own glyphs. A ligature or kerned pair across its start belongs to the text before; one across its end is its own. `SetAmid` changes it. |

## Changing it

`SetText` replaces the words while the game runs, here with the time since the start.

[clock.go](../../examples/44-text/clock.go)

## Try it

[44-text](../../examples/44-text)
