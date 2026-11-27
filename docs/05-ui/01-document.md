# Document

A Document is one screen of UI: a tree of elements, the stylesheets that style it, and where its pictures and fonts come from. Add it to a window: it lays out at the window's size and draws over the 2D world every frame. A window shows one Document, or several stacked (a screen and a dialog of its own, for instance).

![The settings panel, one Document in its window](settings.png "ember · settings")

```go
doc := ui.NewDocument(kit.Theme(), ui.Stylesheet(files, "game.css"))
doc.Body().Append(panel)
win.Add(doc)
```

## Options

`ui.NewDocument` takes options. Each is off, or empty, unless you pass it.

### Styles and pictures

| Option | What it adds |
|---|---|
| `kit.Theme()` | ember's look: the kit's stylesheet and pictures |
| `ui.Stylesheet(fsys, name)` | a CSS file, after the sheets before it. Its `@font-face` files are read from `fsys`: TrueType, OpenType, WOFF or WOFF2 |
| `ui.CSS(text)` | a sheet written in the code, as a `<style>` element, after the sheets before it. An `@font-face` in it is a warning unless `ui.FetchFonts` fetches its file |
| `ui.CSS(text, ui.From(address))` | a sheet fetched from elsewhere: its `url()`s are fetched from its address ([CSS](03-css.md#where-a-sheet-came-from)) |
| `ui.CSS(text, ui.Media(query))` | a sheet whose rules apply only while the query holds, as a `style` element's `media` |
| `ui.Images(fsys)` | where `url()` and an `img`'s `src` find pictures, by name, the last added first. It holds no rules: `ui.Images(crypt.Art)` when a game's art lives apart from its sheets |
| `ui.BrowserSheets()` | every sheet and style attribute is read as a browser reads it ([below](#browser-sheets)) |
| `ui.Emoji(set)` | colour emoji in text, from `twod.LoadEmoji` |

### What no file system holds

| Option | What it fetches or finds |
|---|---|
| `ui.FetchImages(fetch)` | the pictures no file system of the Document holds, off the frame: `ui.FetchImages(web.Images(base))` for pictures on the web ([below](#pictures-that-arent-files)) |
| `ui.FetchFonts(fetch)` | the fonts of the `@font-face` rules no file system holds, off the frame, when text first needs them: `ui.FetchFonts(web.Fonts(base))` for a page's fonts ([below](#fonts-that-arent-files)) |
| `ui.FetchSheets(fetch)` | the sheets `@import` rules name, off the frame, each standing where its import is: `ui.FetchSheets(web.Sheets(base))` ([CSS](03-css.md#imported-sheets)) |
| `ui.SystemFonts()` | the fonts installed on the machine ([below](#installed-fonts)) |

### Scrolling and selecting

| Option | Default | What it changes |
|---|---|---|
| `ui.ScrollingViewport()` | off | the window scrolls, as a browser's ([below](#a-scrolling-viewport)) |
| `ui.OverlayScrollbars()` | off | scrollbars drawn over the content, as macOS and phones draw theirs ([below](#overlay-scrollbars)) |
| `ui.WheelScroll(f)` | 1 | the wheel scrolls `f` times what it says ([below](#wheel-and-glide)) |
| `ui.InstantScrolls()` | off | the wheel, the keys and a click on a scrollbar's track scroll at once instead of gliding ([below](#wheel-and-glide)) |
| `ui.RightDragSelects()` | off | a drag of the right button selects text, as the left button's does ([Events](04-events.md#selected-text), [decision 0109](../99-decisions/0109-a-right-drag-selects-text-only-where-a-document-asks.md)) |

### Browser sheets

`ui.BrowserSheets()` is for pages written for browsers. It applies to every sheet and style attribute of the Document, wherever it stands among the other options:

- `-webkit-` names that Chrome reads as plain ones are read so;
- another engine's prefixed names are dropped without a warning;
- `@supports` is answered and `@layer` orders the cascade;
- a width is its content's (`box-sizing: content-box`);
- the page's text can be selected and copied (`user-select: auto`).

Without it, a game's sheet keeps its warnings ([CSS](03-css.md#browser-sheets), [decision 0085](../99-decisions/0085-browser-sheets-are-an-option-of-a-document.md)).

### Installed fonts

With `ui.SystemFonts()`, `font-family` also finds the fonts installed on the machine, by family name in any case (`font-family: Georgia, serif`), after the Document's own. It is for pages a game does not write itself ([decision 0081](../99-decisions/0081-system-fonts-are-an-option-of-a-document.md)), and not in the browser.

- Installed fonts are read off the frame. Until they land, text shows in the next family of the list.
- A character no face of the list has is drawn with an installed face that has it, as Chrome asks the platform for one: Hiragino, Apple SD Gothic Neo, Geeza Pro… on macOS. The face is chosen by script, by the text's `lang` for Han (Japanese, Korean, Simplified or Traditional Chinese), and among serif faces under `serif` ([decision 0227](../99-decisions/0227-what-no-face-has-falls-back-to-the-machines-faces-as-chrome-asks-the-platform.md)).

### A scrolling viewport

With `ui.ScrollingViewport()`, the viewport scrolls as a browser's. It is for pages, not game screens ([decision 0084](../99-decisions/0084-a-scrolling-viewport-is-an-option-of-a-document.md)).

- `html` is as high as the page. What overflows the window scrolls by the wheel, the page keys, a scrollbar and, with nothing focused, the arrows (40 points) and Space (a page; Shift+Space goes back), as in Chrome.
- The viewport takes `html`'s `overflow`, or `body`'s while `html`'s is `visible`. So `html { overflow: hidden }` stops it.
- A `position: fixed` box stays put and a sticky one sticks to the window.
- The canvas background's picture is placed in `html`'s box and scrolls with the page.
- The whole window takes the pointer.
- A root of a vertical writing mode, its own or `body`'s, scrolls along its blocks: a `vertical-rl` page from the right ([decision 0221](../99-decisions/0221-a-vertical-root-scrolls-its-viewport-along-its-blocks.md)).

### Overlay scrollbars

With `ui.OverlayScrollbars()`, scrollbars are drawn over the content and take no room of their own. A box that scrolls keeps its whole width, where a classic bar takes 15 points. They scroll as classic ones do and show whenever the content overflows, covering its last 8 points. Use it for pages laid out for such bars ([decision 0083](../99-decisions/0083-overlay-scrollbars-are-an-option-of-a-document.md)).

### Wheel and glide

`ui.WheelScroll(f)` scales what the wheel scrolls: 2 is twice as far, a negative `f` the other way, 0 not at all. It is a reader's setting for the wheel ([decision 0110](../99-decisions/0110-a-document-scales-what-the-wheel-scrolls.md)).

- `doc.SetWheelScroll(f)` changes it while the Document shows.
- The factor of the Document in the window holds for the Documents in its iframes.
- Only scrolling is scaled: `win.Input().Wheel()` stays as it comes.

By default, the wheel, the keys and a click on a scrollbar's track glide, as Chrome's smooth scrolling does on the desktop: a notch for 12 frames down to 6, a key or a track for 9, on an ease-in-out curve. The next scroll goes on from where the glide is. `ui.InstantScrolls()` moves them at once ([decision 0169](../99-decisions/0169-the-users-scrolls-glide-on-chromes-curves-and-two-axis-snaps-land-where-both-areas-show.md)).

- A trackpad's two fingers and a smooth wheel's turns always move at once, by their points. They are told on macOS by AppKit, on Windows by a turn of part of a notch, on Wayland by one off the compositor's notches, and on the web by pixels that aren't whole notches. X11's wheel is always notches.
- `doc.SetInstantScrolls(on)` changes it while the Document shows. The Document in the window rules those in its iframes.
- `scroll-behavior: smooth` still glides the program's own scrolls.

## Reading and changing it

| The tree | |
|---|---|
| `doc.Body()`, `doc.HTML()` | where the game's elements go; the root |
| `doc.ByID(id)` | the first element with that `id` in tree order, or nil, as `getElementById` |
| `doc.Warnings()` | what the stylesheets and style attributes got wrong |
| `doc.Layout(size)` | lays it out now, as each frame does; for tests |

| Boxes on screen | |
|---|---|
| `node.Box()` | an element's place on screen, after layout |
| `doc.BoxOf(n)` | its content box in window points; false until laid out or while hidden. A resize reaches it the frame after. Use it to draw over an element that only holds room, and give that element `pointer-events: none` so the game's own widget gets the pointer |
| `doc.BoxesOf(n)` | an element's four boxes, margin, border, padding and content (`ui.Boxes`), in the points of `BoxOf`; false as `BoxOf` |
| `doc.ElementAt(at)` | the innermost element under a point in window points, as `elementFromPoint`: what a click there finds, through `pointer-events: none`; nil where no box takes the pointer. For a Document drawn in another's canvas, `host.PointerOn(canvas)` is the point to ask it |

| Styles, for tools | |
|---|---|
| `doc.MatchedRules(n)` | the rules that match an element, the strongest first, as a browser's tools show them. Each is a `css.Match`: its origin (`css.Author`, `Agent`, `Inline` for the style attribute, `Hint` for presentational attributes), its sheet, selector, place in the sheet, `@media`, `@container`, `@scope` and `@layer`, and its declarations as written, `Overridden` on those another one wins over. Read-only and made when asked, so not for a frame's work |
| `doc.ComputedStyle(n)`, `doc.ComputedValue(n, name)` | every longhand of an element after the cascade, by name, then its custom properties (`css.Property`): the winning declaration's value with `var()` replaced and lengths in `px`, else an inherited or initial value; `Declared` tells a declared one. `ComputedValue` is one of them. Colours and keywords stay as written |
| `doc.AddSheet(text, opts…)`, `doc.InsertSheet(before, text, opts…)`, `doc.StyleSheets()` | a stylesheet added while the Document shows, after the others or before one of them, and all of them in cascade order, each a `*ui.Sheet` to change ([CSS](03-css.md#sheets-changed-while-a-document-shows)) |

| Selection | |
|---|---|
| `doc.SelectionRange(field)`, `doc.SetSelectionRange(field, from, to)` | a text field's own selection in characters of its value, as `selectionStart`, `selectionEnd` and `setSelectionRange`. With `doc.Value` and `doc.SetValue`, enough for a menu's Cut, Copy, Paste and Select all |
| `doc.Selection()`, `doc.Select(anchor, focus)`, `doc.SelectAll()`, `doc.ClearSelection()` | the text selected outside the fields, read and set ([Events](04-events.md#selected-text)) |
| `doc.SelectionHTML()`, `win.SetClipboardHTML(html, text)`, `win.ClipboardHTML()` | the selection's markup as Chrome's copy writes it, with the styles its text shows in inline ([decision 0224](../99-decisions/0224-a-rich-copy-writes-chromes-markup.md)). A copy puts it on the clipboard beside the text where the system takes HTML: macOS, Windows, the web |

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
| `web.Images(base)` (package `ui/web`) | a fetcher over HTTP: an `http` or `https` address as it is, anything else from `base`, as a browser resolves an `img`'s `src` from its page. It resolves a sheet's `url()` from `base` too, not from the sheet's own address, so a sheet that lives elsewhere names its pictures in full. A picture over `web.MaxBytes` (8 MB) or slower than `web.Timeout` (15 s) is an error |

The fetcher is any function, so a game can read from its own cache or a save folder the same way. The page above has one that holds its two pictures back, a second and two, so they land while you watch:

```go
doc := ui.NewDocument(kit.Theme(), ui.FetchImages(pictures.fetch))
portrait := ui.Img(ui.Attr("src", "portrait.png"), ui.Style("width: 72px"))
wall := ui.Div(ui.Style("height: 72px; background-image: url(wall.png)"))
```

```
go run ./examples/92-ui-kit -element fetched-images
```

### When a picture is fetched

- A picture is asked for once, when its element is first laid out (an `img`) or painted (a background), never for an element that doesn't show.
- An `img` with a `srcset`, or in a `picture`, asks for the one it chooses, by the name its candidate is written with ([picture and source](02-elements/87-picture.md)).
- An `img` with `loading="lazy"` waits until it is laid out within 1250 points of the viewport along the page and 625 across, and as near each scroll container it is in, as a desktop Chrome does ([img](02-elements/35-img.md), [decision 0167](../99-decisions/0167-pictures-load-play-and-choose-again-when-a-desktop-chrome-does.md)).
- The fetcher runs off the frame, six pictures at a time, so it may take as long as it needs. It is called from several goroutines.
- Of the pictures a layout asks for, those of an `img` with `fetchpriority="high"` go first and `low` ones last, as in Chrome.
- A fetched picture stays for the Document's life. Nothing is cached between Documents or runs.

### While it loads, and when it fails

- Until a picture lands, its element shows as one without a picture. Nothing is shown in its place: a placeholder is the game's, such as a background colour under the picture.
- When it lands, the Document paints again. It lays out again only what the picture's size moves: an `img` with a width and a height of its own, a background, a mask or a border picture move nothing.
- A fetch that fails, or bytes that are no picture, are in `doc.Warnings()`, once.

### Formats

A picture is known by its first bytes, not its name.

| Format | Read when |
|---|---|
| PNG | always |
| JPEG, GIF, WebP (lossy and lossless, with alpha; [decision 0096](../99-decisions/0096-webp-pictures-are-read-with-the-pictures-of-the-web.md)) | `ui/web` is imported |
| ICO, CUR, BMP (of any depth, run-length encoded or not; an ICO shows its largest picture, as Chrome's does; decisions 0107, 0138) | `ui/web` is imported |
| SVG | `ui/svg` is imported ([below](#svg-pictures)) |

- The `width` and `height` of the `source` an `img` chose its picture by size the `img` when it has either, as in Chrome.
- An animated GIF or WebP plays as Chrome plays it, its frames, delays and loops read as Chrome reads them. Every element showing one address shows the same frame, from the Document's clock (decision 0138).
- It begins when first drawn and stands still while no paint draws it, scrolled out of view or hidden. Drawn again within the play it left, it comes back in step; past it, the next play begins, as in Chrome (decision 0167).
- Its frames past 2^25 pixels in all are drawn as they show. A GIF frame reaching past its picture is cut, a first one making the picture as large as it reaches, as Chrome draws them.

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

### Which file, and when

- The first entry of `src` that ui reads is taken: by its `format()` (`truetype`, `opentype`, `woff`, `woff2`, and their `-variations`), else by its extension. An `.eot`, an SVG font and a `tech(color-…)` are passed over. When one fails, the next is tried.
- `local()` is looked for only with `ui.SystemFonts()`. A `data:` address needs no fetcher.
- A font is asked for once per Document, by all the faces that name it, and only when text needs a character its face draws. A face no text uses, or whose `unicode-range` holds none of the text, is never fetched. So a Google Fonts sheet of thirty subsets fetches the one or two the page is written in.
- `font-weight` takes one weight or two (a variable font's range); `auto` or none takes the font's own. The face nearest the text's weight and style is used, the last declared of two alike. A bold or italic the family lacks is made up from its face.

### While it loads, and when it fails

Until a font lands, the text is laid out in the next family of its list and shown as the face's `font-display` says, timed on the World's real time as Chrome times it ([decision 0148](../99-decisions/0148-variable-fonts-are-read-in-pure-go-and-font-display-is-timed-on-the-documents-clock.md)):

| `font-display` | Text invisible, in its room, for | A font landing later |
|---|---|---|
| `auto`, `block` | 3 s | is used |
| `fallback` | 100 ms | is used up to 3 s; after that, the text stays in the next family |
| `optional` | 100 ms | is used within 100 ms; after that, the text stays in the next family |
| `swap` | not at all | is used |

- Once fonts land, the Document lays out and paints again, once a frame however many landed.
- A Document laid out but never drawn has no clock, and takes its fonts whenever they land.
- A character no family there draws, while a face that would is on its way (an icon of an icon font), shows as nothing and takes no room.
- A fetch that fails, or bytes that are no font, are in `doc.Warnings()` once, with the address, and the text stays as it is.

### What is read of a font file

| File | Read as |
|---|---|
| TrueType (`glyf`) and OpenType (`CFF `) | they are |
| WOFF, WOFF2 | unpacked to the same (zlib; Brotli, with WOFF2's packed `glyf`, `loca` and `hmtx`). A file whose sizes or offsets do not hold, or over 32 MB unpacked, is an error ([decision 0099](../99-decisions/0099-a-pages-fonts-are-fetched-and-woff2-is-read-in-pure-go.md)) |
| variable fonts (`fvar`, `avar`, `gvar`, `HVAR`) | drawn and measured at the instance the text asks: its `font-weight` and `font-stretch`, kept within the face's descriptors, on `wght` and `wdth`, then `font-variation-settings` on any axis. Read in pure Go (package `internal/varfont`), TrueType outlines only |
| colour fonts (`COLR`, `CBDT`, `sbix`, `SVG `) | their plain outlines when the file has some, in the text's colour; a file of bitmaps or SVG alone is not read |
| collections (`.ttc`, a WOFF2 collection), `CFF2`, `.eot`, SVG fonts | not read: a warning |

## SVG pictures

A Document shows an SVG file in an `img` or as a `url()` background once its format is imported, as a sound format is:

```go
import _ "github.com/lantervale/ember/ui/svg"
```

- The import registers ember's `svg` reader with ui (`ui.RegisterVectorFormat`). A game that never imports it carries none of the reader, and an SVG is then a broken image with a warning.
- A file is an SVG by its name ending in `.svg`, or by its bytes: an `svg` element after an optional XML declaration, doctype and comments. So a fetched picture needs no content type: `web.Images` asks for `image/svg+xml` and hands the bytes over as they are.
- An SVG written in the page itself is an [`svg` element](02-elements/85-svg.md), `ui.Svg(markup, args…)`: the same reader, with `currentColor` the colour of the element's text.

### What size it takes

An SVG may ask for a width and a height, have only a `viewBox`, or neither. ui sizes each as Chrome does:

| The file has | In an `img` with no size | As a background, `background-size: auto` |
|---|---|---|
| `width` and `height` | that size; with one side set, the other keeps the ratio | that size |
| a `viewBox` alone | the width of its block, the height by its ratio; nothing in a block that shrinks to its content (a float, an inline block); a flex row's width as a flex item with no size; its containing block's room between its insets when absolutely positioned | the largest size of its ratio the padding box holds, as `contain` |
| neither | 300 by 150; a side that is set leaves the other alone | the whole padding box |

`width` and `height` attributes on the `img`, CSS sizes, `object-fit`, and the other forms of `background-size` (lengths, percentages, `cover`, `contain`) apply as for any picture. A file with a `width` and a `height` and no `viewBox` has their ratio.

### How it is drawn

- The picture is rasterised once at the size it shows at, times the window's pixel ratio and the Document's `Zoom`, so it is sharp at any size. A repeated background draws one tile.
- A picture from a file system is drawn in the frame that first shows it; a fetched one off the frame, as it was read.
- When its size changes (a resize, a transition of `background-size`), the picture already drawn is stretched while the size moves, and drawn again once the size has held for a frame.
- A Document keeps up to 32 sizes of one picture, 16 million pixels in all. Past that, a size no longer shown makes room; if every one is showing, the nearest is stretched.
- A picture larger than 8192 pixels on a side is drawn at that and stretched.
- A `border-image` from an SVG is drawn at the size the file asks for, one pixel a point.

### What is drawn

| Area | Drawn |
|---|---|
| Shapes and structure | paths and the basic shapes, groups, `use`, `symbol`, nested `svg` elements, transforms (the root's too), the view box and `preserveAspectRatio` |
| Paint | solid colours, linear and radial gradients, patterns, strokes with caps, joins and dashes, markers, fill rules, opacity, `vector-effect: non-scaling-stroke`. `currentColor` is black |
| Clips and masks | `clip-path` and `mask` naming a `clipPath` or a `mask` of the file; `clip-path` of a basic shape (`circle()`, `ellipse()`, `inset()` and the others, with a radius per corner) or a box; `mask` of a CSS linear or radial gradient (with `mask-mode`, `mask-repeat`, `mask-origin` and `mask-clip`) |
| Filters | `filter` elements of every primitive: blurs, offsets, floods, colour matrices, component transfers, composites, merges, blends, drop shadows, morphology, tiles, noise, convolution, displacement, diffuse and specular lighting, and `feImage` of an element of the file or a `data:` picture; the CSS filter functions, after a filter element too. Filters run on the CPU at the size the picture shows at ([decision 0150](../99-decisions/0150-svg-filters-run-on-the-cpu-over-the-pixels-they-show-at.md)), in a turned element's own space, as Chrome runs them ([decision 0161](../99-decisions/0161-svg-filters-run-in-the-elements-own-space-with-every-primitive-chrome-draws.md)) |
| Compositing | `mix-blend-mode` and `isolation` |
| Style sheets | simple selectors, with `!important` |
| Images | an `image` of a PNG, JPEG, GIF, WebP or SVG held in a `data:` address, and nothing else: no file is fetched |
| Text | in Go's faces whatever its `font-family` asks (the monospace one for a family naming one), so it is set close to Chrome's but not in the same letters ([decision 0136](../99-decisions/0136-svg-pictures-draw-clips-masks-patterns-markers-images-and-text.md)); with `letter-spacing`, `word-spacing`, `text-decoration` and `textPath` |
| Encoding | UTF-8 or UTF-16 |

### What is not

`foreignObject`, animation, and a `mask` of an image other than a gradient are left out of the picture. Each file that uses one is a warning in `doc.Warnings()`, once, naming the file and what was left out:

```
ui: image badge.svg: image/svg+xml is not drawn: foreignObject
```

- A file that is no SVG, or over the reader's limits (16 MB, 100,000 elements or shapes), is a broken image with a warning, as any picture that can't be read.
- An SVG is never scripted or animated, and what it refers to outside itself is not fetched.

To use an SVG outside ui, read it with the `svg` package itself and make a texture of the picture: `p, err := svg.LoadPicture(fsys, "star.svg")`, `img, err := p.Rasterise(64, 64)`, `twod.NewTexture(img)`.

```
go run ./examples/92-ui-kit -element settings
```
