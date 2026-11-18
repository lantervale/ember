# svg

![A compass drawn twice, small in amber and large in blue](svg.png "ember · svg")

```go
import _ "github.com/lantervale/ember/ui/svg"

ui.Svg(`<circle cx="12" cy="12" r="10" fill="currentColor"/>`,
	ui.Attr("viewBox", "0 0 24 24"), ui.Attr("width", "24"))
```

A picture written in the page: the element holds SVG as text, `markup`, and shows it as an `img` shows a file, drawn at the size the element takes on the screen. It needs `ui/svg` imported, as SVG files do ([decision 0098](../../99-decisions/0098-an-svg-element-is-one-picture-holding-its-markup.md)).

| | |
|---|---|
| `width`, `height` | the element's size, a number of pixels or a length with a unit (`3em`, `50%`), under every rule |
| `viewBox` | what of the drawing shows, and the ratio a missing side follows; with neither a size nor a `viewBox` the element is 300 by 150 |
| `currentColor` | the colour of the element's text, so an icon takes its button's `color`, on hover too |
| `n.Markup()`, `n.SetMarkup(markup)` | what it holds, read and changed; drawn again from the next frame |
| `ui.PictureOf(n)` | the address of the picture it shows, a data URL of its markup |

A rule naming `svg`, its classes or its attributes sizes and places it as any element; no rule reaches the shapes inside, which are not elements. What ember's svg reader doesn't draw is a warning, as for a file ([What is missing](../10-missing.md#svg-pictures)).

| Default CSS | |
|---|---|
| `svg` | `display: inline-block` |

Automation: not recorded.

Sized as Chrome sizes it in `TestSvgElementsAreSizedAsChromeSizesThem`.

```
go run ./examples/93-html-elements -element svg
```
