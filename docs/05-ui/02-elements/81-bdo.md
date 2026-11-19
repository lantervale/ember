# bdo

![The rune reads NEPO, drawn left to right as written](bdo.png "ember · bdo")

```go
ui.P(ui.Text("The rune reads "), ui.Bdo(ui.Attr("dir", "rtl"), ui.Text("NEPO")), ui.Text("."))
```

| Attribute | |
|---|---|
| `ui.Text(s)` | its text; children may be any inline elements |

Text whose direction its `dir` sets. ui has no bidirectional text: `dir` is not read, and the text is drawn left to right as written.

| Default CSS | |
|---|---|
| `bdo` | none: inline, as [`span`](14-span.md) |

Automation: recorded, as a label, only when it has an `id`.

```
go run ./examples/93-html-elements -element bdo
```
