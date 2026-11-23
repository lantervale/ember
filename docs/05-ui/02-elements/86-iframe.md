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
| `ui.Guarded()` | on the iframe: a panic of the Document it shows, while it is laid out, handles its input, runs its callbacks or paints, stops there. The frame then shows nothing, the page round it goes on, and the panic and its stack are a line of `errors.log`. Without it the panic is the program's, as any other |
| `doc.FrameBroken(n)` | the panic that emptied a guarded frame, a `*ui.FrameError` (its `Value` and `Stack`), nil for none, until `SetFrame`. The Document that panicked is let go of and is not to be shown again |
| `doc.FrameShown(n)` | whether that Document was drawn in the last frame: false out of view, under `display: none`, or taken out. A program stops what feeds a frame nobody sees |
| `doc.FramePoint(n, at)` | a point of the page, as `BoxOf` gives them, in the framed Document's own, for its `ElementAt`; false off the frame |

| Attribute | |
|---|---|
| `width`, `height` | its size, pixels or a percentage, under every rule; 300 × 150 without them. Neither side follows the other: a frame has no ratio of its own. With `aspect-ratio` and no size it fills its line, as Chrome's, the ratio holding between the boxes `box-sizing` names |
| `frameborder` | `0` or `no` takes the border off |

| | |
|---|---|
| the framed Document's viewport | the content box: its `@media` rules are answered for it, and it is laid out again when the box changes. It scrolls there with `ui.ScrollingViewport()` |
| zoom | the page's zoom scales the frame and what it shows; the framed Document's own `SetZoom` goes on top |
| the pointer | the framed Document's where the frame is what the page finds under it: an element of the page lying over the frame keeps the pointer. A press begun in the frame stays its own until released |
| the wheel | scrolls the framed Document while it can scroll that way, then the page |
| the focus | a click in the frame, or `Focus` on an element in it, gives the frame the page's focus; the keys then go to the framed Document. Tab walks from the page into the frame, through it and out the other side, backwards too; the arrows and a pad's stick walk out of the frame where nothing of it lies that way, and into a frame they reach, to its element nearest the side they came in by |
| keys and Back | a key or a Back no callback of the framed Document stopped goes on to the page round it, its `OnKey` and `OnBack` hearing it on the iframe, as an event retargeted ([decision 0141](../../99-decisions/0141-keys-and-the-arrows-cross-a-frame-the-frame-fades-and-moves-with-its-iframe.md)): one `OnBack` on a screen's root hears an Escape pressed in a frame on it. The World event is the framed Document's alone |
| a modal dialog | of the framed Document holds that Document's input, as Chrome's; the page round it still takes the pointer and the keys |
| `opacity` | below 1 on the iframe or round it fades the frame with what it shows, drawn as one picture |
| the selection | each Document selects its own text; selecting in one clears the others |
| automation | the framed elements are recorded under the frame's id: `help/got-it` |
| drawing | a frame out of view is not drawn; one nothing changed in replays, and a change inside it paints the frame, not the page |
| not yet | see [What is missing](../10-missing.md#frames) |

Frames nest: a framed Document holds iframes of its own.

An iframe taken out of its Document, itself or with an element round it, lets go of the Document it showed at the page's next frame, as `SetFrame(n, nil)` does: its focus and selection go, and nothing holds it any more. Put back within the frame, in the same Document or moved to another, as an element moved does, it keeps it: Chrome loads a moved frame again, and here what the program gave it is shown there. Put back later, it shows nothing until `SetFrame`.

| Default CSS | |
|---|---|
| `iframe` | `display: inline-block; border: 2px inset #eeeeee` (Chrome: inline) |
| `iframe[frameborder="0"]`, `[frameborder="no"]` | `border-width: 0` |

Automation: recorded by its `role` like any other element; what it shows is under it.

Sized as Chrome sizes it in `TestAnIframesBoxMatchesChromes` and `TestAnIframeWithOnlyARatioFillsItsLineAsInChrome`, drawn as Chrome draws it in the `iframe`, `iframe-borders` and `iframe-faded` pages of `TestPagesLookLikeChrome`.

```
go run ./examples/93-html-elements -element iframe
```
