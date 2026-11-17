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
| `ui.Stylesheet(fsys, name)` | none | a CSS file, after those before it; its `@font-face` files read from `fsys` |
| `ui.Images(fsys)` | none | where `url()` and an `img`'s `src` find pictures, by name, the last added first; no rules: `ui.Images(crypt.Art)` when a game's art lives apart from its sheets |
| `ui.CSS(text)` | none | a sheet written in the code, as a `<style>` element, after those before it; an `@font-face` in it is a warning, fonts come with `Stylesheet`. `ui.CSS(text, ui.From(address))` for a sheet fetched from elsewhere: its `url()` are fetched from its address ([CSS](03-css.md#where-a-sheet-came-from)) |
| `ui.FetchImages(fetch)` | none | gets the pictures no file system of the Document holds, off the frame: `ui.FetchImages(web.Images(base))` for pictures on the web |
| `ui.SystemFonts()` | off | `font-family` also finds the fonts installed on the machine, by family name in any case (`font-family: Georgia, serif`), after the Document's own; read off the frame, text showing in the next family of the list until they land. Not in the browser. For pages a game does not write itself ([decision 0081](../99-decisions/0081-system-fonts-are-an-option-of-a-document.md)) |
| `ui.OverlayScrollbars()` | off | scrollbars drawn over the content, in no room of their own, as macOS and phones draw theirs: a box that scrolls keeps its whole width, where a classic bar takes 15 points. They scroll as classic ones and show whenever the content overflows, covering its last 8 points. For pages laid out for such bars ([decision 0083](../99-decisions/0083-overlay-scrollbars-are-an-option-of-a-document.md)) |
| `ui.ScrollingViewport()` | off | the viewport scrolls, as a browser's: `html` is as high as the page, and what overflows the window scrolls by the wheel, the page keys and a scrollbar. The viewport takes `html`'s `overflow`, or `body`'s while `html`'s is visible, so `html { overflow: hidden }` stops it. A `position: fixed` box stays put, a sticky one sticks to the window, the canvas background's picture is placed in `html`'s box and scrolls with the page, and the whole window takes the pointer. For pages, not game screens ([decision 0084](../99-decisions/0084-a-scrolling-viewport-is-an-option-of-a-document.md)) |
| `ui.BrowserSheets()` | off | every sheet and style attribute of the Document is read as a browser reads it, wherever the option stands among the others: `-webkit-` names Chrome reads as plain ones are, another engine's names go without a warning, `@supports` is answered, `@layer` orders the cascade, a width is its content's (`box-sizing: content-box`), and the page's text is selected and copied (`user-select: auto`). For pages written for browsers; a game's sheet keeps its warnings ([CSS](03-css.md#browser-sheets), [decision 0085](../99-decisions/0085-browser-sheets-are-an-option-of-a-document.md)) |
| `ui.Emoji(set)` | none | colour emoji in text, from `twod.LoadEmoji` |

| Reading and changing it | |
|---|---|
| `doc.Body()`, `doc.HTML()` | where the game's elements go; the root |
| `doc.ByID(id)` | the first element with that `id` in tree order, or nil, as `getElementById` |
| `doc.Warnings()` | what the stylesheets and style attributes got wrong |
| `node.Box()` | an element's place on screen, after layout |
| `doc.BoxOf(n)` | its content box in window points, false until laid out or while hidden; where the game draws over an element that only holds room (give it `pointer-events: none` so the game's own widget gets the pointer); a resize reaches it the frame after |
| `doc.ElementAt(at)` | the innermost element under a point in window points, as `elementFromPoint`: what a click there finds, through `pointer-events: none`; nil where no box takes the pointer. For a Document drawn in another's canvas, `host.PointerOn(canvas)` is the point to ask it |
| `doc.SelectionRange(field)`, `doc.SetSelectionRange(field, from, to)` | a text field's own selection in characters of its value, as `selectionStart`, `selectionEnd` and `setSelectionRange`; with `doc.Value` and `doc.SetValue`, a menu's Cut, Copy, Paste and Select all |
| `doc.Selection()`, `doc.Select(anchor, focus)`, `doc.SelectAll()`, `doc.ClearSelection()` | the text selected outside the fields, read and set ([Events](04-events.md#selected-text)) |
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

- A picture is asked for once, when its element is first laid out (an `img`) or painted (a background), never for an element that doesn't show.
- The fetcher runs off the frame, six pictures at a time, so it may take as long as it needs; it is called from several goroutines.
- Until a picture lands its element shows as one without a picture; then the Document lays out and paints again. Nothing is shown while it loads: a placeholder is the game's, a background colour under the picture.
- A fetch that fails, or bytes that are no picture, are in `doc.Warnings()`, once.
- `web.Images` resolves a sheet's `url()` from `base` too, not from the sheet's own address: a sheet that lives elsewhere names its pictures in full.
- ui reads PNG; importing `ui/web` adds JPEG and GIF, and importing `ui/svg` adds SVG ([below](#svg-pictures)). A picture over `web.MaxBytes` (8 MB) or slower than `web.Timeout` (15 s) is an error.
- A fetched picture stays for the Document's life; nothing is cached between Documents or runs.

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
| a `viewBox` alone | the width of its block, the height by its ratio; nothing in a block that shrinks to its content (a float, an inline block) | the largest size of its ratio the padding box holds, as `contain` |
| neither | 300 by 150; a side that is set leaves the other alone | the whole padding box |

`width` and `height` attributes on the `img`, CSS sizes, `object-fit`, and the other forms of `background-size` (lengths, percentages, `cover`, `contain`) apply as for any picture. A file with a `width` and a `height` and no `viewBox` has their ratio.

**How it is drawn.** The picture is rasterised once at the size it shows at, times the window's pixel ratio and the Document's `Zoom`, so it is sharp at any size. A repeated background draws one tile. A picture from a file system is drawn in the frame that first shows it; a fetched one off the frame, as it was read. When its size changes (a resize, a transition of `background-size`), the picture already drawn is stretched while the size moves, and drawn again once the size has held for a frame. A Document keeps up to 32 sizes of one picture, 16 million pixels in all; past that, a size no longer shown makes room, and if every one is showing the nearest is stretched. A picture larger than 8192 pixels on a side is drawn at that and stretched.

**What is drawn.** Paths and the basic shapes, groups, `use`, transforms, solid colours, linear and radial gradients, strokes with caps, joins and dashes, fill rules, opacity, style sheets of simple selectors, the view box and `preserveAspectRatio`. `currentColor` is black.

**What is not.** Text, `clip-path`, masks, filters, patterns, embedded images, markers and symbols are left out of the picture. Each file that uses one is a warning in `doc.Warnings()`, once, naming the file and what was left out:

```
ui: image badge.svg: image/svg+xml is not drawn: clip-path, text
```

A file that is no SVG, or over the reader's limits (16 MB, 100,000 elements or shapes), is a broken image with a warning, as any picture that can't be read. An SVG is never scripted or animated, and what it refers to outside itself is not fetched.

Limits: an `img` with a `viewBox` alone is not given a width as a flex item with no size or as an absolutely positioned box, where Chrome gives it the room there; give it a width. A `border-image` from an SVG is drawn at the size the file asks for, one pixel a point. To use an SVG outside ui, read it with the `svg` package itself and make a texture of the picture: `p, err := svg.LoadPicture(fsys, "star.svg")`, `img, err := p.Rasterise(64, 64)`, `twod.NewTexture(img)`.

```
go run ./examples/92-ui-kit -element settings
```
