# mark

![Light the three beacons before dawn, beacons on a yellow background](mark.png "ember · mark")

```go
ui.P(ui.Text("Light the three "), ui.Mark(ui.Text("beacons")), ui.Text(" before dawn."))
```

| Attribute | |
|---|---|
| `ui.Text(s)` | its text; children may be any inline elements |

Text marked for the reader, what a search found for one.

| Default CSS | |
|---|---|
| `mark` | `background-color: #ff0; color: #000` |

Automation: recorded, as a label, only when it has an `id`.

```
go run ./examples/93-html-elements -element mark
```
