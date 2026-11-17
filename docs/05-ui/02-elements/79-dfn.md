# dfn

![A beacon is a fire that lights a whole floor, beacon in italics](dfn.png "ember · dfn")

```go
ui.P(ui.Text("A "), ui.Dfn(ui.Text("beacon")), ui.Text(" is a fire that lights a whole floor."))
```

| Attribute | |
|---|---|
| `ui.Text(s)` | its text; children may be any inline elements |

The term a sentence defines.

| Default CSS | |
|---|---|
| `dfn` | `font-style: italic` |

Automation: recorded, as a label, only when it has an `id`.

```
go run ./examples/93-html-elements -element dfn
```
