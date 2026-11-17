# bdi

![Ayla: 4210 points](bdi.png "ember · bdi")

```go
ui.P(ui.Bdi(ui.Text("Ayla")), ui.Text(": 4210 points"))
```

| Attribute | |
|---|---|
| `ui.Text(s)` | its text; children may be any inline elements |

Text kept apart from the direction of the line round it, a player's name for one. ui has no bidirectional text: every line runs left to right, and `bdi` changes nothing.

| Default CSS | |
|---|---|
| `bdi` | none: inline, as [`span`](14-span.md) |

Automation: recorded, as a label, only when it has an `id`.

```
go run ./examples/93-html-elements -element bdi
```
