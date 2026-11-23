# picture and source

![The 24×24 panel drawn at twice its size, the picture a source offers at half a pixel a point](picture.png "ember · picture")

```go
ui.Picture(
	ui.Source(ui.Attr("media", "(min-width: 100px)"), ui.Attr("srcset", "panel.png 0.5x")),
	ui.Img(ui.Attr("src", "panel.png"), ui.Attr("alt", "A panel")),
)
```

An `img` chooses its picture as Chrome does when it has a `srcset` or stands in a `picture` ([decision 0115](../../99-decisions/0115-an-img-chooses-its-picture-as-chrome-does.md)): from the first `source` before it whose `srcset` offers one, whose `media` holds and whose `type` ui reads, else from its own `srcset` and `src`. `ui.PictureOf(n)` is the picture it shows, Chrome's `currentSrc`.

| Attribute | |
|---|---|
| `srcset` (an `img` or a `source`) | pictures and what each says of itself: `door.png 2x` a density, `door-800.png 800w` a width in pixels, a bare address 1x. An address runs to the next space, so it may hold commas, as a `data:` URL does; a candidate whose descriptors don't make sense is left out |
| `sizes` (an `img` or a `source`) | the width of the slot a picture of widths is chosen for: entries apart by commas, the first whose media condition holds giving its length (`(max-width: 600px) 100vw, calc(50vw - 2rem)`), `100vw` when none holds. Font units are the initial font's (16 pixels), viewport units the Document's viewport. `auto` stands for the width the `img` is laid out at when its `loading` is `lazy` |
| `media` (a `source`) | a media query list, answered for the Document's viewport as `@media` is |
| `type` (a `source`) | a media type ui reads, its case and parameters aside: PNG; JPEG, GIF, WebP, ICO and BMP once their decoders are imported (`ui/web` does); SVG once `ui/svg` is. AVIF is not read, so a source of AVIF is passed over |

| Choosing | |
|---|---|
| the density | a candidate's own, or its width over the slot's |
| the one shown | the least density at or above the screen's pixels a point, else the greatest. A denser candidate the Document already has, or a `data:` one, wins over it, as one in Chrome's cache does |
| its size | its pixels over its density: a 2x picture shows at half its pixels, one of widths at its slot's width |
| again | when the viewport resizes, or an attribute or the `source`s change; one in a `picture` with a `source` of `media` also when another media feature changes (the colour scheme, the density with it). The pixels a point changing alone keep the picture, as Chrome keeps it: each `img` chooses at the density it last chose for until something has it choose again. The picture shown stays until the one chosen in its place lands |

An `img` whose `sizes` starts with `auto` is laid out 300 × 150 points, without proportions, whatever its picture, as Chrome's sheet sizes it (`contain: size; contain-intrinsic-size: 300px 150px`): give it a width and a height.

| Default CSS | |
|---|---|
| `picture`, `source` | inline (Chrome: the same); a `source` shows nothing |

Automation: the `img` is recorded as any `img`; `picture` and `source` by their `role`, as any other element.

Chosen as Chrome chooses at one and two pixels a point, and sized as Chrome sizes the pictures, in `TestAnImgChoosesThePictureChromeChoosesAndShowsItAtItsDensity`; chosen again when Chrome's are, on a denser screen, a dark scheme and a wider window, in `TestAnImgChoosesAgainWhenChromesDoes`.

```
go run ./examples/93-html-elements -element picture
```
