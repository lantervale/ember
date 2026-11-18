# iframe

![A dark help page in a frame with an inset border, scrolled by its own bar](iframe.png "ember · iframe")

```go
help := ui.NewDocument(ui.ScrollingViewport(), ui.CSS(sheet))
help.Body().Append(ui.H3(ui.Text("How to play")), ui.P(ui.Text("Light the lantern before the stairs.")))

frame := ui.Iframe(help, ui.ID("help"), ui.Attr("height", "160"))
doc.Body().Append(frame)
```

A Document shown in another: the element holds a `*ui.Document` the program built, lays it out in its content box and draws it there, clipped, where the element stands among the others ([decision 0103](../../99-decisions/0103-an-iframe-shows-a-document-the-program-gives-it.md)). ember loads nothing for it: `src` and `srcdoc` are attributes like any other, for the program to read.

| | |
|---|---|
| `ui.Iframe(child, args…)` | the element showing `child`; nil shows nothing in the box |
| `doc.SetFrame(n, child)` | shows another Document from the next frame, nil for none; the one it showed loses its focus and selection. A Document shows in one frame at a time |
| `doc.Frame(n)` | the Document it shows |
| `doc.FrameShown(n)` | whether that Document was drawn in the last frame: false out of view, under `display: none`, or taken out. A program stops what feeds a frame nobody sees |
| `doc.FramePoint(n, at)` | a point of the page, as `BoxOf` gives them, in the framed Document's own, for its `ElementAt`; false off the frame |

| Attribute | |
|---|---|
| `width`, `height` | its size, pixels or a percentage, under every rule; 300 × 150 without them. Neither side follows the other: a frame has no ratio of its own, `aspect-ratio` gives it one |
| `frameborder` | `0` or `no` takes the border off |

| | |
|---|---|
| the framed Document's viewport | the content box: its `@media` rules are answered for it, and it is laid out again when the box changes. It scrolls there with `ui.ScrollingViewport()` |
| zoom | the page's zoom scales the frame and what it shows; the framed Document's own `SetZoom` goes on top |
| the pointer | the framed Document's where the frame is what the page finds under it: an element of the page lying over the frame keeps the pointer. A press begun in the frame stays its own until released |
| the wheel | scrolls the framed Document while it can scroll that way, then the page |
| the focus | a click in the frame, or `Focus` on an element in it, gives the frame the page's focus; the keys then go to the framed Document alone. Tab walks from the page into the frame, through it and out the other side, backwards too |
| the selection | each Document selects its own text; selecting in one clears the others |
| automation | the framed elements are recorded under the frame's id: `help/got-it` |
| drawing | a frame out of view is not drawn; one nothing changed in replays, and a change inside it paints the frame, not the page |
| not yet | see [What is missing](../10-missing.md#frames) |

Frames nest: a framed Document holds iframes of its own.

| Default CSS | |
|---|---|
| `iframe` | `display: inline-block; border: 2px inset #eeeeee` (Chrome: inline) |
| `iframe[frameborder="0"]`, `[frameborder="no"]` | `border-width: 0` |

Automation: recorded by its `role` like any other element; what it shows is under it.

Sized as Chrome sizes it in `TestAnIframesBoxMatchesChromes`, drawn as Chrome draws it in the `iframe` page of `TestPagesLookLikeChrome`.

```
go run ./examples/93-html-elements -element iframe
```
