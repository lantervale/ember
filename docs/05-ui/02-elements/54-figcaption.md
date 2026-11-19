# figcaption

![The crypt's seal, the caption under the panel picture](figure.png "ember · figure")

```go
ui.Figure(
	ui.Img(ui.Attr("src", "panel.png"), ui.Attr("alt", "A panel")),
	ui.Figcaption(ui.Text("The crypt's seal")),
)
```

A [`figure`](53-figure.md)'s caption: a block, above or below what it captions as it is written, first or last.

| Default CSS | |
|---|---|
| `figcaption` | `display: block` |

Automation: recorded, as a label, only when it has an `id`.

```
go run ./examples/93-html-elements -element figcaption
```
