# center

![Game over and Press Enter to try again, both centred in a dashed box](center.png "ember · center")

```go
ui.Center(
	ui.Div(ui.Text("Game over")),
	ui.Div(ui.Text("Press Enter to try again")),
)
```

The old centring element: a block whose lines are centred. It centres text only: Chrome also centres the blocks in it, and here a block narrower than it stays at its left edge unless its own margins are `auto`. The dashed box in the picture is the example's.

| Default CSS | |
|---|---|
| `center` | `display: block; text-align: center` |

Automation: recorded, as a label, only when it has an `id`.

```
go run ./examples/93-html-elements -element center
```
