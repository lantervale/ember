# sub

![Brewed from H2O and moss, the 2 smaller and below the line](sub.png "ember · sub")

```go
ui.P(ui.Text("Brewed from H"), ui.Sub(ui.Text("2")), ui.Text("O and moss."))
```

| Attribute | |
|---|---|
| `ui.Text(s)` | its text; children may be any inline elements |

A subscript: smaller text whose baseline is a fifth of its parent's font size and a pixel below the line's, as Chrome's. The line grows to hold it.

| Default CSS | |
|---|---|
| `sub` | `font-size: smaller; vertical-align: sub` |

Automation: recorded, as a label, only when it has an `id`.

```
go run ./examples/93-html-elements -element sub
```
