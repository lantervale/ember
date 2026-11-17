# figure

![The panel picture set in from the left, The crypt's seal under it](figure.png "ember · figure")

```go
ui.Figure(
	ui.Img(ui.Attr("src", "panel.png"), ui.Attr("alt", "A panel")),
	ui.Figcaption(ui.Text("The crypt's seal")),
)
```

A picture, a table or anything else shown apart from the text, with its [`figcaption`](54-figcaption.md): a block set in on both sides.

| Default CSS | |
|---|---|
| `figure` | `display: block; margin: 1em 40px` |

Automation: recorded, as a label, only when it has an `id`.

```
go run ./examples/93-html-elements -element figure
```
