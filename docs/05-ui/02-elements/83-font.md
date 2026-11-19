# font

![Gold: 1,250, the number as the text before it](font.png "ember · font")

```go
ui.P(ui.Text("Gold: "), ui.Font(ui.Attr("color", "#b8860b"), ui.Attr("size", "5"), ui.Text("1,250")))
```

| Attribute | |
|---|---|
| `ui.Text(s)` | its text; children may be any inline elements |

The old font element, kept so old pages build. Its `color`, `size` and `face` attributes style nothing: use a [`span`](14-span.md) with a `Style` or a class.

| Default CSS | |
|---|---|
| `font` | none: inline, as [`span`](14-span.md) |

Automation: recorded, as a label, only when it has an `id`.

```
go run ./examples/93-html-elements -element font
```
