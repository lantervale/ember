# Text

Words on screen: a title, a score, a chat line, a button's label. This section shows how ember draws them, and with which fonts.

| Page | What it shows |
|---|---|
| [Text](01-text.md) | The engine's faces, a font file, wrapping, measuring and shaping |
| [Bitmap font](02-bitmap-font.md) | Text drawn pixel for pixel from a picture of its characters |
| [Emoji](03-emoji.md) | Fallback faces, colour emoji from pictures, shortcodes |

## Faces and sizes

A face is a font: its letters, ready to draw. The engine ships five: `twod.Regular()`, `Bold()`, `Italic()`, `SmallCaps()` and `Mono()`. Text picks one with `Typeface`, and its size in points with `FontSize`. With neither, it is 16-point Regular.

## Your own fonts

`twod.LoadFace(files, "fonts/X.ttf")` reads a TrueType font, a `.ttf`, from a file system such as the game's embedded files. `twod.NewFace` takes the same bytes from anywhere.

Only TrueType is read: OpenType with CFF outlines, font collections and web fonts such as WOFF are refused with an error that says so. Keep the font's licence next to it.

## Fallbacks and emoji

A face draws only the characters it has. `With(twod.Fallback(other))` gives it faces to fall back on, character by character: that is how text holds emoji, other alphabets or symbols. Colour emoji are pictures, added with `twod.Emoji(set)`.

## Bitmap fonts

A bitmap font is a picture of its characters, drawn pixel for pixel, as old arcade games did. It suits pixel art, and needs no font file.

## Where text appears

- **In the 2D world:** `twod.NewText`, placed and drawn like any 2D entity.
- **In the UI:** the elements' text follows the CSS `font-family` and `font-size` of their sheet.
- **For a whole game:** a Document's `@font-face` and `font-family` give every element a font at once, as on the [Themes](../05-ui/09-walkthroughs/05-themes.md) page.
