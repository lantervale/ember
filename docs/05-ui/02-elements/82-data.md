# data

![In stock: Lantern](data.png "ember · data")

```go
ui.P(ui.Text("In stock: "), ui.Data(ui.Value("item-042"), ui.Text("Lantern")))
```

| Attribute | |
|---|---|
| `ui.Text(s)` | its text; children may be any inline elements |

Text with the value a program reads for it. Plain semantics: it shows its text, and its `value` attribute is the game's to read with `n.Attr`. Its constructor is `ui.Data`.

| Default CSS | |
|---|---|
| `data` | none: inline, as [`span`](14-span.md) |

Automation: recorded, as a label, only when it has an `id`.

```
go run ./examples/93-html-elements -element data
```
