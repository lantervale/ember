# address

![Lantervale studio over a second line, both in italics](address.png "ember · address")

```go
ui.Address(
	ui.Text("Lantervale studio"),
	ui.Br(),
	ui.Text("Made by two leads and their agents"),
)
```

Who to reach about the page: a block in italics.

| Default CSS | |
|---|---|
| `address` | `display: block; font-style: italic` |

Automation: recorded, as a label, only when it has an `id`.

```
go run ./examples/93-html-elements -element address
```
