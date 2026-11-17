# small

![Slot 1, Aldric, then saved two hours ago in smaller text](small.png "ember · small")

```go
ui.P(ui.Text("Slot 1, Aldric "), ui.Small(ui.Text("saved two hours ago")))
```

| Attribute | |
|---|---|
| `ui.Text(s)` | its text; children may be any inline elements |

Fine print: one size smaller than the text round it.

| Default CSS | |
|---|---|
| `small` | `font-size: smaller` |

Automation: recorded, as a label, only when it has an `id`.

```
go run ./examples/93-html-elements -element small
```
