# Document

![The settings panel, one Document in its window](settings.png "ember · settings")

```go
doc := ui.NewDocument(kit.Theme(), ui.Stylesheet(files, "game.css"))
doc.Body().Append(panel)
win.Add(doc)
```

| Option | Default | |
|---|---|---|
| `kit.Theme()` | none | ember's look: the kit's stylesheet and pictures |
| `ui.Stylesheet(fsys, name)` | none | a CSS file, after those before it; its `@font-face` files read from `fsys`, TrueType, OpenType, WOFF or WOFF2 |
| `ui.Images(fsys)` | none | where `url()` and an `img`'s `src` find pictures, by name, the last added first; no rules: `ui.Images(crypt.Art)` when a game's art lives apart from its sheets |
| `ui.CSS(text)` | none | a sheet written in the code, as a `<style>` element, after those before it; an `@font-face` in it is a warning without `ui.FetchFonts`, which fetches its file. `ui.CSS(text, ui.From(address))` for a sheet fetched from elsewhere: its `url()` are fetched from its address ([CSS](03-css.md#where-a-sheet-came-from)); `ui.Media(query)` has its rules apply only while the query holds, as a `style` element's `media` |
| `ui.FetchImages(fetch)` | none | gets the pictures no file system of the Document holds, off the frame: `ui.FetchImages(web.Images(base))` for pictures on the web |
| `ui.FetchFonts(fetch)` | none | gets the fonts of the `@font-face` rules no file system of the Document holds, off the frame, when text first needs them: `ui.FetchFonts(web.Fonts(base))` for a page's fonts ([below](#fonts-that-arent-files)) |
| `ui.FetchSheets(fetch)` | none | gets the sheets `@import` rules name, off the frame, each standing where its import is: `ui.FetchSheets(web.Sheets(base))` for a page's imports ([CSS](03-css.md#imported-sheets)) |
| `ui.SystemFonts()` | off | `font-family` also finds the fonts installed on the machine, by family name in any case (`font-family: Georgia, serif`), after the Document's own; read off the frame, text showing in the next family of the list until they land. Not in the browser. For pages a game does not write itself ([decision 0081](../99-decisions/0081-system-fonts-are-an-option-of-a-document.md)) |
| `ui.OverlayScrollbars()` | off | scrollbars drawn over the content, in no room of their own, as macOS and phones draw theirs: a box that scrolls keeps its whole width, where a classic bar takes 15 points. They scroll as classic ones and show whenever the content overflows, covering its last 8 points. For pages laid out for such bars ([decision 0083](../99-decisions/0083-overlay-scrollbars-are-an-option-of-a-document.md)) |
| `ui.ScrollingViewport()` | off | the viewport scrolls, as a browser's: `html` is as high as the page, and what overflows the window scrolls by the wheel, the page keys, a scrollbar and, with nothing focused, the arrows (40 points) and Space (a page, Shift+Space back), as in Chrome. The viewport takes `html`'s `overflow`, or `body`'s while `html`'s is visible, so `html { overflow: hidden }` stops it. A `position: fixed` box stays put, a sticky one sticks to the window, the canvas background's picture is placed in `html`'s box and scrolls with the page, and the whole window takes the pointer. For pages, not game screens ([decision 0084](../99-decisions/0084-a-scrolling-viewport-is-an-option-of-a-document.md)) |
| `ui.BrowserSheets()` | off | every sheet and style attribute of the Document is read as a browser reads it, wherever the option stands among the others: `-webkit-` names Chrome reads as plain ones are, another engine's names go without a warning, `@supports` is answered, `@layer` orders the cascade, a width is its content's (`box-sizing: content-box`), and the page's text is selected and copied (`user-select: auto`). For pages written for browsers; a game's sheet keeps its warnings ([CSS](03-css.md#browser-sheets), [decision 0085](../99-decisions/0085-browser-sheets-are-an-option-of-a-document.md)) |
| `ui.RightDragSelects()` | off | a drag of the right button selects text as one of the left does, and says so when it is let go (`ui.RightDragSelected`), in place of the right click; no browser does it ([Events](04-events.md#selected-text), [decision 0109](../99-decisions/0109-a-right-drag-selects-text-only-where-a-document-asks.md)) |
| `ui.WheelScroll(f)` | 1 | the wheel scrolls the Document `f` times what it says: 2 twice as far, a negative `f` the other way, 0 not at all; `doc.SetWheelScroll(f)` changes it while it shows. The factor of the Document in the window holds for those in its iframes. Only scrolling is scaled, `win.Input().Wheel()` stays as it comes: a reader's settings for the wheel ([decision 0110](../99-decisions/0110-a-document-scales-what-the-wheel-scrolls.md)) |
| `ui.InstantScrolls()` | off | the wheel, the keys and a click on a scrollbar's track move a scroll at once, where they glide as Chrome's smooth scrolling on the desktop: a notch for 12 frames down to 6, a key or a track for 9, on an ease-in-out curve, the next scroll going on from where the glide goes. A trackpad's two fingers always move at once, by their points. `doc.SetInstantScrolls(on)` changes it while it shows; the Document in the window rules those in its iframes. `scroll-behavior: smooth` still glides the program's scrolls ([decision 0169](../99-decisions/0169-the-users-scrolls-glide-on-chromes-curves-and-two-axis-snaps-land-where-both-areas-show.md)) |
| `ui.Emoji(set)` | none | colour emoji in text, from `twod.LoadEmoji` |

| Reading and changing it | |
|---|---|
| `doc.Body()`, `doc.HTML()` | where the game's elements go; the root |
| `doc.ByID(id)` | the first element with that `id` in tree order, or nil, as `getElementById` |
| `doc.Warnings()` | what the stylesheets and style attributes got wrong |
| `node.Box()` | an element's place on screen, after layout |
| `doc.BoxOf(n)` | its content box in window points, false until laid out or while hidden; where the game draws over an element that only holds room (give it `pointer-events: none` so the game's own widget gets the pointer); a resize reaches it the frame after |
| `doc.ElementAt(at)` | the innermost element under a point in window points, as `elementFromPoint`: what a click there finds, through `pointer-events: none`; nil where no box takes the pointer. For a Document drawn in another's canvas, `host.PointerOn(canvas)` is the point to ask it |
| `doc.BoxesOf(n)` | an element's four boxes, margin, border, padding and content (`ui.Boxes`), in the points of `BoxOf`; false as `BoxOf` |
| `doc.MatchedRules(n)` | the rules that match an element, the strongest first, as a tool shows them: each a `css.Match` with its origin (`css.Author`, `Agent`, `Inline` for the style attribute, `Hint` for presentational attributes), its sheet, selector, place in the sheet, `@media`, `@container` and `@layer`, and its declarations as written, `Overridden` on those another one wins over. Read-only, and made when asked: for tools, not for a frame's work |
| `doc.ComputedStyle(n)`, `doc.ComputedValue(n, name)` | every longhand of an element after the cascade, by name, then its custom properties (`css.Property`): the winning declaration's value with `var()` replaced and lengths in `px`, an inherited or initial value otherwise, `Declared` telling a declared one; `ComputedValue` is one of them. Colours and keywords stay as written |
| `doc.AddSheet(text, opts…)`, `doc.InsertSheet(before, text, opts…)`, `doc.StyleSheets()` | a stylesheet added while the Document shows, after the others or before one of them, and all of them in cascade order, each a `*ui.Sheet` to change ([CSS](03-css.md#sheets-changed-while-a-document-shows)) |
| `doc.SelectionRange(field)`, `doc.SetSelectionRange(field, from, to)` | a text field's own selection in characters of its value, as `selectionStart`, `selectionEnd` and `setSelectionRange`; with `doc.Value` and `doc.SetValue`, a menu's Cut, Copy, Paste and Select all |
| `doc.Selection()`, `doc.Select(anchor, focus)`, `doc.SelectAll()`, `doc.ClearSelection()` | the text selected outside the fields, read and set ([Events](04-events.md#selected-text)) |
| `doc.SelectionHTML()`, `win.SetClipboardHTML(html, text)`, `win.ClipboardHTML()` | the selection's markup, which a copy puts on the clipboard beside its text where the system's takes HTML |
| `doc.Layout(size)` | lays it out now, as each frame does; for tests |

One Document per window: added to it, it lays out at the window's size and draws over the 2D world each frame.

## Pictures that aren't files

![A portrait and a tiled wall, both fetched after the page showed](fetched-images.png "ember · fetched-images")

A `url()` background or an `img`'s `src` is looked for in the Document's file systems first. With `ui.FetchImages`, a name none of them holds goes to a fetcher instead of being a warning:

```go
doc := ui.NewDocument(
	ui.CSS(`#banner { background-image: url(https://art.example/banner.jpg); }`),
	ui.FetchImages(web.Images("https://art.example/news/")),
)
doc.Body().Append(ui.Div(ui.ID("banner")), ui.Img(ui.Attr("src", "today.png")))
```

| | |
|---|---|
| `ui.ImageFetcher` | `func(url string) ([]byte, error)`: the picture's bytes for the name as it is written in the sheet or the `src` |
| `web.Images(base)` (package `ui/web`) | a fetcher over HTTP: an `http` or `https` address as it is, anything else from `base`, as a browser resolves an `img`'s `src` from its page |

- A picture is asked for once, when its element is first laid out (an `img`) or painted (a background), never for an element that doesn't show. An `img` with a `srcset`, or in a `picture`, asks for the one it chooses, by the name its candidate is written with ([picture and source](02-elements/87-picture.md)).
- The fetcher runs off the frame, six pictures at a time, so it may take as long as it needs; it is called from several goroutines.
- Until a picture lands its element shows as one without a picture; then the Document paints again, and lays out again only what the picture's size moves: an `img` with a width and a height of its own, a background, a mask or a border picture move nothing. Nothing is shown while it loads: a placeholder is the game's, a background colour under the picture.
- A fetch that fails, or bytes that are no picture, are in `doc.Warnings()`, once.
- `web.Images` resolves a sheet's `url()` from `base` too, not from the sheet's own address: a sheet that lives elsewhere names its pictures in full.
- ui reads PNG; importing `ui/web` adds JPEG, GIF, WebP (lossy and lossless, with alpha; decision 0096), ICO, CUR and BMP (of any depth, run-length encoded or not, an ICO showing its largest picture as Chrome's does; decisions 0107, 0138). The `width` and `height` of the `source` an `img` chose its picture by size the `img` when it has either, as in Chrome. An animated GIF or WebP plays as Chrome plays it, its frames, delays and loops read as Chrome reads them: every element showing one address shows the same frame, from the Document's clock (decision 0138). Importing `ui/svg` adds SVG ([below](#svg-pictures)). A picture is known by its first bytes, not its name. A picture over `web.MaxBytes` (8 MB) or slower than `web.Timeout` (15 s) is an error.
- A fetched picture stays for the Document's life; nothing is cached between Documents or runs.

## Fonts that aren't files

An `@font-face` of a `Stylesheet` reads its file from the sheet's file system. With `ui.FetchFonts`, a face whose file no file system holds, and every face of a sheet given as text, is fetched instead of being a warning:

```go
doc := ui.NewDocument(
	ui.CSS(sheet, ui.From("https://site.example/css/site.css")),
	ui.FetchFonts(web.Fonts("https://site.example/")),
)
```

| | |
|---|---|
| `ui.FontFetcher` | `func(url, sheet string) ([]byte, error)`: the font's bytes for the address its `src` names, in full when the sheet says where it came from (`ui.From`); `sheet` is that address, what a request gives as its referrer |
| `web.Fonts(base)` (package `ui/web`) | a fetcher over HTTP, as `web.Images`: a font over `web.MaxFontBytes` (16 MB) or slower than `web.Timeout` is an error |
| `doc.FontsLoading()` | how many fonts are on their way |
| `doc.FontsReady(families, weight, italic, text)`, `doc.LoadFonts(…)` | whether that text waits for no font, and the same after asking for those it needs: what a browser's `document.fonts.check` and `load` answer |

- The first entry of `src` ui reads is taken: by its `format()` (`truetype`, `opentype`, `woff`, `woff2`, and their `-variations`), else by its extension; an `.eot`, an SVG font and a `tech(color-…)` are passed over. When it fails the next is tried. `local()` is looked for only with `ui.SystemFonts()`; a `data:` address needs no fetcher.
- A font is asked for once per Document, by all the faces that name it, and only when text needs a character its face draws: a face no text uses, or whose `unicode-range` holds none of the text, is never fetched. So a Google Fonts sheet of thirty subsets fetches the one or two the page is written in.
- Until it lands the text is laid out in the next family of its list and shown as the face's `font-display` has it, timed on the World's real time as Chrome times it ([decision 0148](../99-decisions/0148-variable-fonts-are-read-in-pure-go-and-font-display-is-timed-on-the-documents-clock.md)): `auto` and `block` keep it invisible, in its room, for 3 s, `fallback` and `optional` for 100 ms, `swap` not at all; then it shows in that family. A font landing after 3 s with `fallback`, or after 100 ms with `optional`, is not used: the text stays in the next family. Once fonts land the Document lays out and paints again, once a frame however many landed. A Document laid out but never drawn has no clock, and takes its fonts whenever they land. A character no family there draws while a face that would is on its way (an icon of an icon font) shows as nothing, and takes no room.
- `font-weight` takes one weight or two (a variable font's range), `auto` or none the font's own; the face nearest the text's weight and style is used, the last declared of two alike. A bold or italic the family lacks is made up from its face.
- A fetch that fails, or bytes that are no font, are in `doc.Warnings()` once, with the address, and the text stays as it is.

What is read of a font file:

| | |
|---|---|
| TrueType (`glyf`) and OpenType (`CFF `) | read as they are |
| WOFF, WOFF2 | unpacked to the same (zlib; Brotli, with WOFF2's packed `glyf`, `loca` and `hmtx`): a file whose sizes or offsets do not hold, or over 32 MB unpacked, is an error ([decision 0099](../99-decisions/0099-a-pages-fonts-are-fetched-and-woff2-is-read-in-pure-go.md)) |
| variable fonts (`fvar`, `avar`, `gvar`, `HVAR`) | drawn and measured at the instance the text asks: its `font-weight` and `font-stretch`, kept within the face's descriptors, on `wght` and `wdth`, then `font-variation-settings` on any axis. Read in pure Go (package `internal/varfont`), TrueType outlines only |
| colour fonts (`COLR`, `CBDT`, `sbix`, `SVG `) | their plain outlines when the file has some, in the text's colour; a file of bitmaps or SVG alone is not read |
| collections (`.ttc`, a WOFF2 collection), `CFF2`, `.eot`, SVG fonts | not read: a warning |

The fetcher is any function, so a game can read from its own cache or a save folder the same way. The page below has one that holds its two pictures back, a second and two, so they land while you watch:

```go
doc := ui.NewDocument(kit.Theme(), ui.FetchImages(pictures.fetch))
portrait := ui.Img(ui.Attr("src", "portrait.png"), ui.Style("width: 72px"))
wall := ui.Div(ui.Style("height: 72px; background-image: url(wall.png)"))
```

```
go run ./examples/92-ui-kit -element fetched-images
```

## SVG pictures

A Document shows an SVG file in an `img` or as a `url()` background once its format is imported, as a sound format is:

```go
import _ "github.com/lantervale/ember/ui/svg"
```

The import registers ember's `svg` reader with ui (`ui.RegisterVectorFormat`); a game that never imports it carries none of the reader, and an SVG is then a broken image with a warning. A file is an SVG by its name ending in `.svg` or by its bytes: an `svg` element after an optional XML declaration, doctype and comments. So a fetched picture needs no content type, and `web.Images` asks for `image/svg+xml` and hands the bytes over as they are.

**What size it takes.** An SVG may ask for a width and a height, have only a `viewBox`, or neither. ui sizes each as Chrome does:

| The file has | In an `img` with no size | As a background, `background-size: auto` |
|---|---|---|
| `width` and `height` | that size; with one side set, the other keeps the ratio | that size |
| a `viewBox` alone | the width of its block, the height by its ratio; nothing in a block that shrinks to its content (a float, an inline block); a flex row's width as a flex item with no size; its containing block's room between its insets when absolutely positioned | the largest size of its ratio the padding box holds, as `contain` |
| neither | 300 by 150; a side that is set leaves the other alone | the whole padding box |

`width` and `height` attributes on the `img`, CSS sizes, `object-fit`, and the other forms of `background-size` (lengths, percentages, `cover`, `contain`) apply as for any picture. A file with a `width` and a `height` and no `viewBox` has their ratio.

**How it is drawn.** The picture is rasterised once at the size it shows at, times the window's pixel ratio and the Document's `Zoom`, so it is sharp at any size. A repeated background draws one tile. A picture from a file system is drawn in the frame that first shows it; a fetched one off the frame, as it was read. When its size changes (a resize, a transition of `background-size`), the picture already drawn is stretched while the size moves, and drawn again once the size has held for a frame. A Document keeps up to 32 sizes of one picture, 16 million pixels in all; past that, a size no longer shown makes room, and if every one is showing the nearest is stretched. A picture larger than 8192 pixels on a side is drawn at that and stretched.

**What is drawn.** Paths and the basic shapes, groups, `use`, `symbol`, nested `svg` elements, transforms (the root's too), solid colours, linear and radial gradients, patterns, strokes with caps, joins and dashes, markers, fill rules, opacity, `clip-path` and `mask` naming a `clipPath` or a `mask` of the file, `clip-path` of a basic shape (`circle()`, `ellipse()`, `inset()` and the others, with a radius per corner) or a box, `mask` of a CSS linear or radial gradient (with `mask-mode`, `mask-repeat`, `mask-origin` and `mask-clip`), filters (`filter` elements of every primitive: blurs, offsets, floods, colour matrices, component transfers, composites, merges, blends, drop shadows, morphology, tiles, noise, convolution, displacement, diffuse and specular lighting, and `feImage` of an element of the file or a `data:` picture, and the CSS filter functions, after a filter element too), `mix-blend-mode` and `isolation`, `vector-effect: non-scaling-stroke`, style sheets of simple selectors with `!important`, the view box and `preserveAspectRatio`. `currentColor` is black. An `image` draws a PNG, JPEG, GIF, WebP or SVG held in a `data:` address, and nothing else: no file is fetched. Text is drawn in Go's faces whatever its `font-family` asks (the monospace one for a family naming one), so it is set close to Chrome's but not in the same letters ([decision 0136](../99-decisions/0136-svg-pictures-draw-clips-masks-patterns-markers-images-and-text.md)), with `letter-spacing`, `word-spacing`, `text-decoration` and `textPath`. Filters run on the CPU at the size the picture shows at ([decision 0150](../99-decisions/0150-svg-filters-run-on-the-cpu-over-the-pixels-they-show-at.md)), in a turned element's own space, as Chrome runs them ([decision 0161](../99-decisions/0161-svg-filters-run-in-the-elements-own-space-with-every-primitive-chrome-draws.md)). A file may be UTF-8 or UTF-16.

**What is not.** `foreignObject` and animation are left out of the picture, and so is a `mask` of an image other than a gradient. Each file that uses one is a warning in `doc.Warnings()`, once, naming the file and what was left out:

```
ui: image badge.svg: image/svg+xml is not drawn: foreignObject
```

A file that is no SVG, or over the reader's limits (16 MB, 100,000 elements or shapes), is a broken image with a warning, as any picture that can't be read. An SVG is never scripted or animated, and what it refers to outside itself is not fetched.

An SVG written in the page itself is an [`svg` element](02-elements/85-svg.md), `ui.Svg(markup, args…)`: the same reader, with `currentColor` the colour of the element's text.

Limits: a `border-image` from an SVG is drawn at the size the file asks for, one pixel a point. To use an SVG outside ui, read it with the `svg` package itself and make a texture of the picture: `p, err := svg.LoadPicture(fsys, "star.svg")`, `img, err := p.Rasterise(64, 64)`, `twod.NewTexture(img)`.

```
go run ./examples/92-ui-kit -element settings
```
