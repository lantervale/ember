# center

![Game over and Press Enter to try again, both centred in a dashed box](center.png "ember · center")

```go
ui.Center(
	ui.Div(ui.Text("Game over")),
	ui.Div(ui.Text("Press Enter to try again")),
)
```

The old centring element: a block whose lines are centred, and the blocks in it that are narrower than it, a table or a sized box, as in Chrome. The dashed box in the picture is the example's.

| Default CSS | |
|---|---|
| `center` | `display: block; text-align: -webkit-center` |

Automation: recorded, as a label, only when it has an `id`.

```
go run ./examples/93-html-elements -element center
```
