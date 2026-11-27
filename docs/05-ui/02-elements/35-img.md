# img

![A framed panel picture at its own 24×24 size, then three times larger with sharp pixels](img.png "ember · img")

```go
doc := ui.NewDocument(ui.Images(fstest.MapFS{"panel.png": {Data: panelPNG}}))

ui.Img(ui.Attr("src", "panel.png"), ui.Attr("alt", "A panel"))
ui.Img(ui.Attr("src", "panel.png"), ui.Style("width: 72px; image-rendering: pixelated"))
```

| Attribute | |
|---|---|
| `src` | a PNG in the Document's `ui.Images` and stylesheets' file systems, the last added first; one missing is in `doc.Warnings()` |
| `srcset`, `sizes` | pictures for the screen's density or the slot's width, chosen as Chrome chooses them ([picture and source](87-picture.md)) |
| `loading` | `lazy`: no picture, and none asked for, until it is laid out within 1250 points of the viewport along the page and 625 across, and as near each scroll container round it, as a desktop Chrome on a fast network has it; then it keeps its picture. `eager` and anything else ask at once |
| `fetchpriority` | `high`: asked for before the other pictures the same layout asks for; `low`: after them, as in Chrome |
| `decoding` | read and left: a picture never shows before it is decoded, fetched ones off the frame |
| `alt` | its name for automation (an `aria-label` wins); a picture that isn't found draws Chrome's broken-image icon and the alt text, wrapped and clipped to the box, a silver frame round the icon without `alt`, nothing with `alt=""`; unsized, it's as wide as the icon and its text |

Its natural size is one point per pixel; with one side set, the other keeps the ratio.

- `width` and `height` attributes size it as `width` and `height` in a sheet do, in pixels or percent, under every rule of the sheets.
- An SVG (with `ui/svg` imported) takes the size its file asks for, and is drawn at the size it shows at ([SVG pictures](../01-document.md#svg-pictures)).
- A block-level `img` keeps its picture's size, as in Chrome.
- `object-fit`, `object-position` and `image-rendering: pixelated` apply, to every frame of an animated one; a pixelated picture snaps to device pixels, as in Chrome, so each texel covers whole pixels wherever the image stands, unless a fractional zoom makes a texel other than a whole number of pixels wide ([CSS](../03-css.md) says which texel a pixel then takes).

| Default CSS | |
|---|---|
| `img` | `display: inline-block` (Chrome: inline; ui has no `vertical-align` for inline pictures) |

Automation: named by its `alt`, recorded as a label; not recorded with neither `alt` nor `id`.

Asked for as Chrome asks, scrolled and in a scroll container, in `TestALazyPictureIsAskedForNearTheViewportAsChromes`, and first with a high priority in `TestAPictureOfHighFetchPriorityIsAskedForFirstAsChromes`; an animation out of view and back as Chrome's in `TestAnAnimationOutOfViewStandsAndComesBackAsChromes`.

```
go run ./examples/93-html-elements -element img
```
